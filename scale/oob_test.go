package scale_test

import (
	"math"
	"testing"

	"github.com/TuSKan/ggplot/scale"
)

func TestOOBCensor(t *testing.T) {
	t.Parallel()

	inner := trainLinear(0, 100)
	cs := scale.Configure(inner, scale.WithOOB(scale.OOBCensor))

	// In-bounds value should map normally.
	got := cs.Map(50)
	if math.Abs(got-0.5) > 1e-10 {
		t.Errorf("Map(50) in-bounds = %v, want 0.5", got)
	}

	// Out-of-bounds value should be NaN.
	got = cs.Map(150)
	if !math.IsNaN(got) {
		t.Errorf("Map(150) OOBCensor = %v, want NaN", got)
	}

	got = cs.Map(-10)
	if !math.IsNaN(got) {
		t.Errorf("Map(-10) OOBCensor = %v, want NaN", got)
	}
}

func TestOOBSquish(t *testing.T) {
	t.Parallel()

	inner := trainLinear(0, 100)
	cs := scale.Configure(inner, scale.WithOOB(scale.OOBSquish))

	// In-bounds.
	got := cs.Map(50)
	if math.Abs(got-0.5) > 1e-10 {
		t.Errorf("Map(50) in-bounds = %v, want 0.5", got)
	}

	// Above bounds → clamped to 1.
	got = cs.Map(200)
	if got != 1.0 {
		t.Errorf("Map(200) OOBSquish = %v, want 1.0", got)
	}

	// Below bounds → clamped to 0.
	got = cs.Map(-50)
	if got != 0.0 {
		t.Errorf("Map(-50) OOBSquish = %v, want 0.0", got)
	}
}

func TestOOBKeep(t *testing.T) {
	t.Parallel()

	inner := trainLinear(0, 100)
	cs := scale.Configure(inner, scale.WithOOB(scale.OOBKeep))

	// Out-of-bounds should be passed through.
	got := cs.Map(200)
	if got != 2.0 {
		t.Errorf("Map(200) OOBKeep = %v, want 2.0", got)
	}
}

func TestBinnedScale_MapFormat(t *testing.T) {
	t.Parallel()

	sc := scale.NewBinned(scale.BinnedBins(5))
	setBounds(t, sc, 0, 100)

	// Ticks should return one data-space center per bin.
	ticks := sc.Ticks(0)
	if len(ticks) != 5 {
		t.Fatalf("Ticks: got %d, want 5", len(ticks))
	}

	// First center = (0+20)/2 = 10 for 5 bins over [0,100].
	if math.Abs(ticks[0]-10) > 1e-10 {
		t.Errorf("ticks[0] = %v, want 10", ticks[0])
	}

	// Format should produce range labels.
	label := sc.Format(ticks[0])
	if label != "[0, 20)" {
		t.Errorf("Format(10) = %q, want %q", label, "[0, 20)")
	}

	// Map: linear normalization through [0,100].
	got := sc.Map(50)

	want := 0.5
	if math.Abs(got-want) > 1e-10 {
		t.Errorf("Map(50) = %v, want %v", got, want)
	}

	// String identifier.
	if sc.String() != "binned" {
		t.Errorf("String() = %q, want %q", sc.String(), "binned")
	}
}

func TestBinnedScale_ExplicitBreaks(t *testing.T) {
	t.Parallel()

	sc := scale.NewBinned(scale.BinnedBreaks([]float64{0, 10, 20, 30, 40, 50}))
	setBounds(t, sc, 0, 50)

	ticks := sc.Ticks(0)
	if len(ticks) != 5 {
		t.Fatalf("Ticks with 6 edges: got %d centers, want 5", len(ticks))
	}

	// First center = (0+10)/2 = 5.
	if math.Abs(ticks[0]-5) > 1e-10 {
		t.Errorf("first tick = %v, want 5", ticks[0])
	}

	// Inverse(0.1) → data value 5 (= 0 + 0.1*50).
	if inv := sc.Inverse(0.1); math.Abs(inv-5) > 1e-10 {
		t.Errorf("Inverse(0.1) = %v, want 5", inv)
	}
}
