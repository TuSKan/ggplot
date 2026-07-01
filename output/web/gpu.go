//go:build js && wasm

package web

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"syscall/js"
	"unsafe"

	"github.com/gogpu/gg"

	"github.com/gogpu/gg/integration/ggcanvas"
	"github.com/gogpu/gpucontext"
	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"

	pcanvas "github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/output"
)

// WithGPU enables WebGPU-accelerated rendering. The GPU accelerator (SDF
// shapes, MSDF text, convex polygons, stencil paths) renders directly to
// the WebGPU surface via ggcanvas.RenderDirect — zero-copy, no readback.
//
// This requires a browser with WebGPU support (Chrome 113+, Edge 113+,
// Firefox behind flag). If WebGPU is unavailable at runtime, Mount returns
// an error.
func WithGPU() Opt { return func(o *Options) { o.GPU = true } }

// mountGPU creates a WebGPU surface inside the container, initialises the
// wgpu pipeline (instance → adapter → device → surface), registers the GPU
// accelerator, and runs a standard output.Session event loop.
//
// Rendering uses ggcanvas.RenderDirect for GPU-accelerated content (SDF
// shapes, text) and queue.WriteTexture as fallback for CPU-rasterized
// content.
func mountGPU(ctx context.Context, src output.Source, fig output.Figure, container js.Value, o Options) error {
	surf, err := newGPUSurface(container, o.Width, o.Height)
	if err != nil {
		return fmt.Errorf("web: gpu: %w", err)
	}
	defer func() { _ = surf.Close() }()

	// Initial render.
	if err := output.Render(ctx, fig, surf); err != nil {
		return fmt.Errorf("web: gpu: initial render: %w", err)
	}

	// Session-based event loop — same as raster/SVG modes.
	sessOpts := []output.SessionOpt{
		output.WithController(o.Controller),
	}
	if o.RebuildDelay > 0 {
		sessOpts = append(sessOpts, output.WithRebuildDelay(o.RebuildDelay))
	}
	if o.OnRebuildErr != nil {
		sessOpts = append(sessOpts, output.WithRebuildError(o.OnRebuildErr))
	}

	sess := output.NewSession(src, surf, sessOpts...)
	return sess.Run(ctx)
}

// ---------------------------------------------------------------------------
// gpuDeviceProvider — gpucontext.DeviceProvider backed by raw wgpu
// ---------------------------------------------------------------------------

// gpuDeviceProvider wraps raw wgpu resources as a gpucontext.DeviceProvider.
// This lets ggcanvas and the GPU accelerator share our device.
type gpuDeviceProvider struct {
	device  *wgpu.Device
	adapter *wgpu.Adapter
	format  wgpu.TextureFormat
}

func (p *gpuDeviceProvider) Device() gpucontext.Device             { return wgpu.DeviceToHandle(p.device) }
func (p *gpuDeviceProvider) Queue() gpucontext.Queue               { return wgpu.QueueToHandle(p.device.Queue()) }
func (p *gpuDeviceProvider) SurfaceFormat() gputypes.TextureFormat { return p.format }
func (p *gpuDeviceProvider) Adapter() gpucontext.Adapter           { return wgpu.AdapterToHandle(p.adapter) }
func (p *gpuDeviceProvider) AdapterInfo() gpucontext.AdapterInfo {
	info := p.adapter.Info()
	var atype gpucontext.AdapterType
	switch info.DeviceType {
	case gputypes.DeviceTypeDiscreteGPU:
		atype = gpucontext.AdapterTypeDiscrete
	case gputypes.DeviceTypeIntegratedGPU:
		atype = gpucontext.AdapterTypeIntegrated
	case gputypes.DeviceTypeCPU:
		atype = gpucontext.AdapterTypeSoftware
	default:
		atype = gpucontext.AdapterTypeUnknown
	}
	return gpucontext.AdapterInfo{
		Name: info.Name,
		Type: atype,
	}
}

// ---------------------------------------------------------------------------
// gpuSurface — output.LiveSurface backed by ggcanvas + GPU accelerator
// ---------------------------------------------------------------------------

// gpuSurface implements [output.LiveSurface] for browser WebGPU rendering.
// It creates a <canvas> inside the container, initialises the wgpu pipeline,
// and uses ggcanvas + the GPU accelerator for zero-copy rendering.
//
// Data flow:
//
//	Draw calls → gg.Context → GPU accelerator (SDF/text/convex/stencil)
//	  → FlushGPUWithView(surfaceView) → GPU render pass → surface → present
//
// CPU-rasterized content (fallback paths) is uploaded via queue.WriteTexture.
type gpuSurface struct {
	container js.Value
	canvas    js.Value

	// wgpu pipeline — owned, released on Close.
	instance *wgpu.Instance
	adapter  *wgpu.Adapter
	device   *wgpu.Device
	surface  *wgpu.Surface
	format   wgpu.TextureFormat

	// ggcanvas for integrated GPU rendering.
	ggcanvas *ggcanvas.Canvas
	provider *gpuDeviceProvider

	// DOM events.
	events    chan output.Event
	callbacks []jsCallback
	fps       *fpsTracker
	bounds    image.Rectangle
}

// newGPUSurface creates a WebGPU-backed surface inside the container.
//
// Pipeline: Instance → Adapter → Device → Surface → Configure → GPU Accelerator.
func newGPUSurface(container js.Value, w, h int) (*gpuSurface, error) {
	doc := js.Global().Get("document")

	// Create <canvas> inside the container (same pattern as rasterSurface).
	canvasEl := doc.Call("createElement", "canvas")
	canvasEl.Set("width", w)
	canvasEl.Set("height", h)

	style := canvasEl.Get("style")
	style.Set("display", "block")
	style.Set("width", "100%")
	style.Set("height", "100%")
	style.Set("touch-action", "none") // disable browser pan/zoom

	container.Call("appendChild", canvasEl)

	// --- wgpu pipeline ---
	instance, err := wgpu.CreateInstance(nil)
	if err != nil {
		return nil, fmt.Errorf("CreateInstance: %w", err)
	}

	adapter, err := instance.RequestAdapter(nil)
	if err != nil {
		instance.Release()
		return nil, fmt.Errorf("RequestAdapter: %w", err)
	}

	slog.Info("adapter selected",
		"name", adapter.Info().Name,
		"type", adapter.Info().DeviceType,
	)

	device, err := adapter.RequestDevice(nil)
	if err != nil {
		adapter.Release()
		instance.Release()
		return nil, fmt.Errorf("RequestDevice: %w", err)
	}

	// Create surface from our canvas (not querySelector — we own it).
	surface, err := instance.CreateSurfaceFromCanvas(canvasEl)
	if err != nil {
		device.Release()
		adapter.Release()
		instance.Release()
		return nil, fmt.Errorf("CreateSurfaceFromCanvas: %w", err)
	}

	// Use RGBA8Unorm to match gg.Context pixmap format (RGBA, 8bpc).
	// This avoids R/B channel swapping. RGBA8Unorm is a supported canvas
	// format per the WebGPU spec.
	format := gputypes.TextureFormatRGBA8Unorm

	// Configure the surface with CopyDst for fallback queue.WriteTexture
	// and RenderAttachment for GPU-direct render passes.
	if cfgErr := surface.Configure(device, &wgpu.SurfaceConfiguration{
		Format: format,
		Usage:  gputypes.TextureUsageRenderAttachment | gputypes.TextureUsageCopyDst,
		Width:  uint32(w), //nolint:gosec // G115: w validated positive by caller
		Height: uint32(h), //nolint:gosec // G115: h validated positive by caller
	}); cfgErr != nil {
		surface.Release()
		device.Release()
		adapter.Release()
		instance.Release()
		return nil, fmt.Errorf("surface.Configure: %w", cfgErr)
	}

	// Create device provider for GPU accelerator + ggcanvas.
	provider := &gpuDeviceProvider{
		device:  device,
		adapter: adapter,
		format:  format,
	}

	// Register the GPU accelerator with our device. This enables SDF shapes,
	// MSDF text, convex polygons, and stencil paths on the GPU.
	if setErr := gg.SetAcceleratorDeviceProvider(provider); setErr != nil {
		slog.Warn("GPU accelerator init failed, will use CPU fallback", "err", setErr)
	} else {
		slog.Info("GPU accelerator registered for browser WebGPU")
	}

	// Create ggcanvas for integrated rendering.
	ggc, err := ggcanvas.NewWithScale(provider, w, h, 1.0)
	if err != nil {
		slog.Warn("ggcanvas creation failed, will use WriteTexture fallback", "err", err)
	}

	events := make(chan output.Event, 64) //nolint:mnd // Buffer size matches output/window event queue depth.

	s := &gpuSurface{
		container: container,
		canvas:    canvasEl,
		instance:  instance,
		adapter:   adapter,
		device:    device,
		surface:   surface,
		format:    format,
		provider:  provider,
		ggcanvas:  ggc,
		events:    events,
		bounds:    image.Rect(0, 0, w, h),
		fps:       newFPSTracker(),
	}

	s.callbacks = registerDOMEvents(canvasEl, events)

	return s, nil
}

// Acquire begins a GPU frame: lazy-inits or resizes the ggcanvas, clears
// it, and returns a canvas.Canvas for drawing.
func (s *gpuSurface) Acquire(_ context.Context) (pcanvas.Canvas, error) {
	s.fps.Begin()

	w, h := s.bounds.Dx(), s.bounds.Dy()

	if s.ggcanvas != nil {
		// Handle resize.
		cw, ch := s.ggcanvas.Context().Width(), s.ggcanvas.Context().Height()
		if cw != w || ch != h {
			if err := s.ggcanvas.Resize(w, h); err != nil {
				return nil, fmt.Errorf("ggcanvas.Resize: %w", err)
			}
			// Reconfigure the surface for new dimensions.
			if err := s.surface.Configure(s.device, &wgpu.SurfaceConfiguration{
				Format: s.format,
				Usage:  gputypes.TextureUsageRenderAttachment | gputypes.TextureUsageCopyDst,
				Width:  uint32(w), //nolint:gosec // G115: w is from bounds, always positive
				Height: uint32(h), //nolint:gosec // G115: h is from bounds, always positive
			}); err != nil {
				return nil, fmt.Errorf("surface.Configure resize: %w", err)
			}
		}

		// Draw through ggcanvas — GPU accelerator is active.
		s.ggcanvas.Draw(func(cc *gg.Context) {
			cc.SetColor(color.White)
			cc.Clear()
		})

		rc := pcanvas.RasterFromContext(s.ggcanvas.Context())
		rc.Clear(color.White)
		return rc, nil
	}

	// Fallback: no ggcanvas, use raw gg.Context (shouldn't happen normally).
	return nil, fmt.Errorf("ggcanvas not initialized")
}

// Commit ends the frame. Tries GPU-direct rendering via ggcanvas.RenderDirect,
// falling back to queue.WriteTexture if GPU-direct is unavailable.
func (s *gpuSurface) Commit(_ context.Context) error {
	w, h := s.bounds.Dx(), s.bounds.Dy()

	// Acquire the surface texture for this frame.
	surfTex, _, err := s.surface.GetCurrentTexture()
	if err != nil {
		return fmt.Errorf("GetCurrentTexture: %w", err)
	}

	uw := uint32(w) //nolint:gosec // G115: w positive
	uh := uint32(h) //nolint:gosec // G115: h positive

	if s.ggcanvas != nil {
		// Create a TextureView handle from the surface texture.
		surfView, vErr := s.device.CreateTextureView(surfTex.Texture(), &wgpu.TextureViewDescriptor{
			Label:  "surface_view",
			Format: s.format,
		})
		if vErr != nil {
			return fmt.Errorf("CreateTextureView: %w", vErr)
		}

		// Wrap as gpucontext.TextureView for ggcanvas.
		tv := gpucontext.NewTextureView(unsafe.Pointer(surfView)) //nolint:gosec // Go spec Rule 1 — same pattern as gg/internal/gpu

		// GPU-direct: flush all GPU-accelerated draws to the surface view.
		if rdErr := s.ggcanvas.RenderDirect(tv, uw, uh); rdErr != nil {
			slog.Debug("RenderDirect failed, falling back to WriteTexture", "err", rdErr)
			// Fall through to WriteTexture fallback below.
			s.writeTextureFallback(surfTex, uw, uh)
		}
	} else {
		s.writeTextureFallback(surfTex, uw, uh)
	}

	// Present — on browser this is a no-op (auto-present on event loop yield).
	_ = s.surface.Present(surfTex)

	s.fps.End()
	return nil
}

// writeTextureFallback uploads the CPU pixmap to the surface texture via
// queue.WriteTexture. Used when GPU-direct rendering is unavailable.
func (s *gpuSurface) writeTextureFallback(surfTex *wgpu.SurfaceTexture, w, h uint32) {
	if s.ggcanvas == nil {
		return
	}

	pixmap := s.ggcanvas.Context().ResizeTarget()
	data := pixmap.Data()

	stride := w * 4 //nolint:mnd // 4 = RGBA bytes per pixel
	if err := s.device.Queue().WriteTexture(
		&wgpu.ImageCopyTexture{
			Texture:  surfTex.Texture(),
			MipLevel: 0,
			Origin:   wgpu.Origin3D{},
			Aspect:   gputypes.TextureAspectAll,
		},
		data,
		&wgpu.ImageDataLayout{
			Offset:       0,
			BytesPerRow:  stride,
			RowsPerImage: h,
		},
		&wgpu.Extent3D{
			Width:              w,
			Height:             h,
			DepthOrArrayLayers: 1,
		},
	); err != nil {
		slog.Warn("writeTextureFallback failed", "err", err)
	}
}

// Bounds returns the current logical drawing size.
func (s *gpuSurface) Bounds() image.Rectangle { return s.bounds }

// Events returns the channel receiving DOM events translated to output.Event.
func (s *gpuSurface) Events() <-chan output.Event { return s.events }

// Close releases all GPU resources, removes DOM listeners, and removes the
// canvas from the container.
func (s *gpuSurface) Close() error {
	releaseCallbacks(s.callbacks)
	s.callbacks = nil

	if s.ggcanvas != nil {
		s.ggcanvas.Close()
		s.ggcanvas = nil
	}

	if s.surface != nil {
		s.surface.Release()
		s.surface = nil
	}
	if s.device != nil {
		s.device.Release()
		s.device = nil
	}
	if s.adapter != nil {
		s.adapter.Release()
		s.adapter = nil
	}
	if s.instance != nil {
		s.instance.Release()
		s.instance = nil
	}

	// Remove canvas from DOM.
	if !s.canvas.IsUndefined() && !s.canvas.IsNull() {
		parent := s.canvas.Get("parentElement")
		if !parent.IsUndefined() && !parent.IsNull() {
			parent.Call("removeChild", s.canvas)
		}
	}

	return nil
}

// Compile-time interface check.
var _ output.LiveSurface = (*gpuSurface)(nil)
