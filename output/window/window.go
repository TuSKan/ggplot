//go:build !js

// Package window presents a ggplot figure in a native desktop GPU window via
// gogpu, with zero-copy presentation through gg's ggcanvas integration. It
// reuses the platform-neutral interaction policy from package output
// ([output.Controller] / [output.State]); only the frame loop is platform
// specific.
//
// gogpu owns the run loop (callback-driven), so this backend does not use
// [output.Session] (a pull/channel loop). Instead [Show] wires gogpu's draw and
// input callbacks to the same controller and viewport policy.
//
// [Show] blocks until the window closes and must be called from the main
// goroutine (GPU/windowing requires the main OS thread).
package window

import (
	"context"
	"fmt"
	"image"
	"image/color"

	"github.com/gogpu/gg/integration/ggcanvas"
	"github.com/gogpu/gogpu"
	"github.com/gogpu/gpucontext"

	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/output"
)

const (
	defaultWidth  = 800
	defaultHeight = 600
)

// Options configures a window opened by [Show].
type Options struct {
	Title      string
	Width      int
	Height     int
	Controller output.Controller
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

	w := &windowLoop{
		ctx:        ctx,
		src:        src,
		controller: o.Controller,
		cur:        fig,
		state:      output.State{Scale: 1, Bounds: image.Rect(0, 0, o.Width, o.Height), Figure: fig},
	}

	cfg := gogpu.DefaultConfig().
		WithTitle(o.Title).
		WithSize(o.Width, o.Height).
		WithContinuousRender(false)

	app := gogpu.NewApp(cfg)
	w.app = app

	app.OnDraw(w.onDraw)
	app.OnResize(w.onResize)
	app.OnClose(w.onClose)
	w.bindInput(app.EventSource())

	if runErr := app.Run(); runErr != nil {
		return fmt.Errorf("window: run: %w", runErr)
	}

	w.closeCanvas()

	return w.err
}

// windowLoop holds the per-window state shared across gogpu callbacks. gogpu
// dispatches all callbacks on the run-loop (main) goroutine, so no
// synchronization is needed between them.
type windowLoop struct {
	ctx        context.Context
	src        output.Source
	app        *gogpu.App
	controller output.Controller

	ggc *ggcanvas.Canvas
	cur output.Figure

	state        output.State
	lastX, lastY float64 // last pointer position, used as the scroll/zoom anchor
	err          error
}

// onDraw renders the current figure (under the viewport transform) into the
// gg canvas and presents it zero-copy onto the window surface.
func (w *windowLoop) onDraw(dc *gogpu.Context) {
	width, height := dc.Size()

	if w.ggc == nil {
		ggc, err := ggcanvas.New(w.app.GPUContextProvider(), width, height)
		if err != nil {
			w.fail(fmt.Errorf("window: create canvas: %w", err))

			return
		}

		w.ggc = ggc
	}

	rc := canvas.RasterFromContext(w.ggc.Context()) // borrowed — must not Close.
	rc.Clear(color.White)

	if err := output.DrawViewport(w.ctx, w.cur, rc, width, height, &w.state); err != nil {
		w.fail(err)

		return
	}

	w.ggc.MarkDirty()

	if err := w.ggc.Render(dc.RenderTarget()); err != nil {
		w.fail(fmt.Errorf("window: present: %w", err))
	}
}

func (w *windowLoop) onResize(width, height int) {
	if w.ggc != nil {
		_ = w.ggc.Resize(width, height)
	}

	w.state.Bounds = image.Rect(0, 0, width, height)
	w.app.RequestRedraw()
}

func (w *windowLoop) onClose() { w.closeCanvas() }

func (w *windowLoop) closeCanvas() {
	if w.ggc != nil {
		_ = w.ggc.Close()
		w.ggc = nil
	}
}

// bindInput translates gogpu input callbacks into platform-neutral
// [output.Event]s and feeds them to the controller.
func (w *windowLoop) bindInput(es gpucontext.EventSource) {
	es.OnMouseMove(func(x, y float64) {
		w.lastX, w.lastY = x, y
		w.dispatch(output.Event{Kind: output.EventPointerMove, X: x, Y: y})
	})
	es.OnMousePress(func(_ gpucontext.MouseButton, x, y float64) {
		w.lastX, w.lastY = x, y
		w.dispatch(output.Event{Kind: output.EventPointerDown, X: x, Y: y})
	})
	es.OnMouseRelease(func(_ gpucontext.MouseButton, x, y float64) {
		w.dispatch(output.Event{Kind: output.EventPointerUp, X: x, Y: y})
	})
	es.OnScroll(func(dx, dy float64) {
		// Scroll carries only deltas; anchor zoom at the last pointer position.
		w.dispatch(output.Event{Kind: output.EventScroll, X: w.lastX, Y: w.lastY, DX: dx, DY: dy})
	})
}

// dispatch runs one event through the controller and performs the action.
func (w *windowLoop) dispatch(ev output.Event) {
	w.state.Figure = w.cur

	switch w.controller.OnEvent(ev, &w.state) {
	case output.ActionIgnore:
	case output.ActionRedraw:
		w.app.RequestRedraw()
	case output.ActionRebuild:
		w.rebuild()
		w.app.RequestRedraw()
	case output.ActionExport:
		// Export from a live window is not supported; ignore.
	case output.ActionClose:
		w.app.Quit()
	}
}

// rebuild recomputes the figure from the source (slow path) and resets the
// viewport. On error the last good figure is kept.
func (w *windowLoop) rebuild() {
	fig, err := w.src.Build(w.ctx)
	if err != nil {
		w.fail(fmt.Errorf("window: rebuild: %w", err))

		return
	}

	w.cur = fig
	w.state.OffsetX, w.state.OffsetY, w.state.Scale = 0, 0, 1
}

// fail records the first error and asks the app to quit; Show returns it.
func (w *windowLoop) fail(err error) {
	if w.err == nil {
		w.err = err
	}

	w.app.Quit()
}
