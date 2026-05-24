package scale_test

import (
	"context"
	"testing"

	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/scale"
)

func TestLinetypeScale(t *testing.T) {
	t.Parallel()

	s := scale.NewLinetype()
	if s.String() != "linetype" {
		t.Errorf("expected 'linetype', got %q", s.String())
	}

	// Train and map
	eng := memory.NewEngine(context.Background())
	mockCol := eng.NewStringColumn("type_col", []string{"A", "B", "C"})

	if err := s.Train(mockCol); err != nil {
		t.Fatalf("unexpected error training: %v", err)
	}

	tests := []struct {
		val      string
		wantName string
	}{
		{"A", "solid"},
		{"B", "dashed"},
		{"C", "dotted"},
	}

	for _, tt := range tests {
		gotName := s.LinetypeName(tt.val)
		if gotName != tt.wantName {
			t.Errorf("LinetypeName(%q) = %q, want %q", tt.val, gotName, tt.wantName)
		}
	}

	// Test manual mapping
	sManual := scale.NewLinetypeManual(map[string]string{
		"A": "dotted",
		"B": "longdash",
	})
	if name := sManual.LinetypeName("A"); name != "dotted" {
		t.Errorf("expected manual linetype 'dotted', got %q", name)
	}

	if name := sManual.LinetypeName("B"); name != "longdash" {
		t.Errorf("expected manual linetype 'longdash', got %q", name)
	}
}

func TestLinetypeScale_WrapAround(t *testing.T) {
	t.Parallel()

	// 6 default linetypes. Train with 8 categories — should wrap.
	eng := memory.NewEngine(context.Background())

	labels := make([]string, 8)
	for i := range labels {
		labels[i] = string(rune('A' + i))
	}

	s := scale.NewLinetype()
	if err := s.Train(eng.NewStringColumn("many", labels)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Category "G" is at sorted index 6 → 6 % 6 = 0 → "solid"
	got := s.LinetypeName("G")
	if got != "solid" {
		t.Errorf("LinetypeName(\"G\") = %q, want \"solid\" (wrap-around)", got)
	}
}

func TestLinetypeScale_DashPattern(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewStringColumn("lt", []string{"A", "B"})

	s := scale.NewLinetype()
	if err := s.Train(col); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "A" → index 0 → "solid" → nil dash pattern
	solidDash := s.DashPattern("A")
	if solidDash != nil {
		t.Errorf("DashPattern for solid = %v, want nil", solidDash)
	}

	// "B" → index 1 → "dashed" → {6, 6}
	dashedDash := s.DashPattern("B")
	if len(dashedDash) != 2 || dashedDash[0] != 6 || dashedDash[1] != 6 {
		t.Errorf("DashPattern for dashed = %v, want [6 6]", dashedDash)
	}
}

func TestLinetypeScale_UntrainedKnownName(t *testing.T) {
	t.Parallel()

	s := scale.NewLinetype()

	// Untrained, label is a known linetype name → returns itself.
	got := s.LinetypeName("dashed")
	if got != "dashed" {
		t.Errorf("untrained LinetypeName(\"dashed\") = %q, want \"dashed\"", got)
	}

	// Untrained, unknown label → fallback to solid.
	got = s.LinetypeName("unknown")
	if got != "solid" {
		t.Errorf("untrained LinetypeName(\"unknown\") = %q, want \"solid\"", got)
	}
}

func TestLinetypeScale_ManualNil(t *testing.T) {
	t.Parallel()

	s := scale.NewLinetypeManual(nil)

	got := s.LinetypeName("anything")
	if got != "solid" {
		t.Errorf("nil manual LinetypeName = %q, want \"solid\"", got)
	}
}

func TestLinetypeScale_ManualAliasingSafety(t *testing.T) {
	t.Parallel()

	m := map[string]string{"A": "dotted"}
	s := scale.NewLinetypeManual(m)

	// Mutate the original map after construction.
	m["A"] = "longdash"

	got := s.LinetypeName("A")
	if got != "dotted" {
		t.Errorf("LinetypeName after map mutation = %q, want \"dotted\" (aliasing bug)", got)
	}
}

func TestLinetypeScale_ManualUnknownPattern(t *testing.T) {
	t.Parallel()

	// Manual maps to a name not in linetypePatterns.
	s := scale.NewLinetypeManual(map[string]string{"A": "custom_unknown"})

	// DashPattern should return nil for unknown names (map lookup returns zero value).
	dash := s.DashPattern("A")
	if dash != nil {
		t.Errorf("DashPattern for unknown manual linetype = %v, want nil", dash)
	}
}
