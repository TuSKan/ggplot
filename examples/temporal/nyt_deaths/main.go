// Example nyt_deaths demonstrates a journalism-style timeseries plot
// representing the 2020 excess deaths peak in New York City, styled
// after the New York Times visualizations with the "newsroom" theme.
package main

import (
	"context"
	"log"
	"math"
	"path/filepath"
	"runtime"
	"time"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output"
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	eng := memory.NewEngine(context.Background())

	n := 52 //nolint:mnd // 52 weeks in a year.
	dates := make([]time.Time, n)
	yminVals := make([]float64, n)
	ymaxVals := make([]float64, n)
	baselineVals := make([]float64, n)
	deaths2020 := make([]float64, n)
	labels := make([]string, n)

	// Start from Jan 1, 2020.
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC) //nolint:mnd // Year, month, day, and time coordinates.

	for i := range n {
		dates[i] = start.AddDate(0, 0, i*7) //nolint:mnd // 7 days per week.
		t := float64(i) / float64(n)

		// Seasonal baseline: higher in winter, lower in summer.
		// Modeled using a cosine wave mimicking the seasonal pattern.
		base := 1250.0 + 150.0*math.Cos(2.0*math.Pi*t) //nolint:mnd // Amplitude and baseline offset.
		baselineVals[i] = base
		yminVals[i] = base - 70.0 //nolint:mnd // Historical lower bound.
		ymaxVals[i] = base + 70.0 //nolint:mnd // Historical upper bound.

		// Model 2020 actual deaths with two COVID-19 wave peaks:
		// 1. Spring peak: extremely sharp and high, peaking around week 15 (mid-April).
		w := float64(i + 1)
		springPeakDiff := (w - 15.0) / 1.5                                  //nolint:mnd
		springPeak := 4400.0 * math.Exp(-0.5*springPeakDiff*springPeakDiff) //nolint:mnd // Gaussian peak matching the spring wave.

		// 2. Winter peak: broader and lower, peaking around week 50 (mid-December).
		winterPeakDiff := (w - 50.0) / 3.0                                 //nolint:mnd
		winterPeak := 350.0 * math.Exp(-0.5*winterPeakDiff*winterPeakDiff) //nolint:mnd // Gaussian peak matching the winter wave.

		deaths2020[i] = base + springPeak + winterPeak
	}

	// Add precise labels for notable timeline events.
	labels[14] = "April Peak:\n~5,800 deaths" //nolint:mnd // Week 14 corresponds to early April peak.
	labels[49] = "December Wave"              //nolint:mnd // Week 49 corresponds to early December wave.

	ds, err := dataset.NewDataset(eng,
		eng.NewDateFromTime("date", dates),
		eng.NewFloat64Column("ymin", yminVals),
		eng.NewFloat64Column("ymax", ymaxVals),
		eng.NewFloat64Column("baseline", baselineVals),
		eng.NewFloat64Column("deaths2020", deaths2020),
		eng.NewStringColumn("label", labels),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("date"), aes.Y("deaths2020"), aes.YMin("ymin"), aes.YMax("ymax")).
		// 1. Soft grey RibbonY representing the historical normal range.
		Layer(geom.RibbonY(nil,
			geom.WithFill("#D1D5DB"),
			geom.WithAlpha(0.5), //nolint:mnd // Standard transparency level.
			geom.WithLabel("Normal Range (2015–2019)"),
		)).
		// 2. Darker grey line for the historical average baseline.
		Layer(geom.Line(
			geom.WithColor("#6B7280"),
			geom.WithLineWidth(1.5), //nolint:mnd // Subtle reference line width.
			geom.WithLabel("Historical Average"),
		), aes.Y("baseline")).
		// 3. Thick crimson line for 2020 actual deaths.
		Layer(geom.Line(
			geom.WithColor("#E63946"),
			geom.WithLineWidth(3.0), //nolint:mnd // Main trend line thickness.
			geom.WithLabel("2020 Actual Deaths"),
		)).
		// 4. Text labels at peaks.
		Layer(geom.Text(
			geom.WithColor("#1F2937"),
			geom.WithFontSize(10), //nolint:mnd // Clean editorial size.
		), aes.Label("label")).
		Labels(
			ggplot.Title("Excess Deaths in New York City During 2020"),
			ggplot.Subtitle("Weekly deaths from all causes compared with the historical normal range"),
			ggplot.XLabel("Date"),
			ggplot.YLabel("Weekly Deaths"),
		).
		Theme("newsroom")

	out := filepath.Join(dir, "nyt_excess_deaths.png")
	// Save the plot with WithCPU() to ensure deterministic rendering in CI environment.
	if err := file.Save(context.Background(), p, out, 1000, 600, output.WithCPU(true)); err != nil { //nolint:mnd // Output aspect ratio.
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}
