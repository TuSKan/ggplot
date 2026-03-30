package dataset_test

import (
	"testing"

	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"

	arrowtype "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func createBenchDataset(n int) dataset.Dataset {
	pool := memory.NewGoAllocator()
	b := array.NewFloat64Builder(pool)
	defer b.Release()

	b.Reserve(n)
	for i := 0; i < n; i++ {
		b.Append(float64(i))
	}
	arr := b.NewFloat64Array()

	schema := arrowtype.NewSchema([]arrowtype.Field{
		{Name: "val", Type: arrowtype.PrimitiveTypes.Float64},
	}, nil)

	colChunk := arrowtype.NewChunked(schema.Field(0).Type, []arrowtype.Array{arr})
	table := array.NewTable(schema, []arrowtype.Column{*arrowtype.NewColumn(schema.Field(0), colChunk)}, int64(n))
	return arrow.NewTableDataset(table)
}

func BenchmarkDatasetFilter(b *testing.B) {
	ds := createBenchDataset(1_000_000)
	// Create boolean mask allocating 50%
	mask := make([]bool, 1_000_000)
	for i := range mask {
		mask[i] = i%2 == 0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Filter should be lazy and basically instantly return its proxy struct
		_ = dataset.Filter(ds, mask)
	}
}

func BenchmarkDatasetMin(b *testing.B) {
	ds := createBenchDataset(1_000_000)
	col, _ := ds.Column("val")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dataset.Min(col)
	}
}
