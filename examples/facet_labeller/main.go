// Example: Facet Labellers, Drop, and Grid Margins.
//
// Demonstrates FacetWrap with LabelBoth, FacetGrid with margins,
// grid strip labels (column headers + rotated row labels), and
// the drop/margins options.
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
	"github.com/TuSKan/ggplot/facet"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	ctx := context.Background()
	eng := memory.NewEngine(ctx)
	_, srcFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(srcFile)

	// Build a small iris-like dataset with species, petal, sepal, and region.
	n := 60 //nolint:mnd // small demo dataset.
	species := make([]string, n)
	region := make([]string, n)
	sepal := make([]float64, n)
	petal := make([]float64, n)

	sp := []string{"setosa", "versicolor", "virginica"}
	rg := []string{"north", "south"}

	for i := range n {
		species[i] = sp[i%3]                                         //nolint:mnd // cycle species.
		region[i] = rg[i%2]                                          //nolint:mnd // cycle regions.
		sepal[i] = 4.5 + float64(i%3)*1.2 + 0.3*math.Sin(float64(i)) //nolint:mnd // deterministic data.
		petal[i] = 1.0 + float64(i%3)*0.8 + 0.2*math.Cos(float64(i)) //nolint:mnd // deterministic data.
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewStringColumn("species", species),
		eng.NewStringColumn("region", region),
		eng.NewFloat64Column("sepal", sepal),
		eng.NewFloat64Column("petal", petal),
	)
	if err != nil {
		panic(err)
	}

	// ---------- 1. FacetWrap + LabelBoth ----------
	log.Println("01_facet_wrap_labelboth.png")

	p1 := ggplot.New(ds, aes.X("sepal"), aes.Y("petal"), aes.Color("species")).
		Layer(geom.Point()).
		FacetWrap("species", facet.WithLabeller(facet.LabelBoth()), facet.NCols(3)).
		Labels(ggplot.Title("FacetWrap + LabelBoth"), ggplot.XLabel("Sepal Length"), ggplot.YLabel("Petal Length")).
		Theme("minimal")
	file.Save(ctx, p1, filepath.Join(dir, "01_facet_wrap_labelboth.png"), 900, 350) //nolint:errcheck,mnd // demo.

	// ---------- 2. FacetGrid + Margins ----------
	log.Println("02_facet_grid_margins.png")

	p2 := ggplot.New(ds, aes.X("sepal"), aes.Y("petal"), aes.Color("species")).
		Layer(geom.Point()).
		FacetGrid("species", "region",
			facet.GridLabeller(facet.LabelBoth()),
			facet.GridMargins(true),
			facet.GridDrop(false),
		).
		Labels(ggplot.Title("FacetGrid + Margins"), ggplot.XLabel("Sepal Length"), ggplot.YLabel("Petal Length")).
		Theme("minimal")
	file.Save(ctx, p2, filepath.Join(dir, "02_facet_grid_margins.png"), 900, 700) //nolint:errcheck,mnd // demo.

	// ---------- 3. FacetWrap + Drop=false ----------
	log.Println("03_facet_wrap_nodrop.png")

	p3 := ggplot.New(ds, aes.X("sepal"), aes.Y("petal")).
		Layer(geom.Point(geom.WithColor("#3b82f6"))).
		FacetWrap("species", facet.WithDrop(false), facet.NCols(3)).
		Labels(ggplot.Title("FacetWrap — Drop=false"), ggplot.XLabel("Sepal"), ggplot.YLabel("Petal")).
		Theme("minimal")
	file.Save(ctx, p3, filepath.Join(dir, "03_facet_wrap_nodrop.png"), 900, 350) //nolint:errcheck,mnd // demo.

	// ---------- 4. FacetGrid + LabelValue (default) ----------
	log.Println("04_facet_grid_default.png")

	p4 := ggplot.New(ds, aes.X("sepal"), aes.Y("petal"), aes.Color("species")).
		Layer(geom.Point()).
		FacetGrid("species", "region").
		Labels(ggplot.Title("FacetGrid — Default Labels"), ggplot.XLabel("Sepal"), ggplot.YLabel("Petal")).
		Theme("minimal")
	file.Save(ctx, p4, filepath.Join(dir, "04_facet_grid_default.png"), 700, 500) //nolint:errcheck,mnd // demo.

	log.Println("All 4 facet examples generated.")
}
