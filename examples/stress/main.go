//go:build !js

// Stress tests: battery of rendering benchmarks at escalating data volumes.
// Measures FPS for scatter, line, bar, facet, and composite workloads
// from 1K to 1M data points.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"strconv"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	_ "github.com/TuSKan/ggplot/canvas/gpu"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/window"
	"github.com/TuSKan/ggplot/scale"
)

func main() {
	ctx := context.Background()

	// Select test by CLI arg: scatter, line, composite, dense, million
	test := "scatter"
	if len(os.Args) > 1 {
		test = os.Args[1]
	}

	switch test {
	case "scatter":
		runScatter(ctx)
	case "line":
		runLine(ctx)
	case "composite":
		runComposite(ctx)
	case "dense":
		runDense(ctx)
	case "million":
		runMillion(ctx)
	default:
		fmt.Fprintf(os.Stderr, "unknown test: %s\nUsage: stress [scatter|line|composite|dense|million]\n", test)
		os.Exit(1)
	}
}

// runScatter: 50K scattered points with 4 color groups.
func runScatter(ctx context.Context) { //nolint:funlen // Stress test.
	eng := memory.NewEngine(ctx)
	rng := rand.New(rand.NewPCG(1, 2)) //nolint:mnd // Deterministic.

	const n = 50_000 //nolint:mnd // 50K points.

	xs := make([]float64, n)
	ys := make([]float64, n)
	groups := make([]string, n)
	labels := []string{"α", "β", "γ", "δ"}

	for i := range n {
		xs[i] = rng.NormFloat64() * 10           //nolint:mnd // Wide spread.
		ys[i] = rng.NormFloat64()*10 + xs[i]*0.3 //nolint:mnd // Correlated.
		groups[i] = labels[i%len(labels)]
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewStringColumn("group", groups),
	)
	if err != nil {
		log.Fatalln(err)
	}

	plot := ggplot.New(ds, aes.X("x")).
		Layer(geom.Point(
			geom.WithSize(2),    //nolint:mnd // Visible dots.
			geom.WithAlpha(0.4), //nolint:mnd // Semi-transparent.
		), aes.Y("y"), aes.Color("group")).
		Labels(
			ggplot.Title("Scatter Stress — 50K points × 4 groups"),
			ggplot.XLabel("X"), ggplot.YLabel("Y"),
		)

	if err := window.Show(ctx, plot,
		window.WithTitle("stress — scatter 50K"),
		window.WithSize(1200, 700), //nolint:mnd
		window.WithFPS(),
		window.WithPprof(),
	); err != nil {
		log.Fatalln(err)
	}
}

// runLine: 10 overlapping time-series lines, each with 100K points.
func runLine(ctx context.Context) { //nolint:funlen // Stress test.
	eng := memory.NewEngine(ctx)
	rng := rand.New(rand.NewPCG(3, 4)) //nolint:mnd // Deterministic.

	const (
		nPts   = 100_000 //nolint:mnd // Points per series.
		nLines = 10      //nolint:mnd // Number of overlapping lines.
	)

	xs := make([]float64, nPts)
	for i := range nPts {
		xs[i] = float64(i)
	}

	palette := []string{
		"#E63946", "#457B9D", "#2A9D8F", "#E9C46A", "#F4A261",
		"#264653", "#D62828", "#003049", "#F77F00", "#FCBF49",
	}

	cols := make([]dataset.AnyColumn, 1, 1+nLines)
	cols[0] = eng.NewFloat64Column("x", xs)
	names := make([]string, nLines)

	for li := range nLines {
		ys := make([]float64, nPts)
		price := rng.Float64() * 100 //nolint:mnd // Random start.

		for i := range nPts {
			price += rng.NormFloat64() * 0.1 //nolint:mnd // Small steps.
			ys[i] = price
		}

		names[li] = "s" + strconv.Itoa(li)
		cols = append(cols, eng.NewFloat64Column(names[li], ys))
	}

	ds, err := dataset.NewDataset(eng, cols...)
	if err != nil {
		log.Fatalln(err)
	}

	plot := ggplot.New(ds, aes.X("x"))
	for li := range nLines {
		plot = plot.Layer(geom.Line(
			geom.WithColor(palette[li]),
			geom.WithLineWidth(1.5), //nolint:mnd
			geom.WithAlpha(0.7),     //nolint:mnd
		), aes.Y(names[li]))
	}

	plot = plot.Labels(
		ggplot.Title(fmt.Sprintf("Line Stress — %d series × %dK points each", nLines, nPts/1000)),
		ggplot.XLabel("Tick"), ggplot.YLabel("Price"),
	)

	if err := window.Show(ctx, plot,
		window.WithTitle(fmt.Sprintf("stress — %d lines × %dK", nLines, nPts/1000)),
		window.WithSize(1200, 700), //nolint:mnd
		window.WithFPS(),
		window.WithPprof(),
	); err != nil {
		log.Fatalln(err)
	}
}

// runComposite: 100K points with scatter + line + area + rug + annotations.
func runComposite(ctx context.Context) { //nolint:funlen // Stress test.
	eng := memory.NewEngine(ctx)
	rng := rand.New(rand.NewPCG(5, 6)) //nolint:mnd // Deterministic.

	const n = 100_000 //nolint:mnd // 100K data points.

	xs := make([]float64, n)
	yWave := make([]float64, n)
	yNoise := make([]float64, n)

	for i := range n {
		x := float64(i) * 0.001 //nolint:mnd
		xs[i] = x
		yWave[i] = math.Sin(x*2*math.Pi) * 10      //nolint:mnd // Clean wave.
		yNoise[i] = yWave[i] + rng.NormFloat64()*2 //nolint:mnd // Noisy scatter.
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("wave", yWave),
		eng.NewFloat64Column("noisy", yNoise),
	)
	if err != nil {
		log.Fatalln(err)
	}

	plot := ggplot.New(ds, aes.X("x")).
		Layer(geom.Point(
			geom.WithSize(1),
			geom.WithAlpha(0.15), //nolint:mnd
			geom.WithColor("#264653"),
		), aes.Y("noisy")).
		Layer(geom.Line(
			geom.WithColor("#E63946"),
			geom.WithLineWidth(2), //nolint:mnd
		), aes.Y("wave")).
		Layer(geom.Rug(
			geom.WithAlpha(0.03), //nolint:mnd
			geom.WithColor("#457B9D"),
		), aes.Y("noisy")).
		Layer(geom.HLine(
			geom.WithIntercept(0),
			geom.WithColor("#2A9D8F"),
			geom.WithLineWidth(1),
		)).
		Labels(
			ggplot.Title("Composite Stress — 100K: scatter + line + rug + hline"),
			ggplot.XLabel("X"), ggplot.YLabel("Y"),
		)

	if err := window.Show(ctx, plot,
		window.WithTitle("stress — composite 100K"),
		window.WithSize(1200, 700), //nolint:mnd
		window.WithFPS(),
		window.WithPprof(),
	); err != nil {
		log.Fatalln(err)
	}
}

// runDense: 500K scatter points — tests pixel decimation at extreme density.
func runDense(ctx context.Context) { //nolint:funlen // Stress test.
	eng := memory.NewEngine(ctx)
	rng := rand.New(rand.NewPCG(7, 8)) //nolint:mnd // Deterministic.

	const n = 500_000 //nolint:mnd // 500K points.

	xs := make([]float64, n)
	ys := make([]float64, n)

	// Galaxy-like distribution: two overlapping Gaussian clusters.
	for i := range n {
		if i%2 == 0 {
			xs[i] = rng.NormFloat64()*3 + 5 //nolint:mnd // Cluster 1.
			ys[i] = rng.NormFloat64()*2 + 3 //nolint:mnd
		} else {
			xs[i] = rng.NormFloat64()*4 - 3 //nolint:mnd // Cluster 2.
			ys[i] = rng.NormFloat64()*3 - 2 //nolint:mnd
		}
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	plot := ggplot.New(ds, aes.X("x")).
		Layer(geom.Point(
			geom.WithSize(1),
			geom.WithAlpha(0.3), //nolint:mnd // Semi-transparent for density.
			geom.WithColor("#E63946"),
		), aes.Y("y")).
		Labels(
			ggplot.Title("Dense Scatter — 500K points (pixel decimation test)"),
			ggplot.XLabel("X"), ggplot.YLabel("Y"),
		)

	if err := window.Show(ctx, plot,
		window.WithTitle("stress — dense 500K"),
		window.WithSize(1200, 700), //nolint:mnd
		window.WithFPS(),
		window.WithPprof(),
	); err != nil {
		log.Fatalln(err)
	}
}

// runMillion: 1M scatter — ultimate GPU stress test.
func runMillion(ctx context.Context) { //nolint:funlen // Stress test.
	eng := memory.NewEngine(ctx)
	rng := rand.New(rand.NewPCG(9, 10)) //nolint:mnd // Deterministic.

	const n = 1_000_000 //nolint:mnd // 1M points.

	xs := make([]float64, n)
	ys := make([]float64, n)

	// Spiral distribution.
	for i := range n {
		t := float64(i) / float64(n) * 20 * math.Pi   //nolint:mnd // 10 revolutions.
		r := t * 0.5                                  //nolint:mnd // Expanding radius.
		xs[i] = r*math.Cos(t) + rng.NormFloat64()*0.5 //nolint:mnd // Noise.
		ys[i] = r*math.Sin(t) + rng.NormFloat64()*0.5 //nolint:mnd
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	plot := ggplot.New(ds, aes.X("x")).
		Layer(geom.Point(
			geom.WithSize(1),
			geom.WithAlpha(0.5), //nolint:mnd // Visible at density.
			geom.WithColor("#1D3557"),
		), aes.Y("y")).
		SecondAxis(scale.SecAxis(
			func(v float64) float64 { return v * 0.1 }, //nolint:mnd
			func(v float64) float64 { return v * 10 },  //nolint:mnd
			"Scaled",
		)).
		Labels(
			ggplot.Title("Million Points — 1M spiral (pixel decimation stress)"),
			ggplot.XLabel("X"), ggplot.YLabel("Y"),
		)

	if err := window.Show(ctx, plot,
		window.WithTitle("stress — 1M spiral"),
		window.WithSize(1200, 700), //nolint:mnd
		window.WithFPS(),
		window.WithPprof(),
	); err != nil {
		log.Fatalln(err)
	}
}
