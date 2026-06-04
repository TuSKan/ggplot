// Example coord_fixed demonstrates fixed aspect ratio — 1:1 equal scaling
// so that one unit of x occupies the same pixel length as one unit of y.
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

	// Generate a unit circle for demonstration.
	n := 100

	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range n {
		theta := float64(i) / float64(n-1) * 2 * math.Pi //nolint:mnd // Full circle.
		xs[i] = math.Cos(theta)
		ys[i] = math.Sin(theta)
	}

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// Without CoordFixed — circle will be elliptical on a non-square canvas.
	noFixed := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#1F77B4"), geom.WithLineWidth(2))).
		Labels(
			ggplot.Title("No CoordFixed"),
			ggplot.Subtitle("Circle appears as an ellipse on a wide canvas"),
			ggplot.XLabel("x"),
			ggplot.YLabel("y"),
		).
		Theme("minimal")

	out := filepath.Join(dir, "no_fixed.png")
	if err := file.Save(context.Background(), noFixed, out, 800, 400); err != nil { //nolint:mnd // Wide canvas.
		log.Fatalln(err)
	}

	fmt.Println("saved", out)

	// With CoordFixed(1) — circle is a true circle regardless of canvas shape.
	fixed := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#E74C3C"), geom.WithLineWidth(2))).
		CoordFixed(1).
		Labels(
			ggplot.Title("CoordFixed(1)"),
			ggplot.Subtitle("Equal scaling — circle is a true circle"),
			ggplot.XLabel("x"),
			ggplot.YLabel("y"),
		).
		Theme("minimal")

	out2 := filepath.Join(dir, "fixed.png")
	if err := file.Save(context.Background(), fixed, out2, 800, 400); err != nil { //nolint:mnd // Wide canvas.
		log.Fatalln(err)
	}

	fmt.Println("saved", out2)
}
