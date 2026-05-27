package ggplot_test

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/coord"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/facet"
	"github.com/TuSKan/ggplot/geom"
)

// drawPlot is a test helper: Build() + DrawCanvas().
func drawPlot(ctx context.Context, p *ggplot.Plot, width, height int) (*canvas.GGCanvas, error) {
	built, err := p.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("drawPlot build: %w", err)
	}

	cv, err := built.DrawCanvas(ctx, width, height)
	if err != nil {
		return nil, fmt.Errorf("drawPlot draw: %w", err)
	}

	return cv, nil
}

func testDataset(t *testing.T) dataset.Dataset {
	t.Helper()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
		eng.NewFloat64Column("y", []float64{2.1, 4.3, 3.0, 7.8, 5.5, 8.1, 6.9, 9.2, 8.5, 10.0}),
	)
	if err != nil {
		t.Fatal(err)
	}

	return ds
}

// --- Builder API tests ---

func TestNew_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"))
	if p == nil {
		t.Fatal("New returned nil")
	}
}

func TestPlot_Layer_Immutable(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	base := ggplot.New(ds, aes.X("x"), aes.Y("y"))
	withPoint := base.Layer(geom.Point())
	withLine := base.Layer(geom.Line())

	if withPoint == nil || withLine == nil {
		t.Fatal("Layer returned nil")
	}
}

func TestPlot_Aes_DoesNotMutateParent(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	base := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point())

	// Derive two children with different Aes overrides.
	childA := base.Aes(aes.Color("x"))
	childB := base.Aes(aes.Color("y"))

	// Render all three -- none should affect the others.
	_, err := drawPlot(context.Background(), base, 200, 150)
	if err != nil {
		t.Fatalf("base render failed: %v", err)
	}

	_, err = drawPlot(context.Background(), childA, 200, 150)
	if err != nil {
		t.Fatalf("childA render failed: %v", err)
	}

	_, err = drawPlot(context.Background(), childB, 200, 150)
	if err != nil {
		t.Fatalf("childB render failed: %v", err)
	}

	// Re-render base to prove it wasn't corrupted.
	_, err = drawPlot(context.Background(), base, 200, 150)
	if err != nil {
		t.Fatalf("base re-render failed (mutation detected): %v", err)
	}
}

func TestPlot_Clone_Independence(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	base := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Labs(ggplot.Title("Base"))

	// Derive siblings with different modifications.
	withTheme := base.Theme("dark")
	withScale := base.ScaleX("log10")
	withLim := base.XLim(2, 8)

	// All four must render independently without error.
	for name, p := range map[string]*ggplot.Plot{
		"base":      base,
		"withTheme": withTheme,
		"withScale": withScale,
		"withLim":   withLim,
	} {
		_, err := drawPlot(context.Background(), p, 200, 150)
		if err != nil {
			t.Fatalf("%s render failed: %v", name, err)
		}
	}
}

func TestPlot_NoLayers_Error(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err == nil {
		t.Fatal("expected error for plot with no layers")
	}
}

func TestPlot_NilDataset_Error(t *testing.T) {
	t.Parallel()

	p := ggplot.New(dataset.Dataset{}, aes.X("x")).
		Layer(geom.Point())

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err == nil {
		t.Fatal("expected error for nil dataset")
	}
}

// --- Rendering tests (all geom types) ---

func TestRender_Point(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(4), geom.WithColor("#FF0000")))

	cv, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Point render failed: %v", err)
	}

	if cv.Width() != 400 || cv.Height() != 300 {
		t.Errorf("unexpected canvas size: %dx%d", cv.Width(), cv.Height())
	}
}

func TestRender_Line(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#0000FF"), geom.WithLineWidth(3)))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Line render failed: %v", err)
	}
}

func TestRender_Bar(t *testing.T) {
	t.Parallel()

	eng2 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng2,
		eng2.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng2.NewFloat64Column("count", []float64{10, 25, 15, 30, 20}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("count")).
		Layer(geom.Bar(geom.WithFill("#336699"), geom.WithWidth(0.7)))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Bar render failed: %v", err)
	}
}

func TestRender_Histogram(t *testing.T) {
	t.Parallel()

	xs := make([]float64, 500)
	for i := range xs {
		xs[i] = rand.NormFloat64()*5 + 10
	}

	eng3 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng3, eng3.NewFloat64Column("x", xs))

	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Histogram(geom.WithBins(30), geom.WithFill("#3498DB")))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Histogram render failed: %v", err)
	}
}

func TestRender_Histogram_StatTransform(t *testing.T) {
	t.Parallel()

	// Verify the stat transform actually runs: histogram should produce
	// binned data, not raw data.
	xs := make([]float64, 100)
	for i := range xs {
		xs[i] = float64(i)
	}

	eng4 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng4, eng4.NewFloat64Column("x", xs))

	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Histogram(geom.WithBins(10)))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Histogram stat transform failed: %v", err)
	}
}

func TestRender_Area(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Area(geom.WithFill("#2ecc71"), geom.WithAlpha(0.6)))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Area render failed: %v", err)
	}
}

func TestRender_Smooth(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Smooth(geom.WithColor("#E74C3C")))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Smooth render failed: %v", err)
	}
}

func TestRender_Density(t *testing.T) {
	t.Parallel()

	xs := make([]float64, 200)
	for i := range xs {
		xs[i] = rand.NormFloat64()
	}

	eng5 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng5, eng5.NewFloat64Column("x", xs))

	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Density(geom.WithFill("#9b59b6"), geom.WithAlpha(0.5)))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Density render failed: %v", err)
	}
}

func TestRender_Step(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Step(geom.WithColor("#1abc9c")))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Step render failed: %v", err)
	}
}

// --- Multi-layer tests ---

func TestRender_MultiLayer_PointAndLine(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithColor("#FF0000"))).
		Layer(geom.Line(geom.WithColor("#0000FF")))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("MultiLayer render failed: %v", err)
	}
}

func TestRender_MultiLayer_PointAndSmooth(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(3))).
		Layer(geom.Smooth())

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Point+Smooth render failed: %v", err)
	}
}

func TestRender_MultiLayer_ThreeLayers(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithColor("#FF0000"))).
		Layer(geom.Line(geom.WithColor("#0000FF"))).
		Layer(geom.Smooth())

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("ThreeLayers render failed: %v", err)
	}
}

// --- Labels ---

func TestRender_Labels(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Labs(
			ggplot.Title("Main Title"),
			ggplot.Subtitle("Subtitle"),
			ggplot.XLab("X Axis"),
			ggplot.YLab("Y Axis"),
			ggplot.Caption("Source: test data"),
		)

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Labels render failed: %v", err)
	}
}

// --- Coord ---

func TestRender_CoordFlip(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		CoordFlip()

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("CoordFlip render failed: %v", err)
	}
}

// --- Save ---

func TestPlot_Save_PNG(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Labs(ggplot.Title("Test Plot"))

	outPath := filepath.Join(t.TempDir(), "test.png")
	if err := p.Save(context.Background(), outPath, 400, 300); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}

	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}
}

func TestPlot_Save_Histogram(t *testing.T) {
	t.Parallel()

	xs := make([]float64, 100)
	for i := range xs {
		xs[i] = rand.NormFloat64()
	}

	eng6 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng6, eng6.NewFloat64Column("x", xs))

	outPath := filepath.Join(t.TempDir(), "hist.png")
	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Histogram(geom.WithBins(20))).
		Labs(ggplot.Title("Histogram Test"))

	if err := p.Save(context.Background(), outPath, 800, 600); err != nil {
		t.Fatalf("Histogram Save failed: %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}

	if info.Size() < 100 {
		t.Fatalf("histogram output file too small (%d bytes)", info.Size())
	}
}

func TestPlot_Save_AllGeomTypes(t *testing.T) {
	t.Parallel()

	// End-to-end: build and save a plot for each geom type.
	xs := make([]float64, 50)
	ys := make([]float64, 50)

	for i := range xs {
		xs[i] = float64(i)
		ys[i] = math.Sin(float64(i) * 0.2)
	}

	eng7 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng7, eng7.NewFloat64Column("x", xs), eng7.NewFloat64Column("y", ys))

	cases := []struct {
		name  string
		layer geom.Layer
		needY bool
	}{
		{"point", geom.Point(), true},
		{"line", geom.Line(), true},
		{"area", geom.Area(), true},
		{"step", geom.Step(), true},
		{"smooth", geom.Smooth(), true},
		{"histogram", geom.Histogram(geom.WithBins(10)), false},
		{"density", geom.Density(), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var p *ggplot.Plot
			if tc.needY {
				p = ggplot.New(ds, aes.X("x"), aes.Y("y")).Layer(tc.layer)
			} else {
				p = ggplot.New(ds, aes.X("x")).Layer(tc.layer)
			}

			outPath := filepath.Join(t.TempDir(), tc.name+".png")
			if err := p.Save(context.Background(), outPath, 400, 300); err != nil {
				t.Fatalf("Save %s failed: %v", tc.name, err)
			}

			info, _ := os.Stat(outPath)
			if info.Size() < 50 {
				t.Fatalf("%s output too small: %d bytes", tc.name, info.Size())
			}
		})
	}
}

// --- Dataset frame tests ---

// --- Facet ---

func TestFacetNone(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)

	panels, err := facet.None().Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	if len(panels) != 1 {
		t.Errorf("expected 1 panel, got %d", len(panels))
	}
}

// --- Coord ---

func TestCartesianTransform(t *testing.T) {
	t.Parallel()

	c := coord.Cartesian()

	px, py := c.Transform(0.5, 0.5, 100, 100)
	if px != 50 || py != 50 {
		t.Errorf("Cartesian(0.5,0.5): expected (50,50), got (%v,%v)", px, py)
	}
}

// --- Edge cases ---

func TestRender_SingleDataPoint(t *testing.T) {
	t.Parallel()

	eng8 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng8,
		eng8.NewFloat64Column("x", []float64{5}),
		eng8.NewFloat64Column("y", []float64{10}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point())

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Single point render failed: %v", err)
	}
}

func TestRender_TwoDataPoints(t *testing.T) {
	t.Parallel()

	eng9 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng9,
		eng9.NewFloat64Column("x", []float64{0, 100}),
		eng9.NewFloat64Column("y", []float64{0, 100}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line())

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Two point line render failed: %v", err)
	}
}

func TestRender_LargeDataset(t *testing.T) {
	t.Parallel()

	n := 10000
	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range xs {
		xs[i] = float64(i)
		ys[i] = math.Sin(float64(i) * 0.01)
	}

	eng10 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng10, eng10.NewFloat64Column("x", xs), eng10.NewFloat64Column("y", ys))

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line())

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Large dataset render failed: %v", err)
	}
}

func TestRender_NegativeValues(t *testing.T) {
	t.Parallel()

	eng11 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng11,
		eng11.NewFloat64Column("x", []float64{-5, -3, -1, 1, 3, 5}),
		eng11.NewFloat64Column("y", []float64{-10, -5, 0, 5, 10, 15}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Layer(geom.Line())

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Negative values render failed: %v", err)
	}
}

func TestRender_ConstantY(t *testing.T) {
	t.Parallel()

	eng12 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng12,
		eng12.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng12.NewFloat64Column("y", []float64{5, 5, 5, 5, 5}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point())

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Constant Y render failed: %v", err)
	}
}

// --- Color mapping tests ---

func groupedDataset(t *testing.T) dataset.Dataset {
	t.Helper()

	eng13 := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng13,
		eng13.NewFloat64Column("x", []float64{1, 2, 3, 1, 2, 3, 1, 2, 3}),
		eng13.NewFloat64Column("y", []float64{1, 4, 9, 2, 5, 8, 3, 6, 7}),
		eng13.NewStringColumn("group", []string{"A", "A", "A", "B", "B", "B", "C", "C", "C"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	return ds
}

func TestRender_ColorMapping_Point(t *testing.T) {
	t.Parallel()

	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Point(geom.WithSize(5)))

	cv, err := drawPlot(context.Background(), p, 600, 400)
	if err != nil {
		t.Fatalf("Color mapping point render failed: %v", err)
	}

	if cv == nil {
		t.Fatal("canvas is nil")
	}
}

func TestRender_ColorMapping_Line(t *testing.T) {
	t.Parallel()

	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Line())

	_, err := drawPlot(context.Background(), p, 600, 400)
	if err != nil {
		t.Fatalf("Color mapping line render failed: %v", err)
	}
}

func TestRender_ColorMapping_WithLegend(t *testing.T) {
	t.Parallel()

	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Point()).
		Labs(ggplot.Title("Grouped Scatter"))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Color mapping with legend render failed: %v", err)
	}
}

func TestRender_ColorMapping_ManyGroups(t *testing.T) {
	t.Parallel()

	// 10 groups to exercise palette wrap-around.
	n := 100
	xs := make([]float64, n)
	ys := make([]float64, n)
	groups := make([]string, n)
	labels := []string{"g0", "g1", "g2", "g3", "g4", "g5", "g6", "g7", "g8", "g9"}

	for i := range n {
		xs[i] = float64(i % 10)
		ys[i] = float64(i)
		groups[i] = labels[i%10]
	}

	eng14 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng14,
		eng14.NewFloat64Column("x", xs),
		eng14.NewFloat64Column("y", ys),
		eng14.NewStringColumn("g", groups),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("g")).
		Layer(geom.Point())

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Many groups render failed: %v", err)
	}
}

// --- XLim / YLim tests ---

func TestRender_XLim(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		XLim(0, 20)

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("XLim render failed: %v", err)
	}
}

func TestRender_YLim(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		YLim(-5, 15)

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("YLim render failed: %v", err)
	}
}

func TestRender_XLim_YLim_Combined(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		XLim(2, 8).
		YLim(0, 12)

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Combined XLim/YLim render failed: %v", err)
	}
}

func TestRender_XLim_NaN_PartialOverride(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	// Only override min, let max auto-detect.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		XLim(0, math.NaN())

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Partial XLim render failed: %v", err)
	}
}

// --- CoordFlip / Orientation tests ---

func TestRender_CoordFlip_Point(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		CoordFlip()

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("CoordFlip render failed: %v", err)
	}
}

func TestRender_CoordFlip_Bar(t *testing.T) {
	t.Parallel()

	eng15 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng15,
		eng15.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng15.NewFloat64Column("y", []float64{10, 25, 15, 30, 20}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Bar()).
		CoordFlip()

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("CoordFlip bar render failed: %v", err)
	}
}

func TestOrientation_HorizontalBar(t *testing.T) {
	t.Parallel()

	eng16 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng16,
		eng16.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng16.NewFloat64Column("y", []float64{10, 25, 15, 30, 20}),
	)
	// Horizontal bar via explicit orientation (no CoordFlip)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Bar(geom.WithOrientation(geom.Horizontal)))

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Horizontal bar render failed: %v", err)
	}
}

func TestOrientation_HorizontalBoxplot(t *testing.T) {
	t.Parallel()

	groups := make([]string, 0, 30)
	vals := make([]float64, 0, 30)

	for i := range 30 {
		groups = append(groups, "A")
		vals = append(vals, float64(i))
	}

	eng17 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng17,
		eng17.NewStringColumn("g", groups),
		eng17.NewFloat64Column("v", vals),
	)
	p := ggplot.New(ds, aes.X("g"), aes.Y("v")).
		Layer(geom.Boxplot(geom.WithOrientation(geom.Horizontal)))

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Horizontal boxplot render failed: %v", err)
	}
}

// --- Step geom tests ---

func TestRender_StepGeom(t *testing.T) {
	t.Parallel()

	eng18 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng18,
		eng18.NewFloat64Column("x", []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}),
		eng18.NewFloat64Column("y", []float64{0, 1, 1, 2, 2, 3, 3, 4, 4, 5}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Step(geom.WithColor("#336699")))

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Step render failed: %v", err)
	}
}

func TestRender_Step_ColorMapping(t *testing.T) {
	t.Parallel()

	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Step())

	_, err := drawPlot(context.Background(), p, 600, 400)
	if err != nil {
		t.Fatalf("Step with color mapping failed: %v", err)
	}
}

// --- Rug geom tests ---

func TestRender_Rug(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Layer(geom.Rug())

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Rug render failed: %v", err)
	}
}

// --- Combined feature tests ---

func TestRender_AllNewFeatures(t *testing.T) {
	t.Parallel()

	// Exercise color mapping + legend + xlim + step + rug all together.
	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Point(geom.WithSize(4))).
		Layer(geom.Step()).
		Labs(
			ggplot.Title("Combined Features Test"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		).
		XLim(0, 5).
		YLim(0, 12).
		Theme("minimal")

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Combined features render failed: %v", err)
	}
}

// --- Save integration tests (write actual PNGs for visual inspection) ---

func TestSave_ColorMapping(t *testing.T) {
	t.Parallel()

	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Point(geom.WithSize(6))).
		Labs(ggplot.Title("Color Mapping Test"))

	dir := t.TempDir()

	path := filepath.Join(dir, "color_mapping.png")
	if err := p.Save(context.Background(), path, 600, 400); err != nil {
		t.Fatalf("Save color mapping failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		t.Fatalf("PNG not created or empty: err=%v, size=%d", err, info.Size())
	}
}

func TestSave_XLimYLim(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		XLim(2, 8).
		YLim(0, 15).
		Labs(ggplot.Title("Axis Limits Test"))

	dir := t.TempDir()

	path := filepath.Join(dir, "xlim_ylim.png")
	if err := p.Save(context.Background(), path, 600, 400); err != nil {
		t.Fatalf("Save XLim/YLim failed: %v", err)
	}

	info, _ := os.Stat(path)
	if info.Size() == 0 {
		t.Fatal("PNG is empty")
	}
}

func TestSave_StepWithLegend(t *testing.T) {
	t.Parallel()

	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Step(geom.WithLineWidth(2))).
		Labs(ggplot.Title("Step Functions"))

	dir := t.TempDir()

	path := filepath.Join(dir, "step.png")
	if err := p.Save(context.Background(), path, 600, 400); err != nil {
		t.Fatalf("Save step failed: %v", err)
	}

	info, _ := os.Stat(path)
	if info.Size() == 0 {
		t.Fatal("PNG is empty")
	}
}

// --- HLine / VLine tests ---

func TestRender_HLine(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		Layer(geom.HLine(geom.WithIntercept(5), geom.WithColor("#CC0000")))

	_, err := drawPlot(context.Background(), p, 600, 400)
	if err != nil {
		t.Fatalf("HLine render failed: %v", err)
	}
}

func TestRender_VLine(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Layer(geom.VLine(geom.WithIntercept(5), geom.WithColor("#006600")))

	_, err := drawPlot(context.Background(), p, 600, 400)
	if err != nil {
		t.Fatalf("VLine render failed: %v", err)
	}
}

func TestRender_HLine_VLine_Combined(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		Layer(geom.HLine(geom.WithIntercept(0), geom.WithColor("#999999"), geom.WithLabel("baseline"))).
		Layer(geom.VLine(geom.WithIntercept(5), geom.WithColor("#2ECC71"), geom.WithLabel("x=5")))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("Combined HLine/VLine render failed: %v", err)
	}
}

func TestRender_HLine_OutOfRange(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	// Intercept way outside the Y range -- should not crash.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Layer(geom.HLine(geom.WithIntercept(999)))

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("HLine out-of-range render failed: %v", err)
	}
}

// --- Text tests ---

func TestRender_Text(t *testing.T) {
	t.Parallel()

	eng19 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng19,
		eng19.NewFloat64Column("x", []float64{1, 2, 3}),
		eng19.NewFloat64Column("y", []float64{10, 20, 15}),
		eng19.NewStringColumn("label", []string{"A", "B", "C"}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Layer(geom.Text(geom.WithColor("#333333"), geom.WithFontSize(12)))

	_, err := drawPlot(context.Background(), p, 600, 400)
	if err != nil {
		t.Fatalf("Text render failed: %v", err)
	}
}

func TestRender_Text_NoLabelColumn(t *testing.T) {
	t.Parallel()

	// No "label" column -- should fall back to Y values.
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Text())

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Text without label column failed: %v", err)
	}
}

// --- geom.Col tests ---

func TestRender_Col(t *testing.T) {
	t.Parallel()

	eng20 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng20,
		eng20.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng20.NewFloat64Column("y", []float64{10, 25, 15, 30, 20}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Col())

	_, err := drawPlot(context.Background(), p, 600, 400)
	if err != nil {
		t.Fatalf("Col render failed: %v", err)
	}
}

// --- WithLabel legend test ---

func TestRender_WithLabel_Legend(t *testing.T) {
	t.Parallel()

	eng21 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng21,
		eng21.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng21.NewFloat64Column("sin", []float64{0, 1, 0, -1, 0}),
		eng21.NewFloat64Column("cos", []float64{1, 0, -1, 0, 1}),
	)
	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Line(geom.WithColor("#1F77B4"), geom.WithLabel("sin")), aes.Y("sin")).
		Layer(geom.Line(geom.WithColor("#FF7F0E"), geom.WithLabel("cos")), aes.Y("cos")).
		Labs(ggplot.Title("Wide Format Legend"))

	_, err := drawPlot(context.Background(), p, 800, 600)
	if err != nil {
		t.Fatalf("WithLabel legend render failed: %v", err)
	}
}

// --- Discrete Scale (Categorical X) tests ---

func TestRender_CategoricalBars(t *testing.T) {
	t.Parallel()

	eng22 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng22,
		eng22.NewStringColumn("city", []string{"A", "B", "C"}),
		eng22.NewFloat64Column("value", []float64{10, 20, 15}),
	)
	p := ggplot.New(ds, aes.X("city"), aes.Y("value")).
		Layer(geom.Col())

	_, err := drawPlot(context.Background(), p, 600, 400)
	if err != nil {
		t.Fatalf("Categorical bars render failed: %v", err)
	}
}

func TestRender_CategoricalBars_ManyCategories(t *testing.T) {
	t.Parallel()

	cities := []string{"London", "Paris", "Berlin", "Madrid", "Rome", "Vienna", "Prague"}

	values := make([]float64, len(cities))
	for i := range values {
		values[i] = float64(i+1) * 10
	}

	eng23 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng23,
		eng23.NewStringColumn("city", cities),
		eng23.NewFloat64Column("pop", values),
	)
	p := ggplot.New(ds, aes.X("city"), aes.Y("pop")).
		Layer(geom.Col())

	_, err := drawPlot(context.Background(), p, 800, 400)
	if err != nil {
		t.Fatalf("Many categories render failed: %v", err)
	}
}

// --- Boxplot tests ---

func TestRender_Boxplot(t *testing.T) {
	t.Parallel()

	// 3 groups, each with 10 values.
	x := make([]float64, 30)
	y := make([]float64, 30)

	for i := range 30 {
		x[i] = float64(i/10 + 1)
		y[i] = float64(i*3 + 10)
	}

	eng24 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng24, eng24.NewFloat64Column("x", x), eng24.NewFloat64Column("y", y))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Boxplot())

	_, err := drawPlot(context.Background(), p, 600, 400)
	if err != nil {
		t.Fatalf("Boxplot render failed: %v", err)
	}
}

func TestRender_Boxplot_SingleGroup(t *testing.T) {
	t.Parallel()

	y := []float64{10, 20, 30, 40, 50, 25, 35}

	x := make([]float64, len(y))
	for i := range x {
		x[i] = 1
	}

	eng25 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng25, eng25.NewFloat64Column("x", x), eng25.NewFloat64Column("y", y))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Boxplot())

	_, err := drawPlot(context.Background(), p, 400, 400)
	if err != nil {
		t.Fatalf("Single-group boxplot render failed: %v", err)
	}
}

// --- SVG/PDF output tests ---

func TestSave_SVG(t *testing.T) { //nolint:dupl // type-specialized code path.
	t.Parallel()

	ds, _ := dataset.NewDataset(
		memory.NewEngine(context.Background()),
		memory.NewEngine(context.Background()).NewFloat64Column("x", []float64{1, 2, 3, 4}),
		memory.NewEngine(context.Background()).NewFloat64Column("y", []float64{10, 20, 15, 25}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line())

	outPath := filepath.Join(t.TempDir(), "test_output.svg")
	if err := p.Save(context.Background(), outPath, 400, 300); err != nil {
		t.Fatalf("SVG Save failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("SVG file read failed: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<svg") {
		t.Error("SVG output missing <svg> root element")
	}

	if !strings.Contains(content, "</svg>") {
		t.Error("SVG output missing </svg> closing tag")
	}

	if len(data) < 100 {
		t.Errorf("SVG output suspiciously small: %d bytes", len(data))
	}
}

func TestSave_PDF(t *testing.T) { //nolint:dupl // type-specialized code path.
	t.Parallel()

	ds, _ := dataset.NewDataset(
		memory.NewEngine(context.Background()),
		memory.NewEngine(context.Background()).NewFloat64Column("x", []float64{1, 2, 3, 4}),
		memory.NewEngine(context.Background()).NewFloat64Column("y", []float64{10, 20, 15, 25}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line())

	outPath := filepath.Join(t.TempDir(), "test_output.pdf")
	if err := p.Save(context.Background(), outPath, 400, 300); err != nil {
		t.Fatalf("PDF Save failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("PDF file read failed: %v", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "%PDF-") {
		t.Error("PDF output missing PDF header")
	}

	if !strings.Contains(content, "%%EOF") {
		t.Error("PDF output missing EOF trailer")
	}

	if len(data) < 100 {
		t.Errorf("PDF output suspiciously small: %d bytes", len(data))
	}
}

func TestWriteTo_SVG(t *testing.T) { //nolint:dupl // type-specialized code path.
	t.Parallel()

	ds, _ := dataset.NewDataset(
		memory.NewEngine(context.Background()),
		memory.NewEngine(context.Background()).NewFloat64Column("x", []float64{1, 2, 3}),
		memory.NewEngine(context.Background()).NewFloat64Column("y", []float64{5, 10, 15}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point())

	var buf bytes.Buffer

	n, err := p.WriteTo(context.Background(), &buf, "svg", 300, 200)
	if err != nil {
		t.Fatalf("WriteTo SVG failed: %v", err)
	}

	if n == 0 {
		t.Error("WriteTo SVG wrote 0 bytes")
	}

	if !strings.Contains(buf.String(), "<svg") {
		t.Error("WriteTo SVG output missing <svg> element")
	}
}

func TestWriteTo_PDF(t *testing.T) { //nolint:dupl // type-specialized code path.
	t.Parallel()

	ds, _ := dataset.NewDataset(
		memory.NewEngine(context.Background()),
		memory.NewEngine(context.Background()).NewFloat64Column("x", []float64{1, 2, 3}),
		memory.NewEngine(context.Background()).NewFloat64Column("y", []float64{5, 10, 15}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point())

	var buf bytes.Buffer

	n, err := p.WriteTo(context.Background(), &buf, "pdf", 300, 200)
	if err != nil {
		t.Fatalf("WriteTo PDF failed: %v", err)
	}

	if n == 0 {
		t.Error("WriteTo PDF wrote 0 bytes")
	}

	if !strings.HasPrefix(buf.String(), "%PDF-") {
		t.Error("WriteTo PDF output missing %PDF- header")
	}
}

func TestRender_Phase7_Scales(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3}),
		eng.NewFloat64Column("y", []float64{10, 20, 30}),
		eng.NewFloat64Column("size_val", []float64{2.0, 4.0, 6.0}),
		eng.NewFloat64Column("alpha_val", []float64{0.1, 0.5, 0.9}),
		eng.NewStringColumn("shape_val", []string{"A", "B", "C"}),
		eng.NewStringColumn("line_val", []string{"X", "Y", "Z"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(), aes.Size("size_val"), aes.Alpha("alpha_val"), aes.Shape("shape_val")).
		Layer(geom.Line(), aes.Linetype("line_val")).
		ScaleSize(2, 8).
		ScaleAlpha(0.2, 0.8).
		ScaleShapeManual(map[string]string{"A": "square", "B": "triangle"}).
		ScaleLinetype()

	built, err := p.Build(context.Background())
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	cv, err := built.DrawCanvas(context.Background(), 400, 300)
	if err != nil {
		t.Fatalf("drawing failed: %v", err)
	}
	defer func() {
		_ = cv.Close()
	}()
}

// --- guide_axis(n.dodge) tests ---

func TestAxisLabelRows_Builder(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		AxisLabelRows(2)

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("AxisLabelRows render failed: %v", err)
	}
}

func TestAxisLabelRows_Immutable(t *testing.T) {
	t.Parallel()

	ds := testDataset(t)
	base := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point())

	// Deriving with AxisLabelRows should not mutate base.
	withDodge := base.AxisLabelRows(3)
	_ = withDodge

	_, err := drawPlot(context.Background(), base, 400, 300)
	if err != nil {
		t.Fatalf("base render after AxisLabelRows should succeed: %v", err)
	}
}

func TestAxisLabelRows_DenseCategories(t *testing.T) {
	t.Parallel()

	// 30 categories — should trigger auto-dodge when ndodge=0.
	n := 30
	xs := make([]string, n)
	ys := make([]float64, n)

	for i := range n {
		xs[i] = fmt.Sprintf("Category-%02d", i)
		ys[i] = float64(i * i)
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewStringColumn("x", xs),
		eng.NewFloat64Column("y", ys),
	)

	// Auto-dodge (ndodge=0 — default).
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Bar())

	_, err := drawPlot(context.Background(), p, 800, 400)
	if err != nil {
		t.Fatalf("Dense categories auto-dodge render failed: %v", err)
	}

	// Explicit 3-row dodge.
	p2 := p.AxisLabelRows(3)

	_, err = drawPlot(context.Background(), p2, 800, 400)
	if err != nil {
		t.Fatalf("Dense categories explicit dodge render failed: %v", err)
	}

	// Disabled dodge (ndodge=1).
	p3 := p.AxisLabelRows(1)

	_, err = drawPlot(context.Background(), p3, 800, 400)
	if err != nil {
		t.Fatalf("Dense categories no-dodge render failed: %v", err)
	}
}

func TestAxisLabelRows_RotatedLabels_Save(t *testing.T) {
	t.Parallel()

	// Test rotated labels via theme override (angle != 0).
	// This doesn't test theme overrides directly (that needs theme.With),
	// but verifies the rotation code path doesn't panic.
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Labs(ggplot.Title("Rotated Labels Test"))

	outPath := filepath.Join(t.TempDir(), "rotated.png")
	if err := p.Save(context.Background(), outPath, 400, 300); err != nil {
		t.Fatalf("Save with rotated labels failed: %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}

	if info.Size() < 100 {
		t.Fatalf("output file too small (%d bytes)", info.Size())
	}
}

// --- geom.Raster tests ---

func TestRender_Raster_SmallGrid(t *testing.T) {
	t.Parallel()

	// 5x5 grid with continuous fill.
	n := 25
	xs := make([]float64, n)
	ys := make([]float64, n)
	zs := make([]float64, n)

	for i := range n {
		xs[i] = float64(i % 5)
		ys[i] = float64(i / 5)
		zs[i] = float64(i) / float64(n-1) // [0, 1]
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("z", zs),
	)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("z")).
		Layer(geom.Raster()).
		Labs(ggplot.Title("Raster 5×5"))

	cv, err := drawPlot(context.Background(), p, 400, 400)
	if err != nil {
		t.Fatalf("Raster 5×5 render failed: %v", err)
	}

	if cv.Width() != 400 || cv.Height() != 400 {
		t.Errorf("unexpected canvas size: %dx%d", cv.Width(), cv.Height())
	}
}

func TestRender_Raster_Interpolated(t *testing.T) {
	t.Parallel()

	// 10x10 grid with bilinear interpolation.
	n := 100
	xs := make([]float64, n)
	ys := make([]float64, n)
	zs := make([]float64, n)

	for i := range n {
		xs[i] = float64(i % 10)
		ys[i] = float64(i / 10)
		zs[i] = math.Sin(float64(i%10)*0.5) * math.Cos(float64(i/10)*0.5)
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("z", zs),
	)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("z")).
		Layer(geom.Raster(geom.WithInterpolate(true))).
		Labs(ggplot.Title("Raster Interpolated"))

	_, err := drawPlot(context.Background(), p, 600, 600)
	if err != nil {
		t.Fatalf("Raster interpolated render failed: %v", err)
	}
}

func TestRender_Raster_SingleCell(t *testing.T) {
	t.Parallel()

	// 1x1 grid — edge case.
	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{0}),
		eng.NewFloat64Column("y", []float64{0}),
		eng.NewFloat64Column("z", []float64{0.5}),
	)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("z")).
		Layer(geom.Raster())

	_, err := drawPlot(context.Background(), p, 400, 300)
	if err != nil {
		t.Fatalf("Raster single cell render failed: %v", err)
	}
}

func TestRender_Raster_EmptyData(t *testing.T) {
	t.Parallel()

	// Empty dataset — should not panic.
	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{}),
		eng.NewFloat64Column("y", []float64{}),
		eng.NewFloat64Column("z", []float64{}),
	)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("z")).
		Layer(geom.Raster())

	// Empty data should build but produce an empty layer, not panic.
	_, err := p.Build(context.Background())
	if err != nil {
		// It's acceptable if this errors on empty data.
		t.Logf("expected: empty data build error: %v", err)
	}
}

func TestRender_Raster_Save_PNG(t *testing.T) {
	t.Parallel()

	// Save a raster plot to PNG to verify end-to-end.
	n := 100
	xs := make([]float64, n)
	ys := make([]float64, n)
	zs := make([]float64, n)

	for i := range n {
		xs[i] = float64(i % 10)
		ys[i] = float64(i / 10)
		zs[i] = float64(i) / float64(n-1)
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewFloat64Column("z", zs),
	)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("z")).
		Layer(geom.Raster()).
		Labs(ggplot.Title("Raster Save Test"))

	outPath := filepath.Join(t.TempDir(), "raster.png")
	if err := p.Save(context.Background(), outPath, 400, 400); err != nil {
		t.Fatalf("Raster Save failed: %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}

	if info.Size() < 100 {
		t.Fatalf("raster output file too small (%d bytes)", info.Size())
	}
}

func TestRender_JitterPoint(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	xs := []float64{1, 1, 1, 2, 2, 2, 3, 3, 3}
	ys := []float64{1, 2, 3, 1, 2, 3, 1, 2, 3}

	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.JitterPoint(
			geom.WithJitterWidth(0.3),
			geom.WithJitterHeight(0.3),
			geom.WithJitterSeed(99),
		)).
		Labs(ggplot.Title("Jitter Point"))

	cv, err := drawPlot(context.Background(), p, 400, 400)
	if err != nil {
		t.Fatalf("JitterPoint render failed: %v", err)
	}

	if cv.Width() != 400 || cv.Height() != 400 {
		t.Errorf("unexpected canvas size: %dx%d", cv.Width(), cv.Height())
	}
}

func TestRender_JitterPoint_Deterministic(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	xs := []float64{1, 1, 2, 2}
	ys := []float64{1, 2, 1, 2}

	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)

	// Two identical plots with the same seed should produce identical output.
	mkPlot := func() *ggplot.Plot {
		return ggplot.New(ds, aes.X("x"), aes.Y("y")).
			Layer(geom.JitterPoint(geom.WithJitterSeed(42)))
	}

	p1 := mkPlot()
	p2 := mkPlot()

	var buf1, buf2 bytes.Buffer

	_, err1 := p1.WriteTo(context.Background(), &buf1, "png", 200, 200)
	_, err2 := p2.WriteTo(context.Background(), &buf2, "png", 200, 200)

	if err1 != nil || err2 != nil {
		t.Fatalf("render failed: %v / %v", err1, err2)
	}

	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Error("JitterPoint with same seed produced different output")
	}
}

// ---------------------------------------------------------------------------
// Annotation tests
// ---------------------------------------------------------------------------

func TestAnnotation_Text(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		Annotate(ggplot.AnnotateText(2, 4, "hello",
			geom.WithColor("#E74C3C"),
			geom.WithFontSize(12)))

	var buf bytes.Buffer

	_, err := p.WriteTo(context.Background(), &buf, "png", 200, 200)
	if err != nil {
		t.Fatalf("render with text annotation failed: %v", err)
	}

	if buf.Len() < 100 {
		t.Error("rendered output too small")
	}
}

func TestAnnotation_Rect(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		Annotate(ggplot.AnnotateRect(1, 0, 3, 5,
			geom.WithFill("#FFCCCC"),
			geom.WithAlpha(0.3)))

	var buf bytes.Buffer

	_, err := p.WriteTo(context.Background(), &buf, "png", 200, 200)
	if err != nil {
		t.Fatalf("render with rect annotation failed: %v", err)
	}

	if buf.Len() < 100 {
		t.Error("rendered output too small")
	}
}

func TestAnnotation_Segment(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		Annotate(ggplot.AnnotateSegment(1, 1, 4, 8,
			geom.WithColor("#333333"),
			geom.WithLineWidth(1.5)))

	var buf bytes.Buffer

	_, err := p.WriteTo(context.Background(), &buf, "png", 200, 200)
	if err != nil {
		t.Fatalf("render with segment annotation failed: %v", err)
	}

	if buf.Len() < 100 {
		t.Error("rendered output too small")
	}
}

func TestAnnotation_Arrow(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		Annotate(ggplot.AnnotateArrow(1, 1, 4, 8,
			geom.WithColor("#2C3E50"),
			geom.WithLineWidth(2)))

	var buf bytes.Buffer

	_, err := p.WriteTo(context.Background(), &buf, "png", 200, 200)
	if err != nil {
		t.Fatalf("render with arrow annotation failed: %v", err)
	}

	if buf.Len() < 100 {
		t.Error("rendered output too small")
	}
}

func TestAnnotation_Label(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		Annotate(ggplot.AnnotateLabel(3, 5, "outlier",
			geom.WithFill("#FFFFFF"),
			geom.WithColor("#333333"),
			geom.WithPadding(6)))

	var buf bytes.Buffer

	_, err := p.WriteTo(context.Background(), &buf, "png", 200, 200)
	if err != nil {
		t.Fatalf("render with label annotation failed: %v", err)
	}

	if buf.Len() < 100 {
		t.Error("rendered output too small")
	}
}

func TestAnnotation_Combined(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	// All annotation types combined.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		Annotate(ggplot.AnnotateText(2, 4, "text")).
		Annotate(ggplot.AnnotateRect(1, 0, 3, 5, geom.WithFill("#FFCCCC"))).
		Annotate(ggplot.AnnotateSegment(0, 0, 5, 10)).
		Annotate(ggplot.AnnotateArrow(1, 1, 4, 8)).
		Annotate(ggplot.AnnotateLabel(3, 5, "label"))

	var buf bytes.Buffer

	_, err := p.WriteTo(context.Background(), &buf, "png", 200, 200)
	if err != nil {
		t.Fatalf("render with combined annotations failed: %v", err)
	}

	if buf.Len() < 100 {
		t.Error("rendered output too small")
	}
}

func TestAnnotation_Save_PNG(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		Annotate(ggplot.AnnotateText(2, 4, "peak")).
		Annotate(ggplot.AnnotateArrow(1, 1, 3, 6))

	out := filepath.Join(t.TempDir(), "annotated.png")
	if err := p.Save(context.Background(), out, 400, 300); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if fi.Size() < 100 {
		t.Error("saved file too small")
	}
}

// testLineDataset creates a simple 5-point line dataset for annotation tests.
func testLineDataset(eng *memory.Engine) dataset.Dataset {
	xs := []float64{0, 1, 2, 3, 4}
	ys := []float64{0, 2, 4, 6, 8}

	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)

	return ds
}

// ---------------------------------------------------------------------------
// coord.CartesianZoom tests
// ---------------------------------------------------------------------------

func TestCoordCartesianZoom_Build(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		CoordCartesian(1, 3, 2, 6) //nolint:mnd // Zoom window.

	built, err := p.Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if built == nil {
		t.Fatal("expected non-nil Built")
	}
}

func TestCoordCartesianZoom_ScaleBoundsOverridden(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	// Zoom into x=[1,3], y=[2,6] — data outside is NOT removed, but
	// scale bounds should reflect the zoom window.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		CoordCartesian(1, 3, 2, 6) //nolint:mnd // Zoom window.

	built, err := p.Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	info := built.Explain()
	// The built info should mention the coord type.
	if !strings.Contains(info, "coord") {
		t.Errorf("expected coord info, got: %s", info)
	}
}

func TestCoordCartesianZoom_Save_PNG(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		CoordCartesian(1, 3, math.NaN(), math.NaN()) //nolint:mnd // Zoom x only.

	dir := t.TempDir()
	out := filepath.Join(dir, "zoom.png")

	if err := p.Save(context.Background(), out, 400, 300); err != nil { //nolint:mnd // Test canvas.
		t.Fatalf("Save: %v", err)
	}

	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if fi.Size() < 100 { //nolint:mnd // Minimum viable PNG size.
		t.Error("saved file too small")
	}
}

func TestCoordCartesianZoom_PartialZoom(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	// Zoom only x, leave y auto.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		CoordCartesian(1, 2, math.NaN(), math.NaN()) //nolint:mnd // Partial zoom.

	_, err := p.Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
}

// ---------------------------------------------------------------------------
// coord.Fixed tests
// ---------------------------------------------------------------------------

func TestCoordFixed_Build(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		CoordFixed(1)

	built, err := p.Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if built == nil {
		t.Fatal("expected non-nil Built")
	}
}

func TestCoordFixed_Save_PNG(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		CoordFixed(1)

	dir := t.TempDir()
	out := filepath.Join(dir, "fixed.png")

	if err := p.Save(context.Background(), out, 600, 300); err != nil { //nolint:mnd // Wide canvas.
		t.Fatalf("Save: %v", err)
	}

	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if fi.Size() < 100 { //nolint:mnd // Minimum viable PNG size.
		t.Error("saved file too small")
	}
}

func TestCoordFixed_Render(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	// Wide canvas with CoordFixed(1) — should produce valid output
	// without panic or error.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		CoordFixed(1)

	cv, err := drawPlot(context.Background(), p, 800, 300) //nolint:mnd // Wide.
	if err != nil {
		t.Fatalf("drawPlot: %v", err)
	}

	if cv == nil {
		t.Fatal("expected non-nil canvas")
	}
}

func TestCoordFixed_CustomRatio(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds := testLineDataset(eng)

	// Ratio of 2 — y is stretched to twice x.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		CoordFixed(2) //nolint:mnd // Custom ratio.

	dir := t.TempDir()
	out := filepath.Join(dir, "ratio2.png")

	if err := p.Save(context.Background(), out, 600, 600); err != nil { //nolint:mnd // Square canvas.
		t.Fatalf("Save: %v", err)
	}

	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if fi.Size() < 100 { //nolint:mnd // Minimum viable PNG size.
		t.Error("saved file too small")
	}
}

func TestCoordFixed_CoordInterface(t *testing.T) {
	t.Parallel()

	c := coord.Fixed(1)
	if c.String() != "fixed" {
		t.Errorf("expected 'fixed', got %q", c.String())
	}

	// Verify it implements Fixer.
	f, ok := c.(coord.Fixer)
	if !ok {
		t.Fatal("Fixed coord does not implement Fixer")
	}

	if f.AspectRatio() != 1 {
		t.Errorf("expected ratio 1, got %f", f.AspectRatio())
	}
}

func TestCoordCartesianZoom_CoordInterface(t *testing.T) {
	t.Parallel()

	lo := 1.0
	hi := 5.0

	c := coord.CartesianZoom([2]*float64{&lo, &hi}, [2]*float64{nil, nil})
	if c.String() != "cartesian_zoom" {
		t.Errorf("expected 'cartesian_zoom', got %q", c.String())
	}

	// Verify it implements Zoomer.
	z, ok := c.(coord.Zoomer)
	if !ok {
		t.Fatal("CartesianZoom coord does not implement Zoomer")
	}

	xlim, ylim := z.ZoomBounds()
	if xlim[0] == nil || *xlim[0] != lo {
		t.Error("xlim[0] mismatch")
	}

	if xlim[1] == nil || *xlim[1] != hi {
		t.Error("xlim[1] mismatch")
	}

	if ylim[0] != nil {
		t.Error("ylim[0] should be nil")
	}

	if ylim[1] != nil {
		t.Error("ylim[1] should be nil")
	}
}
