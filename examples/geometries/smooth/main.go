package main

import (
	"log"
	"math/rand"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/output/file"
	"github.com/TuSKan/ggplot/pkg/plot"
	"github.com/TuSKan/ggplot/pkg/stat"
)

func main() {
	buf := arrow.NewBuffer(150)
	xCol := buf.Float64("x_data")
	yCol := buf.Float64("y_data")

	for i := 0; i < 150; i++ {
		xCol[i] = float64(i)
		yCol[i] = xCol[i]*0.5 + rand.NormFloat64()*10.0 // Noisy data
	}

	ds, err := buf.Build()
	if err != nil {
		log.Fatalln(err)
	}

	p := plot.New(ds).
		Aes(
			aes.X(aes.Col("x_data")),
			aes.Y(aes.Col("y_data")),
		).
		AddLayer(
			geom.Point(geom.Opts{Radius: 2.0, Opacity: 0.8}),
		).
		AddLayer(
			geom.Smooth(stat.MethodLoess),
		)

	out, err := p.Render(800, 600)
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	err = file.NewFileExporter().Export(out, filepath.Join(filepath.Dir(filename), "smooth.png"))
	if err != nil {
		log.Fatalln(err)
	}
}
