package main

import (
	"fmt"
	"log"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/output/file"
	"github.com/TuSKan/ggplot/pkg/output/ui"
	"github.com/TuSKan/ggplot/pkg/plot"
)

func main() {
	buf := arrow.NewBuffer(3)
	x := buf.Float64("x")
	y := buf.Float64("y")
	x[0], y[0] = 1, 1
	x[1], y[1] = 2, 2
	x[2], y[2] = 3, 1
	ds, _ := buf.Build()

	p := plot.New(ds).
		AddLayer(
			geom.Point(geom.Opts{Radius: 10}),
			aes.X(aes.Col("x")),
			aes.Y(aes.Col("y")),
		)

	out, err := p.Render(600, 400)
	if err != nil {
		log.Fatalln(err)
	}

	presenter := ui.NewGPUWindowPresenter()
	exporter := file.NewFileExporter()

	exportPath := "test_snapshot.png"
	fmt.Printf("Exporting snapshot to %s...\n", exportPath)
	if err := exporter.Export(out, exportPath); err != nil {
		log.Printf("Export failed: %v", err)
	}

	fmt.Println("Launching interactive GPU presenter overlay...")
	if err := presenter.Show(out); err != nil {
		log.Fatalf("Interactive presentation failed: %v", err)
	}
}
