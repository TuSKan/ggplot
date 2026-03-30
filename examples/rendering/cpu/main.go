package main

import (
	"log"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/output/file"
	"github.com/TuSKan/ggplot/pkg/plot"
)

func main() {
	buf := arrow.NewBuffer(3)
	xCol := buf.Float64("x")
	yCol := buf.Float64("y")
	xCol[0], yCol[0] = 1, 10
	xCol[1], yCol[1] = 2, 20
	xCol[2], yCol[2] = 3, 30
	ds, _ := buf.Build()

	p := plot.New(ds).
		AddLayer(
			geom.Point(geom.Opts{}),
			aes.X(aes.Col("x")),
			aes.Y(aes.Col("y")),
		)

	out, err := p.Render(800, 600)
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	err = file.NewFileExporter().Export(out, filepath.Join(filepath.Dir(filename), "output_cpu.png"))
	if err != nil {
		log.Fatalf("Software engine failed disk bounds output : %v", err)
	}

	log.Println("Standalone rendering written identically completely to output_cpu.png.")
}

