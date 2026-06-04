// Tabular Figures Example
package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	tabularFiguresExample(dir)
}

func tabularFiguresExample(dir string) {
	eng := memory.NewEngine(context.Background())

	// Large values with varying digit structures like 111 vs 888 to demonstrate
	// monospaced digit columns provided by font-variant-numeric: tabular-nums
	ds, _ := dataset.NewDataset(eng,
		eng.NewStringColumn("Category", []string{"A", "B", "C", "D"}),
		eng.NewFloat64Column("Value", []float64{111111.1, 888888.8, 123456.7, 999999.9}),
	)

	p := ggplot.New(ds, aes.X("Category"), aes.Y("Value")).
		Layer(geom.Col(geom.WithFill("#3498DB"), geom.WithAlpha(0.85))).
		Labels(ggplot.Title("Tabular Figures in Action"), ggplot.Subtitle("Quantitative axes align correctly (e.g. 111111.1 vs 888888.8)")).
		Theme("minimal")

	save(p, dir, "tabular_figures", 800, 500)
}

func save(p *ggplot.Plot, dir, name string, w, h int) {
	outPNG := filepath.Join(dir, name+".png")
	outSVG := filepath.Join(dir, name+".svg")

	if err := file.Save(context.Background(), p, outPNG, w, h); err != nil {
		log.Fatalln(err)
	}

	if err := file.Save(context.Background(), p, outSVG, w, h); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s and %s", outPNG, outSVG)
}
