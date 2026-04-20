// Example: Feature Showcase
//
// Demonstrates all new Phase 1 features in a single program:
//   - Color mapping with aes.Color()
//   - Automatic legend generation
//   - Step and Rug geometries
//   - XLim/YLim axis bounds
//   - CoordFlip
//   - Multiple themes
package main

import (
	"log"
	"math"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/geom"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// --- 1. Step function with color mapping ---
	stepExample(dir)

	// --- 2. Scatter with XLim/YLim ---
	limitsExample(dir)

	// --- 3. Horizontal bar chart (CoordFlip) ---
	flipExample(dir)

	// --- 4. Multi-layer: points + rug ---
	rugExample(dir)
}

func stepExample(dir string) {
	// Generate staircase data for 3 signals.
	n := 50
	xs := make([]float64, 0, n*3)
	ys := make([]float64, 0, n*3)
	labels := make([]string, 0, n*3)

	for i := 0; i < n; i++ {
		t := float64(i)
		xs = append(xs, t, t, t)
		ys = append(ys,
			math.Floor(math.Sin(t*0.2)*5)+5,  // signal A (0–10 range)
			math.Floor(math.Cos(t*0.15)*3)+6, // signal B (3–9 range)
			math.Floor(math.Sin(t*0.3)*2)+4,  // signal C (2–6 range)
		)
		labels = append(labels, "Sensor A", "Sensor B", "Sensor C")
	}

	ds, err := dataset.NewMixedDataFrame(
		dataset.WithFloat64s("time", xs),
		dataset.WithFloat64s("level", ys),
		dataset.WithStrings("sensor", labels),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds,
		aes.X("time"),
		aes.Y("level"),
		aes.Color("sensor"),
	).
		Layer(geom.Step(geom.WithLineWidth(2))).
		Labs(
			ggplot.Title("Step Function: Sensor Readings"),
			ggplot.Subtitle("Discrete signal levels over time"),
			ggplot.XLab("Time (s)"),
			ggplot.YLab("Level"),
		).
		Theme("minimal")

	out := filepath.Join(dir, "step_signals.png")
	if err := p.Save(out, 900, 500); err != nil {
		log.Fatalln(err)
	}
	log.Printf("Saved %s", out)
}

func limitsExample(dir string) {
	// Scatter with zoomed-in axis limits.
	n := 200
	xs := make([]float64, n)
	ys := make([]float64, n)
	groups := make([]string, n)

	for i := 0; i < n; i++ {
		t := float64(i) / float64(n) * 10
		xs[i] = t
		if i < n/2 {
			ys[i] = math.Sin(t) + 0.3*math.Sin(t*3)
			groups[i] = "Composite"
		} else {
			ys[i] = math.Cos(t) * 0.8
			groups[i] = "Damped"
		}
	}

	ds, _ := dataset.NewMixedDataFrame(
		dataset.WithFloat64s("x", xs),
		dataset.WithFloat64s("y", ys),
		dataset.WithStrings("type", groups),
	)

	p := ggplot.New(ds,
		aes.X("x"),
		aes.Y("y"),
		aes.Color("type"),
	).
		Layer(geom.Point(geom.WithSize(2), geom.WithAlpha(0.7))).
		XLim(2, 8).
		YLim(-1.5, 1.5).
		Labs(
			ggplot.Title("Scatter with Axis Limits"),
			ggplot.Subtitle("Zoomed into x=[2,8], y=[-1.5,1.5]"),
			ggplot.XLab("x"),
			ggplot.YLab("f(x)"),
		).
		Theme("bw")

	out := filepath.Join(dir, "axis_limits.png")
	if err := p.Save(out, 900, 500); err != nil {
		log.Fatalln(err)
	}
	log.Printf("Saved %s", out)
}

func flipExample(dir string) {
	ds, _ := dataset.NewDataFrame(map[string][]float64{
		"category": {1, 2, 3, 4, 5},
		"value":    {42, 28, 65, 53, 37},
	})

	p := ggplot.New(ds,
		aes.X("category"),
		aes.Y("value"),
	).
		Layer(geom.Col(geom.WithFill("#4A90D9"), geom.WithAlpha(0.85))).
		CoordFlip().
		Labs(
			ggplot.Title("Horizontal Bar Chart"),
			ggplot.Subtitle("Using CoordFlip()"),
			ggplot.XLab("Category"),
			ggplot.YLab("Value"),
		).
		Theme("classic")

	out := filepath.Join(dir, "coord_flip.png")
	if err := p.Save(out, 700, 500); err != nil {
		log.Fatalln(err)
	}
	log.Printf("Saved %s", out)
}

func rugExample(dir string) {
	n := 80
	xs := make([]float64, n)
	ys := make([]float64, n)
	groups := make([]string, n)

	for i := 0; i < n; i++ {
		if i < n/2 {
			xs[i] = 2 + 3*math.Sin(float64(i)*0.15)
			ys[i] = 1.5 + 2*math.Cos(float64(i)*0.2)
			groups[i] = "Cluster A"
		} else {
			xs[i] = 6 + 2*math.Cos(float64(i)*0.1)
			ys[i] = 4 + 1.5*math.Sin(float64(i)*0.12)
			groups[i] = "Cluster B"
		}
	}

	ds, _ := dataset.NewMixedDataFrame(
		dataset.WithFloat64s("x", xs),
		dataset.WithFloat64s("y", ys),
		dataset.WithStrings("cluster", groups),
	)

	p := ggplot.New(ds,
		aes.X("x"),
		aes.Y("y"),
		aes.Color("cluster"),
	).
		Layer(geom.Point(geom.WithSize(3), geom.WithAlpha(0.7))).
		Layer(geom.Rug(geom.WithAlpha(0.4))).
		Labs(
			ggplot.Title("Scatter with Marginal Rug"),
			ggplot.Subtitle("Rug marks show marginal distributions"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)

	out := filepath.Join(dir, "rug_scatter.png")
	if err := p.Save(out, 800, 600); err != nil {
		log.Fatalln(err)
	}
	log.Printf("Saved %s", out)
}
