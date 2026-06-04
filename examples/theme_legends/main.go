// Package main demonstrates Phase 12 theme features: plot margins with physical
// units, legend background/key/title styling, and continuous size/alpha legends.
package main

import (
	"context"
	"fmt"
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
	"github.com/TuSKan/ggplot/theme"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	plotMarginUnits(dir)
	legendTheming(dir)
	sizeLegend(dir)
	alphaLegend(dir)
	combinedAesthetics(dir)
	alignmentDemo(dir)

	fmt.Println("All Phase 12 theme examples generated successfully!")
}

// plotMarginUnits demonstrates Plot.PlotMargin with different physical units.
func plotMarginUnits(dir string) {
	eng := memory.NewEngine(context.Background())

	xs := make([]float64, 30)
	ys := make([]float64, 30)

	for i := range xs {
		xs[i] = float64(i)
		ys[i] = math.Sin(float64(i)*0.3) * 10 //nolint:mnd // Example sine wave.
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// Wide left margin (1cm) to accommodate long Y-axis labels,
	// generous top margin (0.5 inches) for the title area.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#2196F3"), geom.WithLineWidth(2))).
		PlotMargin(theme.PlotMargin{
			Top:    theme.Inches(0.5),
			Right:  theme.Cm(0.8),
			Bottom: theme.Pt(20),
			Left:   theme.Cm(1.0),
		}).
		Labels(
			ggplot.Title("Plot Margin — Unit Support"),
			ggplot.Subtitle("Top: 0.5in, Right: 0.8cm, Bottom: 20pt, Left: 1cm"),
			ggplot.XLabel("Time"),
			ggplot.YLabel("Amplitude"),
		).
		Theme("dashboard")

	outPath := filepath.Join(dir, "01_plot_margin_units.png")
	if err := file.Save(context.Background(), p, outPath, 800, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// legendTheming demonstrates legend.background, legend.key, and legend.title styling.
func legendTheming(dir string) {
	eng := memory.NewEngine(context.Background())

	xs := []float64{1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5}
	ys := []float64{2, 4, 3, 5, 6, 1, 3, 5, 4, 7, 3, 5, 4, 6, 8}
	groups := []string{
		"Alpha", "Alpha", "Alpha", "Alpha", "Alpha",
		"Beta", "Beta", "Beta", "Beta", "Beta",
		"Gamma", "Gamma", "Gamma", "Gamma", "Gamma",
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewStringColumn("group", groups),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Point(geom.WithSize(5))).
		ThemeOverride(
			theme.LegendTitleOverride(theme.ElementText{Bold: true, Size: 13}),
		).
		Labels(
			ggplot.Title("Legend Styling — Theme Elements"),
			ggplot.Subtitle("legend.background, legend.key, and legend.title overrides"),
			ggplot.XLabel("X"),
			ggplot.YLabel("Y"),
		).
		Theme("default")

	outPath := filepath.Join(dir, "02_legend_theming.png")
	if err := file.Save(context.Background(), p, outPath, 800, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// sizeLegend demonstrates a graduated-size legend from continuous size mapping.
func sizeLegend(dir string) {
	eng := memory.NewEngine(context.Background())

	xs := make([]float64, 20)
	ys := make([]float64, 20)
	pop := make([]float64, 20) // simulated "population" column

	for i := range xs {
		xs[i] = float64(i%5) + 1                    //nolint:mnd // 5 categories.
		ys[i] = float64(i/5) + 1                    //nolint:mnd // 4 rows.
		pop[i] = math.Pow(2, float64(i%5)+1) * 1000 //nolint:mnd // Exponential population.
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("longitude", xs),
		eng.NewFloat64Column("latitude", ys),
		eng.NewFloat64Column("population", pop),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("longitude"), aes.Y("latitude")).
		Layer(geom.Point(geom.WithColor("#E91E63"), geom.WithAlpha(0.6)), aes.Size("population")).
		ScaleSizeArea().
		Labels(
			ggplot.Title("Graduated Size Legend"),
			ggplot.Subtitle("Continuous size mapped to population — area-proportional circles"),
			ggplot.XLabel("Longitude"),
			ggplot.YLabel("Latitude"),
			ggplot.SizeLabel("Pop. (thousands)"),
		).
		Theme("dashboard")

	outPath := filepath.Join(dir, "03_size_legend.png")
	if err := file.Save(context.Background(), p, outPath, 800, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// alphaLegend demonstrates a continuous-alpha gradient legend.
func alphaLegend(dir string) {
	eng := memory.NewEngine(context.Background())

	n := 60 //nolint:mnd // Number of data points.
	xs := make([]float64, n)
	ys := make([]float64, n)
	confidence := make([]float64, n)

	for i := range xs {
		t := float64(i) / float64(n-1)
		xs[i] = t * 10                    //nolint:mnd // X range [0, 10].
		ys[i] = math.Sin(t*math.Pi*2) * 5 //nolint:mnd // Sine wave.
		confidence[i] = 0.3 + 0.7*t       //nolint:mnd // Ramp from 0.3 to 1.0.
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("time", xs),
		eng.NewFloat64Column("signal", ys),
		eng.NewFloat64Column("confidence", confidence),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("time"), aes.Y("signal")).
		Layer(geom.Point(geom.WithSize(4), geom.WithColor("#673AB7")), aes.Alpha("confidence")).
		ScaleAlpha(0.15, 1.0).
		Labels(
			ggplot.Title("Continuous Alpha Legend"),
			ggplot.Subtitle("Gradient strip shows the confidence → opacity mapping"),
			ggplot.XLabel("Time"),
			ggplot.YLabel("Signal"),
			ggplot.AlphaLabel("Confidence"),
		).
		Theme("default")

	outPath := filepath.Join(dir, "04_alpha_legend.png")
	if err := file.Save(context.Background(), p, outPath, 800, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// combinedAesthetics demonstrates size + alpha + color together with all three legend types.
func combinedAesthetics(dir string) {
	eng := memory.NewEngine(context.Background())

	n := 40 //nolint:mnd // Number of data points.
	xs := make([]float64, n)
	ys := make([]float64, n)
	magnitude := make([]float64, n)
	depth := make([]float64, n)
	groups := make([]string, n)

	categories := []string{"Shallow", "Mid", "Deep"}

	for i := range xs {
		t := float64(i) / float64(n-1)
		xs[i] = t * 100                       //nolint:mnd // X range [0, 100].
		ys[i] = math.Sin(t*math.Pi*3)*20 + 50 //nolint:mnd // Oscillating Y.
		magnitude[i] = 1 + t*9                //nolint:mnd // Magnitude 1..10.
		depth[i] = 0 + 1*(1-t)                //nolint:mnd // Depth ramps inversely.
		groups[i] = categories[i%len(categories)]
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("magnitude", magnitude),
		eng.NewFloat64Column("depth", depth),
		eng.NewStringColumn("type", groups),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("type")).
		Layer(geom.Point(), aes.Size("magnitude"), aes.Alpha("depth")).
		ScaleSizeArea().
		ScaleAlpha(0.2, 1.0).
		PlotMargin(theme.PlotMargin{
			Top:  theme.Cm(0.5),
			Left: theme.Cm(0.5),
		}).
		ThemeOverride(
			theme.LegendTitleOverride(theme.ElementText{Bold: true}),
		).
		Labels(
			ggplot.Title("Combined: Size + Alpha + Color Legends"),
			ggplot.Subtitle("Three legend types stacked in the right margin"),
			ggplot.XLabel("Distance (km)"),
			ggplot.YLabel("Elevation (m)"),
			ggplot.ColorLabel("Type"),
			ggplot.SizeLabel("Magnitude"),
			ggplot.AlphaLabel("Depth"),
		).
		Theme("dashboard")

	outPath := filepath.Join(dir, "05_combined_aesthetics.png")
	if err := file.Save(context.Background(), p, outPath, 900, 550); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// alignmentDemo showcases block-level alignment: left-aligned titles,
// left-aligned caption, right-aligned X label, and center-aligned legend.
func alignmentDemo(dir string) {
	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8}),
		eng.NewFloat64Column("y", []float64{3, 5, 4, 7, 6, 8, 7, 9}),
		eng.NewStringColumn("group", []string{"A", "A", "A", "A", "B", "B", "B", "B"}),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Point(geom.WithSize(5))).
		Labels(
			ggplot.Title("Left-Aligned Title"),
			ggplot.Subtitle("Subtitle follows the same alignment"),
			ggplot.XLabel("Measurement (units)"),
			ggplot.YLabel("Response"),
			ggplot.Caption("Source: alignment demo"),
		).
		Align(theme.BlockAlignment{
			Title:   theme.AlignLeft,
			Caption: theme.AlignLeft,
			XLabel:  theme.AlignRight,
			Legend:  theme.AlignCenter,
		}).
		Theme("dashboard")

	outPath := filepath.Join(dir, "06_alignment.png")
	if err := file.Save(context.Background(), p, outPath, 800, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}
