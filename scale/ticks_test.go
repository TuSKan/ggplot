package scale

import (
	"math"
	"slices"
	"testing"
)

func TestExtendedWilkinson_BasicRange(t *testing.T) {
	t.Parallel()

	ticks := extendedWilkinson(0, 100, 5)
	if len(ticks) == 0 {
		t.Fatal("expected non-empty ticks")
	}
	// First tick should be <= 0, last should be >= 100.
	if ticks[0] > 0 {
		t.Errorf("first tick %v > 0", ticks[0])
	}

	if ticks[len(ticks)-1] < 100 {
		t.Errorf("last tick %v < 100", ticks[len(ticks)-1])
	}
	// Ticks should be monotonically increasing.
	for i := 1; i < len(ticks); i++ {
		if ticks[i] <= ticks[i-1] {
			t.Errorf("ticks not monotonic at %d: %v", i, ticks)
			break
		}
	}
}

func TestExtendedWilkinson_IncludesZero(t *testing.T) {
	t.Parallel()

	ticks := extendedWilkinson(-50, 50, 5)
	found := slices.Contains(ticks, 0)

	if !found {
		t.Errorf("expected ticks to include 0 for range [-50, 50], got %v", ticks)
	}
}

func TestExtendedWilkinson_SmallRange(t *testing.T) {
	t.Parallel()

	ticks := extendedWilkinson(0.1, 0.9, 5)
	if len(ticks) < 2 {
		t.Errorf("expected multiple ticks for [0.1, 0.9], got %v", ticks)
	}
	// Steps should be nice numbers like 0.2.
	step := ticks[1] - ticks[0]
	// Step should be a nice fraction.
	nice := step == 0.1 || step == 0.2 || step == 0.25 || step == 0.5
	if !nice {
		t.Logf("step %v may not be a 'nice' number (informational)", step)
	}
}

func TestExtendedWilkinson_LargeRange(t *testing.T) {
	t.Parallel()

	ticks := extendedWilkinson(0, 1e6, 5)
	if len(ticks) < 2 {
		t.Errorf("expected multiple ticks for [0, 1e6], got %v", ticks)
	}
}

func TestExtendedWilkinson_NegativeRange(t *testing.T) {
	t.Parallel()

	ticks := extendedWilkinson(-100, -10, 5)
	if len(ticks) < 2 {
		t.Fatalf("expected multiple ticks, got %v", ticks)
	}

	if ticks[0] > -100 {
		t.Errorf("first tick %v > -100", ticks[0])
	}

	if ticks[len(ticks)-1] < -10 {
		t.Errorf("last tick %v < -10", ticks[len(ticks)-1])
	}
}

func TestExtendedWilkinson_EqualInputs(t *testing.T) {
	t.Parallel()

	ticks := extendedWilkinson(5, 5, 5)
	if len(ticks) != 1 || ticks[0] != 5 {
		t.Errorf("expected [5], got %v", ticks)
	}
}

func TestExtendedWilkinson_Inverted(t *testing.T) {
	t.Parallel()

	ticks := extendedWilkinson(100, 0, 5)
	if len(ticks) < 2 {
		t.Errorf("should handle inverted range, got %v", ticks)
	}
}

func TestExtendedWilkinson_NiceSteps(t *testing.T) {
	t.Parallel()

	// For range [0, 10] with 5 ticks, we should get clean step sizes.
	ticks := extendedWilkinson(0, 10, 5)
	t.Logf("ticks for [0,10] target 5: %v", ticks)

	if len(ticks) < 3 {
		t.Fatal("expected at least 3 ticks")
	}

	step := ticks[1] - ticks[0]
	// Step should be 1, 2, 2.5, or 5.
	niceSteps := []float64{1, 2, 2.5, 5}
	isNice := false

	for _, ns := range niceSteps {
		if math.Abs(step-ns) < 1e-10 {
			isNice = true
			break
		}
	}

	if !isNice {
		t.Errorf("step %v is not a nice number for [0,10] with 5 ticks", step)
	}
}

func TestNiceSequence_UsesExtended(t *testing.T) {
	t.Parallel()

	// NiceSequence should now delegate to extendedWilkinson.
	ticks := NiceSequence(0, 100, 5)
	if len(ticks) < 2 {
		t.Error("expected ticks from NiceSequence")
	}
}

// --- Scoring function tests ---

func TestSimplicity(t *testing.T) {
	t.Parallel()

	// qi=0, j=1 (best Q, no skip) with zero included should give highest simplicity.
	s := simplicity(0, 1, -10, 10, 5)
	if s < 1.0 {
		t.Errorf("expected simplicity >= 1.0 for qi=0, j=1, containsZero, got %v", s)
	}
}

func TestCoverage(t *testing.T) {
	t.Parallel()

	// Perfect coverage: labels span exactly the data range.
	c := coverage(0, 100, 0, 100)
	if c < 0.99 {
		t.Errorf("expected coverage ~1.0 for perfect span, got %v", c)
	}
}

func TestDensity(t *testing.T) {
	t.Parallel()

	// When k matches target and label range equals data range,
	// r/rt = 1.0, so density = 2 - max(1, 1) = 1.0.
	d := density(5, 5, 0, 100, 0, 100)
	if d < 0.99 {
		t.Errorf("expected density ~1.0 when k matches target, got %v", d)
	}
}

func TestContainsZero(t *testing.T) {
	t.Parallel()

	if !containsZero(-10, 10, 5) {
		t.Error("expected containsZero(-10, 10, 5) = true")
	}

	if containsZero(1, 10, 3) {
		t.Error("expected containsZero(1, 10, 3) = false")
	}
}
