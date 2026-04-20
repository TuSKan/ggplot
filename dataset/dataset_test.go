package dataset_test

import (
	"math"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
)

func testDS(t *testing.T) dataset.Dataset {
	t.Helper()
	ds, err := dataset.NewDataFrame(map[string][]float64{
		"x": {1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		"y": {10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	return ds
}

func TestNewDataFrame(t *testing.T) {
	ds := testDS(t)
	if ds.Len() != 10 {
		t.Errorf("expected 10 rows, got %d", ds.Len())
	}
	cols := ds.Columns()
	if len(cols) != 2 {
		t.Errorf("expected 2 columns, got %d", len(cols))
	}
}

func TestNewDataFrame_EmptyError(t *testing.T) {
	_, err := dataset.NewDataFrame(nil)
	if err == nil {
		t.Fatal("expected error for nil data")
	}
}

func TestNewDataFrame_MismatchError(t *testing.T) {
	_, err := dataset.NewDataFrame(map[string][]float64{
		"x": {1, 2, 3},
		"y": {1, 2},
	})
	if err == nil {
		t.Fatal("expected error for mismatched lengths")
	}
}

func TestNewMixedDataFrame(t *testing.T) {
	ds, err := dataset.NewMixedDataFrame(
		dataset.WithFloat64s("x", []float64{1, 2, 3}),
		dataset.WithStrings("label", []string{"a", "b", "c"}),
		dataset.WithInt64s("id", []int64{10, 20, 30}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if ds.Len() != 3 {
		t.Errorf("expected 3 rows, got %d", ds.Len())
	}
	cols := ds.Columns()
	if len(cols) != 3 {
		t.Errorf("expected 3 columns, got %d", len(cols))
	}
}

func TestColumnDType(t *testing.T) {
	ds, _ := dataset.NewMixedDataFrame(
		dataset.WithFloat64s("f", []float64{1}),
		dataset.WithInt64s("i", []int64{1}),
		dataset.WithStrings("s", []string{"a"}),
	)
	tests := []struct {
		name string
		want dataset.DType
	}{
		{"f", dataset.DTypeFloat64},
		{"i", dataset.DTypeInt64},
		{"s", dataset.DTypeString},
	}
	for _, tt := range tests {
		col, err := ds.Column(tt.name)
		if err != nil {
			t.Fatalf("Column(%q): %v", tt.name, err)
		}
		if col.DType() != tt.want {
			t.Errorf("Column(%q).DType() = %v, want %v", tt.name, col.DType(), tt.want)
		}
	}
}

func TestMinMax(t *testing.T) {
	ds := testDS(t)
	col, _ := ds.Column("x")

	mn, err := dataset.Min(col)
	if err != nil || mn != 1 {
		t.Errorf("Min = %v, err = %v", mn, err)
	}
	mx, err := dataset.Max(col)
	if err != nil || mx != 10 {
		t.Errorf("Max = %v, err = %v", mx, err)
	}
}

func TestCollectFloat64s(t *testing.T) {
	ds := testDS(t)
	col, _ := ds.Column("y")
	vals, err := dataset.CollectFloat64s(col)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 10 {
		t.Errorf("expected 10 values, got %d", len(vals))
	}
	if vals[0] != 10 {
		t.Errorf("first value = %v, want 10", vals[0])
	}
}

func TestFrame_Select(t *testing.T) {
	ds := testDS(t)
	f := dataset.From(ds).Select("x")
	if len(f.Columns()) != 1 || f.Columns()[0] != "x" {
		t.Errorf("Select: got %v", f.Columns())
	}
}

func TestFrame_Filter(t *testing.T) {
	ds := testDS(t)
	f := dataset.From(ds).Filter(dataset.Gt("x", 5))
	if f.Len() != 5 {
		t.Errorf("Filter: expected 5 rows, got %d", f.Len())
	}
}

func TestFrame_Head(t *testing.T) {
	ds := testDS(t)
	f := dataset.From(ds).Head(3)
	if f.Len() != 3 {
		t.Errorf("Head: expected 3, got %d", f.Len())
	}
}

func TestFrame_Tail(t *testing.T) {
	ds := testDS(t)
	f := dataset.From(ds).Tail(3)
	if f.Len() != 3 {
		t.Errorf("Tail: expected 3, got %d", f.Len())
	}
}

func TestFrame_Mutate(t *testing.T) {
	ds := testDS(t)
	f := dataset.From(ds).Mutate("z", dataset.MapFloat64("x", func(v float64) float64 {
		return v * 10
	}))
	col, err := f.Column("z")
	if err != nil {
		t.Fatal(err)
	}
	vals, _ := dataset.CollectFloat64s(col)
	if vals[0] != 10 || vals[9] != 100 {
		t.Errorf("Mutate: unexpected values: %v", vals)
	}
}

func TestFrame_Rename(t *testing.T) {
	ds := testDS(t)
	f := dataset.From(ds).Rename("x", "xx")
	_, err := f.Column("xx")
	if err != nil {
		t.Error("Rename: xx not found")
	}
	_, err = f.Column("x")
	if err == nil {
		t.Error("Rename: old name x should not exist")
	}
}

func TestFrame_Arrange(t *testing.T) {
	ds, _ := dataset.NewDataFrame(map[string][]float64{
		"x": {5, 3, 1, 4, 2},
		"y": {50, 30, 10, 40, 20},
	})
	f := dataset.From(ds).Arrange("x")
	col, _ := f.Column("x")
	vals, _ := dataset.CollectFloat64s(col)
	for i := 1; i < len(vals); i++ {
		if vals[i] < vals[i-1] {
			t.Errorf("Arrange: not sorted at index %d: %v", i, vals)
			break
		}
	}
}

func TestFrame_ArrangeDesc(t *testing.T) {
	ds, _ := dataset.NewDataFrame(map[string][]float64{
		"x": {5, 3, 1, 4, 2},
	})
	f := dataset.From(ds).Arrange("x", true)
	col, _ := f.Column("x")
	vals, _ := dataset.CollectFloat64s(col)
	for i := 1; i < len(vals); i++ {
		if vals[i] > vals[i-1] {
			t.Errorf("ArrangeDesc: not sorted descending at index %d: %v", i, vals)
			break
		}
	}
}

func TestFrame_Distinct(t *testing.T) {
	ds, _ := dataset.NewDataFrame(map[string][]float64{
		"x": {1, 2, 2, 3, 3, 3},
		"y": {10, 20, 20, 30, 30, 30},
	})
	f := dataset.From(ds).Distinct("x")
	if f.Len() != 3 {
		t.Errorf("Distinct: expected 3 unique rows, got %d", f.Len())
	}
}

func TestFrame_Collect(t *testing.T) {
	ds := testDS(t)
	f := dataset.From(ds).
		Filter(dataset.Gt("x", 7)).
		Select("x")

	collected, err := f.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if collected.Len() != 3 {
		t.Errorf("Collect: expected 3, got %d", collected.Len())
	}
}

func TestFrame_GroupBySummarize(t *testing.T) {
	ds, _ := dataset.NewMixedDataFrame(
		dataset.WithFloat64s("value", []float64{10, 20, 30, 40}),
		dataset.WithStrings("group", []string{"A", "A", "B", "B"}),
	)

	result := dataset.From(ds).
		GroupBy("group").
		Summarize(
			dataset.Mean("avg_value", "value"),
			dataset.Sum("total_value", "value"),
			dataset.Count("n", "value"),
		)

	if result.Len() != 2 {
		t.Fatalf("Summarize: expected 2 groups, got %d", result.Len())
	}

	avgCol, err := result.Column("avg_value")
	if err != nil {
		t.Fatal(err)
	}
	avgVals, _ := dataset.CollectFloat64s(avgCol)
	// Group A: mean(10,20)=15, Group B: mean(30,40)=35
	if avgVals[0] != 15 || avgVals[1] != 35 {
		t.Errorf("Mean: got %v, want [15, 35]", avgVals)
	}

	sumCol, _ := result.Column("total_value")
	sumVals, _ := dataset.CollectFloat64s(sumCol)
	if sumVals[0] != 30 || sumVals[1] != 70 {
		t.Errorf("Sum: got %v, want [30, 70]", sumVals)
	}

	countCol, _ := result.Column("n")
	countVals, _ := dataset.CollectFloat64s(countCol)
	if countVals[0] != 2 || countVals[1] != 2 {
		t.Errorf("Count: got %v, want [2, 2]", countVals)
	}
}

func TestSchema(t *testing.T) {
	ds, _ := dataset.NewMixedDataFrame(
		dataset.WithFloat64s("x", []float64{1}),
		dataset.WithStrings("s", []string{"a"}),
	)
	schema := dataset.Schema(ds)
	if schema["x"] != dataset.DTypeFloat64 {
		t.Errorf("Schema[x] = %v", schema["x"])
	}
	if schema["s"] != dataset.DTypeString {
		t.Errorf("Schema[s] = %v", schema["s"])
	}
}

func TestDescribe(t *testing.T) {
	ds := testDS(t)
	desc := dataset.Describe(ds)
	if desc == "" {
		t.Error("Describe returned empty string")
	}
}

// --- Predicate tests ---

func TestPredicate_Between(t *testing.T) {
	ds := testDS(t)
	f := dataset.From(ds).Filter(dataset.Between("x", 3, 7))
	if f.Len() != 5 {
		t.Errorf("Between: expected 5 rows, got %d", f.Len())
	}
}

func TestPredicate_And(t *testing.T) {
	ds := testDS(t)
	f := dataset.From(ds).Filter(
		dataset.And(dataset.Gt("x", 3), dataset.Lt("x", 8)),
	)
	if f.Len() != 4 {
		t.Errorf("And: expected 4 rows, got %d", f.Len())
	}
}

func TestPredicate_Or(t *testing.T) {
	ds := testDS(t)
	f := dataset.From(ds).Filter(
		dataset.Or(dataset.Lt("x", 3), dataset.Gt("x", 8)),
	)
	if f.Len() != 4 {
		t.Errorf("Or: expected 4 rows, got %d", f.Len())
	}
}

func TestPredicate_Not(t *testing.T) {
	ds := testDS(t)
	f := dataset.From(ds).Filter(dataset.Not(dataset.Gt("x", 5)))
	if f.Len() != 5 {
		t.Errorf("Not: expected 5 rows, got %d", f.Len())
	}
}

func TestPredicate_Eq(t *testing.T) {
	ds := testDS(t)
	f := dataset.From(ds).Filter(dataset.Eq("x", 5))
	if f.Len() != 1 {
		t.Errorf("Eq: expected 1 row, got %d", f.Len())
	}
}

func TestFilterMask(t *testing.T) {
	ds := testDS(t)
	mask := []bool{true, false, true, false, true, false, true, false, true, false}
	filtered := dataset.FilterMask(ds, mask)
	if filtered.Len() != 5 {
		t.Errorf("FilterMask: expected 5, got %d", filtered.Len())
	}
}

// --- Int64 column cross-type iteration ---

func TestInt64Column_Float64Iteration(t *testing.T) {
	ds, _ := dataset.NewMixedDataFrame(
		dataset.WithInt64s("id", []int64{10, 20, 30}),
	)
	col, _ := ds.Column("id")
	flt, err := col.(dataset.IterableColumn).Float64s()
	if err != nil {
		t.Fatal(err)
	}
	v, _, ok := flt.Next()
	if !ok || v != 10 {
		t.Errorf("Int64→Float64: got %v", v)
	}
}

// --- NaN handling in Min/Max fallback ---

func TestMinMax_Fallback(t *testing.T) {
	// Use a transformed column (no Aggregator) to test the fallback path.
	ds := testDS(t)
	f := dataset.From(ds).Mutate("z", dataset.MapFloat64("x", func(v float64) float64 {
		return v * 2
	}))
	col, _ := f.Column("z")

	mn, err := dataset.Min(col)
	if err != nil || mn != 2 {
		t.Errorf("Min fallback: got %v, err=%v", mn, err)
	}
	mx, err := dataset.Max(col)
	if err != nil || mx != 20 {
		t.Errorf("Max fallback: got %v, err=%v", mx, err)
	}
}

// --- Float64Column slice/filter ---

func TestFloat64Column_SliceColumn(t *testing.T) {
	col := &dataset.Float64Column{Data: []float64{10, 20, 30, 40, 50}}
	sliced := col.SliceColumn(1, 4)
	if sliced.Len() != 3 {
		t.Errorf("SliceColumn: expected 3, got %d", sliced.Len())
	}
}

func TestFloat64Column_FilterColumn(t *testing.T) {
	col := &dataset.Float64Column{Data: []float64{10, 20, 30, 40, 50}}
	mask := []bool{true, false, true, false, true}
	filtered, err := col.FilterColumn(mask, 3)
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Len() != 3 {
		t.Errorf("FilterColumn: expected 3, got %d", filtered.Len())
	}
}

func TestFloat64Column_Nulls(t *testing.T) {
	col := &dataset.Float64Column{
		Data:  []float64{10, math.NaN(), 30},
		Nulls: []bool{false, true, false},
	}
	mn, _ := col.Min()
	if mn != 10 {
		t.Errorf("Min with nulls: got %v", mn)
	}
}

// --- GroupBy tests ---

func TestGroupBy_StringColumn(t *testing.T) {
	ds, err := dataset.NewMixedDataFrame(
		dataset.WithFloat64s("x", []float64{1, 2, 3, 4, 5, 6}),
		dataset.WithStrings("grp", []string{"A", "B", "A", "B", "A", "B"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	groups, subsets := dataset.GroupBy(ds, "grp")
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0] != "A" || groups[1] != "B" {
		t.Errorf("expected [A, B], got %v", groups)
	}
	if subsets[0].Len() != 3 {
		t.Errorf("group A: expected 3 rows, got %d", subsets[0].Len())
	}
	if subsets[1].Len() != 3 {
		t.Errorf("group B: expected 3 rows, got %d", subsets[1].Len())
	}

	// Verify group A has the correct x values (1, 3, 5).
	col, _ := subsets[0].Column("x")
	vals, _ := dataset.CollectFloat64s(col)
	if vals[0] != 1 || vals[1] != 3 || vals[2] != 5 {
		t.Errorf("group A x values: got %v, want [1, 3, 5]", vals)
	}
}

func TestGroupBy_Float64Column(t *testing.T) {
	ds, err := dataset.NewDataFrame(map[string][]float64{
		"x":   {10, 20, 30, 40},
		"cls": {1, 2, 1, 2},
	})
	if err != nil {
		t.Fatal(err)
	}

	groups, subsets := dataset.GroupBy(ds, "cls")
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	// Float64 groups are formatted as %g, then sorted as strings.
	if groups[0] != "1" || groups[1] != "2" {
		t.Errorf("expected [1, 2], got %v", groups)
	}
	if subsets[0].Len() != 2 || subsets[1].Len() != 2 {
		t.Errorf("group sizes: got %d, %d", subsets[0].Len(), subsets[1].Len())
	}
}

func TestGroupBy_ThreeGroups(t *testing.T) {
	ds, err := dataset.NewMixedDataFrame(
		dataset.WithFloat64s("val", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}),
		dataset.WithStrings("species", []string{
			"setosa", "versicolor", "virginica",
			"setosa", "versicolor", "virginica",
			"setosa", "versicolor", "virginica",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	groups, subsets := dataset.GroupBy(ds, "species")
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	// Sorted alphabetically.
	if groups[0] != "setosa" || groups[1] != "versicolor" || groups[2] != "virginica" {
		t.Errorf("groups = %v", groups)
	}
	for i, g := range groups {
		if subsets[i].Len() != 3 {
			t.Errorf("group %q: expected 3 rows, got %d", g, subsets[i].Len())
		}
	}
}

func TestGroupBy_MissingColumn(t *testing.T) {
	ds, _ := dataset.NewDataFrame(map[string][]float64{"x": {1, 2, 3}})
	groups, subsets := dataset.GroupBy(ds, "nonexistent")

	if len(groups) != 1 || groups[0] != "" {
		t.Errorf("expected single empty group, got %v", groups)
	}
	if subsets[0].Len() != 3 {
		t.Errorf("expected full dataset returned, got %d rows", subsets[0].Len())
	}
}

func TestGroupBy_EmptyDataset(t *testing.T) {
	ds, _ := dataset.NewDataFrame(map[string][]float64{"x": {}})
	groups, subsets := dataset.GroupBy(ds, "x")
	if groups != nil || subsets != nil {
		t.Errorf("expected nil for empty dataset, got groups=%v subsets=%v", groups, subsets)
	}
}

func TestGroupBy_SingleGroup(t *testing.T) {
	ds, _ := dataset.NewMixedDataFrame(
		dataset.WithFloat64s("x", []float64{1, 2, 3}),
		dataset.WithStrings("g", []string{"only", "only", "only"}),
	)

	groups, subsets := dataset.GroupBy(ds, "g")
	if len(groups) != 1 || groups[0] != "only" {
		t.Errorf("expected single group 'only', got %v", groups)
	}
	if subsets[0].Len() != 3 {
		t.Errorf("expected 3 rows, got %d", subsets[0].Len())
	}
}

func TestGroupBy_PreservesAllColumns(t *testing.T) {
	ds, _ := dataset.NewMixedDataFrame(
		dataset.WithFloat64s("x", []float64{1, 2, 3, 4}),
		dataset.WithFloat64s("y", []float64{10, 20, 30, 40}),
		dataset.WithStrings("g", []string{"a", "b", "a", "b"}),
	)

	_, subsets := dataset.GroupBy(ds, "g")
	// Each subset should still have all 3 columns.
	for i, sub := range subsets {
		cols := sub.Columns()
		if len(cols) < 3 {
			t.Errorf("subset %d: expected 3 columns, got %d: %v", i, len(cols), cols)
		}
	}
}
