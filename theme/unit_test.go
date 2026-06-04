package theme

import (
	"math"
	"testing"
)

func TestUnit_ToPixels(t *testing.T) {
	t.Parallel()

	const lineHeight = 14.0 // 11pt × ~1.27

	tests := []struct {
		name      string
		unit      Unit
		lineH     float64
		wantPx    float64
		tolerance float64
	}{
		{"Pt zero", Pt(0), lineHeight, 0, 0},
		{"Pt 10", Pt(10), lineHeight, 10, 0},
		{"Pt negative", Pt(-5), lineHeight, -5, 0},
		{"Cm 1", Cm(1), lineHeight, 28.3464566929, 0.001},
		{"Cm 2.54", Cm(2.54), lineHeight, 72, 0.001},
		{"Inch 1", Inches(1), lineHeight, 72, 0},
		{"Inch 0.5", Inches(0.5), lineHeight, 36, 0},
		{"Lines 1", Lines(1), lineHeight, 14, 0},
		{"Lines 2", Lines(2), lineHeight, 28, 0},
		{"Lines zero lineH", Lines(2), 0, 28, 0},      // fallback lineHeight = 14
		{"Lines negative lineH", Lines(1), -5, 14, 0}, // fallback lineHeight = 14
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.unit.ToPixels(tc.lineH)
			if tc.tolerance == 0 {
				if got != tc.wantPx {
					t.Errorf("ToPixels() = %v, want %v", got, tc.wantPx)
				}
			} else {
				if math.Abs(got-tc.wantPx) > tc.tolerance {
					t.Errorf("ToPixels() = %v, want %v (±%v)", got, tc.wantPx, tc.tolerance)
				}
			}
		})
	}
}

func TestPlotMargin_AllUnits(t *testing.T) {
	t.Parallel()

	m := PlotMargin{
		Top:    Pt(10),
		Right:  Cm(1),
		Bottom: Inches(0.5),
		Left:   Lines(2),
	}

	const lineH = 14.0

	if got := m.Top.ToPixels(lineH); got != 10 {
		t.Errorf("Top = %v, want 10", got)
	}

	if got := m.Right.ToPixels(lineH); math.Abs(got-28.3464566929) > 0.001 {
		t.Errorf("Right = %v, want ~28.35", got)
	}

	if got := m.Bottom.ToPixels(lineH); got != 36 {
		t.Errorf("Bottom = %v, want 36", got)
	}

	if got := m.Left.ToPixels(lineH); got != 28 {
		t.Errorf("Left = %v, want 28", got)
	}
}

func TestResolvedPlotMargin_NilFallback(t *testing.T) {
	t.Parallel()

	th := baseTheme("test")
	// PlotMargin is nil → should fall back to Spacing values.
	top, right, bottom, left := th.ResolvedPlotMargin(14)

	if top != th.Spacing.MarginTop {
		t.Errorf("top = %v, want %v", top, th.Spacing.MarginTop)
	}

	if right != th.Spacing.MarginRight {
		t.Errorf("right = %v, want %v", right, th.Spacing.MarginRight)
	}

	if bottom != th.Spacing.MarginBottom {
		t.Errorf("bottom = %v, want %v", bottom, th.Spacing.MarginBottom)
	}

	if left != th.Spacing.MarginLeft {
		t.Errorf("left = %v, want %v", left, th.Spacing.MarginLeft)
	}
}

func TestResolvedPlotMargin_Custom(t *testing.T) {
	t.Parallel()

	th := baseTheme("test")
	th.PlotMargin = &PlotMargin{
		Top:    Pt(20),
		Right:  Pt(30),
		Bottom: Pt(25),
		Left:   Pt(15),
	}

	top, right, bottom, left := th.ResolvedPlotMargin(14)

	if top != 20 {
		t.Errorf("top = %v, want 20", top)
	}

	if right != 30 {
		t.Errorf("right = %v, want 30", right)
	}

	if bottom != 25 {
		t.Errorf("bottom = %v, want 25", bottom)
	}

	if left != 15 {
		t.Errorf("left = %v, want 15", left)
	}
}
