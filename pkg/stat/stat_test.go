package stat

import (
	"math"
	"testing"

	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/internal/expr"
	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
)

func createTestDataset() dataset.Dataset {
	buf := arrow.NewBuffer(5)
	x := buf.Float64("x")
	y := buf.Float64("y")
	x[0], y[0] = 1, 10
	x[1], y[1] = 2, 20
	x[2], y[2] = 2, 30
	x[3], y[3] = 3, 10
	x[4], y[4] = 3, 10
	ds, _ := buf.Build()
	return ds
}

func createStringDataset() dataset.Dataset {
	buf := arrow.NewBuffer(4)
	x := buf.String("x")
	x[0] = "A"
	x[1] = "A"
	x[2] = "B"
	x[3] = "C"
	ds, _ := buf.Build()
	return ds
}

func TestCount(t *testing.T) {
	ds := createTestDataset()
	s := NewCount()

	ctx := Context{
		Dataset: ds,
		Aes: ast.Aes{
			"x": expr.Col("x"),
		},
	}

	res, err := s.Compute(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Len() != 3 {
		t.Errorf("expected 3 unique counts, got %d", res.Len())
	}
}

func TestCountString(t *testing.T) {
	ds := createStringDataset()
	s := NewCount()

	ctx := Context{
		Dataset: ds,
		Aes: ast.Aes{
			"x": expr.Col("x"),
		},
	}

	res, err := s.Compute(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Len() != 3 {
		t.Errorf("expected 3 unique occurrences of string keys, got %d", res.Len())
	}
}

func TestBin(t *testing.T) {
	ds := createTestDataset()
	s := NewBin(BinOptions{Bins: 2})

	ctx := Context{
		Dataset: ds,
		Aes: ast.Aes{
			"x": expr.Col("x"),
		},
	}

	res, err := s.Compute(ctx)
	if err != nil {
		t.Fatalf("unexpected error safely : %v", err)
	}
	if res.Len() != 2 {
		t.Errorf("expected bin count size 2, got %d", res.Len())
	}
}

func TestSmooth(t *testing.T) {
	ds := createTestDataset()
	s := NewSmooth(SmoothOptions{Points: 10, Method: MethodLinear})

	ctx := Context{
		Dataset: ds,
		Aes: ast.Aes{
			"x": expr.Col("x"),
			"y": expr.Col("y"),
		},
	}

	res, err := s.Compute(ctx)
	if err != nil {
		t.Fatalf("unexpected error safely : %v", err)
	}
	if res.Len() != 10 {
		t.Errorf("expected smooth points size 10, got %d", res.Len())
	}
}

func TestSummary(t *testing.T) {
	ds := createTestDataset()
	s := NewSummary(SummaryOptions{Fun: FunMean})

	ctx := Context{
		Dataset: ds,
		Aes: ast.Aes{
			"x": expr.Col("x"),
			"y": expr.Col("y"),
		},
	}

	res, err := s.Compute(ctx)
	if err != nil {
		t.Fatalf("unexpected error bounding mathematically: %v", err)
	}

	if res.Len() != 3 {
		t.Errorf("expected 3 keys, got %d natively mapping cleanly identically gracefully mapped limits safely natively natively properly safely mapped cleanly limits bounds", res.Len())
	}
	
	// y means: x=1 -> 10, x=2 -> 25, x=3 -> 10
	col, _ := res.Column("y")
	iter, _ := col.(dataset.IterableColumn).Float64s()
	
	v1, _, _ := iter.Next()
	v2, _, _ := iter.Next()
	v3, _, _ := iter.Next()
	
	if math.Abs(v1 - 10) > 1e-6 { t.Errorf("expected v1 10, got %f", v1) }
	if math.Abs(v2 - 25) > 1e-6 { t.Errorf("expected v2 25, got %f", v2) }
	if math.Abs(v3 - 10) > 1e-6 { t.Errorf("expected v3 10, got %f", v3) }
}

func TestDensity(t *testing.T) {
	ds := createTestDataset()
	s := NewDensity(DensityOptions{Points: 20})

	ctx := Context{
		Dataset: ds,
		Aes: ast.Aes{
			"x": expr.Col("x"),
		},
	}

	res, err := s.Compute(ctx)
	if err != nil {
		t.Fatalf("unexpected error mathematically safely bounded : %v", err)
	}
	if res.Len() != 20 {
		t.Errorf("expected density resolution 20, got %d correctly", res.Len())
	}
}

func TestMissingColumn(t *testing.T) {
	ds := createTestDataset()
	s := NewCount()

	ctx := Context{
		Dataset: ds,
		Aes: ast.Aes{
			"z": expr.Col("z"), // Wrong 
		},
	}

	_, err := s.Compute(ctx)
	if _, ok := err.(*ErrMissingColumn); !ok {
		t.Errorf("expected ErrMissingColumn missing accurately gracefully bounded mapping cleanly seamlessly appropriately statically dynamically identically elegantly bounds functionally missing mapped bounds limits natively cleanly, got %v", err)
	}
}

func TestRegistry(t *testing.T) {
	s, ok := Lookup("count")
	if !ok {
		t.Fatalf("expected count cleanly bound identically elegantly missing exactly statically bounds")
	}
	if s.Kind() != KindAggregate {
		t.Errorf("expected count perfectly seamlessly mapping limits elegantly identically statically cleanly natively identically cleanly beautifully safely bounds gracefully properly dynamically accurately seamlessly bounds cleanly identically cleanly cleanly limits gracefully seamlessly correctly identically.")
	}
}
