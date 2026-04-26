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
)

func main() {
	ds, err := dataset.NewDataset(memory.NewEngine(context.Background()),
		memory.NewEngine(context.Background()).NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		memory.NewEngine(context.Background()).NewFloat64Column("count", []float64{10, 25, 15, 30, 20}),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("count")).
		Layer(geom.Col(
			geom.WithFill("#9B59B6"),
			geom.WithAlpha(0.85),
		)).
		Labs(
			ggplot.Title("Sales by Category"),
			ggplot.XLab("Category"),
			ggplot.YLab("Count"),
		)

	_, filename, _, _ := runtime.Caller(0)
	if err := p.Save(context.Background(), filepath.Join(filepath.Dir(filename), "bar.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
