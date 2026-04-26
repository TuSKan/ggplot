// Phase 2: Statistics — Identity, Bin/Count, Density (KDE), Smooth (LOESS), Summary, BoxPlot
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
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	identityStat(dir)
	binCountStat(dir)
	densityStat(dir)
	smoothStat(dir)
	boxplotStat(dir)
}

func save(p *ggplot.Plot, dir, name string, w, h int) {
	out := filepath.Join(dir, name+".png")
	if err := p.Save(context.Background(), out, w, h); err != nil {
		log.Fatalln(err)
	}
	log.Printf("Saved %s", out)
}

// Identity stat — raw data passed through (default for geom.Point, geom.Line)
func identityStat(dir string) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8}),
		memory.NewEngine().NewFloat64Column("y", []float64{3, 1, 4, 1, 5, 9, 2, 6}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#3498DB"), geom.WithLineWidth(2))).
		Layer(geom.Point(geom.WithSize(4), geom.WithColor("#E74C3C"))).
		Labs(ggplot.Title("stat: Identity"), ggplot.Subtitle("Raw data → line + points")).
		Theme("dark")
	save(p, dir, "01_stat_identity", 800, 500)
}

// Bin/Count stat — automatic histogram binning
func binCountStat(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 600
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = rng.NormFloat64()*12 + 100
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("weight", xs))
	p := ggplot.New(ds, aes.X("weight")).
		Layer(geom.Histogram(geom.WithFill("#1ABC9C"), geom.WithAlpha(0.8))).
		Labs(ggplot.Title("stat: Bin/Count"), ggplot.Subtitle("Automatic binning of weight distribution")).
		Theme("minimal")
	save(p, dir, "02_stat_bin", 800, 500)
}

// Density stat — kernel density estimation
func densityStat(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 500
	xs := make([]float64, n)
	for i := range xs {
		if rng.Float64() < 0.4 {
			xs[i] = rng.NormFloat64()*3 + 20
		} else {
			xs[i] = rng.NormFloat64()*5 + 40
		}
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("measurement", xs))
	p := ggplot.New(ds, aes.X("measurement")).
		Layer(geom.Density(geom.WithFill("#9B59B6"), geom.WithAlpha(0.5), geom.WithColor("#6C3483"))).
		Labs(ggplot.Title("stat: Density (KDE)"), ggplot.Subtitle("Kernel density of bimodal measurement data")).
		Theme("bw")
	save(p, dir, "03_stat_density", 800, 500)
}

// Smooth stat — LOESS regression
func smoothStat(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 100
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i) * 0.12
		ys[i] = 2*math.Sin(xs[i]) + 0.5*xs[i] + rng.NormFloat64()*0.8
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs), memory.NewEngine().NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(2), geom.WithAlpha(0.3), geom.WithColor("#95A5A6"))).
		Layer(geom.Smooth(geom.WithColor("#E74C3C"), geom.WithLineWidth(2.5))).
		Labs(ggplot.Title("stat: Smooth (LOESS)"), ggplot.Subtitle("Local polynomial regression through noisy data")).
		Theme("dark")
	save(p, dir, "04_stat_smooth", 800, 500)
}

// BoxPlot stat — five-number summary
func boxplotStat(dir string) {
	rng := rand.New(rand.NewSource(42))
	var x, y []float64
	for g := 1; g <= 4; g++ {
		for i := 0; i < 40; i++ {
			x = append(x, float64(g))
			y = append(y, float64(g)*10+rng.NormFloat64()*float64(g)*3)
		}
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("group", x), memory.NewEngine().NewFloat64Column("value", y))
	p := ggplot.New(ds, aes.X("group"), aes.Y("value")).
		Layer(geom.Boxplot(geom.WithFill("#F0E68C"), geom.WithColor("#2C3E50"), geom.WithWidth(0.5))).
		Labs(ggplot.Title("stat: BoxPlot"), ggplot.Subtitle("Five-number summary per group (variance increases with group)")).
		Theme("classic")
	save(p, dir, "05_stat_boxplot", 800, 500)
}
