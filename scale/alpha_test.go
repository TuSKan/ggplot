package scale_test

import (
	"context"
	"testing"

	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/scale"
)

func TestAlphaScale(t *testing.T) {
	t.Parallel()

	// Default scale
	s := scale.NewAlphaDefault()
	if s.String() != "alpha" {
		t.Errorf("expected 'alpha', got %q", s.String())
	}

	rmin, rmax := s.Range()
	if rmin != 0.1 || rmax != 1.0 {
		t.Errorf("expected default range [0.1, 1.0], got [%v, %v]", rmin, rmax)
	}

	// Custom range
	sCustom := scale.NewAlpha(0.2, 0.8)
	rmin, rmax = sCustom.Range()

	if rmin != 0.2 || rmax != 0.8 {
		t.Errorf("expected range [0.2, 0.8], got [%v, %v]", rmin, rmax)
	}

	// Train and map
	eng := memory.NewEngine(context.Background())
	mockCol := eng.NewFloat64Column("alpha_data", []float64{10, 20, 30})

	if err := sCustom.Train(mockCol); err != nil {
		t.Fatalf("unexpected error training: %v", err)
	}

	tests := []struct {
		val  float64
		want float64
	}{
		{10, 0.2},
		{20, 0.5},
		{30, 0.8},
	}

	for _, tt := range tests {
		got := sCustom.MapValue(tt.val)
		if got != tt.want {
			t.Errorf("MapValue(%v) = %v, want %v", tt.val, got, tt.want)
		}
	}
}

func TestAlphaScale_ClampOutOfDomain(t *testing.T) {
	t.Parallel()

	s := scale.NewAlpha(0.2, 0.8)
	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("data", []float64{10, 20})

	if err := s.Train(col); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Value below domain → should clamp to rangeMin.
	got := s.MapValue(0)
	if got < 0.2 {
		t.Errorf("MapValue(0) = %v, want >= 0.2 (clamped)", got)
	}

	// Value above domain → should clamp to rangeMax.
	got = s.MapValue(100)
	if got > 0.8 {
		t.Errorf("MapValue(100) = %v, want <= 0.8 (clamped)", got)
	}
}

func TestAlphaScale_ZeroRange(t *testing.T) {
	t.Parallel()

	// NewAlpha(0, 0) should now give literal zero range, not defaults.
	s := scale.NewAlpha(0, 0)

	rmin, rmax := s.Range()
	if rmin != 0 || rmax != 0 {
		t.Errorf("NewAlpha(0,0) range = [%v, %v], want [0, 0]", rmin, rmax)
	}
}

func TestAlphaScale_ValueMapper(t *testing.T) {
	t.Parallel()

	s := scale.NewAlphaDefault()

	var vm scale.ValueMapper = s

	_ = vm // compile-time check
}
