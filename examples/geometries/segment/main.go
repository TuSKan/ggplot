// Example segment demonstrates the geom.Segment geometry for drawing
// line segments from (x, y) to (xend, yend).
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

	// Dumbbell chart: before/after values for five items.
	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{20, 35, 50, 15, 45}),
		eng.NewFloat64Column("y", []float64{1, 2, 3, 4, 5}),
		eng.NewFloat64Column("xend", []float64{55, 50, 65, 40, 75}),
		eng.NewFloat64Column("yend", []float64{1, 2, 3, 4, 5}),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.XEnd("xend"), aes.YEnd("yend")).
		Layer(geom.Segment(
			geom.WithColor("#BDC3C7"),
			geom.WithLineWidth(3),
		)).
		Layer(geom.Point( // "before" dots
			geom.WithColor("#E74C3C"),
			geom.WithSize(5),
		)).
		Layer(geom.Point( // "after" dots
			geom.WithColor("#2ECC71"),
			geom.WithSize(5),
		), aes.X("xend"), aes.Y("yend")).
		Labs(
			ggplot.Title("Dumbbell Chart — Before vs After"),
			ggplot.XLab("Value"),
			ggplot.YLab("Item"),
		)

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "segment.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
