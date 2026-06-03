//go:build js && wasm

package web

import (
	"context"
	"fmt"
	"image"
	"syscall/js"

	pcanvas "github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/output"
)

// rasterSurface implements [output.LiveSurface] for browser rendering.
// It creates a <canvas> element, draws via CPU rasterizer, and transfers
// pixels to the screen via Canvas2D putImageData.
type rasterSurface struct {
	container js.Value // parent <div>
	canvas    js.Value // <canvas> element
	ctx2d     js.Value // CanvasRenderingContext2D

	cv     *pcanvas.RasterCanvas // CPU rasterizer
	events chan output.Event     // fed by DOM listeners
	bounds image.Rectangle

	callbacks []jsCallback // tracked for cleanup
	fps       *fpsTracker  // toolbar FPS display
}

// newRasterSurface creates a <canvas> element inside container and returns
// a LiveSurface ready for Session integration.
func newRasterSurface(container js.Value, w, h int) (*rasterSurface, error) {
	doc := js.Global().Get("document")
	canvasEl := doc.Call("createElement", "canvas")
	canvasEl.Set("width", w)
	canvasEl.Set("height", h)

	// CSS sizing: fill the container.
	style := canvasEl.Get("style")
	style.Set("display", "block")
	style.Set("width", "100%")
	style.Set("height", "100%")
	style.Set("touch-action", "none") // disable browser pan/zoom for pointer events

	container.Call("appendChild", canvasEl)

	ctx2d := canvasEl.Call("getContext", "2d")
	if ctx2d.IsNull() || ctx2d.IsUndefined() {
		return nil, fmt.Errorf("web: failed to get 2d context from canvas")
	}

	events := make(chan output.Event, 64) //nolint:mnd // Buffer size matches output/window event queue depth.

	s := &rasterSurface{
		container: container,
		canvas:    canvasEl,
		ctx2d:     ctx2d,
		events:    events,
		bounds:    image.Rect(0, 0, w, h),
		fps:       newFPSTracker(),
	}

	s.callbacks = registerDOMEvents(canvasEl, events)

	return s, nil
}

// Acquire returns a CPU-rasterized Canvas for drawing the next frame.
// The canvas is reused across frames; a new one is created only on resize.
func (s *rasterSurface) Acquire(_ context.Context) (pcanvas.Canvas, error) {
	s.fps.Begin()

	w, h := s.bounds.Dx(), s.bounds.Dy()

	if s.cv == nil || s.cv.Context().Width() != w || s.cv.Context().Height() != h {
		if s.cv != nil {
			_ = s.cv.Close()
		}
		s.cv = pcanvas.NewRasterCanvasCPU(w, h)
	}

	return s.cv, nil
}

// Commit transfers the CPU-rasterized image to the browser canvas via
// putImageData. This is called after Figure.Draw completes.
func (s *rasterSurface) Commit(_ context.Context) error {
	if s.cv == nil {
		return nil
	}

	img := s.cv.Image()
	putImageData(s.ctx2d, img)

	s.fps.End()

	return nil
}

// Bounds returns the current logical drawing size.
func (s *rasterSurface) Bounds() image.Rectangle { return s.bounds }

// Events returns the channel receiving DOM events translated to output.Event.
func (s *rasterSurface) Events() <-chan output.Event { return s.events }

// Close removes DOM event listeners, releases JS callbacks, removes the canvas
// element, and closes the raster canvas.
func (s *rasterSurface) Close() error {
	releaseCallbacks(s.callbacks)
	s.callbacks = nil

	// Remove the canvas from the DOM.
	if !s.canvas.IsUndefined() && !s.canvas.IsNull() {
		parent := s.canvas.Get("parentElement")
		if !parent.IsUndefined() && !parent.IsNull() {
			parent.Call("removeChild", s.canvas)
		}
	}

	if s.cv != nil {
		return s.cv.Close()
	}

	return nil
}

// Compile-time interface check.
var _ output.LiveSurface = (*rasterSurface)(nil)
