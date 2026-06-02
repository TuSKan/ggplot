// Example coord_trans demonstrates post-stat axis transforms.
// The histogram is computed in linear space, but the y-axis is
// displayed with a sqrt transform for better readability of
// skewed count distributions.
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
	"github.com/TuSKan/ggplot/coord"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// Generate exponentially distributed data — highly skewed counts.
	n := 1000

	xs := make([]float64, n)
	for i := range n {
		// Exponential distribution via inverse CDF.
		u := (float64(i) + 0.5) / float64(n) //nolint:mnd // Uniform spacing.
		xs[i] = -2 * math.Log(1-u)           //nolint:mnd // Lambda = 0.5.
	}

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// Histogram with sqrt y-axis transform — bins are computed in linear
	// space, but the y-axis is sqrt-transformed for better visibility of
	// small counts alongside large counts.
	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Histogram(geom.WithColor("#2196F3"), geom.WithAlpha(0.7))).
		CoordTrans(coord.TransIdentity, coord.TransSqrt).
		Labs(
			ggplot.Title("Histogram with sqrt(y) Transform"),
			ggplot.Subtitle("Stats computed in linear space, y-axis sqrt-transformed for display"),
			ggplot.XLab("Value"),
			ggplot.YLab("Count (sqrt scale)"),
		).
		Theme("minimal")

	out := filepath.Join(dir, "sqrt_y.png")
	if err := file.Save(context.Background(), p, out, 800, 400); err != nil { //nolint:mnd // Canvas size.
		log.Fatalln(err)
	}

	fmt.Println("saved", out)

	// Log10 x-axis transform — useful for data spanning orders of magnitude.
	p2 := ggplot.New(ds, aes.X("x")).
		Layer(geom.Histogram(geom.WithColor("#E74C3C"), geom.WithAlpha(0.7))).
		CoordTrans(coord.TransLog10, coord.TransIdentity).
		Labs(
			ggplot.Title("Histogram with log10(x) Transform"),
			ggplot.Subtitle("Bins in linear space, x-axis log10-transformed"),
			ggplot.XLab("Value (log10 scale)"),
			ggplot.YLab("Count"),
		).
		Theme("minimal")

	out2 := filepath.Join(dir, "log10_x.png")
	if err := file.Save(context.Background(), p2, out2, 800, 400); err != nil { //nolint:mnd // Canvas size.
		log.Fatalln(err)
	}

	fmt.Println("saved", out2)
}
