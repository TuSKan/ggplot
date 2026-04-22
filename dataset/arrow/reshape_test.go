package arrow

import (
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func TestArrowPivotLonger(t *testing.T) {
	eng := NewEngine(memory.DefaultAllocator)

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
		Cols: []string{"Q1", "Q2", "Q3"}, NamesTo: "quarter", ValuesTo: "revenue",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 6 {
		t.Fatalf("expected 6 rows, got %d", result.NumRows())
	}
	if result.Schema().NumFields() != 3 {
		t.Fatalf("expected 3 fields, got %d", result.Schema().NumFields())
	}

	ids := reshapeStrings(t, result, "id")
	quarters := reshapeStrings(t, result, "quarter")
	revenues := joinFloat64(t, result, "revenue")

	expectedIDs := []string{"A", "A", "A", "B", "B", "B"}
	expectedQs := []string{"Q1", "Q2", "Q3", "Q1", "Q2", "Q3"}
	expectedVs := []float64{10, 20, 30, 40, 50, 60}

	for i := 0; i < 6; i++ {
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

func TestArrowPivotWider(t *testing.T) {
	eng := NewEngine(memory.DefaultAllocator)

	schema := dataset.NewSchema(
		dataset.StringCol("id"),
		dataset.StringCol("quarter"),
		dataset.FloatCol("revenue"),
	)
	ds, _ := eng.FromColumns(schema,
		eng.NewStringColumn("id", []string{"A", "A", "A", "B", "B", "B"}),
		eng.NewStringColumn("quarter", []string{"Q1", "Q2", "Q3", "Q1", "Q2", "Q3"}),
		eng.NewFloat64Column("revenue", []float64{10, 20, 30, 40, 50, 60}),
	)

	result, err := eng.PivotWider(ds, dataset.PivotWiderSpec{
		NamesFrom: "quarter", ValuesFrom: "revenue",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}
	if result.Schema().NumFields() != 4 {
		t.Fatalf("expected 4 fields, got %d", result.Schema().NumFields())
	}

	q1 := joinFloat64(t, result, "Q1")
	q2 := joinFloat64(t, result, "Q2")
	q3 := joinFloat64(t, result, "Q3")

	if q1[0] != 10 || q1[1] != 40 {
		t.Errorf("Q1 = %v, want [10, 40]", q1)
	}
	if q2[0] != 20 || q2[1] != 50 {
		t.Errorf("Q2 = %v", q2)
	}
	if q3[0] != 30 || q3[1] != 60 {
		t.Errorf("Q3 = %v", q3)
	}
}

func TestArrowPivotRoundTrip(t *testing.T) {
	eng := NewEngine(memory.DefaultAllocator)

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

	long, err := eng.PivotLonger(ds, dataset.PivotLongerSpec{
		Cols: []string{"Q1", "Q2"}, NamesTo: "q", ValuesTo: "v",
	})
	if err != nil {
		t.Fatal(err)
	}
	if long.NumRows() != 4 {
		t.Fatalf("long: expected 4, got %d", long.NumRows())
	}

	wide, err := eng.PivotWider(long, dataset.PivotWiderSpec{
		NamesFrom: "q", ValuesFrom: "v",
	})
	if err != nil {
		t.Fatal(err)
	}
	if wide.NumRows() != 2 {
		t.Fatalf("wide: expected 2, got %d", wide.NumRows())
	}

	q1 := joinFloat64(t, wide, "Q1")
	q2 := joinFloat64(t, wide, "Q2")
	if q1[0] != 10 || q1[1] != 40 {
		t.Errorf("Q1 = %v", q1)
	}
	if q2[0] != 20 || q2[1] != 50 {
		t.Errorf("Q2 = %v", q2)
	}
}

func TestArrowSeparate(t *testing.T) {
	eng := NewEngine(memory.DefaultAllocator)

	schema := dataset.NewSchema(dataset.StringCol("date"), dataset.FloatCol("x"))
	ds, _ := eng.FromColumns(schema,
		eng.NewStringColumn("date", []string{"2024-01-15", "2024-06-20", "2025-03-10"}),
		eng.NewFloat64Column("x", []float64{1, 2, 3}),
	)

	result, err := eng.Separate(ds, "date", []string{"year", "month", "day"}, "-")
	if err != nil {
		t.Fatal(err)
	}

	if result.Schema().NumFields() != 4 {
		t.Fatalf("expected 4 fields, got %d", result.Schema().NumFields())
	}

	years := reshapeStrings(t, result, "year")
	months := reshapeStrings(t, result, "month")
	days := reshapeStrings(t, result, "day")

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

func TestArrowConcatenate(t *testing.T) {
	eng := NewEngine(memory.DefaultAllocator)

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

	if result.Schema().NumFields() != 2 {
		t.Fatalf("expected 2 fields, got %d", result.Schema().NumFields())
	}

	dates := reshapeStrings(t, result, "date")
	if dates[0] != "2024-01-15" || dates[1] != "2025-06-20" {
		t.Errorf("dates = %v", dates)
	}
}

func TestArrowSeparateConcatenateRoundTrip(t *testing.T) {
	eng := NewEngine(memory.DefaultAllocator)

	schema := dataset.NewSchema(dataset.StringCol("date"))
	ds, _ := eng.FromColumns(schema,
		eng.NewStringColumn("date", []string{"2024-01-15", "2025-06-20"}),
	)

	sep, _ := eng.Separate(ds, "date", []string{"y", "m", "d"}, "-")
	cat, _ := eng.Concatenate(sep, "date", []string{"y", "m", "d"}, "-")

	dates := reshapeStrings(t, cat, "date")
	if dates[0] != "2024-01-15" || dates[1] != "2025-06-20" {
		t.Errorf("round-trip: %v", dates)
	}
}

func TestArrowComplete(t *testing.T) {
	eng := NewEngine(memory.DefaultAllocator)

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

	result, err := eng.Complete(ds, "group", "year")
	if err != nil {
		t.Fatal(err)
	}
	if result.NumRows() != 4 {
		t.Fatalf("expected 4 rows, got %d", result.NumRows())
	}
}

func TestArrowCompleteWithMissing(t *testing.T) {
	eng := NewEngine(memory.DefaultAllocator)

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

	if result.NumRows() != 4 {
		t.Fatalf("expected 4 rows, got %d", result.NumRows())
	}

	// Arrow uses null bitmap for missing values.
	nulls := joinIsNull(t, result, "value")
	hasNull := false
	for _, v := range nulls {
		if v {
			hasNull = true
			break
		}
	}
	if !hasNull {
		t.Error("expected null for missing (B, 2025) row")
	}
}

func TestArrowPivotLongerFrameAPI(t *testing.T) {
	eng := NewEngine(memory.DefaultAllocator)

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
	}).Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", result.NumRows())
	}
}

func reshapeStrings(t *testing.T, ds dataset.Dataset, name string) []string {
	t.Helper()
	col, err := ds.Column(name)
	if err != nil {
		t.Fatalf("column %q: %v", name, err)
	}
	c, ok := col.(dataset.Column[string])
	if !ok {
		t.Fatalf("column %q is not string", name)
	}
	return c.Values()
}
