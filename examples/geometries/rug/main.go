// Example rug demonstrates the geom.Rug geometry.
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
	"github.com/TuSKan/ggplot/output"
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	rng := rand.New(rand.NewSource(42)) //nolint:mnd // Example uses deterministic seed.
	n := 80
	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range n {
		xs[i] = rng.Float64() * 10
		ys[i] = math.Sin(xs[i]) + rng.NormFloat64()*0.3
	}

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(2.5), geom.WithColor("#8E44AD"), geom.WithAlpha(0.6))).
		Layer(geom.Rug(geom.WithAlpha(0.4), geom.WithColor("#8E44AD"))).
		Labs(ggplot.Title("Rug Plot"), ggplot.Subtitle("Marginal rug marks on scatter"))

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "rug.png"), 800, 600, output.WithCPU(true)); err != nil {
		log.Fatalln(err)
	}
}
