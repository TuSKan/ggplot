// Example time demonstrates DTypeTime (time-of-day) columns, plotting
// intraday patterns with nanoseconds-since-midnight values.
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
)

const (
	nsPerSecond = int64(1e9)
	nsPerMinute = 60 * nsPerSecond //nolint:mnd // seconds per minute.
	nsPerHour   = 60 * nsPerMinute //nolint:mnd // minutes per hour.
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	eng := memory.NewEngine(context.Background())

	// Simulate 5-minute interval heart-rate readings from 06:00 to 22:00.
	startHour, endHour := 6, 22                   //nolint:mnd // observation window.
	intervalMin := 5                              //nolint:mnd // sampling interval.
	n := (endHour - startHour) * 60 / intervalMin //nolint:mnd // total samples.

	times := make([]int64, n)
	heartRate := make([]float64, n)

	for i := range n {
		minuteOfDay := startHour*60 + i*intervalMin //nolint:mnd // minutes per hour.
		times[i] = int64(minuteOfDay) * nsPerMinute

		// Simulated heart rate: resting ~65, peaks during exercise at 08:00 and 18:00.
		hour := float64(minuteOfDay) / 60 //nolint:mnd // convert back to fractional hours.
		mornDist := hour - 8              //nolint:mnd // morning exercise peak center.
		eveDist := hour - 18              //nolint:mnd // evening exercise peak center.
		heartRate[i] = 65 +               //nolint:mnd // resting heart rate.
			30*math.Exp(-0.5*mornDist*mornDist/0.5) + //nolint:mnd // morning exercise peak.
			25*math.Exp(-0.5*eveDist*eveDist/0.8) //nolint:mnd // evening exercise peak.
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewTimeColumn("time_of_day", times),
		eng.NewFloat64Column("bpm", heartRate),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("time_of_day"), aes.Y("bpm")).
		Layer(geom.Line(geom.WithColor("#E74C3C"), geom.WithLineWidth(2))).
		Labs(
			ggplot.Title("Heart Rate — Daily Pattern"),
			ggplot.Subtitle("DTypeTime: nanoseconds since midnight, 5-min intervals"),
			ggplot.XLab("Time of Day"),
			ggplot.YLab("Heart Rate (bpm)"),
		).
		Theme("bmh")

	out := filepath.Join(dir, "time.png")
	if err := p.Save(context.Background(), out, 900, 550, ggplot.WithCPU()); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}
