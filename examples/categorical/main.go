// Example: Boxplot and Categorical (Discrete) X Axis
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
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	categoricalBars(dir)
	boxplot(dir)
}

// categoricalBars demonstrates a bar chart with string categories on the X axis.
func categoricalBars(dir string) {
	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewStringColumn("city", []string{
			"London", "Paris", "Berlin", "Madrid", "Rome",
		}),
		eng.NewFloat64Column("population", []float64{
			8.982, 2.161, 3.645, 3.223, 2.873,
		}),
	)

	p := ggplot.New(ds, aes.X("city"), aes.Y("population")).
		Layer(geom.Col(geom.WithFill("#3498DB"))).
		Labels(
			ggplot.Title("European City Populations"),
			ggplot.Subtitle("In millions"),
			ggplot.XLabel("City"),
			ggplot.YLabel("Population (M)"),
		)

	out := filepath.Join(dir, "categorical_bars.png")
	if err := file.Save(context.Background(), p, out, 800, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}

// boxplot demonstrates a box-and-whisker plot with multiple groups.
func boxplot(dir string) {
	rng := rand.New(rand.NewSource(42))

	// Generate 3 groups of normally distributed data with different means.
	x := make([]float64, 0, 150)
	y := make([]float64, 0, 150)

	groups := []struct {
		name string
		mean float64
		std  float64
	}{
		{"Control", 50, 10},
		{"Treatment A", 65, 12},
		{"Treatment B", 55, 8},
	}

	for _, g := range groups {
		for range 50 {
			// Use numeric group IDs for X.
			gID := float64(0)

			switch g.name {
			case "Control":
				gID = 1
			case "Treatment A":
				gID = 2
			case "Treatment B":
				gID = 3
			}

			x = append(x, gID)
			y = append(y, g.mean+g.std*rng.NormFloat64())
		}
	}

	// Clamp extreme values for cleaner display.
	for i := range y {
		y[i] = math.Max(0, y[i])
	}

	eng2 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng2,
		eng2.NewFloat64Column("group", x),
		eng2.NewFloat64Column("score", y),
	)

	p := ggplot.New(ds, aes.X("group"), aes.Y("score")).
		Layer(geom.Boxplot(
			geom.WithFill("#E8E8E8"),
			geom.WithColor("#2C3E50"),
			geom.WithWidth(0.6),
		)).
		Labels(
			ggplot.Title("Treatment Comparison"),
			ggplot.Subtitle("Box-and-whisker plot"),
			ggplot.XLabel("Group"),
			ggplot.YLabel("Score"),
		)

	out := filepath.Join(dir, "boxplot.png")
	if err := file.Save(context.Background(), p, out, 700, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}
