package scale

import (
	"testing"
)

func TestDiscrete_DefaultBounds(t *testing.T) {
	t.Parallel()

	s := Discrete()
	s.TrainValues([]string{"a", "b", "c"})

	lo, hi := s.Bounds()
	if lo != -0.5 || hi != 2.5 {
		t.Errorf("default bounds: got (%.2f, %.2f), want (-0.50, 2.50)", lo, hi)
	}
}

func TestDiscrete_CustomPaddingOuter(t *testing.T) {
	t.Parallel()

	s := Discrete(WithPaddingOuter(1.0))
	s.TrainValues([]string{"a", "b", "c"})

	lo, hi := s.Bounds()
	if lo != -1.0 || hi != 3.0 {
		t.Errorf("outer=1.0 bounds: got (%.2f, %.2f), want (-1.00, 3.00)", lo, hi)
	}
}

func TestDiscrete_ZeroPaddingOuter(t *testing.T) {
	t.Parallel()

	s := Discrete(WithPaddingOuter(0))
	s.TrainValues([]string{"a", "b", "c"})

	lo, hi := s.Bounds()
	if lo != 0 || hi != 2.0 {
		t.Errorf("outer=0 bounds: got (%.2f, %.2f), want (0.00, 2.00)", lo, hi)
	}
}

func TestDiscrete_BandWidthDefault(t *testing.T) {
	t.Parallel()

	s := Discrete()
	if w := s.BandWidth(); w != 0.8 {
		t.Errorf("default BandWidth: got %.2f, want 0.80", w)
	}
}

func TestDiscrete_BandWidthCustom(t *testing.T) {
	t.Parallel()

	s := Discrete(WithPaddingInner(0.3))
	if w := s.BandWidth(); w != 0.7 {
		t.Errorf("inner=0.3 BandWidth: got %.2f, want 0.70", w)
	}
}

func TestDiscrete_BandWidthFloor(t *testing.T) {
	t.Parallel()

	s := Discrete(WithPaddingInner(0.95))
	if w := s.BandWidth(); w != 0.1 {
		t.Errorf("inner=0.95 BandWidth: got %.2f, want 0.10 (floor)", w)
	}
}

func TestDiscrete_MapCategory(t *testing.T) {
	t.Parallel()

	s := Discrete()
	s.TrainValues([]string{"x", "y", "z"})

	tests := []struct {
		label string
		want  float64
	}{
		{"x", 0},
		{"y", 1},
		{"z", 2},
	}
	for _, tt := range tests {
		if got := s.MapCategory(tt.label); got != tt.want {
			t.Errorf("MapCategory(%q) = %.2f, want %.2f", tt.label, got, tt.want)
		}
	}
}

func TestDiscrete_EmptyBounds(t *testing.T) {
	t.Parallel()

	s := Discrete()

	lo, hi := s.Bounds()
	if lo != 0 || hi != 1 {
		t.Errorf("empty bounds: got (%.2f, %.2f), want (0.00, 1.00)", lo, hi)
	}
}
