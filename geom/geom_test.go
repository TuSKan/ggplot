package geom_test

import (
	"testing"

	"github.com/TuSKan/ggplot/geom"
)

// --- Warning tests (irrelevant options) ---

func TestValidate_PointWithWidth_Warning(t *testing.T) {
	t.Parallel()

	// WithWidth is for bar/histogram, not point.
	layer := geom.Point(geom.WithWidth(0.5))

	warnings := layer.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected warning for WithWidth on Point")
	}
}

func TestValidate_PointWithFontSize_Warning(t *testing.T) {
	t.Parallel()

	// WithFontSize is for text, not point.
	layer := geom.Point(geom.WithFontSize(14))

	warnings := layer.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected warning for WithFontSize on Point")
	}
}

func TestValidate_LineWithFontSize_Warning(t *testing.T) {
	t.Parallel()

	layer := geom.Line(geom.WithFontSize(10))

	warnings := layer.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected warning for WithFontSize on Line")
	}
}

func TestValidate_BarWithAngle_Warning(t *testing.T) {
	t.Parallel()

	layer := geom.Bar(geom.WithAngle(45))

	warnings := layer.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected warning for WithAngle on Bar")
	}
}

func TestValidate_TextWithWidth_Warning(t *testing.T) {
	t.Parallel()

	// Both Width and Orientation are irrelevant for Text.
	layer := geom.Text(geom.WithWidth(0.5), geom.WithOrientation(geom.Horizontal))

	warnings := layer.Validate()
	if len(warnings) < 2 {
		t.Errorf("expected >=2 warnings for WithWidth+WithOrientation on Text, got %d: %v", len(warnings), warnings)
	}
}

func TestValidate_SmoothWithWidth_Warning(t *testing.T) {
	t.Parallel()

	layer := geom.Smooth(geom.WithWidth(0.5))

	warnings := layer.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected warning for WithWidth on Smooth")
	}
}

func TestValidate_HistogramWithFontSize_Warning(t *testing.T) {
	t.Parallel()

	layer := geom.Histogram(geom.WithFontSize(14))

	warnings := layer.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected warning for WithFontSize on Histogram")
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

func TestValidate_HistogramWithAlpha_NoWarning(t *testing.T) {
	t.Parallel()

	layer := geom.Histogram(geom.WithAlpha(0.7))

	warnings := layer.Validate()
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings for valid Histogram opts: %v", warnings)
	}
}

func TestValidate_SmoothWithColor_NoWarning(t *testing.T) {
	t.Parallel()

	layer := geom.Smooth(geom.WithColor("#FF0000"), geom.WithLineWidth(2))

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

func TestValidate_DensityWithFill_NoWarning(t *testing.T) {
	t.Parallel()

	layer := geom.Density(geom.WithFill("#993366"))

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
