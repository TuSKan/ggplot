// Example: the interactive Session loop, headless.
//
// output.Session drives a Source (here a *ggplot.Plot) onto a LiveSurface,
// running the build → draw → event loop and the fast-path / slow-path policy:
//
//   - fast path (ActionRedraw): re-render the current figure under a viewport
//     transform — pan and zoom, no rebuild.
//   - slow path (ActionRebuild): call Source.Build again when an interaction
//     crosses the trained data extent (scales retrain, stats recompute).
//     WithRebuildDelay makes this asynchronous and debounced.
//
// output/window wires this same Session policy to a real GPU window (see
// examples/window). Here we use a headless LiveSurface that scripts a sequence
// of events and writes each committed frame to a PNG, so the loop is fully
// runnable without a display:
//
//	go run ./examples/session
package main

import (
	"context"
	"fmt"
	"image"
	"log"
	"math"
	"path/filepath"
	"runtime"
	"time"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output"
)

func main() {
	ctx := context.Background()

	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)

	plot := buildPlot(ctx)

	// A frame-capturing LiveSurface stands in for a window: every committed
	// frame is written to dir as frame_NN.png.
	surf := newFilmstrip(720, 450, dir)

	// A controller turns each event into a Session action. This one adds
	// "rebuild on the 'r' key" on top of the built-in pan/zoom policy. The base
	// controller is created once (it holds drag state across events).
	base := output.DefaultController()
	ctrl := output.ControllerFunc(func(ev output.Event, st *output.State) output.Action {
		if ev.Kind == output.EventKey && ev.Key == "r" {
			return output.ActionRebuild
		}

		return base.OnEvent(ev, st)
	})

	sess := output.NewSession(plot, surf,
		output.WithController(ctrl),
		// Async, debounced rebuild: bursts of 'r' within 30ms collapse into one
		// background Build; the last good frame keeps drawing until it lands.
		output.WithRebuildDelay(30*time.Millisecond),
		output.WithRebuildError(func(err error) { log.Println("rebuild:", err) }),
	)

	// Script an interaction: drag to pan, scroll to zoom, then a burst of
	// rebuilds. In a real backend these come from the OS/DOM instead. We end by
	// closing the event channel (not an EventClose) so the session flushes the
	// pending debounced rebuild before returning — EventClose would instead end
	// immediately and drop it.
	go surf.script(
		output.Event{Kind: output.EventPointerDown, X: 360, Y: 225},
		output.Event{Kind: output.EventPointerMove, X: 400, Y: 245}, // pan → redraw
		output.Event{Kind: output.EventPointerUp, X: 400, Y: 245},
		output.Event{Kind: output.EventScroll, X: 400, Y: 245, DY: 1}, // zoom in → redraw
		output.Event{Kind: output.EventKey, Key: "r"},                 // rebuild (debounced)
		output.Event{Kind: output.EventKey, Key: "r"},                 // coalesced
		output.Event{Kind: output.EventKey, Key: "r"},                 // coalesced
	)

	// Run blocks until the surface's event channel closes (or a controller
	// returns ActionClose / an EventClose arrives, or ctx is cancelled).
	if err := sess.Run(ctx); err != nil {
		log.Fatalln(err)
	}

	log.Printf("session ran: %d frames written to %s", surf.frames, dir)
}

func buildPlot(ctx context.Context) *ggplot.Plot {
	eng := memory.NewEngine(ctx)

	const n = 120

	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range n {
		x := float64(i) / 6
		xs[i] = x
		ys[i] = math.Sin(x) + 0.15*x
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	return ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(3), geom.WithColor("coral"))).
		Layer(geom.Smooth(geom.WithColor("steelblue"))).
		Labs(ggplot.Title("Session loop — pan, zoom, rebuild"), ggplot.XLab("x"), ggplot.YLab("y"))
}

// filmstrip is a headless [output.LiveSurface]: tests/examples push events onto
// it, and each Commit writes the frame to a numbered PNG. A real backend
// (output/window, output/web) instead presents to a GPU surface and feeds
// Events() from the platform.
type filmstrip struct {
	w, h   int
	dir    string
	events chan output.Event
	cv     *canvas.RasterCanvas
	frames int
}

func newFilmstrip(w, h int, dir string) *filmstrip {
	return &filmstrip{
		w:      w,
		h:      h,
		dir:    dir,
		events: make(chan output.Event, 16),
	}
}

// Acquire returns the canvas for the next frame.
func (f *filmstrip) Acquire(_ context.Context) (canvas.Canvas, error) {
	if f.cv != nil {
		_ = f.cv.Close()
	}

	f.cv = canvas.NewRasterCanvasCPU(f.w, f.h)

	return f.cv, nil
}

// Commit writes the acquired frame to disk, standing in for a GPU present.
func (f *filmstrip) Commit(_ context.Context) error {
	path := filepath.Join(f.dir, fmt.Sprintf("frame_%02d.png", f.frames))
	if err := f.cv.SavePNG(path); err != nil {
		return fmt.Errorf("filmstrip: save frame: %w", err)
	}

	f.frames++

	return nil
}

func (f *filmstrip) Bounds() image.Rectangle     { return image.Rect(0, 0, f.w, f.h) }
func (f *filmstrip) Events() <-chan output.Event { return f.events }

func (f *filmstrip) Close() error {
	if f.cv != nil {
		return f.cv.Close() //nolint:wrapcheck // example surface.
	}

	return nil
}

// script pushes events in order, with a brief pause so the debounced rebuild
// timer fires before the closing event, then closes the channel.
func (f *filmstrip) script(events ...output.Event) {
	for _, ev := range events {
		f.events <- ev

		if ev.Kind == output.EventKey {
			time.Sleep(10 * time.Millisecond)
		}
	}

	time.Sleep(60 * time.Millisecond) // let the final rebuild land
	close(f.events)
}
