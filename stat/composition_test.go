package stat_test

import (
	"context"
	"math"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/stat"
)

// collectAndFloat64 collects a lazy Dataset and extracts a float64 column.
// If the column is int64 (e.g. from count aggregation), it auto-casts.
func collectAndFloat64(ctx context.Context, t *testing.T, ds dataset.Dataset, col string) []float64 {
	t.Helper()

	collected, err := ds.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	vals, err := collected.Float64(col, dataset.Clean)
	if err == nil {
		return vals
	}

	// Try int64 → float64 cast.
	ivals, ierr := collected.Int64(col)
	if ierr != nil {
		t.Fatalf("Float64(%q): %v; Int64(%q): %v", col, err, col, ierr)
	}

	out := make([]float64, len(ivals))
	for i, v := range ivals {
		out[i] = float64(v)
	}

	return out
}

func TestNormalizeY_AfterBinX(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Pipeline: BinX → NormalizeY (proportions should sum to 1.0)
	pipeline := []stat.Transform{stat.BinX(), stat.NormalizeY()}

	outDS, outMapping, err := stat.RunPipeline(ctx, pipeline, ds, map[string]string{"x": "x"})
	if err != nil {
		t.Fatal(err)
	}

	counts := collectAndFloat64(ctx, t, outDS, "count")

	sum := 0.0
	for _, c := range counts {
		sum += c
	}

	if math.Abs(sum-1.0) > 1e-10 {
		t.Errorf("normalized proportions sum to %v, want 1.0", sum)
	}

	if outMapping["y"] != "count" {
		t.Errorf("expected y→count mapping, got y→%q", outMapping["y"])
	}
}

func TestNormalizeY_WithTotal100(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
	)
	if err != nil {
		t.Fatal(err)
	}

	pipeline := []stat.Transform{stat.BinX(), stat.NormalizeY(stat.WithTotal(100))}

	outDS, _, err := stat.RunPipeline(ctx, pipeline, ds, map[string]string{"x": "x"})
	if err != nil {
		t.Fatal(err)
	}

	counts := collectAndFloat64(ctx, t, outDS, "count")

	sum := 0.0
	for _, c := range counts {
		sum += c
	}

	if math.Abs(sum-100.0) > 1e-10 {
		t.Errorf("percentage proportions sum to %v, want 100.0", sum)
	}
}

func TestFilterY(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng.NewFloat64Column("y", []float64{10, 20, 30, 40, 50}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Keep only rows where y > 25.
	tf := stat.FilterY(dataset.Gt("y", 25.0))

	result, err := tf.Apply(ctx, stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x", "y": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	xs := collectAndFloat64(ctx, t, result.Data, "x")
	ys := collectAndFloat64(ctx, t, result.Data, "y")

	if len(xs) != 3 {
		t.Fatalf("expected 3 rows after filter, got %d", len(xs))
	}

	if xs[0] != 3 || xs[1] != 4 || xs[2] != 5 {
		t.Errorf("expected x=[3,4,5], got %v", xs)
	}

	if ys[0] != 30 || ys[1] != 40 || ys[2] != 50 {
		t.Errorf("expected y=[30,40,50], got %v", ys)
	}
}

func TestFilterX(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng.NewFloat64Column("y", []float64{10, 20, 30, 40, 50}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Keep only rows where x <= 2.
	tf := stat.FilterX(dataset.Le("x", 2.0))

	result, err := tf.Apply(ctx, stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x", "y": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	xs := collectAndFloat64(ctx, t, result.Data, "x")
	if len(xs) != 2 {
		t.Fatalf("expected 2 rows after filter, got %d", len(xs))
	}
}

func TestSortBy_Ascending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{3, 1, 4, 1, 5}),
		eng.NewFloat64Column("y", []float64{30, 10, 40, 11, 50}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.SortBy("x")

	result, err := tf.Apply(ctx, stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x", "y": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	xs := collectAndFloat64(ctx, t, result.Data, "x")
	for i := 1; i < len(xs); i++ {
		if xs[i] < xs[i-1] {
			t.Errorf("not sorted ascending at index %d: %v", i, xs)

			break
		}
	}
}

func TestSortBy_Descending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{3, 1, 4, 1, 5}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.SortBy("x", stat.Desc())

	result, err := tf.Apply(ctx, stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}

	xs := collectAndFloat64(ctx, t, result.Data, "x")
	for i := 1; i < len(xs); i++ {
		if xs[i] > xs[i-1] {
			t.Errorf("not sorted descending at index %d: %v", i, xs)

			break
		}
	}
}

func TestReverseRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.ReverseRows()

	result, err := tf.Apply(ctx, stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}

	xs := collectAndFloat64(ctx, t, result.Data, "x")
	want := []float64{5, 4, 3, 2, 1}

	for i, v := range xs {
		if v != want[i] {
			t.Errorf("reversed[%d] = %v, want %v", i, v, want[i])
		}
	}
}

func TestTopN_LargestFirst(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.TopN(3, "x")

	result, err := tf.Apply(ctx, stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}

	xs := collectAndFloat64(ctx, t, result.Data, "x")
	if len(xs) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(xs))
	}

	// Should be the 3 largest values.
	for _, v := range xs {
		if v < 8 {
			t.Errorf("topN(3, desc) should contain values >= 8, got %v in %v", v, xs)
		}
	}
}

func TestTopN_SmallestFirst(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.TopN(3, "x", stat.Ascending())

	result, err := tf.Apply(ctx, stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}

	xs := collectAndFloat64(ctx, t, result.Data, "x")
	if len(xs) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(xs))
	}

	for _, v := range xs {
		if v > 3 {
			t.Errorf("topN(3, asc) should contain values <= 3, got %v in %v", v, xs)
		}
	}
}

func TestStackY(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 1, 2, 2}),
		eng.NewFloat64Column("y", []float64{10, 20, 30, 40}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.StackY()

	result, err := tf.Apply(ctx, stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x", "y": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	ys := collectAndFloat64(ctx, t, result.Data, "y")
	ymins := collectAndFloat64(ctx, t, result.Data, "ymin")

	// Global CumSum: [10, 20, 30, 40] → cumsum [10, 30, 60, 100]
	// ymin = cumsum - y: [0, 10, 30, 60]
	wantY := []float64{10, 30, 60, 100}
	wantYmin := []float64{0, 10, 30, 60}

	for i := range ys {
		if ys[i] != wantY[i] {
			t.Errorf("stacked y[%d] = %v, want %v", i, ys[i], wantY[i])
		}

		if ymins[i] != wantYmin[i] {
			t.Errorf("stacked ymin[%d] = %v, want %v", i, ymins[i], wantYmin[i])
		}
	}
}

func TestGroupX_Mean(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 1, 2, 2, 3}),
		eng.NewFloat64Column("y", []float64{10, 20, 30, 40, 50}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.GroupX("mean")

	result, err := tf.Apply(ctx, stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x", "y": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	xs := collectAndFloat64(ctx, t, result.Data, "x")
	ys := collectAndFloat64(ctx, t, result.Data, "y")

	if len(xs) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(xs))
	}

	// x=1: mean(10,20) = 15
	// x=2: mean(30,40) = 35
	// x=3: mean(50) = 50
	wantY := []float64{15, 35, 50}

	for i := range ys {
		if math.Abs(ys[i]-wantY[i]) > 1e-10 {
			t.Errorf("groupX mean y[%d] = %v, want %v", i, ys[i], wantY[i])
		}
	}
}

func TestGroupX_Sum(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 1, 2, 2}),
		eng.NewFloat64Column("y", []float64{10, 20, 30, 40}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.GroupX("sum")

	result, err := tf.Apply(ctx, stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x", "y": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	ys := collectAndFloat64(ctx, t, result.Data, "y")

	wantY := []float64{30, 70}

	for i := range ys {
		if math.Abs(ys[i]-wantY[i]) > 1e-10 {
			t.Errorf("groupX sum y[%d] = %v, want %v", i, ys[i], wantY[i])
		}
	}
}

func TestGroupX_Count(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 1, 2, 2, 2}),
		eng.NewFloat64Column("y", []float64{10, 20, 30, 40, 50}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tf := stat.GroupX("count")

	result, err := tf.Apply(ctx, stat.TransformInput{
		Data:    ds,
		Mapping: map[string]string{"x": "x", "y": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Count returns int64; use Float64 accessor which auto-casts.
	ys := collectAndFloat64(ctx, t, result.Data, "y")

	wantY := []float64{2, 3} // 2 rows for x=1, 3 rows for x=2

	for i := range ys {
		if ys[i] != wantY[i] {
			t.Errorf("groupX count y[%d] = %v, want %v", i, ys[i], wantY[i])
		}
	}
}

func TestComposePipeline_BinX_NormalizeY_TopN(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 2, 3, 3, 3, 4, 4, 4, 4}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Full composition: bin → normalize → top 2 by proportion
	pipeline := []stat.Transform{
		stat.BinX(stat.WithBins(4)),
		stat.NormalizeY(),
		stat.TopN(2, "count"),
	}

	outDS, _, err := stat.RunPipeline(ctx, pipeline, ds, map[string]string{"x": "x"})
	if err != nil {
		t.Fatal(err)
	}

	collected, err := outDS.Collect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if collected.NumRows() != 2 {
		t.Errorf("expected 2 rows after topN(2), got %d", collected.NumRows())
	}

	// Proportions should sum to less than 1 (we took top 2 of 4 bins).
	counts := collectAndFloat64(ctx, t, outDS, "count")

	sum := 0.0
	for _, c := range counts {
		sum += c
	}

	if sum >= 1.0 {
		t.Errorf("top-2 proportions should sum to < 1.0, got %v", sum)
	}
}
