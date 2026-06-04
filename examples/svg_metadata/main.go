// Example: SVG metadata channels, responsive output, and band padding.
//
// Demonstrates three features:
//
//  1. Responsive SVG — root <svg> includes style="max-width:100%;height:auto"
//  2. SVG metadata — <title> tooltips, <a href> links, aria-label for a11y
//  3. Band padding — WithPaddingInner / WithPaddingOuter on discrete scales
//
// Metadata channels flow through the standard ggplot pipeline: map a column
// via aes.Title / aes.Href / aes.AriaLabel, and the SVG output automatically
// emits the corresponding <title>, <a href>, and aria-label attributes.
package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/file"
	"github.com/TuSKan/ggplot/scale"
)

func main() {
	_, srcFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(srcFile)
	ctx := context.Background()

	svgMetadataPipeline(ctx, dir)
	bandPadding(ctx, dir)
}

// svgMetadataPipeline demonstrates per-bar SVG metadata wired through the
// standard ggplot pipeline. Each bar gets a <title> tooltip, <a href> link,
// and aria-label — all from data columns mapped via aes.Title/Href/AriaLabel.
//
// Open metadata_bars.svg in a browser: hover a bar for the tooltip, click for
// the link. The SVG is also responsive — it scales to container width.
func svgMetadataPipeline(ctx context.Context, dir string) {
	eng := memory.NewEngine(ctx)

	fruits := []string{"Apple", "Banana", "Cherry", "Date", "Elderberry"}
	sales := []float64{120, 85, 200, 60, 150}

	// Build tooltip and link columns from the data.
	tooltips := make([]string, len(fruits))
	links := make([]string, len(fruits))
	labels := make([]string, len(fruits))

	for i, f := range fruits {
		tooltips[i] = fmt.Sprintf("%s: %.0f units", f, sales[i])
		links[i] = "https://example.com/" + f
		labels[i] = "Bar for " + f
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewStringColumn("fruit", fruits),
		eng.NewFloat64Column("sales", sales),
		eng.NewStringColumn("tooltip", tooltips),
		eng.NewStringColumn("link", links),
		eng.NewStringColumn("a11y", labels),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds,
		aes.X("fruit"),
		aes.Y("sales"),
		aes.Title("tooltip"),  // → <title> tooltip on hover
		aes.Href("link"),      // → <a href> clickable link
		aes.AriaLabel("a11y"), // → aria-label for screen readers
	).
		Layer(geom.Col(geom.WithFill("#E67E22"))).
		Labels(
			ggplot.Title("Fruit Sales — SVG Metadata Demo"),
			ggplot.Subtitle("Hover for tooltips, click for links"),
			ggplot.XLabel("Fruit"),
			ggplot.YLabel("Units Sold"),
		).
		Theme("minimal")

	// Save as SVG — metadata is emitted automatically.
	out := filepath.Join(dir, "metadata_bars.svg")
	if err := file.Save(ctx, p, out, 700, 400); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Metadata SVG -> %s", filepath.Base(out))
	log.Println("  Hover bars for tooltips, click for links, inspect for aria-label")

	// Also save as PNG to show metadata is silently ignored for raster output.
	outPNG := filepath.Join(dir, "metadata_bars.png")
	if err := file.Save(ctx, p, outPNG, 700, 400); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Raster PNG -> %s (metadata silently ignored)", filepath.Base(outPNG))
}

// bandPadding demonstrates the new WithPaddingInner/WithPaddingOuter options
// on discrete scales that control bar spacing.
func bandPadding(ctx context.Context, dir string) {
	eng := memory.NewEngine(ctx)
	ds, _ := dataset.NewDataset(eng,
		eng.NewStringColumn("lang", []string{"Go", "Rust", "Python", "TypeScript", "Java"}),
		eng.NewFloat64Column("stars", []float64{120, 95, 180, 130, 80}),
	)

	// Default padding (inner=0.2, outer=0.5) — the standard look.
	p1 := ggplot.New(ds, aes.X("lang"), aes.Y("stars")).
		Layer(geom.Col(geom.WithFill("#3498DB"))).
		Labels(
			ggplot.Title("Default Padding"),
			ggplot.Subtitle("inner=0.2, outer=0.5"),
		).
		Theme("minimal")

	out1 := filepath.Join(dir, "band_default.png")
	if err := file.Save(ctx, p1, out1, 600, 400); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Band padding (default) -> %s", filepath.Base(out1))

	// Show the scale API directly.
	s := scale.Discrete(
		scale.WithPaddingInner(0.4), // wider gaps between bars
		scale.WithPaddingOuter(1.0), // more edge space
	)
	s.TrainValues([]string{"Go", "Rust", "Python", "TypeScript", "Java"})

	lo, hi := s.Bounds()

	log.Printf("Band padding API:")
	log.Printf("  PaddingInner=0.4, PaddingOuter=1.0")
	log.Printf("  Bounds: [%.2f, %.2f]", lo, hi)
	log.Printf("  BandWidth: %.2f", s.BandWidth())
}
