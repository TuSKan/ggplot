package stat_test

import (
	"context"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/stat"
)

func TestIdentityTransform(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng, eng.NewFloat64Column("x", []float64{1, 2, 3}))

	tf := stat.IdentityTransform()

	if tf.Name() != "identity" {
		t.Errorf("expected identity, got %s", tf.Name())
	}

	result, err := tf.Apply(context.Background(), stat.TransformInput{Data: ds, Mapping: nil})
	if err != nil {
		t.Fatal(err)
	}

	if result.Data.Table() != ds.Table() {
		t.Error("identity transform should return the same dataset")
	}
}

func TestBinX(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.BinX()

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Total counts should equal input length.
	vals := getFloat64Values(t, result.Data, "count")

	sum := 0.0
	for _, v := range vals {
		sum += v
	}

	if sum != 10 {
		t.Errorf("bin stat total count: expected 10, got %v", sum)
	}
}

func TestBinX_MissingX(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng, eng.NewFloat64Column("y", []float64{1}))

	tf := stat.BinX()

	_, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{},
	})
	if err == nil {
		t.Fatal("expected error for missing x aesthetic")
	}
}

func TestCount(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 2, 3, 3, 3}),
	)

	tf := stat.Count()

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}

	counts := getFloat64Values(t, result.Data, "count")
	// x=1→1, x=2→2, x=3→3
	if len(counts) != 3 {
		t.Fatalf("expected 3 unique values, got %d", len(counts))
	}

	if counts[0] != 1 || counts[1] != 2 || counts[2] != 3 {
		t.Errorf("count: got %v", counts)
	}
}

func TestDensityX(t *testing.T) {
	t.Parallel()

	xs := make([]float64, 100)
	for i := range xs {
		xs[i] = float64(i)
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng, eng.NewFloat64Column("x", xs))

	tf := stat.DensityX()

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Collect lazy result before checking.
	result.Data, err = result.Data.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.Data.NumRows() == 0 {
		t.Fatal("density produced empty dataset")
	}

	// Density values should be non-negative.
	vals := getFloat64Values(t, result.Data, "density")
	for i, v := range vals {
		if v < 0 {
			t.Errorf("density[%d] = %v < 0", i, v)
			break
		}
	}
}

func TestSmoothXY(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng.NewFloat64Column("y", []float64{2, 4, 6, 8, 10}),
	)

	tf := stat.SmoothXY()

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x", "y": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Collect lazy result before checking.
	result.Data, err = result.Data.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.Data.NumRows() < 2 {
		t.Errorf("smooth produced too few points: %d", result.Data.NumRows())
	}

	// For perfect linear data y=2x, smooth should be close.
	xVals := getFloat64Values(t, result.Data, "x")
	yVals := getFloat64Values(t, result.Data, "y")

	for i := range xVals {
		expected := 2 * xVals[i]

		diff := yVals[i] - expected
		if diff > 0.5 || diff < -0.5 {
			t.Errorf("smooth[%d]: x=%v y=%v expected ~%v", i, xVals[i], yVals[i], expected)
			break
		}
	}
}

func TestSummaryXY(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 1, 2, 2}),
		eng.NewFloat64Column("y", []float64{10, 20, 30, 40}),
	)

	tf := stat.SummaryXY()

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x", "y": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Collect lazy result before checking.
	result.Data, err = result.Data.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.Data.NumRows() != 2 {
		t.Fatalf("summary: expected 2 groups, got %d", result.Data.NumRows())
	}

	means := getFloat64Values(t, result.Data, "y")
	// mean(10,20)=15, mean(30,40)=35
	if means[0] != 15 || means[1] != 35 {
		t.Errorf("summary: expected [15,35], got %v", means)
	}
}

func TestOutputSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		tf     stat.Transform
		expect []string
	}{
		{"identity", stat.IdentityTransform(), nil},
		{"binX", stat.BinX(), []string{"count", "x"}},
		{"count", stat.Count(), []string{"count", "x"}},
		{"densityX", stat.DensityX(), []string{"density", "x"}},
		{"smoothXY", stat.SmoothXY(), []string{"x", "y"}},
		{"summaryXY", stat.SummaryXY(), []string{"x", "y"}},
		{"boxplotY", stat.BoxplotY(), []string{"lower", "middle", "notch_lower", "notch_upper", "q1", "q3", "upper", "x"}},
	}
	for _, tc := range tests {
		schema := tc.tf.OutputSchema()
		if len(schema) != len(tc.expect) {
			t.Errorf("OutputSchema(%q) = %v, want %v", tc.name, schema, tc.expect)
		}
	}
}

func TestRunPipeline_Empty(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng, eng.NewFloat64Column("x", []float64{1, 2, 3}))
	mapping := map[string]string{"x": "x"}

	outDS, outMapping, err := stat.RunPipeline(context.Background(), nil, ds, mapping)
	if err != nil {
		t.Fatal(err)
	}

	if outDS.Table() != ds.Table() {
		t.Error("empty pipeline should return same dataset")
	}

	if outMapping["x"] != "x" {
		t.Error("empty pipeline should preserve mapping")
	}
}

func TestRunPipeline_Chain(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 2, 3, 3, 3, 4, 4, 4, 4}),
	)

	// Single-element pipeline: BinX
	pipeline := []stat.Transform{stat.BinX(stat.WithBins(4))}

	outDS, outMapping, err := stat.RunPipeline(context.Background(), pipeline, ds, map[string]string{"x": "x"})
	if err != nil {
		t.Fatal(err)
	}

	// Collect lazy result before checking.
	outDS, err = outDS.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if outDS.NumRows() != 4 {
		t.Errorf("expected 4 bins, got %d rows", outDS.NumRows())
	}

	// Mapping should be rewritten: y → count
	if outMapping["y"] != "count" {
		t.Errorf("expected y→count mapping, got y→%q", outMapping["y"])
	}
}

func getFloat64Values(t *testing.T, ds dataset.Dataset, colName string) []float64 {
	t.Helper()

	// Collect lazy pipeline before reading columns.
	ds, err := ds.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect for column %s: %v", colName, err)
	}

	col, err := ds.Column(colName)
	if err != nil {
		t.Fatalf("missing column %s: %v", colName, err)
	}

	// Handle both float64 and int64 columns (e.g. AggCount produces int64).
	if floatCol, ok := col.(dataset.Column[float64]); ok {
		return floatCol.Values()
	}

	if intCol, ok := col.(dataset.Column[int64]); ok {
		vals := intCol.Values()

		out := make([]float64, len(vals))
		for i, v := range vals {
			out[i] = float64(v)
		}

		return out
	}

	t.Fatalf("column %s: unsupported type %T", colName, col)

	return nil
}
