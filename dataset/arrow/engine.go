// Package arrow provides an Apache Arrow-backed compute engine for the
// dataset package. It implements [dataset.ColumnFactory], [dataset.BuilderFactory],
// [dataset.Aggregator], and [dataset.Caster] using Arrow arrays and
// arrow/math SIMD kernels.
//
// Usage:
//
//	eng := arrow.NewEngine(ctx, memory.DefaultAllocator)
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
	gomath "math"
	"slices"

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
	ctx   context.Context
}

// NewEngine creates an Arrow engine with the given lifecycle context and memory allocator.
func NewEngine(ctx context.Context, alloc memory.Allocator) *Engine {
	return &Engine{ctx: ctx, alloc: alloc}
}

// Name returns "arrow".
func (e *Engine) Name() string { return "arrow" }

// Context returns the engine's lifecycle context.
func (e *Engine) Context() context.Context {
	if e.ctx == nil {
		return context.Background()
	}

	return e.ctx
}

// Alloc returns the engine's memory allocator.
func (e *Engine) Alloc() memory.Allocator { return e.alloc }

// --- ColumnFactory ---

// NewFloat64Column creates a float64 column backed by an Arrow array.
func (e *Engine) NewFloat64Column(name string, data []float64) dataset.AnyColumn {
	b := array.NewFloat64Builder(e.alloc)
	defer b.Release()

	b.AppendValues(data, nil)

	return &arrowFloat64Column{name: name, arr: b.NewFloat64Array()}
}

// NewInt64Column creates an int64 column backed by an Arrow array.
func (e *Engine) NewInt64Column(name string, data []int64) dataset.AnyColumn {
	b := array.NewInt64Builder(e.alloc)
	defer b.Release()

	b.AppendValues(data, nil)

	return &arrowInt64Column{name: name, arr: b.NewInt64Array()}
}

// NewStringColumn creates a string column backed by an Arrow array.
func (e *Engine) NewStringColumn(name string, data []string) dataset.AnyColumn {
	b := array.NewStringBuilder(e.alloc)
	defer b.Release()

	for _, s := range data {
		b.Append(s)
	}

	return &arrowStringColumn{name: name, arr: b.NewStringArray()}
}

// NewBoolColumn creates a bool column backed by an Arrow array.
func (e *Engine) NewBoolColumn(name string, data []bool) dataset.AnyColumn {
	b := array.NewBooleanBuilder(e.alloc)
	defer b.Release()

	b.AppendValues(data, nil)

	return &arrowBoolColumn{name: name, arr: b.NewBooleanArray()}
}

// NewTimestampColumn creates a timestamp column stored as Arrow int64.
func (e *Engine) NewTimestampColumn(name string, data []int64) dataset.AnyColumn {
	b := array.NewInt64Builder(e.alloc)
	defer b.Release()

	b.AppendValues(data, nil)

	return &arrowInt64Column{name: name, arr: b.NewInt64Array(), dtype: dataset.DTypeTimestamp}
}

// NewDateColumn creates a date column (days since epoch) stored as Arrow int64.
func (e *Engine) NewDateColumn(name string, data []int64) dataset.AnyColumn {
	b := array.NewInt64Builder(e.alloc)
	defer b.Release()

	b.AppendValues(data, nil)

	return &arrowInt64Column{name: name, arr: b.NewInt64Array(), dtype: dataset.DTypeDate}
}

// NewTimeColumn creates a time-of-day column (nanoseconds since midnight)
// stored as Arrow int64.
func (e *Engine) NewTimeColumn(name string, data []int64) dataset.AnyColumn {
	b := array.NewInt64Builder(e.alloc)
	defer b.Release()

	b.AppendValues(data, nil)

	return &arrowInt64Column{name: name, arr: b.NewInt64Array(), dtype: dataset.DTypeTime}
}

// newInt64ColumnWithDType creates an int64-backed column with the given DType.
// This is used internally to preserve temporal dtype when creating result columns.
func (e *Engine) newInt64ColumnWithDType(name string, dtype dataset.DType, data []int64) dataset.AnyColumn {
	b := array.NewInt64Builder(e.alloc)
	defer b.Release()

	b.AppendValues(data, nil)

	return &arrowInt64Column{name: name, arr: b.NewInt64Array(), dtype: dtype}
}

// FromColumns builds a Table from pre-built Arrow columns.
func (e *Engine) FromColumns(schema *dataset.Schema, cols ...dataset.AnyColumn) (dataset.Table, error) {
	if len(cols) == 0 {
		return nil, fmt.Errorf("FromColumns: %w", ErrEmptyColumn)
	}

	length := cols[0].Len()

	columns := make(map[string]dataset.AnyColumn, len(cols))
	for _, col := range cols {
		if col.Len() != length {
			return nil, fmt.Errorf("arrow: column %q has length %d, expected %d", //nolint:err113 // error contains dynamic context values that vary per call site.
				col.Name(), col.Len(), length)
		}

		columns[col.Name()] = col
	}

	return &arrowDataset{schema: schema, columns: columns, length: length, engine: e}, nil
}

// --- BuilderFactory ---

// NewBuilder creates a row-at-a-time Table builder.
func (e *Engine) NewBuilder(schema *dataset.Schema) dataset.Builder {
	b := &arrowBuilder{eng: e, schema: schema}

	b.builders = make(map[string]any, schema.NumFields())
	for i := range schema.NumFields() {
		f := schema.Field(i)
		switch f.Dtype { //nolint:exhaustive // intentional subset; default case handles the rest.
		case dataset.DTypeFloat64:
			b.builders[f.Name] = &arrowFloat64Appender{b: array.NewFloat64Builder(e.alloc)}
		case dataset.DTypeInt64, dataset.DTypeTimestamp, dataset.DTypeDate, dataset.DTypeTime:
			b.builders[f.Name] = &arrowInt64Appender{b: array.NewInt64Builder(e.alloc)}
		case dataset.DTypeString:
			b.builders[f.Name] = &arrowStringAppender{b: array.NewStringBuilder(e.alloc)}
		case dataset.DTypeBool:
			b.builders[f.Name] = &arrowBoolAppender{b: array.NewBooleanBuilder(e.alloc)}
		default:
		}
	}

	return b
}

// --- Aggregator (returns AnyColumn, preserves input type) ---

// Sum returns a single-element column containing the sum.
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
		return nil, fmt.Errorf("Sum: %T: %w", col, ErrUnsupportedType)
	}
}

// Mean returns a single-element column containing the arithmetic mean.
func (e *Engine) Mean(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	if col.Len() == 0 {
		return nil, fmt.Errorf("Mean: %w", ErrEmptyColumn)
	}

	switch c := col.(type) {
	case *arrowFloat64Column:
		s := math.Float64.Sum(c.arr)
		return e.NewFloat64Column(c.name, []float64{s / float64(c.arr.Len())}), nil
	case *arrowInt64Column:
		s := math.Int64.Sum(c.arr)
		return e.NewFloat64Column(c.name, []float64{float64(s) / float64(c.arr.Len())}), nil
	default:
		return nil, fmt.Errorf("Mean: %T: %w", col, ErrUnsupportedType)
	}
}

// MinMax returns two single-element columns containing the min and max.
func (e *Engine) MinMax(col dataset.AnyColumn) (dataset.AnyColumn, dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *arrowFloat64Column:
		vals := c.arr.Float64Values()
		if len(vals) == 0 {
			return nil, nil, fmt.Errorf("MinMax: %w", ErrEmptyColumn)
		}

		lo, hi := simd.SliceMinMax(vals)

		return e.NewFloat64Column(c.name, []float64{lo}),
			e.NewFloat64Column(c.name, []float64{hi}), nil
	case *arrowInt64Column:
		vals := c.arr.Int64Values()
		if len(vals) == 0 {
			return nil, nil, fmt.Errorf("MinMax: %w", ErrEmptyColumn)
		}

		lo, hi := simd.SliceMinMax(vals)
		if c.dtype != 0 && c.dtype != dataset.DTypeInt64 {
			return e.newInt64ColumnWithDType(c.name, c.dtype, []int64{lo}),
				e.newInt64ColumnWithDType(c.name, c.dtype, []int64{hi}), nil
		}

		return e.NewInt64Column(c.name, []int64{lo}),
			e.NewInt64Column(c.name, []int64{hi}), nil
	case *arrowStringColumn:
		vals := stringValues(c.arr)
		if len(vals) == 0 {
			return nil, nil, fmt.Errorf("MinMax: %w", ErrEmptyColumn)
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
		return nil, nil, fmt.Errorf("MinMax: %T: %w", col, ErrUnsupportedType)
	}
}

// Count returns a single-element int64 column with the non-null count.
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
		n = col.Len()
	}

	return e.NewInt64Column(col.Name(), []int64{n}), nil
}

// Median returns a single-element column containing the median.
func (e *Engine) Median(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	ac, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("Median: got %T: %w", col, ErrRequiresFloat64)
	}

	n := ac.arr.Len()
	if n == 0 {
		return nil, fmt.Errorf("Median: %w", ErrEmptyColumn)
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

// Variance returns a single-element column containing the sample variance.
func (e *Engine) Variance(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	ac, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("Variance: got %T: %w", col, ErrRequiresFloat64)
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

// StdDev returns the sample standard deviation as a single-row float64 column.
func (e *Engine) StdDev(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	vCol, err := e.Variance(col)
	if err != nil {
		return nil, err
	}

	v := vCol.(*arrowFloat64Column).arr.Float64Values()[0] //nolint:errcheck,forcetypeassert // Variance always returns *arrowFloat64Column.

	return e.NewFloat64Column(col.Name(), []float64{gomath.Sqrt(v)}), nil
}

// First returns the first element of a column as a single-row column.
func (e *Engine) First(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	if col.Len() == 0 {
		return nil, fmt.Errorf("First: %w", ErrEmptyColumn)
	}

	return e.Slice(col, 0, 1)
}

// Last returns the last element of a column as a single-row column.
func (e *Engine) Last(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	n := int(col.Len())
	if n == 0 {
		return nil, fmt.Errorf("Last: %w", ErrEmptyColumn)
	}

	return e.Slice(col, n-1, n)
}

// Mode returns the most frequent value as a single-row column.
// For ties, the first sorted value wins (deterministic).
// Float64 and int64 use a sort-based scan (O(n log n), no map overhead).
// Strings iterate over the Arrow array directly (no materialization to []string).
func (e *Engine) Mode(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	if col.Len() == 0 {
		return nil, fmt.Errorf("Mode: %w", ErrEmptyColumn)
	}

	switch c := col.(type) {
	case *arrowFloat64Column:
		// Zero-copy slice from Arrow buffer → copy → sort → scan.
		vals := c.arr.Float64Values()
		tmp := make([]float64, len(vals))
		copy(tmp, vals)
		slices.Sort(tmp)

		bestVal := tmp[0]
		bestCount, curCount := 1, 1

		for i := 1; i < len(tmp); i++ {
			if tmp[i] == tmp[i-1] {
				curCount++
			} else {
				curCount = 1
			}

			if curCount > bestCount {
				bestCount = curCount
				bestVal = tmp[i]
			}
		}

		return e.NewFloat64Column(c.name, []float64{bestVal}), nil
	case *arrowInt64Column:
		vals := c.arr.Int64Values()
		tmp := make([]int64, len(vals))
		copy(tmp, vals)
		slices.Sort(tmp)

		bestVal := tmp[0]
		bestCount, curCount := 1, 1

		for i := 1; i < len(tmp); i++ {
			if tmp[i] == tmp[i-1] {
				curCount++
			} else {
				curCount = 1
			}

			if curCount > bestCount {
				bestCount = curCount
				bestVal = tmp[i]
			}
		}

		if c.dtype != 0 && c.dtype != dataset.DTypeInt64 {
			return e.newInt64ColumnWithDType(c.name, c.dtype, []int64{bestVal}), nil
		}

		return e.NewInt64Column(c.name, []int64{bestVal}), nil
	case *arrowStringColumn:
		// Iterate directly over Arrow array — no []string materialization.
		n := c.arr.Len()
		counts := make(map[string]int, n/2) //nolint:mnd // reasonable pre-alloc hint.

		var bestVal string

		bestCount := 0

		for i := range n {
			v := c.arr.Value(i)
			counts[v]++

			if counts[v] > bestCount {
				bestCount = counts[v]
				bestVal = v
			}
		}

		return e.NewStringColumn(c.name, []string{bestVal}), nil
	default:
		return nil, fmt.Errorf("Mode: %T: %w", col, ErrUnsupportedType)
	}
}

// Percentile returns the p-th quantile as a single-row float64 column.
// p ∈ [0,1]. Uses sort-based R-7 linear interpolation.
// Float64: zero-copy slice → copy → sort → interpolate.
// Int64: zero-copy → convert to float64 → sort → interpolate.
func (e *Engine) Percentile(col dataset.AnyColumn, p float64) (dataset.AnyColumn, error) {
	if col.Len() == 0 {
		return nil, fmt.Errorf("Percentile: %w", ErrEmptyColumn)
	}

	if p < 0 || p > 1 {
		return nil, fmt.Errorf("Percentile: p=%f out of range [0,1]: %w", p, ErrOutOfRange)
	}

	switch c := col.(type) {
	case *arrowFloat64Column:
		vals := c.arr.Float64Values()
		tmp := make([]float64, len(vals))
		copy(tmp, vals)
		slices.Sort(tmp)

		return e.NewFloat64Column(c.name, []float64{interpolateQuantile(tmp, p)}), nil
	case *arrowInt64Column:
		vals := c.arr.Int64Values()
		tmp := make([]float64, len(vals))

		for i, v := range vals {
			tmp[i] = float64(v)
		}

		slices.Sort(tmp)

		return e.NewFloat64Column(c.name, []float64{interpolateQuantile(tmp, p)}), nil
	default:
		return nil, fmt.Errorf("Percentile: %T: %w", col, ErrUnsupportedType)
	}
}

// interpolateQuantile computes the p-th quantile from a sorted slice
// using R-7 linear interpolation (same as NumPy default / Excel PERCENTILE).
func interpolateQuantile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 1 {
		return sorted[0]
	}

	idx := p * float64(n-1)
	lo := int(gomath.Floor(idx))
	hi := lo + 1
	frac := idx - float64(lo)

	if hi >= n {
		return sorted[n-1]
	}

	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

// --- Caster ---

// Cast converts a column to the target DType.
func (e *Engine) Cast(col dataset.AnyColumn, target dataset.DType) (dataset.AnyColumn, error) {
	switch target { //nolint:exhaustive // intentional subset; default case handles the rest.
	case dataset.DTypeFloat64:
		return e.castToFloat64(col)
	case dataset.DTypeInt64:
		return e.castToInt64(col)
	case dataset.DTypeString:
		return e.castToString(col)
	default:
		return nil, fmt.Errorf("unsupported cast to %s: %w", target, ErrUnsupportedType)
	}
}

func (e *Engine) castToFloat64(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *arrowFloat64Column:
		return c, nil
	case *arrowInt64Column:
		b := array.NewFloat64Builder(e.alloc)
		defer b.Release()

		for i := range c.arr.Len() {
			if c.arr.IsNull(i) {
				b.AppendNull()
			} else {
				b.Append(float64(c.arr.Value(i)))
			}
		}

		return &arrowFloat64Column{name: c.name, arr: b.NewFloat64Array()}, nil
	default:
		return nil, fmt.Errorf("arrow: cannot cast %s to float64: %w", col.DType(), ErrUnsupportedType)
	}
}

func (e *Engine) castToInt64(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *arrowInt64Column:
		return c, nil
	case *arrowFloat64Column:
		b := array.NewInt64Builder(e.alloc)
		defer b.Release()

		for i := range c.arr.Len() {
			if c.arr.IsNull(i) {
				b.AppendNull()
			} else {
				b.Append(int64(c.arr.Value(i)))
			}
		}

		return &arrowInt64Column{name: c.name, arr: b.NewInt64Array()}, nil
	default:
		return nil, fmt.Errorf("arrow: cannot cast %s to int64: %w", col.DType(), ErrUnsupportedType)
	}
}

func (e *Engine) castToString(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *arrowStringColumn:
		return c, nil
	case *arrowFloat64Column:
		b := array.NewStringBuilder(e.alloc)
		defer b.Release()

		for i := range c.arr.Len() {
			if c.arr.IsNull(i) {
				b.AppendNull()
			} else {
				b.Append(fmt.Sprintf("%g", c.arr.Value(i)))
			}
		}

		return &arrowStringColumn{name: c.name, arr: b.NewStringArray()}, nil
	default:
		return nil, fmt.Errorf("arrow: cannot cast %s to string: %w", col.DType(), ErrUnsupportedType)
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
		return nil, &dataset.ColumnNotFoundError{Name: name}
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
	return b.builders[col].(*arrowFloat64Appender) //nolint:errcheck,forcetypeassert // Builder interface cannot return error.
}

func (b *arrowBuilder) Int64(col string) dataset.Int64Appender {
	return b.builders[col].(*arrowInt64Appender) //nolint:errcheck,forcetypeassert // Builder interface cannot return error.
}

func (b *arrowBuilder) String(col string) dataset.StringAppender {
	return b.builders[col].(*arrowStringAppender) //nolint:errcheck,forcetypeassert // Builder interface cannot return error.
}

func (b *arrowBuilder) Bool(col string) dataset.BoolAppender {
	return b.builders[col].(*arrowBoolAppender) //nolint:errcheck,forcetypeassert // Builder interface cannot return error.
}

func (b *arrowBuilder) Build() (dataset.Table, error) {
	cols := make([]dataset.AnyColumn, b.schema.NumFields())
	for i := range b.schema.NumFields() {
		f := b.schema.Field(i)
		switch f.Dtype { //nolint:exhaustive // intentional subset; default case handles the rest.
		case dataset.DTypeFloat64:
			a := b.builders[f.Name].(*arrowFloat64Appender) //nolint:errcheck,forcetypeassert // type guaranteed by builder schema.
			cols[i] = &arrowFloat64Column{name: f.Name, arr: a.b.NewFloat64Array()}
		case dataset.DTypeInt64, dataset.DTypeTimestamp, dataset.DTypeDate, dataset.DTypeTime:
			a := b.builders[f.Name].(*arrowInt64Appender) //nolint:errcheck,forcetypeassert // type guaranteed by builder schema.
			cols[i] = &arrowInt64Column{name: f.Name, arr: a.b.NewInt64Array(), dtype: f.Dtype}
		case dataset.DTypeString:
			a := b.builders[f.Name].(*arrowStringAppender) //nolint:errcheck,forcetypeassert // type guaranteed by builder schema.
			cols[i] = &arrowStringColumn{name: f.Name, arr: a.b.NewStringArray()}
		case dataset.DTypeBool:
			a := b.builders[f.Name].(*arrowBoolAppender) //nolint:errcheck,forcetypeassert // type guaranteed by builder schema.
			cols[i] = &arrowBoolColumn{name: f.Name, arr: a.b.NewBooleanArray()}
		default:
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

// Select gathers elements at the given indices.
func (e *Engine) Select(col dataset.AnyColumn, indices []int) (dataset.AnyColumn, error) {
	// Build Arrow Int32 indices array for compute.TakeArray
	ib := array.NewInt32Builder(e.alloc)
	ib.Reserve(len(indices))

	for _, idx := range indices {
		ib.Append(int32(idx)) //nolint:gosec // G115: safe — metadata values bounded by platform.
	}

	idxArr := ib.NewInt32Array()

	ib.Release()
	defer idxArr.Release()

	switch c := col.(type) {
	case *arrowFloat64Column:
		result, err := compute.TakeArray(e.Context(), c.arr, idxArr)
		if err != nil {
			return nil, fmt.Errorf("arrow: TakeArray float64: %w", err)
		}

		return &arrowFloat64Column{name: c.name, arr: result.(*array.Float64)}, nil //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
	case *arrowInt64Column:
		result, err := compute.TakeArray(e.Context(), c.arr, idxArr)
		if err != nil {
			return nil, fmt.Errorf("arrow: TakeArray int64: %w", err)
		}

		return &arrowInt64Column{name: c.name, arr: result.(*array.Int64), dtype: c.dtype}, nil //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
	case *arrowStringColumn:
		result, err := compute.TakeArray(e.Context(), c.arr, idxArr)
		if err != nil {
			return nil, fmt.Errorf("arrow: TakeArray string: %w", err)
		}

		return &arrowStringColumn{name: c.name, arr: result.(*array.String)}, nil //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
	case *arrowBoolColumn:
		result, err := compute.TakeArray(e.Context(), c.arr, idxArr)
		if err != nil {
			return nil, fmt.Errorf("arrow: TakeArray bool: %w", err)
		}

		return &arrowBoolColumn{name: c.name, arr: result.(*array.Boolean)}, nil //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
	default:
		return nil, fmt.Errorf("Select: %T: %w", col, ErrUnsupportedType)
	}
}

// Slice returns a contiguous sub-range [start, end) of the column.
func (e *Engine) Slice(col dataset.AnyColumn, start, end int) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *arrowFloat64Column:
		sliced := array.NewSlice(c.arr, int64(start), int64(end)).(*array.Float64) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
		return &arrowFloat64Column{name: c.name, arr: sliced}, nil
	case *arrowInt64Column:
		sliced := array.NewSlice(c.arr, int64(start), int64(end)).(*array.Int64) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
		return &arrowInt64Column{name: c.name, arr: sliced, dtype: c.dtype}, nil
	case *arrowStringColumn:
		sliced := array.NewSlice(c.arr, int64(start), int64(end)).(*array.String) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
		return &arrowStringColumn{name: c.name, arr: sliced}, nil
	case *arrowBoolColumn:
		sliced := array.NewSlice(c.arr, int64(start), int64(end)).(*array.Boolean) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
		return &arrowBoolColumn{name: c.name, arr: sliced}, nil
	default:
		return nil, fmt.Errorf("SliceColumn: %T: %w", col, ErrUnsupportedType)
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
		return nil, fmt.Errorf("SortIndices: %T: %w", col, ErrUnsupportedType)
	}

	ctx := compute.WithAllocator(e.Context(), e.alloc)
	key := compute.DefaultSortKey()

	result, err := compute.SortIndicesArray(ctx, arr, key)
	if err != nil {
		return nil, fmt.Errorf("arrow: SortIndicesArray: %w", err)
	}
	defer result.Release()

	idxArr := result.(*array.Uint64) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
	n := idxArr.Len()

	indices := make([]int, n)
	for i := range n {
		indices[i] = int(idxArr.Value(i)) //nolint:gosec // G115: safe — metadata values bounded by platform.
	}

	return indices, nil
}

// FilterIndices returns the indices where mask is true.
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

// Filter returns a new Table keeping only rows where mask is true.
func (e *Engine) Filter(ds dataset.Table, mask dataset.Masker) (dataset.Table, error) {
	bools, err := mask.Mask(ds)
	if err != nil {
		return nil, fmt.Errorf("arrow: %w", err)
	}

	// Build Arrow boolean filter array
	fb := array.NewBooleanBuilder(e.alloc)
	fb.Reserve(len(bools))
	fb.AppendValues(bools, nil)
	filterArr := fb.NewBooleanArray()

	fb.Release()
	defer filterArr.Release()

	ctx := e.Context()
	schema := ds.Schema()
	cols := make([]dataset.AnyColumn, schema.NumFields())
	opts := compute.FilterOptions{}

	for i := range schema.NumFields() {
		col, err := ds.Column(schema.Field(i).Name)
		if err != nil {
			return nil, fmt.Errorf("arrow: %w", err)
		}

		switch c := col.(type) {
		case *arrowFloat64Column:
			result, err := compute.FilterArray(ctx, c.arr, filterArr, opts)
			if err != nil {
				return nil, fmt.Errorf("arrow: FilterArray float64: %w", err)
			}

			cols[i] = &arrowFloat64Column{name: c.name, arr: result.(*array.Float64)} //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
		case *arrowInt64Column:
			result, err := compute.FilterArray(ctx, c.arr, filterArr, opts)
			if err != nil {
				return nil, fmt.Errorf("arrow: FilterArray int64: %w", err)
			}

			cols[i] = &arrowInt64Column{name: c.name, arr: result.(*array.Int64), dtype: c.dtype} //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
		case *arrowStringColumn:
			result, err := compute.FilterArray(ctx, c.arr, filterArr, opts)
			if err != nil {
				return nil, fmt.Errorf("arrow: FilterArray string: %w", err)
			}

			cols[i] = &arrowStringColumn{name: c.name, arr: result.(*array.String)} //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
		case *arrowBoolColumn:
			result, err := compute.FilterArray(ctx, c.arr, filterArr, opts)
			if err != nil {
				return nil, fmt.Errorf("arrow: FilterArray bool: %w", err)
			}

			cols[i] = &arrowBoolColumn{name: c.name, arr: result.(*array.Boolean)} //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
		default:
			return nil, fmt.Errorf("Filter: %T: %w", col, ErrUnsupportedType)
		}
	}

	return e.FromColumns(schema, cols...)
}

// --- Filler ---

// Fill forward- or backward-fills null values in a column.
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

			for i := range n {
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

			for i := range n {
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

			for i := range n {
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
		return nil, fmt.Errorf("Fill: %T: %w", col, ErrUnsupportedType)
	}
}

// fillUpFloat64 implements FillUp via reverse → FillDown → reverse.
// Zero temp Go slices — only Arrow builders that release after each step.
func fillUpFloat64(e *Engine, c *arrowFloat64Column, n int) (dataset.AnyColumn, error) { //nolint:dupl // type-specialized code path.
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

	for i := range n {
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

	for i := range n {
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

func fillUpString(e *Engine, c *arrowStringColumn, n int) (dataset.AnyColumn, error) { //nolint:dupl // type-specialized code path.
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

	for i := range n {
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

// DropNA removes rows containing null values in the specified columns.
func (e *Engine) DropNA(ds dataset.Table, cols ...string) (dataset.Table, error) { //nolint:gocognit // DropNA is a complex pipeline — splitting reduces clarity.
	n := int(ds.NumRows())
	if n == 0 {
		return ds, nil
	}

	if len(cols) == 0 {
		schema := ds.Schema()

		cols = make([]string, schema.NumFields())
		for i := range schema.NumFields() {
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
			return nil, fmt.Errorf("arrow: %w", err)
		}
		// Use Arrow's native null bitmap
		switch c := col.(type) {
		case *arrowFloat64Column:
			if c.arr.NullN() > 0 {
				for i := range c.arr.Len() {
					if c.arr.IsNull(i) {
						keep[i] = false
					}
				}
			}
		case *arrowInt64Column:
			if c.arr.NullN() > 0 {
				for i := range c.arr.Len() {
					if c.arr.IsNull(i) {
						keep[i] = false
					}
				}
			}
		case *arrowStringColumn:
			if c.arr.NullN() > 0 {
				for i := range c.arr.Len() {
					if c.arr.IsNull(i) {
						keep[i] = false
					}
				}
			}
		case *arrowBoolColumn:
			if c.arr.NullN() > 0 {
				for i := range c.arr.Len() {
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
	for i := range schema.NumFields() {
		col, err := ds.Column(schema.Field(i).Name)
		if err != nil {
			return nil, fmt.Errorf("arrow: %w", err)
		}

		taken, err := e.Select(col, indices)
		if err != nil {
			return nil, err
		}

		outCols[i] = taken
	}

	return e.FromColumns(schema, outCols...)
}

// ReplaceNA replaces null values with a default float64.
func (e *Engine) ReplaceNA(col dataset.AnyColumn, defaultVal float64) (dataset.AnyColumn, error) {
	c, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("ReplaceNA: got %T: %w", col, ErrRequiresFloat64)
	}

	if c.arr.NullN() == 0 {
		return c, nil
	}

	n := c.arr.Len()
	b := array.NewFloat64Builder(e.alloc)
	b.Reserve(n)

	for i := range n {
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

// Stack vertically concatenates datasets with identical schemas.
func (e *Engine) Stack(datasets ...dataset.Table) (dataset.Table, error) {
	if len(datasets) == 0 {
		return nil, fmt.Errorf("Stack requires at least one dataset: %w", ErrUnsupportedType)
	}

	schema := datasets[0].Schema()
	for i := 1; i < len(datasets); i++ {
		s := datasets[i].Schema()
		if s.NumFields() != schema.NumFields() {
			return nil, fmt.Errorf("arrow: Stack schema mismatch: expected %d fields, dataset %d has %d", //nolint:err113 // error contains dynamic context values that vary per call site.
				schema.NumFields(), i, s.NumFields())
		}

		for j := range schema.NumFields() {
			if s.Field(j).Name != schema.Field(j).Name || s.Field(j).Dtype != schema.Field(j).Dtype {
				return nil, fmt.Errorf("arrow: Stack schema mismatch at field %d: %q(%s) vs %q(%s)", //nolint:err113 // error contains dynamic context values that vary per call site.
					j, schema.Field(j).Name, schema.Field(j).Dtype, s.Field(j).Name, s.Field(j).Dtype)
			}
		}
	}

	totalLen := 0
	for _, ds := range datasets {
		totalLen += int(ds.NumRows())
	}

	cols := make([]dataset.AnyColumn, schema.NumFields())
	for ci := range schema.NumFields() {
		name := schema.Field(ci).Name
		switch schema.Field(ci).Dtype { //nolint:exhaustive // intentional subset; default case handles the rest.
		case dataset.DTypeFloat64:
			vals := make([]float64, 0, totalLen)

			for _, ds := range datasets {
				col, _ := ds.Column(name)
				vals = append(vals, col.(dataset.Column[float64]).Values()...) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
			}

			cols[ci] = e.NewFloat64Column(name, vals)
		case dataset.DTypeInt64:
			vals := make([]int64, 0, totalLen)

			for _, ds := range datasets {
				col, _ := ds.Column(name)
				vals = append(vals, col.(dataset.Column[int64]).Values()...) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
			}

			cols[ci] = e.NewInt64Column(name, vals)
		case dataset.DTypeString:
			vals := make([]string, 0, totalLen)

			for _, ds := range datasets {
				col, _ := ds.Column(name)
				vals = append(vals, col.(dataset.Column[string]).Values()...) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
			}

			cols[ci] = e.NewStringColumn(name, vals)
		case dataset.DTypeBool:
			vals := make([]bool, 0, totalLen)

			for _, ds := range datasets {
				col, _ := ds.Column(name)
				vals = append(vals, col.(dataset.Column[bool]).Values()...) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
			}

			cols[ci] = e.NewBoolColumn(name, vals)
		case dataset.DTypeTimestamp, dataset.DTypeDate, dataset.DTypeTime:
			vals := make([]int64, 0, totalLen)

			for _, ds := range datasets {
				col, _ := ds.Column(name)
				vals = append(vals, col.(dataset.Column[int64]).Values()...) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
			}

			cols[ci] = e.newInt64ColumnWithDType(name, schema.Field(ci).Dtype, vals)
		default:
		}
	}

	return e.FromColumns(schema, cols...)
}

// Combine horizontally concatenates datasets of equal row count.
func (e *Engine) Combine(datasets ...dataset.Table) (dataset.Table, error) {
	if len(datasets) == 0 {
		return nil, fmt.Errorf("Combine requires at least one dataset: %w", ErrUnsupportedType)
	}

	n := datasets[0].NumRows()
	for i := 1; i < len(datasets); i++ {
		if datasets[i].NumRows() != n {
			return nil, fmt.Errorf("arrow: Combine length mismatch: expected %d, dataset %d has %d", //nolint:err113 // error contains dynamic context values that vary per call site.
				n, i, datasets[i].NumRows())
		}
	}

	var (
		fields []dataset.Field
		cols   []dataset.AnyColumn
	)

	for _, ds := range datasets {
		s := ds.Schema()
		for i := range s.NumFields() {
			fields = append(fields, s.Field(i))
			col, _ := ds.Column(s.Field(i).Name)
			cols = append(cols, col)
		}
	}

	return e.FromColumns(dataset.NewSchema(fields...), cols...)
}

// --- Windower ---

// Lag shifts column values down by offset positions.
func (e *Engine) Lag(col dataset.AnyColumn, offset int) (dataset.AnyColumn, error) {
	length := int(col.Len())
	switch c := col.(type) {
	case *arrowFloat64Column:
		b := array.NewFloat64Builder(e.alloc)
		b.Reserve(length)

		for i := range length {
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

		for i := range length {
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
		return nil, fmt.Errorf("Lag: %T: %w", col, ErrUnsupportedType)
	}
}

// Lead shifts column values up by offset positions.
func (e *Engine) Lead(col dataset.AnyColumn, offset int) (dataset.AnyColumn, error) {
	length := int(col.Len())
	switch c := col.(type) {
	case *arrowFloat64Column:
		b := array.NewFloat64Builder(e.alloc)
		b.Reserve(length)

		for i := range length {
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

		for i := range length {
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
		return nil, fmt.Errorf("Lead: %T: %w", col, ErrUnsupportedType)
	}
}

// CumSum returns the cumulative sum of the column.
func (e *Engine) CumSum(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	length := int(col.Len())
	switch c := col.(type) {
	case *arrowFloat64Column:
		b := array.NewFloat64Builder(e.alloc)
		b.Reserve(length)

		acc := 0.0
		for i := range length {
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
		for i := range length {
			acc += c.arr.Value(i)
			b.Append(acc)
		}

		arr := b.NewInt64Array()
		b.Release()

		return &arrowInt64Column{name: c.name, arr: arr, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("CumSum: %T: %w", col, ErrUnsupportedType)
	}
}

// CumMax returns the cumulative maximum of a numeric column.
func (e *Engine) CumMax(col dataset.AnyColumn) (dataset.AnyColumn, error) { //nolint:dupl // type-specialized code path.
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
		return nil, fmt.Errorf("CumMax: %T: %w", col, ErrUnsupportedType)
	}
}

// CumMin returns the cumulative minimum of a numeric column.
func (e *Engine) CumMin(col dataset.AnyColumn) (dataset.AnyColumn, error) { //nolint:dupl // type-specialized code path.
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
		return nil, fmt.Errorf("CumMin: %T: %w", col, ErrUnsupportedType)
	}
}

// Rank returns the 1-based rank of each element (ties get the same rank).
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
		return nil, fmt.Errorf("Rank: %T: %w", col, ErrUnsupportedType)
	}

	b := array.NewInt64Builder(e.alloc)
	b.Reserve(n)
	b.AppendValues(ranks, nil)
	arr := b.NewInt64Array()
	b.Release()

	return &arrowInt64Column{name: col.Name(), arr: arr, dtype: dataset.DTypeInt64}, nil
}

// DenseRank returns the dense rank (no gaps between ranks for ties).
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
		return nil, fmt.Errorf("DenseRank: %T: %w", col, ErrUnsupportedType)
	}

	b := array.NewInt64Builder(e.alloc)
	b.Reserve(n)
	b.AppendValues(ranks, nil)
	arr := b.NewInt64Array()
	b.Release()

	return &arrowInt64Column{name: col.Name(), arr: arr, dtype: dataset.DTypeInt64}, nil
}

// PercentRank returns the percent rank ((rank-1) / (n-1)).
func (e *Engine) PercentRank(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	rankCol, err := e.Rank(col)
	if err != nil {
		return nil, err
	}

	rc, ok := rankCol.(*arrowInt64Column)
	if !ok {
		return nil, fmt.Errorf("expected *arrowInt64Column, got %T: %w", rankCol, ErrTakeTypeMismatch)
	}

	n := rc.arr.Len()
	b := array.NewFloat64Builder(e.alloc)
	b.Reserve(n)

	if n <= 1 {
		for range n {
			b.Append(0)
		}
	} else {
		denom := float64(n - 1)
		for i := range n {
			b.Append(float64(rc.arr.Value(i)-1) / denom)
		}
	}

	arr := b.NewFloat64Array()
	b.Release()

	return &arrowFloat64Column{name: col.Name(), arr: arr}, nil
}

// RowNumber returns a 1-based sequential row-number column of length n.
func (e *Engine) RowNumber(n int) (dataset.AnyColumn, error) {
	b := array.NewInt64Builder(e.alloc)
	b.Reserve(n)

	for i := range n {
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
