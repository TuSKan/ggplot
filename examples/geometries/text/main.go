// Example text demonstrates the geom.Text geometry.
package main

import (
	"context"
	"log"
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
	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng.NewFloat64Column("y", []float64{2, 5, 3, 7, 4}),
		eng.NewStringColumn("label", []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Label("label")).
		Layer(geom.Point(geom.WithSize(5), geom.WithColor("#E74C3C"))).
		Layer(geom.Text()).
		Labels(ggplot.Title("Text Labels"))

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "text.png"), 800, 600, output.WithCPU(true)); err != nil {
		log.Fatalln(err)
	}
}
