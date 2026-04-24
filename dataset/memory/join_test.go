package memory

import (
	"math"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
)

// helper: build two test datasets for join testing.
// left:  id=[1,2,3,4]  x=[10,20,30,40]
// right: id=[2,3,5]    y=[200,300,500]
func makeJoinDatasets(t *testing.T) (dataset.Table, dataset.Table) {
	t.Helper()
	eng := NewEngine()
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
	return left, right
}

func getFloat64Values(t *testing.T, ds dataset.Table, name string) []float64 {
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

func getInt64Values(t *testing.T, ds dataset.Table, name string) []int64 {
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

func TestInnerJoin(t *testing.T) {
	left, right := makeJoinDatasets(t)
	eng := NewEngine()

	spec := dataset.On("id")
	spec.Type = dataset.JoinInner
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	// Inner: only id=2,3 match
	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}

	ids := getInt64Values(t, result, "id")
	xs := getFloat64Values(t, result, "x")
	ys := getFloat64Values(t, result, "y")

	// id=2 → x=20, y=200; id=3 → x=30, y=300
	if ids[0] != 2 || ids[1] != 3 {
		t.Errorf("ids = %v, want [2, 3]", ids)
	}
	if xs[0] != 20 || xs[1] != 30 {
		t.Errorf("xs = %v, want [20, 30]", xs)
	}
	if ys[0] != 200 || ys[1] != 300 {
		t.Errorf("ys = %v, want [200, 300]", ys)
	}
}

func TestLeftJoin(t *testing.T) {
	left, right := makeJoinDatasets(t)
	eng := NewEngine()

	spec := dataset.On("id")
	spec.Type = dataset.JoinLeft
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	// Left: all 4 left rows; id=1,4 → y=NaN
	if result.NumRows() != 4 {
		t.Fatalf("expected 4 rows, got %d", result.NumRows())
	}

	ids := getInt64Values(t, result, "id")
	ys := getFloat64Values(t, result, "y")

	if ids[0] != 1 || ids[1] != 2 || ids[2] != 3 || ids[3] != 4 {
		t.Errorf("ids = %v, want [1,2,3,4]", ids)
	}
	if !math.IsNaN(ys[0]) {
		t.Errorf("y[0] = %v, want NaN", ys[0])
	}
	if ys[1] != 200 || ys[2] != 300 {
		t.Errorf("y[1:3] = %v, want [200, 300]", ys[1:3])
	}
	if !math.IsNaN(ys[3]) {
		t.Errorf("y[3] = %v, want NaN", ys[3])
	}
}

func TestRightJoin(t *testing.T) {
	left, right := makeJoinDatasets(t)
	eng := NewEngine()

	spec := dataset.On("id")
	spec.Type = dataset.JoinRight
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	// Right: matched (id=2,3) + unmatched right (id=5) = 3 rows
	if result.NumRows() != 3 {
		t.Fatalf("expected 3 rows, got %d", result.NumRows())
	}

	xs := getFloat64Values(t, result, "x")
	ys := getFloat64Values(t, result, "y")

	// Matched: x=20,y=200; x=30,y=300; Unmatched right: x=NaN,y=500
	if xs[0] != 20 || xs[1] != 30 {
		t.Errorf("xs[0:2] = %v, want [20,30]", xs[:2])
	}
	if !math.IsNaN(xs[2]) {
		t.Errorf("xs[2] = %v, want NaN", xs[2])
	}
	if ys[0] != 200 || ys[1] != 300 || ys[2] != 500 {
		t.Errorf("ys = %v, want [200,300,500]", ys)
	}
}

func TestFullJoin(t *testing.T) {
	left, right := makeJoinDatasets(t)
	eng := NewEngine()

	spec := dataset.On("id")
	spec.Type = dataset.JoinFull
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	// Full: 4 left rows + 1 unmatched right (id=5) = 5 rows
	if result.NumRows() != 5 {
		t.Fatalf("expected 5 rows, got %d", result.NumRows())
	}

	ids := getInt64Values(t, result, "id")
	xs := getFloat64Values(t, result, "x")
	ys := getFloat64Values(t, result, "y")

	// Rows: (1,10,NaN), (2,20,200), (3,30,300), (4,40,NaN), (0,NaN,500)
	// The last row has id=0 (int64 zero-fill) for unmatched right.
	if ids[0] != 1 || ids[1] != 2 || ids[2] != 3 || ids[3] != 4 {
		t.Errorf("ids[0:4] = %v, want [1,2,3,4]", ids[:4])
	}
	if ys[1] != 200 || ys[2] != 300 || ys[4] != 500 {
		t.Errorf("ys = %v", ys)
	}
	if !math.IsNaN(ys[0]) || !math.IsNaN(ys[3]) {
		t.Errorf("unmatched ys should be NaN: %v", ys)
	}
	if !math.IsNaN(xs[4]) {
		t.Errorf("unmatched right x should be NaN: %v", xs[4])
	}
}

func TestSemiJoin(t *testing.T) {
	left, right := makeJoinDatasets(t)
	eng := NewEngine()

	spec := dataset.On("id")
	spec.Type = dataset.JoinSemi
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	// Semi: left rows with match = id=2,3
	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}

	// Should only have left columns (id, x) — no y.
	if result.Schema().HasField("y") {
		t.Error("semi join should not include right columns")
	}

	ids := getInt64Values(t, result, "id")
	if ids[0] != 2 || ids[1] != 3 {
		t.Errorf("ids = %v, want [2,3]", ids)
	}
}

func TestAntiJoin(t *testing.T) {
	left, right := makeJoinDatasets(t)
	eng := NewEngine()

	spec := dataset.On("id")
	spec.Type = dataset.JoinAnti
	result, err := eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}

	// Anti: left rows with no match = id=1,4
	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}

	if result.Schema().HasField("y") {
		t.Error("anti join should not include right columns")
	}

	ids := getInt64Values(t, result, "id")
	if ids[0] != 1 || ids[1] != 4 {
		t.Errorf("ids = %v, want [1,4]", ids)
	}
}

func TestJoinCompositeKey(t *testing.T) {
	eng := NewEngine()

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

	// Only (2024,Feb) and (2025,Jan) match
	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}
	xs := getFloat64Values(t, result, "x")
	ys := getFloat64Values(t, result, "y")
	if xs[0] != 2 || xs[1] != 3 {
		t.Errorf("x = %v, want [2,3]", xs)
	}
	if ys[0] != 200 || ys[1] != 300 {
		t.Errorf("y = %v, want [200,300]", ys)
	}
}

func TestJoinDuplicateKeys(t *testing.T) {
	eng := NewEngine()

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

func TestJoinNoMatch(t *testing.T) {
	eng := NewEngine()

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
		t.Fatalf("expected 0 rows for inner join with no matches, got %d", result.NumRows())
	}

	// Left with no matches → all left rows, y=NaN.
	spec.Type = dataset.JoinLeft
	result, err = eng.Join(left, right, spec)
	if err != nil {
		t.Fatal(err)
	}
	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows for left join, got %d", result.NumRows())
	}
	ys := getFloat64Values(t, result, "y")
	if !math.IsNaN(ys[0]) || !math.IsNaN(ys[1]) {
		t.Errorf("all ys should be NaN: %v", ys)
	}
}

// TestJoinFrameAPI tests that the Frame fluent API correctly dispatches to the Joiner.
func TestJoinFrameAPI(t *testing.T) {
	left, right := makeJoinDatasets(t)

	result, err := dataset.From(left).
		LeftJoin(right, dataset.On("id")).
		Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.NumRows() != 4 {
		t.Fatalf("Frame LeftJoin: expected 4 rows, got %d", result.NumRows())
	}

	// Test InnerJoin through Frame.
	result, err = dataset.From(left).
		InnerJoin(right, dataset.On("id")).
		Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.NumRows() != 2 {
		t.Fatalf("Frame InnerJoin: expected 2 rows, got %d", result.NumRows())
	}
}
