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
)

func main() {
	buf := arrow.NewBuffer(200)
	xCol := buf.Float64("x")
	yCol := buf.Float64("y")

	for i := 0; i < 200; i++ {
		xCol[i] = rand.NormFloat64() * 10.0
		yCol[i] = xCol[i]*0.5 + rand.NormFloat64()*2.0
	}

	ds, err := buf.Build()
	if err != nil {
		log.Fatalln(err)
	}

	p := plot.New(ds).
		AddLayer(
			geom.Point(geom.Opts{Radius: 3.0, Opacity: 0.8}),
			aes.X(aes.Col("x")),
			aes.Y(aes.Col("y")),
		)

	out, err := p.Render(800, 600)
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	err = file.NewFileExporter().Export(out, filepath.Join(filepath.Dir(filename), "point.png"))
	if err != nil {
		log.Fatalln(err)
	}
}
