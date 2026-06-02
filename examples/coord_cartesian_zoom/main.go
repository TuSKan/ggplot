// Example coord_cartesian_zoom demonstrates viewport zoom without data clipping.
// All data participates in stat computations — only the visible window changes.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// Generate a sine wave with points — we will zoom into the first peak.
	n := 200

	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range n {
		t := float64(i) / float64(n-1) * 4 * math.Pi //nolint:mnd // 2 full periods.
		xs[i] = t
		ys[i] = math.Sin(t)
	}

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// Full view — no zoom.
	full := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#1F77B4"), geom.WithLineWidth(2))).
		Layer(geom.Point(geom.WithColor("#E74C3C"), geom.WithSize(2), geom.WithAlpha(0.5))).
		Labs(
			ggplot.Title("Full View"),
			ggplot.Subtitle("All data visible"),
			ggplot.XLab("x (radians)"),
			ggplot.YLab("sin(x)"),
		).
		Theme("minimal")

	out := filepath.Join(dir, "full.png")
	if err := file.Save(context.Background(), full, out, 800, 400); err != nil { //nolint:mnd // Canvas size.
		log.Fatalln(err)
	}

	fmt.Println("saved", out)

	// Zoomed view — zoom into the first peak region.
	// Data outside the window is NOT clipped — stats and aesthetics are computed
	// from the full dataset; only the viewport changes.
	zoomed := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#1F77B4"), geom.WithLineWidth(2))).
		Layer(geom.Point(geom.WithColor("#E74C3C"), geom.WithSize(3), geom.WithAlpha(0.6))).
		CoordCartesian(0, math.Pi, 0.5, 1.1). //nolint:mnd // Zoom into the first peak.
		Labs(
			ggplot.Title("Zoomed View — CoordCartesian"),
			ggplot.Subtitle("Viewport zoom into the first peak (data NOT clipped)"),
			ggplot.XLab("x (radians)"),
			ggplot.YLab("sin(x)"),
		).
		Theme("minimal")

	out2 := filepath.Join(dir, "zoomed.png")
	if err := file.Save(context.Background(), zoomed, out2, 800, 400); err != nil { //nolint:mnd // Canvas size.
		log.Fatalln(err)
	}

	fmt.Println("saved", out2)
}
