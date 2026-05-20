// Phase 5: Composable Stat Transforms — Filter, Sort, Group, Normalize,
// Stack, TopN, Select, Ribbon, Difference
//
// Demonstrates the engine-native, lazy stat transform pipeline. Every
// transform composes Dataset verbs and engine interfaces — no manual
// []float64 iteration, no materialization until Draw.
package main

import (
	"context"
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
	stackedArea(dir)
	groupReducers(dir)
	selectRow(dir)
	ribbonBand(dir)
	differenceFill(dir)
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

// stackedArea: StackY as a data transform — cumulative stacking within a group.
// Unlike position.Stack (which offsets bar X positions across groups),
// StackY accumulates Y values within a single series.
func stackedArea(dir string) {
	// Monthly values that accumulate
	months := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	sales := []float64{10, 15, 12, 18, 22, 25, 20, 28, 30, 35, 32, 40}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("month", months),
		eng.NewFloat64Column("sales", sales),
	)

	// StackY: cumulative sum of sales — each bar shows the running total.
	p := ggplot.New(ds, aes.X("month"), aes.Y("sales")).
		Layer(geom.Area(
			geom.Stat(stat.StackY()),
			geom.WithFill("#2ECC71"),
			geom.WithAlpha(0.7),
		)).
		Labs(
			ggplot.Title("Cumulative Sales"),
			ggplot.Subtitle("StackY — cumulative sum of Y within a single series"),
			ggplot.XLab("Month"),
			ggplot.YLab("Cumulative Sales"),
		).
		Theme("minimal")

	save(p, dir, "07_stacked_area", 800, 500)
}

// groupReducers: GroupX with extended reducer vocabulary.
// Demonstrates deviation, first, last, and mode reducers.
func groupReducers(dir string) {
	// Sensor readings: 5 sensors, 8 readings each
	sensors := make([]float64, 0, 40)
	readings := make([]float64, 0, 40)

	rng := rand.New(rand.NewSource(42))

	for s := 1; s <= 5; s++ {
		base := float64(s) * 10

		for range 8 {
			sensors = append(sensors, float64(s))
			readings = append(readings, base+rng.NormFloat64()*3)
		}
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("sensor", sensors),
		eng.NewFloat64Column("reading", readings),
	)

	// GroupX("deviation"): standard deviation of readings per sensor
	p := ggplot.New(ds, aes.X("sensor"), aes.Y("reading")).
		Layer(geom.Col(
			geom.Stat(stat.GroupX("deviation")),
			geom.WithFill("#E74C3C"),
			geom.WithAlpha(0.85),
			geom.WithWidth(0.6),
		)).
		Labs(
			ggplot.Title("Sensor Reading Variability"),
			ggplot.Subtitle("GroupX(\"deviation\") — std deviation per sensor"),
			ggplot.XLab("Sensor"),
			ggplot.YLab("Std Dev"),
		).
		Theme("minimal")

	save(p, dir, "08_group_deviation", 800, 500)

	// GroupX("first"): first reading per sensor
	p2 := ggplot.New(ds, aes.X("sensor"), aes.Y("reading")).
		Layer(geom.Col(
			geom.Stat(stat.GroupX("first")),
			geom.WithFill("#3498DB"),
			geom.WithAlpha(0.85),
			geom.WithWidth(0.6),
		)).
		Labs(
			ggplot.Title("First Reading per Sensor"),
			ggplot.Subtitle("GroupX(\"first\") — engine-native Aggregator.First"),
			ggplot.XLab("Sensor"),
			ggplot.YLab("First Reading"),
		).
		Theme("bw")

	save(p2, dir, "09_group_first", 800, 500)
}

// selectRow: SelectRow transform — keeps a single row from the dataset
// based on mode. Demonstrates SelectMin and SelectMax.
func selectRow(dir string) {
	// Temperature readings over 24 hours
	hours := make([]float64, 24)
	temps := make([]float64, 24)

	rng := rand.New(rand.NewSource(42))

	for i := range 24 {
		hours[i] = float64(i)
		// Simulate a daily temperature curve: warm midday, cool at night
		temps[i] = 15 + 10*rng.Float64() + 5*float64(12-abs(i-12))/12.0
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("hour", hours),
		eng.NewFloat64Column("temp", temps),
	)

	// Full scatter + highlight the row with max temperature
	p := ggplot.New(ds, aes.X("hour"), aes.Y("temp")).
		Layer(geom.Line(
			geom.WithColor("#BDC3C7"),
			geom.WithLineWidth(1.5),
		)).
		Layer(geom.Point(
			geom.WithColor("#95A5A6"),
			geom.WithSize(3),
		)).
		Layer(geom.Point(
			geom.Stat(stat.SelectRow(stat.SelectMax, "temp")),
			geom.WithColor("#E74C3C"),
			geom.WithSize(8),
			geom.WithLabel("Peak"),
		)).
		Layer(geom.Point(
			geom.Stat(stat.SelectRow(stat.SelectMin, "temp")),
			geom.WithColor("#3498DB"),
			geom.WithSize(8),
			geom.WithLabel("Trough"),
		)).
		Labs(
			ggplot.Title("Daily Temperature with Extremes"),
			ggplot.Subtitle("SelectRow(Max/Min, \"temp\") — highlight peak and trough"),
			ggplot.XLab("Hour"),
			ggplot.YLab("Temperature (°C)"),
		).
		Theme("dark")

	save(p, dir, "10_select_row", 800, 500)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}

// ribbonBand: RibbonY with pre-computed confidence bands.
// Demonstrates the ribbon geom rendering a filled band between
// ymin and ymax around a central trend line.
func ribbonBand(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 50

	xs := make([]float64, n)
	ys := make([]float64, n)
	yminVals := make([]float64, n)
	ymaxVals := make([]float64, n)

	for i := range n {
		x := float64(i) / float64(n-1) * 10 // 0..10
		xs[i] = x

		// Sinusoidal trend with growing uncertainty
		trend := math.Sin(x) * 3
		noise := rng.NormFloat64() * 0.5
		uncertainty := 0.5 + x*0.2 // band widens with x

		ys[i] = trend + noise
		yminVals[i] = trend - uncertainty
		ymaxVals[i] = trend + uncertainty
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("ymin", yminVals),
		eng.NewFloat64Column("ymax", ymaxVals),
	)

	// RibbonY layer: filled band between ymin/ymax.
	// Line layer: central trend overlaid on top.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.YMin("ymin"), aes.YMax("ymax")).
		Layer(geom.RibbonY(nil,
			geom.WithFill("#3498DB"),
			geom.WithAlpha(0.25),
		)).
		Layer(geom.Line(
			geom.WithColor("#2980B9"),
			geom.WithLineWidth(2),
		)).
		Labs(
			ggplot.Title("Confidence Band"),
			ggplot.Subtitle("RibbonY — filled band between ymin/ymax"),
			ggplot.XLab("Time"),
			ggplot.YLab("Signal"),
		).
		Theme("minimal")

	save(p, dir, "11_ribbon_band", 800, 500)
}

// differenceFill: Difference geom with two series.
// Shows the area between an observed and a reference series,
// rendered as a filled region.
func differenceFill(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 40

	xs := make([]float64, n)
	yminVals := make([]float64, n) // reference series (lower)
	ymaxVals := make([]float64, n) // observed series (upper)

	for i := range n {
		x := float64(i)
		xs[i] = x

		// Reference: steady baseline
		yminVals[i] = 50 + math.Sin(x/5)*5

		// Observed: baseline + random walk deviation
		ymaxVals[i] = yminVals[i] + rng.Float64()*20 - 5 // can cross below reference
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("ymin", yminVals),
		eng.NewFloat64Column("ymax", ymaxVals),
	)

	// Difference layer: fills between reference (ymin) and observed (ymax).
	// Overlaid with two lines showing the individual series.
	p := ggplot.New(ds, aes.X("x"), aes.YMin("ymin"), aes.YMax("ymax")).
		Layer(geom.Difference(nil,
			geom.WithFill("#E74C3C"),
			geom.WithAlpha(0.6),
		)).
		Labs(
			ggplot.Title("Series Difference"),
			ggplot.Subtitle("Difference — filled area between observed and reference"),
			ggplot.XLab("Day"),
			ggplot.YLab("Value"),
		).
		Theme("dark")

	save(p, dir, "12_difference_fill", 800, 500)
}
