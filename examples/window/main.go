//go:build !js

// Stress test: 100K-point financial-style chart with multiple overlays.
// Tests rendering throughput with realistic production-like data volumes.
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
	rng := rand.New(rand.NewPCG(42, 99)) //nolint:mnd // Deterministic seed.

	// --- Simulate tick-level price data (100K ticks) ---
	const n = 10_000 //nolint:mnd // 10K data points — realistic daily tick volume.

	time := make([]float64, n)    // seconds since market open
	price := make([]float64, n)   // price series (random walk)
	volume := make([]float64, n)  // trade volume
	maShort := make([]float64, n) // 50-tick moving average
	maLong := make([]float64, n)  // 200-tick moving average
	signal := make([]string, n)   // buy/sell signal

	// Random walk price with drift and volatility.
	p := 100.0 //nolint:mnd // Initial price.

	for i := range n {
		time[i] = float64(i) * 0.36          //nolint:mnd // 0.36s per tick ≈ 10h trading day.
		p += rng.NormFloat64()*0.15 + 0.0001 //nolint:mnd // Brownian motion + slight drift.
		price[i] = p
		volume[i] = math.Abs(rng.NormFloat64()*50 + 100) //nolint:mnd // Volume spike distribution.

		switch {
		case i%500 < 250: //nolint:mnd // Alternating signal bands.
			signal[i] = "hold"
		case rng.Float64() < 0.5: //nolint:mnd // Random buy/sell within active zone.
			signal[i] = "buy"
		default:
			signal[i] = "sell"
		}
	}

	// Compute moving averages.
	const (
		shortWindow = 50  //nolint:mnd // Short MA window.
		longWindow  = 200 //nolint:mnd // Long MA window.
	)

	var shortSum, longSum float64

	for i := range n {
		shortSum += price[i]
		longSum += price[i]

		if i >= shortWindow {
			shortSum -= price[i-shortWindow]
			maShort[i] = shortSum / shortWindow
		} else {
			maShort[i] = shortSum / float64(i+1)
		}

		if i >= longWindow {
			longSum -= price[i-longWindow]
			maLong[i] = longSum / longWindow
		} else {
			maLong[i] = longSum / float64(i+1)
		}
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("time", time),
		eng.NewFloat64Column("price", price),
		eng.NewFloat64Column("volume", volume),
		eng.NewFloat64Column("ma_short", maShort),
		eng.NewFloat64Column("ma_long", maLong),
		eng.NewStringColumn("signal", signal),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// --- Build realistic overlay chart ---
	plot := ggplot.New(ds, aes.X("time")).
		// Layer 1: price scatter colored by signal
		Layer(geom.Point(
			geom.WithSize(1),
			geom.WithAlpha(0.3), //nolint:mnd // Semi-transparent scatter.
		), aes.Y("price"), aes.Color("signal")).
		// Layer 2: short MA line
		Layer(geom.Line(
			geom.WithColor("#E63946"), //nolint:mnd // Red.
			geom.WithLineWidth(2),     //nolint:mnd // Thick.
		), aes.Y("ma_short")).
		// Layer 3: long MA line
		Layer(geom.Line(
			geom.WithColor("#457B9D"), //nolint:mnd // Blue.
			geom.WithLineWidth(2),     //nolint:mnd // Thick.
		), aes.Y("ma_long")).
		// Layer 4: rug showing trade density
		Layer(geom.Rug(
			geom.WithAlpha(0.05), //nolint:mnd // Very subtle density rug.
			geom.WithColor("#264653"),
		), aes.Y("price")).
		// Annotations
		Annotate(ggplot.AnnotateText(18000, price[n-1]+2, "Latest", //nolint:mnd // Annotation.
			geom.WithColor("#2A9D8F"),
			geom.WithFontSize(10), //nolint:mnd // Label font.
		)).
		Annotate(ggplot.AnnotateText(5000, maShort[5000]+1, "MA(50)", //nolint:mnd // Label.
			geom.WithColor("#E63946"),
			geom.WithFontSize(9), //nolint:mnd // Small font.
		)).
		Annotate(ggplot.AnnotateText(5000, maLong[5000]-1, "MA(200)", //nolint:mnd // Label.
			geom.WithColor("#457B9D"),
			geom.WithFontSize(9), //nolint:mnd // Small font.
		)).
		// Secondary axis
		SecondAxis(scale.SecAxis(
			func(v float64) float64 { return (v - 100) * 100 }, //nolint:mnd // Price → basis points.
			func(v float64) float64 { return v/100 + 100 },     //nolint:mnd // BPS → price.
			"Change (bps)",
		)).
		Labs(
			ggplot.Title("Intraday Price — 100K ticks × 4 layers + annotations"),
			ggplot.XLab("Time (s)"),
			ggplot.YLab("Price ($)"),
		)

	if err := window.Show(ctx, plot,
		window.WithTitle("ggplot — 100K stress test"),
		window.WithSize(1200, 700), //nolint:mnd // Larger window.
		window.WithFPS(),
		window.WithPprof(),
	); err != nil {
		log.Fatalln(err)
	}
}
