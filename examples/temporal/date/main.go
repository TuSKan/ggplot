// Example date demonstrates the DTypeDate column using NewDateFromString
// and NewDateFromTime, plotting monthly metrics with calendar-aligned ticks.
package main

import (
	"context"
	"log"
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

	dateStrFromString(eng, dir)
	dateFromTime(eng, dir)
}

// dateStrFromString parses ISO date strings with NewDateFromString.
func dateStrFromString(eng *memory.Engine, dir string) {
	months := []string{
		"2024-01-15", "2024-02-15", "2024-03-15",
		"2024-04-15", "2024-05-15", "2024-06-15",
		"2024-07-15", "2024-08-15", "2024-09-15",
		"2024-10-15", "2024-11-15", "2024-12-15",
	}
	subscribers := []float64{
		1200, 1450, 1680, 1920, 2340, 2780,
		3150, 3520, 3890, 4200, 4650, 5100,
	}

	dateCol, err := eng.NewDateFromString("month", months)
	if err != nil {
		log.Fatalln(err)
	}

	ds, err := dataset.NewDataset(eng,
		dateCol,
		eng.NewFloat64Column("subscribers", subscribers),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("month"), aes.Y("subscribers")).
		Layer(geom.Line(geom.WithColor("#27AE60"), geom.WithLineWidth(2.5))).
		Layer(geom.Point(geom.WithSize(4), geom.WithColor("#27AE60"))).
		Labs(
			ggplot.Title("Subscriber Growth"),
			ggplot.Subtitle("Monthly count — DTypeDate from ISO strings"),
			ggplot.XLab("Month"),
			ggplot.YLab("Subscribers"),
		).
		Theme("seaborn_whitegrid")

	out := filepath.Join(dir, "date_string.png")
	if err := file.Save(context.Background(), p, out, 900, 550, output.WithCPU(true)); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}

// dateFromTime creates a date column from Go time.Time values.
func dateFromTime(eng *memory.Engine, dir string) {
	start := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	n := 90 //nolint:mnd // 90 days ≈ one quarter.

	times := make([]time.Time, n)
	revenue := make([]float64, n)

	for i := range n {
		times[i] = start.AddDate(0, 0, i)
		// Simulated daily revenue with weekly cycle.
		day := float64(i)
		revenue[i] = 5000 + 2000*dayOfWeekFactor(times[i]) + day*20 //nolint:mnd // synthetic revenue curve.
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewDateFromTime("date", times),
		eng.NewFloat64Column("revenue", revenue),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("date"), aes.Y("revenue")).
		Layer(geom.Line(geom.WithColor("#8E44AD"), geom.WithLineWidth(1.5))).
		Labs(
			ggplot.Title("Daily Revenue — Q2 2024"),
			ggplot.Subtitle("90 days from time.Time — weekly cycle visible"),
			ggplot.XLab("Date"),
			ggplot.YLab("Revenue ($)"),
		).
		Theme("fivethirtyeight")

	out := filepath.Join(dir, "date_time.png")
	if err := file.Save(context.Background(), p, out, 1000, 550, output.WithCPU(true)); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}

// dayOfWeekFactor returns a multiplier that drops on weekends.
func dayOfWeekFactor(t time.Time) float64 {
	switch t.Weekday() { //nolint:exhaustive // only weekend is special-cased.
	case time.Saturday, time.Sunday:
		return 0.4 //nolint:mnd // weekend drop factor.
	default:
		return 1.0
	}
}
