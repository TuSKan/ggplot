package position

import (
	"math"
	"testing"
)

func TestIdentity(t *testing.T) {
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
	xs := []float64{1, 2, 3}
	ys := []float64{4, 5, 6}
	rx, _ := Dodge().Adjust(xs, ys, 0.8, 0, 1)
	// Single group → no adjustment
	if &rx[0] != &xs[0] {
		t.Error("Dodge with 1 group should return same slice")
	}
}

func TestDodge_MultiGroup(t *testing.T) {
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

func TestStack_GroupZero_Passthrough(t *testing.T) {
	xs := []float64{1, 2}
	ys := []float64{3, 4}
	rx, ry := Stack().Adjust(xs, ys, 1, 0, 2)
	if &rx[0] != &xs[0] || &ry[0] != &ys[0] {
		t.Error("Stack groupIdx=0 should return same slices")
	}
}

func TestStack_GroupNonZero_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Stack with groupIdx > 0 should panic")
		}
	}()
	Stack().Adjust([]float64{1}, []float64{2}, 1, 1, 2)
}

func TestJitter_Distribution(t *testing.T) {
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
