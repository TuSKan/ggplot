package arrow_test

import (
	"testing"

	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
	ar "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func TestTableColumn_FilterColumn_Empty(t *testing.T) {
	pool := memory.NewGoAllocator()
	b := array.NewFloat64Builder(pool)
	arr := b.NewFloat64Array()
	defer arr.Release()

	chk := ar.NewChunked(ar.PrimitiveTypes.Float64, []ar.Array{arr})
	defer chk.Release()

	col := arrow.NewTableColumn(chk)
	defer col.Release()

	filtered, err := col.FilterColumn([]bool{}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filtered.Len() != 0 {
		t.Errorf("expected length 0, got %d", filtered.Len())
	}
}

func TestTableColumn_MinMax_Nulls(t *testing.T) {
	pool := memory.NewGoAllocator()
	b := array.NewFloat64Builder(pool)
	b.AppendNull()
	b.Append(10)
	b.AppendNull()
	b.Append(-5)

	arr := b.NewFloat64Array()
	defer arr.Release()

	chk := ar.NewChunked(ar.PrimitiveTypes.Float64, []ar.Array{arr})
	defer chk.Release()

	col := arrow.NewTableColumn(chk)
	defer col.Release()

	min, err := col.Min()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if min != -5 {
		t.Errorf("expected min -5, got %v", min)
	}

	max, err := col.Max()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if max != 10 {
		t.Errorf("expected max 10, got %v", max)
	}
}

func TestTableColumn_MinMax_AllNulls(t *testing.T) {
	pool := memory.NewGoAllocator()
	b := array.NewFloat64Builder(pool)
	b.AppendNull()
	b.AppendNull()

	arr := b.NewFloat64Array()
	defer arr.Release()

	chk := ar.NewChunked(ar.PrimitiveTypes.Float64, []ar.Array{arr})
	defer chk.Release()

	col := arrow.NewTableColumn(chk)
	defer col.Release()

	min, _ := col.Min()
	if min != 0 {
		t.Errorf("expected min 0 for all nulls, got %v", min)
	}
}

func TestTableColumn_FilterColumn_MultiChunk(t *testing.T) {
	pool := memory.NewGoAllocator()
	b := array.NewFloat64Builder(pool)

	b.Append(1)
	b.Append(2)
	arr1 := b.NewFloat64Array()

	b.Append(3)
	b.Append(4)
	arr2 := b.NewFloat64Array()

	chk := ar.NewChunked(ar.PrimitiveTypes.Float64, []ar.Array{arr1, arr2})
	col := arrow.NewTableColumn(chk)

	mask := []bool{true, false, false, true}

	filtered, err := col.FilterColumn(mask, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filtered.Len() != 2 {
		t.Errorf("expected length 2, got %d", filtered.Len())
	}

	// Check Min/Max on Dictionary array
	min, _ := filtered.(interface{ Min() (float64, error) }).Min()
	max, _ := filtered.(interface{ Max() (float64, error) }).Max()

	if min != 1 {
		t.Errorf("expected min 1, got %v", min)
	}
	if max != 4 {
		t.Errorf("expected max 4, got %v", max)
	}
}
