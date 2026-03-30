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
	buf := arrow.NewBuffer(10)
	xCol := buf.Float64("density")
	yCol := buf.Float64("mass")
	for i := 0; i < 10; i++ {
		xCol[i] = float64(i)
		yCol[i] = float64(i * i)
	}
	ds, _ := buf.Build()

	p := plot.New(ds).
		AddLayer(
			geom.Point(geom.Opts{}),
			aes.X(aes.Col("density")),
			aes.Y(aes.Col("mass")),
		)

	out, err := p.Render(800, 600)
	if err != nil {
		log.Fatalf("validated bounds : %v", err)
	}

	_, filename, _, _ := runtime.Caller(0)
	err = file.NewFileExporter().Export(out, filepath.Join(filepath.Dir(filename), "facet.png"))
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Facet grids routed out")
}
