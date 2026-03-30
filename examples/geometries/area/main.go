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
	buf := arrow.NewBuffer(100)
	step := buf.Float64("step")
	value := buf.Float64("value")

	for i := 0; i < 100; i++ {
		step[i] = float64(i)
		value[i] = 10.0 + math.Sin(float64(i)*0.1)*5.0
	}

	ds, err := buf.Build()
	if err != nil {
		log.Fatalln(err)
	}

	p := plot.New(ds).
		AddLayer(
			geom.Area(geom.Opts{}),
			aes.X(aes.Col("step")),
			aes.Y(aes.Col("value")),
		)

	out, err := p.Render(800, 600)
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	err = file.NewFileExporter().Export(out, filepath.Join(filepath.Dir(filename), "area.png"))
	if err != nil {
		log.Fatalln(err)
	}
}
