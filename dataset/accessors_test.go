package dataset_test

import (
	"context"
	"math"
	"reflect"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
)

func newTestDataset(t *testing.T, cols ...dataset.AnyColumn) dataset.Dataset {
	t.Helper()

	eng := memory.NewEngine(context.Background())

	schemaFields := make([]dataset.Field, len(cols))
	for i, col := range cols {
		schemaFields[i] = dataset.Field{Name: col.Name(), Dtype: col.DType()}
	}

	schema := dataset.NewSchema(schemaFields...)

	tbl, err := eng.FromColumns(schema, cols...)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	return dataset.From(tbl)
}

func TestAccessors_ZeroCopy(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("x", []float64{1, 2, 3})
	ds := newTestDataset(t, col)

	vals, err := ds.Float64("x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate returned slice
	vals[0] = 99

	// Check if original column mutated (aliasing)
	origVals := col.(dataset.Column[float64]).Values() //nolint:errcheck,forcetypeassert // type guaranteed by test setup.
	if origVals[0] != 99 {
		t.Errorf("expected zero-copy alias, original column not mutated")
	}
}

func TestAccessors_OptClones(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("x", []float64{1, 2, 3})
	ds := newTestDataset(t, col)

	// dummy opt that does nothing
	noop := func(s []float64) []float64 { return s }

	vals, err := ds.Float64("x", noop)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate returned slice
	vals[0] = 99

	// Check if original column mutated
	origVals := col.(dataset.Column[float64]).Values() //nolint:errcheck,forcetypeassert // type guaranteed by test setup.
	if origVals[0] == 99 {
		t.Errorf("expected fresh slice with opts, but original column was mutated")
	}
}

func TestAccessors_Float64_Clean(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("x", []float64{1, math.NaN(), math.Inf(1), math.Inf(-1), 2})
	ds := newTestDataset(t, col)

	vals, err := ds.Float64("x", dataset.Clean)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []float64{1, 2}
	if !reflect.DeepEqual(vals, want) {
		t.Errorf("got %v, want %v", vals, want)
	}
}

func TestAccessors_Float64_FillNaN(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("x", []float64{1, math.NaN(), 2, math.Inf(1)})
	ds := newTestDataset(t, col)

	vals, err := ds.Float64("x", dataset.FillNaN(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []float64{1, 0, 2, math.Inf(1)}
	if !reflect.DeepEqual(vals, want) {
		t.Errorf("got %v, want %v", vals, want)
	}
}

func TestAccessors_Float64_Abs(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("x", []float64{-1.5, 2.0, -3.1})
	ds := newTestDataset(t, col)

	vals, err := ds.Float64("x", dataset.Abs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []float64{1.5, 2.0, 3.1}
	if !reflect.DeepEqual(vals, want) {
		t.Errorf("got %v, want %v", vals, want)
	}
}

func TestAccessors_Float64_Clamp(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("x", []float64{-2.0, 0.5, 2.0})
	ds := newTestDataset(t, col)

	vals, err := ds.Float64("x", dataset.Clamp(0.0, 1.0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []float64{0.0, 0.5, 1.0}
	if !reflect.DeepEqual(vals, want) {
		t.Errorf("got %v, want %v", vals, want)
	}
}

func TestAccessors_Float64_Sorted(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("x", []float64{3.0, 1.0, 2.0})
	ds := newTestDataset(t, col)

	vals, err := ds.Float64("x", dataset.Sorted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []float64{1.0, 2.0, 3.0}
	if !reflect.DeepEqual(vals, want) {
		t.Errorf("got %v, want %v", vals, want)
	}
}

func TestAccessors_Chaining(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("x", []float64{2, math.NaN(), 1, math.Inf(1)})
	ds := newTestDataset(t, col)

	t.Run("Clean then Sorted", func(t *testing.T) {
		t.Parallel()

		vals, err := ds.Float64("x", dataset.Clean, dataset.Sorted)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := []float64{1, 2}
		if !reflect.DeepEqual(vals, want) {
			t.Errorf("got %v, want %v", vals, want)
		}
	})

	t.Run("FillNaN then Abs", func(t *testing.T) {
		t.Parallel()

		col2 := eng.NewFloat64Column("y", []float64{-2, math.NaN(), -1})
		ds2 := newTestDataset(t, col2)

		vals, err := ds2.Float64("y", dataset.FillNaN(-5), dataset.Abs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := []float64{2, 5, 1}
		if !reflect.DeepEqual(vals, want) {
			t.Errorf("got %v, want %v", vals, want)
		}
	})
}

func TestAccessors_Errors(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("x", []float64{1, 2, 3})
	strCol := eng.NewStringColumn("lbl", []string{"a", "b", "c"})
	ds := newTestDataset(t, col, strCol)

	t.Run("Missing column", func(t *testing.T) {
		t.Parallel()

		_, err := ds.Float64("missing")
		if err == nil {
			t.Errorf("expected error for missing column")
		}
	})

	t.Run("Wrong type", func(t *testing.T) {
		t.Parallel()

		_, err := ds.Float64("lbl")
		if err == nil {
			t.Errorf("expected error for wrong typed column")
		}
	})

	t.Run("Pre-existing Dataset err", func(t *testing.T) {
		t.Parallel()

		dsErr := ds.Select("non_existent_column")
		// We'll simulate a pre-existing error using a bad column lookup
		_, err := dsErr.Float64("x")
		if err == nil {
			t.Errorf("expected error propagated from earlier dataset chain")
		}
	})
}

func TestAccessors_Int64(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewInt64Column("x", []int64{3, 1, 2, -10, 10})
	ds := newTestDataset(t, col)

	t.Run("Sorted inference", func(t *testing.T) {
		t.Parallel()

		vals, err := ds.Int64("x", dataset.Sorted)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := []int64{-10, 1, 2, 3, 10}
		if !reflect.DeepEqual(vals, want) {
			t.Errorf("got %v, want %v", vals, want)
		}
	})

	t.Run("Clamp explicit", func(t *testing.T) {
		t.Parallel()

		vals, err := ds.Int64("x", dataset.Clamp[int64](-5, 5))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := []int64{3, 1, 2, -5, 5}
		if !reflect.DeepEqual(vals, want) {
			t.Errorf("got %v, want %v", vals, want)
		}
	})
}

func TestAccessors_Strings(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewStringColumn("x", []string{"c", "a", "b"})
	ds := newTestDataset(t, col)

	vals, err := ds.Strings("x", dataset.Sorted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(vals, want) {
		t.Errorf("got %v, want %v", vals, want)
	}
}

func TestAccessors_Bools(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewBoolColumn("x", []bool{true, false, true})
	ds := newTestDataset(t, col)

	vals, err := ds.Bools("x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []bool{true, false, true}
	if !reflect.DeepEqual(vals, want) {
		t.Errorf("got %v, want %v", vals, want)
	}
}
