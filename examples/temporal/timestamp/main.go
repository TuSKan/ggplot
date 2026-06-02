// Example timestamp demonstrates creating a time-series plot with
// timestamps parsed from strings. The DateTimeScale is auto-detected.
package main

import (
	"context"
	"log"
	"math"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output"
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	// Daily server response times over two weeks.
	dates := []string{
		"2024-01-01 08:00:00", "2024-01-02 08:00:00",
		"2024-01-03 08:00:00", "2024-01-04 08:00:00",
		"2024-01-05 08:00:00", "2024-01-06 08:00:00",
		"2024-01-07 08:00:00", "2024-01-08 08:00:00",
		"2024-01-09 08:00:00", "2024-01-10 08:00:00",
		"2024-01-11 08:00:00", "2024-01-12 08:00:00",
		"2024-01-13 08:00:00", "2024-01-14 08:00:00",
	}
	latency := []float64{
		120, 135, 128, 142, 155, 148, 130,
		125, 162, 178, 165, 145, 132, 118,
	}

	eng := memory.NewEngine(context.Background())

	ts, err := eng.NewTimestampFromString("time", dates)
	if err != nil {
		log.Fatalln(err)
	}

	ds, err := dataset.NewDataset(eng,
		ts,
		eng.NewFloat64Column("latency_ms", latency),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("time"), aes.Y("latency_ms")).
		Layer(geom.Line(geom.WithColor("#3498DB"), geom.WithLineWidth(2))).
		Layer(geom.Point(geom.WithSize(3), geom.WithColor("#2C3E50"))).
		Labs(
			ggplot.Title("Server Response Time"),
			ggplot.Subtitle("Two-week daily latency — auto DateTimeScale on X"),
			ggplot.XLab("Date"),
			ggplot.YLab("Latency (ms)"),
		).
		Theme("seaborn_whitegrid")

	_, filename, _, _ := runtime.Caller(0)

	out := filepath.Join(filepath.Dir(filename), "timestamp.png")
	if err := file.Save(context.Background(), p, out, 900, 550, output.WithCPU(true)); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)

	// --- Second plot: intraday timestamps (hours) ---
	hours := make([]string, 24) //nolint:mnd // 24 hours in a day.
	load := make([]float64, 24) //nolint:mnd // 24 hours in a day.

	for h := range 24 { //nolint:mnd // 24 hours in a day.
		hours[h] = "2024-06-15T" + pad(h) + ":00:00Z"
		// Simulated CPU load: peaks at 10:00 and 15:00.
		load[h] = 30 + 40*math.Exp(-0.5*math.Pow(float64(h-10), 2)/4) + //nolint:mnd // synthetic load curve.
			25*math.Exp(-0.5*math.Pow(float64(h-15), 2)/3) //nolint:mnd // synthetic load curve.
	}

	ts2, err := eng.NewTimestampFromString("hour", hours)
	if err != nil {
		log.Fatalln(err)
	}

	ds2, err := dataset.NewDataset(eng,
		ts2,
		eng.NewFloat64Column("cpu", load),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p2 := ggplot.New(ds2, aes.X("hour"), aes.Y("cpu")).
		Layer(geom.Line(geom.WithColor("#E74C3C"), geom.WithLineWidth(2.5))).
		Labs(
			ggplot.Title("CPU Load — Intraday"),
			ggplot.Subtitle("Hourly samples on 2024-06-15 — auto hour-level ticks"),
			ggplot.XLab("Time"),
			ggplot.YLab("CPU %"),
		).
		Theme("dark_background")

	out2 := filepath.Join(filepath.Dir(filename), "timestamp_intraday.png")
	if err := file.Save(context.Background(), p2, out2, 900, 550, output.WithCPU(true)); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out2)
}

func pad(h int) string {
	if h < 10 { //nolint:mnd // single-digit hour padding.
		return "0" + string(rune('0'+h))
	}

	return string(rune('0'+h/10)) + string(rune('0'+h%10)) //nolint:mnd // decimal digit extraction.
}
