// Example: Theme Showcase
//
// Renders the same multi-series plot under every built-in theme so the
// visual differences in panel chrome and discrete color cycle are
// directly comparable.
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
	// Generate sample data: five sin waves with different phase shifts so
	// the theme palette is visible across multiple series.
	const n = 80
	const series = 5
	xs := make([]float64, 0, n*series)
	ys := make([]float64, 0, n*series)
	groups := make([]string, 0, n*series)
	for s := 0; s < series; s++ {
		phase := float64(s) * math.Pi / 4
		label := []string{"A", "B", "C", "D", "E"}[s]
		for i := 0; i < n; i++ {
			x := float64(i) / float64(n) * 4 * math.Pi
			y := math.Sin(x+phase) + rand.NormFloat64()*0.15
			xs = append(xs, x)
			ys = append(ys, y)
			groups = append(groups, label)
		}
	}

	eng := memory.NewEngine(context.Background())
	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewStringColumn("series", groups),
	)
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	for _, name := range theme.AllNames() {
		p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("series")).
			Layer(geom.Line(geom.WithLineWidth(1.5))).
			Layer(geom.Point(geom.WithSize(2.5), geom.WithAlpha(0.7))).
			Theme(name).
			Labs(
				ggplot.Title("Theme: "+string(name)),
				ggplot.Subtitle("Phase-shifted sine waves with Gaussian noise"),
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
