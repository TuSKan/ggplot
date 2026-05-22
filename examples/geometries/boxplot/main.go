// Example boxplot demonstrates the geom.Boxplot geometry.
package main

import (
	"context"
	"log"
	"math"
	"math/rand"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
)

func main() {
	rng := rand.New(rand.NewSource(42)) //nolint:mnd // Example uses deterministic seed.

	means := []float64{50, 65, 55}
	xs := make([]float64, 0, 50*len(means)) //nolint:mnd // 50 samples per group.
	ys := make([]float64, 0, 50*len(means)) //nolint:mnd // 50 samples per group.

	for g, m := range means {
		for range 50 {
			xs = append(xs, float64(g+1))
			ys = append(ys, math.Max(0, m+rng.NormFloat64()*10))
		}
	}

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("group", xs),
		eng.NewFloat64Column("score", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("group"), aes.Y("score")).
		Layer(geom.Boxplot(geom.WithFill("#E8E8E8"), geom.WithColor("#2C3E50"), geom.WithWidth(0.6))).
		Labs(ggplot.Title("Boxplot"), ggplot.Subtitle("Three treatment groups"))

	_, filename, _, _ := runtime.Caller(0)
	if err := p.Save(context.Background(), filepath.Join(filepath.Dir(filename), "boxplot.png"), 800, 600, ggplot.WithCPU()); err != nil {
		log.Fatalln(err)
	}
}
