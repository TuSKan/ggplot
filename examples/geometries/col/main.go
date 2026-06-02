// Example col demonstrates the geom.Col geometry (pre-computed bar chart).
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
	"github.com/TuSKan/ggplot/output"
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewStringColumn("fruit", []string{"Apple", "Banana", "Cherry", "Date", "Elderberry"}),
		eng.NewFloat64Column("sales", []float64{45, 32, 58, 21, 39}),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("fruit"), aes.Y("sales")).
		Layer(geom.Col(geom.WithFill("#9B59B6"), geom.WithAlpha(0.85))).
		Labs(ggplot.Title("Column Chart"), ggplot.Subtitle("Fruit sales by category"))

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "col.png"), 800, 600, output.WithCPU(true)); err != nil {
		log.Fatalln(err)
	}
}
