// Phase 5a: Scale Configuration — Breaks, Labels, Formatter, Expand,
// MinorBreaks, ClipBounds, and Binned.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/scale"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	breaksAndLabels(dir)
	currencyFormatter(dir)
	expandPadding(dir)
	minorGridLines(dir)
	clipBoundsZoom(dir)
	composedFeatures(dir)
	binnedScale(dir)
}

func save(p *ggplot.Plot, dir, name string, w, h int) {
	out := filepath.Join(dir, name+".png")
	if err := p.Save(context.Background(), out, w, h); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}

// ── 1. WithBreaks + WithLabels ──────────────────────────────────────────
// Custom tick positions and percentage labels on a completion chart.
func breaksAndLabels(dir string) {
	n := 100

	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i)
		ys[i] = 100 * (1 - math.Exp(-float64(i)*0.04)) // saturation curve
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("day", xs),
		eng.NewFloat64Column("completion", ys),
	)
	p := ggplot.New(ds, aes.X("day"), aes.Y("completion")).
		Layer(geom.Line(geom.WithColor("#3498DB"), geom.WithLineWidth(2.5))).
		Layer(geom.Point(geom.WithSize(1.5), geom.WithColor("#3498DB"), geom.WithAlpha(0.4))).
		ScaleY(scale.Linear,
			scale.WithBreaks([]float64{0, 25, 50, 75, 100}),
			scale.WithLabels([]string{"0%", "25%", "50%", "75%", "100%"}),
		).
		ScaleX(scale.Linear,
			scale.WithBreaks([]float64{0, 30, 60, 90}),
			scale.WithLabels([]string{"Start", "Month 1", "Month 2", "Month 3"}),
		).
		Labs(
			ggplot.Title("Project Completion Tracker"),
			ggplot.Subtitle("Custom breaks and labels on both axes"),
			ggplot.XLab("Timeline"),
			ggplot.YLab("Completion"),
		).
		Theme("seaborn_whitegrid")
	save(p, dir, "01_breaks_labels", 900, 550)
}

// ── 2. WithFormatter ────────────────────────────────────────────────────
// Currency formatting on Y-axis for revenue data.
func currencyFormatter(dir string) {
	months := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	revenue := []float64{
		12400, 15800, 14200, 18900, 22300, 25600,
		28100, 31400, 29700, 34200, 38900, 42500,
	}
	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("month", months),
		eng.NewFloat64Column("revenue", revenue),
	)
	p := ggplot.New(ds, aes.X("month"), aes.Y("revenue")).
		Layer(geom.Line(geom.WithColor("#27AE60"), geom.WithLineWidth(2.5))).
		Layer(geom.Point(geom.WithSize(4), geom.WithColor("#27AE60"))).
		ScaleY(scale.Linear,
			scale.WithFormatter(func(v float64) string {
				if v >= 1000 {
					return fmt.Sprintf("$%.0fk", v/1000)
				}

				return fmt.Sprintf("$%.0f", v)
			}),
		).
		ScaleX(scale.Linear,
			scale.WithBreaks([]float64{1, 3, 6, 9, 12}),
			scale.WithLabels([]string{"Jan", "Mar", "Jun", "Sep", "Dec"}),
		).
		Labs(
			ggplot.Title("Monthly Revenue"),
			ggplot.Subtitle("Y-axis with custom currency formatter"),
			ggplot.XLab(""),
			ggplot.YLab("Revenue (USD)"),
		).
		Theme("fivethirtyeight")
	save(p, dir, "02_formatter", 900, 550)
}

// ── 3. WithExpand ───────────────────────────────────────────────────────
// Side-by-side: default 5% padding vs explicit tight expand.
func expandPadding(dir string) {
	n := 40

	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i)
		ys[i] = math.Sin(float64(i)*0.2) * 10
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)

	// Tight padding: 1% multiplicative, 0 additive.
	tight := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#E74C3C"), geom.WithLineWidth(2))).
		ScaleX(scale.Linear, scale.WithExpand(0.01, 0)).
		ScaleY(scale.Linear, scale.WithExpand(0.01, 0)).
		Labs(
			ggplot.Title("Tight Expand (1%)"),
			ggplot.Subtitle("scale.WithExpand(0.01, 0) — minimal padding"),
			ggplot.XLab("x"), ggplot.YLab("sin(x)"),
		).
		Theme("bmh")
	save(tight, dir, "03_expand_tight", 900, 550)

	// Wide padding: 20% multiplicative + 2 additive units.
	wide := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#8E44AD"), geom.WithLineWidth(2))).
		ScaleX(scale.Linear, scale.WithExpand(0.20, 2)).
		ScaleY(scale.Linear, scale.WithExpand(0.15, 0)).
		Labs(
			ggplot.Title("Wide Expand (20% + 2)"),
			ggplot.Subtitle("scale.WithExpand(0.20, 2) — generous padding"),
			ggplot.XLab("x"), ggplot.YLab("sin(x)"),
		).
		Theme("bmh")
	save(wide, dir, "04_expand_wide", 900, 550)
}

// ── 4. WithMinorBreaks ──────────────────────────────────────────────────
// Minor grid lines for a scientific data plot.
func minorGridLines(dir string) {
	n := 200

	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		x := float64(i) * 0.1
		xs[i] = x
		ys[i] = math.Exp(-x*0.15) * math.Cos(x)
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("time", xs),
		eng.NewFloat64Column("amplitude", ys),
	)

	// Build minor breaks between majors: 1, 3, 5, 7, ...
	minorX := []float64{1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
	minorY := []float64{-0.75, -0.25, 0.25, 0.75}

	p := ggplot.New(ds, aes.X("time"), aes.Y("amplitude")).
		Layer(geom.Line(geom.WithColor("#2980B9"), geom.WithLineWidth(1.8))).
		ScaleX(scale.Linear,
			scale.WithBreaks([]float64{0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20}),
			scale.WithMinorBreaks(minorX),
		).
		ScaleY(scale.Linear,
			scale.WithBreaks([]float64{-1, -0.5, 0, 0.5, 1}),
			scale.WithMinorBreaks(minorY),
		).
		Labs(
			ggplot.Title("Damped Oscillation"),
			ggplot.Subtitle("Minor grid lines between major ticks"),
			ggplot.XLab("Time (s)"),
			ggplot.YLab("Amplitude"),
		).
		Theme("seaborn_whitegrid")
	save(p, dir, "05_minor_breaks", 1000, 550)
}

// ── 5. WithClipBounds ───────────────────────────────────────────────────
// Zoom into a specific region without filtering data.
func clipBoundsZoom(dir string) {
	n := 200

	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		x := float64(i) * 0.5
		xs[i] = x
		ys[i] = math.Sin(x*0.1)*50 + 100 + float64(i)*0.3
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)

	// Full view.
	full := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#E67E22"), geom.WithLineWidth(2))).
		Labs(
			ggplot.Title("Full Data Range"),
			ggplot.Subtitle("All 200 points visible"),
			ggplot.XLab("x"), ggplot.YLab("y"),
		).
		Theme("dark_background")
	save(full, dir, "06_clip_full", 900, 500)

	// Zoomed view using ClipBounds (data still present — not filtered).
	zoomed := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#E67E22"), geom.WithLineWidth(2))).
		ScaleX(scale.Linear, scale.WithClipBounds(20, 60)).
		ScaleY(scale.Linear, scale.WithClipBounds(90, 130)).
		Labs(
			ggplot.Title("Zoomed via ClipBounds"),
			ggplot.Subtitle("WithClipBounds(20,60) × (90,130) — data is NOT filtered"),
			ggplot.XLab("x"), ggplot.YLab("y"),
		).
		Theme("dark_background")
	save(zoomed, dir, "07_clip_zoomed", 900, 500)
}

// ── 6. Composed — all features together ─────────────────────────────────
func composedFeatures(dir string) {
	quarters := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	sales := []float64{45200, 52800, 61300, 58900, 71200, 83400, 92100, 105600}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("quarter", quarters),
		eng.NewFloat64Column("sales", sales),
	)

	p := ggplot.New(ds, aes.X("quarter"), aes.Y("sales")).
		Layer(geom.Line(geom.WithColor("#1ABC9C"), geom.WithLineWidth(2.5))).
		Layer(geom.Point(geom.WithSize(5), geom.WithColor("#1ABC9C"))).
		ScaleX(scale.Linear,
			scale.WithBreaks([]float64{1, 2, 3, 4, 5, 6, 7, 8}),
			scale.WithLabels([]string{
				"Q1'23", "Q2'23", "Q3'23", "Q4'23",
				"Q1'24", "Q2'24", "Q3'24", "Q4'24",
			}),
			scale.WithMinorBreaks([]float64{1.5, 2.5, 3.5, 4.5, 5.5, 6.5, 7.5}),
			scale.WithExpand(0.08, 0),
		).
		ScaleY(scale.Linear,
			scale.WithFormatter(func(v float64) string {
				return fmt.Sprintf("$%.0fk", v/1000)
			}),
			scale.WithExpand(0.05, 2000),
			scale.WithMinorBreaks([]float64{50000, 60000, 70000, 80000, 90000, 100000}),
		).
		Labs(
			ggplot.Title("Quarterly Sales Performance"),
			ggplot.Subtitle("Breaks + Labels + Formatter + Expand + MinorBreaks — all combined"),
			ggplot.XLab(""),
			ggplot.YLab("Revenue"),
		).
		Theme("seaborn_whitegrid")
	save(p, dir, "08_composed", 1000, 600)
}

// ── 9. BinnedScale ──────────────────────────────────────────────────────
// Discretize a continuous X axis into range-labeled bins.
func binnedScale(dir string) {
	rng := rand.New(rand.NewSource(42)) //nolint:mnd // Example uses deterministic seed.
	n := 150                            //nolint:mnd // number of students.

	scores := make([]float64, n)
	grades := make([]float64, n)

	for i := range n {
		scores[i] = 40 + rng.Float64()*60               //nolint:mnd // exam scores 40–100.
		grades[i] = scores[i]*0.8 + rng.NormFloat64()*8 //nolint:mnd // correlated course grade.
	}

	eng := memory.NewEngine(context.Background())

	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("exam_score", scores),
		eng.NewFloat64Column("course_grade", grades),
	)

	p := ggplot.New(ds, aes.X("exam_score"), aes.Y("course_grade")).
		Layer(geom.Point(geom.WithSize(2.5), geom.WithColor("#3498DB"), geom.WithAlpha(0.6))).
		ScaleX(scale.Binned, scale.WithBinBreaks([]float64{40, 50, 60, 70, 80, 90, 100})).
		Labs(
			ggplot.Title("Exam Score Bins"),
			ggplot.Subtitle("ScaleX(scale.Binned) — continuous axis grouped into range labels"),
			ggplot.XLab("Exam Score Range"),
			ggplot.YLab("Course Grade"),
		).
		Theme("seaborn_whitegrid")
	save(p, dir, "09_binned", 900, 550)
}
