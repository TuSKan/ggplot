// Example polygon demonstrates the geom.Polygon geometry for drawing
// closed filled shapes from grouped x/y coordinates.
package main

import (
	"context"
	"log"
	"math"
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
	eng := memory.NewEngine(context.Background())

	// Regular pentagon centred at (5, 5) with radius 3.
	const n = 5

	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range n {
		angle := 2*math.Pi*float64(i)/float64(n) - math.Pi/2
		xs[i] = 5 + 3*math.Cos(angle)
		ys[i] = 5 + 3*math.Sin(angle)
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Polygon(
			geom.WithFill("#2ECC71"),
			geom.WithColor("#27AE60"),
			geom.WithAlpha(0.7),
			geom.WithLineWidth(2),
		)).
		Labs(
			ggplot.Title("Regular Pentagon"),
			ggplot.XLab("x"),
			ggplot.YLab("y"),
		)

	_, filename, _, _ := runtime.Caller(0)
	if err := file.Save(context.Background(), p, filepath.Join(filepath.Dir(filename), "polygon.png"), 800, 600); err != nil {
		log.Fatalln(err)
	}
}
