// Example crossbar demonstrates the geom.Crossbar geometry for showing
// a filled box between ymin/ymax with a median line at y.
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

	// Treatment groups with median and interquartile ranges.
	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4}),
		eng.NewFloat64Column("y", []float64{12, 18, 14, 22}),
		eng.NewFloat64Column("ymin", []float64{9, 14, 11, 18}),
		eng.NewFloat64Column("ymax", []float64{15, 22, 17, 26}),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.YMin("ymin"), aes.YMax("ymax")).
		Layer(geom.Crossbar(
			geom.WithFill("#3498DB"),
			geom.WithColor("#2C3E50"),
			geom.WithWidth(0.6),
			geom.WithLineWidth(1.5),
		)).
		Labels(
			ggplot.Title("Treatment Response — Crossbar"),
			ggplot.XLabel("Treatment Group"),
			ggplot.YLabel("Response (median ± IQR)"),
		)

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "crossbar.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
