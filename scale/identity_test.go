package scale_test

import (
	"context"
	"testing"

	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/scale"
)

func TestIdentityScale(t *testing.T) {
	t.Parallel()

	s := scale.NewIdentity()
	if s.String() != "identity" {
		t.Errorf("expected string 'identity', got %q", s.String())
	}

	// Test mapping before training
	if got := s.Map(42.5); got != 42.5 {
		t.Errorf("expected Map(42.5) = 42.5, got %v", got)
	}

	if got := s.Inverse(99.9); got != 99.9 {
		t.Errorf("expected Inverse(99.9) = 99.9, got %v", got)
	}

	// Test training from column
	eng := memory.NewEngine(context.Background())
	mockCol := eng.NewFloat64Column("test", []float64{10, 5, 20, 15})

	if err := s.Train(mockCol); err != nil {
		t.Fatalf("unexpected error training: %v", err)
	}

	mn, mx := s.Bounds()
	if mn != 5 || mx != 20 {
		t.Errorf("expected bounds [5, 20], got [%v, %v]", mn, mx)
	}

	// Ticks
	ticks := s.Ticks(3)
	if len(ticks) == 0 {
		t.Error("expected ticks to not be empty")
	}

	// SetBounds
	s.SetBounds(0, 100)
	mn, mx = s.Bounds()

	if mn != 0 || mx != 100 {
		t.Errorf("expected bounds [0, 100], got [%v, %v]", mn, mx)
	}

	// MapValue — identity should pass through.
	if got := s.MapValue(42.5); got != 42.5 {
		t.Errorf("expected MapValue(42.5) = 42.5, got %v", got)
	}

	// ValueMapper interface compliance.
	var vm scale.ValueMapper = s
	if got := vm.MapValue(7.0); got != 7.0 {
		t.Errorf("ValueMapper.MapValue(7.0) = %v, want 7.0", got)
	}
}
