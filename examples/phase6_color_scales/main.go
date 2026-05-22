// Example: Phase 6 — Colour Scales & Legends
//
// Demonstrates the new colour features introduced in Phase 6:
//
//  1. CIELAB gradient constructors (Gradient, Gradient2, GradientN)
//  2. Theme-aware auto-selection (dark themes get Plasma, light get Viridis)
//  3. Legend key glyphs (circles for points, lines for smooth/line)
//  4. Guide customization (ColorBarWidth, ColorBarNBin)
//  5. NA color on colormap.Scale
package main

import (
	"context"
	"log"
	"math"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/colormap"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/theme"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	cieLABGradient(dir)
	divergingGradient(dir)
	multiStopGradient(dir)
	themeAwareAutoSelection(dir)
	legendKeyGlyphs(dir)
	guideCustomization(dir)

	log.Println("All Phase 6 examples generated successfully!")
}

// cieLABGradient demonstrates a 2-stop CIELAB gradient from white → blue.
// CIELAB interpolation produces perceptually uniform transitions — the
// midpoint looks equally different from both endpoints.
func cieLABGradient(dir string) {
	eng := memory.NewEngine(context.Background())

	// Generate a grid for a heatmap.
	n := 20
	xs := make([]float64, n*n)
	ys := make([]float64, n*n)
	zs := make([]float64, n*n)

	for i := range n {
		for j := range n {
			idx := i*n + j
			xs[idx] = float64(i)
			ys[idx] = float64(j)
			zs[idx] = math.Sin(float64(i)*0.3) * math.Cos(float64(j)*0.3)
		}
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("z", zs),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// 2-stop CIELAB gradient: white → blue
	white := colormap.Color{R: 1, G: 1, B: 1, A: 1}
	blue := colormap.Color{R: 0.1, G: 0.2, B: 0.8, A: 1}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("z")).
		Layer(geom.Tile()).
		ScaleColor(colormap.Gradient(white, blue)).
		Labs(
			ggplot.Title("CIELAB 2-Stop Gradient"),
			ggplot.Subtitle("White → Blue with perceptually uniform interpolation"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)

	outPath := filepath.Join(dir, "cielab_gradient.png")
	if err := p.Save(context.Background(), outPath, 700, 600); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// divergingGradient demonstrates a 3-stop diverging gradient (red → white → blue)
// for data with a meaningful midpoint (e.g. temperature anomalies).
func divergingGradient(dir string) {
	eng := memory.NewEngine(context.Background())

	n := 25
	xs := make([]float64, n*n)
	ys := make([]float64, n*n)
	zs := make([]float64, n*n)

	for i := range n {
		for j := range n {
			idx := i*n + j
			xs[idx] = float64(i) - float64(n)/2
			ys[idx] = float64(j) - float64(n)/2
			// Radial pattern centered at origin — positive near center, negative at edges.
			r := math.Sqrt(xs[idx]*xs[idx] + ys[idx]*ys[idx])
			zs[idx] = 1.0 - r/float64(n)*2
		}
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("anomaly", zs),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// 3-stop diverging gradient: cool blue → white → warm red
	cold := colormap.Color{R: 0.15, G: 0.30, B: 0.80, A: 1}
	neutral := colormap.Color{R: 0.97, G: 0.97, B: 0.97, A: 1}
	hot := colormap.Color{R: 0.85, G: 0.15, B: 0.15, A: 1}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("anomaly")).
		Layer(geom.Tile()).
		ScaleColor(colormap.Gradient2(cold, neutral, hot)).
		Labs(
			ggplot.Title("Diverging Gradient (3-Stop)"),
			ggplot.Subtitle("Blue → White → Red — midpoint at zero"),
			ggplot.XLab("Longitude"),
			ggplot.YLab("Latitude"),
		)

	outPath := filepath.Join(dir, "diverging_gradient.png")
	if err := p.Save(context.Background(), outPath, 700, 600); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// multiStopGradient demonstrates an N-stop custom gradient through
// multiple colors for terrain/elevation-style visualization.
func multiStopGradient(dir string) {
	eng := memory.NewEngine(context.Background())

	n := 30
	xs := make([]float64, n*n)
	ys := make([]float64, n*n)
	zs := make([]float64, n*n)

	for i := range n {
		for j := range n {
			idx := i*n + j
			xs[idx] = float64(i)
			ys[idx] = float64(j)
			// Mountain-like elevation pattern.
			cx, cy := float64(n)/2, float64(n)/2
			dx, dy := float64(i)-cx, float64(j)-cy
			zs[idx] = math.Exp(-(dx*dx+dy*dy)/(2*100)) * 4000
		}
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("elevation", zs),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// Terrain colormap: ocean blue → beach sand → forest green → snow white
	ocean := colormap.Color{R: 0.10, G: 0.20, B: 0.55, A: 1}
	sand := colormap.Color{R: 0.85, G: 0.80, B: 0.55, A: 1}
	forest := colormap.Color{R: 0.15, G: 0.55, B: 0.20, A: 1}
	snow := colormap.Color{R: 0.98, G: 0.98, B: 1.00, A: 1}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("elevation")).
		Layer(geom.Tile()).
		ScaleColor(colormap.GradientN([]colormap.Color{ocean, sand, forest, snow})).
		Labs(
			ggplot.Title("Multi-Stop Gradient (4 Colors)"),
			ggplot.Subtitle("Ocean → Sand → Forest → Snow — terrain elevation"),
			ggplot.XLab("Easting"),
			ggplot.YLab("Northing"),
		)

	outPath := filepath.Join(dir, "multistop_gradient.png")
	if err := p.Save(context.Background(), outPath, 700, 600); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// themeAwareAutoSelection demonstrates that different themes automatically
// select appropriate color palettes — dark themes get Plasma/Inferno,
// light themes get Viridis/Blues, accessibility themes get Cividis.
func themeAwareAutoSelection(dir string) {
	eng := memory.NewEngine(context.Background())

	// Simple grouped data.
	xs := []float64{1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5}
	ys := []float64{2, 4, 3, 7, 5, 1, 3, 5, 6, 8, 4, 2, 6, 4, 9}
	groups := []string{
		"Group A", "Group A", "Group A", "Group A", "Group A",
		"Group B", "Group B", "Group B", "Group B", "Group B",
		"Group C", "Group C", "Group C", "Group C", "Group C",
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewStringColumn("group", groups),
	)
	if err != nil {
		log.Fatalln(err)
	}

	themes := []theme.Name{"default", "dark", "okabe_ito"}
	labels := []string{
		"Default (Tab10)",
		"Dark (Observable10)",
		"Okabe-Ito (colorblind-safe)",
	}

	for i, th := range themes {
		p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
			Layer(geom.Point(geom.WithSize(6))).
			Layer(geom.Line()).
			Theme(th).
			Labs(
				ggplot.Title("Theme-Aware Palette: "+labels[i]),
				ggplot.Subtitle("No explicit ScaleColor — theme provides defaults"),
			)

		outPath := filepath.Join(dir, "theme_auto_"+string(th)+".png")
		if err := p.Save(context.Background(), outPath, 700, 500); err != nil {
			log.Fatalln(err)
		}

		log.Printf("Saved %s", outPath)
	}
}

// legendKeyGlyphs demonstrates that legend keys now match their geom type:
// circles for points, lines for smooth/line, rectangles for bars.
func legendKeyGlyphs(dir string) {
	eng := memory.NewEngine(context.Background())

	xs := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	ys := []float64{2.1, 4.3, 3.0, 7.8, 5.5, 8.1, 6.9, 9.2, 8.5, 10.0}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// Multi-layer plot with different geom types — each legend entry
	// gets a glyph matching its geom: point=circle, smooth=line.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(
			geom.WithColor("#E74C3C"),
			geom.WithSize(5),
			geom.WithLabel("Data Points"),
		)).
		Layer(geom.Smooth(
			geom.WithColor("#2980B9"),
			geom.WithLineWidth(2),
			geom.WithLabel("Trend Line"),
		)).
		Layer(geom.HLine(
			geom.WithIntercept(6),
			geom.WithColor("#27AE60"),
			geom.WithLineWidth(1),
			geom.WithLabel("Target"),
		)).
		Labs(
			ggplot.Title("Legend Key Glyphs"),
			ggplot.Subtitle("Each geom type gets its own legend shape: ● circle, ── line, ■ rect"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)

	outPath := filepath.Join(dir, "legend_key_glyphs.png")
	if err := p.Save(context.Background(), outPath, 800, 550); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}

// guideCustomization demonstrates ColorBarWidth, ColorBarNBin, and LegendCols
// for controlling the appearance of color bar and legend guides.
func guideCustomization(dir string) {
	eng := memory.NewEngine(context.Background())

	n := 20
	xs := make([]float64, n*n)
	ys := make([]float64, n*n)
	zs := make([]float64, n*n)

	for i := range n {
		for j := range n {
			idx := i*n + j
			xs[idx] = float64(i)
			ys[idx] = float64(j)
			zs[idx] = float64(i+j) / float64(2*n)
		}
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("value", zs),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// Wide color bar with stepped (8-bin) gradient.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("value")).
		Layer(geom.Tile()).
		ScaleColor(colormap.Plasma).
		ColorBarWidth(20).
		ColorBarNBin(8).
		Labs(
			ggplot.Title("Guide Customization"),
			ggplot.Subtitle("Wide color bar (20px) with 8 discrete steps"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)

	outPath := filepath.Join(dir, "guide_customization.png")
	if err := p.Save(context.Background(), outPath, 700, 600); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}
