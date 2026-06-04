// Example tile demonstrates the geom.Tile heatmap geometry.
package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/colormap"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	eng := memory.NewEngine(context.Background())

	// 5×5 grid of values for a correlation-style heatmap.
	xs := make([]float64, 0, 25)
	ys := make([]float64, 0, 25)
	vals := make([]float64, 0, 25)

	for row := range 5 {
		for col := range 5 {
			xs = append(xs, float64(col))
			ys = append(ys, float64(row))
			// Synthetic value: distance from diagonal ⟶ brighter near diagonal.
			d := float64((row - col) * (row - col))
			vals = append(vals, 1.0/(1.0+d))
		}
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("value", vals),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("value")).
		Layer(geom.Tile()).
		ScaleColorContinuous(colormap.Viridis, &colormap.LinearNorm{Vmin: 0, Vmax: 1}).
		Labels(
			ggplot.Title("Correlation Heatmap"),
			ggplot.XLabel("Variable"),
			ggplot.YLabel("Variable"),
		)

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "tile.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
