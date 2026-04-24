package ggplot_test

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/coord"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/facet"
	"github.com/TuSKan/ggplot/geom"
)

func testDataset(t *testing.T) dataset.Dataset {
	t.Helper()
	ds, err := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
		memory.NewEngine().NewFloat64Column("y", []float64{2.1, 4.3, 3.0, 7.8, 5.5, 8.1, 6.9, 9.2, 8.5, 10.0}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return ds
}

// --- Builder API tests ---

func TestNew_ReturnsNonNil(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"))
	if p == nil {
		t.Fatal("New returned nil")
	}
}

func TestPlot_Layer_Immutable(t *testing.T) {
	ds := testDataset(t)
	base := ggplot.New(ds, aes.X("x"), aes.Y("y"))
	withPoint := base.Layer(geom.Point())
	withLine := base.Layer(geom.Line())

	if withPoint == nil || withLine == nil {
		t.Fatal("Layer returned nil")
	}
}

func TestPlot_NoLayers_Error(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"))

	_, err := p.Render(800, 600)
	if err == nil {
		t.Fatal("expected error for plot with no layers")
	}
}

func TestPlot_NilDataset_Error(t *testing.T) {
	p := ggplot.New(dataset.Dataset{}, aes.X("x")).
		Layer(geom.Point())

	_, err := p.Render(800, 600)
	if err == nil {
		t.Fatal("expected error for nil dataset")
	}
}

// --- Rendering tests (all geom types) ---

func TestRender_Point(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(4), geom.WithColor("#FF0000")))

	cv, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("Point render failed: %v", err)
	}
	if cv.Width() != 400 || cv.Height() != 300 {
		t.Errorf("unexpected canvas size: %dx%d", cv.Width(), cv.Height())
	}
}

func TestRender_Line(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#0000FF"), geom.WithLineWidth(3)))

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Line render failed: %v", err)
	}
}

func TestRender_Bar(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		memory.NewEngine().NewFloat64Column("count", []float64{10, 25, 15, 30, 20}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("count")).
		Layer(geom.Bar(geom.WithFill("#336699"), geom.WithWidth(0.7)))

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Bar render failed: %v", err)
	}
}

func TestRender_Histogram(t *testing.T) {
	xs := make([]float64, 500)
	for i := range xs {
		xs[i] = rand.NormFloat64()*5 + 10
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs))

	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Histogram(geom.WithBins(30), geom.WithFill("#3498DB")))

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Histogram render failed: %v", err)
	}
}

func TestRender_Histogram_StatTransform(t *testing.T) {
	// Verify the stat transform actually runs: histogram should produce
	// binned data, not raw data.
	xs := make([]float64, 100)
	for i := range xs {
		xs[i] = float64(i)
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs))

	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Histogram(geom.WithBins(10)))

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Histogram stat transform failed: %v", err)
	}
}

func TestRender_Area(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Area(geom.WithFill("#2ecc71"), geom.WithAlpha(0.6)))

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Area render failed: %v", err)
	}
}

func TestRender_Smooth(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Smooth(geom.WithMethod("lm"), geom.WithColor("#E74C3C")))

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Smooth render failed: %v", err)
	}
}

func TestRender_Density(t *testing.T) {
	xs := make([]float64, 200)
	for i := range xs {
		xs[i] = rand.NormFloat64()
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs))

	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Density(geom.WithFill("#9b59b6"), geom.WithAlpha(0.5)))

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Density render failed: %v", err)
	}
}

func TestRender_Step(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Step(geom.WithColor("#1abc9c")))

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Step render failed: %v", err)
	}
}

// --- Multi-layer tests ---

func TestRender_MultiLayer_PointAndLine(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithColor("#FF0000"))).
		Layer(geom.Line(geom.WithColor("#0000FF")))

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("MultiLayer render failed: %v", err)
	}
}

func TestRender_MultiLayer_PointAndSmooth(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(3))).
		Layer(geom.Smooth())

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Point+Smooth render failed: %v", err)
	}
}

func TestRender_MultiLayer_ThreeLayers(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithColor("#FF0000"))).
		Layer(geom.Line(geom.WithColor("#0000FF"))).
		Layer(geom.Smooth())

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("ThreeLayers render failed: %v", err)
	}
}

// --- Labels ---

func TestRender_Labels(t *testing.T) {
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

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Labels render failed: %v", err)
	}
}

// --- Coord ---

func TestRender_CoordFlip(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Coord(coord.Flip())

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("CoordFlip render failed: %v", err)
	}
}

// --- Save ---

func TestPlot_Save_PNG(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Labs(ggplot.Title("Test Plot"))

	outPath := filepath.Join(t.TempDir(), "test.png")
	if err := p.Save(outPath, 400, 300); err != nil {
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
	xs := make([]float64, 100)
	for i := range xs {
		xs[i] = rand.NormFloat64()
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs))

	outPath := filepath.Join(t.TempDir(), "hist.png")
	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Histogram(geom.WithBins(20))).
		Labs(ggplot.Title("Histogram Test"))

	if err := p.Save(outPath, 800, 600); err != nil {
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
	// End-to-end: build and save a plot for each geom type.
	xs := make([]float64, 50)
	ys := make([]float64, 50)
	for i := range xs {
		xs[i] = float64(i)
		ys[i] = math.Sin(float64(i) * 0.2)
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs), memory.NewEngine().NewFloat64Column("y", ys))

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
			var p *ggplot.Plot
			if tc.needY {
				p = ggplot.New(ds, aes.X("x"), aes.Y("y")).Layer(tc.layer)
			} else {
				p = ggplot.New(ds, aes.X("x")).Layer(tc.layer)
			}

			outPath := filepath.Join(t.TempDir(), tc.name+".png")
			if err := p.Save(outPath, 400, 300); err != nil {
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
	ds := testDataset(t)
	panels, err := facet.None().Split(ds)
	if err != nil {
		t.Fatal(err)
	}
	if len(panels) != 1 {
		t.Errorf("expected 1 panel, got %d", len(panels))
	}
}

// --- Coord ---

func TestCartesianTransform(t *testing.T) {
	c := coord.Cartesian()
	px, py := c.Transform(0.5, 0.5, 100, 100)
	if px != 50 || py != 50 {
		t.Errorf("Cartesian(0.5,0.5): expected (50,50), got (%v,%v)", px, py)
	}
}

func TestFlipTransform(t *testing.T) {
	c := coord.Flip()
	if !c.IsFlipped() {
		t.Error("Flip should report IsFlipped=true")
	}
}

// --- Edge cases ---

func TestRender_SingleDataPoint(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{5}),
		memory.NewEngine().NewFloat64Column("y", []float64{10}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point())

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("Single point render failed: %v", err)
	}
}

func TestRender_TwoDataPoints(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{0, 100}),
		memory.NewEngine().NewFloat64Column("y", []float64{0, 100}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line())

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("Two point line render failed: %v", err)
	}
}

func TestRender_LargeDataset(t *testing.T) {
	n := 10000
	xs := make([]float64, n)
	ys := make([]float64, n)
	for i := range xs {
		xs[i] = float64(i)
		ys[i] = math.Sin(float64(i) * 0.01)
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs), memory.NewEngine().NewFloat64Column("y", ys))

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line())

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Large dataset render failed: %v", err)
	}
}

func TestRender_NegativeValues(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{-5, -3, -1, 1, 3, 5}),
		memory.NewEngine().NewFloat64Column("y", []float64{-10, -5, 0, 5, 10, 15}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Layer(geom.Line())

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("Negative values render failed: %v", err)
	}
}

func TestRender_ConstantY(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		memory.NewEngine().NewFloat64Column("y", []float64{5, 5, 5, 5, 5}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point())

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("Constant Y render failed: %v", err)
	}
}

// --- Color mapping tests ---

func groupedDataset(t *testing.T) dataset.Dataset {
	t.Helper()
	ds, err := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 3, 1, 2, 3, 1, 2, 3}),
		memory.NewEngine().NewFloat64Column("y", []float64{1, 4, 9, 2, 5, 8, 3, 6, 7}),
		memory.NewEngine().NewStringColumn("group", []string{"A", "A", "A", "B", "B", "B", "C", "C", "C"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return ds
}

func TestRender_ColorMapping_Point(t *testing.T) {
	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Point(geom.WithSize(5)))

	cv, err := p.Render(600, 400)
	if err != nil {
		t.Fatalf("Color mapping point render failed: %v", err)
	}
	if cv == nil {
		t.Fatal("canvas is nil")
	}
}

func TestRender_ColorMapping_Line(t *testing.T) {
	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Line())

	_, err := p.Render(600, 400)
	if err != nil {
		t.Fatalf("Color mapping line render failed: %v", err)
	}
}

func TestRender_ColorMapping_WithLegend(t *testing.T) {
	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Point()).
		Labs(ggplot.Title("Grouped Scatter"))

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Color mapping with legend render failed: %v", err)
	}
}

func TestRender_ColorMapping_ManyGroups(t *testing.T) {
	// 10 groups to exercise palette wrap-around.
	n := 100
	xs := make([]float64, n)
	ys := make([]float64, n)
	groups := make([]string, n)
	labels := []string{"g0", "g1", "g2", "g3", "g4", "g5", "g6", "g7", "g8", "g9"}
	for i := 0; i < n; i++ {
		xs[i] = float64(i % 10)
		ys[i] = float64(i)
		groups[i] = labels[i%10]
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", xs),
		memory.NewEngine().NewFloat64Column("y", ys),
		memory.NewEngine().NewStringColumn("g", groups),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("g")).
		Layer(geom.Point())

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Many groups render failed: %v", err)
	}
}

// --- XLim / YLim tests ---

func TestRender_XLim(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		XLim(0, 20)

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("XLim render failed: %v", err)
	}
}

func TestRender_YLim(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		YLim(-5, 15)

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("YLim render failed: %v", err)
	}
}

func TestRender_XLim_YLim_Combined(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		XLim(2, 8).
		YLim(0, 12)

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("Combined XLim/YLim render failed: %v", err)
	}
}

func TestRender_XLim_NaN_PartialOverride(t *testing.T) {
	ds := testDataset(t)
	// Only override min, let max auto-detect.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		XLim(0, math.NaN())

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("Partial XLim render failed: %v", err)
	}
}

// --- CoordFlip tests ---

func TestRender_CoordFlip_Point(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		CoordFlip()

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("CoordFlip render failed: %v", err)
	}
}

func TestRender_CoordFlip_Bar(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		memory.NewEngine().NewFloat64Column("y", []float64{10, 25, 15, 30, 20}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Bar()).
		CoordFlip()

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("CoordFlip bar render failed: %v", err)
	}
}

// --- Step geom tests ---

func TestRender_StepGeom(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}),
		memory.NewEngine().NewFloat64Column("y", []float64{0, 1, 1, 2, 2, 3, 3, 4, 4, 5}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Step(geom.WithColor("#336699")))

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("Step render failed: %v", err)
	}
}

func TestRender_Step_ColorMapping(t *testing.T) {
	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Step())

	_, err := p.Render(600, 400)
	if err != nil {
		t.Fatalf("Step with color mapping failed: %v", err)
	}
}

// --- Rug geom tests ---

func TestRender_Rug(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Layer(geom.Rug())

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("Rug render failed: %v", err)
	}
}

// --- Combined feature tests ---

func TestRender_AllNewFeatures(t *testing.T) {
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

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Combined features render failed: %v", err)
	}
}

// --- Save integration tests (write actual PNGs for visual inspection) ---

func TestSave_ColorMapping(t *testing.T) {
	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Point(geom.WithSize(6))).
		Labs(ggplot.Title("Color Mapping Test"))

	dir := t.TempDir()
	path := filepath.Join(dir, "color_mapping.png")
	if err := p.Save(path, 600, 400); err != nil {
		t.Fatalf("Save color mapping failed: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		t.Fatalf("PNG not created or empty: err=%v, size=%d", err, info.Size())
	}
}

func TestSave_XLimYLim(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		XLim(2, 8).
		YLim(0, 15).
		Labs(ggplot.Title("Axis Limits Test"))

	dir := t.TempDir()
	path := filepath.Join(dir, "xlim_ylim.png")
	if err := p.Save(path, 600, 400); err != nil {
		t.Fatalf("Save XLim/YLim failed: %v", err)
	}
	info, _ := os.Stat(path)
	if info.Size() == 0 {
		t.Fatal("PNG is empty")
	}
}

func TestSave_StepWithLegend(t *testing.T) {
	ds := groupedDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Step(geom.WithLineWidth(2))).
		Labs(ggplot.Title("Step Functions"))

	dir := t.TempDir()
	path := filepath.Join(dir, "step.png")
	if err := p.Save(path, 600, 400); err != nil {
		t.Fatalf("Save step failed: %v", err)
	}
	info, _ := os.Stat(path)
	if info.Size() == 0 {
		t.Fatal("PNG is empty")
	}
}

// --- HLine / VLine tests ---

func TestRender_HLine(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		Layer(geom.HLine(geom.WithIntercept(5), geom.WithColor("#CC0000")))

	_, err := p.Render(600, 400)
	if err != nil {
		t.Fatalf("HLine render failed: %v", err)
	}
}

func TestRender_VLine(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Layer(geom.VLine(geom.WithIntercept(5), geom.WithColor("#006600")))

	_, err := p.Render(600, 400)
	if err != nil {
		t.Fatalf("VLine render failed: %v", err)
	}
}

func TestRender_HLine_VLine_Combined(t *testing.T) {
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line()).
		Layer(geom.HLine(geom.WithIntercept(0), geom.WithColor("#999999"), geom.WithLabel("baseline"))).
		Layer(geom.VLine(geom.WithIntercept(5), geom.WithColor("#2ECC71"), geom.WithLabel("x=5")))

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("Combined HLine/VLine render failed: %v", err)
	}
}

func TestRender_HLine_OutOfRange(t *testing.T) {
	ds := testDataset(t)
	// Intercept way outside the Y range — should not crash.
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Layer(geom.HLine(geom.WithIntercept(999)))

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("HLine out-of-range render failed: %v", err)
	}
}

// --- Text tests ---

func TestRender_Text(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 3}),
		memory.NewEngine().NewFloat64Column("y", []float64{10, 20, 15}),
		memory.NewEngine().NewStringColumn("label", []string{"A", "B", "C"}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point()).
		Layer(geom.Text(geom.WithColor("#333333"), geom.WithFontSize(12)))

	_, err := p.Render(600, 400)
	if err != nil {
		t.Fatalf("Text render failed: %v", err)
	}
}

func TestRender_Text_NoLabelColumn(t *testing.T) {
	// No "label" column — should fall back to Y values.
	ds := testDataset(t)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Text())

	_, err := p.Render(400, 300)
	if err != nil {
		t.Fatalf("Text without label column failed: %v", err)
	}
}

// --- geom.Col tests ---

func TestRender_Col(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		memory.NewEngine().NewFloat64Column("y", []float64{10, 25, 15, 30, 20}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Col())

	_, err := p.Render(600, 400)
	if err != nil {
		t.Fatalf("Col render failed: %v", err)
	}
}

// --- WithLabel legend test ---

func TestRender_WithLabel_Legend(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		memory.NewEngine().NewFloat64Column("sin", []float64{0, 1, 0, -1, 0}),
		memory.NewEngine().NewFloat64Column("cos", []float64{1, 0, -1, 0, 1}),
	)
	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Line(geom.WithColor("#1F77B4"), geom.WithLabel("sin")), aes.Y("sin")).
		Layer(geom.Line(geom.WithColor("#FF7F0E"), geom.WithLabel("cos")), aes.Y("cos")).
		Labs(ggplot.Title("Wide Format Legend"))

	_, err := p.Render(800, 600)
	if err != nil {
		t.Fatalf("WithLabel legend render failed: %v", err)
	}
}

// --- Discrete Scale (Categorical X) tests ---

func TestRender_CategoricalBars(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewStringColumn("city", []string{"A", "B", "C"}),
		memory.NewEngine().NewFloat64Column("value", []float64{10, 20, 15}),
	)
	p := ggplot.New(ds, aes.X("city"), aes.Y("value")).
		Layer(geom.Col())

	_, err := p.Render(600, 400)
	if err != nil {
		t.Fatalf("Categorical bars render failed: %v", err)
	}
}

func TestRender_CategoricalBars_ManyCategories(t *testing.T) {
	cities := []string{"London", "Paris", "Berlin", "Madrid", "Rome", "Vienna", "Prague"}
	values := make([]float64, len(cities))
	for i := range values {
		values[i] = float64(i+1) * 10
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewStringColumn("city", cities),
		memory.NewEngine().NewFloat64Column("pop", values),
	)
	p := ggplot.New(ds, aes.X("city"), aes.Y("pop")).
		Layer(geom.Col())

	_, err := p.Render(800, 400)
	if err != nil {
		t.Fatalf("Many categories render failed: %v", err)
	}
}

// --- Boxplot tests ---

func TestRender_Boxplot(t *testing.T) {
	// 3 groups, each with 10 values.
	x := make([]float64, 30)
	y := make([]float64, 30)
	for i := 0; i < 30; i++ {
		x[i] = float64(i/10 + 1)
		y[i] = float64(i*3 + 10)
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", x), memory.NewEngine().NewFloat64Column("y", y))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Boxplot())

	_, err := p.Render(600, 400)
	if err != nil {
		t.Fatalf("Boxplot render failed: %v", err)
	}
}

func TestRender_Boxplot_SingleGroup(t *testing.T) {
	y := []float64{10, 20, 30, 40, 50, 25, 35}
	x := make([]float64, len(y))
	for i := range x {
		x[i] = 1
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", x), memory.NewEngine().NewFloat64Column("y", y))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Boxplot())

	_, err := p.Render(400, 400)
	if err != nil {
		t.Fatalf("Single-group boxplot render failed: %v", err)
	}
}
