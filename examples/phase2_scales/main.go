// Phase 2: Scales — Linear, Log10, Sqrt, Reverse, Discrete
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
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	linearScale(dir)
	log10Scale(dir)
	sqrtScale(dir)
	reverseScale(dir)
	discreteScale(dir)
}

func save(p *ggplot.Plot, dir, name string, w, h int) { //nolint:unparam // Example helper keeps w generic.
	out := filepath.Join(dir, name+".png")
	if err := file.Save(context.Background(), p, out, w, h); err != nil {
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

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng, eng.NewFloat64Column("x", xs), eng.NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#3498DB"), geom.WithLineWidth(2))).
		Labels(ggplot.Title("Scale: Linear (default)"), ggplot.Subtitle("y = 2.5x")).
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

	eng2 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng2, eng2.NewFloat64Column("x", xs), eng2.NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#E74C3C"), geom.WithLineWidth(2))).
		Layer(geom.Point(geom.WithSize(2), geom.WithColor("#E74C3C"))).
		ScaleY("log10").
		Labels(ggplot.Title("Scale: Log10 Y-Axis"), ggplot.Subtitle("Exponential growth on logarithmic axis")).
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

	eng3 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng3, eng3.NewFloat64Column("x", xs), eng3.NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#27AE60"), geom.WithLineWidth(2))).
		ScaleY("sqrt").
		Labels(ggplot.Title("Scale: Sqrt Y-Axis"), ggplot.Subtitle("Quadratic data linearized via sqrt transform")).
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

	eng4 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng4, eng4.NewFloat64Column("x", xs), eng4.NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#F39C12"), geom.WithLineWidth(2))).
		ScaleY("reverse").
		Labels(ggplot.Title("Scale: Reverse Y-Axis"), ggplot.Subtitle("Y axis inverted — high values at bottom")).
		Theme("dark")
	save(p, dir, "04_scale_reverse", 800, 500)
}

// Discrete scale (categorical X)
func discreteScale(dir string) {
	eng5 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng5,
		eng5.NewStringColumn("language", []string{"Go", "Python", "Rust", "Java", "TypeScript", "C++"}),
		eng5.NewFloat64Column("stars", []float64{122, 215, 98, 178, 142, 67}),
	)
	p := ggplot.New(ds, aes.X("language"), aes.Y("stars")).
		Layer(geom.Col(geom.WithFill("#9B59B6"), geom.WithAlpha(0.85))).
		Labels(ggplot.Title("Scale: Discrete X-Axis"), ggplot.Subtitle("Categorical programming languages")).
		Theme("classic")
	save(p, dir, "05_scale_discrete", 800, 500)
}
