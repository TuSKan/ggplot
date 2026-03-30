package arrow_test

import (
	"testing"

	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
	ar "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func BenchmarkTableColumn_Min(b *testing.B) {
	pool := memory.NewGoAllocator()
	builder := array.NewFloat64Builder(pool)
	for i := 0; i < 10000; i++ {
		if i%10 == 0 {
			builder.AppendNull()
		} else {
			builder.Append(float64(i))
		}
	}
	arr := builder.NewFloat64Array()
	chk := ar.NewChunked(ar.PrimitiveTypes.Float64, []ar.Array{arr})
	col := arrow.NewTableColumn(chk)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = col.Min()
	}
}

func BenchmarkTableColumn_FilterColumn(b *testing.B) {
	pool := memory.NewGoAllocator()
	builder := array.NewFloat64Builder(pool)
	for i := 0; i < 10000; i++ {
		builder.Append(float64(i))
	}
	arr := builder.NewFloat64Array()
	chk := ar.NewChunked(ar.PrimitiveTypes.Float64, []ar.Array{arr})
	col := arrow.NewTableColumn(chk)

	mask := make([]bool, 10000)
	count := 0
	for i := 0; i < 10000; i++ {
		if i%2 == 0 {
			mask[i] = true
			count++
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filtered, _ := col.FilterColumn(mask, count)
		if filtered != nil {
			// Ensure cleanup during tight loops if we were retaining,
			// though typical GC covers the wrapper interface
		}
	}
}
