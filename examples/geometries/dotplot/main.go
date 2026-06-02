// Example dotplot demonstrates the geom.Dotplot geometry for showing
// individual observations as stacked dots within bins.
package main

import (
	"context"
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
	eng := memory.NewEngine(context.Background())

	// Small dataset — each observation gets its own dot.
	n := 40 //nolint:mnd // 40 observations for a small dot plot.
	xs := make([]float64, n)

	for i := range n {
		t := float64(i) / float64(n)
		// Cluster around 3 and 7 with some spread.
		if i%3 == 0 { //nolint:mnd // Every third point in the upper cluster.
			xs[i] = 7 + 1.5*math.Sin(t*math.Pi*11) //nolint:mnd // Upper cluster center.
		} else {
			xs[i] = 3 + 1.2*math.Cos(t*math.Pi*13) //nolint:mnd // Lower cluster center.
		}
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Dotplot(
			geom.WithFill("#E67E22"),
			geom.WithColor("#2C3E50"),
			geom.WithSize(4),
			geom.WithAlpha(0.9),
		)).
		Labs(
			ggplot.Title("Observation Distribution — Dot Plot"),
			ggplot.XLab("Value"),
			ggplot.YLab("Count"),
		)

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "dotplot.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
