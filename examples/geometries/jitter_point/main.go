// Example jitter_point demonstrates geom.JitterPoint — a scatter plot
// with random displacement to reduce overplotting. The jitter is
// deterministic via WithJitterSeed for reproducible output.
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
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

	// Simulate overlapping data: 5 categories × 50 points each,
	// with moderate vertical spread to create dense clusters.
	const nPerGroup = 50

	rng := rand.New(rand.NewPCG(1, 2))

	groups := make([]string, 0, 5*nPerGroup) //nolint:mnd // 5 categories for demo.
	ys := make([]float64, 0, 5*nPerGroup)    //nolint:mnd // 5 categories for demo.
	cats := []string{"A", "B", "C", "D", "E"}

	for _, cat := range cats {
		base := float64(len(groups)/nPerGroup+1) * 8 //nolint:mnd // Increasing group means.

		for range nPerGroup {
			groups = append(groups, cat)
			ys = append(ys, base+rng.NormFloat64()*2) //nolint:mnd // σ=2 spread around group mean.
		}
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewStringColumn("category", groups),
		eng.NewFloat64Column("value", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// JitterPoint with categorical X — width=0.4 matches ggplot2's default.
	p := ggplot.New(ds, aes.X("category"), aes.Y("value")).
		Layer(geom.JitterPoint(
			geom.WithJitterWidth(0.4),
			geom.WithJitterHeight(0.0),
			geom.WithJitterSeed(42),
			geom.WithColor("#2980B9"),
			geom.WithAlpha(0.6),
			geom.WithSize(4),
		)).
		Labs(
			ggplot.Title("Jitter Point — Overplotting Reduction"),
			ggplot.XLab("Category"),
			ggplot.YLab("Value"),
		)

	if err := file.Save(context.Background(), p, filepath.Join(dir, "jitter_point.png"), 800, 500); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("saved jitter_point.png")
}
