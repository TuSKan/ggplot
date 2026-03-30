package arrow

import (
	"fmt"

	"github.com/TuSKan/ggplot/pkg/dataset"
	arrowtype "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

type columnMeta struct {
	name     string
	dataType arrowtype.DataType
	buf      *memory.Buffer // Zero-copy C-memory buffer for primitives
	anySlice any            // General slice holding user's strings/bools or the cast primitive slice
}

// Buffer acts as an elegant pre-allocator for extreme zero-copy dataset ingestion.
type Buffer struct {
	pool    memory.Allocator
	length  int
	columns []columnMeta
}

// NewBuffer asks the system to pre-allocate memory chunks for exactly 'length' entries.
func NewBuffer(length int) *Buffer {
	return &Buffer{
		pool:    memory.NewGoAllocator(),
		length:  length,
		columns: []columnMeta{},
	}
}

// Float64 allocates a new raw C-memory block instantly and binds it
// to a Go slice for direct O(1) mutations without garbage collection overhead.
func (b *Buffer) Float64(name string) []float64 {
	buf := memory.NewResizableBuffer(b.pool)
	buf.Resize(b.length * 8)
	slice := arrowtype.Float64Traits.CastFromBytes(buf.Bytes())

	b.columns = append(b.columns, columnMeta{
		name:     name,
		dataType: arrowtype.PrimitiveTypes.Float64,
		buf:      buf,
		anySlice: slice,
	})
	return slice
}

// Int64 reserves 8-byte integers.
func (b *Buffer) Int64(name string) []int64 {
	buf := memory.NewResizableBuffer(b.pool)
	buf.Resize(b.length * 8)
	slice := arrowtype.Int64Traits.CastFromBytes(buf.Bytes())

	b.columns = append(b.columns, columnMeta{
		name:     name,
		dataType: arrowtype.PrimitiveTypes.Int64,
		buf:      buf,
		anySlice: slice,
	})
	return slice
}

// Float32 maps 4-byte static buffers.
func (b *Buffer) Float32(name string) []float32 {
	buf := memory.NewResizableBuffer(b.pool)
	buf.Resize(b.length * 4)
	slice := arrowtype.Float32Traits.CastFromBytes(buf.Bytes())

	b.columns = append(b.columns, columnMeta{
		name:     name,
		dataType: arrowtype.PrimitiveTypes.Float32,
		buf:      buf,
		anySlice: slice,
	})
	return slice
}

// Int32 maps 4-byte static integer buffers.
func (b *Buffer) Int32(name string) []int32 {
	buf := memory.NewResizableBuffer(b.pool)
	buf.Resize(b.length * 4)
	slice := arrowtype.Int32Traits.CastFromBytes(buf.Bytes())

	b.columns = append(b.columns, columnMeta{
		name:     name,
		dataType: arrowtype.PrimitiveTypes.Int32,
		buf:      buf,
		anySlice: slice,
	})
	return slice
}

// String returns a standard -managed variable-length Go array that
// copies itself silently into an Arrow String Builder during Build().
func (b *Buffer) String(name string) []string {
	slice := make([]string, b.length)
	b.columns = append(b.columns, columnMeta{
		name:     name,
		dataType: arrowtype.BinaryTypes.String,
		anySlice: slice,
	})
	return slice
}

// Bool returns a standard Go boolean array.
// Bools cannot be zero-copy indexed as a slice since Arrow bit-packs them.
func (b *Buffer) Bool(name string) []bool {
	slice := make([]bool, b.length)
	b.columns = append(b.columns, columnMeta{
		name:     name,
		dataType: arrowtype.FixedWidthTypes.Boolean,
		anySlice: slice,
	})
	return slice
}

// Build finalizes the allocations wrapping the C-memory immediately into Immutable Datasets.
func (b *Buffer) Build() (dataset.Dataset, error) {
	if len(b.columns) == 0 {
		return nil, fmt.Errorf("buffer contains no structured columns")
	}

	fields := make([]arrowtype.Field, len(b.columns))
	cols := make([]arrowtype.Column, len(b.columns))

	for i, col := range b.columns {
		fields[i] = arrowtype.Field{Name: col.name, Type: col.dataType}

		var chunk *arrowtype.Chunked

		if col.buf != nil {
			// Zero-copy realization. Data wraps the buffer immediately with absolutely no iteration.
			data := array.NewData(
				col.dataType,
				b.length,
				[]*memory.Buffer{nil, col.buf}, // nil bitmap implies no null values
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
			data.Release() // Release our initial ref, the chunk retains it

			// We can release our hold on the buffer since Data retains it.
			col.buf.Release()
		} else {
			// Hybrid graceful variable-length / bit-packed structs resolving memory.
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
