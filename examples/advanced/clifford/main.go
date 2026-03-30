package main

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/output/file"
	"github.com/TuSKan/ggplot/pkg/output/ui"
	"github.com/TuSKan/ggplot/pkg/plot"
)

func main() {
	numPoints := 100000

	buf := arrow.NewBuffer(numPoints)
	xData := buf.Float64("Space X")
	yData := buf.Float64("Space Y")
	cData := buf.Float64("Density")

	generateCliffordAttractor(numPoints, xData, yData, cData)

	ds, err := buf.Build()
	if err != nil {
		log.Fatalln(err)
	}

	chart := plot.New(ds).
		AddLayer(
			geom.Point(geom.Opts{
				Radius:  0.5,
				Opacity: 0.9,
			}),
			aes.X(aes.Col("Space X")),
			aes.Y(aes.Col("Space Y")),
			aes.Color(aes.Col("Density")),
		)

	out, err := chart.Render(800, 800)
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	err = file.NewFileExporter().Export(out, filepath.Join(filepath.Dir(filename), "clifford.png"))
	if err != nil {
		log.Fatalln(err)
	}

	presenter := ui.NewGPUWindowPresenter()
	fmt.Println("Launching interactive GPU presenter overlay...")
	if err := presenter.Show(out); err != nil {
		log.Fatalln(err)
	}
}

func generateCliffordAttractor(numPoints int, xData, yData, cData []float64) {
	a, b, c, d := -1.4, 1.6, 1.0, 0.7
	x, y := 0.0, 0.0

	for i := 0; i < numPoints; i++ {
		nextX := math.Sin(a*y) + c*math.Cos(a*x)
		nextY := math.Sin(b*x) + d*math.Cos(b*y)

		xData[i] = nextX
		yData[i] = nextY
		cData[i] = math.Sqrt(nextX*nextX + nextY*nextY)

		x, y = nextX, nextY
	}
}
