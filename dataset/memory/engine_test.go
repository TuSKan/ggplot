package memory_test

import (
	"context"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
)

func TestColumnFactory(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	schema := dataset.NewSchema(
		dataset.FloatCol("x"),
		dataset.StringCol("label"),
		dataset.IntCol("id"),
	)

	f := eng

	ds, err := f.FromColumns(schema,
		f.NewFloat64Column("x", []float64{1, 2, 3}),
		f.NewStringColumn("label", []string{"a", "b", "c"}),
		f.NewInt64Column("id", []int64{10, 20, 30}),
	)
	if err != nil {
		t.Fatal(err)
	}

	if ds.NumRows() != 3 {
		t.Fatalf("expected 3 rows, got %d", ds.NumRows())
	}

	if ds.Schema().NumFields() != 3 {
		t.Fatalf("expected 3 fields, got %d", ds.Schema().NumFields())
	}

	// Typed retrieval
	col, err := dataset.GetColumn[float64](ds, "x")
	if err != nil {
		t.Fatal(err)
	}

	vals := col.Values()
	if len(vals) != 3 || vals[0] != 1 || vals[2] != 3 {
		t.Fatalf("unexpected float64 values: %v", vals)
	}

	// String column
	scol, err := dataset.GetColumn[string](ds, "label")
	if err != nil {
		t.Fatal(err)
	}

	if scol.Values()[1] != "b" {
		t.Fatalf("expected 'b', got %q", scol.Values()[1])
	}

	// Engine propagation
	if dataset.GetEngine(ds) != eng {
		t.Fatal("engine propagation failed")
	}
}

func TestAggregator(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	agg := eng
	f := eng

	fcol := f.NewFloat64Column("x", []float64{10, 20, 30})
	icol := f.NewInt64Column("id", []int64{5, 15, 25})
	scol := f.NewStringColumn("name", []string{"charlie", "alice", "bob"})

	// Sum float64
	result, err := agg.Sum(fcol)
	if err != nil {
		t.Fatal(err)
	}

	sumCol, _ := result.(dataset.Column[float64])
	if sumCol.Values()[0] != 60 {
		t.Fatalf("expected Sum=60, got %v", sumCol.Values()[0])
	}

	// Sum int64 → preserves type
	result, err = agg.Sum(icol)
	if err != nil {
		t.Fatal(err)
	}

	if result.DType() != dataset.DTypeInt64 {
		t.Fatalf("expected int64, got %s", result.DType())
	}

	iSumCol, _ := result.(dataset.Column[int64])
	if iSumCol.Values()[0] != 45 {
		t.Fatalf("expected Sum=45, got %v", iSumCol.Values()[0])
	}

	// MinMax string → lexicographic
	minResult, maxResult, err := agg.MinMax(scol)
	if err != nil {
		t.Fatal(err)
	}

	minCol, _ := minResult.(dataset.Column[string])
	if minCol.Values()[0] != "alice" {
		t.Fatalf("expected Min='alice', got %q", minCol.Values()[0])
	}

	maxCol, _ := maxResult.(dataset.Column[string])
	if maxCol.Values()[0] != "charlie" {
		t.Fatalf("expected Max='charlie', got %q", maxCol.Values()[0])
	}

	// MinMax int64
	iMin, _, _ := agg.MinMax(icol)

	iMinCol, _ := iMin.(dataset.Column[int64])
	if iMinCol.Values()[0] != 5 {
		t.Fatalf("expected Min=5, got %v", iMinCol.Values()[0])
	}

	// Count
	result, _ = agg.Count(fcol)

	cntCol, _ := result.(dataset.Column[int64])
	if cntCol.Values()[0] != 3 {
		t.Fatalf("expected Count=3, got %v", cntCol.Values()[0])
	}
}

func TestBuilder(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	schema := dataset.NewSchema(
		dataset.FloatCol("x"),
		dataset.StringCol("label"),
	)

	bf := eng
	builder := bf.NewBuilder(schema)

	xApp := builder.Float64("x")
	lApp := builder.String("label")

	xApp.Reserve(3)
	xApp.Append(1.0)
	xApp.Append(2.0)
	xApp.Append(3.0)

	lApp.Append("a")
	lApp.Append("b")
	lApp.Append("c")

	ds, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}

	if ds.NumRows() != 3 {
		t.Fatalf("expected 3 rows, got %d", ds.NumRows())
	}

	col, _ := dataset.GetColumn[float64](ds, "x")
	if col.Values()[2] != 3.0 {
		t.Fatalf("expected 3.0, got %v", col.Values()[2])
	}
}

func makeGroupDS(tb testing.TB) dataset.Table {
	tb.Helper()

	eng := memory.NewEngine(context.Background())
	schema := dataset.NewSchema(
		dataset.StringCol("group"),
		dataset.FloatCol("x"),
		dataset.IntCol("id"),
	)

	ds, err := eng.FromColumns(schema,
		eng.NewStringColumn("group", []string{"a", "a", "b", "b", "b"}),
		eng.NewFloat64Column("x", []float64{10, 20, 30, 40, 50}),
		eng.NewInt64Column("id", []int64{1, 2, 3, 4, 5}),
	)
	if err != nil {
		tb.Fatal(err)
	}

	return ds
}

func TestGroupBySummarize(t *testing.T) {
	ds := makeGroupDS(t)

	result, err := dataset.From(ds).
		GroupBy("group").
		Summarize(
			dataset.Sum("total_x", "x"),
			dataset.Count("n", "x"),
			dataset.Mean("avg_x", "x"),
			dataset.Min("min_x", "x"),
			dataset.Max("max_x", "x"),
		).Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// 2 groups: "a", "b"
	if result.NumRows() != 2 {
		t.Fatalf("expected 2 groups, got %d", result.NumRows())
	}

	if result.Schema().NumFields() != 6 {
		t.Fatalf("expected 6 fields, got %d", result.Schema().NumFields())
	}

	// group column
	gcol, _ := dataset.GetColumn[string](result, "group")
	if gcol.Values()[0] != "a" || gcol.Values()[1] != "b" {
		t.Fatalf("unexpected groups: %v", gcol.Values())
	}

	// Sum: a=30, b=120
	sumCol, _ := dataset.GetColumn[float64](result, "total_x")
	if sumCol.Values()[0] != 30 || sumCol.Values()[1] != 120 {
		t.Fatalf("unexpected sums: %v", sumCol.Values())
	}

	// Count: a=2, b=3
	cntCol, _ := dataset.GetColumn[int64](result, "n")
	if cntCol.Values()[0] != 2 || cntCol.Values()[1] != 3 {
		t.Fatalf("unexpected counts: %v", cntCol.Values())
	}

	// Mean: a=15, b=40
	meanCol, _ := dataset.GetColumn[float64](result, "avg_x")
	if meanCol.Values()[0] != 15 || meanCol.Values()[1] != 40 {
		t.Fatalf("unexpected means: %v", meanCol.Values())
	}

	// Min: a=10, b=30
	minCol, _ := dataset.GetColumn[float64](result, "min_x")
	if minCol.Values()[0] != 10 || minCol.Values()[1] != 30 {
		t.Fatalf("unexpected mins: %v", minCol.Values())
	}

	// Max: a=20, b=50
	maxCol, _ := dataset.GetColumn[float64](result, "max_x")
	if maxCol.Values()[0] != 20 || maxCol.Values()[1] != 50 {
		t.Fatalf("unexpected maxs: %v", maxCol.Values())
	}
}

// MutateFunc implementation for testing
type doubleX struct{}

func (doubleX) Apply(ds dataset.Table) (dataset.AnyColumn, error) {
	eng := dataset.GetEngine(ds)
	factory := eng.(dataset.ColumnFactory)

	col, err := dataset.GetColumn[float64](ds, "x")
	if err != nil {
		return nil, err
	}

	vals := col.Values()

	out := make([]float64, len(vals))
	for i, v := range vals {
		out[i] = v * 2
	}

	return factory.NewFloat64Column("x2", out), nil
}

func TestMutateAppend(t *testing.T) {
	ds := makeGroupDS(t)

	result, err := dataset.From(ds).
		Mutate("x2", doubleX{}).
		Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.Schema().NumFields() != 4 {
		t.Fatalf("expected 4 fields, got %d", result.Schema().NumFields())
	}

	col, _ := dataset.GetColumn[float64](result, "x2")

	vals := col.Values()
	if vals[0] != 20 || vals[2] != 60 || vals[4] != 100 {
		t.Fatalf("unexpected x2 values: %v", vals)
	}
}

func TestFullPipeline(t *testing.T) {
	ds := makeGroupDS(t)

	result, err := dataset.From(ds).
		Select("group", "x").
		Arrange("x").
		Head(4).
		GroupBy("group").
		Summarize(
			dataset.Sum("total", "x"),
			dataset.Count("n", "x"),
		).Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// After Arrange: [10,20,30,40,50], Head(4): [10,20,30,40]
	// Groups: a=[10,20], b=[30,40]
	if result.NumRows() != 2 {
		t.Fatalf("expected 2 groups, got %d", result.NumRows())
	}
}

func makeTestDS(t *testing.T) dataset.Table {
	t.Helper()

	eng := memory.NewEngine(context.Background())
	schema := dataset.NewSchema(
		dataset.FloatCol("x"),
		dataset.StringCol("label"),
		dataset.IntCol("id"),
	)

	ds, err := eng.FromColumns(schema,
		eng.NewFloat64Column("x", []float64{3, 1, 4, 1, 5}),
		eng.NewStringColumn("label", []string{"c", "a", "d", "a", "e"}),
		eng.NewInt64Column("id", []int64{30, 10, 40, 10, 50}),
	)
	if err != nil {
		t.Fatal(err)
	}

	return ds
}

func TestFrameSelect(t *testing.T) {
	ds := makeTestDS(t)

	result, err := dataset.From(ds).Select("x", "label").Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.Schema().NumFields() != 2 {
		t.Fatalf("expected 2 fields, got %d", result.Schema().NumFields())
	}

	if result.Schema().Field(0).Name != "x" {
		t.Fatalf("expected first field 'x', got %q", result.Schema().Field(0).Name)
	}

	if result.NumRows() != 5 {
		t.Fatalf("expected 5 rows, got %d", result.NumRows())
	}
}

func TestFrameHead(t *testing.T) {
	ds := makeTestDS(t)

	result, err := dataset.From(ds).Head(3).Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 3 {
		t.Fatalf("expected 3 rows, got %d", result.NumRows())
	}

	col, _ := dataset.GetColumn[float64](result, "x")

	vals := col.Values()
	if vals[0] != 3 || vals[1] != 1 || vals[2] != 4 {
		t.Fatalf("unexpected values: %v", vals)
	}
}

func TestFrameTail(t *testing.T) {
	ds := makeTestDS(t)

	result, err := dataset.From(ds).Tail(2).Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}

	col, _ := dataset.GetColumn[float64](result, "x")

	vals := col.Values()
	if vals[0] != 1 || vals[1] != 5 {
		t.Fatalf("unexpected values: %v", vals)
	}
}

func TestFrameArrange(t *testing.T) {
	ds := makeTestDS(t)

	result, err := dataset.From(ds).Arrange("x").Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	col, _ := dataset.GetColumn[float64](result, "x")

	vals := col.Values()
	for i := 1; i < len(vals); i++ {
		if vals[i] < vals[i-1] {
			t.Fatalf("not sorted: %v", vals)
		}
	}
}

func TestFrameDistinct(t *testing.T) {
	ds := makeTestDS(t)

	result, err := dataset.From(ds).Distinct("label").Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// "c", "a", "d", "a", "e" → distinct by label → 4 rows (first "a" kept)
	if result.NumRows() != 4 {
		t.Fatalf("expected 4 distinct rows, got %d", result.NumRows())
	}
}

func TestFrameChain(t *testing.T) {
	ds := makeTestDS(t)

	result, err := dataset.From(ds).
		Select("x", "label").
		Head(4).
		Arrange("x").
		Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 4 {
		t.Fatalf("expected 4 rows, got %d", result.NumRows())
	}

	col, _ := dataset.GetColumn[float64](result, "x")
	vals := col.Values()
	// After Head(4): [3,1,4,1], After Arrange: [1,1,3,4]
	if vals[0] != 1 || vals[1] != 1 || vals[2] != 3 || vals[3] != 4 {
		t.Fatalf("unexpected sorted values: %v", vals)
	}
}

// --- Filter tests using Predicate ---

func TestFilter(t *testing.T) {
	ds := makeTestDS(t)
	// x = [3, 1, 4, 1, 5], filter x > 3 → [4, 5]
	result, err := dataset.From(ds).
		Filter(dataset.Gt("x", 3.0)).
		Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}

	col, _ := dataset.GetColumn[float64](result, "x")

	vals := col.Values()
	if vals[0] != 4 || vals[1] != 5 {
		t.Fatalf("unexpected values: %v", vals)
	}
	// Labels should follow
	lcol, _ := dataset.GetColumn[string](result, "label")
	if lcol.Values()[0] != "d" || lcol.Values()[1] != "e" {
		t.Fatalf("unexpected labels: %v", lcol.Values())
	}
}

func TestFilterEmpty(t *testing.T) {
	ds := makeTestDS(t)
	// x = [3, 1, 4, 1, 5], filter x > 100 → empty
	result, err := dataset.From(ds).
		Filter(dataset.Gt("x", 100.0)).
		Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 0 {
		t.Fatalf("expected 0 rows, got %d", result.NumRows())
	}

	if result.Schema().NumFields() != 3 {
		t.Fatalf("expected 3 fields, got %d", result.Schema().NumFields())
	}
}

func TestDropNA(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	schema := dataset.NewSchema(
		dataset.FloatCol("x"),
		dataset.StringCol("label"),
	)
	// Use builder to inject NaN nulls
	b := eng.NewBuilder(schema)
	xApp := b.Float64("x")
	lApp := b.String("label")

	xApp.Append(1.0)
	lApp.Append("a")

	xApp.AppendNull() // NaN
	lApp.Append("b")

	xApp.Append(3.0)
	lApp.Append("c")

	ds, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	result, err := dataset.From(ds).DropNA("x").Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows after DropNA, got %d", result.NumRows())
	}

	col, _ := dataset.GetColumn[float64](result, "x")

	vals := col.Values()
	if vals[0] != 1 || vals[1] != 3 {
		t.Fatalf("unexpected values: %v", vals)
	}

	lcol, _ := dataset.GetColumn[string](result, "label")
	if lcol.Values()[0] != "a" || lcol.Values()[1] != "c" {
		t.Fatalf("unexpected labels: %v", lcol.Values())
	}
}

func TestFillDown(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	schema := dataset.NewSchema(dataset.FloatCol("x"))
	b := eng.NewBuilder(schema)
	xApp := b.Float64("x")
	xApp.Append(1.0)
	xApp.AppendNull()
	xApp.AppendNull()
	xApp.Append(4.0)
	xApp.AppendNull()

	ds, _ := b.Build()

	result, err := dataset.From(ds).Fill("x", dataset.FillDown).Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	col, _ := dataset.GetColumn[float64](result, "x")
	vals := col.Values()
	// [1, NaN, 4, NaN] → FillDown → [1, 1, 4, 4]
	expected := []float64{1, 1, 1, 4, 4}
	for i, v := range vals {
		if v != expected[i] {
			t.Fatalf("at %d: expected %f, got %f (full: %v)", i, expected[i], v, vals)
		}
	}
}

func TestReplaceNA(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	schema := dataset.NewSchema(dataset.FloatCol("x"))
	b := eng.NewBuilder(schema)
	xApp := b.Float64("x")
	xApp.Append(1.0)
	xApp.AppendNull()
	xApp.Append(3.0)

	ds, _ := b.Build()

	filler := dataset.Engine(eng).(dataset.Filler)
	col, _ := ds.Column("x")

	replaced, err := filler.ReplaceNA(col, -999)
	if err != nil {
		t.Fatal(err)
	}

	rCol := replaced.(dataset.Column[float64])

	vals := rCol.Values()
	if vals[0] != 1 || vals[1] != -999 || vals[2] != 3 {
		t.Fatalf("unexpected values: %v", vals)
	}
}

func TestStack(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	schema := dataset.NewSchema(
		dataset.FloatCol("x"),
		dataset.StringCol("label"),
	)
	ds1, _ := eng.FromColumns(schema,
		eng.NewFloat64Column("x", []float64{1, 2}),
		eng.NewStringColumn("label", []string{"a", "b"}),
	)
	ds2, _ := eng.FromColumns(schema,
		eng.NewFloat64Column("x", []float64{3, 4, 5}),
		eng.NewStringColumn("label", []string{"c", "d", "e"}),
	)

	result, err := dataset.From(ds1).Stack(ds2).Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 5 {
		t.Fatalf("expected 5 rows, got %d", result.NumRows())
	}

	col, _ := dataset.GetColumn[float64](result, "x")
	vals := col.Values()

	expected := []float64{1, 2, 3, 4, 5}
	for i, v := range vals {
		if v != expected[i] {
			t.Fatalf("at %d: expected %f, got %f", i, expected[i], v)
		}
	}

	lcol, _ := dataset.GetColumn[string](result, "label")
	if lcol.Values()[4] != "e" {
		t.Fatalf("unexpected last label: %q", lcol.Values()[4])
	}
}

func TestCombine(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	ds1, _ := eng.FromColumns(
		dataset.NewSchema(dataset.FloatCol("x")),
		eng.NewFloat64Column("x", []float64{1, 2, 3}),
	)
	ds2, _ := eng.FromColumns(
		dataset.NewSchema(dataset.StringCol("label")),
		eng.NewStringColumn("label", []string{"a", "b", "c"}),
	)

	result, err := dataset.From(ds1).Combine(ds2).Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.Schema().NumFields() != 2 {
		t.Fatalf("expected 2 fields, got %d", result.Schema().NumFields())
	}

	if result.NumRows() != 3 {
		t.Fatalf("expected 3 rows, got %d", result.NumRows())
	}

	xcol, _ := dataset.GetColumn[float64](result, "x")

	lcol, _ := dataset.GetColumn[string](result, "label")
	if xcol.Values()[0] != 1 || lcol.Values()[2] != "c" {
		t.Fatalf("unexpected values")
	}
}

// --- Windower ---

func TestLagLead(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("x", []float64{10, 20, 30, 40, 50})

	lag, err := eng.Lag(col, 2)
	if err != nil {
		t.Fatal(err)
	}

	lv := lag.(dataset.Column[float64]).Values()
	// [0, 0, 10, 20, 30]
	if lv[0] != 0 || lv[1] != 0 || lv[2] != 10 || lv[3] != 20 || lv[4] != 30 {
		t.Fatalf("Lag(2) unexpected: %v", lv)
	}

	lead, err := eng.Lead(col, 1)
	if err != nil {
		t.Fatal(err)
	}

	dv := lead.(dataset.Column[float64]).Values()
	// [20, 30, 40, 50, 0]
	if dv[0] != 20 || dv[3] != 50 || dv[4] != 0 {
		t.Fatalf("Lead(1) unexpected: %v", dv)
	}
}

func TestCumSum(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5})

	cs, err := eng.CumSum(col)
	if err != nil {
		t.Fatal(err)
	}

	vals := cs.(dataset.Column[float64]).Values()
	// [1, 3, 6, 10, 15]
	expected := []float64{1, 3, 6, 10, 15}
	for i, v := range vals {
		if v != expected[i] {
			t.Fatalf("CumSum[%d]: expected %f, got %f", i, expected[i], v)
		}
	}
}

func TestCumMaxMin(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("x", []float64{3, 1, 4, 1, 5})

	mx, _ := eng.CumMax(col)
	mxv := mx.(dataset.Column[float64]).Values()
	// [3, 3, 4, 4, 5]
	if mxv[0] != 3 || mxv[1] != 3 || mxv[2] != 4 || mxv[3] != 4 || mxv[4] != 5 {
		t.Fatalf("CumMax unexpected: %v", mxv)
	}

	mn, _ := eng.CumMin(col)
	mnv := mn.(dataset.Column[float64]).Values()
	// [3, 1, 1, 1, 1]
	if mnv[0] != 3 || mnv[1] != 1 || mnv[2] != 1 || mnv[4] != 1 {
		t.Fatalf("CumMin unexpected: %v", mnv)
	}
}

func TestRankDenseRank(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	// Values with ties: [10, 30, 20, 20, 40]
	col := eng.NewFloat64Column("x", []float64{10, 30, 20, 20, 40})

	// Rank: competition (ties share, next skips)
	// Sorted order: 10(0), 20(2), 20(3), 30(1), 40(4)
	// Ranks by position: 10→1, 20→2, 20→2, 30→4, 40→5
	r, err := eng.Rank(col)
	if err != nil {
		t.Fatal(err)
	}

	rv := r.(dataset.Column[int64]).Values()
	if rv[0] != 1 || rv[1] != 4 || rv[2] != 2 || rv[3] != 2 || rv[4] != 5 {
		t.Fatalf("Rank unexpected: %v", rv)
	}

	// DenseRank: no gaps
	// 10→1, 20→2, 20→2, 30→3, 40→4
	dr, err := eng.DenseRank(col)
	if err != nil {
		t.Fatal(err)
	}

	drv := dr.(dataset.Column[int64]).Values()
	if drv[0] != 1 || drv[1] != 3 || drv[2] != 2 || drv[3] != 2 || drv[4] != 4 {
		t.Fatalf("DenseRank unexpected: %v", drv)
	}
}

func TestRowNumber(t *testing.T) {
	eng := memory.NewEngine(context.Background())

	rn, err := eng.RowNumber(5)
	if err != nil {
		t.Fatal(err)
	}

	vals := rn.(dataset.Column[int64]).Values()
	for i, v := range vals {
		if v != int64(i+1) {
			t.Fatalf("RowNumber[%d]: expected %d, got %d", i, i+1, v)
		}
	}
}
