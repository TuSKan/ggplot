package scale

import (
	"fmt"
	"math"
	"testing"
)

// --- helpers ---

// trainLinear creates a trained linear scale with the given bounds.
func trainLinear(mn, mx float64) Scale {
	s := NewLinear()
	s.(*LinearScale).SetBounds(mn, mx)
	return s
}

// --- WithBreaks ---

func TestConfiguredScale_Breaks(t *testing.T) {
	inner := trainLinear(0, 100)
	cs := Configure(inner, WithBreaks([]float64{10, 30, 50, 70, 90}))

	ticks := cs.Ticks(5)
	want := []float64{10, 30, 50, 70, 90}
	if len(ticks) != len(want) {
		t.Fatalf("Ticks: got %v, want %v", ticks, want)
	}
	for i := range ticks {
		if ticks[i] != want[i] {
			t.Errorf("Ticks[%d] = %v, want %v", i, ticks[i], want[i])
		}
	}
}

func TestConfiguredScale_Breaks_NoMutation(t *testing.T) {
	input := []float64{1, 2, 3}
	inner := trainLinear(0, 10)
	cs := Configure(inner, WithBreaks(input))

	// Mutate original slice.
	input[0] = 999

	ticks := cs.Ticks(0)
	if ticks[0] != 1 {
		t.Errorf("expected 1, got %v — input slice mutation leaked", ticks[0])
	}
}

// --- WithLabels ---

func TestConfiguredScale_Labels(t *testing.T) {
	inner := trainLinear(0, 100)
	cs := Configure(inner,
		WithBreaks([]float64{0, 25, 50, 75, 100}),
		WithLabels([]string{"0%", "25%", "50%", "75%", "100%"}),
	)

	tests := []struct {
		v    float64
		want string
	}{
		{0, "0%"},
		{25, "25%"},
		{50, "50%"},
		{75, "75%"},
		{100, "100%"},
	}
	for _, tt := range tests {
		got := cs.Format(tt.v)
		if got != tt.want {
			t.Errorf("Format(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestConfiguredScale_Labels_Fallback(t *testing.T) {
	// Labels shorter than breaks → excess ticks use inner.Format.
	inner := trainLinear(0, 100)
	cs := Configure(inner,
		WithBreaks([]float64{0, 50, 100}),
		WithLabels([]string{"low", "mid"}), // no label for 100
	)

	if got := cs.Format(100); got != "100" {
		t.Errorf("Format(100) = %q, want fallback to inner (\"100\")", got)
	}
}

func TestConfiguredScale_Labels_NoBreaks(t *testing.T) {
	// Labels without explicit breaks → match against auto-generated ticks.
	inner := trainLinear(0, 10)
	cs := Configure(inner,
		WithLabels([]string{"a", "b", "c", "d", "e", "f"}),
	)
	// Auto ticks for [0,10] are typically [0, 2, 4, 6, 8, 10].
	autoTicks := inner.Ticks(5)
	if len(autoTicks) > 0 {
		got := cs.Format(autoTicks[0])
		if got != "a" {
			t.Errorf("Format(autoTicks[0]=%v) = %q, want \"a\"", autoTicks[0], got)
		}
	}
}

// --- WithFormatter ---

func TestConfiguredScale_Formatter(t *testing.T) {
	inner := trainLinear(0, 1000)
	cs := Configure(inner,
		WithFormatter(func(v float64) string {
			return fmt.Sprintf("$%.0f", v)
		}),
	)

	got := cs.Format(500)
	if got != "$500" {
		t.Errorf("Format(500) = %q, want \"$500\"", got)
	}
}

func TestConfiguredScale_Formatter_LabelsPriority(t *testing.T) {
	// When both labels and formatter are set, labels win for matching ticks.
	inner := trainLinear(0, 100)
	cs := Configure(inner,
		WithBreaks([]float64{0, 50, 100}),
		WithLabels([]string{"lo", "mid", "hi"}),
		WithFormatter(func(v float64) string {
			return fmt.Sprintf("fmt:%.0f", v)
		}),
	)

	// Tick value → label.
	if got := cs.Format(50); got != "mid" {
		t.Errorf("Format(50) = %q, want \"mid\" (labels take priority)", got)
	}
	// Non-tick value → formatter.
	if got := cs.Format(33); got != "fmt:33" {
		t.Errorf("Format(33) = %q, want \"fmt:33\" (formatter for non-tick value)", got)
	}
}

// --- WithExpand ---

func TestConfiguredScale_Expand(t *testing.T) {
	inner := trainLinear(0, 100) // range = 100
	cs := Configure(inner, WithExpand(0.05, 0))

	mn, mx := cs.Bounds()
	// 5% of 100 = 5 on each side → [-5, 105]
	if math.Abs(mn-(-5)) > 1e-10 {
		t.Errorf("Bounds min = %v, want -5", mn)
	}
	if math.Abs(mx-105) > 1e-10 {
		t.Errorf("Bounds max = %v, want 105", mx)
	}
}

func TestConfiguredScale_Expand_Additive(t *testing.T) {
	inner := trainLinear(10, 20) // range = 10
	cs := Configure(inner, WithExpand(0, 2))

	mn, mx := cs.Bounds()
	// add = 2 on each side → [8, 22]
	if math.Abs(mn-8) > 1e-10 {
		t.Errorf("Bounds min = %v, want 8", mn)
	}
	if math.Abs(mx-22) > 1e-10 {
		t.Errorf("Bounds max = %v, want 22", mx)
	}
}

func TestConfiguredScale_Expand_Combined(t *testing.T) {
	inner := trainLinear(0, 100) // range = 100
	cs := Configure(inner, WithExpand(0.1, 5))

	mn, mx := cs.Bounds()
	// mult=0.1*100=10, add=5 → 15 on each side → [-15, 115]
	if math.Abs(mn-(-15)) > 1e-10 {
		t.Errorf("Bounds min = %v, want -15", mn)
	}
	if math.Abs(mx-115) > 1e-10 {
		t.Errorf("Bounds max = %v, want 115", mx)
	}
}

func TestConfiguredScale_Expand_HasExpand(t *testing.T) {
	inner := trainLinear(0, 100)

	// Without WithExpand → HasExpand false.
	cs := Configure(inner, WithBreaks([]float64{0, 50, 100}))
	if exp, ok := cs.(Expander); ok && exp.HasExpand() {
		t.Error("HasExpand should be false when WithExpand not called")
	}

	// With WithExpand → HasExpand true.
	cs2 := Configure(inner, WithExpand(0, 0))
	if exp, ok := cs2.(Expander); !ok || !exp.HasExpand() {
		t.Error("HasExpand should be true when WithExpand was called")
	}
}

// --- WithMinorBreaks ---

func TestConfiguredScale_MinorBreaks_Explicit(t *testing.T) {
	inner := trainLinear(0, 100)
	cs := Configure(inner, WithMinorBreaks([]float64{5, 15, 25, 35}))

	mt, ok := cs.(MinorTicker)
	if !ok {
		t.Fatal("ConfiguredScale does not implement MinorTicker")
	}
	minor := mt.MinorTicks()
	want := []float64{5, 15, 25, 35}
	if len(minor) != len(want) {
		t.Fatalf("MinorTicks: got %v, want %v", minor, want)
	}
	for i := range minor {
		if minor[i] != want[i] {
			t.Errorf("MinorTicks[%d] = %v, want %v", i, minor[i], want[i])
		}
	}
}

func TestConfiguredScale_MinorBreaks_Auto(t *testing.T) {
	// Without WithMinorBreaks, auto-generate midpoints.
	inner := trainLinear(0, 100)
	cs := Configure(inner, WithBreaks([]float64{0, 20, 40, 60, 80, 100}))

	mt := cs.(MinorTicker)
	minor := mt.MinorTicks()
	// Midpoints: 10, 30, 50, 70, 90
	want := []float64{10, 30, 50, 70, 90}
	if len(minor) != len(want) {
		t.Fatalf("auto MinorTicks: got %v, want %v", minor, want)
	}
	for i := range minor {
		if math.Abs(minor[i]-want[i]) > 1e-10 {
			t.Errorf("auto MinorTicks[%d] = %v, want %v", i, minor[i], want[i])
		}
	}
}

// --- WithClipBounds ---

func TestConfiguredScale_ClipBounds(t *testing.T) {
	inner := trainLinear(0, 100) // data domain [0, 100]
	cs := Configure(inner, WithClipBounds(20, 80))

	mn, mx := cs.Bounds()
	if mn != 20 || mx != 80 {
		t.Errorf("Bounds = [%v, %v], want [20, 80]", mn, mx)
	}

	// Map/Inverse should use the clipped bounds.
	if got := cs.Map(50); math.Abs(got-0.5) > 1e-10 {
		t.Errorf("Map(50) = %v, want 0.5 (midpoint of [20,80])", got)
	}
}

func TestConfiguredScale_ClipBounds_PartialNaN(t *testing.T) {
	inner := trainLinear(0, 100)
	cs := Configure(inner, WithClipBounds(math.NaN(), 50))

	mn, mx := cs.Bounds()
	if mn != 0 {
		t.Errorf("min should remain 0 (NaN = auto), got %v", mn)
	}
	if mx != 50 {
		t.Errorf("max should be 50, got %v", mx)
	}
}

func TestConfiguredScale_ClipBounds_OverridesExpand(t *testing.T) {
	inner := trainLinear(0, 100)
	cs := Configure(inner,
		WithExpand(0.1, 0),       // would make [-10, 110]
		WithClipBounds(10, 90),   // clip overrides expand
	)

	mn, mx := cs.Bounds()
	if mn != 10 || mx != 90 {
		t.Errorf("ClipBounds should override expand: got [%v, %v], want [10, 90]", mn, mx)
	}
}

// --- Composition ---

func TestConfiguredScale_Compose(t *testing.T) {
	inner := trainLinear(0, 100)
	cs := Configure(inner,
		WithBreaks([]float64{0, 25, 50, 75, 100}),
		WithLabels([]string{"0%", "25%", "50%", "75%", "100%"}),
		WithExpand(0.02, 0),
		WithMinorBreaks([]float64{12.5, 37.5, 62.5, 87.5}),
	)

	// Ticks should be the custom breaks.
	ticks := cs.Ticks(0)
	if len(ticks) != 5 {
		t.Fatalf("expected 5 ticks, got %d", len(ticks))
	}

	// Labels should work.
	if got := cs.Format(50); got != "50%" {
		t.Errorf("Format(50) = %q, want \"50%%\"", got)
	}

	// Expand should widen bounds.
	mn, mx := cs.Bounds()
	if mn >= 0 || mx <= 100 {
		t.Errorf("Bounds should be expanded: got [%v, %v]", mn, mx)
	}

	// Minor ticks should be the explicit ones.
	mt := cs.(MinorTicker)
	if len(mt.MinorTicks()) != 4 {
		t.Errorf("expected 4 minor ticks, got %d", len(mt.MinorTicks()))
	}
}

// --- BoundsSetter delegation ---

func TestConfiguredScale_BoundsSetter(t *testing.T) {
	inner := trainLinear(0, 100)
	cs := Configure(inner, WithExpand(0.05, 0))

	bs, ok := cs.(BoundsSetter)
	if !ok {
		t.Fatal("ConfiguredScale does not implement BoundsSetter")
	}

	// SetBounds should delegate to inner.
	bs.SetBounds(10, 50) // inner now [10, 50]

	// Bounds should reflect inner [10,50] + expand (5% of 40 = 2).
	mn, mx := cs.Bounds()
	if math.Abs(mn-8) > 1e-10 {
		t.Errorf("after SetBounds: min = %v, want 8", mn)
	}
	if math.Abs(mx-52) > 1e-10 {
		t.Errorf("after SetBounds: max = %v, want 52", mx)
	}
}

// --- Configure with no opts ---

func TestConfigure_NoOpts(t *testing.T) {
	inner := trainLinear(0, 100)
	got := Configure(inner)
	// Should return the exact same scale, not a wrapper.
	if got != inner {
		t.Error("Configure with no opts should return inner unchanged")
	}
}

// --- Map/Inverse consistency ---

func TestConfiguredScale_MapInverse_Roundtrip(t *testing.T) {
	inner := trainLinear(0, 100)
	cs := Configure(inner, WithExpand(0.1, 5))

	for _, v := range []float64{0, 25, 50, 75, 100} {
		norm := cs.Map(v)
		back := cs.Inverse(norm)
		if math.Abs(back-v) > 1e-10 {
			t.Errorf("roundtrip failed: %v → %v → %v", v, norm, back)
		}
	}
}
