// Example pointrange demonstrates the geom.Pointrange geometry for showing
// a point at the central estimate with a vertical line spanning the range.
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
	eng := memory.NewEngine(context.Background())

	// Model coefficients with 95% confidence intervals.
	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng.NewFloat64Column("y", []float64{2.3, 4.1, 3.7, 5.5, 6.2}),
		eng.NewFloat64Column("ymin", []float64{1.5, 3.2, 2.8, 4.6, 5.0}),
		eng.NewFloat64Column("ymax", []float64{3.1, 5.0, 4.6, 6.4, 7.4}),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.YMin("ymin"), aes.YMax("ymax")).
		Layer(geom.Pointrange(
			geom.WithColor("#8E44AD"),
			geom.WithSize(4),
			geom.WithLineWidth(1.5),
		)).
		Labs(
			ggplot.Title("Model Coefficients with 95% CI"),
			ggplot.XLab("Predictor"),
			ggplot.YLab("Estimate ± CI"),
		)

	_, filename, _, _ := runtime.Caller(0)
	if err := p.Save(context.Background(), filepath.Join(filepath.Dir(filename), "pointrange.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
