package main

import (
	"log"
	"math/rand"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/geom"
)

func main() {
	// Generate random data.
	n := 200
	xs := make([]float64, n)
	ys := make([]float64, n)
	for i := 0; i < n; i++ {
		xs[i] = rand.NormFloat64() * 10.0
		ys[i] = xs[i]*0.5 + rand.NormFloat64()*2.0
	}

	// Create dataset using the new dplyr-style API.
	ds, err := dataset.NewDataFrame(map[string][]float64{
		"x": xs,
		"y": ys,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Build the plot — ggplot2-style grammar.
	p := ggplot.New(ds,
		aes.X("x"),
		aes.Y("y"),
	).
		Layer(geom.Point(
			geom.WithSize(3),
			geom.WithAlpha(0.7),
			geom.WithColor("#4C72B0"),
		)).
		Labs(
			ggplot.Title("Scatter Plot Example"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)

	// Save to PNG.
	_, filename, _, _ := runtime.Caller(0)
	outPath := filepath.Join(filepath.Dir(filename), "point.png")

	if err := p.Save(outPath, 800, 600); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved to %s\n", outPath)
}
