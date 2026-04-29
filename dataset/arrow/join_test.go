package arrow

import (
	"context"
	"math"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// helper: build two test datasets for join testing.
// left:  id=[1,2,3,4]  x=[10,20,30,40]
// right: id=[2,3,5]    y=[200,300,500]
func makeJoinDatasets(t *testing.T) (*Engine, dataset.Table, dataset.Table) {
	t.Helper()
	eng := NewEngine(context.Background(), memory.DefaultAllocator)
	schema := dataset.NewSchema(dataset.IntCol("id"), dataset.FloatCol("x"))
	left, err := eng.FromColumns(schema,
		eng.NewInt64Column("id", []int64{1, 2, 3, 4}),
		eng.NewFloat64Column("x", []float64{10, 20, 30, 40}),
	)
	if err != nil {
		t.Fatal(err)
	}

	rSchema := dataset.NewSchema(dataset.IntCol("id"), dataset.FloatCol("y"))
	right, err := eng.FromColumns(rSchema,
		eng.NewInt64Column("id", []int64{2, 3, 5}),
		eng.NewFloat64Column("y", []float64{200, 300, 500}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return eng, left, right
}

func joinFloat64(t *testing.T, ds dataset.Table, name string) []float64 {
	t.Helper()
	col, err := ds.Column(name)
	if err != nil {
		t.Fatalf("column %q: %v", name, err)
	}
	c, ok := col.(dataset.Column[float64])
	if !ok {
		t.Fatalf("column %q is not float64", name)
	}
	return c.Values()
}

func joinInt64(t *testing.T, ds dataset.Table, name string) []int64 {
	t.Helper()
	col, err := ds.Column(name)
	if err != nil {
		t.Fatalf("column %q: %v", name, err)
	}
	c, ok := col.(dataset.Column[int64])
	if !ok {
		t.Fatalf("column %q is not int64", name)
	}
	return c.Values()
}

func joinIsNull(t *testing.T, ds dataset.Table, name string) []bool {
	t.Helper()
	col, err := ds.Column(name)
	if err != nil {
		t.Fatalf("column %q: %v", name, err)
	}
	return col.(interface{ IsNull() []bool }).IsNull()
}

func TestArrowInnerJoin(t *testing.T) {
	eng, left, right := makeJoinDatasets(t)

	spec := dataset.On("id")
	spec.Type = dataset.JoinInner
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}

	ids := joinInt64(t, result, "id")
	xs := joinFloat64(t, result, "x")
	ys := joinFloat64(t, result, "y")

	if ids[0] != 2 || ids[1] != 3 {
		t.Errorf("ids = %v, want [2,3]", ids)
	}
	if xs[0] != 20 || xs[1] != 30 {
		t.Errorf("xs = %v, want [20,30]", xs)
	}
	if ys[0] != 200 || ys[1] != 300 {
		t.Errorf("ys = %v, want [200,300]", ys)
	}
}

func TestArrowLeftJoin(t *testing.T) {
	eng, left, right := makeJoinDatasets(t)

	spec := dataset.On("id")
	spec.Type = dataset.JoinLeft
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 4 {
		t.Fatalf("expected 4 rows, got %d", result.NumRows())
	}

	ids := joinInt64(t, result, "id")
	ys := joinFloat64(t, result, "y")
	nulls := joinIsNull(t, result, "y")

	if ids[0] != 1 || ids[1] != 2 || ids[2] != 3 || ids[3] != 4 {
		t.Errorf("ids = %v, want [1,2,3,4]", ids)
	}
	// Arrow uses null bitmap, not NaN.
	if nulls == nil {
		t.Fatal("expected nulls for y column")
	}
	if !nulls[0] {
		t.Error("y[0] should be null")
	}
	if !nulls[3] {
		t.Error("y[3] should be null")
	}
	if ys[1] != 200 || ys[2] != 300 {
		t.Errorf("ys[1:3] = [%v,%v], want [200,300]", ys[1], ys[2])
	}
}

func TestArrowRightJoin(t *testing.T) {
	eng, left, right := makeJoinDatasets(t)

	spec := dataset.On("id")
	spec.Type = dataset.JoinRight
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	// 2 matched + 1 unmatched right = 3
	if result.NumRows() != 3 {
		t.Fatalf("expected 3 rows, got %d", result.NumRows())
	}

	xNulls := joinIsNull(t, result, "x")
	ys := joinFloat64(t, result, "y")

	// Unmatched right row (id=5): x should be null.
	if xNulls == nil || !xNulls[2] {
		t.Error("x[2] should be null for unmatched right row")
	}
	if ys[0] != 200 || ys[1] != 300 || ys[2] != 500 {
		t.Errorf("ys = %v, want [200,300,500]", ys)
	}
}

func TestArrowFullJoin(t *testing.T) {
	eng, left, right := makeJoinDatasets(t)

	spec := dataset.On("id")
	spec.Type = dataset.JoinFull
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	// 4 left rows + 1 unmatched right = 5
	if result.NumRows() != 5 {
		t.Fatalf("expected 5 rows, got %d", result.NumRows())
	}

	yNulls := joinIsNull(t, result, "y")
	xNulls := joinIsNull(t, result, "x")

	// y should be null for unmatched left rows (0, 3)
	if yNulls == nil || !yNulls[0] || !yNulls[3] {
		t.Error("y should be null for unmatched left rows")
	}
	// x should be null for unmatched right row (4)
	if xNulls == nil || !xNulls[4] {
		t.Error("x should be null for unmatched right row")
	}
}

func TestArrowSemiJoin(t *testing.T) {
	eng, left, right := makeJoinDatasets(t)

	spec := dataset.On("id")
	spec.Type = dataset.JoinSemi
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}

	if result.Schema().HasField("y") {
		t.Error("semi join should not include right columns")
	}

	ids := joinInt64(t, result, "id")
	if ids[0] != 2 || ids[1] != 3 {
		t.Errorf("ids = %v, want [2,3]", ids)
	}
}

func TestArrowAntiJoin(t *testing.T) {
	eng, left, right := makeJoinDatasets(t)

	spec := dataset.On("id")
	spec.Type = dataset.JoinAnti
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}

	if result.Schema().HasField("y") {
		t.Error("anti join should not include right columns")
	}

	ids := joinInt64(t, result, "id")
	if ids[0] != 1 || ids[1] != 4 {
		t.Errorf("ids = %v, want [1,4]", ids)
	}
}

func TestArrowJoinCompositeKey(t *testing.T) {
	eng := NewEngine(context.Background(), memory.DefaultAllocator)

	lSchema := dataset.NewSchema(dataset.IntCol("year"), dataset.StringCol("month"), dataset.FloatCol("x"))
	left, _ := eng.FromColumns(lSchema,
		eng.NewInt64Column("year", []int64{2024, 2024, 2025}),
		eng.NewStringColumn("month", []string{"Jan", "Feb", "Jan"}),
		eng.NewFloat64Column("x", []float64{1, 2, 3}),
	)

	rSchema := dataset.NewSchema(dataset.IntCol("year"), dataset.StringCol("month"), dataset.FloatCol("y"))
	right, _ := eng.FromColumns(rSchema,
		eng.NewInt64Column("year", []int64{2024, 2025}),
		eng.NewStringColumn("month", []string{"Feb", "Jan"}),
		eng.NewFloat64Column("y", []float64{200, 300}),
	)

	spec := dataset.On("year", "month")
	spec.Type = dataset.JoinInner
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}
	xs := joinFloat64(t, result, "x")
	ys := joinFloat64(t, result, "y")
	if xs[0] != 2 || xs[1] != 3 {
		t.Errorf("x = %v, want [2,3]", xs)
	}
	if ys[0] != 200 || ys[1] != 300 {
		t.Errorf("y = %v, want [200,300]", ys)
	}
}

func TestArrowJoinDuplicateKeys(t *testing.T) {
	eng := NewEngine(context.Background(), memory.DefaultAllocator)

	lSchema := dataset.NewSchema(dataset.IntCol("id"), dataset.FloatCol("x"))
	left, _ := eng.FromColumns(lSchema,
		eng.NewInt64Column("id", []int64{1, 1}),
		eng.NewFloat64Column("x", []float64{10, 20}),
	)

	rSchema := dataset.NewSchema(dataset.IntCol("id"), dataset.FloatCol("y"))
	right, _ := eng.FromColumns(rSchema,
		eng.NewInt64Column("id", []int64{1, 1}),
		eng.NewFloat64Column("y", []float64{100, 200}),
	)

	spec := dataset.On("id")
	spec.Type = dataset.JoinInner
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	// Many-to-many: 2×2 = 4 rows
	if result.NumRows() != 4 {
		t.Fatalf("expected 4 rows (2×2 cross), got %d", result.NumRows())
	}
}

func TestArrowJoinNoMatch(t *testing.T) {
	eng := NewEngine(context.Background(), memory.DefaultAllocator)

	lSchema := dataset.NewSchema(dataset.IntCol("id"), dataset.FloatCol("x"))
	left, _ := eng.FromColumns(lSchema,
		eng.NewInt64Column("id", []int64{1, 2}),
		eng.NewFloat64Column("x", []float64{10, 20}),
	)

	rSchema := dataset.NewSchema(dataset.IntCol("id"), dataset.FloatCol("y"))
	right, _ := eng.FromColumns(rSchema,
		eng.NewInt64Column("id", []int64{3, 4}),
		eng.NewFloat64Column("y", []float64{300, 400}),
	)

	// Inner with no matches → 0 rows.
	spec := dataset.On("id")
	spec.Type = dataset.JoinInner
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}
	if result.NumRows() != 0 {
		t.Fatalf("expected 0 rows, got %d", result.NumRows())
	}

	// Left with no matches → all left rows, y nulls.
	spec.Type = dataset.JoinLeft
	result, err = eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}
	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}
	nulls := joinIsNull(t, result, "y")
	if nulls == nil || !nulls[0] || !nulls[1] {
		t.Error("all ys should be null")
	}
}

// TestArrowJoinFrameAPI tests the Frame fluent API dispatches to the Arrow Joiner.
func TestArrowJoinFrameAPI(t *testing.T) {
	_, left, right := makeJoinDatasets(t)

	result, err := dataset.From(left).
		LeftJoin(right, dataset.On("id")).
		Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.NumRows() != 4 {
		t.Fatalf("Frame LeftJoin: expected 4 rows, got %d", result.NumRows())
	}

	result, err = dataset.From(left).
		InnerJoin(right, dataset.On("id")).
		Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.NumRows() != 2 {
		t.Fatalf("Frame InnerJoin: expected 2 rows, got %d", result.NumRows())
	}
}

// Ensure unused import is satisfied
var _ = math.NaN
