package scale_test

import (
	"context"
	"testing"

	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/scale"
)

func TestShapeScale(t *testing.T) {
	t.Parallel()

	s := scale.NewShape()
	if s.String() != "shape" {
		t.Errorf("expected 'shape', got %q", s.String())
	}

	// Train and map
	eng := memory.NewEngine(context.Background())
	mockCol := eng.NewStringColumn("cat_col", []string{"X", "Y", "Z"})

	if err := s.Train(mockCol); err != nil {
		t.Fatalf("unexpected error training: %v", err)
	}

	tests := []struct {
		val      string
		wantName string
	}{
		{"X", canvas.ShapeCircle},
		{"Y", canvas.ShapeSquare},
		{"Z", canvas.ShapeTriangle},
	}

	for _, tt := range tests {
		gotName := s.ShapeName(tt.val)
		if gotName != tt.wantName {
			t.Errorf("ShapeName(%q) = %q, want %q", tt.val, gotName, tt.wantName)
		}
	}

	// Manual
	sManual := scale.NewShapeManual(map[string]string{
		"X": canvas.ShapeStar,
		"Y": canvas.ShapeDiamond,
	})
	if name := sManual.ShapeName("X"); name != canvas.ShapeStar {
		t.Errorf("expected %q, got %q", canvas.ShapeStar, name)
	}

	if name := sManual.ShapeName("Y"); name != canvas.ShapeDiamond {
		t.Errorf("expected %q, got %q", canvas.ShapeDiamond, name)
	}
}

func TestShapeScale_WrapAround(t *testing.T) {
	t.Parallel()

	// Train with more categories than available shapes (10).
	// The 11th category should wrap back to the first shape.
	eng := memory.NewEngine(context.Background())

	labels := make([]string, 12)
	for i := range labels {
		labels[i] = string(rune('A' + i))
	}

	s := scale.NewShape()
	if err := s.Train(eng.NewStringColumn("many", labels)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	shapes := canvas.Shapes()

	// The 11th category (index 10, label "K") should wrap to shapes[10 % 10] = shapes[0].
	// But categories are sorted alphabetically: A, B, C, D, E, F, G, H, I, J, K, L
	// So index 10 is "K" and maps to shapes[10 % 10] = shapes[0] = "circle".
	got := s.ShapeName("K")
	want := shapes[10%len(shapes)]

	if got != want {
		t.Errorf("ShapeName(\"K\") = %q, want %q (wrap-around)", got, want)
	}
}

func TestShapeScale_EmptyStrings(t *testing.T) {
	t.Parallel()

	// Empty strings should be skipped during training.
	eng := memory.NewEngine(context.Background())
	col := eng.NewStringColumn("with_empty", []string{"A", "", "B", "", "C"})

	s := scale.NewShape()
	if err := s.Train(col); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have exactly 3 categories: A, B, C
	cats := s.Categories()
	if len(cats) != 3 {
		t.Fatalf("expected 3 categories, got %d: %v", len(cats), cats)
	}
}

func TestShapeScale_UntrainedCategory(t *testing.T) {
	t.Parallel()

	s := scale.NewShape()

	// Untrained scale, unknown label → fallback to circle.
	got := s.ShapeName("unknown")
	if got != canvas.ShapeCircle {
		t.Errorf("untrained ShapeName(\"unknown\") = %q, want %q", got, canvas.ShapeCircle)
	}

	// Untrained scale, label that IS a known shape name → returns itself.
	got = s.ShapeName(canvas.ShapeTriangle)
	if got != canvas.ShapeTriangle {
		t.Errorf("untrained ShapeName(%q) = %q, want itself", canvas.ShapeTriangle, got)
	}
}

func TestShapeScale_ManualNil(t *testing.T) {
	t.Parallel()

	// NewShapeManual(nil) should not panic.
	s := scale.NewShapeManual(nil)

	// Should fall back to default behavior.
	got := s.ShapeName("anything")
	if got != canvas.ShapeCircle {
		t.Errorf("nil manual ShapeName = %q, want %q", got, canvas.ShapeCircle)
	}
}

func TestShapeScale_ManualAliasingSafety(t *testing.T) {
	t.Parallel()

	// The constructor should defensively copy the map.
	m := map[string]string{"A": canvas.ShapeStar}
	s := scale.NewShapeManual(m)

	// Mutate the original map after construction.
	m["A"] = canvas.ShapeHexagon

	got := s.ShapeName("A")
	if got != canvas.ShapeStar {
		t.Errorf("ShapeName after map mutation = %q, want %q (aliasing bug)", got, canvas.ShapeStar)
	}
}

func TestShapeScale_Duplicates(t *testing.T) {
	t.Parallel()

	// Training with duplicate values should deduplicate.
	eng := memory.NewEngine(context.Background())
	col := eng.NewStringColumn("dupes", []string{"A", "B", "A", "B", "A"})

	s := scale.NewShape()
	if err := s.Train(col); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cats := s.Categories()
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories after dedup, got %d: %v", len(cats), cats)
	}
}
