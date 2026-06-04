// Example hline_vline demonstrates the geom.HLine and geom.VLine geometries.
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
	"github.com/TuSKan/ggplot/output"
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	rng := rand.New(rand.NewSource(42)) //nolint:mnd // Example uses deterministic seed.
	n := 100
	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range n {
		xs[i] = rng.Float64() * 20
		ys[i] = xs[i]*1.5 + rng.NormFloat64()*5
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
		Layer(geom.Point(geom.WithSize(2), geom.WithAlpha(0.6), geom.WithColor("#2980B9"))).
		Layer(geom.HLine(geom.WithIntercept(15), geom.WithColor("#E74C3C"), geom.WithLineWidth(1.5))).
		Layer(geom.VLine(geom.WithIntercept(10), geom.WithColor("#27AE60"), geom.WithLineWidth(1.5))).
		Labels(ggplot.Title("HLine + VLine"), ggplot.Subtitle("Reference lines at y=15, x=10"))

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "hline_vline.png"), 800, 600, output.WithCPU(true)); err != nil {
		log.Fatalln(err)
	}
}
