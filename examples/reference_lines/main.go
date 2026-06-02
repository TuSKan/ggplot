// Reference Lines — HLine, VLine, and ABLine examples.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/file"
	"github.com/TuSKan/ggplot/scale"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	hlineThreshold(dir)
	vlineEvents(dir)
	ablineRegression(dir)
	combined(dir)
}

func save(p *ggplot.Plot, dir, name string, w, h int) {
	out := filepath.Join(dir, name+".png")
	if err := file.Save(context.Background(), p, out, w, h); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}

// ── 1. HLine — threshold / target line ──────────────────────────────────
func hlineThreshold(dir string) {
	n := 30

	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i + 1)
		ys[i] = 60 + 40*rand.Float64() // sales between 60–100
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("day", xs),
		eng.NewFloat64Column("sales", ys),
	)
	p := ggplot.New(ds, aes.X("day"), aes.Y("sales")).
		Layer(geom.Line(geom.WithColor("#3498DB"), geom.WithLineWidth(2))).
		Layer(geom.Point(geom.WithColor("#3498DB"), geom.WithSize(3))).
		Layer(geom.HLine(
			geom.WithIntercept(85),
			geom.WithColor("#E74C3C"),
			geom.WithLineWidth(2),
		)).
		Layer(geom.HLine(
			geom.WithIntercept(70),
			geom.WithColor("#F39C12"),
			geom.WithLineWidth(1.5),
		)).
		ScaleY(scale.Linear,
			scale.WithFormatter(func(v float64) string {
				return fmt.Sprintf("$%.0f", v)
			}),
		).
		Labs(
			ggplot.Title("Daily Sales with Targets"),
			ggplot.Subtitle("Red = stretch goal ($85) · Orange = minimum ($70)"),
			ggplot.XLab("Day"), ggplot.YLab("Sales"),
		).
		Theme("seaborn_whitegrid")
	save(p, dir, "01_hline_threshold", 900, 550)
}

// ── 2. VLine — marking events ───────────────────────────────────────────
func vlineEvents(dir string) {
	n := 100

	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i)

		growth := 0.0
		if i > 30 {
			growth = 0.5
		}

		if i > 70 {
			growth = 1.2
		}

		ys[i] = 10 + growth*float64(i-30) + 5*rand.Float64()
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("day", xs),
		eng.NewFloat64Column("users", ys),
	)
	p := ggplot.New(ds, aes.X("day"), aes.Y("users")).
		Layer(geom.Line(geom.WithColor("#2ECC71"), geom.WithLineWidth(2))).
		Layer(geom.VLine(
			geom.WithIntercept(30),
			geom.WithColor("#E74C3C"),
			geom.WithLineWidth(2),
		)).
		Layer(geom.VLine(
			geom.WithIntercept(70),
			geom.WithColor("#9B59B6"),
			geom.WithLineWidth(2),
		)).
		Labs(
			ggplot.Title("User Growth with Event Markers"),
			ggplot.Subtitle("Red = v1.0 launch (day 30) · Purple = viral moment (day 70)"),
			ggplot.XLab("Day"), ggplot.YLab("Active Users"),
		).
		Theme("fivethirtyeight")
	save(p, dir, "02_vline_events", 900, 550)
}

// ── 3. ABLine — regression / trend line ─────────────────────────────────
func ablineRegression(dir string) {
	n := 50

	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i)
		ys[i] = 2.3*xs[i] + 5 + 8*(rand.Float64()-0.5) // y ≈ 2.3x + 5 + noise
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)

	// Simple linear regression: y = mx + b
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i := range xs {
		sumX += xs[i]
		sumY += ys[i]
		sumXY += xs[i] * ys[i]
		sumX2 += xs[i] * xs[i]
	}

	fn := float64(n)
	slope := (fn*sumXY - sumX*sumY) / (fn*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / fn

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithColor("#3498DB"), geom.WithSize(4), geom.WithAlpha(0.6))).
		Layer(geom.ABLine(
			geom.WithSlope(slope),
			geom.WithIntercept(intercept),
			geom.WithColor("#E74C3C"),
			geom.WithLineWidth(2.5),
		)).
		Labs(
			ggplot.Title("Scatter with Regression Line"),
			ggplot.Subtitle(fmt.Sprintf("y = %.2fx + %.2f (ABLine)", slope, intercept)),
			ggplot.XLab("x"), ggplot.YLab("y"),
		).
		Theme("bmh")
	save(p, dir, "03_abline_regression", 900, 550)
}

// ── 4. Combined — all three reference line types ────────────────────────
func combined(dir string) {
	n := 80

	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i)
		ys[i] = 50*math.Sin(float64(i)*0.08) + 50 + 10*(rand.Float64()-0.5)
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#2C3E50"), geom.WithLineWidth(2))).
		// Mean line
		Layer(geom.HLine(
			geom.WithIntercept(50),
			geom.WithColor("#E74C3C"),
			geom.WithLineWidth(1.5),
		)).
		// Phase boundary
		Layer(geom.VLine(
			geom.WithIntercept(40),
			geom.WithColor("#27AE60"),
			geom.WithLineWidth(1.5),
		)).
		// Trend line
		Layer(geom.ABLine(
			geom.WithSlope(0.3),
			geom.WithIntercept(30),
			geom.WithColor("#8E44AD"),
			geom.WithLineWidth(2),
		)).
		Labs(
			ggplot.Title("Reference Lines — All Three Types"),
			ggplot.Subtitle("HLine (red=mean) · VLine (green=boundary) · ABLine (purple=trend)"),
			ggplot.XLab("x"), ggplot.YLab("y"),
		).
		Theme("seaborn_whitegrid")
	save(p, dir, "04_combined", 1000, 600)
}
