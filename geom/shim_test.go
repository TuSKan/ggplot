package geom_test

// Backward-compatibility shim verification: sugar constructors (Histogram,
// Bar, Smooth, Density, Area, Boxplot) must produce Layers structurally
// equivalent to their pipeline counterparts (RectY, LineY, AreaY, etc.).

import (
	"testing"

	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/stat"
)

// layerSpec captures the structural properties of a Layer that must
// match between sugar and pipeline constructors.
type layerSpec struct {
	geomType      geom.Type
	pipelineNames []string // stat.Transform.Name() for each stage
	position      string   // Pos.String()
	alpha         float64
	width         float64
	lineWidth     float64
	inset         float64
	orientation   geom.Orientation
}

func extractSpec(l geom.Layer) layerSpec {
	names := make([]string, len(l.Pipeline))
	for i, t := range l.Pipeline {
		names[i] = t.Name()
	}

	return layerSpec{
		geomType:      l.Geom,
		pipelineNames: names,
		position:      l.Position.String(),
		alpha:         l.Params.Alpha,
		width:         l.Params.Width,
		lineWidth:     l.Params.LineWidth,
		inset:         l.Params.Inset,
		orientation:   l.Params.Orientation,
	}
}

func assertSpecEqual(t *testing.T, name string, sugar, pipeline layerSpec) {
	t.Helper()

	if sugar.geomType != pipeline.geomType {
		t.Errorf("%s: Geom type mismatch: sugar=%q pipeline=%q", name, sugar.geomType, pipeline.geomType)
	}

	if len(sugar.pipelineNames) != len(pipeline.pipelineNames) {
		t.Errorf("%s: pipeline length mismatch: sugar=%v pipeline=%v", name, sugar.pipelineNames, pipeline.pipelineNames)
	} else {
		for i := range sugar.pipelineNames {
			if sugar.pipelineNames[i] != pipeline.pipelineNames[i] {
				t.Errorf("%s: pipeline[%d] mismatch: sugar=%q pipeline=%q", name, i, sugar.pipelineNames[i], pipeline.pipelineNames[i])
			}
		}
	}

	if sugar.position != pipeline.position {
		t.Errorf("%s: position mismatch: sugar=%q pipeline=%q", name, sugar.position, pipeline.position)
	}
}

// --- Sugar vs Pipeline structural equivalence ---

func TestShim_Histogram_vs_RectY_BinX(t *testing.T) {
	t.Parallel()

	sugar := geom.Histogram()
	pipeline := geom.RectY([]stat.Transform{stat.BinX(stat.WithBins(30))})

	sugarSpec := extractSpec(sugar)
	pipelineSpec := extractSpec(pipeline)

	assertSpecEqual(t, "Histogram≡RectY(BinX)", sugarSpec, pipelineSpec)

	// Both must use TypeRect.
	if sugar.Geom != geom.TypeRect {
		t.Errorf("Histogram should use TypeRect, got %q", sugar.Geom)
	}

	// Both must have Inset > 0 for continuous bin separation.
	if sugarSpec.inset == 0 {
		t.Error("Histogram should have non-zero Inset for bin separation")
	}

	if pipelineSpec.inset == 0 {
		t.Error("RectY(BinX) should have non-zero Inset for bin separation")
	}
}

func TestShim_Bar_Pipeline(t *testing.T) {
	t.Parallel()

	sugar := geom.Bar()

	// Bar uses TypeBar (discrete) with Count transform + Stack position.
	if sugar.Geom != geom.TypeBar {
		t.Errorf("Bar should use TypeBar, got %q", sugar.Geom)
	}

	spec := extractSpec(sugar)
	if len(spec.pipelineNames) != 1 || spec.pipelineNames[0] != "count" {
		t.Errorf("Bar pipeline should be [count], got %v", spec.pipelineNames)
	}

	if spec.position != "stack" {
		t.Errorf("Bar position should be stack, got %q", spec.position)
	}

	// Bar should NOT have Inset (discrete bars have gaps via Width < 1).
	if spec.inset != 0 {
		t.Errorf("Bar should have zero Inset (discrete), got %f", spec.inset)
	}
}

func TestShim_Col_Pipeline(t *testing.T) {
	t.Parallel()

	sugar := geom.Col()

	// Col uses TypeBar (discrete) with no pipeline (identity stat) + Stack position.
	if sugar.Geom != geom.TypeBar {
		t.Errorf("Col should use TypeBar, got %q", sugar.Geom)
	}

	spec := extractSpec(sugar)
	if len(spec.pipelineNames) != 0 {
		t.Errorf("Col pipeline should be empty (identity), got %v", spec.pipelineNames)
	}

	if spec.position != "stack" {
		t.Errorf("Col position should be stack, got %q", spec.position)
	}
}

func TestShim_Smooth_vs_LineY_SmoothXY(t *testing.T) {
	t.Parallel()

	sugar := geom.Smooth()
	pipeline := geom.LineY(
		[]stat.Transform{stat.SmoothXY(stat.WithMethod("lm"), stat.WithSmoothPoints(80))},
	)

	sugarSpec := extractSpec(sugar)
	pipelineSpec := extractSpec(pipeline)

	// Both should have smoothxy transform.
	if len(sugarSpec.pipelineNames) != 1 || sugarSpec.pipelineNames[0] != "smoothXY" {
		t.Errorf("Smooth pipeline should be [smoothxy], got %v", sugarSpec.pipelineNames)
	}

	if len(pipelineSpec.pipelineNames) != 1 || pipelineSpec.pipelineNames[0] != "smoothXY" {
		t.Errorf("LineY(SmoothXY) pipeline should be [smoothxy], got %v", pipelineSpec.pipelineNames)
	}

	// Both should have identity position.
	if sugarSpec.position != "identity" {
		t.Errorf("Smooth position should be identity, got %q", sugarSpec.position)
	}

	if pipelineSpec.position != "identity" {
		t.Errorf("LineY position should be identity, got %q", pipelineSpec.position)
	}
}

func TestShim_Density_vs_AreaY_DensityX(t *testing.T) {
	t.Parallel()

	sugar := geom.Density()
	pipeline := geom.AreaY([]stat.Transform{stat.DensityX()})

	sugarSpec := extractSpec(sugar)
	pipelineSpec := extractSpec(pipeline)

	// Both should have densityx transform.
	if len(sugarSpec.pipelineNames) != 1 || sugarSpec.pipelineNames[0] != "densityx" {
		t.Errorf("Density pipeline should be [densityx], got %v", sugarSpec.pipelineNames)
	}

	if len(pipelineSpec.pipelineNames) != 1 || pipelineSpec.pipelineNames[0] != "densityx" {
		t.Errorf("AreaY(DensityX) pipeline should be [densityx], got %v", pipelineSpec.pipelineNames)
	}

	// Both should have identity position.
	if sugarSpec.position != pipelineSpec.position {
		t.Errorf("Density vs AreaY(DensityX) position mismatch: %q vs %q", sugarSpec.position, pipelineSpec.position)
	}
}

func TestShim_RectX_Horizontal(t *testing.T) {
	t.Parallel()

	layer := geom.RectX([]stat.Transform{stat.BinY(stat.WithBins(20))})

	if layer.Geom != geom.TypeRect {
		t.Errorf("RectX should use TypeRect, got %q", layer.Geom)
	}

	spec := extractSpec(layer)
	if spec.orientation != geom.Horizontal {
		t.Errorf("RectX should be Horizontal, got %q", spec.orientation)
	}

	if spec.position != "stack" {
		t.Errorf("RectX position should be stack, got %q", spec.position)
	}

	if spec.inset == 0 {
		t.Error("RectX should have non-zero Inset")
	}
}

func TestShim_Boxplot_Pipeline(t *testing.T) {
	t.Parallel()

	sugar := geom.Boxplot()

	if sugar.Geom != geom.TypeBoxPlot {
		t.Errorf("Boxplot should use TypeBoxPlot, got %q", sugar.Geom)
	}

	spec := extractSpec(sugar)
	if len(spec.pipelineNames) != 1 || spec.pipelineNames[0] != "boxplotY" {
		t.Errorf("Boxplot pipeline should be [boxplotY], got %v", spec.pipelineNames)
	}

	if spec.position != "identity" {
		t.Errorf("Boxplot position should be identity, got %q", spec.position)
	}
}

// --- Sugar option forwarding ---

func TestShim_Histogram_WithBins_Forwarded(t *testing.T) {
	t.Parallel()

	// WithBins should rebuild the pipeline via finalizePipeline.
	layer := geom.Histogram(geom.WithBins(50))

	if layer.Geom != geom.TypeRect {
		t.Errorf("Histogram(WithBins) should use TypeRect, got %q", layer.Geom)
	}

	spec := extractSpec(layer)
	if len(spec.pipelineNames) != 1 || spec.pipelineNames[0] != "binx" {
		t.Errorf("Histogram(WithBins(50)) pipeline should be [binx], got %v", spec.pipelineNames)
	}
}

func TestShim_Histogram_StatOverride(t *testing.T) {
	t.Parallel()

	// Explicit Stat() should override default pipeline.
	layer := geom.Histogram(geom.Stat(stat.Count()))

	spec := extractSpec(layer)
	if len(spec.pipelineNames) != 1 || spec.pipelineNames[0] != "count" {
		t.Errorf("Histogram(Stat(Count)) pipeline should be [count], got %v", spec.pipelineNames)
	}
}

func TestShim_Smooth_WithMethod_Forwarded(t *testing.T) {
	t.Parallel()

	layer := geom.Smooth(geom.WithMethod("loess"))

	spec := extractSpec(layer)
	if len(spec.pipelineNames) != 1 || spec.pipelineNames[0] != "smoothXY" {
		t.Errorf("Smooth(WithMethod) pipeline should be [smoothxy], got %v", spec.pipelineNames)
	}
}

// --- TypeRect vs TypeBar distinction ---

func TestShim_TypeRect_vs_TypeBar_Distinction(t *testing.T) {
	t.Parallel()

	// TypeRect is for continuous marks (histogram, pipeline constructors).
	// TypeBar is for discrete marks (bar chart, col chart).
	rect := geom.RectY([]stat.Transform{stat.BinX()})
	bar := geom.Bar()
	col := geom.Col()

	if rect.Geom != geom.TypeRect {
		t.Errorf("RectY should use TypeRect, got %q", rect.Geom)
	}

	if bar.Geom != geom.TypeBar {
		t.Errorf("Bar should use TypeBar, got %q", bar.Geom)
	}

	if col.Geom != geom.TypeBar {
		t.Errorf("Col should use TypeBar, got %q", col.Geom)
	}

	// Histogram now uses TypeRect (continuous bins).
	hist := geom.Histogram()
	if hist.Geom != geom.TypeRect {
		t.Errorf("Histogram should use TypeRect, got %q", hist.Geom)
	}
}
