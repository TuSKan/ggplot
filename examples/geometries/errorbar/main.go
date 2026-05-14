// Example errorbar demonstrates the geom.ErrorBar geometry for
// showing measurement uncertainty with vertical bars and caps.
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

	// Measurements with ±1σ uncertainty bands.
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{10, 15, 13, 17, 20}
	ymin := []float64{8, 12, 10, 14, 17}
	ymax := []float64{12, 18, 16, 20, 23}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", x),
		eng.NewFloat64Column("y", y),
		eng.NewFloat64Column("ymin", ymin),
		eng.NewFloat64Column("ymax", ymax),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.YMin("ymin"), aes.YMax("ymax")).
		Layer(geom.Point(
			geom.WithColor("#E74C3C"),
			geom.WithSize(4),
		)).
		Layer(geom.ErrorBar(
			geom.WithColor("#2C3E50"),
			geom.WithLineWidth(1.5),
		)).
		Labs(
			ggplot.Title("Measurements with Error Bars"),
			ggplot.XLab("Sample"),
			ggplot.YLab("Value ± σ"),
		)

	_, filename, _, _ := runtime.Caller(0)
	if err := p.Save(context.Background(), filepath.Join(filepath.Dir(filename), "errorbar.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
