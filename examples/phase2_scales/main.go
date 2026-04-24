// Phase 2: Scales — Linear, Log10, Sqrt, Reverse, Discrete
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

	linearScale(dir)
	log10Scale(dir)
	sqrtScale(dir)
	reverseScale(dir)
	discreteScale(dir)
}

func save(p *ggplot.Plot, dir, name string, w, h int) {
	out := filepath.Join(dir, name+".png")
	if err := p.Save(out, w, h); err != nil {
		log.Fatalln(err)
	}
	log.Printf("Saved %s", out)
}

// Linear scale (default)
func linearScale(dir string) {
	n := 50
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i)
		ys[i] = float64(i) * 2.5
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs), memory.NewEngine().NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#3498DB"), geom.WithLineWidth(2))).
		Labs(ggplot.Title("Scale: Linear (default)"), ggplot.Subtitle("y = 2.5x")).
		Theme("dark")
	save(p, dir, "01_scale_linear", 800, 500)
}

// Log10 scale
func log10Scale(dir string) {
	n := 50
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i + 1)
		ys[i] = math.Pow(10, float64(i)*0.08)
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs), memory.NewEngine().NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#E74C3C"), geom.WithLineWidth(2))).
		Layer(geom.Point(geom.WithSize(2), geom.WithColor("#E74C3C"))).
		ScaleY("log10").
		Labs(ggplot.Title("Scale: Log10 Y-Axis"), ggplot.Subtitle("Exponential growth on logarithmic axis")).
		Theme("minimal")
	save(p, dir, "02_scale_log10", 800, 500)
}

// Sqrt scale
func sqrtScale(dir string) {
	n := 40
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i)
		ys[i] = float64(i * i)
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs), memory.NewEngine().NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#27AE60"), geom.WithLineWidth(2))).
		ScaleY("sqrt").
		Labs(ggplot.Title("Scale: Sqrt Y-Axis"), ggplot.Subtitle("Quadratic data linearized via sqrt transform")).
		Theme("bw")
	save(p, dir, "03_scale_sqrt", 800, 500)
}

// Reverse scale
func reverseScale(dir string) {
	n := 30
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i)
		ys[i] = math.Sin(float64(i) * 0.3)
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs), memory.NewEngine().NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#F39C12"), geom.WithLineWidth(2))).
		ScaleY("reverse").
		Labs(ggplot.Title("Scale: Reverse Y-Axis"), ggplot.Subtitle("Y axis inverted — high values at bottom")).
		Theme("dark")
	save(p, dir, "04_scale_reverse", 800, 500)
}

// Discrete scale (categorical X)
func discreteScale(dir string) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewStringColumn("language", []string{"Go", "Python", "Rust", "Java", "TypeScript", "C++"}),
		memory.NewEngine().NewFloat64Column("stars", []float64{122, 215, 98, 178, 142, 67}),
	)
	p := ggplot.New(ds, aes.X("language"), aes.Y("stars")).
		Layer(geom.Col(geom.WithFill("#9B59B6"), geom.WithAlpha(0.85))).
		Labs(ggplot.Title("Scale: Discrete X-Axis"), ggplot.Subtitle("Categorical programming languages")).
		Theme("classic")
	save(p, dir, "05_scale_discrete", 800, 500)
}
