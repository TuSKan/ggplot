// Example line demonstrates the geom.line geometry.
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
	n := 200
	xs := make([]float64, n)

	ys := make([]float64, n)
	for i := range n {
		xs[i] = float64(i) / float64(n) * 10.0
		ys[i] = xs[i]*0.3 + rand.NormFloat64()*1.5
	}

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng, eng.NewFloat64Column("x", xs), eng.NewFloat64Column("y", ys))
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#E74C3C"), geom.WithLineWidth(2))).
		Labels(ggplot.Title("Line Plot"))

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "line.png"), 800, 600, output.WithCPU(true)); err != nil {
		log.Fatalln(err)
	}
}
