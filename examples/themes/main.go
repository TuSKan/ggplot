// Example: Theme Showcase
//
// Demonstrates how to apply different built-in themes to the same plot.
// Available themes: "default", "dark", "minimal", "classic", "bw"
package main

import (
	"log"
	"math"
	"math/rand"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/geom"
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

	ds, err := dataset.NewDataFrame(map[string][]float64{"x": xs, "y": ys})
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// Each theme renders the same plot with a different visual identity.
	themes := []string{"default", "dark", "minimal", "classic", "bw"}

	for _, name := range themes {
		p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
			Layer(geom.Line(geom.WithColor("#3498DB"), geom.WithLineWidth(1.5))).
			Layer(geom.Point(geom.WithColor("#E74C3C"), geom.WithSize(2), geom.WithAlpha(0.5))).
			Theme(name).
			Labs(
				ggplot.Title("Theme: "+name),
				ggplot.Subtitle("Sin wave with Gaussian noise"),
				ggplot.XLab("Angle (rad)"),
				ggplot.YLab("Amplitude"),
				ggplot.Caption("ggplot theme showcase"),
			)

		outPath := filepath.Join(dir, "theme_"+name+".png")
		if err := p.Save(outPath, 800, 600); err != nil {
			log.Fatalf("theme %q: %v", name, err)
		}
		log.Printf("Saved %s", outPath)
	}
}
