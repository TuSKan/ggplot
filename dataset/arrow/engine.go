// Package arrow provides an Apache Arrow-backed compute engine for the
// dataset package. It implements [dataset.ColumnFactory], [dataset.BuilderFactory],
// [dataset.Aggregator], and [dataset.Caster] using Arrow arrays and
// arrow/math SIMD kernels.
//
// Usage:
//
//	eng := arrow.NewEngine(memory.DefaultAllocator)
//	f := eng.(dataset.ColumnFactory)
//	ds, _ := f.FromColumns(
//	    dataset.NewSchema(dataset.FloatCol("x"), dataset.StringCol("label")),
//	    f.NewFloat64Column("x", []float64{1, 2, 3}),
//	    f.NewStringColumn("label", []string{"a", "b", "c"}),
//	)
package arrow

import (
	"context"
	"fmt"

	"github.com/TuSKan/ggplot/dataset"
	simd "github.com/TuSKan/ggplot/dataset/compute"
	dsort "github.com/TuSKan/ggplot/dataset/sort"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/compute"
	"github.com/apache/arrow-go/v18/arrow/math"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// Engine is the Arrow compute backend.
type Engine struct {
	alloc memory.Allocator
}

// NewEngine creates an Arrow engine with the given memory allocator.
func NewEngine(alloc memory.Allocator) *Engine {
	return &Engine{alloc: alloc}
}

// Name returns "arrow".
func (e *Engine) Name() string { return "arrow" }

// Alloc returns the engine's memory allocator.
func (e *Engine) Alloc() memory.Allocator { return e.alloc }

// --- ColumnFactory ---

func (e *Engine) NewFloat64Column(name string, data []float64) dataset.AnyColumn {
	b := array.NewFloat64Builder(e.alloc)
	defer b.Release()
	b.AppendValues(data, nil)
	return &arrowFloat64Column{name: name, arr: b.NewFloat64Array()}
}

func (e *Engine) NewInt64Column(name string, data []int64) dataset.AnyColumn {
	b := array.NewInt64Builder(e.alloc)
	defer b.Release()
	b.AppendValues(data, nil)
	return &arrowInt64Column{name: name, arr: b.NewInt64Array()}
}

func (e *Engine) NewStringColumn(name string, data []string) dataset.AnyColumn {
	b := array.NewStringBuilder(e.alloc)
	defer b.Release()
	for _, s := range data {
		b.Append(s)
	}
	return &arrowStringColumn{name: name, arr: b.NewStringArray()}
}

func (e *Engine) NewBoolColumn(name string, data []bool) dataset.AnyColumn {
	b := array.NewBooleanBuilder(e.alloc)
	defer b.Release()
	b.AppendValues(data, nil)
	return &arrowBoolColumn{name: name, arr: b.NewBooleanArray()}
}

func (e *Engine) NewTimestampColumn(name string, data []int64) dataset.AnyColumn {
	b := array.NewInt64Builder(e.alloc)
	defer b.Release()
	b.AppendValues(data, nil)
	return &arrowInt64Column{name: name, arr: b.NewInt64Array(), dtype: dataset.DTypeTimestamp}
}

func (e *Engine) FromColumns(schema *dataset.Schema, cols ...dataset.AnyColumn) (dataset.Table, error) {
	if len(cols) == 0 {
		return nil, fmt.Errorf("arrow: FromColumns requires at least one column")
	}
	length := cols[0].Len()
	columns := make(map[string]dataset.AnyColumn, len(cols))
	for _, col := range cols {
		if col.Len() != length {
			return nil, fmt.Errorf("arrow: column %q has length %d, expected %d",
				col.Name(), col.Len(), length)
		}
		columns[col.Name()] = col
	}
	return &arrowDataset{schema: schema, columns: columns, length: length, engine: e}, nil
}

// --- BuilderFactory ---

func (e *Engine) NewBuilder(schema *dataset.Schema) dataset.Builder {
	b := &arrowBuilder{eng: e, schema: schema}
	b.builders = make(map[string]any, schema.NumFields())
	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		switch f.Dtype {
		case dataset.DTypeFloat64:
			b.builders[f.Name] = &arrowFloat64Appender{b: array.NewFloat64Builder(e.alloc)}
		case dataset.DTypeInt64, dataset.DTypeTimestamp:
			b.builders[f.Name] = &arrowInt64Appender{b: array.NewInt64Builder(e.alloc)}
		case dataset.DTypeString:
			b.builders[f.Name] = &arrowStringAppender{b: array.NewStringBuilder(e.alloc)}
		case dataset.DTypeBool:
			b.builders[f.Name] = &arrowBoolAppender{b: array.NewBooleanBuilder(e.alloc)}
		}
	}
	return b
}

// --- Aggregator (returns AnyColumn, preserves input type) ---

func (e *Engine) Sum(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *arrowFloat64Column:
		// Arrow official SIMD sum
		s := math.Float64.Sum(c.arr)
		return e.NewFloat64Column(c.name, []float64{s}), nil
	case *arrowInt64Column:
		s := math.Int64.Sum(c.arr)
		return e.NewInt64Column(c.name, []int64{s}), nil
	default:
		return nil, fmt.Errorf("arrow: Sum not supported for %T", col)
	}
}

func (e *Engine) Mean(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	if col.Len() == 0 {
		return nil, fmt.Errorf("arrow: Mean of empty column")
	}
	switch c := col.(type) {
	case *arrowFloat64Column:
		s := math.Float64.Sum(c.arr)
		return e.NewFloat64Column(c.name, []float64{s / float64(c.arr.Len())}), nil
	case *arrowInt64Column:
		s := math.Int64.Sum(c.arr)
		return e.NewFloat64Column(c.name, []float64{float64(s) / float64(c.arr.Len())}), nil
	default:
		return nil, fmt.Errorf("arrow: Mean not supported for %T", col)
	}
}

func (e *Engine) MinMax(col dataset.AnyColumn) (dataset.AnyColumn, dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *arrowFloat64Column:
		vals := c.arr.Float64Values()
		if len(vals) == 0 {
			return nil, nil, fmt.Errorf("arrow: MinMax of empty column")
		}
		lo, hi := simd.SliceMinMax(vals)
		return e.NewFloat64Column(c.name, []float64{lo}),
			e.NewFloat64Column(c.name, []float64{hi}), nil
	case *arrowInt64Column:
		vals := c.arr.Int64Values()
		if len(vals) == 0 {
			return nil, nil, fmt.Errorf("arrow: MinMax of empty column")
		}
		lo, hi := simd.SliceMinMax(vals)
		if c.dtype == dataset.DTypeTimestamp {
			return e.NewTimestampColumn(c.name, []int64{lo}),
				e.NewTimestampColumn(c.name, []int64{hi}), nil
		}
		return e.NewInt64Column(c.name, []int64{lo}),
			e.NewInt64Column(c.name, []int64{hi}), nil
	case *arrowStringColumn:
		vals := stringValues(c.arr)
		if len(vals) == 0 {
			return nil, nil, fmt.Errorf("arrow: MinMax of empty column")
		}
		lo, hi := vals[0], vals[0]
		for _, v := range vals[1:] {
			if v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
		}
		return e.NewStringColumn(c.name, []string{lo}),
			e.NewStringColumn(c.name, []string{hi}), nil
	default:
		return nil, nil, fmt.Errorf("arrow: MinMax not supported for %T", col)
	}
}

func (e *Engine) Count(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	var n int64
	switch c := col.(type) {
	case *arrowFloat64Column:
		n = int64(c.arr.Len() - c.arr.NullN())
	case *arrowInt64Column:
		n = int64(c.arr.Len() - c.arr.NullN())
	case *arrowStringColumn:
		n = int64(c.arr.Len() - c.arr.NullN())
	case *arrowBoolColumn:
		n = int64(c.arr.Len() - c.arr.NullN())
	default:
		n = int64(col.Len())
	}
	return e.NewInt64Column(col.Name(), []int64{n}), nil
}

func (e *Engine) Median(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	ac, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: Median requires float64 column, got %T", col)
	}
	n := ac.arr.Len()
	if n == 0 {
		return nil, fmt.Errorf("arrow: Median of empty column")
	}
	// O(n) partial sort via NthElement — no full sort needed.
	// Read-only access to Arrow buffer, copy into temp for in-place mutation.
	tmp := make([]float64, n)
	copy(tmp, ac.arr.Float64Values())

	mid := n / 2
	dsort.NthElement(tmp, mid)
	var v float64
	if n%2 == 0 {
		// After NthElement(mid), tmp[:mid] are all ≤ tmp[mid].
		// Lower median = max(tmp[:mid]) — O(n/2) scan.
		upper := tmp[mid]
		lower := tmp[0]
		for _, x := range tmp[1:mid] {
			if x > lower {
				lower = x
			}
		}
		v = (lower + upper) / 2
	} else {
		v = tmp[mid]
	}
	b := array.NewFloat64Builder(e.alloc)
	b.Append(v)
	arr := b.NewFloat64Array()
	b.Release()
	return &arrowFloat64Column{name: ac.name, arr: arr}, nil
}

func (e *Engine) Variance(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	ac, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: Variance requires float64 column, got %T", col)
	}
	vals := ac.arr.Float64Values()
	if len(vals) < 2 {
		return e.NewFloat64Column(ac.name, []float64{0}), nil
	}
	// Arrow official SIMD sum
	sum := math.Float64.Sum(ac.arr)
	mean := sum / float64(len(vals))
	ss := 0.0
	for _, v := range vals {
		d := v - mean
		ss += d * d
	}
	return e.NewFloat64Column(ac.name, []float64{ss / float64(len(vals)-1)}), nil
}

// --- Caster ---

func (e *Engine) Cast(col dataset.AnyColumn, target dataset.DType) (dataset.AnyColumn, error) {
	switch target {
	case dataset.DTypeFloat64:
		return e.castToFloat64(col)
	case dataset.DTypeInt64:
		return e.castToInt64(col)
	case dataset.DTypeString:
		return e.castToString(col)
	default:
		return nil, fmt.Errorf("arrow: unsupported cast to %s", target)
	}
}

func (e *Engine) castToFloat64(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *arrowFloat64Column:
		return c, nil
	case *arrowInt64Column:
		b := array.NewFloat64Builder(e.alloc)
		defer b.Release()
		for i := 0; i < c.arr.Len(); i++ {
			if c.arr.IsNull(i) {
				b.AppendNull()
			} else {
				b.Append(float64(c.arr.Value(i)))
			}
		}
		return &arrowFloat64Column{name: c.name, arr: b.NewFloat64Array()}, nil
	default:
		return nil, fmt.Errorf("arrow: cannot cast %s to float64", col.DType())
	}
}

func (e *Engine) castToInt64(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *arrowInt64Column:
		return c, nil
	case *arrowFloat64Column:
		b := array.NewInt64Builder(e.alloc)
		defer b.Release()
		for i := 0; i < c.arr.Len(); i++ {
			if c.arr.IsNull(i) {
				b.AppendNull()
			} else {
				b.Append(int64(c.arr.Value(i)))
			}
		}
		return &arrowInt64Column{name: c.name, arr: b.NewInt64Array()}, nil
	default:
		return nil, fmt.Errorf("arrow: cannot cast %s to int64", col.DType())
	}
}

func (e *Engine) castToString(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *arrowStringColumn:
		return c, nil
	case *arrowFloat64Column:
		b := array.NewStringBuilder(e.alloc)
		defer b.Release()
		for i := 0; i < c.arr.Len(); i++ {
			if c.arr.IsNull(i) {
				b.AppendNull()
			} else {
				b.Append(fmt.Sprintf("%g", c.arr.Value(i)))
			}
		}
		return &arrowStringColumn{name: c.name, arr: b.NewStringArray()}, nil
	default:
		return nil, fmt.Errorf("arrow: cannot cast %s to string", col.DType())
	}
}

// --- Arrow column types (satisfy AnyColumn + Column[T]) ---

type arrowFloat64Column struct {
	name string
	arr  *array.Float64
}

func (c *arrowFloat64Column) Name() string         { return c.name }
func (c *arrowFloat64Column) Len() int64           { return int64(c.arr.Len()) }
func (c *arrowFloat64Column) DType() dataset.DType { return dataset.DTypeFloat64 }
func (c *arrowFloat64Column) Values() []float64    { return c.arr.Float64Values() }
func (c *arrowFloat64Column) IsNull() []bool {
	if c.arr.NullN() == 0 {
		return nil
	}
	nulls := make([]bool, c.arr.Len())
	for i := range nulls {
		nulls[i] = c.arr.IsNull(i)
	}
	return nulls
}

type arrowInt64Column struct {
	name  string
	arr   *array.Int64
	dtype dataset.DType
}

func (c *arrowInt64Column) Name() string { return c.name }
func (c *arrowInt64Column) Len() int64   { return int64(c.arr.Len()) }
func (c *arrowInt64Column) DType() dataset.DType {
	if c.dtype != 0 {
		return c.dtype
	}
	return dataset.DTypeInt64
}
func (c *arrowInt64Column) Values() []int64 { return c.arr.Int64Values() }
func (c *arrowInt64Column) IsNull() []bool {
	if c.arr.NullN() == 0 {
		return nil
	}
	nulls := make([]bool, c.arr.Len())
	for i := range nulls {
		nulls[i] = c.arr.IsNull(i)
	}
	return nulls
}

type arrowStringColumn struct {
	name string
	arr  *array.String
}

func (c *arrowStringColumn) Name() string         { return c.name }
func (c *arrowStringColumn) Len() int64           { return int64(c.arr.Len()) }
func (c *arrowStringColumn) DType() dataset.DType { return dataset.DTypeString }
func (c *arrowStringColumn) Values() []string {
	vals := make([]string, c.arr.Len())
	for i := range vals {
		vals[i] = c.arr.Value(i)
	}
	return vals
}
func (c *arrowStringColumn) IsNull() []bool {
	if c.arr.NullN() == 0 {
		return nil
	}
	nulls := make([]bool, c.arr.Len())
	for i := range nulls {
		nulls[i] = c.arr.IsNull(i)
	}
	return nulls
}

type arrowBoolColumn struct {
	name string
	arr  *array.Boolean
}

func (c *arrowBoolColumn) Name() string         { return c.name }
func (c *arrowBoolColumn) Len() int64           { return int64(c.arr.Len()) }
func (c *arrowBoolColumn) DType() dataset.DType { return dataset.DTypeBool }
func (c *arrowBoolColumn) Values() []bool {
	vals := make([]bool, c.arr.Len())
	for i := range vals {
		vals[i] = c.arr.Value(i)
	}
	return vals
}
func (c *arrowBoolColumn) IsNull() []bool {
	if c.arr.NullN() == 0 {
		return nil
	}
	nulls := make([]bool, c.arr.Len())
	for i := range nulls {
		nulls[i] = c.arr.IsNull(i)
	}
	return nulls
}

// --- Arrow dataset ---

type arrowDataset struct {
	schema  *dataset.Schema
	columns map[string]dataset.AnyColumn
	length  int64
	engine  *Engine
}

func (d *arrowDataset) Schema() *dataset.Schema { return d.schema }
func (d *arrowDataset) NumRows() int64          { return d.length }
func (d *arrowDataset) NumCols() int64          { return int64(d.schema.NumFields()) }
func (d *arrowDataset) Engine() dataset.Engine  { return d.engine }
func (d *arrowDataset) Column(name string) (dataset.AnyColumn, error) {
	col, ok := d.columns[name]
	if !ok {
		return nil, &dataset.ErrColumnNotFound{Name: name}
	}
	return col, nil
}

// --- Arrow Builder ---

type arrowBuilder struct {
	eng      *Engine
	schema   *dataset.Schema
	builders map[string]any
}

func (b *arrowBuilder) Float64(col string) dataset.Float64Appender {
	return b.builders[col].(*arrowFloat64Appender)
}
func (b *arrowBuilder) Int64(col string) dataset.Int64Appender {
	return b.builders[col].(*arrowInt64Appender)
}
func (b *arrowBuilder) String(col string) dataset.StringAppender {
	return b.builders[col].(*arrowStringAppender)
}
func (b *arrowBuilder) Bool(col string) dataset.BoolAppender {
	return b.builders[col].(*arrowBoolAppender)
}

func (b *arrowBuilder) Build() (dataset.Table, error) {
	cols := make([]dataset.AnyColumn, b.schema.NumFields())
	for i := 0; i < b.schema.NumFields(); i++ {
		f := b.schema.Field(i)
		switch f.Dtype {
		case dataset.DTypeFloat64:
			a := b.builders[f.Name].(*arrowFloat64Appender)
			cols[i] = &arrowFloat64Column{name: f.Name, arr: a.b.NewFloat64Array()}
		case dataset.DTypeInt64, dataset.DTypeTimestamp:
			a := b.builders[f.Name].(*arrowInt64Appender)
			cols[i] = &arrowInt64Column{name: f.Name, arr: a.b.NewInt64Array(), dtype: f.Dtype}
		case dataset.DTypeString:
			a := b.builders[f.Name].(*arrowStringAppender)
			cols[i] = &arrowStringColumn{name: f.Name, arr: a.b.NewStringArray()}
		case dataset.DTypeBool:
			a := b.builders[f.Name].(*arrowBoolAppender)
			cols[i] = &arrowBoolColumn{name: f.Name, arr: a.b.NewBooleanArray()}
		}
	}
	return b.eng.FromColumns(b.schema, cols...)
}

// --- Arrow Appenders (wrap Arrow builders directly) ---

type arrowFloat64Appender struct{ b *array.Float64Builder }

func (a *arrowFloat64Appender) Append(v float64)          { a.b.Append(v) }
func (a *arrowFloat64Appender) AppendNull()               { a.b.AppendNull() }
func (a *arrowFloat64Appender) AppendValues(vs []float64) { a.b.AppendValues(vs, nil) }
func (a *arrowFloat64Appender) Reserve(n int)             { a.b.Reserve(n) }

type arrowInt64Appender struct{ b *array.Int64Builder }

func (a *arrowInt64Appender) Append(v int64)          { a.b.Append(v) }
func (a *arrowInt64Appender) AppendNull()             { a.b.AppendNull() }
func (a *arrowInt64Appender) AppendValues(vs []int64) { a.b.AppendValues(vs, nil) }
func (a *arrowInt64Appender) Reserve(n int)           { a.b.Reserve(n) }

type arrowStringAppender struct{ b *array.StringBuilder }

func (a *arrowStringAppender) Append(v string) { a.b.Append(v) }
func (a *arrowStringAppender) AppendNull()     { a.b.AppendNull() }
func (a *arrowStringAppender) AppendValues(vs []string) {
	for _, v := range vs {
		a.b.Append(v)
	}
}
func (a *arrowStringAppender) Reserve(n int) { a.b.Reserve(n) }

type arrowBoolAppender struct{ b *array.BooleanBuilder }

func (a *arrowBoolAppender) Append(v bool)          { a.b.Append(v) }
func (a *arrowBoolAppender) AppendNull()            { a.b.AppendNull() }
func (a *arrowBoolAppender) AppendValues(vs []bool) { a.b.AppendValues(vs, nil) }
func (a *arrowBoolAppender) Reserve(n int)          { a.b.Reserve(n) }

// --- Helpers ---

// stringValues extracts a Go string slice from an Arrow string array.
func stringValues(arr *array.String) []string {
	vals := make([]string, arr.Len())
	for i := range vals {
		vals[i] = arr.Value(i)
	}
	return vals
}

// --- Selector ---

func (e *Engine) Select(col dataset.AnyColumn, indices []int) (dataset.AnyColumn, error) {
	// Build Arrow Int32 indices array for compute.TakeArray
	ib := array.NewInt32Builder(e.alloc)
	ib.Reserve(len(indices))
	for _, idx := range indices {
		ib.Append(int32(idx))
	}
	idxArr := ib.NewInt32Array()
	ib.Release()
	defer idxArr.Release()

	switch c := col.(type) {
	case *arrowFloat64Column:
		result, err := compute.TakeArray(context.Background(), c.arr, idxArr)
		if err != nil {
			return nil, fmt.Errorf("arrow: TakeArray float64: %w", err)
		}
		return &arrowFloat64Column{name: c.name, arr: result.(*array.Float64)}, nil
	case *arrowInt64Column:
		result, err := compute.TakeArray(context.Background(), c.arr, idxArr)
		if err != nil {
			return nil, fmt.Errorf("arrow: TakeArray int64: %w", err)
		}
		return &arrowInt64Column{name: c.name, arr: result.(*array.Int64), dtype: c.dtype}, nil
	case *arrowStringColumn:
		result, err := compute.TakeArray(context.Background(), c.arr, idxArr)
		if err != nil {
			return nil, fmt.Errorf("arrow: TakeArray string: %w", err)
		}
		return &arrowStringColumn{name: c.name, arr: result.(*array.String)}, nil
	case *arrowBoolColumn:
		result, err := compute.TakeArray(context.Background(), c.arr, idxArr)
		if err != nil {
			return nil, fmt.Errorf("arrow: TakeArray bool: %w", err)
		}
		return &arrowBoolColumn{name: c.name, arr: result.(*array.Boolean)}, nil
	default:
		return nil, fmt.Errorf("arrow: Select not supported for %T", col)
	}
}

func (e *Engine) Slice(col dataset.AnyColumn, start, end int) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *arrowFloat64Column:
		sliced := array.NewSlice(c.arr, int64(start), int64(end)).(*array.Float64)
		return &arrowFloat64Column{name: c.name, arr: sliced}, nil
	case *arrowInt64Column:
		sliced := array.NewSlice(c.arr, int64(start), int64(end)).(*array.Int64)
		return &arrowInt64Column{name: c.name, arr: sliced, dtype: c.dtype}, nil
	case *arrowStringColumn:
		sliced := array.NewSlice(c.arr, int64(start), int64(end)).(*array.String)
		return &arrowStringColumn{name: c.name, arr: sliced}, nil
	case *arrowBoolColumn:
		sliced := array.NewSlice(c.arr, int64(start), int64(end)).(*array.Boolean)
		return &arrowBoolColumn{name: c.name, arr: sliced}, nil
	default:
		return nil, fmt.Errorf("arrow: SliceColumn not supported for %T", col)
	}
}

// SortIndices uses Arrow compute's SortIndicesArray kernel.
// Arrow's implementation handles null placement and type dispatch natively.
func (e *Engine) SortIndices(col dataset.AnyColumn) ([]int, error) {
	var arr arrow.Array
	switch c := col.(type) {
	case *arrowFloat64Column:
		arr = c.arr
	case *arrowInt64Column:
		arr = c.arr
	case *arrowStringColumn:
		arr = c.arr
	case *arrowBoolColumn:
		arr = c.arr
	default:
		return nil, fmt.Errorf("arrow: SortIndices not supported for %T", col)
	}

	ctx := compute.WithAllocator(context.Background(), e.alloc)
	key := compute.DefaultSortKey()
	result, err := compute.SortIndicesArray(ctx, arr, key)
	if err != nil {
		return nil, fmt.Errorf("arrow: SortIndicesArray: %w", err)
	}
	defer result.Release()

	idxArr := result.(*array.Uint64)
	n := idxArr.Len()
	indices := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = int(idxArr.Value(i))
	}
	return indices, nil
}

func (e *Engine) FilterIndices(mask []bool) []int {
	n := 0
	for _, v := range mask {
		if v {
			n++
		}
	}
	indices := make([]int, 0, n)
	for i, v := range mask {
		if v {
			indices = append(indices, i)
		}
	}
	return indices
}

// --- Filterer ---

func (e *Engine) Filter(ds dataset.Table, mask dataset.Masker) (dataset.Table, error) {
	bools, err := mask.Mask(ds)
	if err != nil {
		return nil, err
	}

	// Build Arrow boolean filter array
	fb := array.NewBooleanBuilder(e.alloc)
	fb.Reserve(len(bools))
	fb.AppendValues(bools, nil)
	filterArr := fb.NewBooleanArray()
	fb.Release()
	defer filterArr.Release()

	ctx := context.Background()
	schema := ds.Schema()
	cols := make([]dataset.AnyColumn, schema.NumFields())
	opts := compute.FilterOptions{}

	for i := 0; i < schema.NumFields(); i++ {
		col, err := ds.Column(schema.Field(i).Name)
		if err != nil {
			return nil, err
		}
		switch c := col.(type) {
		case *arrowFloat64Column:
			result, err := compute.FilterArray(ctx, c.arr, filterArr, opts)
			if err != nil {
				return nil, fmt.Errorf("arrow: FilterArray float64: %w", err)
			}
			cols[i] = &arrowFloat64Column{name: c.name, arr: result.(*array.Float64)}
		case *arrowInt64Column:
			result, err := compute.FilterArray(ctx, c.arr, filterArr, opts)
			if err != nil {
				return nil, fmt.Errorf("arrow: FilterArray int64: %w", err)
			}
			cols[i] = &arrowInt64Column{name: c.name, arr: result.(*array.Int64), dtype: c.dtype}
		case *arrowStringColumn:
			result, err := compute.FilterArray(ctx, c.arr, filterArr, opts)
			if err != nil {
				return nil, fmt.Errorf("arrow: FilterArray string: %w", err)
			}
			cols[i] = &arrowStringColumn{name: c.name, arr: result.(*array.String)}
		case *arrowBoolColumn:
			result, err := compute.FilterArray(ctx, c.arr, filterArr, opts)
			if err != nil {
				return nil, fmt.Errorf("arrow: FilterArray bool: %w", err)
			}
			cols[i] = &arrowBoolColumn{name: c.name, arr: result.(*array.Boolean)}
		default:
			return nil, fmt.Errorf("arrow: Filter not supported for %T", col)
		}
	}
	return e.FromColumns(schema, cols...)
}

// --- Filler ---

func (e *Engine) Fill(col dataset.AnyColumn, dir dataset.FillDirection) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *arrowFloat64Column:
		if c.arr.NullN() == 0 {
			return c, nil
		}
		n := c.arr.Len()
		if dir == dataset.FillDown {
			b := array.NewFloat64Builder(e.alloc)
			b.Reserve(n)
			var last float64
			for i := 0; i < n; i++ {
				if c.arr.IsNull(i) {
					b.Append(last)
				} else {
					last = c.arr.Value(i)
					b.Append(last)
				}
			}
			arr := b.NewFloat64Array()
			b.Release()
			return &arrowFloat64Column{name: c.name, arr: arr}, nil
		}
		return fillUpFloat64(e, c, n)
	case *arrowInt64Column:
		if c.arr.NullN() == 0 {
			return c, nil
		}
		n := c.arr.Len()
		if dir == dataset.FillDown {
			b := array.NewInt64Builder(e.alloc)
			b.Reserve(n)
			var last int64
			for i := 0; i < n; i++ {
				if c.arr.IsNull(i) {
					b.Append(last)
				} else {
					last = c.arr.Value(i)
					b.Append(last)
				}
			}
			arr := b.NewInt64Array()
			b.Release()
			return &arrowInt64Column{name: c.name, arr: arr, dtype: c.dtype}, nil
		}
		return fillUpInt64(e, c, n)
	case *arrowStringColumn:
		if c.arr.NullN() == 0 {
			return c, nil
		}
		n := c.arr.Len()
		if dir == dataset.FillDown {
			b := array.NewStringBuilder(e.alloc)
			b.Reserve(n)
			var last string
			for i := 0; i < n; i++ {
				if c.arr.IsNull(i) {
					b.Append(last)
				} else {
					last = c.arr.Value(i)
					b.Append(last)
				}
			}
			arr := b.NewStringArray()
			b.Release()
			return &arrowStringColumn{name: c.name, arr: arr}, nil
		}
		return fillUpString(e, c, n)
	default:
		return nil, fmt.Errorf("arrow: Fill not supported for %T", col)
	}
}

// fillUpFloat64 implements FillUp via reverse → FillDown → reverse.
// Zero temp Go slices — only Arrow builders that release after each step.
func fillUpFloat64(e *Engine, c *arrowFloat64Column, n int) (dataset.AnyColumn, error) {
	rb := array.NewFloat64Builder(e.alloc)
	rb.Reserve(n)
	for i := n - 1; i >= 0; i-- {
		if c.arr.IsNull(i) {
			rb.AppendNull()
		} else {
			rb.Append(c.arr.Value(i))
		}
	}
	rev := rb.NewFloat64Array()
	rb.Release()

	fb := array.NewFloat64Builder(e.alloc)
	fb.Reserve(n)
	var last float64
	for i := 0; i < n; i++ {
		if rev.IsNull(i) {
			fb.Append(last)
		} else {
			last = rev.Value(i)
			fb.Append(last)
		}
	}
	rev.Release()
	filled := fb.NewFloat64Array()
	fb.Release()

	ob := array.NewFloat64Builder(e.alloc)
	ob.Reserve(n)
	for i := n - 1; i >= 0; i-- {
		ob.Append(filled.Value(i))
	}
	filled.Release()
	arr := ob.NewFloat64Array()
	ob.Release()
	return &arrowFloat64Column{name: c.name, arr: arr}, nil
}

func fillUpInt64(e *Engine, c *arrowInt64Column, n int) (dataset.AnyColumn, error) {
	rb := array.NewInt64Builder(e.alloc)
	rb.Reserve(n)
	for i := n - 1; i >= 0; i-- {
		if c.arr.IsNull(i) {
			rb.AppendNull()
		} else {
			rb.Append(c.arr.Value(i))
		}
	}
	rev := rb.NewInt64Array()
	rb.Release()

	fb := array.NewInt64Builder(e.alloc)
	fb.Reserve(n)
	var last int64
	for i := 0; i < n; i++ {
		if rev.IsNull(i) {
			fb.Append(last)
		} else {
			last = rev.Value(i)
			fb.Append(last)
		}
	}
	rev.Release()
	filled := fb.NewInt64Array()
	fb.Release()

	ob := array.NewInt64Builder(e.alloc)
	ob.Reserve(n)
	for i := n - 1; i >= 0; i-- {
		ob.Append(filled.Value(i))
	}
	filled.Release()
	arr := ob.NewInt64Array()
	ob.Release()
	return &arrowInt64Column{name: c.name, arr: arr, dtype: c.dtype}, nil
}

func fillUpString(e *Engine, c *arrowStringColumn, n int) (dataset.AnyColumn, error) {
	rb := array.NewStringBuilder(e.alloc)
	rb.Reserve(n)
	for i := n - 1; i >= 0; i-- {
		if c.arr.IsNull(i) {
			rb.AppendNull()
		} else {
			rb.Append(c.arr.Value(i))
		}
	}
	rev := rb.NewStringArray()
	rb.Release()

	fb := array.NewStringBuilder(e.alloc)
	fb.Reserve(n)
	var last string
	for i := 0; i < n; i++ {
		if rev.IsNull(i) {
			fb.Append(last)
		} else {
			last = rev.Value(i)
			fb.Append(last)
		}
	}
	rev.Release()
	filled := fb.NewStringArray()
	fb.Release()

	ob := array.NewStringBuilder(e.alloc)
	ob.Reserve(n)
	for i := n - 1; i >= 0; i-- {
		ob.Append(filled.Value(i))
	}
	filled.Release()
	arr := ob.NewStringArray()
	ob.Release()
	return &arrowStringColumn{name: c.name, arr: arr}, nil
}
func (e *Engine) DropNA(ds dataset.Table, cols ...string) (dataset.Table, error) {
	n := int(ds.NumRows())
	if n == 0 {
		return ds, nil
	}
	if len(cols) == 0 {
		schema := ds.Schema()
		cols = make([]string, schema.NumFields())
		for i := 0; i < schema.NumFields(); i++ {
			cols[i] = schema.Field(i).Name
		}
	}

	keep := make([]bool, n)
	for i := range keep {
		keep[i] = true
	}
	for _, name := range cols {
		col, err := ds.Column(name)
		if err != nil {
			return nil, err
		}
		// Use Arrow's native null bitmap
		switch c := col.(type) {
		case *arrowFloat64Column:
			if c.arr.NullN() > 0 {
				for i := 0; i < c.arr.Len(); i++ {
					if c.arr.IsNull(i) {
						keep[i] = false
					}
				}
			}
		case *arrowInt64Column:
			if c.arr.NullN() > 0 {
				for i := 0; i < c.arr.Len(); i++ {
					if c.arr.IsNull(i) {
						keep[i] = false
					}
				}
			}
		case *arrowStringColumn:
			if c.arr.NullN() > 0 {
				for i := 0; i < c.arr.Len(); i++ {
					if c.arr.IsNull(i) {
						keep[i] = false
					}
				}
			}
		case *arrowBoolColumn:
			if c.arr.NullN() > 0 {
				for i := 0; i < c.arr.Len(); i++ {
					if c.arr.IsNull(i) {
						keep[i] = false
					}
				}
			}
		}
	}

	indices := e.FilterIndices(keep)
	if len(indices) == n {
		return ds, nil
	}
	schema := ds.Schema()
	outCols := make([]dataset.AnyColumn, schema.NumFields())
	for i := 0; i < schema.NumFields(); i++ {
		col, err := ds.Column(schema.Field(i).Name)
		if err != nil {
			return nil, err
		}
		taken, err := e.Select(col, indices)
		if err != nil {
			return nil, err
		}
		outCols[i] = taken
	}
	return e.FromColumns(schema, outCols...)
}

func (e *Engine) ReplaceNA(col dataset.AnyColumn, defaultVal float64) (dataset.AnyColumn, error) {
	c, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: ReplaceNA requires float64 column, got %T", col)
	}
	if c.arr.NullN() == 0 {
		return c, nil
	}
	n := c.arr.Len()
	b := array.NewFloat64Builder(e.alloc)
	b.Reserve(n)
	for i := 0; i < n; i++ {
		if c.arr.IsNull(i) {
			b.Append(defaultVal)
		} else {
			b.Append(c.arr.Value(i))
		}
	}
	arr := b.NewFloat64Array()
	b.Release()
	return &arrowFloat64Column{name: c.name, arr: arr}, nil
}

// --- Composer ---

func (e *Engine) Stack(datasets ...dataset.Table) (dataset.Table, error) {
	if len(datasets) == 0 {
		return nil, fmt.Errorf("arrow: Stack requires at least one dataset")
	}
	schema := datasets[0].Schema()
	for i := 1; i < len(datasets); i++ {
		s := datasets[i].Schema()
		if s.NumFields() != schema.NumFields() {
			return nil, fmt.Errorf("arrow: Stack schema mismatch: expected %d fields, dataset %d has %d",
				schema.NumFields(), i, s.NumFields())
		}
		for j := 0; j < schema.NumFields(); j++ {
			if s.Field(j).Name != schema.Field(j).Name || s.Field(j).Dtype != schema.Field(j).Dtype {
				return nil, fmt.Errorf("arrow: Stack schema mismatch at field %d: %q(%s) vs %q(%s)",
					j, schema.Field(j).Name, schema.Field(j).Dtype, s.Field(j).Name, s.Field(j).Dtype)
			}
		}
	}

	totalLen := 0
	for _, ds := range datasets {
		totalLen += int(ds.NumRows())
	}

	cols := make([]dataset.AnyColumn, schema.NumFields())
	for ci := 0; ci < schema.NumFields(); ci++ {
		name := schema.Field(ci).Name
		switch schema.Field(ci).Dtype {
		case dataset.DTypeFloat64:
			vals := make([]float64, 0, totalLen)
			for _, ds := range datasets {
				col, _ := ds.Column(name)
				vals = append(vals, col.(dataset.Column[float64]).Values()...)
			}
			cols[ci] = e.NewFloat64Column(name, vals)
		case dataset.DTypeInt64:
			vals := make([]int64, 0, totalLen)
			for _, ds := range datasets {
				col, _ := ds.Column(name)
				vals = append(vals, col.(dataset.Column[int64]).Values()...)
			}
			cols[ci] = e.NewInt64Column(name, vals)
		case dataset.DTypeString:
			vals := make([]string, 0, totalLen)
			for _, ds := range datasets {
				col, _ := ds.Column(name)
				vals = append(vals, col.(dataset.Column[string]).Values()...)
			}
			cols[ci] = e.NewStringColumn(name, vals)
		case dataset.DTypeBool:
			vals := make([]bool, 0, totalLen)
			for _, ds := range datasets {
				col, _ := ds.Column(name)
				vals = append(vals, col.(dataset.Column[bool]).Values()...)
			}
			cols[ci] = e.NewBoolColumn(name, vals)
		case dataset.DTypeTimestamp:
			vals := make([]int64, 0, totalLen)
			for _, ds := range datasets {
				col, _ := ds.Column(name)
				vals = append(vals, col.(dataset.Column[int64]).Values()...)
			}
			cols[ci] = e.NewTimestampColumn(name, vals)
		}
	}
	return e.FromColumns(schema, cols...)
}

func (e *Engine) Combine(datasets ...dataset.Table) (dataset.Table, error) {
	if len(datasets) == 0 {
		return nil, fmt.Errorf("arrow: Combine requires at least one dataset")
	}
	n := datasets[0].NumRows()
	for i := 1; i < len(datasets); i++ {
		if datasets[i].NumRows() != n {
			return nil, fmt.Errorf("arrow: Combine length mismatch: expected %d, dataset %d has %d",
				n, i, datasets[i].NumRows())
		}
	}

	var fields []dataset.Field
	var cols []dataset.AnyColumn
	for _, ds := range datasets {
		s := ds.Schema()
		for i := 0; i < s.NumFields(); i++ {
			fields = append(fields, s.Field(i))
			col, _ := ds.Column(s.Field(i).Name)
			cols = append(cols, col)
		}
	}
	return e.FromColumns(dataset.NewSchema(fields...), cols...)
}

// --- Windower ---

func (e *Engine) Lag(col dataset.AnyColumn, offset int) (dataset.AnyColumn, error) {
	length := int(col.Len())
	switch c := col.(type) {
	case *arrowFloat64Column:
		b := array.NewFloat64Builder(e.alloc)
		b.Reserve(length)
		for i := 0; i < length; i++ {
			if i-offset >= 0 {
				b.Append(c.arr.Value(i - offset))
			} else {
				b.Append(0)
			}
		}
		arr := b.NewFloat64Array()
		b.Release()
		return &arrowFloat64Column{name: c.name, arr: arr}, nil
	case *arrowInt64Column:
		b := array.NewInt64Builder(e.alloc)
		b.Reserve(length)
		for i := 0; i < length; i++ {
			if i-offset >= 0 {
				b.Append(c.arr.Value(i - offset))
			} else {
				b.Append(0)
			}
		}
		arr := b.NewInt64Array()
		b.Release()
		return &arrowInt64Column{name: c.name, arr: arr, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("arrow: Lag not supported for %T", col)
	}
}

func (e *Engine) Lead(col dataset.AnyColumn, offset int) (dataset.AnyColumn, error) {
	length := int(col.Len())
	switch c := col.(type) {
	case *arrowFloat64Column:
		b := array.NewFloat64Builder(e.alloc)
		b.Reserve(length)
		for i := 0; i < length; i++ {
			if i+offset < length {
				b.Append(c.arr.Value(i + offset))
			} else {
				b.Append(0)
			}
		}
		arr := b.NewFloat64Array()
		b.Release()
		return &arrowFloat64Column{name: c.name, arr: arr}, nil
	case *arrowInt64Column:
		b := array.NewInt64Builder(e.alloc)
		b.Reserve(length)
		for i := 0; i < length; i++ {
			if i+offset < length {
				b.Append(c.arr.Value(i + offset))
			} else {
				b.Append(0)
			}
		}
		arr := b.NewInt64Array()
		b.Release()
		return &arrowInt64Column{name: c.name, arr: arr, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("arrow: Lead not supported for %T", col)
	}
}

func (e *Engine) CumSum(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	length := int(col.Len())
	switch c := col.(type) {
	case *arrowFloat64Column:
		b := array.NewFloat64Builder(e.alloc)
		b.Reserve(length)
		acc := 0.0
		for i := 0; i < length; i++ {
			acc += c.arr.Value(i)
			b.Append(acc)
		}
		arr := b.NewFloat64Array()
		b.Release()
		return &arrowFloat64Column{name: c.name, arr: arr}, nil
	case *arrowInt64Column:
		b := array.NewInt64Builder(e.alloc)
		b.Reserve(length)
		acc := int64(0)
		for i := 0; i < length; i++ {
			acc += c.arr.Value(i)
			b.Append(acc)
		}
		arr := b.NewInt64Array()
		b.Release()
		return &arrowInt64Column{name: c.name, arr: arr, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("arrow: CumSum not supported for %T", col)
	}
}

func (e *Engine) CumMax(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	length := int(col.Len())
	switch c := col.(type) {
	case *arrowFloat64Column:
		b := array.NewFloat64Builder(e.alloc)
		b.Reserve(length)
		if length > 0 {
			cur := c.arr.Value(0)
			b.Append(cur)
			for i := 1; i < length; i++ {
				v := c.arr.Value(i)
				if v > cur {
					cur = v
				}
				b.Append(cur)
			}
		}
		arr := b.NewFloat64Array()
		b.Release()
		return &arrowFloat64Column{name: c.name, arr: arr}, nil
	case *arrowInt64Column:
		b := array.NewInt64Builder(e.alloc)
		b.Reserve(length)
		if length > 0 {
			cur := c.arr.Value(0)
			b.Append(cur)
			for i := 1; i < length; i++ {
				v := c.arr.Value(i)
				if v > cur {
					cur = v
				}
				b.Append(cur)
			}
		}
		arr := b.NewInt64Array()
		b.Release()
		return &arrowInt64Column{name: c.name, arr: arr, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("arrow: CumMax not supported for %T", col)
	}
}

func (e *Engine) CumMin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	length := int(col.Len())
	switch c := col.(type) {
	case *arrowFloat64Column:
		b := array.NewFloat64Builder(e.alloc)
		b.Reserve(length)
		if length > 0 {
			cur := c.arr.Value(0)
			b.Append(cur)
			for i := 1; i < length; i++ {
				v := c.arr.Value(i)
				if v < cur {
					cur = v
				}
				b.Append(cur)
			}
		}
		arr := b.NewFloat64Array()
		b.Release()
		return &arrowFloat64Column{name: c.name, arr: arr}, nil
	case *arrowInt64Column:
		b := array.NewInt64Builder(e.alloc)
		b.Reserve(length)
		if length > 0 {
			cur := c.arr.Value(0)
			b.Append(cur)
			for i := 1; i < length; i++ {
				v := c.arr.Value(i)
				if v < cur {
					cur = v
				}
				b.Append(cur)
			}
		}
		arr := b.NewInt64Array()
		b.Release()
		return &arrowInt64Column{name: c.name, arr: arr, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("arrow: CumMin not supported for %T", col)
	}
}

func (e *Engine) Rank(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	n := int(col.Len())
	sorted, err := e.SortIndices(col)
	if err != nil {
		return nil, err
	}
	ranks := make([]int64, n)
	switch c := col.(type) {
	case *arrowFloat64Column:
		for pos, idx := range sorted {
			if pos == 0 {
				ranks[idx] = 1
			} else {
				prevIdx := sorted[pos-1]
				if c.arr.Value(idx) == c.arr.Value(prevIdx) {
					ranks[idx] = ranks[prevIdx]
				} else {
					ranks[idx] = int64(pos + 1)
				}
			}
		}
	case *arrowInt64Column:
		for pos, idx := range sorted {
			if pos == 0 {
				ranks[idx] = 1
			} else {
				prevIdx := sorted[pos-1]
				if c.arr.Value(idx) == c.arr.Value(prevIdx) {
					ranks[idx] = ranks[prevIdx]
				} else {
					ranks[idx] = int64(pos + 1)
				}
			}
		}
	default:
		return nil, fmt.Errorf("arrow: Rank not supported for %T", col)
	}
	b := array.NewInt64Builder(e.alloc)
	b.Reserve(n)
	b.AppendValues(ranks, nil)
	arr := b.NewInt64Array()
	b.Release()
	return &arrowInt64Column{name: col.Name(), arr: arr, dtype: dataset.DTypeInt64}, nil
}

func (e *Engine) DenseRank(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	n := int(col.Len())
	sorted, err := e.SortIndices(col)
	if err != nil {
		return nil, err
	}
	ranks := make([]int64, n)
	switch c := col.(type) {
	case *arrowFloat64Column:
		denseRank := int64(1)
		for pos, idx := range sorted {
			if pos > 0 {
				prevIdx := sorted[pos-1]
				if c.arr.Value(idx) != c.arr.Value(prevIdx) {
					denseRank++
				}
			}
			ranks[idx] = denseRank
		}
	case *arrowInt64Column:
		denseRank := int64(1)
		for pos, idx := range sorted {
			if pos > 0 {
				prevIdx := sorted[pos-1]
				if c.arr.Value(idx) != c.arr.Value(prevIdx) {
					denseRank++
				}
			}
			ranks[idx] = denseRank
		}
	default:
		return nil, fmt.Errorf("arrow: DenseRank not supported for %T", col)
	}
	b := array.NewInt64Builder(e.alloc)
	b.Reserve(n)
	b.AppendValues(ranks, nil)
	arr := b.NewInt64Array()
	b.Release()
	return &arrowInt64Column{name: col.Name(), arr: arr, dtype: dataset.DTypeInt64}, nil
}

func (e *Engine) PercentRank(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	rankCol, err := e.Rank(col)
	if err != nil {
		return nil, err
	}
	rc := rankCol.(*arrowInt64Column)
	n := rc.arr.Len()
	b := array.NewFloat64Builder(e.alloc)
	b.Reserve(n)
	if n <= 1 {
		for i := 0; i < n; i++ {
			b.Append(0)
		}
	} else {
		denom := float64(n - 1)
		for i := 0; i < n; i++ {
			b.Append(float64(rc.arr.Value(i)-1) / denom)
		}
	}
	arr := b.NewFloat64Array()
	b.Release()
	return &arrowFloat64Column{name: col.Name(), arr: arr}, nil
}

func (e *Engine) RowNumber(n int) (dataset.AnyColumn, error) {
	b := array.NewInt64Builder(e.alloc)
	b.Reserve(n)
	for i := 0; i < n; i++ {
		b.Append(int64(i + 1))
	}
	arr := b.NewInt64Array()
	b.Release()
	return &arrowInt64Column{name: "row_number", arr: arr, dtype: dataset.DTypeInt64}, nil
}

// Compile-time interface assertions.
var (
	_ dataset.Engine         = (*Engine)(nil)
	_ dataset.ColumnFactory  = (*Engine)(nil)
	_ dataset.BuilderFactory = (*Engine)(nil)
	_ dataset.Aggregator     = (*Engine)(nil)
	_ dataset.Caster         = (*Engine)(nil)
	_ dataset.Selector       = (*Engine)(nil)
	_ dataset.Filterer       = (*Engine)(nil)
	_ dataset.Filler         = (*Engine)(nil)
	_ dataset.Composer       = (*Engine)(nil)
	_ dataset.Windower       = (*Engine)(nil)
	_ dataset.HasEngine      = (*arrowDataset)(nil)

	_ dataset.AnyColumn       = (*arrowFloat64Column)(nil)
	_ dataset.AnyColumn       = (*arrowInt64Column)(nil)
	_ dataset.AnyColumn       = (*arrowStringColumn)(nil)
	_ dataset.AnyColumn       = (*arrowBoolColumn)(nil)
	_ dataset.Column[float64] = (*arrowFloat64Column)(nil)
	_ dataset.Column[int64]   = (*arrowInt64Column)(nil)
	_ dataset.Column[string]  = (*arrowStringColumn)(nil)
	_ dataset.Column[bool]    = (*arrowBoolColumn)(nil)
)
