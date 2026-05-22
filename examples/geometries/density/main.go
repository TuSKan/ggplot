// Example density demonstrates the geom.Density geometry.
package main

import (
	"context"
	"log"
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
	rng := rand.New(rand.NewSource(42)) //nolint:gosec // Example uses deterministic seed.
	n := 400
	xs := make([]float64, n)

	for i := range n {
		if i < n/2 {
			xs[i] = rng.NormFloat64()*5 + 30
		} else {
			xs[i] = rng.NormFloat64()*3 + 50
		}
	}

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng, eng.NewFloat64Column("value", xs))
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("value")).
		Layer(geom.Density(geom.WithFill("#3498DB"), geom.WithAlpha(0.5), geom.WithColor("#2C3E50"))).
		Labs(ggplot.Title("Density Plot"), ggplot.Subtitle("Bimodal distribution"))

	_, filename, _, _ := runtime.Caller(0)
	if err := p.Save(context.Background(), filepath.Join(filepath.Dir(filename), "density.png"), 800, 600, ggplot.WithCPU()); err != nil {
		log.Fatalln(err)
	}
}
