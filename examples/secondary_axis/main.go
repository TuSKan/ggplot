// Example: Secondary Y-Axis.
//
// Demonstrates SecondAxis with a °C → °F transform and DupAxis.
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
	"github.com/TuSKan/ggplot/scale"
)

func main() {
	ctx := context.Background()
	eng := memory.NewEngine(ctx)
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)

	// Temperature data (°C).
	n := 24 //nolint:mnd // 24 hours.
	hours := make([]float64, n)
	tempC := make([]float64, n)

	for i := range n {
		hours[i] = float64(i)
		tempC[i] = 20 + 8*math.Sin(float64(i)*math.Pi/12) + 0.5*math.Cos(float64(i)*math.Pi/6) //nolint:mnd // deterministic temp wave.
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("hour", hours),
		eng.NewFloat64Column("temp_c", tempC),
	)
	if err != nil {
		panic(err)
	}

	// 1. Celsius primary + Fahrenheit secondary.
	err = ggplot.New(ds, aes.X("hour"), aes.Y("temp_c")).
		Layer(geom.Line(geom.WithColor("steelblue"), geom.WithLineWidth(2))). //nolint:mnd // Line styling.
		Layer(geom.Point(geom.WithColor("steelblue"))).
		Labs(
			ggplot.Title("Temperature — Dual Axis"),
			ggplot.XLab("Hour of Day"),
			ggplot.YLab("Temperature (°C)"),
		).
		SecondAxis(scale.SecAxis(
			func(c float64) float64 { return c*9/5 + 32 },       //nolint:mnd // °C → °F.
			func(f float64) float64 { return (f - 32) * 5 / 9 }, //nolint:mnd // °F → °C.
			"Temperature (°F)",
		)).
		Theme("default").
		Save(ctx, filepath.Join(dir, "01_secondary_axis_temp.png"), 800, 500) //nolint:mnd // example output.
	if err != nil {
		panic(err)
	}

	log.Println("01_secondary_axis_temp.png")

	// 2. DupAxis — mirror the same axis on the right.
	err = ggplot.New(ds, aes.X("hour"), aes.Y("temp_c")).
		Layer(geom.Line(geom.WithColor("coral"), geom.WithLineWidth(2))). //nolint:mnd // Line styling.
		Labs(
			ggplot.Title("Temperature — Duplicated Axis"),
			ggplot.XLab("Hour of Day"),
			ggplot.YLab("Temperature (°C)"),
		).
		SecondAxis(scale.DupAxis("Temperature (°C)")).
		Theme("dark").
		Save(ctx, filepath.Join(dir, "02_dup_axis.png"), 800, 500) //nolint:mnd // example output.
	if err != nil {
		panic(err)
	}

	log.Println("02_dup_axis.png")
	log.Println("All 2 secondary axis examples generated.")
}
