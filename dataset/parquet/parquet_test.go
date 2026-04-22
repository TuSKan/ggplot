package parquet

import (
	"bytes"
	"math"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	arrowEngine "github.com/TuSKan/ggplot/dataset/arrow"
	memEngine "github.com/TuSKan/ggplot/dataset/memory"

	"github.com/apache/arrow-go/v18/arrow/memory"
)

// --- Memory engine tests ---

func TestRoundTripMemory(t *testing.T) {
	eng := memEngine.NewEngine()
	schema := dataset.NewSchema(
		dataset.StringCol("city"),
		dataset.IntCol("pop"),
		dataset.FloatCol("area"),
	)
	ds, err := eng.FromColumns(schema,
		eng.NewStringColumn("city", []string{"SP", "RJ", "BH"}),
		eng.NewInt64Column("pop", []int64{12000000, 6700000, 2500000}),
		eng.NewFloat64Column("area", []float64{1521.1, 1200.3, 331.4}),
	)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := Write(&buf, ds, eng); err != nil {
		t.Fatal(err)
	}

	data := buf.Bytes()
	ds2, err := Read(bytes.NewReader(data), int64(len(data)), eng)
	if err != nil {
		t.Fatal(err)
	}

	if ds2.NumRows() != 3 {
		t.Fatalf("expected 3 rows, got %d", ds2.NumRows())
	}
	if ds2.Schema().NumFields() != 3 {
		t.Fatalf("expected 3 fields, got %d", ds2.Schema().NumFields())
	}

	cityCol, _ := ds2.Column("city")
	cities := cityCol.(dataset.Column[string]).Values()
	if cities[0] != "SP" || cities[1] != "RJ" || cities[2] != "BH" {
		t.Errorf("cities = %v", cities)
	}

	popCol, _ := ds2.Column("pop")
	pops := popCol.(dataset.Column[int64]).Values()
	if pops[0] != 12000000 || pops[1] != 6700000 || pops[2] != 2500000 {
		t.Errorf("pops = %v", pops)
	}

	areaCol, _ := ds2.Column("area")
	areas := areaCol.(dataset.Column[float64]).Values()
	if areas[0] != 1521.1 || areas[1] != 1200.3 || areas[2] != 331.4 {
		t.Errorf("areas = %v", areas)
	}
}

func TestMemoryNullHandling(t *testing.T) {
	eng := memEngine.NewEngine()
	schema := dataset.NewSchema(
		dataset.StringCol("name"),
		dataset.FloatCol("val"),
	)
	ds, _ := eng.FromColumns(schema,
		eng.NewStringColumn("name", []string{"a", "b"}),
		eng.NewFloat64Column("val", []float64{1.5, math.NaN()}),
	)

	var buf bytes.Buffer
	if err := Write(&buf, ds, eng); err != nil {
		t.Fatal(err)
	}

	data := buf.Bytes()
	ds2, err := Read(bytes.NewReader(data), int64(len(data)), eng)
	if err != nil {
		t.Fatal(err)
	}

	valCol, _ := ds2.Column("val")
	vals := valCol.(dataset.Column[float64]).Values()
	if vals[0] != 1.5 {
		t.Errorf("val[0] = %v, want 1.5", vals[0])
	}
	if !math.IsNaN(vals[1]) {
		t.Errorf("val[1] = %v, want NaN", vals[1])
	}
}

// --- Arrow engine tests ---

func TestRoundTripArrow(t *testing.T) {
	eng := arrowEngine.NewEngine(memory.DefaultAllocator)
	schema := dataset.NewSchema(
		dataset.StringCol("city"),
		dataset.IntCol("pop"),
		dataset.FloatCol("area"),
	)
	ds, err := eng.FromColumns(schema,
		eng.NewStringColumn("city", []string{"SP", "RJ"}),
		eng.NewInt64Column("pop", []int64{12000000, 6700000}),
		eng.NewFloat64Column("area", []float64{1521.1, 1200.3}),
	)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := Write(&buf, ds, eng); err != nil {
		t.Fatal(err)
	}

	data := buf.Bytes()
	ds2, err := Read(bytes.NewReader(data), int64(len(data)), eng)
	if err != nil {
		t.Fatal(err)
	}

	if ds2.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", ds2.NumRows())
	}

	cityCol, _ := ds2.Column("city")
	cities := cityCol.(dataset.Column[string]).Values()
	if cities[0] != "SP" || cities[1] != "RJ" {
		t.Errorf("cities = %v", cities)
	}

	popCol, _ := ds2.Column("pop")
	pops := popCol.(dataset.Column[int64]).Values()
	if pops[0] != 12000000 || pops[1] != 6700000 {
		t.Errorf("pops = %v", pops)
	}
}

func TestArrowNullHandling(t *testing.T) {
	eng := arrowEngine.NewEngine(memory.DefaultAllocator)
	schema := dataset.NewSchema(
		dataset.FloatCol("x"),
	)
	ds, _ := eng.FromColumns(schema,
		eng.NewFloat64Column("x", []float64{42.0, math.NaN(), 3.14}),
	)

	var buf bytes.Buffer
	if err := Write(&buf, ds, eng); err != nil {
		t.Fatal(err)
	}

	data := buf.Bytes()
	ds2, err := Read(bytes.NewReader(data), int64(len(data)), eng)
	if err != nil {
		t.Fatal(err)
	}

	if ds2.NumRows() != 3 {
		t.Fatalf("expected 3 rows, got %d", ds2.NumRows())
	}

	xCol, _ := ds2.Column("x")
	xs := xCol.(dataset.Column[float64]).Values()
	if xs[0] != 42.0 {
		t.Errorf("x[0] = %v, want 42.0", xs[0])
	}
	if !math.IsNaN(xs[1]) {
		t.Errorf("x[1] = %v, want NaN", xs[1])
	}
	if xs[2] != 3.14 {
		t.Errorf("x[2] = %v, want 3.14", xs[2])
	}
}

func TestBoolRoundTrip(t *testing.T) {
	eng := memEngine.NewEngine()
	schema := dataset.NewSchema(
		dataset.BoolCol("flag"),
	)
	ds, _ := eng.FromColumns(schema,
		eng.NewBoolColumn("flag", []bool{true, false, true}),
	)

	var buf bytes.Buffer
	if err := Write(&buf, ds, eng); err != nil {
		t.Fatal(err)
	}

	data := buf.Bytes()
	ds2, err := Read(bytes.NewReader(data), int64(len(data)), eng)
	if err != nil {
		t.Fatal(err)
	}

	flagCol, _ := ds2.Column("flag")
	flags := flagCol.(dataset.Column[bool]).Values()
	if !flags[0] || flags[1] || !flags[2] {
		t.Errorf("flags = %v", flags)
	}
}

// --- Cross-engine test ---

func TestCrossEngineReadWrite(t *testing.T) {
	// Write with memory, read with arrow.
	memEng := memEngine.NewEngine()
	arrowEng := arrowEngine.NewEngine(memory.DefaultAllocator)

	schema := dataset.NewSchema(
		dataset.StringCol("name"),
		dataset.FloatCol("val"),
	)
	ds, _ := memEng.FromColumns(schema,
		memEng.NewStringColumn("name", []string{"a", "b", "c"}),
		memEng.NewFloat64Column("val", []float64{1.0, 2.0, 3.0}),
	)

	var buf bytes.Buffer
	if err := Write(&buf, ds, memEng); err != nil {
		t.Fatal(err)
	}

	data := buf.Bytes()
	ds2, err := Read(bytes.NewReader(data), int64(len(data)), arrowEng)
	if err != nil {
		t.Fatal(err)
	}

	if ds2.NumRows() != 3 {
		t.Fatalf("cross-engine: expected 3 rows, got %d", ds2.NumRows())
	}

	nameCol, _ := ds2.Column("name")
	names := nameCol.(dataset.Column[string]).Values()
	if names[0] != "a" || names[2] != "c" {
		t.Errorf("names = %v", names)
	}
}

func TestWithCompression(t *testing.T) {
	eng := memEngine.NewEngine()
	schema := dataset.NewSchema(
		dataset.IntCol("x"),
	)
	ds, _ := eng.FromColumns(schema,
		eng.NewInt64Column("x", []int64{1, 2, 3, 4, 5}),
	)

	var buf bytes.Buffer
	if err := Write(&buf, ds, eng, WithCompression("snappy")); err != nil {
		t.Fatal(err)
	}

	data := buf.Bytes()
	ds2, err := Read(bytes.NewReader(data), int64(len(data)), eng)
	if err != nil {
		t.Fatal(err)
	}

	if ds2.NumRows() != 5 {
		t.Fatalf("expected 5 rows, got %d", ds2.NumRows())
	}
}
