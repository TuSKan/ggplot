package stat_test

import (
	"context"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/stat"
)

func TestLookup_Identity(t *testing.T) {
	s, err := stat.Lookup(stat.Identity)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name() != stat.Identity {
		t.Errorf("expected identity, got %s", s.Name())
	}
}

func TestLookup_Unknown_ReturnsError(t *testing.T) {
	_, err := stat.Lookup("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown stat name")
	}
}

func TestBinStat(t *testing.T) {
	ds, err := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
	)
	if err != nil {
		t.Fatal(err)
	}

	s, err := stat.Lookup(stat.Bin)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Compute(context.Background(), ds, map[string]string{"x": "x"}, stat.Options{})
	if err != nil {
		t.Fatal(err)
	}

	// Bin stat produces "x" (centers) and "count" columns.
	cols := []string{"x", "count"}
	hasX, hasCount := false, false
	for _, c := range cols {
		if c == "x" {
			hasX = true
		}
		if c == "count" {
			hasCount = true
		}
	}
	if !hasX || !hasCount {
		t.Errorf("bin stat should produce x and count columns, got %v", cols)
	}

	// Total counts should equal input length.
	vals := getFloat64Values(t, result, "count")
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	if sum != 10 {
		t.Errorf("bin stat total count: expected 10, got %v", sum)
	}
}

func TestBinStat_MissingX(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("y", []float64{1}))
	s, err := stat.Lookup(stat.Bin)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Compute(context.Background(), ds, map[string]string{}, stat.Options{})
	if err == nil {
		t.Fatal("expected error for missing x aesthetic")
	}
}

func TestCountStat(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 2, 3, 3, 3}),
	)

	s, err := stat.Lookup(stat.Count)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Compute(context.Background(), ds, map[string]string{"x": "x"}, stat.Options{})
	if err != nil {
		t.Fatal(err)
	}

	counts := getFloat64Values(t, result, "count")
	// x=1→1, x=2→2, x=3→3
	if len(counts) != 3 {
		t.Fatalf("expected 3 unique values, got %d", len(counts))
	}
	if counts[0] != 1 || counts[1] != 2 || counts[2] != 3 {
		t.Errorf("count stat: got %v", counts)
	}
}

func TestDensityStat(t *testing.T) {
	xs := make([]float64, 100)
	for i := range xs {
		xs[i] = float64(i)
	}
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", xs))

	s, err := stat.Lookup(stat.Density)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Compute(context.Background(), ds, map[string]string{"x": "x"}, stat.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() == 0 {
		t.Fatal("density stat produced empty dataset")
	}

	// Density values should be non-negative.

	vals := getFloat64Values(t, result, "density")
	for i, v := range vals {
		if v < 0 {
			t.Errorf("density[%d] = %v < 0", i, v)
			break
		}
	}
}

func TestSmoothStat(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		memory.NewEngine().NewFloat64Column("y", []float64{2, 4, 6, 8, 10}),
	)

	s, err := stat.Lookup(stat.Smooth)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Compute(context.Background(), ds, map[string]string{"x": "x", "y": "y"}, stat.Options{})
	if err != nil {
		t.Fatal(err)
	}

	// Smooth should produce 80 points.
	if result.NumRows() < 2 {
		t.Errorf("smooth stat produced too few points: %d", result.NumRows())
	}

	// For perfect linear data y=2x, smooth should be close.

	xVals := getFloat64Values(t, result, "x")
	yVals := getFloat64Values(t, result, "y")

	for i := range xVals {
		expected := 2 * xVals[i]
		diff := yVals[i] - expected
		if diff > 0.5 || diff < -0.5 {
			t.Errorf("smooth[%d]: x=%v y=%v expected ~%v", i, xVals[i], yVals[i], expected)
			break
		}
	}
}

func TestSummaryStat(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(),
		memory.NewEngine().NewFloat64Column("x", []float64{1, 1, 2, 2}),
		memory.NewEngine().NewFloat64Column("y", []float64{10, 20, 30, 40}),
	)

	s, err := stat.Lookup(stat.Summary)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Compute(context.Background(), ds, map[string]string{"x": "x", "y": "y"}, stat.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 2 {
		t.Fatalf("summary: expected 2 groups, got %d", result.NumRows())
	}

	means := getFloat64Values(t, result, "y")
	// mean(10,20)=15, mean(30,40)=35
	if means[0] != 15 || means[1] != 35 {
		t.Errorf("summary: expected [15,35], got %v", means)
	}
}

func TestIdentityStat(t *testing.T) {
	ds, _ := dataset.NewDataset(memory.NewEngine(), memory.NewEngine().NewFloat64Column("x", []float64{1, 2, 3}))
	s, err := stat.Lookup(stat.Identity)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Compute(context.Background(), ds, nil, stat.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if result != ds {
		t.Error("identity stat should return the same dataset")
	}
}

func TestOutputSchema(t *testing.T) {
	tests := []struct {
		name   stat.Name
		expect []string
	}{
		{stat.Identity, nil},
		{stat.Bin, []string{"x", "count", "xmin", "xmax"}},
		{stat.Count, []string{"x", "count"}},
		{stat.Density, []string{"x", "density"}},
		{stat.Smooth, []string{"x", "y"}},
		{stat.Summary, []string{"x", "y"}},
		{stat.Boxplot, []string{"x", "lower", "q1", "middle", "q3", "upper"}},
	}
	for _, tc := range tests {
		s, err := stat.Lookup(tc.name)
		if err != nil {
			t.Fatalf("Lookup(%q): %v", tc.name, err)
		}
		schema := s.OutputSchema()
		if len(schema) != len(tc.expect) {
			t.Errorf("OutputSchema(%q) = %v, want %v", tc.name, schema, tc.expect)
		}
	}
}

func getFloat64Values(t *testing.T, ds dataset.Table, colName string) []float64 {
	t.Helper()
	col, err := ds.Column(colName)
	if err != nil {
		t.Fatalf("missing column %s: %v", colName, err)
	}
	floatCol, ok := col.(dataset.Column[float64])
	if !ok {
		t.Fatalf("column %s is not float64", colName)
	}
	return floatCol.Values()
}
