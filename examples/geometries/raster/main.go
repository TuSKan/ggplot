// Example raster demonstrates the geom.Raster dense pixel-aligned image grid.
// Unlike geom.Tile (individual rectangles per cell), Raster composites
// the entire grid into a single image and renders via native canvas
// transforms — orders of magnitude faster for dense grids.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
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

	// 50×50 grid — a continuous 2D function (ripple pattern).
	const gridSize = 50

	n := gridSize * gridSize
	xs := make([]float64, 0, n)
	ys := make([]float64, 0, n)
	zs := make([]float64, 0, n)

	for row := range gridSize {
		for col := range gridSize {
			x := float64(col) - float64(gridSize)/2
			y := float64(row) - float64(gridSize)/2
			r := math.Sqrt(x*x + y*y)
			z := math.Sin(r*0.5) * math.Exp(-r*0.03) //nolint:mnd // Dampened ripple function.

			xs = append(xs, float64(col))
			ys = append(ys, float64(row))
			zs = append(zs, z)
		}
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("z", zs),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("z")).
		Layer(geom.Raster()).
		ScaleColorContinuous(colormap.Viridis, nil).
		Labs(
			ggplot.Title("Raster — Dampened Ripple (50×50)"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "raster.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("saved raster.png")
}
