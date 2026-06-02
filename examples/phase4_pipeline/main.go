// Phase 4 Pipeline: System Columns (PANEL, group) & Position Adjustments
//
// This example demonstrates:
//  1. Faceted grouped scatter — PANEL + group columns are injected into data
//  2. Stacked bar chart — geom.Stack accumulates Y offsets
//  3. Dodged grouped bars — geom.Dodge shifts groups side by side
//  4. 100% stacked (fill) bars — geom.Fill normalizes to [0,1]
package main

import (
	"context"
	"log"
	"math/rand"
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
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	facetedGrouped(dir)
	positionBars(dir)
}

func save(p *ggplot.Plot, dir, name string, w int) {
	out := filepath.Join(dir, name+".png")
	if err := file.Save(context.Background(), p, out, w, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}

// revenueDataset returns a shared dataset used by the bar chart examples.
func revenueDataset() dataset.Dataset {
	eng := memory.NewEngine(context.Background())

	ds, _ := dataset.NewDataset(eng,
		eng.NewStringColumn("quarter", []string{
			"Q1", "Q2", "Q3", "Q4",
			"Q1", "Q2", "Q3", "Q4",
			"Q1", "Q2", "Q3", "Q4",
		}),
		eng.NewFloat64Column("revenue", []float64{
			120, 150, 180, 200, // Hardware
			80, 90, 110, 130, // Software
			40, 60, 70, 90, // Services
		}),
		eng.NewStringColumn("product", []string{
			"Hardware", "Hardware", "Hardware", "Hardware",
			"Software", "Software", "Software", "Software",
			"Services", "Services", "Services", "Services",
		}),
	)

	return ds
}

// facetedGrouped demonstrates PANEL + group system columns:
// - Data is faceted by "region" -> each panel gets a PANEL column
// - Within each panel, aes.Color("product") creates group splits
// - Each group subset carries both PANEL and group int64 columns
func facetedGrouped(dir string) {
	rng := rand.New(rand.NewSource(42))

	const n = 120

	xs := make([]float64, 0, n)
	ys := make([]float64, 0, n)
	products := make([]string, 0, n)
	regions := make([]string, 0, n)

	for _, region := range []string{"East", "West"} {
		for _, product := range []string{"Alpha", "Beta", "Gamma"} {
			for i := range 20 {
				xs = append(xs, float64(i))
				base := map[string]float64{"Alpha": 10, "Beta": 20, "Gamma": 15}[product]

				if region == "West" {
					base += 5
				}

				ys = append(ys, base+rng.NormFloat64()*3)
				products = append(products, product)
				regions = append(regions, region)
			}
		}
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("day", xs),
		eng.NewFloat64Column("sales", ys),
		eng.NewStringColumn("product", products),
		eng.NewStringColumn("region", regions),
	)

	p := ggplot.New(ds, aes.X("day"), aes.Y("sales"), aes.Color("product")).
		Layer(geom.Line(geom.WithLineWidth(2))).
		FacetWrap("region", facet.NCols(2)).
		Labs(
			ggplot.Title("Phase 4.2: PANEL + group System Columns"),
			ggplot.Subtitle("Faceted by region, grouped by product"),
			ggplot.XLab("Day"),
			ggplot.YLab("Sales"),
		).
		Theme(theme.Dark)

	save(p, dir, "01_faceted_grouped", 1000)
}

// positionBars demonstrates geom.Stack, geom.Dodge, and geom.Fill.
func positionBars(dir string) {
	ds := revenueDataset()

	// Stacked bars (default for geom.Col).
	p1 := ggplot.New(ds, aes.X("quarter"), aes.Y("revenue"), aes.Color("product")).
		Layer(geom.Col()).
		Labs(
			ggplot.Title("Phase 4.3: Stacked Bars"),
			ggplot.Subtitle("Stack() — cumulative Y offsets"),
			ggplot.XLab("Quarter"),
			ggplot.YLab("Revenue ($K)"),
		).
		Theme(theme.Dark)
	save(p1, dir, "02_stacked_bars", 800)

	// Dodged bars.
	p2 := ggplot.New(ds, aes.X("quarter"), aes.Y("revenue"), aes.Color("product")).
		Layer(geom.Col(geom.WithPosition(geom.Dodge()))).
		Labs(
			ggplot.Title("Phase 4.3: Dodged Bars"),
			ggplot.Subtitle("Dodge() — side-by-side groups"),
			ggplot.XLab("Quarter"),
			ggplot.YLab("Revenue ($K)"),
		).
		Theme(theme.Minimal)
	save(p2, dir, "03_dodged_bars", 800)

	// Filled (100% stacked) bars.
	p3 := ggplot.New(ds, aes.X("quarter"), aes.Y("revenue"), aes.Color("product")).
		Layer(geom.Col(geom.WithPosition(geom.Fill()))).
		Labs(
			ggplot.Title("Phase 4.3: Filled Bars (100% Stacked)"),
			ggplot.Subtitle("Fill() — proportional stacking"),
			ggplot.XLab("Quarter"),
			ggplot.YLab("Proportion"),
		).
		Theme(theme.Classic)
	save(p3, dir, "04_filled_bars", 800)
}
