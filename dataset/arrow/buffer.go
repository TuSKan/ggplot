package arrow

import (
	"fmt"

	"github.com/TuSKan/ggplot/dataset"
	arrowtype "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// columnMeta holds pre-allocation state for a single column in [Buffer].
type columnMeta struct {
	name     string
	dataType arrowtype.DataType
	buf      *memory.Buffer
	anySlice any
}

// Buffer is a zero-copy pre-allocator for high-throughput data ingestion.
// It allocates contiguous C-memory blocks and wraps them directly into
// immutable Arrow arrays without any copy.
//
// Usage:
//
//	buf := arrow.NewBuffer(1000)
//	xs := buf.Float64("x")
//	ys := buf.Float64("y")
//	labels := buf.String("label")
//	// Fill slices directly...
//	ds, err := buf.Build()
type Buffer struct {
	pool    memory.Allocator
	length  int
	columns []columnMeta
}

// NewBuffer creates a Buffer pre-allocating for exactly n rows.
func NewBuffer(n int) *Buffer {
	return &Buffer{
		pool:   memory.NewGoAllocator(),
		length: n,
	}
}

// Float64 allocates a raw C-memory block for float64 and returns a
// mutable slice backed by that block. Mutations are O(1) with zero
// garbage collection overhead.
func (b *Buffer) Float64(name string) []float64 {
	buf := memory.NewResizableBuffer(b.pool)
	buf.Resize(b.length * 8)
	slice := arrowtype.Float64Traits.CastFromBytes(buf.Bytes())
	b.columns = append(b.columns, columnMeta{
		name: name, dataType: arrowtype.PrimitiveTypes.Float64,
		buf: buf, anySlice: slice,
	})
	return slice
}

// Int64 allocates a raw C-memory block for int64 values.
func (b *Buffer) Int64(name string) []int64 {
	buf := memory.NewResizableBuffer(b.pool)
	buf.Resize(b.length * 8)
	slice := arrowtype.Int64Traits.CastFromBytes(buf.Bytes())
	b.columns = append(b.columns, columnMeta{
		name: name, dataType: arrowtype.PrimitiveTypes.Int64,
		buf: buf, anySlice: slice,
	})
	return slice
}

// Float32 allocates a raw C-memory block for float32 values.
func (b *Buffer) Float32(name string) []float32 {
	buf := memory.NewResizableBuffer(b.pool)
	buf.Resize(b.length * 4)
	slice := arrowtype.Float32Traits.CastFromBytes(buf.Bytes())
	b.columns = append(b.columns, columnMeta{
		name: name, dataType: arrowtype.PrimitiveTypes.Float32,
		buf: buf, anySlice: slice,
	})
	return slice
}

// Int32 allocates a raw C-memory block for int32 values.
func (b *Buffer) Int32(name string) []int32 {
	buf := memory.NewResizableBuffer(b.pool)
	buf.Resize(b.length * 4)
	slice := arrowtype.Int32Traits.CastFromBytes(buf.Bytes())
	b.columns = append(b.columns, columnMeta{
		name: name, dataType: arrowtype.PrimitiveTypes.Int32,
		buf: buf, anySlice: slice,
	})
	return slice
}

// String returns a standard Go string slice. Strings cannot be zero-copy
// (they're variable-length) so they are copied into an Arrow StringArray
// during Build.
func (b *Buffer) String(name string) []string {
	slice := make([]string, b.length)
	b.columns = append(b.columns, columnMeta{
		name: name, dataType: arrowtype.BinaryTypes.String,
		anySlice: slice,
	})
	return slice
}

// Bool returns a standard Go boolean slice. Arrow bit-packs booleans,
// so they are copied during Build.
func (b *Buffer) Bool(name string) []bool {
	slice := make([]bool, b.length)
	b.columns = append(b.columns, columnMeta{
		name: name, dataType: arrowtype.FixedWidthTypes.Boolean,
		anySlice: slice,
	})
	return slice
}

// Build finalizes the pre-allocated buffers into an immutable [dataset.Dataset]
// backed by an Arrow Table. Numeric columns are zero-copy; strings and bools
// are copied into Arrow arrays.
func (b *Buffer) Build() (dataset.Dataset, error) {
	if len(b.columns) == 0 {
		return nil, fmt.Errorf("arrow: buffer contains no columns")
	}

	fields := make([]arrowtype.Field, len(b.columns))
	cols := make([]arrowtype.Column, len(b.columns))

	for i, col := range b.columns {
		fields[i] = arrowtype.Field{Name: col.name, Type: col.dataType}

		var chunk *arrowtype.Chunked

		if col.buf != nil {
			// Zero-copy: wrap the buffer directly into an Arrow array.
			data := array.NewData(
				col.dataType,
				b.length,
				[]*memory.Buffer{nil, col.buf}, // nil bitmap = no nulls
				nil, 0, 0,
			)

			var arr arrowtype.Array
			switch col.dataType.ID() {
			case arrowtype.FLOAT64:
				arr = array.NewFloat64Data(data)
			case arrowtype.INT64:
				arr = array.NewInt64Data(data)
			case arrowtype.FLOAT32:
				arr = array.NewFloat32Data(data)
			case arrowtype.INT32:
				arr = array.NewInt32Data(data)
			}

			chunk = arrowtype.NewChunked(col.dataType, []arrowtype.Array{arr})
			arr.Release()
			data.Release()
			col.buf.Release()
		} else {
			// Variable-length types require a copy through builders.
			switch col.dataType.ID() {
			case arrowtype.STRING:
				builder := array.NewStringBuilder(b.pool)
				builder.AppendValues(col.anySlice.([]string), nil)
				arr := builder.NewStringArray()
				builder.Release()
				chunk = arrowtype.NewChunked(col.dataType, []arrowtype.Array{arr})
				arr.Release()

			case arrowtype.BOOL:
				builder := array.NewBooleanBuilder(b.pool)
				builder.AppendValues(col.anySlice.([]bool), nil)
				arr := builder.NewBooleanArray()
				builder.Release()
				chunk = arrowtype.NewChunked(col.dataType, []arrowtype.Array{arr})
				arr.Release()
			}
		}

		cols[i] = *arrowtype.NewColumn(fields[i], chunk)
	}

	schema := arrowtype.NewSchema(fields, nil)
	table := array.NewTable(schema, cols, int64(b.length))

	return NewTableDataset(table), nil
}
