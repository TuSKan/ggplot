package main

import (
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
	n := 200
	xs := make([]float64, n)
	ys := make([]float64, n)
	for i := 0; i < n; i++ {
		xs[i] = float64(i) / float64(n) * 10.0
		ys[i] = xs[i]*0.3 + rand.NormFloat64()*1.5
	}

	ds, err := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs), memory.NewEngine().NewFloat64Column("y", ys))
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#E74C3C"), geom.WithLineWidth(2))).
		Labs(ggplot.Title("Line Plot"))

	_, filename, _, _ := runtime.Caller(0)
	if err := p.Save(filepath.Join(filepath.Dir(filename), "line.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
