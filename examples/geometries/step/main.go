// Example step demonstrates the geom.Step geometry.
package main

import (
	"context"
	"log"
	"math"
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
	n := 40
	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range n {
		xs[i] = float64(i)
		ys[i] = math.Floor(math.Sin(float64(i)*0.3)*4) + 5
	}

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("time", xs),
		eng.NewFloat64Column("level", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("time"), aes.Y("level")).
		Layer(geom.Step(geom.WithColor("#2ECC71"), geom.WithLineWidth(2))).
		Labs(ggplot.Title("Step Plot"))

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "step.png"), 800, 600, output.WithCPU(true)); err != nil {
		log.Fatalln(err)
	}
}
