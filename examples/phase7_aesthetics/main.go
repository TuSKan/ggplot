//nolint:dupl // repetitive example structures are expected
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
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	sizeMapping(dir)
	shapeMapping(dir)
	linetypeMapping(dir)
	alphaMapping(dir)
	identityMapping(dir)

	log.Println("All Phase 7 examples generated successfully!")
}

// sizeMapping demonstrates continuous size mapping (linear and area-proportional).
func sizeMapping(dir string) {
	eng := memory.NewEngine(context.Background())

	xs := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	ys := []float64{1.5, 3.2, 2.5, 4.8, 4.0, 5.5, 6.2, 5.8, 7.5, 8.0}
	sizes := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("magnitude", sizes),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithColor("#4C72B0"), geom.WithAlpha(0.7)), aes.Size("magnitude")).
		ScaleSizeArea().
		Labs(
			ggplot.Title("Area-Proportional Size Scale"),
			ggplot.Subtitle("Circle radius proportional to sqrt(magnitude)"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)

	outPath := filepath.Join(dir, "size_mapping_area.png")
	if err := file.Save(context.Background(), p, outPath, 700, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// shapeMapping demonstrates discrete shape mapping with category labels.
func shapeMapping(dir string) {
	eng := memory.NewEngine(context.Background())

	xs := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}
	ys := []float64{2, 3, 5, 4, 6, 5, 8, 7, 9}
	categories := []string{"Type A", "Type B", "Type C", "Type A", "Type B", "Type C", "Type A", "Type B", "Type C"}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewStringColumn("type", categories),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// Shape manual configuration: maps Type A->square, Type B->triangle, Type C->star
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(6), geom.WithColor("#E74C3C")), aes.Shape("type")).
		ScaleShapeManual(map[string]string{
			"Type A": "square",
			"Type B": "triangle",
			"Type C": "star",
		}).
		Labs(
			ggplot.Title("Categorical Shape Mapping"),
			ggplot.Subtitle("Discrete types mapped to custom glyphs: square, triangle, star"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)

	outPath := filepath.Join(dir, "shape_mapping.png")
	if err := file.Save(context.Background(), p, outPath, 700, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// linetypeMapping demonstrates discrete linetype mapping and linetype-based grouping.
func linetypeMapping(dir string) {
	eng := memory.NewEngine(context.Background())

	// Build two line series
	xs := []float64{1, 2, 3, 4, 5, 1, 2, 3, 4, 5}
	ys := []float64{1, 3, 2, 4, 3, 2, 4, 5, 7, 6}
	groups := []string{"Series A", "Series A", "Series A", "Series A", "Series A", "Series B", "Series B", "Series B", "Series B", "Series B"}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewStringColumn("series", groups),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithLineWidth(2.5), geom.WithColor("#2ECC71")), aes.Linetype("series")).
		ScaleLinetype().
		Labs(
			ggplot.Title("Linetype Mapping & Grouping"),
			ggplot.Subtitle("Linetype mappings automatically group lines and assign dash patterns"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)

	outPath := filepath.Join(dir, "linetype_mapping.png")
	if err := file.Save(context.Background(), p, outPath, 700, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// alphaMapping demonstrates opacity mapping on points.
func alphaMapping(dir string) {
	eng := memory.NewEngine(context.Background())

	xs := make([]float64, 100)
	ys := make([]float64, 100)
	density := make([]float64, 100)

	for i := range xs {
		xs[i] = float64(i)
		ys[i] = math.Sin(float64(i) * 0.1)
		density[i] = math.Abs(ys[i])
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("density", density),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(5), geom.WithColor("#9B59B6")), aes.Alpha("density")).
		ScaleAlpha(0.1, 0.9).
		Labs(
			ggplot.Title("Continuous Alpha Mapping"),
			ggplot.Subtitle("Opacity maps continuously from 0.1 to 0.9 based on density"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)

	outPath := filepath.Join(dir, "alpha_mapping.png")
	if err := file.Save(context.Background(), p, outPath, 700, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// identityMapping demonstrates using raw column values directly for sizes and opacities.
func identityMapping(dir string) {
	eng := memory.NewEngine(context.Background())

	// Data column values represent exact pixel sizes and exact opacity fractions
	xs := []float64{1, 2, 3, 4, 5}
	ys := []float64{5, 4, 3, 2, 1}
	exactSizes := []float64{2, 6, 12, 18, 24} // physical pixel radii
	exactAlphas := []float64{0.2, 0.4, 0.6, 0.8, 1.0}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("my_size", exactSizes),
		eng.NewFloat64Column("my_alpha", exactAlphas),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithColor("#FF8C00")), aes.Size("my_size"), aes.Alpha("my_alpha")).
		ScaleSizeIdentity().
		ScaleAlphaIdentity().
		Labs(
			ggplot.Title("Identity Scale mapping"),
			ggplot.Subtitle("Bypasses normalization: size maps to [2..24]px, alpha maps to [0.2..1.0]"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)

	outPath := filepath.Join(dir, "identity_scale_mapping.png")
	if err := file.Save(context.Background(), p, outPath, 700, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}
