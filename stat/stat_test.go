package stat_test

import (
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/stat"
)

func TestLookup_Identity(t *testing.T) {
	s := stat.Lookup("identity")
	if s.Name() != "identity" {
		t.Errorf("expected identity, got %s", s.Name())
	}
}

func TestLookup_Unknown_FallsBackToIdentity(t *testing.T) {
	s := stat.Lookup("nonexistent")
	if s.Name() != "identity" {
		t.Errorf("expected identity fallback, got %s", s.Name())
	}
}

func TestBinStat(t *testing.T) {
	ds, err := dataset.NewDataFrame(map[string][]float64{
		"x": {1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	})
	if err != nil {
		t.Fatal(err)
	}

	s := stat.Lookup("bin")
	result, err := s.Compute(ds, map[string]string{"x": "x"})
	if err != nil {
		t.Fatal(err)
	}

	// Bin stat produces "x" (centers) and "count" columns.
	cols := result.Columns()
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
	countCol, _ := result.Column("count")
	vals, _ := dataset.CollectFloat64s(countCol)
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	if sum != 10 {
		t.Errorf("bin stat total count: expected 10, got %v", sum)
	}
}

func TestBinStat_MissingX(t *testing.T) {
	ds, _ := dataset.NewDataFrame(map[string][]float64{"y": {1}})
	s := stat.Lookup("bin")
	_, err := s.Compute(ds, map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing x aesthetic")
	}
}

func TestCountStat(t *testing.T) {
	ds, _ := dataset.NewDataFrame(map[string][]float64{
		"x": {1, 2, 2, 3, 3, 3},
	})

	s := stat.Lookup("count")
	result, err := s.Compute(ds, map[string]string{"x": "x"})
	if err != nil {
		t.Fatal(err)
	}

	countCol, _ := result.Column("count")
	counts, _ := dataset.CollectFloat64s(countCol)
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
	ds, _ := dataset.NewDataFrame(map[string][]float64{"x": xs})

	s := stat.Lookup("density")
	result, err := s.Compute(ds, map[string]string{"x": "x"})
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() == 0 {
		t.Fatal("density stat produced empty dataset")
	}

	// Density values should be non-negative.
	densityCol, _ := result.Column("density")
	vals, _ := dataset.CollectFloat64s(densityCol)
	for i, v := range vals {
		if v < 0 {
			t.Errorf("density[%d] = %v < 0", i, v)
			break
		}
	}
}

func TestSmoothStat(t *testing.T) {
	ds, _ := dataset.NewDataFrame(map[string][]float64{
		"x": {1, 2, 3, 4, 5},
		"y": {2, 4, 6, 8, 10},
	})

	s := stat.Lookup("smooth")
	result, err := s.Compute(ds, map[string]string{"x": "x", "y": "y"})
	if err != nil {
		t.Fatal(err)
	}

	// Smooth should produce 80 points.
	if result.NumRows() < 2 {
		t.Errorf("smooth stat produced too few points: %d", result.NumRows())
	}

	// For perfect linear data y=2x, smooth should be close.
	yCol, _ := result.Column("y")
	xCol, _ := result.Column("x")
	xVals, _ := dataset.CollectFloat64s(xCol)
	yVals, _ := dataset.CollectFloat64s(yCol)

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
	ds, _ := dataset.NewDataFrame(map[string][]float64{
		"x": {1, 1, 2, 2},
		"y": {10, 20, 30, 40},
	})

	s := stat.Lookup("summary")
	result, err := s.Compute(ds, map[string]string{"x": "x", "y": "y"})
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 2 {
		t.Fatalf("summary: expected 2 groups, got %d", result.NumRows())
	}

	yCol, _ := result.Column("y")
	means, _ := dataset.CollectFloat64s(yCol)
	// mean(10,20)=15, mean(30,40)=35
	if means[0] != 15 || means[1] != 35 {
		t.Errorf("summary: expected [15,35], got %v", means)
	}
}

func TestIdentityStat(t *testing.T) {
	ds, _ := dataset.NewDataFrame(map[string][]float64{"x": {1, 2, 3}})
	s := stat.Lookup("identity")
	result, err := s.Compute(ds, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != ds {
		t.Error("identity stat should return the same dataset")
	}
}
