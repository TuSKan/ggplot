// Example: Facet Free Scales.
//
// Demonstrates FreeX, FreeY, FreeXY on faceted plots,
// strip styling, and ThemeOverride.
package main

import (
	"context"
	"log"
	"math"
	"math/rand/v2"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/facet"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/file"
	"github.com/TuSKan/ggplot/theme"
)

func main() {
	ctx := context.Background()
	eng := memory.NewEngine(ctx)
	_, srcFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(srcFile)

	// Deterministic RNG.
	rng := rand.New(rand.NewPCG(42, 99)) //nolint:mnd // Deterministic seed for example.

	// Build dataset: 3 species with very different Y ranges.
	n := 30 //nolint:mnd // 30 samples per species.
	species := make([]string, 0, n*3)
	xs := make([]float64, 0, n*3)
	ys := make([]float64, 0, n*3)

	for i := range n {
		x := float64(i) + rng.Float64()*0.5 //nolint:mnd // Jitter.

		species = append(species, "Setosa", "Versicolor", "Virginica")
		xs = append(xs, x, x+rng.Float64(), x+rng.Float64()*2) //nolint:mnd // Offset.
		ys = append(ys,
			1+rng.Float64()*2,                               //nolint:mnd // Setosa: [1, 3].
			10+rng.Float64()*5+math.Sin(float64(i)*0.3),     //nolint:mnd // Versicolor: [10, 15].
			50+rng.Float64()*30+10*math.Cos(float64(i)*0.2), //nolint:mnd // Virginica: [40, 90].
		)
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewStringColumn("species", species),
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		panic(err)
	}

	// 1. Shared scales (default) — all panels same Y range.
	p1 := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithColor("steelblue"))).
		FacetWrap("species").
		Labs(ggplot.Title("Shared Scales (Default)")).
		Theme("default")

	if err := file.Save(ctx, p1, filepath.Join(dir, "01_shared_scales.png"), 900, 300); err != nil { //nolint:mnd // Example.
		panic(err)
	}

	log.Println("01_shared_scales.png")

	// 2. FreeY — each panel has its own Y range.
	p2 := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithColor("coral"))).
		FacetWrap("species", facet.FreeY()).
		Labs(ggplot.Title("Free Y Scales")).
		Theme("default")

	if err := file.Save(ctx, p2, filepath.Join(dir, "02_free_y.png"), 900, 300); err != nil { //nolint:mnd // Example.
		panic(err)
	}

	log.Println("02_free_y.png")

	// 3. FreeXY — both axes independent.
	p3 := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithColor("seagreen"))).
		FacetWrap("species", facet.FreeXY()).
		Labs(ggplot.Title("Free X & Y Scales")).
		Theme("default")

	if err := file.Save(ctx, p3, filepath.Join(dir, "03_free_xy.png"), 900, 300); err != nil { //nolint:mnd // Example.
		panic(err)
	}

	log.Println("03_free_xy.png")

	// 4. Theme override — custom strip text color.
	p4 := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithColor("orchid"))).
		FacetWrap("species", facet.FreeY()).
		Labs(ggplot.Title("Custom Strip Styling")).
		Theme("default").
		ThemeOverride(
			theme.StripTextOverride(theme.ElementText{Bold: true, Size: 13}), //nolint:mnd // Custom size.
		)

	if err := file.Save(ctx, p4, filepath.Join(dir, "04_strip_style.png"), 900, 300); err != nil { //nolint:mnd // Example.
		panic(err)
	}

	log.Println("04_strip_style.png")
	log.Println("All 4 facet free-scale examples generated.")
}
