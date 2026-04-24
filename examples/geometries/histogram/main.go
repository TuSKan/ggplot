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
	n := 5000
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = rand.NormFloat64()*5 + 10
	}

	ds, err := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs))
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Histogram(geom.WithBins(30), geom.WithFill("#3498DB"))).
		Labs(ggplot.Title("Histogram"))

	_, filename, _, _ := runtime.Caller(0)
	if err := p.Save(filepath.Join(filepath.Dir(filename), "histogram.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
