package scale_test

import (
	"context"
	"math"
	"testing"

	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/scale"
)

func TestSizeScale_Linear(t *testing.T) {
	t.Parallel()

	s := scale.NewSizeDefault()
	rmin, rmax := s.Range()

	if rmin != 1.0 || rmax != 6.0 {
		t.Errorf("expected default range [1.0, 6.0], got [%v, %v]", rmin, rmax)
	}

	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("size_data", []float64{10, 20, 30})

	if err := s.Train(col); err != nil {
		t.Fatalf("unexpected error training: %v", err)
	}

	tests := []struct {
		val  float64
		want float64
	}{
		{10, 1.0},
		{20, 3.5},
		{30, 6.0},
	}

	for _, tt := range tests {
		got := s.MapValue(tt.val)
		if got != tt.want {
			t.Errorf("Linear MapValue(%v) = %v, want %v", tt.val, got, tt.want)
		}
	}
}

func TestSizeScale_Area(t *testing.T) {
	t.Parallel()

	s := scale.NewSizeArea()
	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("size_data", []float64{0, 4, 16})

	if err := s.Train(col); err != nil {
		t.Fatalf("unexpected error training: %v", err)
	}

	tests := []struct {
		val  float64
		want float64
	}{
		{0, 1.0},  // sqrt(0) = 0 -> maps to rangeMin (1.0)
		{4, 2.25}, // sqrt(4) = 2, min=0, max=4. Map: (2-0)/(4-0) = 0.5. MapValue: 1.0 + 0.5 * 5 = 3.5. Wait, sqrt(16) = 4, sqrt(0) = 0. Map of 4: (2-0)/(4-0) = 0.5. MapValue: 1.0 + 0.5*(6-1) = 3.5?
		{16, 6.0}, // sqrt(16) = 4. Map: (4-0)/(4-0) = 1. MapValue: 6.0
	}

	// Wait, let's calculate:
	// t = (sqrt(4) - sqrt(0)) / (sqrt(16) - sqrt(0)) = (2 - 0) / (4 - 0) = 0.5
	// MapValue = 1.0 + 0.5 * (6.0 - 1.0) = 3.5
	// Let's adjust the test case:
	tests[1].want = 3.5

	for _, tt := range tests {
		got := s.MapValue(tt.val)
		if math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("Area MapValue(%v) = %v, want %v", tt.val, got, tt.want)
		}
	}
}

func TestSizeScale_ClampOutOfDomain(t *testing.T) {
	t.Parallel()

	s := scale.NewSize(2.0, 8.0)
	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("data", []float64{10, 20})

	if err := s.Train(col); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Value below domain → should clamp to rangeMin.
	got := s.MapValue(0)
	if got < 2.0 {
		t.Errorf("MapValue(0) = %v, want >= 2.0 (clamped)", got)
	}

	// Value above domain → should clamp to rangeMax.
	got = s.MapValue(100)
	if got > 8.0 {
		t.Errorf("MapValue(100) = %v, want <= 8.0 (clamped)", got)
	}
}

func TestSizeScale_StringConsistency(t *testing.T) {
	t.Parallel()

	s := scale.NewSizeDefault()
	if got := s.String(); got != "size" {
		t.Errorf("String() = %q, want \"size\"", got)
	}

	sa := scale.NewSizeArea()
	if got := sa.String(); got != "size" {
		t.Errorf("area String() = %q, want \"size\"", got)
	}

	if sa.Mode() != scale.SizeModeArea {
		t.Errorf("Mode() = %v, want SizeModeArea", sa.Mode())
	}
}

func TestSizeScale_ValueMapper(t *testing.T) {
	t.Parallel()

	s := scale.NewSizeDefault()

	var vm scale.ValueMapper = s

	_ = vm // compile-time check
}
