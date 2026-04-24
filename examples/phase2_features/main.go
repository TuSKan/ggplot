// Phase 2: Coordinates, Faceting, Themes, Guides, Aesthetics, LegendPosition
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
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/facet"
	"github.com/TuSKan/ggplot/geom"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	coordCartesian(dir)
	coordFlipped(dir)
	facetWrap(dir)
	facetGrid(dir)
	allThemes(dir)
	legendPositions(dir)
	aestheticsShowcase(dir)
}

func save(p *ggplot.Plot, dir, name string, w, h int) {
	out := filepath.Join(dir, name+".png")
	if err := p.Save(out, w, h); err != nil {
		log.Fatalln(err)
	}
	log.Printf("Saved %s", out)
}

// --- Coordinates ---

func coordCartesian(dir string) {
	n := 60
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i) * 0.1
		ys[i] = math.Sin(xs[i]) * 3
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs), memory.NewEngine().NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#3498DB"), geom.WithLineWidth(2))).
		Labs(ggplot.Title("Coord: Cartesian (default)"), ggplot.Subtitle("Standard x-y axes")).
		Theme("dark")
	save(p, dir, "01_coord_cartesian", 800, 500)
}

func coordFlipped(dir string) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewStringColumn("city", []string{"Tokyo", "Delhi", "Shanghai", "São Paulo", "Mumbai"}),
		memory.NewEngine().NewFloat64Column("population", []float64{37.4, 30.3, 27.1, 22.0, 20.7}),
	)
	p := ggplot.New(ds, aes.X("city"), aes.Y("population")).
		Layer(geom.Col(geom.WithFill("#E74C3C"), geom.WithAlpha(0.85))).
		CoordFlip().
		Labs(ggplot.Title("Coord: Flipped"), ggplot.Subtitle("Horizontal bar chart via CoordFlip()")).
		Theme("minimal")
	save(p, dir, "02_coord_flipped", 800, 500)
}

// --- Faceting ---

func facetWrap(dir string) {
	rng := rand.New(rand.NewSource(42))
	var xs, ys []float64
	var seasons []string
	for _, s := range []string{"Spring", "Summer", "Autumn", "Winter"} {
		for i := 0; i < 30; i++ {
			xs = append(xs, float64(i))
			base := map[string]float64{"Spring": 15, "Summer": 28, "Autumn": 18, "Winter": 5}[s]
			ys = append(ys, base+rng.NormFloat64()*3)
			seasons = append(seasons, s)
		}
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("day", xs),
		memory.NewEngine().NewFloat64Column("temp", ys),
		memory.NewEngine().NewStringColumn("season", seasons),
	)
	p := ggplot.New(ds, aes.X("day"), aes.Y("temp")).
		Layer(geom.Line(geom.WithColor("#2ECC71"), geom.WithLineWidth(1.5))).
		FacetWrap("season", facet.NCols(2)).
		Labs(ggplot.Title("Facet: Wrap"), ggplot.Subtitle("Temperature by season, wrapped 2 columns")).
		Theme("dark")
	save(p, dir, "03_facet_wrap", 900, 700)
}

func facetGrid(dir string) {
	rng := rand.New(rand.NewSource(42))
	var xs, ys []float64
	var regions, types []string
	for _, r := range []string{"North", "South"} {
		for _, t := range []string{"Urban", "Rural"} {
			for i := 0; i < 20; i++ {
				xs = append(xs, float64(i))
				base := 50.0
				if r == "North" {
					base += 10
				}
				if t == "Urban" {
					base += 15
				}
				ys = append(ys, base+rng.NormFloat64()*5)
				regions = append(regions, r)
				types = append(types, t)
			}
		}
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("month", xs),
		memory.NewEngine().NewFloat64Column("sales", ys),
		memory.NewEngine().NewStringColumn("region", regions),
		memory.NewEngine().NewStringColumn("type", types),
	)
	p := ggplot.New(ds, aes.X("month"), aes.Y("sales")).
		Layer(geom.Point(geom.WithSize(2.5), geom.WithColor("#9B59B6"), geom.WithAlpha(0.7))).
		FacetGrid("region", "type").
		Labs(ggplot.Title("Facet: Grid"), ggplot.Subtitle("Region × Type matrix")).
		Theme("bw")
	save(p, dir, "04_facet_grid", 900, 700)
}

// --- Themes ---

func allThemes(dir string) {
	n := 50
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i) * 0.15
		ys[i] = math.Sin(xs[i]) * 5
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs), memory.NewEngine().NewFloat64Column("y", ys))

	for _, name := range []string{"default", "classic", "minimal", "dark", "bw"} {
		p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
			Layer(geom.Line(geom.WithColor("#E74C3C"), geom.WithLineWidth(2))).
			Labs(ggplot.Title("Theme: "+name), ggplot.Subtitle("Same data, different theme")).
			Theme(name)
		save(p, dir, "05_theme_"+name, 700, 450)
	}
}

// --- Legend Positions ---

func legendPositions(dir string) {
	rng := rand.New(rand.NewSource(42))
	var xs, ys []float64
	var groups []string
	for _, g := range []string{"Alpha", "Beta", "Gamma"} {
		for i := 0; i < 30; i++ {
			xs = append(xs, float64(i)*0.2)
			base := map[string]float64{"Alpha": 0, "Beta": 2, "Gamma": 4}[g]
			ys = append(ys, base+math.Sin(float64(i)*0.2)+rng.NormFloat64()*0.3)
			groups = append(groups, g)
		}
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", xs),
		memory.NewEngine().NewFloat64Column("y", ys),
		memory.NewEngine().NewStringColumn("series", groups),
	)

	for _, pos := range []string{"right", "left", "top", "bottom", "none"} {
		p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("series")).
			Layer(geom.Line(geom.WithLineWidth(2))).
			LegendPosition(pos).
			Labs(ggplot.Title("Legend: "+pos), ggplot.Subtitle("LegendPosition(\""+pos+"\")")).
			Theme("dark")
		save(p, dir, "06_legend_"+pos, 700, 500)
	}
}

// --- Aesthetics: X, Y, Color, Group, Fill, Label, Size, Alpha ---

func aestheticsShowcase(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 60
	xs, ys, sizes := make([]float64, n), make([]float64, n), make([]float64, n)
	var groups []string
	for i := range xs {
		xs[i] = rng.Float64() * 10
		ys[i] = xs[i]*0.8 + rng.NormFloat64()*2
		sizes[i] = rng.Float64()*4 + 1
		if i < n/3 {
			groups = append(groups, "Group A")
		} else if i < 2*n/3 {
			groups = append(groups, "Group B")
		} else {
			groups = append(groups, "Group C")
		}
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", xs),
		memory.NewEngine().NewFloat64Column("y", ys),
		memory.NewEngine().NewStringColumn("group", groups),
	)
	p := ggplot.New(ds,
		aes.X("x"),
		aes.Y("y"),
		aes.Color("group"),
	).
		Layer(geom.Point(geom.WithSize(4), geom.WithAlpha(0.7))).
		Labs(
			ggplot.Title("Aesthetics: X, Y, Color, Alpha"),
			ggplot.Subtitle("Grouped scatter with color mapping"),
			ggplot.XLab("X value"),
			ggplot.YLab("Y value"),
		).
		Theme("dark")
	save(p, dir, "07_aesthetics", 800, 600)
}
