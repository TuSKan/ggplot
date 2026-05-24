// Example linerange demonstrates the geom.Linerange geometry for showing
// vertical lines from ymin to ymax without caps — a minimal range indicator.
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

	// Daily temperature range — min to max.
	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("day", []float64{1, 2, 3, 4, 5, 6, 7}),
		eng.NewFloat64Column("y", []float64{22, 24, 21, 25, 23, 20, 26}),
		eng.NewFloat64Column("ymin", []float64{18, 19, 16, 20, 18, 15, 21}),
		eng.NewFloat64Column("ymax", []float64{26, 29, 26, 30, 28, 25, 31}),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("day"), aes.Y("y"), aes.YMin("ymin"), aes.YMax("ymax")).
		Layer(geom.Linerange(
			geom.WithColor("#16A085"),
			geom.WithLineWidth(2),
		)).
		Layer(geom.Point(
			geom.WithColor("#E74C3C"),
			geom.WithSize(3),
		)).
		Labs(
			ggplot.Title("Daily Temperature Range"),
			ggplot.XLab("Day"),
			ggplot.YLab("Temperature (°C)"),
		)

	_, filename, _, _ := runtime.Caller(0)
	if err := p.Save(context.Background(), filepath.Join(filepath.Dir(filename), "linerange.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
