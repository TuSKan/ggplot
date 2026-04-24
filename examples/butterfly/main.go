// Example: Butterfly curve with continuous color gradient.
//
// Demonstrates:
//   - Parametric curve rendering (Butterfly curve)
//   - Continuous color mapping via aes.Color() on a numeric column
//   - Viridis colormap applied per-segment
package main

import (
	"log"
	"math"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	numPoints := 2000
	maxT := 12.0 * math.Pi
	step := maxT / float64(numPoints-1)

	xData := make([]float64, numPoints)
	yData := make([]float64, numPoints)
	zData := make([]float64, numPoints)

	for i := 0; i < numPoints; i++ {
		t := float64(i) * step

		eCosT := math.Exp(math.Cos(t))
		cos4T := math.Cos(4.0 * t)
		sinT12 := math.Sin(t / 12.0)
		sinT12_5 := sinT12 * sinT12 * sinT12 * sinT12 * sinT12

		term := eCosT - 2.0*cos4T - sinT12_5

		xData[i] = math.Sin(t) * term
		yData[i] = math.Cos(t) * term
		zData[i] = math.Sqrt(xData[i]*xData[i] + yData[i]*yData[i])
	}

	ds, err := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", xData),
		memory.NewEngine().NewFloat64Column("y", yData),
		memory.NewEngine().NewFloat64Column("z", zData),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds,
		aes.X("x"),
		aes.Y("y"),
		aes.Color("z"),
	).
		Layer(geom.Line(geom.WithLineWidth(1.2))).
		Labs(
			ggplot.Title("Butterfly Curve"),
			ggplot.Subtitle("Parametric curve with continuous color by radius"),
			ggplot.XLab("x"),
			ggplot.YLab("y"),
		).
		Theme("dark")

	out := filepath.Join(dir, "butterfly.png")
	if err := p.Save(out, 800, 800); err != nil {
		log.Fatalln(err)
	}
	log.Printf("Saved %s", out)
}
