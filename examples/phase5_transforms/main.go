// Phase 5: Composable Stat Transforms — Filter, Sort, Group, Normalize, Stack, TopN
//
// Demonstrates the engine-native, lazy stat transform pipeline. Every
// transform composes Dataset verbs and engine interfaces — no manual
// []float64 iteration, no materialization until Draw.
package main

import (
	"context"
	"log"
	"math/rand"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/stat"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	normalizedHistogram(dir)
	percentageHistogram(dir)
	filteredScatter(dir)
	groupMean(dir)
	topNBars(dir)
	sortedScatter(dir)
}

func save(p *ggplot.Plot, dir, name string, w, h int) { //nolint:unparam // Example helper keeps w generic.
	out := filepath.Join(dir, name+".png")
	if err := p.Save(context.Background(), out, w, h); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}

// normalizedHistogram: BinX → NormalizeY pipeline
// Bins raw values then rescales counts to proportions (sum = 1.0).
func normalizedHistogram(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 800

	xs := make([]float64, n)
	for i := range xs {
		xs[i] = rng.NormFloat64()*15 + 100
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng, eng.NewFloat64Column("weight", xs))

	// BinX → NormalizeY: proportions that sum to 1.0
	p := ggplot.New(ds, aes.X("weight")).
		Layer(geom.Histogram(
			geom.Stat(stat.BinX(), stat.NormalizeY()),
			geom.WithFill("#1ABC9C"),
			geom.WithAlpha(0.85),
		)).
		Labs(
			ggplot.Title("Normalized Histogram"),
			ggplot.Subtitle("BinX → NormalizeY (proportions sum to 1.0)"),
		).
		Theme("minimal")

	save(p, dir, "01_normalized_histogram", 800, 500)
}

// percentageHistogram: BinX → NormalizeY(100) pipeline
// Like above, but rescaled so proportions sum to 100%.
func percentageHistogram(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 600

	xs := make([]float64, n)
	for i := range xs {
		xs[i] = rng.NormFloat64()*10 + 50
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng, eng.NewFloat64Column("score", xs))

	// BinX → NormalizeY(100): percentage histogram
	p := ggplot.New(ds, aes.X("score")).
		Layer(geom.Histogram(
			geom.Stat(stat.BinX(stat.WithBins(20)), stat.NormalizeY(stat.WithTotal(100))),
			geom.WithFill("#9B59B6"),
			geom.WithAlpha(0.8),
		)).
		Labs(
			ggplot.Title("Percentage Histogram"),
			ggplot.Subtitle("BinX → NormalizeY(100) — bars sum to 100%"),
		).
		Theme("bw")

	save(p, dir, "02_percentage_histogram", 800, 500)
}

// filteredScatter: FilterY using engine-native dataset.Gt predicate
// Keeps only rows where y > threshold, with no materialization.
func filteredScatter(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 200

	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = rng.Float64() * 100
		ys[i] = xs[i]*0.5 + rng.NormFloat64()*15
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)

	// FilterY keeps only rows where y > 30 (engine-native predicate)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(
			geom.Stat(stat.FilterY(dataset.Gt("y", 30.0))),
			geom.WithColor("#E74C3C"),
			geom.WithSize(3),
			geom.WithAlpha(0.7),
		)).
		Labs(
			ggplot.Title("Filtered Scatter"),
			ggplot.Subtitle("FilterY(Gt(\"y\", 30)) — engine-native predicate"),
		).
		Theme("dark")

	save(p, dir, "03_filtered_scatter", 800, 500)
}

// groupMean: GroupX("mean") — groups by x, computes mean of y per group.
// Uses engine Aggregator.Mean via GroupBy().Summarize().
func groupMean(dir string) {
	// Quarterly revenue for 3 regions
	regions := []float64{1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3}
	revenue := []float64{
		120, 135, 128, 142, // region 1
		200, 195, 210, 225, // region 2
		80, 92, 85, 78, // region 3
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("region", regions),
		eng.NewFloat64Column("revenue", revenue),
	)

	// GroupX("mean"): average revenue per region
	p := ggplot.New(ds, aes.X("region"), aes.Y("revenue")).
		Layer(geom.Col(
			geom.Stat(stat.GroupX("mean")),
			geom.WithFill("#3498DB"),
			geom.WithAlpha(0.85),
			geom.WithWidth(0.6),
		)).
		Labs(
			ggplot.Title("Average Revenue by Region"),
			ggplot.Subtitle("GroupX(\"mean\") — engine-native aggregation"),
		).
		Theme("minimal")

	save(p, dir, "04_group_mean", 800, 500)
}

// topNBars: BinX → TopN(5) — bin values then keep top 5 most frequent bins.
// Demonstrates multi-stage composition with engine-native Arrange + Tail.
func topNBars(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 1000

	// Skewed distribution — some bins will have many more values
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = rng.ExpFloat64() * 20
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng, eng.NewFloat64Column("latency", xs))

	// BinX(15 bins) → TopN(5, "count"): keep 5 most populated bins
	p := ggplot.New(ds, aes.X("latency")).
		Layer(geom.Histogram(
			geom.Stat(stat.BinX(stat.WithBins(15)), stat.TopN(5, "count")),
			geom.WithFill("#E67E22"),
			geom.WithAlpha(0.9),
		)).
		Labs(
			ggplot.Title("Top 5 Latency Bins"),
			ggplot.Subtitle("BinX(15) → TopN(5, \"count\") — most frequent bins only"),
		).
		Theme("classic")

	save(p, dir, "05_topn_bars", 800, 500)
}

// sortedScatter: SortBy("y", Desc()) on raw data → step line
// Demonstrates descending sort — the line connects points top-to-bottom,
// visually proving that data rows are reordered by y value.
func sortedScatter(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 30

	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i)
		ys[i] = rng.Float64()*80 + 10
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)

	// SortBy("y", Desc()): reorder rows so highest y comes first,
	// then plot as line — the line traces high→low visually.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(
			geom.WithColor("#95A5A6"),
			geom.WithSize(3),
			geom.WithAlpha(0.4),
		)).
		Layer(geom.Line(
			geom.Stat(stat.SortBy("y", stat.Desc())),
			geom.WithColor("#E74C3C"),
			geom.WithLineWidth(1.5),
			geom.WithAlpha(0.7),
		)).
		Labs(
			ggplot.Title("Sorted Line — Descending Y"),
			ggplot.Subtitle("SortBy(\"y\", Desc()) — line traces high → low"),
		).
		Theme("dark")

	save(p, dir, "06_sorted_scatter", 800, 500)
}
