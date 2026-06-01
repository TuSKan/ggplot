//go:build !js

// Package window presents a ggplot figure in a native desktop GPU window via
// gogpu + ggcanvas, the zero-copy GPU rendering integration. Drawing goes
// through gg.Context (GPU-accelerated), and ggcanvas.Canvas presents via
// RenderDirect (zero-copy to surface) with automatic fallback to CPU upload.
//
// Pan (mouse drag) and zoom (mouse wheel) are handled through gogpu's event
// system, translated to ggplot's platform-neutral [output.Controller].
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

	"github.com/gogpu/gg"
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
	o := Options{Title: "ggplot", Width: defaultWidth, Height: defaultHeight, Controller: output.DataSpaceController()}
	for _, opt := range opts {
		opt(&o)
	}

	fig, err := src.Build(ctx)
	if err != nil {
		return fmt.Errorf("window: initial build: %w", err)
	}

	app := gogpu.NewApp(gogpu.DefaultConfig().
		WithTitle(o.Title).
		WithSize(o.Width, o.Height).
		WithContinuousRender(false)) // event-driven: 0% CPU when idle

	ws := &windowState{
		ctx:          ctx,
		src:          src,
		app:          app,
		controller:   o.Controller,
		cur:          fig,
		state:        output.State{Scale: 1, Bounds: image.Rect(0, 0, o.Width, o.Height), Figure: fig},
		rebuildDelay: o.RebuildDelay,
		onRebuildErr: o.OnRebuildErr,
	}

	// Register input event handlers before Run() starts the main loop.
	ws.registerEvents(app) //nolint:contextcheck // Context stored on struct (ws.ctx), not threaded through gogpu callbacks.

	// Set up the draw callback.
	app.OnDraw(func(dc *gogpu.Context) {
		ws.draw(dc)
	})

	// Close ggcanvas on shutdown (while GPU is still alive).
	app.OnClose(func() {
		if ws.cv != nil {
			_ = ws.cv.Close()
			ws.cv = nil
		}
	})

	if runErr := app.Run(); runErr != nil {
		return fmt.Errorf("window: run: %w", runErr)
	}

	return ws.err
}

// windowState holds all mutable state for the interactive window.
type windowState struct {
	ctx        context.Context
	src        output.Source
	app        *gogpu.App
	controller output.Controller

	cur   output.Figure
	state output.State
	err   error

	// ggcanvas — created lazily on first draw frame.
	cv    *ggcanvas.Canvas
	lastW int
	lastH int

	// Mouse tracking for pan.
	mouseX float64
	mouseY float64

	// Double-click detection.
	lastClickTime time.Time

	// Async rebuild — active only when rebuildDelay > 0.
	rebuildDelay time.Duration
	onRebuildErr func(error)

	mu          sync.Mutex    // protects pendingFig, building, dirty, buildCancel
	pendingFig  output.Figure // non-nil: a background build completed; swap on next draw
	buildCancel context.CancelFunc
	building    bool
	dirty       bool
}

// registerEvents hooks the gogpu event system for mouse/scroll input.
func (ws *windowState) registerEvents(app *gogpu.App) {
	es := app.EventSource()

	es.OnMousePress(func(_ gpucontext.MouseButton, x, y float64) {
		ws.mouseX, ws.mouseY = x, y

		// Double-click detection: two clicks within 400ms.
		now := time.Now()
		if now.Sub(ws.lastClickTime) < 400*time.Millisecond { //nolint:mnd // Standard double-click threshold.
			ws.dispatch(output.Event{Kind: output.EventDoubleClick, X: x, Y: y})
			ws.lastClickTime = time.Time{} // reset to avoid triple-click
		} else {
			ws.dispatch(output.Event{Kind: output.EventPointerDown, X: x, Y: y})
			ws.lastClickTime = now
		}
	})

	es.OnMouseRelease(func(_ gpucontext.MouseButton, x, y float64) {
		ws.mouseX, ws.mouseY = x, y
		ws.dispatch(output.Event{Kind: output.EventPointerUp, X: x, Y: y})
	})

	es.OnMouseMove(func(x, y float64) {
		ws.mouseX, ws.mouseY = x, y
		ws.dispatch(output.Event{Kind: output.EventPointerMove, X: x, Y: y})
	})

	// Use detailed ScrollEvent when available (provides cursor position at
	// scroll time). Fall back to basic OnScroll + tracked cursor position.
	if ses, ok := es.(gpucontext.ScrollEventSource); ok {
		ses.OnScrollEvent(func(ev gpucontext.ScrollEvent) {
			ws.dispatch(output.Event{
				Kind: output.EventScroll,
				X:    ev.X,
				Y:    ev.Y,
				DX:   ev.DeltaX,
				DY:   ev.DeltaY,
			})
		})
	} else {
		es.OnScroll(func(dx, dy float64) {
			ws.dispatch(output.Event{
				Kind: output.EventScroll,
				X:    ws.mouseX,
				Y:    ws.mouseY,
				DX:   dx,
				DY:   dy,
			})
		})
	}
}

// draw is the OnDraw callback — runs on the render thread each frame.
func (ws *windowState) draw(dc *gogpu.Context) {
	w, h := dc.Width(), dc.Height()
	if w <= 0 || h <= 0 {
		return
	}

	// Swap in a pending async-rebuilt figure, if any.
	ws.mu.Lock()
	if ws.pendingFig != nil {
		ws.cur = ws.pendingFig
		ws.pendingFig = nil
		ws.state.OffsetX, ws.state.OffsetY, ws.state.Scale = 0, 0, 1
	}
	ws.mu.Unlock()

	// Update bounds on resize.
	ws.state.Bounds = image.Rect(0, 0, w, h)

	// Lazy init or resize the ggcanvas.
	if ws.cv == nil {
		provider := ws.app.GPUContextProvider()
		if provider == nil {
			return // GPU not ready yet; will retry next frame.
		}

		cv, err := ggcanvas.New(provider, w, h)
		if err != nil {
			ws.fail(fmt.Errorf("window: ggcanvas.New: %w", err))

			return
		}

		ws.cv = cv
		ws.lastW, ws.lastH = w, h
	} else if w != ws.lastW || h != ws.lastH {
		if err := ws.cv.Resize(w, h); err != nil {
			ws.fail(fmt.Errorf("window: ggcanvas.Resize: %w", err))

			return
		}

		ws.lastW, ws.lastH = w, h
	}

	// Draw the figure into gg.Context via ggcanvas.
	if err := ws.cv.Draw(func(cc *gg.Context) {
		c := canvas.RasterFromContext(cc)
		c.Clear(color.White)

		if drawErr := output.DrawViewport(ws.ctx, ws.cur, c, w, h, &ws.state); drawErr != nil {
			ws.fail(drawErr)
		}
	}); err != nil {
		ws.fail(fmt.Errorf("window: ggcanvas.Draw: %w", err))

		return
	}

	// Present: zero-copy GPU-direct via RenderDirect, or universal fallback.
	if err := ws.cv.Render(dc.RenderTarget()); err != nil {
		ws.fail(fmt.Errorf("window: ggcanvas.Render: %w", err))
	}
}

// dispatch runs one event through the controller and performs the action.
func (ws *windowState) dispatch(ev output.Event) {
	ws.state.Figure = ws.cur

	switch ws.controller.OnEvent(ev, &ws.state) {
	case output.ActionIgnore:
	case output.ActionRedraw:
		ws.app.RequestRedraw()
	case output.ActionRebuild:
		if ws.rebuildDelay > 0 {
			ws.scheduleRebuild()
		} else {
			ws.rebuildSync()
		}

		ws.app.RequestRedraw()
	case output.ActionExport:
		// Export from a live window is not supported; ignore.
	case output.ActionClose:
		ws.app.Quit()
	}
}

// rebuildSync recomputes the figure from the source synchronously (slow path)
// and resets the viewport. On error the last good figure is kept.
func (ws *windowState) rebuildSync() {
	fig, err := ws.src.Build(ws.ctx)
	if err != nil {
		ws.fail(fmt.Errorf("window: rebuild: %w", err))

		return
	}

	ws.cur = fig
	ws.state.OffsetX, ws.state.OffsetY, ws.state.Scale = 0, 0, 1
}

// scheduleRebuild arms a debounced async rebuild. Rapid calls coalesce: if a
// build is already in flight, the request is recorded and replayed when the
// current build finishes.
func (ws *windowState) scheduleRebuild() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.building {
		ws.dirty = true

		return
	}

	// Cancel any pending (not-yet-started) build.
	if ws.buildCancel != nil {
		ws.buildCancel()
	}

	bctx, cancel := context.WithCancel(ws.ctx)
	ws.buildCancel = cancel
	ws.building = true

	delay := ws.rebuildDelay
	src := ws.src

	go func() {
		// Debounce delay.
		select {
		case <-time.After(delay):
		case <-bctx.Done():
			ws.mu.Lock()
			ws.building = false
			ws.mu.Unlock()

			return
		}

		fig, err := src.Build(bctx)

		ws.mu.Lock()
		ws.building = false
		reschedule := ws.dirty
		ws.dirty = false

		switch {
		case err != nil && !errors.Is(err, context.Canceled):
			if ws.onRebuildErr != nil {
				ws.onRebuildErr(fmt.Errorf("window: async rebuild: %w", err))
			}
		case err == nil:
			ws.pendingFig = fig
		}

		ws.mu.Unlock()

		if err == nil {
			ws.app.RequestRedraw()
		}

		if reschedule {
			ws.scheduleRebuild() //nolint:contextcheck // Context stored on struct (ws.ctx), not threaded through gogpu callbacks.
		}
	}()
}

// fail records the first error and asks the app to quit; Show returns it.
func (ws *windowState) fail(err error) {
	if ws.err == nil {
		ws.err = err
	}

	ws.app.Quit()
}
