package csv

import (
	"bytes"
	"context"
	"math"
	"strings"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	arrowEngine "github.com/TuSKan/ggplot/dataset/arrow"
	memEngine "github.com/TuSKan/ggplot/dataset/memory"

	"github.com/apache/arrow-go/v18/arrow/memory"
)

const testCSV = `name,age,score,active
Alice,30,95.5,true
Bob,25,87.3,false
Charlie,35,NA,true
`

// --- Memory engine tests ---

func TestReadMemory(t *testing.T) {
	eng := memEngine.NewEngine()
	ds, err := Read(context.Background(), strings.NewReader(testCSV), eng)
	if err != nil {
		t.Fatal(err)
	}

	if ds.NumRows() != 3 {
		t.Fatalf("expected 3 rows, got %d", ds.NumRows())
	}
	if ds.Schema().NumFields() != 4 {
		t.Fatalf("expected 4 fields, got %d", ds.Schema().NumFields())
	}

	// name: string
	nameCol, _ := ds.Column("name")
	names := nameCol.(dataset.Column[string]).Values()
	if names[0] != "Alice" || names[1] != "Bob" || names[2] != "Charlie" {
		t.Errorf("names = %v", names)
	}

	// age: int64
	ageCol, _ := ds.Column("age")
	ages := ageCol.(dataset.Column[int64]).Values()
	if ages[0] != 30 || ages[1] != 25 || ages[2] != 35 {
		t.Errorf("ages = %v", ages)
	}

	// score: float64 (has "NA" → NaN)
	scoreCol, _ := ds.Column("score")
	scores := scoreCol.(dataset.Column[float64]).Values()
	if scores[0] != 95.5 || scores[1] != 87.3 {
		t.Errorf("scores = %v", scores)
	}
	if !math.IsNaN(scores[2]) {
		t.Errorf("scores[2] should be NaN, got %v", scores[2])
	}

	// active: bool
	activeCol, _ := ds.Column("active")
	actives := activeCol.(dataset.Column[bool]).Values()
	if !actives[0] || actives[1] || !actives[2] {
		t.Errorf("actives = %v", actives)
	}
}

func TestReadMemoryNoHeader(t *testing.T) {
	csv := "Alice,30\nBob,25\n"
	eng := memEngine.NewEngine()
	ds, err := Read(context.Background(), strings.NewReader(csv), eng, WithHeader(false))
	if err != nil {
		t.Fatal(err)
	}
	if ds.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", ds.NumRows())
	}
	if !ds.Schema().HasField("V1") || !ds.Schema().HasField("V2") {
		t.Errorf("expected V1, V2 columns")
	}
}

func TestReadMemoryTSV(t *testing.T) {
	csv := "name\tage\nAlice\t30\nBob\t25\n"
	eng := memEngine.NewEngine()
	ds, err := Read(context.Background(), strings.NewReader(csv), eng, WithComma('\t'))
	if err != nil {
		t.Fatal(err)
	}
	if ds.NumRows() != 2 {
		t.Fatalf("expected 2 rows, got %d", ds.NumRows())
	}
	nameCol, _ := ds.Column("name")
	names := nameCol.(dataset.Column[string]).Values()
	if names[0] != "Alice" {
		t.Errorf("name[0] = %q", names[0])
	}
}

func TestWriteMemory(t *testing.T) {
	eng := memEngine.NewEngine()
	ds, _ := dataset.NewDataset(eng,
		eng.NewStringColumn("name", []string{"Alice", "Bob"}),
		eng.NewInt64Column("age", []int64{30, 25}),
		eng.NewFloat64Column("score", []float64{95.5, math.NaN()}),
	)

	var buf bytes.Buffer
	err := Write(context.Background(), &buf, ds, eng)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "name,age,score") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "Alice,30,95.5") {
		t.Errorf("missing Alice row in:\n%s", output)
	}
	if !strings.Contains(output, "Bob,25,NA") {
		t.Errorf("NaN should be written as NA, got:\n%s", output)
	}
}

func TestRoundTripMemory(t *testing.T) {
	eng := memEngine.NewEngine()
	ds, _ := dataset.NewDataset(eng,
		eng.NewStringColumn("city", []string{"SP", "RJ", "BH"}),
		eng.NewInt64Column("pop", []int64{12000000, 6700000, 2500000}),
		eng.NewFloat64Column("area", []float64{1521.1, 1200.3, 331.4}),
	)

	var buf bytes.Buffer
	if err := Write(context.Background(), &buf, ds, eng); err != nil {
		t.Fatal(err)
	}

	ds2, err := Read(context.Background(), &buf, eng)
	if err != nil {
		t.Fatal(err)
	}

	if ds2.NumRows() != 3 {
		t.Fatalf("round-trip: expected 3 rows, got %d", ds2.NumRows())
	}

	cityCol, _ := ds2.Column("city")
	cities := cityCol.(dataset.Column[string]).Values()
	if cities[0] != "SP" || cities[1] != "RJ" || cities[2] != "BH" {
		t.Errorf("cities = %v", cities)
	}

	popCol, _ := ds2.Column("pop")
	pops := popCol.(dataset.Column[int64]).Values()
	if pops[0] != 12000000 {
		t.Errorf("pop[0] = %d, want 12000000", pops[0])
	}
}

// --- Arrow engine tests ---

func TestReadArrow(t *testing.T) {
	eng := arrowEngine.NewEngine(memory.DefaultAllocator)
	ds, err := Read(context.Background(), strings.NewReader(testCSV), eng)
	if err != nil {
		t.Fatal(err)
	}

	if ds.NumRows() != 3 {
		t.Fatalf("expected 3 rows, got %d", ds.NumRows())
	}

	nameCol, _ := ds.Column("name")
	names := nameCol.(dataset.Column[string]).Values()
	if names[0] != "Alice" {
		t.Errorf("name[0] = %q", names[0])
	}
}

func TestWriteArrow(t *testing.T) {
	eng := arrowEngine.NewEngine(memory.DefaultAllocator)
	ds, _ := dataset.NewDataset(eng,
		eng.NewStringColumn("x", []string{"a", "b"}),
		eng.NewFloat64Column("y", []float64{1.5, 2.5}),
	)

	var buf bytes.Buffer
	err := Write(context.Background(), &buf, ds, eng)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "x,y") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "a,1.5") {
		t.Errorf("missing row in:\n%s", output)
	}
}

func TestRoundTripArrow(t *testing.T) {
	eng := arrowEngine.NewEngine(memory.DefaultAllocator)
	ds, _ := dataset.NewDataset(eng,
		eng.NewStringColumn("city", []string{"SP", "RJ"}),
		eng.NewFloat64Column("pop", []float64{12.5, 6.7}),
	)

	var buf bytes.Buffer
	if err := Write(context.Background(), &buf, ds, eng); err != nil {
		t.Fatal(err)
	}

	ds2, err := Read(context.Background(), &buf, eng)
	if err != nil {
		t.Fatal(err)
	}

	if ds2.NumRows() != 2 {
		t.Fatalf("round-trip: expected 2 rows, got %d", ds2.NumRows())
	}
}

func TestHeaderOnlyCSV(t *testing.T) {
	eng := memEngine.NewEngine()
	ds, err := Read(context.Background(), strings.NewReader("name,age\n"), eng)
	if err != nil {
		t.Fatal(err)
	}
	if ds.NumRows() != 0 {
		t.Fatalf("expected 0 rows, got %d", ds.NumRows())
	}
	if ds.Schema().NumFields() != 2 {
		t.Fatalf("expected 2 fields, got %d", ds.Schema().NumFields())
	}
}
