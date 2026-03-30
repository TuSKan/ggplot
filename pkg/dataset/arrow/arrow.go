package arrow

import (
	"math"

	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// TableDataset wraps an Apache Arrow Table into a Dataset.
type TableDataset struct {
	table arrow.Table // Use arrow.Table directly since Table is an interface in apache arrow
}

var _ dataset.Dataset = (*TableDataset)(nil)
var _ dataset.NativeSliceProvider = (*TableDataset)(nil)

// NewTableDataset creates a Dataset from an Arrow Table.
func NewTableDataset(t arrow.Table) *TableDataset {
	t.Retain()
	return &TableDataset{table: t}
}

func (d *TableDataset) Release() {
	if d.table != nil {
		d.table.Release()
	}
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

func (d *TableDataset) SliceDataset(i, j int) dataset.Dataset {
	schema := d.table.Schema()
	cols := make([]arrow.Column, d.table.NumCols())
	for idx := 0; idx < int(d.table.NumCols()); idx++ {
		origCol := d.table.Column(idx)
		sliced := array.NewChunkedSlice(origCol.Data(), int64(i), int64(j))
		cols[idx] = *arrow.NewColumn(schema.Field(idx), sliced)
	}
	slicedTable := array.NewTable(schema, cols, int64(j-i))
	return NewTableDataset(slicedTable)
}

// TableColumn wraps a Chunked array.
type TableColumn struct {
	chunked *arrow.Chunked
}

var _ dataset.Column = (*TableColumn)(nil)
var _ dataset.NativeColumnSliceProvider = (*TableColumn)(nil)
var _ dataset.NativeFilterProvider = (*TableColumn)(nil)
var _ dataset.Aggregator = (*TableColumn)(nil)

func NewTableColumn(c *arrow.Chunked) *TableColumn {
	c.Retain()
	return &TableColumn{chunked: c}
}

func (c *TableColumn) Release() {
	if c.chunked != nil {
		c.chunked.Release()
	}
}

func (c *TableColumn) Len() int {
	return c.chunked.Len()
}

func (c *TableColumn) SliceColumn(i, j int) dataset.Column {
	sliced := array.NewChunkedSlice(c.chunked, int64(i), int64(j))
	return NewTableColumn(sliced)
}

func (c *TableColumn) FilterColumn(mask []bool, count int) (dataset.Column, error) {
	if count == 0 {
		empty := arrow.NewChunked(c.chunked.DataType(), []arrow.Array{})
		defer empty.Release()
		return NewTableColumn(empty), nil
	}

	chunks := c.chunked.Chunks()
	filteredChunks := make([]arrow.Array, 0, len(chunks))
	pool := memory.NewGoAllocator()

	var baseType arrow.DataType
	if dt, isDict := c.chunked.DataType().(*arrow.DictionaryType); isDict {
		baseType = dt.ValueType
	} else {
		baseType = c.chunked.DataType()
	}
	outType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: baseType}

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
			continue // skip empty chunks
		}

		var dictVals arrow.Array
		var origIndices []int32
		if dictArr, ok := chk.(*array.Dictionary); ok {
			dictVals = dictArr.Dictionary()
			origIndices = make([]int32, chkLen)
			for i := 0; i < chkLen; i++ {
				// GetValueIndex handles nulls as 0 or so, we assume validity masks later
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
		empty := arrow.NewChunked(outType, []arrow.Array{})
		defer empty.Release()
		return NewTableColumn(empty), nil
	}

	res := arrow.NewChunked(outType, filteredChunks)
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

// computeMinMax iterates Arrow native arrays directly, circumventing unboxing.
func computeMinMax(chk *arrow.Chunked, findMin bool) (float64, error) {
	if chk.Len() == 0 || chk.NullN() == chk.Len() {
		return 0, nil
	}
	var res float64
	if findMin {
		res = math.MaxFloat64
	} else {
		res = -math.MaxFloat64
	}

	for _, c := range chk.Chunks() {
		switch arr := c.(type) {
		case *array.Float64:
			if arr.NullN() == 0 {
				for _, v := range arr.Float64Values() {
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
			} else {
				for i := 0; i < arr.Len(); i++ {
					if arr.IsValid(i) {
						v := arr.Value(i)
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
				}
			}
		case *array.Int64:
			for i := 0; i < arr.Len(); i++ {
				if arr.IsValid(i) {
					v := float64(arr.Value(i))
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
			}
		// Dictionary fallback support
		case *array.Dictionary:
			// Fallback: iterate over indices and fetch values
			dictVals := arr.Dictionary()
			for i := 0; i < arr.Len(); i++ {
				if arr.IsValid(i) {
					idx := arr.GetValueIndex(i)
					if dictVals.IsValid(idx) {
						var v float64
						switch vArr := dictVals.(type) {
						case *array.Float64:
							v = vArr.Value(idx)
						case *array.Int64:
							v = float64(vArr.Value(idx))
						}
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
				}
			}
		}
	}
	return res, nil
}
