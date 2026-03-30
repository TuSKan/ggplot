package main

import (
	"log"
	"math"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/output/file"
	"github.com/TuSKan/ggplot/pkg/plot"
)

func main() {
	numPoints := 5 // Pentagram
	// Need to close polygon
	buf := arrow.NewBuffer(numPoints*2 + 1)
	xCol := buf.Float64("x")
	yCol := buf.Float64("y")

	r1 := 1.0
	r2 := 0.4
	for i := 0; i < numPoints*2; i++ {
		r := r1
		if i%2 != 0 {
			r = r2
		}
		angle := float64(i) * math.Pi / float64(numPoints)
		xCol[i] = r * math.Cos(angle)
		yCol[i] = r * math.Sin(angle)
	}
	// Close polygon functionally magnetically!
	xCol[numPoints*2] = xCol[0]
	yCol[numPoints*2] = yCol[0]

	ds, err := buf.Build()
	if err != nil {
		log.Fatalln(err)
	}

	p := plot.New(ds).
		AddLayer(
			geom.Polygon(geom.Opts{Opacity: 0.6}),
			aes.X(aes.Col("x")),
			aes.Y(aes.Col("y")),
		)

	out, err := p.Render(800, 600)
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	err = file.NewFileExporter().Export(out, filepath.Join(filepath.Dir(filename), "polygon.png"))
	if err != nil {
		log.Fatalln(err)
	}
}
