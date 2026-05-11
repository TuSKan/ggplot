package geom_test

import (
	"strings"
	"testing"

	"github.com/TuSKan/ggplot/geom"
)

func TestValidate_PointWithBins_Warning(t *testing.T) {
	t.Parallel()

	layer := geom.Point(geom.WithBins(30))

	warnings := layer.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected warning for WithBins on Point")
	}

	found := false

	for _, w := range warnings {
		if strings.Contains(w, "WithBins") && strings.Contains(w, "geom_point") {
			found = true
		}
	}

	if !found {
		t.Errorf("expected WithBins warning, got: %v", warnings)
	}
}

func TestValidate_PointWithMethod_Warning(t *testing.T) {
	t.Parallel()

	layer := geom.Point(geom.WithMethod("lm"))

	warnings := layer.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected warning for WithMethod on Point")
	}
}

func TestValidate_LineWithBins_Warning(t *testing.T) {
	t.Parallel()

	layer := geom.Line(geom.WithBins(10))

	warnings := layer.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected warning for WithBins on Line")
	}
}

func TestValidate_BarWithMethod_Warning(t *testing.T) {
	t.Parallel()

	layer := geom.Bar(geom.WithMethod("lm"))

	warnings := layer.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected warning for WithMethod on Bar")
	}
}

func TestValidate_TextWithBins_Warning(t *testing.T) {
	t.Parallel()

	layer := geom.Text(geom.WithBins(10), geom.WithWidth(0.5))

	warnings := layer.Validate()
	if len(warnings) < 2 {
		t.Errorf("expected >=2 warnings for WithBins+WithWidth on Text, got %d: %v", len(warnings), warnings)
	}
}

func TestValidate_SmoothWithBins_Warning(t *testing.T) {
	t.Parallel()

	layer := geom.Smooth(geom.WithBins(30))

	warnings := layer.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected warning for WithBins on Smooth")
	}
}

func TestValidate_HistogramWithMethod_Warning(t *testing.T) {
	t.Parallel()

	layer := geom.Histogram(geom.WithMethod("lm"))

	warnings := layer.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected warning for WithMethod on Histogram")
	}
}

// --- No warning tests (valid combinations) ---

func TestValidate_PointWithSize_NoWarning(t *testing.T) {
	t.Parallel()

	layer := geom.Point(geom.WithSize(5), geom.WithColor("#FF0000"))

	warnings := layer.Validate()
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings for valid Point opts: %v", warnings)
	}
}

func TestValidate_LineWithColor_NoWarning(t *testing.T) {
	t.Parallel()

	layer := geom.Line(geom.WithColor("#00FF00"), geom.WithLineWidth(3))

	warnings := layer.Validate()
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings for valid Line opts: %v", warnings)
	}
}

func TestValidate_BarWithWidth_NoWarning(t *testing.T) {
	t.Parallel()

	layer := geom.Bar(geom.WithWidth(0.6), geom.WithFill("#336699"))

	warnings := layer.Validate()
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings for valid Bar opts: %v", warnings)
	}
}

func TestValidate_HistogramWithBins_NoWarning(t *testing.T) {
	t.Parallel()

	layer := geom.Histogram(geom.WithBins(50), geom.WithAlpha(0.7))

	warnings := layer.Validate()
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings for valid Histogram opts: %v", warnings)
	}
}

func TestValidate_SmoothWithMethod_NoWarning(t *testing.T) {
	t.Parallel()

	layer := geom.Smooth(geom.WithMethod("loess"), geom.WithPoints(100))

	warnings := layer.Validate()
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings for valid Smooth opts: %v", warnings)
	}
}

func TestValidate_TextWithFontSize_NoWarning(t *testing.T) {
	t.Parallel()

	layer := geom.Text(geom.WithFontSize(14), geom.WithAngle(45))

	warnings := layer.Validate()
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings for valid Text opts: %v", warnings)
	}
}

func TestValidate_NoExplicitOpts_NoWarning(t *testing.T) {
	t.Parallel()

	layer := geom.Point()

	warnings := layer.Validate()
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings for default Point: %v", warnings)
	}
}

func TestValidate_DensityWithPoints_NoWarning(t *testing.T) {
	t.Parallel()

	layer := geom.Density(geom.WithPoints(1024), geom.WithFill("#993366"))

	warnings := layer.Validate()
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings for valid Density opts: %v", warnings)
	}
}

// --- WithSize portability test ---

func TestValidate_LineWithSize_NoWarning(t *testing.T) {
	t.Parallel()

	// WithSize sets both Size and LineWidth, and Size is relevant for Line
	layer := geom.Line(geom.WithSize(3))

	warnings := layer.Validate()
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings for Line(WithSize): %v", warnings)
	}
}
