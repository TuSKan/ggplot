// Example violin demonstrates the geom.Violin geometry for showing
// mirrored kernel density estimates per group — like a boxplot but
// revealing the full distribution shape.
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

	// Three groups with different distributions.
	// Group 1: normal(10, 2), Group 2: normal(15, 3), Group 3: bimodal.
	n := 200 //nolint:mnd // 200 observations per group.
	xs := make([]float64, 0, n*3)
	ys := make([]float64, 0, n*3)

	// Simple deterministic pseudo-data using sine waves.
	for i := range n {
		t := float64(i) / float64(n)

		// Group 1 — narrow distribution centered at 10.
		xs = append(xs, 1)
		ys = append(ys, 10+2*math.Sin(t*math.Pi*7)+0.5*math.Cos(t*math.Pi*13)) //nolint:mnd // Synthetic waveform data.

		// Group 2 — wider distribution centered at 15.
		xs = append(xs, 2)
		ys = append(ys, 15+3*math.Sin(t*math.Pi*5)+math.Cos(t*math.Pi*11)) //nolint:mnd // Synthetic waveform data.

		// Group 3 — bimodal near 8 and 18.
		xs = append(xs, 3)
		if i%2 == 0 { //nolint:mnd // Even/odd split for bimodal distribution.
			ys = append(ys, 8+1.5*math.Sin(t*math.Pi*9)) //nolint:mnd // Synthetic mode 1.
		} else {
			ys = append(ys, 18+1.5*math.Cos(t*math.Pi*9)) //nolint:mnd // Synthetic mode 2.
		}
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("group", xs),
		eng.NewFloat64Column("value", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("group"), aes.Y("value")).
		Layer(geom.Violin(
			geom.WithFill("#9B59B6"),
			geom.WithColor("#2C3E50"),
			geom.WithAlpha(0.7),
			geom.WithWidth(0.8),
		)).
		Labs(
			ggplot.Title("Distribution Shape — Violin Plot"),
			ggplot.XLab("Group"),
			ggplot.YLab("Value"),
		)

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "violin.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
