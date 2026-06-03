//go:build js && wasm

package web

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"syscall/js"
	"time"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/integration/ggcanvas"
	"github.com/gogpu/gogpu"
	"github.com/gogpu/gpucontext"

	pcanvas "github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/output"
)

// WithGPU enables WebGPU-accelerated rendering via gogpu + ggcanvas. The
// figure is drawn into a gg.Context (GPU-accelerated SDF shapes, MSDF text)
// and presented zero-copy to the browser's WebGPU surface.
//
// This requires a browser with WebGPU support (Chrome 113+, Edge 113+,
// Firefox behind flag). If WebGPU is unavailable at runtime, Mount returns
// an error.
func WithGPU() Opt { return func(o *Options) { o.GPU = true } }

// mountGPU creates a gogpu.App with a browser canvas, sets up the draw
// callback via ggcanvas, and runs the gogpu event loop. Like output/window,
// gogpu owns the event loop — we wire draw + input callbacks and dispatch
// through the output.Controller.
func mountGPU(ctx context.Context, src output.Source, fig output.Figure, _ js.Value, o Options) error {
	app := gogpu.NewApp(gogpu.DefaultConfig().
		WithTitle("ggplot").
		WithSize(o.Width, o.Height).
		WithContinuousRender(false)) // event-driven: 0% CPU when idle

	ws := &gpuState{
		ctx:        ctx,
		src:        src,
		app:        app,
		controller: o.Controller,
		cur:        fig,
		state:      output.State{Bounds: image.Rect(0, 0, o.Width, o.Height), Figure: fig},
		drawErr:    make(chan error, 1),
	}

	// Register input event handlers.
	ws.registerEvents(app)

	// Draw callback: ggcanvas lazy-init + figure render + present.
	app.OnDraw(func(dc *gogpu.Context) {
		ws.draw(dc)
	})

	// Cleanup on close.
	app.OnClose(func() {
		if ws.cv != nil {
			_ = ws.cv.Close()
			ws.cv = nil
		}
	})

	// Run app.Run() in a goroutine — it blocks on the gogpu event loop.
	appDone := make(chan error, 1)
	go func() {
		appDone <- app.Run()
	}()

	// Wait for: draw panic, context cancel (mode switch), or app exit.
	select {
	case err := <-ws.drawErr:
		// Draw callback panicked. Don't call app.Quit() synchronously —
		// schedule it for after we return to let gogpu unwind cleanly.
		go app.Quit()

		return err

	case <-ctx.Done():
		// Mode switching — cancel is normal.
		app.Quit()
		<-appDone // Wait for Run to return.

		return nil

	case err := <-appDone:
		// app.Run() returned — could be from draw panic's go app.Quit().
		if ws.err != nil {
			return ws.err
		}

		if err != nil && ctx.Err() == nil {
			return fmt.Errorf("web: gpu: %w", err)
		}

		return nil
	}
}

// gpuState holds mutable state for the WebGPU-accelerated browser surface.
// Mirrors output/window's windowState but simplified for the browser.
type gpuState struct {
	ctx        context.Context
	src        output.Source
	app        *gogpu.App
	controller output.Controller

	cur   output.Figure
	state output.State
	err   error

	// drawErr signals draw-callback panics to mountGPU without calling
	// app.Quit() inside the callback (which kills the WASM runtime).
	drawErr chan error
	failed  bool // set after first draw panic — prevents retries

	// ggcanvas — created lazily on first draw frame.
	cv    *ggcanvas.Canvas
	lastW int
	lastH int

	// Mouse tracking for pan.
	mouseX float64
	mouseY float64

	// Double-click detection.
	lastClickTime time.Time
}

// registerEvents hooks the gogpu event system for mouse/scroll input.
func (ws *gpuState) registerEvents(app *gogpu.App) {
	es := app.EventSource()

	es.OnMousePress(func(_ gpucontext.MouseButton, x, y float64) {
		ws.mouseX, ws.mouseY = x, y

		now := time.Now()
		if now.Sub(ws.lastClickTime) < 400*time.Millisecond { //nolint:mnd // Standard double-click threshold.
			ws.dispatch(output.Event{Kind: output.EventDoubleClick, X: x, Y: y})
			ws.lastClickTime = time.Time{}
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

// draw is the OnDraw callback — runs each frame.
// The defer/recover guards against panics in upstream wgpu/ggcanvas code (e.g.,
// CreateTexture returning nil in the browser WebGPU backend) that we cannot fix
// from ggplot. CRITICAL: we must NOT call app.Quit() inside the draw callback —
// doing so kills the WASM runtime before mountGPU can handle the error.
func (ws *gpuState) draw(dc *gogpu.Context) {
	// After a panic, all future draw calls are no-ops.
	if ws.failed {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			ws.failed = true
			ws.err = fmt.Errorf("web: gpu: draw panic: %v", r)

			// Signal the error to mountGPU via channel (non-blocking).
			select {
			case ws.drawErr <- ws.err:
			default:
			}

			// Schedule app.Quit() on a new goroutine so app.Run() returns
			// on the next event cycle. We must NOT call Quit() synchronously
			// inside the draw callback — it corrupts the WASM call stack.
			go ws.app.Quit()

			// Log to browser console so the user sees the error.
			js.Global().Get("console").Call("error",
				fmt.Sprintf("[ggplot] GPU draw panic: %v", r))
		}
	}()

	w, h := dc.Width(), dc.Height()
	if w <= 0 || h <= 0 {
		return
	}

	ws.state.Bounds = image.Rect(0, 0, w, h)

	// Lazy init or resize the ggcanvas.
	if ws.cv == nil {
		provider := ws.app.GPUContextProvider()
		if provider == nil {
			return // GPU not ready yet; will retry next frame.
		}

		cv, err := ggcanvas.New(provider, w, h)
		if err != nil {
			ws.fail(fmt.Errorf("web: gpu: ggcanvas.New: %w", err))

			return
		}

		ws.cv = cv
		ws.lastW, ws.lastH = w, h
	} else if w != ws.lastW || h != ws.lastH {
		if err := ws.cv.Resize(w, h); err != nil {
			ws.fail(fmt.Errorf("web: gpu: ggcanvas.Resize: %w", err))

			return
		}

		ws.lastW, ws.lastH = w, h
	}

	// Draw the figure into gg.Context via ggcanvas.
	if err := ws.cv.Draw(func(cc *gg.Context) {
		c := pcanvas.RasterFromContext(cc)
		c.Clear(color.White)

		if drawErr := ws.cur.Draw(ws.ctx, c, w, h); drawErr != nil {
			ws.fail(drawErr)
		}
	}); err != nil {
		ws.fail(fmt.Errorf("web: gpu: ggcanvas.Draw: %w", err))

		return
	}

	// Present: zero-copy GPU-direct via RenderDirect, or universal fallback.
	if err := ws.cv.Render(dc.RenderTarget()); err != nil {
		ws.fail(fmt.Errorf("web: gpu: ggcanvas.Render: %w", err))
	}
}

// dispatch runs one event through the controller and performs the action.
func (ws *gpuState) dispatch(ev output.Event) {
	ws.state.Figure = ws.cur

	switch ws.controller.OnEvent(ev, &ws.state) {
	case output.ActionIgnore:
	case output.ActionRedraw:
		ws.app.RequestRedraw()
	case output.ActionRebuild:
		ws.rebuildSync()
		ws.app.RequestRedraw()
	case output.ActionExport:
	case output.ActionClose:
		ws.app.Quit()
	}
}

// rebuildSync recomputes the figure from the source synchronously.
func (ws *gpuState) rebuildSync() {
	fig, err := ws.src.Build(ws.ctx)
	if err != nil {
		ws.fail(fmt.Errorf("web: gpu: rebuild: %w", err))

		return
	}

	ws.cur = fig
}

// fail records the first error and asks the app to quit.
func (ws *gpuState) fail(err error) {
	if ws.err == nil {
		ws.err = err
	}

	ws.app.Quit()
}
