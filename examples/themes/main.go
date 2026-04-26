// Example: Theme Showcase
//
// Demonstrates how to apply different built-in themes to the same plot.
// Available themes: "default", "dark", "minimal", "classic", "bw"
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
	"github.com/TuSKan/ggplot/theme"
)

func main() {
	// Generate sample data: sin wave with noise.
	n := 80
	xs := make([]float64, n)
	ys := make([]float64, n)
	for i := 0; i < n; i++ {
		xs[i] = float64(i) / float64(n) * 4 * math.Pi
		ys[i] = math.Sin(xs[i]) + rand.NormFloat64()*0.3
	}

	ds, err := dataset.NewDataset(memory.NewEngine(context.Background()), memory.NewEngine(context.Background()).NewFloat64Column("x", xs), memory.NewEngine(context.Background()).NewFloat64Column("y", ys))
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// Each theme renders the same plot with a different visual identity.
	themes := []theme.Name{theme.Default, theme.Dark, theme.Minimal, theme.Classic, theme.BW}

	for _, name := range themes {
		p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
			Layer(geom.Line(geom.WithColor("#3498DB"), geom.WithLineWidth(1.5))).
			Layer(geom.Point(geom.WithColor("#E74C3C"), geom.WithSize(2), geom.WithAlpha(0.5))).
			Theme(name).
			Labs(
				ggplot.Title("Theme: "+string(name)),
				ggplot.Subtitle("Sin wave with Gaussian noise"),
				ggplot.XLab("Angle (rad)"),
				ggplot.YLab("Amplitude"),
				ggplot.Caption("ggplot theme showcase"),
			)

		outPath := filepath.Join(dir, "theme_"+string(name)+".png")
		if err := p.Save(context.Background(), outPath, 800, 600); err != nil {
			log.Fatalf("theme %q: %v", name, err)
		}
		log.Printf("Saved %s", outPath)
	}
}
