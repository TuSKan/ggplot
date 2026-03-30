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
	buf := arrow.NewBuffer(5)
	xCol := buf.Float64("category")
	yCol := buf.Float64("value")
	for i := 0; i < 5; i++ {
		xCol[i] = float64(i)
		yCol[i] = float64(i * 10)
	}
	ds, _ := buf.Build()

	p := plot.New(ds).
		AddLayer(
			geom.Point(geom.Opts{}),
			aes.X(aes.Col("category")),
			aes.Y(aes.Col("value")),
		)
	// Apply layout modifiers (e.g.,.WithTheme(theme.Classic())) mapping constraints.

	out, err := p.Render(800, 600)
	if err != nil {
		log.Fatalf("tracked boundary exactly : %v", err)
	}

	_, filename, _, _ := runtime.Caller(0)
	err = file.NewFileExporter().Export(out, filepath.Join(filepath.Dir(filename), "theme.png"))
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Themes compiled statically appropriately")
}
