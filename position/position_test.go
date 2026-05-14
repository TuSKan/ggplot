package position

import (
	"math"
	"testing"
)

func TestIdentity(t *testing.T) {
	t.Parallel()

	xs := []float64{1, 2, 3}
	ys := []float64{4, 5, 6}

	rx, ry := Identity().Adjust(xs, ys, 1, 0, 1)
	if &rx[0] != &xs[0] {
		t.Error("Identity should return the same slice")
	}

	if &ry[0] != &ys[0] {
		t.Error("Identity should return the same slice")
	}
}

func TestDodge_SingleGroup(t *testing.T) {
	t.Parallel()

	xs := []float64{1, 2, 3}
	ys := []float64{4, 5, 6}
	rx, _ := Dodge().Adjust(xs, ys, 0.8, 0, 1)
	// Single group -> no adjustment
	if &rx[0] != &xs[0] {
		t.Error("Dodge with 1 group should return same slice")
	}
}

func TestDodge_MultiGroup(t *testing.T) {
	t.Parallel()

	xs := []float64{1, 1, 1}
	ys := []float64{10, 20, 30}

	rx0, _ := Dodge().Adjust(xs, ys, 0.9, 0, 3)
	rx1, _ := Dodge().Adjust(xs, ys, 0.9, 1, 3)
	rx2, _ := Dodge().Adjust(xs, ys, 0.9, 2, 3)

	// All three groups should be at different x positions
	if rx0[0] == rx1[0] || rx1[0] == rx2[0] {
		t.Errorf("Dodge groups overlap: %v, %v, %v", rx0[0], rx1[0], rx2[0])
	}
	// Should be approximately centered around original x=1
	mean := (rx0[0] + rx1[0] + rx2[0]) / 3
	if math.Abs(mean-1) > 0.01 {
		t.Errorf("Dodge mean = %f, want ~1.0", mean)
	}
}

func TestStack_Accumulates(t *testing.T) {
	t.Parallel()

	s := Stack()

	// Group 0: x=1 -> y=10, x=2 -> y=20
	_, ry0 := s.Adjust([]float64{1, 2}, []float64{10, 20}, 1, 0, 2)

	// Group 0 sees no prior offset.
	if ry0[0] != 10 {
		t.Errorf("Stack group0 x=1: got %f, want 10", ry0[0])
	}

	if ry0[1] != 20 {
		t.Errorf("Stack group0 x=2: got %f, want 20", ry0[1])
	}

	// Group 1: x=1 -> y=5, x=2 -> y=8
	_, ry1 := s.Adjust([]float64{1, 2}, []float64{5, 8}, 1, 1, 2)

	// Group 1 should be stacked on top of group 0.
	if ry1[0] != 15 { // 10 + 5
		t.Errorf("Stack group1 x=1: got %f, want 15", ry1[0])
	}

	if ry1[1] != 28 { // 20 + 8
		t.Errorf("Stack group1 x=2: got %f, want 28", ry1[1])
	}
}

func TestStack_FreshPerPanel(t *testing.T) {
	t.Parallel()

	// Two separate Stack instances should be independent.
	s1 := Stack()
	s2 := Stack()

	s1.Adjust([]float64{1}, []float64{100}, 1, 0, 1)
	_, ry := s2.Adjust([]float64{1}, []float64{5}, 1, 0, 1)

	// s2 should NOT see s1's offsets.
	if ry[0] != 5 {
		t.Errorf("Fresh Stack contaminated: got %f, want 5", ry[0])
	}
}

func TestFill_Normalization(t *testing.T) {
	t.Parallel()

	f := Fill()

	// Setup with all groups.
	allXs := [][]float64{{1, 2}, {1, 2}}
	allYs := [][]float64{{30, 40}, {70, 60}}

	fs, ok := f.(FillSetup)
	if !ok {
		t.Fatal("Fill does not implement FillSetup")
	}

	fs.Setup(allXs, allYs)

	// Group 0: x=1 -> 30/100=0.3, x=2 -> 40/100=0.4
	_, ry0 := f.Adjust([]float64{1, 2}, []float64{30, 40}, 1, 0, 2)

	if math.Abs(ry0[0]-0.3) > 1e-9 {
		t.Errorf("Fill group0 x=1: got %f, want 0.3", ry0[0])
	}

	if math.Abs(ry0[1]-0.4) > 1e-9 {
		t.Errorf("Fill group0 x=2: got %f, want 0.4", ry0[1])
	}

	// Group 1: x=1 -> 0.3 + 70/100 = 1.0, x=2 -> 0.4 + 60/100 = 1.0
	_, ry1 := f.Adjust([]float64{1, 2}, []float64{70, 60}, 1, 1, 2)

	if math.Abs(ry1[0]-1.0) > 1e-9 {
		t.Errorf("Fill group1 x=1: got %f, want 1.0", ry1[0])
	}

	if math.Abs(ry1[1]-1.0) > 1e-9 {
		t.Errorf("Fill group1 x=2: got %f, want 1.0", ry1[1])
	}
}

func TestNew_Factory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name Name
		want string
	}{
		{NameIdentity, "identity"},
		{NameDodge, "dodge"},
		{NameStack, "stack"},
		{NameFill, "fill"},
		{"", "identity"},
		{"unknown", "identity"},
	}

	for _, tt := range tests {
		if got := New(tt.name).String(); got != tt.want {
			t.Errorf("New(%q).String() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestJitter_Distribution(t *testing.T) {
	t.Parallel()

	xs := make([]float64, 100)
	ys := make([]float64, 100)

	for i := range xs {
		xs[i] = 5
		ys[i] = 10
	}

	rx, ry := Jitter(0.5, 0.3).Adjust(xs, ys, 1, 0, 1)

	// Check that values are spread around the original
	var sumDX, sumDY float64

	for i := range rx {
		dx := rx[i] - 5
		dy := ry[i] - 10
		sumDX += dx
		sumDY += dy

		if math.Abs(dx) > 0.25 {
			t.Errorf("Jitter x[%d] out of range: dx=%f", i, dx)
		}

		if math.Abs(dy) > 0.15 {
			t.Errorf("Jitter y[%d] out of range: dy=%f", i, dy)
		}
	}
	// Mean displacement should be approximately 0 (unbiased)
	meanDX := sumDX / float64(len(xs))
	if math.Abs(meanDX) > 0.05 {
		t.Errorf("Jitter mean X displacement = %f, want ~0", meanDX)
	}
}

func TestJitter_Reproducible(t *testing.T) {
	t.Parallel()

	xs := []float64{1, 2, 3, 4, 5}
	ys := []float64{5, 4, 3, 2, 1}
	j := Jitter(0.4, 0.4)
	rx1, ry1 := j.Adjust(xs, ys, 1, 0, 1)

	rx2, ry2 := j.Adjust(xs, ys, 1, 0, 1)
	for i := range rx1 {
		if rx1[i] != rx2[i] || ry1[i] != ry2[i] {
			t.Errorf("Jitter not reproducible at index %d", i)
		}
	}
}

func TestNudge(t *testing.T) {
	t.Parallel()

	xs := []float64{1, 2, 3}
	ys := []float64{4, 5, 6}

	rx, ry := Nudge(0.5, -1).Adjust(xs, ys, 1, 0, 1)
	for i := range rx {
		if rx[i] != xs[i]+0.5 {
			t.Errorf("Nudge x[%d] = %f, want %f", i, rx[i], xs[i]+0.5)
		}

		if ry[i] != ys[i]-1 {
			t.Errorf("Nudge y[%d] = %f, want %f", i, ry[i], ys[i]-1)
		}
	}
}
