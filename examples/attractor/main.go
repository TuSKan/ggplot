// Example: Strange Attractors Gallery
//
// Demonstrates:
//   - 3D ODE integration with RK4 for chaotic systems
//   - Continuous color mapping via aes.Color() on Z depth
//   - 100,000-point dense scatter plots with Z-depth coloring
//   - Five classic attractors: Lorenz, Rössler, Halvorsen, Thomas, Chen
package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
)

const numPoints = 100_000

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	lorenzExample(dir)
	rosslerExample(dir)
	halvorsenExample(dir)
	thomasExample(dir)
	chenExample(dir)
}

func plotAttractor(dir, filename, title, subtitle string, data Attractor3DData) {
	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", data.X),
		eng.NewFloat64Column("y", data.Y),
		eng.NewFloat64Column("z", data.Z),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds,
		aes.X("x"),
		aes.Y("y"),
		aes.Color("z"),
	).
		Layer(geom.Point(geom.WithSize(1.5), geom.WithAlpha(0.7))).
		Labs(
			ggplot.Title(title),
			ggplot.Subtitle(subtitle),
			ggplot.XLab("x"),
			ggplot.YLab("y"),
		).
		LegendPosition("none").
		Theme("dark")

	out := filepath.Join(dir, filename)
	if err := p.Save(context.Background(), out, 900, 900); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out) //nolint:forbidigo // Example intentionally logs output path.
}

func lorenzExample(dir string) {
	data := LorenzBeautiful(numPoints)
	plotAttractor(dir, "lorenz.png",
		"Lorenz Attractor",
		"100k points · σ=10, ρ=28, β=8/3 · XY projection",
		data,
	)
}

func rosslerExample(dir string) {
	data := RosslerBeautiful(numPoints)
	plotAttractor(dir, "rossler.png",
		"Rössler Attractor",
		"100k points · a=0.2, b=0.2, c=5.7 · XY projection",
		data,
	)
}

func halvorsenExample(dir string) {
	data := HalvorsenBeautiful(numPoints)
	plotAttractor(dir, "halvorsen.png",
		"Halvorsen Attractor",
		"100k points · a=1.3 · XY projection",
		data,
	)
}

func thomasExample(dir string) {
	data := ThomasBeautiful(numPoints)
	plotAttractor(dir, "thomas.png",
		"Thomas Attractor",
		"100k points · b=0.208186 · XY projection",
		data,
	)
}

func chenExample(dir string) {
	data := ChenBeautiful(numPoints)
	plotAttractor(dir, "chen.png",
		"Chen Attractor",
		"100k points · a=35, b=3, c=28 · XY projection",
		data,
	)
}
