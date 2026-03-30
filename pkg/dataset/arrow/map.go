package arrow

import (
	"fmt"

	"github.com/TuSKan/ggplot/pkg/dataset"
	arrowtype "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// FromMap constructs a zero-copy Arrow dataset from standard map arrays.
// Supported map slice types include []float64, []int, []string, and []bool.
func FromMap(data map[string]any) (dataset.Dataset, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("dataset map is empty")
	}

	var length int
	first := true

	// 1. Verify structural limits.
	for _, v := range data {
		var l int
		switch arr := v.(type) {
		case []float64:
			l = len(arr)
		case []int:
			l = len(arr)
		case []string:
			l = len(arr)
		case []bool:
			l = len(arr)
		default:
			return nil, fmt.Errorf("unsupported column array type: %T", v)
		}

		if first {
			length = l
			first = false
		} else if l != length {
			return nil, fmt.Errorf("inconsistent column lengths: expected %d, got %d", length, l)
		}
	}

	if length == 0 {
		return nil, fmt.Errorf("cannot construct arrow table from empty slices")
	}

	pool := memory.NewGoAllocator()
	fields := make([]arrowtype.Field, len(data))
	cols := make([]arrowtype.Column, len(data))

	i := 0
	for name, v := range data {
		var field arrowtype.Field
		var chunk *arrowtype.Chunked

		switch arr := v.(type) {
		case []float64:
			b := array.NewFloat64Builder(pool)
			b.AppendValues(arr, nil)
			a := b.NewFloat64Array()
			b.Release()
			field = arrowtype.Field{Name: name, Type: arrowtype.PrimitiveTypes.Float64}
			chunk = arrowtype.NewChunked(arrowtype.PrimitiveTypes.Float64, []arrowtype.Array{a})
			a.Release()
		case []int:
			b := array.NewInt64Builder(pool)
			for _, val := range arr {
				b.Append(int64(val))
			}
			a := b.NewInt64Array()
			b.Release()
			field = arrowtype.Field{Name: name, Type: arrowtype.PrimitiveTypes.Int64}
			chunk = arrowtype.NewChunked(arrowtype.PrimitiveTypes.Int64, []arrowtype.Array{a})
			a.Release()
		case []string:
			b := array.NewStringBuilder(pool)
			b.AppendValues(arr, nil)
			a := b.NewStringArray()
			b.Release()
			field = arrowtype.Field{Name: name, Type: arrowtype.BinaryTypes.String}
			chunk = arrowtype.NewChunked(arrowtype.BinaryTypes.String, []arrowtype.Array{a})
			a.Release()
		case []bool:
			b := array.NewBooleanBuilder(pool)
			b.AppendValues(arr, nil)
			a := b.NewBooleanArray()
			b.Release()
			field = arrowtype.Field{Name: name, Type: arrowtype.FixedWidthTypes.Boolean}
			chunk = arrowtype.NewChunked(arrowtype.FixedWidthTypes.Boolean, []arrowtype.Array{a})
			a.Release()
		}

		fields[i] = field
		cols[i] = *arrowtype.NewColumn(field, chunk)
		i++
	}

	schema := arrowtype.NewSchema(fields, nil)
	table := array.NewTable(schema, cols, int64(length))

	// Transfer table ownership to Dataset wrapper.
	return NewTableDataset(table), nil
}
