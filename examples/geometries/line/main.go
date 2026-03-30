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
	buf := arrow.NewBuffer(150)
	xCol := buf.Float64("x")
	yCol := buf.Float64("y")

	for i := 0; i < 150; i++ {
		xCol[i] = float64(i)
		yCol[i] = math.Log(float64(i+1)) + math.Sin(float64(i)*0.2)
	}

	ds, err := buf.Build()
	if err != nil {
		log.Fatalln(err)
	}

	p := plot.New(ds).
		AddLayer(
			geom.Line(geom.Opts{}),
			aes.X(aes.Col("x")),
			aes.Y(aes.Col("y")),
		)

	out, err := p.Render(800, 600)
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	err = file.NewFileExporter().Export(out, filepath.Join(filepath.Dir(filename), "line.png"))
	if err != nil {
		log.Fatalln(err)
	}
}
