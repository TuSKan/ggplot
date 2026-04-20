// Package arrow provides a zero-copy Apache Arrow adapter for the dataset
// package. It wraps Arrow tables and columns behind the [dataset.Dataset]
// and [dataset.Column] interfaces, enabling the Grammar of Graphics pipeline
// to operate directly on Arrow memory without materialization.
//
// # Usage
//
//	import "github.com/TuSKan/ggplot/dataset/arrow"
//
//	ds := arrow.NewTableDataset(myArrowTable)
//	defer ds.Close()
//
//	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
//	    Layer(geom.Point()).
//	    Save("output.png", 800, 600)
//
// # Zero-Copy Buffer
//
// For data ingestion with zero GC overhead, use [Buffer]:
//
//	buf := arrow.NewBuffer(1000)
//	xs := buf.Float64("x")
//	ys := buf.Float64("y")
//	// Fill xs, ys directly...
//	ds, _ := buf.Build()
package arrow

import (
	"fmt"
	"math"

	"github.com/TuSKan/ggplot/dataset"
	arrowtype "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// TableDataset wraps an Apache Arrow Table as a [dataset.Dataset].
// It retains the table on creation and releases it on Close.
type TableDataset struct {
	table arrowtype.Table
}

// NewTableDataset creates a Dataset from an Arrow Table.
// The table is retained; call [TableDataset.Close] when done.
func NewTableDataset(t arrowtype.Table) *TableDataset {
	t.Retain()
	return &TableDataset{table: t}
}

func (d *TableDataset) Columns() []string {
	schema := d.table.Schema()
	fields := schema.Fields()
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return names
}

func (d *TableDataset) Column(name string) (dataset.Column, error) {
	schema := d.table.Schema()
	indices := schema.FieldIndices(name)
	if len(indices) == 0 {
		return nil, &dataset.ErrColumnNotFound{Name: name}
	}
	col := d.table.Column(indices[0])
	return NewTableColumn(col.Data()), nil
}

func (d *TableDataset) Len() int {
	return int(d.table.NumRows())
}

// Close releases the underlying Arrow table.
func (d *TableDataset) Close() error {
	if d.table != nil {
		d.table.Release()
		d.table = nil
	}
	return nil
}

func (d *TableDataset) SliceDataset(i, j int) dataset.Dataset {
	schema := d.table.Schema()
	cols := make([]arrowtype.Column, d.table.NumCols())
	for idx := 0; idx < int(d.table.NumCols()); idx++ {
		origCol := d.table.Column(idx)
		sliced := array.NewChunkedSlice(origCol.Data(), int64(i), int64(j))
		cols[idx] = *arrowtype.NewColumn(schema.Field(idx), sliced)
	}
	slicedTable := array.NewTable(schema, cols, int64(j-i))
	return NewTableDataset(slicedTable)
}

// --- TableColumn ---

// TableColumn wraps an Arrow Chunked array as a [dataset.Column].
type TableColumn struct {
	chunked *arrowtype.Chunked
}

// NewTableColumn creates a Column from a Chunked array.
func NewTableColumn(c *arrowtype.Chunked) *TableColumn {
	c.Retain()
	return &TableColumn{chunked: c}
}

// Close releases the underlying chunked data.
func (c *TableColumn) Close() error {
	if c.chunked != nil {
		c.chunked.Release()
		c.chunked = nil
	}
	return nil
}

func (c *TableColumn) Len() int { return c.chunked.Len() }

func (c *TableColumn) DType() dataset.DType {
	switch c.chunked.DataType().ID() {
	case arrowtype.FLOAT64, arrowtype.FLOAT32:
		return dataset.DTypeFloat64
	case arrowtype.INT64, arrowtype.INT32, arrowtype.INT16, arrowtype.INT8:
		return dataset.DTypeInt64
	case arrowtype.STRING, arrowtype.LARGE_STRING:
		return dataset.DTypeString
	case arrowtype.BOOL:
		return dataset.DTypeBool
	case arrowtype.DICTIONARY:
		dt := c.chunked.DataType().(*arrowtype.DictionaryType)
		switch dt.ValueType.ID() {
		case arrowtype.STRING, arrowtype.LARGE_STRING:
			return dataset.DTypeString
		case arrowtype.FLOAT64, arrowtype.FLOAT32:
			return dataset.DTypeFloat64
		default:
			return dataset.DTypeString
		}
	default:
		return dataset.DTypeUnknown
	}
}

func (c *TableColumn) SliceColumn(i, j int) dataset.Column {
	sliced := array.NewChunkedSlice(c.chunked, int64(i), int64(j))
	return NewTableColumn(sliced)
}

func (c *TableColumn) FilterColumn(mask []bool, count int) (dataset.Column, error) {
	if count == 0 {
		empty := arrowtype.NewChunked(c.chunked.DataType(), []arrowtype.Array{})
		defer empty.Release()
		return NewTableColumn(empty), nil
	}

	chunks := c.chunked.Chunks()
	filteredChunks := make([]arrowtype.Array, 0, len(chunks))
	pool := memory.NewGoAllocator()

	// Determine base type for dictionary.
	var baseType arrowtype.DataType
	if dt, isDict := c.chunked.DataType().(*arrowtype.DictionaryType); isDict {
		baseType = dt.ValueType
	} else {
		baseType = c.chunked.DataType()
	}
	outType := &arrowtype.DictionaryType{IndexType: arrowtype.PrimitiveTypes.Int32, ValueType: baseType}

	offset := 0
	for _, chk := range chunks {
		chkLen := chk.Len()
		if offset+chkLen > len(mask) {
			break
		}
		chkMask := mask[offset : offset+chkLen]
		offset += chkLen

		keep := 0
		for _, b := range chkMask {
			if b {
				keep++
			}
		}
		if keep == 0 {
			continue
		}

		var dictVals arrowtype.Array
		var origIndices []int32
		if dictArr, ok := chk.(*array.Dictionary); ok {
			dictVals = dictArr.Dictionary()
			origIndices = make([]int32, chkLen)
			for i := 0; i < chkLen; i++ {
				origIndices[i] = int32(dictArr.GetValueIndex(i))
			}
		} else {
			dictVals = chk
		}

		b := array.NewInt32Builder(pool)
		b.Reserve(keep)
		for i, m := range chkMask {
			if m {
				if origIndices != nil {
					b.Append(origIndices[i])
				} else {
					b.Append(int32(i))
				}
			}
		}
		indices := b.NewInt32Array()

		dict := array.NewDictionaryArray(outType, indices, dictVals)
		filteredChunks = append(filteredChunks, dict)
		indices.Release()
	}

	if len(filteredChunks) == 0 {
		empty := arrowtype.NewChunked(outType, []arrowtype.Array{})
		defer empty.Release()
		return NewTableColumn(empty), nil
	}

	res := arrowtype.NewChunked(outType, filteredChunks)
	for _, fc := range filteredChunks {
		fc.Release()
	}

	col := NewTableColumn(res)
	res.Release()
	return col, nil
}

func (c *TableColumn) Min() (float64, error) {
	return computeMinMax(c.chunked, true)
}

func (c *TableColumn) Max() (float64, error) {
	return computeMinMax(c.chunked, false)
}

// computeMinMax iterates Arrow native arrays directly for O(n) min/max.
func computeMinMax(chk *arrowtype.Chunked, findMin bool) (float64, error) {
	if chk.Len() == 0 || chk.NullN() == chk.Len() {
		return 0, fmt.Errorf("dataset: empty or all-null column")
	}
	var res float64
	if findMin {
		res = math.MaxFloat64
	} else {
		res = -math.MaxFloat64
	}

	cmp := func(v float64) {
		if findMin {
			if v < res {
				res = v
			}
		} else {
			if v > res {
				res = v
			}
		}
	}

	for _, c := range chk.Chunks() {
		switch arr := c.(type) {
		case *array.Float64:
			if arr.NullN() == 0 {
				for _, v := range arr.Float64Values() {
					cmp(v)
				}
			} else {
				for i := 0; i < arr.Len(); i++ {
					if arr.IsValid(i) {
						cmp(arr.Value(i))
					}
				}
			}
		case *array.Float32:
			for i := 0; i < arr.Len(); i++ {
				if arr.IsValid(i) {
					cmp(float64(arr.Value(i)))
				}
			}
		case *array.Int64:
			for i := 0; i < arr.Len(); i++ {
				if arr.IsValid(i) {
					cmp(float64(arr.Value(i)))
				}
			}
		case *array.Int32:
			for i := 0; i < arr.Len(); i++ {
				if arr.IsValid(i) {
					cmp(float64(arr.Value(i)))
				}
			}
		case *array.Dictionary:
			dictVals := arr.Dictionary()
			for i := 0; i < arr.Len(); i++ {
				if arr.IsValid(i) {
					idx := arr.GetValueIndex(i)
					if dictVals.IsValid(idx) {
						switch vArr := dictVals.(type) {
						case *array.Float64:
							cmp(vArr.Value(idx))
						case *array.Int64:
							cmp(float64(vArr.Value(idx)))
						case *array.Float32:
							cmp(float64(vArr.Value(idx)))
						case *array.Int32:
							cmp(float64(vArr.Value(idx)))
						}
					}
				}
			}
		}
	}
	return res, nil
}

// --- Compile-time interface checks ---

var (
	_ dataset.Dataset             = (*TableDataset)(nil)
	_ dataset.NativeSliceProvider = (*TableDataset)(nil)
	_ dataset.Closer              = (*TableDataset)(nil)

	_ dataset.Column                    = (*TableColumn)(nil)
	_ dataset.IterableColumn            = (*TableColumn)(nil)
	_ dataset.Aggregator                = (*TableColumn)(nil)
	_ dataset.NativeColumnSliceProvider = (*TableColumn)(nil)
	_ dataset.NativeFilterProvider      = (*TableColumn)(nil)
)
