//go:build !js

// Stress test: heavy interactive window with many layers, secondary axis,
// annotations, and 2000+ data points to benchmark rendering performance.
package main

import (
	"context"
	"log"
	"math"
	"math/rand/v2"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	_ "github.com/TuSKan/ggplot/canvas/gpu"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/window"
	"github.com/TuSKan/ggplot/scale"
)

func main() { //nolint:funlen // stress-test example; length is intentional.
	ctx := context.Background()
	eng := memory.NewEngine(ctx)
	rng := rand.New(rand.NewPCG(42, 99)) //nolint:mnd // Deterministic seed for reproducible benchmark.

	// --- Generate data ---
	const n = 1_000_000 //nolint:mnd // 1M data points for extreme stress test.

	xs := make([]float64, n)
	ySin := make([]float64, n)     // sin wave
	yCos := make([]float64, n)     // cos wave
	yScatter := make([]float64, n) // noisy scatter
	yBar := make([]float64, n)     // bar heights
	groups := make([]string, n)    // categorical grouping
	groupList := []string{"A", "B", "C", "D"}

	for i := range n {
		x := float64(i) * 0.05 //nolint:mnd // 0.05 step gives x range [0, 100].
		xs[i] = x
		ySin[i] = 10*math.Sin(x*0.3) + 2*math.Cos(x*0.7) //nolint:mnd // Complex wave.
		yCos[i] = 8*math.Cos(x*0.2) + 3*math.Sin(x*0.5)  //nolint:mnd // Second wave.
		yScatter[i] = ySin[i] + rng.NormFloat64()*3      //nolint:mnd // Noisy scatter around sin.
		yBar[i] = math.Abs(ySin[i]) * 0.5                //nolint:mnd // Positive bar heights.
		groups[i] = groupList[i%len(groupList)]
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("sin", ySin),
		eng.NewFloat64Column("cos", yCos),
		eng.NewFloat64Column("scatter", yScatter),
		eng.NewFloat64Column("bar", yBar),
		eng.NewStringColumn("group", groups),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// --- Build heavy plot ---
	plot := ggplot.New(ds, aes.X("x")).
		// Layer 1: scatter points with group color mapping
		Layer(geom.Point(
			geom.WithSize(2),    //nolint:mnd // Small scatter points.
			geom.WithAlpha(0.4), //nolint:mnd // Semi-transparent.
		), aes.Y("scatter"), aes.Color("group")).
		// Layer 2: sin wave line
		Layer(geom.Line(
			geom.WithColor("#E63946"), //nolint:mnd // Red.
			geom.WithLineWidth(2),     //nolint:mnd // Thick line.
		), aes.Y("sin")).
		// Layer 3: cos wave line
		Layer(geom.Line(
			geom.WithColor("#457B9D"), //nolint:mnd // Blue.
			geom.WithLineWidth(2),     //nolint:mnd // Thick line.
		), aes.Y("cos")).
		// Layer 4: smooth trend through scatter
		Layer(geom.Smooth(
			geom.WithColor("#2A9D8F"), //nolint:mnd // Teal.
			geom.WithLineWidth(3),     //nolint:mnd // Extra thick.
		), aes.Y("scatter")).
		// Layer 5: horizontal reference lines
		Layer(geom.HLine(
			geom.WithIntercept(0),
			geom.WithColor("#6C757D"), //nolint:mnd // Gray.
			geom.WithLineWidth(1),
		)).
		Layer(geom.HLine(
			geom.WithIntercept(10),    //nolint:mnd // Upper threshold.
			geom.WithColor("#F4A261"), //nolint:mnd // Orange.
		)).
		Layer(geom.HLine(
			geom.WithIntercept(-10),   //nolint:mnd // Lower threshold.
			geom.WithColor("#F4A261"), //nolint:mnd // Orange.
		)).
		// Layer 6: rug plot on X axis
		Layer(geom.Rug(
			geom.WithAlpha(0.1), //nolint:mnd // Very transparent rug.
			geom.WithColor("#264653"),
		), aes.Y("scatter")).
		// Annotations
		Annotate(ggplot.AnnotateText(50, 11, "Upper threshold", //nolint:mnd // Annotation position.
			geom.WithColor("#F4A261"),
			geom.WithFontSize(9), //nolint:mnd // Small font.
		)).
		Annotate(ggplot.AnnotateText(50, -11, "Lower threshold", //nolint:mnd // Annotation position.
			geom.WithColor("#F4A261"),
			geom.WithFontSize(9), //nolint:mnd // Small font.
		)).
		Annotate(ggplot.AnnotateRect(20, -5, 40, 5, //nolint:mnd // Highlight region.
			geom.WithFill("#2A9D8F"),
			geom.WithAlpha(0.08), //nolint:mnd // Very subtle highlight.
		)).
		Annotate(ggplot.AnnotateArrow(75, 8, 80, 5, //nolint:mnd // Arrow annotation.
			geom.WithColor("#E63946"),
		)).
		Annotate(ggplot.AnnotateLabel(80, 5, "Peak", //nolint:mnd // Label annotation.
			geom.WithColor("#E63946"),
			geom.WithFontSize(10), //nolint:mnd // Label font.
		)).
		// Secondary axis (2x scaling)
		SecondAxis(scale.SecAxis(
			func(v float64) float64 { return v * 1.8 }, //nolint:mnd // Primary → secondary.
			func(v float64) float64 { return v / 1.8 }, //nolint:mnd // Secondary → primary.
			"Scaled Value",
		)).
		// Labels
		Labs(
			ggplot.Title("Stress Test — 2000pts × 8 layers + annotations + secondary axis"),
			ggplot.XLab("Time (s)"),
			ggplot.YLab("Amplitude"),
		)

	if err := window.Show(ctx, plot,
		window.WithTitle("ggplot — stress test"),
		window.WithSize(1200, 700), //nolint:mnd // Larger window for complex plot.
		window.WithFPS(),
		window.WithPprof(),
	); err != nil {
		log.Fatalln(err)
	}
}
