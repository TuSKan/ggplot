//go:build !js

// Package window presents a ggplot figure in a native desktop GPU window via
// gogpu/ui, the enterprise-grade GUI toolkit. It creates a custom widget that
// renders the figure directly into gogpu's GPU scene graph for hardware-
// accelerated vector rendering.
//
// Pan (mouse drag) and zoom (mouse wheel) are handled through gogpu/ui's
// event system, translated to ggplot's platform-neutral [output.Controller].
//
// [Show] blocks until the window closes and must be called from the main
// goroutine (GPU/windowing requires the main OS thread).
package window

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"sync"
	"time"

	"github.com/gogpu/gg/scene"
	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/desktop"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/output"
)

const (
	defaultWidth  = 800
	defaultHeight = 600
)

// Options configures a window opened by [Show].
type Options struct {
	Title        string
	Width        int
	Height       int
	Controller   output.Controller
	RebuildDelay time.Duration // 0 = synchronous rebuild (default)
	OnRebuildErr func(error)   // called when an async rebuild fails (nil = silent)
}

// Opt functionally configures [Options].
type Opt func(*Options)

// WithTitle sets the window title.
func WithTitle(title string) Opt { return func(o *Options) { o.Title = title } }

// WithSize sets the initial window size in logical pixels.
func WithSize(w, h int) Opt { return func(o *Options) { o.Width, o.Height = w, h } }

// WithController overrides the default pan/wheel-zoom controller.
func WithController(c output.Controller) Opt {
	return func(o *Options) {
		if c != nil {
			o.Controller = c
		}
	}
}

// WithRebuildDelay enables async, debounced rebuilds. When set, rapid rebuild
// requests (e.g. repeated zoom-past-extent) are coalesced: the rebuild fires
// only after delay elapses with no new request. The last good figure keeps
// drawing until the background build completes. Zero (default) means synchronous.
func WithRebuildDelay(d time.Duration) Opt {
	return func(o *Options) { o.RebuildDelay = d }
}

// WithRebuildError sets a handler invoked when an async rebuild fails. The last
// good figure is kept on screen; the error is non-fatal. Nil (default) means
// async rebuild errors are silent (but the first fatal error still stops Show).
func WithRebuildError(fn func(error)) Opt {
	return func(o *Options) { o.OnRebuildErr = fn }
}

// Show builds src into a figure and presents it in a native window, processing
// pan/zoom and other interactions until the window is closed. It blocks and
// must be called from the main goroutine.
func Show(ctx context.Context, src output.Source, opts ...Opt) error {
	o := Options{Title: "ggplot", Width: defaultWidth, Height: defaultHeight, Controller: output.DefaultController()}
	for _, opt := range opts {
		opt(&o)
	}

	fig, err := src.Build(ctx)
	if err != nil {
		return fmt.Errorf("window: initial build: %w", err)
	}

	gogpuApp := gogpu.NewApp(gogpu.DefaultConfig().
		WithTitle(o.Title+" — window.Show").
		WithSize(o.Width, o.Height).
		WithContinuousRender(false)) // event-driven: 0% CPU when idle

	pw := newPlotWidget(ctx, src, fig, o, gogpuApp)

	uiApp := app.New(
		app.WithWindowProvider(gogpuApp),
		app.WithPlatformProvider(gogpuApp),
		app.WithEventSource(gogpuApp.EventSource()),
	)

	uiApp.SetRoot(pw)

	if runErr := desktop.Run(gogpuApp, uiApp); runErr != nil {
		return fmt.Errorf("window: run: %w", runErr)
	}

	return pw.err
}

// sceneProvider is satisfied by gogpu/ui's internal render.SceneCanvas, which
// records drawing commands into a scene.Scene for GPU-accelerated rendering.
//
// When plotWidget is the root widget, SetRoot calls SetRepaintBoundary(true),
// so Draw receives a SceneCanvas (not a render.Canvas). The scene.Scene is
// the GPU scene graph — drawing into it is fully GPU-accelerated via
// SDF shapes, MSDF text, and hardware path processing.
type sceneProvider interface {
	Scene() *scene.Scene
}

// plotWidget is a gogpu/ui widget that renders a ggplot figure.
//
// It implements [widget.Widget] with:
//   - Layout: fills all available space
//   - Draw: renders the figure into the GPU scene graph via [canvas.SceneCanvas]
//   - Event: translates mouse/wheel events to [output.Event] for pan/zoom
type plotWidget struct {
	widget.WidgetBase

	ctx        context.Context
	src        output.Source
	gogpuApp   *gogpu.App
	controller output.Controller

	cur   output.Figure
	state output.State
	err   error

	// Async rebuild — active only when rebuildDelay > 0.
	rebuildDelay time.Duration
	onRebuildErr func(error)

	mu          sync.Mutex    // protects pendingFig, building, dirty, buildCancel
	pendingFig  output.Figure // non-nil: a background build completed; swap on next draw
	buildCancel context.CancelFunc
	building    bool
	dirty       bool
}

func newPlotWidget(
	ctx context.Context,
	src output.Source,
	fig output.Figure,
	o Options,
	gogpuApp *gogpu.App,
) *plotWidget {
	pw := &plotWidget{
		ctx:          ctx,
		src:          src,
		gogpuApp:     gogpuApp,
		controller:   o.Controller,
		cur:          fig,
		state:        output.State{Scale: 1, Bounds: image.Rect(0, 0, o.Width, o.Height), Figure: fig},
		rebuildDelay: o.RebuildDelay,
		onRebuildErr: o.OnRebuildErr,
	}

	pw.SetVisible(true)
	pw.SetEnabled(true)

	return pw
}

// Layout fills all available space — the plot takes the entire window.
func (pw *plotWidget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Biggest()
}

// Draw renders the ggplot figure onto the widget canvas.
//
// Because plotWidget is the root widget, SetRoot calls SetRepaintBoundary(true),
// so the canvas passed here is a SceneCanvas recording into a scene.Scene —
// the GPU scene graph used by gogpu/ui's compositor pipeline.
//
// Primary path: type-assert for [sceneProvider] to get the scene.Scene.
// Create ggplot's [canvas.SceneCanvas] wrapping it and draw the figure
// directly into the GPU scene graph. All shapes (paths, fills, strokes, text)
// are recorded as GPU-accelerated scene commands (SDF shapes, MSDF text,
// hardware path processing). The compositor renders the scene into a
// per-boundary GPU texture via FlushGPUWithView. Zero bitmap copies.
//
// Fallback path: for non-scene canvases (testing, headless, or custom
// canvas implementations), create an independent [canvas.RasterCanvas],
// render, flush GPU, and blit via DrawImage.
func (pw *plotWidget) Draw(_ widget.Context, cv widget.Canvas) {
	bounds := pw.Bounds()
	width := int(bounds.Width())
	height := int(bounds.Height())

	if width <= 0 || height <= 0 {
		return
	}

	// Swap in a pending async-rebuilt figure, if any.
	pw.mu.Lock()
	if pw.pendingFig != nil {
		pw.cur = pw.pendingFig
		pw.pendingFig = nil
		pw.state.OffsetX, pw.state.OffsetY, pw.state.Scale = 0, 0, 1
	}
	pw.mu.Unlock()

	// Update bounds if window resized.
	pw.state.Bounds = image.Rect(0, 0, width, height)

	// Primary path: draw directly into the GPU scene graph.
	if sp, ok := cv.(sceneProvider); ok {
		pw.drawScene(sp.Scene(), width, height)

		return
	}

	// Fallback path: independent raster canvas + DrawImage.
	pw.drawFallback(cv, width, height)
}

// drawScene renders the figure into the GPU scene graph via [canvas.SceneCanvas].
//
// The scene.Scene is owned by gogpu/ui's boundary recording pipeline. Drawing
// into it records GPU-accelerated commands that the compositor renders via
// FlushGPUWithView into a per-boundary GPU texture. No intermediate bitmap,
// no CPU rasterization, no DrawImage.
func (pw *plotWidget) drawScene(sc *scene.Scene, width, height int) {
	c := canvas.NewSceneCanvas(sc, width, height)
	defer func() { _ = c.Close() }()

	c.Clear(color.White)

	if err := output.DrawViewport(pw.ctx, pw.cur, c, width, height, &pw.state); err != nil {
		pw.fail(err)
	}
}

// drawFallback renders the figure into an independent [canvas.RasterCanvas]
// and blits the result via DrawImage. Used when the widget.Canvas doesn't
// expose a scene.Scene (e.g. testing, headless, or render.Canvas).
func (pw *plotWidget) drawFallback(cv widget.Canvas, width, height int) {
	rc := canvas.NewRasterCanvas(width, height)
	defer func() { _ = rc.Close() }()

	rc.Clear(color.White)

	if err := output.DrawViewport(pw.ctx, pw.cur, rc, width, height, &pw.state); err != nil {
		pw.fail(err)

		return
	}

	// Flush pending GPU shapes to the CPU pixmap before reading pixels.
	// gg.Context.Image() reads from the pixmap but GPU-accelerated shapes
	// are queued in the GPU render context until FlushGPU is called.
	_ = rc.Context().FlushGPU()

	img := rc.Context().Image()
	cv.DrawImage(img, pw.Bounds().Min)
}

// Event handles mouse and wheel events for pan/zoom.
func (pw *plotWidget) Event(_ widget.Context, e event.Event) bool {
	switch ev := e.(type) {
	case *event.MouseEvent:
		return pw.handleMouse(ev)
	case *event.WheelEvent:
		return pw.handleWheel(ev)
	}

	return false
}

// handleMouse translates gogpu/ui mouse events to output.Event.
func (pw *plotWidget) handleMouse(ev *event.MouseEvent) bool {
	pos := ev.Position

	var kind output.EventKind

	switch ev.MouseType {
	case event.MousePress:
		kind = output.EventPointerDown
	case event.MouseRelease:
		kind = output.EventPointerUp
	case event.MouseMove, event.MouseDrag:
		kind = output.EventPointerMove
	case event.MouseEnter, event.MouseLeave, event.MouseDoubleClick:
		return false // Not handled yet; future tooltip/selection support.
	default:
		return false
	}

	pw.dispatch(output.Event{Kind: kind, X: float64(pos.X), Y: float64(pos.Y)})

	return true
}

// handleWheel translates gogpu/ui wheel events to output.Event.
func (pw *plotWidget) handleWheel(ev *event.WheelEvent) bool {
	pos := ev.Position

	pw.dispatch(output.Event{
		Kind: output.EventScroll,
		X:    float64(pos.X),
		Y:    float64(pos.Y),
		DX:   float64(ev.Delta.X),
		DY:   float64(ev.Delta.Y),
	})

	return true
}

// dispatch runs one event through the controller and performs the action.
func (pw *plotWidget) dispatch(ev output.Event) {
	pw.state.Figure = pw.cur

	switch pw.controller.OnEvent(ev, &pw.state) {
	case output.ActionIgnore:
	case output.ActionRedraw:
		pw.SetNeedsRedraw(true)
		pw.gogpuApp.RequestRedraw()
	case output.ActionRebuild:
		if pw.rebuildDelay > 0 {
			pw.scheduleRebuild()
		} else {
			pw.rebuildSync()
		}

		pw.SetNeedsRedraw(true)
		pw.gogpuApp.RequestRedraw()
	case output.ActionExport:
		// Export from a live window is not supported; ignore.
	case output.ActionClose:
		pw.gogpuApp.Quit()
	}
}

// Children returns nil — plotWidget is a leaf widget.
func (pw *plotWidget) Children() []widget.Widget { return nil }

// rebuildSync recomputes the figure from the source synchronously (slow path)
// and resets the viewport. On error the last good figure is kept.
func (pw *plotWidget) rebuildSync() {
	fig, err := pw.src.Build(pw.ctx)
	if err != nil {
		pw.fail(fmt.Errorf("window: rebuild: %w", err))

		return
	}

	pw.cur = fig
	pw.state.OffsetX, pw.state.OffsetY, pw.state.Scale = 0, 0, 1
}

// scheduleRebuild arms a debounced async rebuild. Rapid calls coalesce: if a
// build is already in flight, the request is recorded and replayed when the
// current build finishes.
func (pw *plotWidget) scheduleRebuild() {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.building {
		pw.dirty = true

		return
	}

	// Cancel any pending (not-yet-started) build.
	if pw.buildCancel != nil {
		pw.buildCancel()
	}

	bctx, cancel := context.WithCancel(pw.ctx)
	pw.buildCancel = cancel
	pw.building = true

	delay := pw.rebuildDelay
	src := pw.src

	go func() {
		// Debounce delay.
		select {
		case <-time.After(delay):
		case <-bctx.Done():
			pw.mu.Lock()
			pw.building = false
			pw.mu.Unlock()

			return
		}

		fig, err := src.Build(bctx)

		pw.mu.Lock()
		pw.building = false
		reschedule := pw.dirty
		pw.dirty = false

		switch {
		case err != nil && !errors.Is(err, context.Canceled):
			if pw.onRebuildErr != nil {
				pw.onRebuildErr(fmt.Errorf("window: async rebuild: %w", err))
			}
		case err == nil:
			pw.pendingFig = fig
		}

		pw.mu.Unlock()

		if err == nil {
			pw.gogpuApp.RequestRedraw()
		}

		if reschedule {
			pw.scheduleRebuild() //nolint:contextcheck // Context stored on struct (pw.ctx), not threaded through gogpu callbacks.
		}
	}()
}

// fail records the first error and asks the app to quit; Show returns it.
func (pw *plotWidget) fail(err error) {
	if pw.err == nil {
		pw.err = err
	}

	pw.gogpuApp.Quit()
}
