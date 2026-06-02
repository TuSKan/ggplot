//go:build !js

// Example: interactive desktop window.
//
// Opens a native GPU window via output/window and presents a ggplot figure
// with pan (drag) and wheel-zoom from the default controller. window.Show
// blocks until the window is closed and must run on the main goroutine, so —
// unlike the other examples — this one needs a display/GPU and cannot run
// headless or in CI. Run it locally:
//
//	go run ./examples/window
//
// For non-interactive output (PNG/SVG/PDF, in-memory image, io.Writer, custom
// surfaces) see examples/output instead.
package main

import (
	"context"
	"log"
	"math"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	_ "github.com/TuSKan/ggplot/canvas/gpu"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/window"
)

func main() {

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	const n = 200

	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range n {
		x := float64(i) / 10
		xs[i] = x
		ys[i] = math.Sin(x) + 0.1*x
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// *Plot implements output.Source, so it can be handed to window.Show
	// directly — the window rebuilds from it when needed (e.g. on reset).
	plot := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(3), geom.WithColor("coral"))).
		Layer(geom.Smooth(geom.WithColor("steelblue"))).
		Labs(
			ggplot.Title("Interactive window — drag to pan, scroll to zoom"),
			ggplot.XLab("x"),
			ggplot.YLab("y"),
		)

	// Blocks until the window is closed. Drag pans and the wheel zooms via the
	// default pan/zoom policy; pass window.WithController to customize it.
	if err := window.Show(ctx, plot,
		window.WithTitle("ggplot — window.Show"),
		window.WithSize(900, 600),
	); err != nil {
		log.Fatalln(err)
	}
}
