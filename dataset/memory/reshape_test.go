package memory

import (
	"context"
	"math"
	"slices"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
)

func TestPivotLonger(t *testing.T) {
	t.Parallel()

	eng := NewEngine(context.Background())

	// Wide format: id, Q1, Q2, Q3
	schema := dataset.NewSchema(
		dataset.StringCol("id"),
		dataset.FloatCol("Q1"),
		dataset.FloatCol("Q2"),
		dataset.FloatCol("Q3"),
	)

	ds, err := eng.FromColumns(schema,
		eng.NewStringColumn("id", []string{"A", "B"}),
		eng.NewFloat64Column("Q1", []float64{10, 40}),
		eng.NewFloat64Column("Q2", []float64{20, 50}),
		eng.NewFloat64Column("Q3", []float64{30, 60}),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := eng.PivotLonger(ds, dataset.PivotLongerSpec{
		Cols:     []string{"Q1", "Q2", "Q3"},
		NamesTo:  "quarter",
		ValuesTo: "revenue",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 2 rows × 3 pivots = 6 rows
	if result.NumRows() != 6 {
		t.Fatalf("expected 6 rows, got %d", result.NumRows())
	}

	// Check schema: id, quarter, revenue
	if result.Schema().NumFields() != 3 {
		t.Fatalf("expected 3 fields, got %d", result.Schema().NumFields())
	}

	ids := getStringValues(t, result, "id")
	quarters := getStringValues(t, result, "quarter")
	revenues := getFloat64Values(t, result, "revenue")

	// Row 0: A, Q1, 10
	// Row 1: A, Q2, 20
	// Row 2: A, Q3, 30
	// Row 3: B, Q1, 40
	// Row 4: B, Q2, 50
	// Row 5: B, Q3, 60
	expectedIDs := []string{"A", "A", "A", "B", "B", "B"}
	expectedQs := []string{"Q1", "Q2", "Q3", "Q1", "Q2", "Q3"}
	expectedVs := []float64{10, 20, 30, 40, 50, 60}

	for i := range 6 {
		if ids[i] != expectedIDs[i] {
			t.Errorf("row %d: id=%q, want %q", i, ids[i], expectedIDs[i])
		}

		if quarters[i] != expectedQs[i] {
			t.Errorf("row %d: quarter=%q, want %q", i, quarters[i], expectedQs[i])
		}

		if revenues[i] != expectedVs[i] {
			t.Errorf("row %d: revenue=%v, want %v", i, revenues[i], expectedVs[i])
		}
	}
}

func TestPivotWider(t *testing.T) {
	t.Parallel()

	eng := NewEngine(context.Background())

	// Long format: id, quarter, revenue
	schema := dataset.NewSchema(
		dataset.StringCol("id"),
		dataset.StringCol("quarter"),
		dataset.FloatCol("revenue"),
	)

	ds, err := eng.FromColumns(schema,
		eng.NewStringColumn("id", []string{"A", "A", "A", "B", "B", "B"}),
		eng.NewStringColumn("quarter", []string{"Q1", "Q2", "Q3", "Q1", "Q2", "Q3"}),
		eng.NewFloat64Column("revenue", []float64{10, 20, 30, 40, 50, 60}),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := eng.PivotWider(ds, dataset.PivotWiderSpec{
		NamesFrom:  "quarter",
		ValuesFrom: "revenue",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 2 unique ids = 2 rows
	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}

	// Schema: id, Q1, Q2, Q3
	if result.Schema().NumFields() != 4 {
		t.Fatalf("expected 4 fields, got %d", result.Schema().NumFields())
	}

	ids := getStringValues(t, result, "id")
	q1 := getFloat64Values(t, result, "Q1")
	q2 := getFloat64Values(t, result, "Q2")
	q3 := getFloat64Values(t, result, "Q3")

	if ids[0] != "A" || ids[1] != "B" {
		t.Errorf("ids = %v, want [A, B]", ids)
	}

	if q1[0] != 10 || q1[1] != 40 {
		t.Errorf("Q1 = %v, want [10, 40]", q1)
	}

	if q2[0] != 20 || q2[1] != 50 {
		t.Errorf("Q2 = %v, want [20, 50]", q2)
	}

	if q3[0] != 30 || q3[1] != 60 {
		t.Errorf("Q3 = %v, want [30, 60]", q3)
	}
}

func TestPivotRoundTrip(t *testing.T) {
	t.Parallel()

	eng := NewEngine(context.Background())

	schema := dataset.NewSchema(
		dataset.StringCol("id"),
		dataset.FloatCol("Q1"),
		dataset.FloatCol("Q2"),
	)
	ds, _ := eng.FromColumns(schema,
		eng.NewStringColumn("id", []string{"A", "B"}),
		eng.NewFloat64Column("Q1", []float64{10, 40}),
		eng.NewFloat64Column("Q2", []float64{20, 50}),
	)

	// Wide → Long
	long, err := eng.PivotLonger(ds, dataset.PivotLongerSpec{
		Cols: []string{"Q1", "Q2"}, NamesTo: "quarter", ValuesTo: "revenue",
	})
	if err != nil {
		t.Fatal(err)
	}

	if long.NumRows() != 4 {
		t.Fatalf("long: expected 4 rows, got %d", long.NumRows())
	}

	// Long → Wide
	wide, err := eng.PivotWider(long, dataset.PivotWiderSpec{
		NamesFrom: "quarter", ValuesFrom: "revenue",
	})
	if err != nil {
		t.Fatal(err)
	}

	if wide.NumRows() != 2 {
		t.Fatalf("wide: expected 2 rows, got %d", wide.NumRows())
	}

	q1 := getFloat64Values(t, wide, "Q1")

	q2 := getFloat64Values(t, wide, "Q2")
	if q1[0] != 10 || q1[1] != 40 {
		t.Errorf("round-trip Q1 = %v", q1)
	}

	if q2[0] != 20 || q2[1] != 50 {
		t.Errorf("round-trip Q2 = %v", q2)
	}
}

func TestSeparate(t *testing.T) {
	t.Parallel()

	eng := NewEngine(context.Background())

	schema := dataset.NewSchema(dataset.StringCol("date"), dataset.FloatCol("x"))
	ds, _ := eng.FromColumns(schema,
		eng.NewStringColumn("date", []string{"2024-01-15", "2024-06-20", "2025-03-10"}),
		eng.NewFloat64Column("x", []float64{1, 2, 3}),
	)

	result, err := eng.Separate(ds, "date", []string{"year", "month", "day"}, "-")
	if err != nil {
		t.Fatal(err)
	}

	// Should have 4 columns: year, month, day, x
	if result.Schema().NumFields() != 4 {
		t.Fatalf("expected 4 fields, got %d", result.Schema().NumFields())
	}

	years := getStringValues(t, result, "year")
	months := getStringValues(t, result, "month")
	days := getStringValues(t, result, "day")

	if years[0] != "2024" || years[2] != "2025" {
		t.Errorf("years = %v", years)
	}

	if months[0] != "01" || months[1] != "06" {
		t.Errorf("months = %v", months)
	}

	if days[0] != "15" || days[1] != "20" || days[2] != "10" {
		t.Errorf("days = %v", days)
	}
}

func TestConcatenate(t *testing.T) {
	t.Parallel()

	eng := NewEngine(context.Background())

	schema := dataset.NewSchema(
		dataset.StringCol("year"),
		dataset.StringCol("month"),
		dataset.StringCol("day"),
		dataset.FloatCol("x"),
	)
	ds, _ := eng.FromColumns(schema,
		eng.NewStringColumn("year", []string{"2024", "2025"}),
		eng.NewStringColumn("month", []string{"01", "06"}),
		eng.NewStringColumn("day", []string{"15", "20"}),
		eng.NewFloat64Column("x", []float64{1, 2}),
	)

	result, err := eng.Concatenate(ds, "date", []string{"year", "month", "day"}, "-")
	if err != nil {
		t.Fatal(err)
	}

	// Should have 2 columns: date, x (year/month/day removed)
	if result.Schema().NumFields() != 2 {
		t.Fatalf("expected 2 fields, got %d", result.Schema().NumFields())
	}

	dates := getStringValues(t, result, "date")
	if dates[0] != "2024-01-15" || dates[1] != "2025-06-20" {
		t.Errorf("dates = %v", dates)
	}
}

func TestSeparateConcatenateRoundTrip(t *testing.T) {
	t.Parallel()

	eng := NewEngine(context.Background())

	schema := dataset.NewSchema(dataset.StringCol("date"))
	ds, _ := eng.FromColumns(schema,
		eng.NewStringColumn("date", []string{"2024-01-15", "2025-06-20"}),
	)

	sep, _ := eng.Separate(ds, "date", []string{"y", "m", "d"}, "-")
	cat, _ := eng.Concatenate(sep, "date", []string{"y", "m", "d"}, "-")

	dates := getStringValues(t, cat, "date")
	if dates[0] != "2024-01-15" || dates[1] != "2025-06-20" {
		t.Errorf("round-trip dates = %v", dates)
	}
}

func TestComplete(t *testing.T) {
	t.Parallel()

	eng := NewEngine(context.Background())

	// Missing combination: (A, 2025) is not present.
	schema := dataset.NewSchema(
		dataset.StringCol("group"),
		dataset.IntCol("year"),
		dataset.FloatCol("value"),
	)
	ds, _ := eng.FromColumns(schema,
		eng.NewStringColumn("group", []string{"A", "A", "B", "B"}),
		eng.NewInt64Column("year", []int64{2024, 2025, 2024, 2025}),
		eng.NewFloat64Column("value", []float64{10, 20, 30, 40}),
	)

	// Complete should maintain all combinations (already complete).
	result, err := eng.Complete(ds, "group", "year")
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 4 {
		t.Fatalf("expected 4 rows (already complete), got %d", result.NumRows())
	}
}

func TestCompleteWithMissing(t *testing.T) {
	t.Parallel()

	eng := NewEngine(context.Background())

	// Missing combination: (B, 2025) is absent.
	schema := dataset.NewSchema(
		dataset.StringCol("group"),
		dataset.IntCol("year"),
		dataset.FloatCol("value"),
	)
	ds, _ := eng.FromColumns(schema,
		eng.NewStringColumn("group", []string{"A", "A", "B"}),
		eng.NewInt64Column("year", []int64{2024, 2025, 2024}),
		eng.NewFloat64Column("value", []float64{10, 20, 30}),
	)

	result, err := eng.Complete(ds, "group", "year")
	if err != nil {
		t.Fatal(err)
	}

	// 2 groups × 2 years = 4 rows
	if result.NumRows() != 4 {
		t.Fatalf("expected 4 rows, got %d", result.NumRows())
	}

	// The missing (B, 2025) row should have NaN for value.
	values := getFloat64Values(t, result, "value")
	hasNaN := slices.ContainsFunc(values, math.IsNaN)

	if !hasNaN {
		t.Error("expected NaN for missing (B, 2025) row")
	}
}

func TestPivotLongerFrameAPI(t *testing.T) {
	t.Parallel()

	eng := NewEngine(context.Background())

	schema := dataset.NewSchema(
		dataset.StringCol("id"),
		dataset.FloatCol("Q1"),
		dataset.FloatCol("Q2"),
	)
	ds, _ := eng.FromColumns(schema,
		eng.NewStringColumn("id", []string{"A"}),
		eng.NewFloat64Column("Q1", []float64{10}),
		eng.NewFloat64Column("Q2", []float64{20}),
	)

	result, err := dataset.From(ds).PivotLonger(dataset.PivotLongerSpec{
		Cols: []string{"Q1", "Q2"}, NamesTo: "q", ValuesTo: "v",
	}).Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}
}

func getStringValues(t *testing.T, tbl dataset.Table, name string) []string {
	t.Helper()

	col, err := tbl.Column(name)
	if err != nil {
		t.Fatalf("column %q: %v", name, err)
	}

	c, ok := col.(dataset.Column[string])
	if !ok {
		t.Fatalf("column %q is not string", name)
	}

	return c.Values()
}
