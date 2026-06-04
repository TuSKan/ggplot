// Example curve demonstrates the geom.Curve geometry for drawing
// quadratic bezier curves between pairs of endpoints.
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
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	eng := memory.NewEngine(context.Background())

	// Network-style connections between nodes.
	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 1, 2, 3}),
		eng.NewFloat64Column("y", []float64{1, 3, 2, 1}),
		eng.NewFloat64Column("xend", []float64{3, 4, 4, 5}),
		eng.NewFloat64Column("yend", []float64{3, 4, 1, 3}),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.XEnd("xend"), aes.YEnd("yend")).
		Layer(geom.Curve(
			geom.WithColor("#2980B9"),
			geom.WithLineWidth(2),
			geom.WithCurvature(0.4),
		)).
		Layer(geom.Point( // source nodes
			geom.WithColor("#E74C3C"),
			geom.WithSize(5),
		)).
		Layer(geom.Point( // target nodes
			geom.WithColor("#27AE60"),
			geom.WithSize(5),
		), aes.X("xend"), aes.Y("yend")).
		Labels(
			ggplot.Title("Network Connections — Quadratic Curves"),
			ggplot.XLabel("X"),
			ggplot.YLabel("Y"),
		)

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "curve.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
