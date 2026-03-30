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

	_ "github.com/gogpu/gg/gpu" // Opt-in explicit hardware parallel acceleration
)

func main() {
	buf := arrow.NewBuffer(3)
	x := buf.Float64("x")
	y := buf.Float64("y")
	x[0], y[0] = 1, 1
	x[1], y[1] = 2, 8
	x[2], y[2] = 3, 4
	ds, _ := buf.Build()

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
	err = file.NewFileExporter().Export(out, filepath.Join(filepath.Dir(filename), "output_gpu.png"))
	if err != nil {
		log.Fatalf("Hardware accelerator bounded missing components functionally : %v", err)
	}

	log.Println("GPU Rendering wired layouts exactly.")
}
