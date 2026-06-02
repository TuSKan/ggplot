// Example: (Clifford[https://paulbourke.net/fractals/clifford/] attractor with continuous color gradient.
//
// Demonstrates:
//   - Strange attractor visualization
//   - Continuous color mapping on points via aes.Color()
//   - Viridis colormap applied per-point
package main

import (
	"context"
	"log"
	"math"
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
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	numPoints := 100000
	xData := make([]float64, numPoints)
	yData := make([]float64, numPoints)
	cData := make([]float64, numPoints)

	a, b, c, d := -1.4, 1.6, 1.0, 0.7
	x, y := 0.0, 0.0

	for i := range numPoints {
		nextX := math.Sin(a*y) + c*math.Cos(a*x)
		nextY := math.Sin(b*x) + d*math.Cos(b*y)

		xData[i] = nextX
		yData[i] = nextY
		cData[i] = math.Sqrt(nextX*nextX + nextY*nextY)

		x, y = nextX, nextY
	}

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("Space X", xData),
		eng.NewFloat64Column("Space Y", yData),
		eng.NewFloat64Column("Density", cData),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds,
		aes.X("Space X"),
		aes.Y("Space Y"),
		aes.Color("Density"),
	).
		Layer(geom.Point(geom.WithSize(0.5), geom.WithAlpha(0.6))).
		Labs(
			ggplot.Title("Clifford Attractor"),
			ggplot.Subtitle("100,000 iterations · a=-1.4, b=1.6, c=1.0, d=0.7"),
			ggplot.XLab("Space X"),
			ggplot.YLab("Space Y"),
		).
		LegendPosition("bottom").
		Theme("dark")

	out := filepath.Join(dir, "clifford.png")
	if err := file.Save(context.Background(), p, out, 900, 900); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}
