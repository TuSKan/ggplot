package dataset_test

import (
	"testing"

	"github.com/TuSKan/ggplot/internal/adapter/arrow"
	"github.com/TuSKan/ggplot/internal/dataset"

	arrowtype "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func TestDatasetSlice(t *testing.T) {
	pool := memory.NewGoAllocator()
	b := array.NewFloat64Builder(pool)
	defer b.Release()

	b.AppendValues([]float64{1.0, 2.0, 3.0, 4.0, 5.0}, nil)
	arr := b.NewFloat64Array()
	defer arr.Release()

	field := arrowtype.Field{Name: "val", Type: arrowtype.PrimitiveTypes.Float64}
	schema := arrowtype.NewSchema([]arrowtype.Field{field}, nil)
	table := array.NewTable(schema, []arrowtype.Column{*arrowtype.NewColumn(field, arrowtype.NewChunked(arrowtype.PrimitiveTypes.Float64, []arrowtype.Array{arr}))}, 5)
	defer table.Release()

	ds := arrow.NewTableDataset(table)

	// Test basic slicing delegation
	sliced := dataset.Slice(ds, 1, 4)
	if sliced.Len() != 3 {
		t.Errorf("expected len 3, got %d", sliced.Len())
	}

	col, err := sliced.Column("val")
	if err != nil {
		t.Fatal(err)
	}
	if col.Len() != 3 {
		t.Errorf("expected col len 3, got %d", col.Len())
	}
}

func TestDatasetFilter(t *testing.T) {
	pool := memory.NewGoAllocator()
	b := array.NewFloat64Builder(pool)
	defer b.Release()

	b.AppendValues([]float64{10, 20, 30, 40}, nil)
	arr := b.NewFloat64Array()
	defer arr.Release()

	schema := arrowtype.NewSchema([]arrowtype.Field{
		{Name: "val", Type: arrowtype.PrimitiveTypes.Float64},
	}, nil)

	colChunk := arrowtype.NewChunked(schema.Field(0).Type, []arrowtype.Array{arr})
	table := array.NewTable(schema, []arrowtype.Column{*arrowtype.NewColumn(schema.Field(0), colChunk)}, 4)
	defer table.Release()

	ds := arrow.NewTableDataset(table)
	mask := []bool{true, false, true, false}

	filtered := dataset.Filter(ds, mask)
	if filtered.Len() != 2 {
		t.Errorf("expected 2 elements, got %d", filtered.Len())
	}

	col, err := filtered.Column("val")
	if err != nil {
		t.Fatal(err)
	}

	// Assert native filtering has occurred
	if col.Len() != 2 {
		t.Errorf("expected 2 remaining rows in column, got %d", col.Len())
	}
}

func TestMinMax_Nulls(t *testing.T) {
	pool := memory.NewGoAllocator()
	b := array.NewFloat64Builder(pool)
	defer b.Release()

	// Contains Nulls
	b.AppendValues([]float64{10, 5, 0, 100}, []bool{true, true, false, true})
	arr := b.NewFloat64Array()
	defer arr.Release()

	c := arrow.NewTableColumn(arrowtype.NewChunked(arrowtype.PrimitiveTypes.Float64, []arrowtype.Array{arr}))

	min, err := dataset.Min(c)
	if err != nil {
		t.Fatal(err)
	}
	if min != 5 {
		t.Errorf("expected min 5 (0 is null), got %f", min)
	}

	max, err := dataset.Max(c)
	if err != nil {
		t.Fatal(err)
	}
	if max != 100 {
		t.Errorf("expected max 100, got %f", max)
	}
}

func TestEmptyDataset(t *testing.T) {
	pool := memory.NewGoAllocator()
	b := array.NewFloat64Builder(pool)
	defer b.Release()

	arr := b.NewFloat64Array()
	defer arr.Release()
	c := arrow.NewTableColumn(arrowtype.NewChunked(arrowtype.PrimitiveTypes.Float64, []arrowtype.Array{arr}))

	_, err := dataset.Min(c)
	if err == nil {
		t.Errorf("expected error when getting min of empty dataset")
	}
}
