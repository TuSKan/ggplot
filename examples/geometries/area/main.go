package main

import (
	"log"
	"math"
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
		xs[i] = float64(i) / float64(n) * 4 * math.Pi
		ys[i] = math.Sin(xs[i])
	}

	ds, err := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs), memory.NewEngine().NewFloat64Column("y", ys))
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Area(geom.WithFill("#2ECC71"), geom.WithAlpha(0.5))).
		Labs(ggplot.Title("Area Plot"))

	_, filename, _, _ := runtime.Caller(0)
	if err := p.Save(filepath.Join(filepath.Dir(filename), "area.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
