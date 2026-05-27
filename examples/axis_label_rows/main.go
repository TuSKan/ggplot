// Example axis_label_rows demonstrates the AxisLabelRows feature for
// dense categorical X-axis labels. Shows auto-dodge (default), explicit
// 3-row stagger, and disabled dodge for comparison.
package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
)

func main() {
	eng := memory.NewEngine(context.Background())

	// 20 categories with long names — guaranteed to overlap on 800px width.
	n := 20
	xs := make([]string, n)
	ys := make([]float64, n)

	for i := range n {
		xs[i] = fmt.Sprintf("Category-%02d", i)
		ys[i] = float64((i + 1) * (i + 1))
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewStringColumn("category", xs),
		eng.NewFloat64Column("value", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// 1. Auto-dodge (default, n=0) — detects overlap and staggers to 2 rows.
	p1 := ggplot.New(ds, aes.X("category"), aes.Y("value")).
		Layer(geom.Col(geom.WithFill("#3498DB"))).
		Labs(
			ggplot.Title("Auto-Dodge (default)"),
			ggplot.XLab("Category"),
			ggplot.YLab("Value"),
		)

	// 2. Explicit 3-row stagger — derived before any Save to avoid
	// Build() mutating the shared dataset reference.
	p2 := p1.AxisLabelRows(3).
		Labs(ggplot.Title("AxisLabelRows(3) — 3-row stagger"))

	if err := p1.Save(context.Background(), filepath.Join(dir, "axis_label_rows_auto.png"), 800, 500); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("saved axis_label_rows_auto.png")

	if err := p2.Save(context.Background(), filepath.Join(dir, "axis_label_rows_3.png"), 800, 500); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("saved axis_label_rows_3.png")
}
