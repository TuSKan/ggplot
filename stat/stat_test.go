package stat_test

import (
	"context"
	"math"
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

// --- BinY ---

func TestBinY(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("y", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.BinY()

	if tf.Name() != "biny" {
		t.Errorf("expected biny, got %s", tf.Name())
	}

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"y": "y"},
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
		t.Errorf("BinY total count: expected 10, got %v", sum)
	}

	// Output should have "y" column (not "x").
	yVals := getFloat64Values(t, result.Data, "y")
	if len(yVals) == 0 {
		t.Error("BinY: expected non-empty y column")
	}

	// Mapping should map x → count.
	if result.Mapping["x"] != "count" {
		t.Errorf("BinY: expected mapping x→count, got %q", result.Mapping["x"])
	}
}

func TestBinY_MissingY(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng, eng.NewFloat64Column("x", []float64{1}))

	tf := stat.BinY()

	_, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{},
	})
	if err == nil {
		t.Error("expected error for missing y aesthetic")
	}
}

// --- DensityY ---

func TestDensityY(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("y", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.DensityY(stat.WithDensityPoints(32))

	if tf.Name() != "densityy" {
		t.Errorf("expected densityy, got %s", tf.Name())
	}

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"y": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should produce "y" (grid) and "density" columns.
	yVals := getFloat64Values(t, result.Data, "y")
	dVals := getFloat64Values(t, result.Data, "density")

	if len(yVals) != 32 {
		t.Errorf("DensityY: expected 32 grid points, got %d", len(yVals))
	}

	if len(dVals) != 32 {
		t.Errorf("DensityY: expected 32 density values, got %d", len(dVals))
	}

	// Density should be non-negative.
	for i, d := range dVals {
		if d < 0 {
			t.Errorf("DensityY: density[%d] = %f < 0", i, d)
		}
	}

	// Mapping should map x → density.
	if result.Mapping["x"] != "density" {
		t.Errorf("DensityY: expected mapping x→density, got %q", result.Mapping["x"])
	}
}

// --- WithCumulative ---

func TestBinX_Cumulative(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.BinX(stat.WithCumulative(1))

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}

	vals := getFloat64Values(t, result.Data, "count")

	// Cumulative: each value should be >= previous.
	for i := 1; i < len(vals); i++ {
		if vals[i] < vals[i-1] {
			t.Errorf("cumulative histogram not monotonically increasing at index %d: %v < %v",
				i, vals[i], vals[i-1])
		}
	}

	// Last value should equal total count.
	if vals[len(vals)-1] != 10 {
		t.Errorf("cumulative last value: expected 10, got %v", vals[len(vals)-1])
	}
}

func TestBinX_CumulativeReverse(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.BinX(stat.WithCumulative(-1))

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}

	vals := getFloat64Values(t, result.Data, "count")

	// Reverse cumulative: each value should be <= previous.
	for i := 1; i < len(vals); i++ {
		if vals[i] > vals[i-1] {
			t.Errorf("reverse cumulative not monotonically decreasing at index %d: %v > %v",
				i, vals[i], vals[i-1])
		}
	}

	// First value should equal total count.
	if vals[0] != 10 {
		t.Errorf("reverse cumulative first value: expected 10, got %v", vals[0])
	}
}

// --- WithSE ---

func TestSmoothXY_WithSE_Linear(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	// y = 2x + noise
	xs := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	ys := []float64{2.1, 3.9, 6.1, 7.9, 10.1, 11.9, 14.1, 15.9, 18.1, 19.9}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.SmoothXY(stat.WithMethod("linear"), stat.WithSE(true), stat.WithNOut(5))

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x", "y": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should produce x, y, ymin, ymax.
	xOut := getFloat64Values(t, result.Data, "x")
	yOut := getFloat64Values(t, result.Data, "y")
	yminOut := getFloat64Values(t, result.Data, "ymin")
	ymaxOut := getFloat64Values(t, result.Data, "ymax")

	if len(xOut) != 5 {
		t.Fatalf("expected 5 output points, got %d", len(xOut))
	}

	for i := range xOut {
		if yminOut[i] > yOut[i] {
			t.Errorf("ymin[%d]=%f > y[%d]=%f", i, yminOut[i], i, yOut[i])
		}

		if ymaxOut[i] < yOut[i] {
			t.Errorf("ymax[%d]=%f < y[%d]=%f", i, ymaxOut[i], i, yOut[i])
		}
	}
}

func TestSmoothXY_WithSE_Loess(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	xs := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	ys := []float64{1, 4, 9, 16, 25, 36, 49, 64, 81, 100}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.SmoothXY(stat.WithSE(true), stat.WithNOut(5))

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x", "y": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	yOut := getFloat64Values(t, result.Data, "y")
	yminOut := getFloat64Values(t, result.Data, "ymin")
	ymaxOut := getFloat64Values(t, result.Data, "ymax")

	for i := range yOut {
		if yminOut[i] > yOut[i] {
			t.Errorf("ymin[%d]=%f > y[%d]=%f", i, yminOut[i], i, yOut[i])
		}

		if ymaxOut[i] < yOut[i] {
			t.Errorf("ymax[%d]=%f < y[%d]=%f", i, ymaxOut[i], i, yOut[i])
		}
	}
}

// --- Percentile via GroupX ---

func TestGroupX_Percentile(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	// Two groups: a=[1,2,3,4,5], b=[10,20,30,40,50]
	ds, err := dataset.NewDataset(eng,
		eng.NewStringColumn("cat", []string{"a", "a", "a", "a", "a", "b", "b", "b", "b", "b"}),
		eng.NewFloat64Column("val", []float64{1, 2, 3, 4, 5, 10, 20, 30, 40, 50}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.GroupX("p50")

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "cat", "y": "val"},
	})
	if err != nil {
		t.Fatal(err)
	}

	vals := getFloat64Values(t, result.Data, "val")

	if len(vals) != 2 { //nolint:mnd // Two groups expected.
		t.Fatalf("expected 2 groups, got %d", len(vals))
	}

	// p50 of [1,2,3,4,5] = 3, p50 of [10,20,30,40,50] = 30
	if math.Abs(vals[0]-3) > 0.01 {
		t.Errorf("p50 of group a: expected 3, got %v", vals[0])
	}

	if math.Abs(vals[1]-30) > 0.01 {
		t.Errorf("p50 of group b: expected 30, got %v", vals[1])
	}
}

// --- Proportion ---

func TestGroupX_Proportion(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	// 3 a's and 7 b's
	ds, err := dataset.NewDataset(eng,
		eng.NewStringColumn("cat", []string{"a", "a", "a", "b", "b", "b", "b", "b", "b", "b"}),
		eng.NewFloat64Column("val", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.GroupX("proportion")

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "cat", "y": "val"},
	})
	if err != nil {
		t.Fatal(err)
	}

	vals := getFloat64Values(t, result.Data, "val")

	if len(vals) != 2 { //nolint:mnd // Two groups expected.
		t.Fatalf("expected 2 groups, got %d", len(vals))
	}

	// a = 3/10 = 0.3, b = 7/10 = 0.7
	sum := 0.0
	for _, v := range vals {
		sum += v
	}

	if math.Abs(sum-1.0) > 0.01 {
		t.Errorf("proportions should sum to 1.0, got %v", sum)
	}
}

// --- Group (dual-axis) ---

func TestGroupDual(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewStringColumn("color", []string{"a", "a", "a", "b", "b", "b"}),
		eng.NewFloat64Column("x", []float64{1, 2, 3, 10, 20, 30}),
		eng.NewFloat64Column("y", []float64{10, 20, 30, 100, 200, 300}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.Group("mean", "sum")

	result, err := tf.Apply(context.Background(), stat.TransformInput{
		Data: ds,
		Mapping: map[string]string{
			"x":     "x",
			"y":     "y",
			"color": "color",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	xVals := getFloat64Values(t, result.Data, "x")
	yVals := getFloat64Values(t, result.Data, "y")

	if len(xVals) != 2 { //nolint:mnd // Two groups expected.
		t.Fatalf("expected 2 groups, got %d", len(xVals))
	}

	// mean(x) for a = 2, for b = 20
	if math.Abs(xVals[0]-2) > 0.01 {
		t.Errorf("mean(x) group a: expected 2, got %v", xVals[0])
	}

	if math.Abs(xVals[1]-20) > 0.01 {
		t.Errorf("mean(x) group b: expected 20, got %v", xVals[1])
	}

	// sum(y) for a = 60, for b = 600
	if math.Abs(yVals[0]-60) > 0.01 {
		t.Errorf("sum(y) group a: expected 60, got %v", yVals[0])
	}

	if math.Abs(yVals[1]-600) > 0.01 {
		t.Errorf("sum(y) group b: expected 600, got %v", yVals[1])
	}
}

func TestGroupDual_MissingGroupCol(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1}),
		eng.NewFloat64Column("y", []float64{2}),
	)

	tf := stat.Group("mean", "sum")

	_, err := tf.Apply(context.Background(), stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x", "y": "y"},
	})
	if err == nil {
		t.Error("expected error for missing group-by column")
	}
}

// --- Percentile engine test (memory) ---

func TestPercentile_Memory(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewStringColumn("cat", []string{"a", "a", "a", "a", "a"}),
		eng.NewFloat64Column("val", []float64{1, 2, 3, 4, 5}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// GroupBy + Summarize with Percentile(0.5) should give median.
	result := ds.GroupBy("cat").Summarize(dataset.Percentile("p50", "val", 0.5))

	collected, err := result.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	col, err := collected.Column("p50")
	if err != nil {
		t.Fatal(err)
	}

	fc, ok := col.(dataset.Column[float64])
	if !ok {
		t.Fatalf("expected float64 column, got %T", col)
	}

	vals := fc.Values()
	if len(vals) != 1 || math.Abs(vals[0]-3) > 0.01 {
		t.Errorf("p50 of [1,2,3,4,5]: expected 3, got %v", vals)
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
