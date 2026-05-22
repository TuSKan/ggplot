// Example scale demonstrates explicit use of ScaleX(scale.DateTime, ...)
// with custom options, and the OOB policies (Censor, Squish, Keep).
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
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	eng := memory.NewEngine(context.Background())

	explicitDateTimeScale(eng, dir)
	oobPolicies(eng, dir)
}

func save(p *ggplot.Plot, dir, name string, w, h int) {
	out := filepath.Join(dir, name+".png")
	if err := p.Save(context.Background(), out, w, h, ggplot.WithCPU()); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}

// explicitDateTimeScale shows ScaleX(scale.DateTime, ...) with formatter override.
func explicitDateTimeScale(eng *memory.Engine, dir string) {
	// Multi-year data: 2020–2024 quarterly GDP growth.
	dates := []string{
		"2020-01-01", "2020-04-01", "2020-07-01", "2020-10-01",
		"2021-01-01", "2021-04-01", "2021-07-01", "2021-10-01",
		"2022-01-01", "2022-04-01", "2022-07-01", "2022-10-01",
		"2023-01-01", "2023-04-01", "2023-07-01", "2023-10-01",
		"2024-01-01", "2024-04-01", "2024-07-01", "2024-10-01",
	}
	growth := []float64{
		2.1, -31.2, 33.8, 4.5,
		6.3, 6.7, 2.3, 6.9,
		-1.6, -0.6, 3.2, 2.6,
		2.0, 2.1, 4.9, 3.3,
		1.6, 2.8, 3.0, 2.5,
	}

	ts, err := eng.NewTimestampFromString("quarter", dates)
	if err != nil {
		log.Fatalln(err)
	}

	ds, err := dataset.NewDataset(eng,
		ts,
		eng.NewFloat64Column("gdp_growth", growth),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("quarter"), aes.Y("gdp_growth")).
		Layer(geom.Line(geom.WithColor("#2980B9"), geom.WithLineWidth(2))).
		Layer(geom.Point(geom.WithSize(3), geom.WithColor("#2980B9"))).
		ScaleX(scale.DateTime). // explicit DateTime scale
		Labs(
			ggplot.Title("US GDP Growth Rate"),
			ggplot.Subtitle("Quarterly annualized — explicit ScaleX(scale.DateTime)"),
			ggplot.XLab("Quarter"),
			ggplot.YLab("GDP Growth (%)"),
		).
		Theme("seaborn_whitegrid")

	save(p, dir, "datetime_scale", 1000, 550)
}

// oobPolicies demonstrates scale.WithOOB with Censor and Squish.
func oobPolicies(eng *memory.Engine, dir string) {
	// Generate a sine wave that exceeds [0,1] bounds.
	n := 100
	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range n {
		x := float64(i) * 0.1 //nolint:mnd // step size.
		xs[i] = x
		ys[i] = 0.5 + 0.7*math.Sin(x) //nolint:mnd // oscillates outside [0,1].
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// OOB Squish — clamps to [0, 1].
	pSquish := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#E67E22"), geom.WithLineWidth(2))).
		ScaleY(scale.Linear,
			scale.WithClipBounds(0, 1),
			scale.WithOOB(scale.OOBSquish),
		).
		Labs(
			ggplot.Title("OOB Squish"),
			ggplot.Subtitle("Out-of-bounds values clamped to [0, 1]"),
			ggplot.XLab("x"),
			ggplot.YLab("y"),
		).
		Theme("bmh")

	save(pSquish, dir, "oob_squish", 900, 500)
}
