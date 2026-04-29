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
	n := 100
	xs := make([]float64, n)
	ys := make([]float64, n)
	for i := range xs {
		xs[i] = rand.Float64() * 10
		ys[i] = xs[i]*0.5 + rand.NormFloat64()*1.5
	}

	eng := memory.NewEngine(context.Background())
	ds, err := dataset.NewDataset(eng, eng.NewFloat64Column("x", xs), eng.NewFloat64Column("y", ys))
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(2), geom.WithColor("#95A5A6"))).
		Layer(geom.Smooth(geom.WithColor("#E74C3C"), geom.WithLineWidth(3))).
		Labs(ggplot.Title("Scatter + OLS Smooth"))

	_, filename, _, _ := runtime.Caller(0)
	if err := p.Save(context.Background(), filepath.Join(filepath.Dir(filename), "smooth.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
