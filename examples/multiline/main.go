// Example: Multi-Line Plot
//
// Demonstrates two approaches for plotting multiple series:
//
//  1. Wide format — each series is its own column, added as separate layers.
//  2. Long format — all data in one column with a group identifier, using aes.Color.
//
// Both approaches produce the same visual result.
package main

import (
	"log"
	"math"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	wideFormat(dir)
	longFormat(dir)
}

// wideFormat uses separate columns for each series — the most intuitive
// approach when your data already has one column per variable.
func wideFormat(dir string) {
	n := 100
	x := make([]float64, n)
	sin := make([]float64, n)
	cos := make([]float64, n)
	sin2 := make([]float64, n)

	for i := range x {
		t := float64(i) / float64(n-1) * 2 * math.Pi
		x[i] = t
		sin[i] = math.Sin(t)
		cos[i] = math.Cos(t)
		sin2[i] = math.Sin(2*t) * 0.5
	}

	ds, err := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", x),
		memory.NewEngine().NewFloat64Column("sin", sin),
		memory.NewEngine().NewFloat64Column("cos", cos),
		memory.NewEngine().NewFloat64Column("sin_2x", sin2),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// Each series is a separate layer with its own Y mapping, color, and label.
	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Line(geom.WithColor("#1F77B4"), geom.WithLineWidth(2), geom.WithLabel("sin(x)")), aes.Y("sin")).
		Layer(geom.Line(geom.WithColor("#FF7F0E"), geom.WithLineWidth(2), geom.WithLabel("cos(x)")), aes.Y("cos")).
		Layer(geom.Line(geom.WithColor("#2CA02C"), geom.WithLineWidth(2), geom.WithLabel("sin(2x)/2")), aes.Y("sin_2x")).
		Labs(
			ggplot.Title("Trigonometric Functions (Wide Format)"),
			ggplot.XLab("x (radians)"),
			ggplot.YLab("f(x)"),
		)

	out := filepath.Join(dir, "multiline_wide.png")
	if err := p.Save(out, 900, 600); err != nil {
		log.Fatalln(err)
	}
	log.Printf("Saved %s", out)
}

// longFormat uses the tidy/long data layout where all Y values are in one
// column with a grouping column. This enables automatic color mapping and
// legend generation via aes.Color().
func longFormat(dir string) {
	n := 100
	xs := make([]float64, 0, n*3)
	ys := make([]float64, 0, n*3)
	groups := make([]string, 0, n*3)

	for i := 0; i < n; i++ {
		t := float64(i) / float64(n-1) * 2 * math.Pi
		xs = append(xs, t, t, t)
		ys = append(ys, math.Sin(t), math.Cos(t), math.Sin(2*t)*0.5)
		groups = append(groups, "sin(x)", "cos(x)", "sin(2x)/2")
	}

	ds, err := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", xs),
		memory.NewEngine().NewFloat64Column("y", ys),
		memory.NewEngine().NewStringColumn("func", groups),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// One layer + aes.Color gives automatic grouping, colors, and legend.
	p := ggplot.New(ds,
		aes.X("x"),
		aes.Y("y"),
		aes.Color("func"),
	).
		Layer(geom.Line(geom.WithLineWidth(2))).
		Labs(
			ggplot.Title("Trigonometric Functions (Long Format)"),
			ggplot.XLab("x (radians)"),
			ggplot.YLab("f(x)"),
		)

	out := filepath.Join(dir, "multiline.png")
	if err := p.Save(out, 900, 600); err != nil {
		log.Fatalln(err)
	}
	log.Printf("Saved %s", out)
}
