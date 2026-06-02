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
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
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
	ShowFPS      bool          // show FPS counter overlay
	PprofAddr    string        // if non-empty, start pprof HTTP server on this address
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

// WithFPS enables an FPS counter overlay in the top-right corner of the window.
// Uses an exponential moving average for smooth readings.
func WithFPS() Opt { return func(o *Options) { o.ShowFPS = true } }

// WithPprof enables a pprof HTTP server on localhost:6060 for the lifetime of
// the window. While the window is open and you are interacting, capture a CPU
// profile with:
//
//	go tool pprof http://localhost:6060/debug/pprof/profile?seconds=5
//
// Or open http://localhost:6060/debug/pprof/ in a browser for all profiles.
func WithPprof() Opt { return func(o *Options) { o.PprofAddr = "localhost:6060" } }

// Show builds src into a figure and presents it in a native window, processing
// pan/zoom and other interactions until the window is closed. It blocks and
// must be called from the main goroutine.
//
// Show automatically pins the calling goroutine to its OS thread
// (runtime.LockOSThread) because Win32 and Cocoa require the window event
// loop to run on the thread that created the window.
func Show(ctx context.Context, src output.Source, opts ...Opt) error {
	// Pin this goroutine to one OS thread for the lifetime of the window.
	// Win32/Cocoa dispatch window messages only on the thread that created
	// the window; Go's scheduler would otherwise migrate this goroutine
	// between OS threads, breaking the event pump.
	runtime.LockOSThread()

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
		state:        output.State{Bounds: image.Rect(0, 0, o.Width, o.Height), Figure: fig},
		showFPS:      o.ShowFPS,
		rebuildDelay: o.RebuildDelay,
		onRebuildErr: o.OnRebuildErr,
	}

	// Optional pprof HTTP server for live profiling.
	var pprofServer *http.Server

	if o.PprofAddr != "" {
		pprofServer = startPprof(ctx, o.PprofAddr)
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

		if pprofServer != nil {
			_ = pprofServer.Close()
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

	// FPS overlay.
	showFPS      bool
	lastDrawTime time.Time
	fpsEMA       float64 // exponential moving average of FPS

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
	//
	// WORKAROUND: On Windows HiDPI, gogpu's WM_MOUSEWHEEL handler converts
	// screen→client coordinates via ScreenToClient but does NOT divide by
	// the DPI scale factor. Pointer events (WM_MOUSEMOVE etc.) DO divide by
	// scale in createPointerEvent. This mismatch means scroll event X,Y are
	// in physical pixels while panel geometry (from dc.Width()/dc.Height(),
	// which returns logical) is in logical pixels. Without correction, the
	// cursor appears outside the panel bounds on HiDPI and hit-testing fails.
	// We normalise scroll coordinates to logical here.
	if ses, ok := es.(gpucontext.ScrollEventSource); ok {
		ses.OnScrollEvent(func(ev gpucontext.ScrollEvent) {
			sx, sy := ev.X, ev.Y
			if s := app.ScaleFactor(); s > 1 {
				sx /= s
				sy /= s
			}

			ws.dispatch(output.Event{
				Kind: output.EventScroll,
				X:    sx,
				Y:    sy,
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

		if drawErr := ws.cur.Draw(ws.ctx, c, w, h); drawErr != nil {
			ws.fail(drawErr)
		}

		if ws.showFPS {
			ws.drawFPS(c, w)
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

// drawFPS renders the FPS counter overlay in the top-right corner.
func (ws *windowState) drawFPS(c canvas.Canvas, width int) {
	now := time.Now()

	if !ws.lastDrawTime.IsZero() {
		dt := now.Sub(ws.lastDrawTime).Seconds()
		if dt > 0 {
			instantFPS := 1.0 / dt

			const alpha = 0.1 // EMA smoothing factor.

			if ws.fpsEMA == 0 {
				ws.fpsEMA = instantFPS
			} else {
				ws.fpsEMA = alpha*instantFPS + (1-alpha)*ws.fpsEMA
			}
		}
	}

	ws.lastDrawTime = now

	if ws.fpsEMA == 0 {
		return // not enough data yet
	}

	text := fmt.Sprintf("%.0f FPS", ws.fpsEMA)

	const fontSize = 11 //nolint:mnd // Small, unobtrusive overlay text.

	c.Save()

	defer c.Restore()

	c.SetFontSize(fontSize)

	tw, th := c.MeasureString(text)

	const padX = 6.0 //nolint:mnd // Horizontal padding inside the FPS badge.

	const padY = 3.0 //nolint:mnd // Vertical padding inside the FPS badge.

	badgeW := tw + 2*padX
	badgeH := th + 2*padY
	bx := float64(width) - badgeW - padX
	by := padY

	// Semi-transparent dark background.
	c.SetColor(color.NRGBA{R: 0, G: 0, B: 0, A: 140}) //nolint:mnd // 55% opaque black background.
	c.DrawRectangle(bx, by, badgeW, badgeH)
	c.Fill()

	// White text, centered in badge.
	c.SetColor(color.White)
	c.DrawStringAnchored(text, bx+badgeW/2, by+badgeH/2, 0.5, 0.5) //nolint:mnd // Center anchor.
}

// startPprof starts an HTTP pprof server on addr and returns it for shutdown.
// Uses a dedicated ServeMux so it does not affect [http.DefaultServeMux].
func startPprof(ctx context.Context, addr string) *http.Server {
	// Enable block and mutex profiling so /debug/pprof/block and
	// /debug/pprof/mutex produce useful data. Rate=1 captures every
	// event; this has negligible overhead for interactive plotting.
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1) //nolint:mnd // Fraction=1 captures all mutex contention events.

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second, //nolint:mnd // Reasonable HTTP read-header timeout.
	}

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		slog.Error("pprof: listen failed", "addr", addr, "err", err)

		return nil
	}

	slog.Info("pprof server started", "addr", "http://"+ln.Addr().String()+"/debug/pprof/")

	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			slog.Error("pprof: serve failed", "err", serveErr)
		}
	}()

	return srv
}
