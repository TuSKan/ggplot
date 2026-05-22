// Package memory provides a lightweight Go-slice-backed compute engine
// for the dataset package. It implements [dataset.ColumnFactory],
// [dataset.BuilderFactory], [dataset.Aggregator], and [dataset.Caster].
//
// Usage:
//
//	eng := memory.NewEngine(context.Background())
//	f := eng.(dataset.ColumnFactory)
//	ds, _ := f.FromColumns(
//	    dataset.NewSchema(dataset.FloatCol("x"), dataset.StringCol("label")),
//	    f.NewFloat64Column("x", []float64{1, 2, 3}),
//	    f.NewStringColumn("label", []string{"a", "b", "c"}),
//	)
package memory

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
	"github.com/rickb777/date/v2"

	"github.com/TuSKan/ggplot/dataset"
	simd "github.com/TuSKan/ggplot/dataset/compute"
	dsort "github.com/TuSKan/ggplot/dataset/sort"
)

// Engine is the Go-slice compute backend.
type Engine struct {
	ctx context.Context
}

// NewEngine creates a memory engine with the given lifecycle context.
func NewEngine(ctx context.Context) *Engine { return &Engine{ctx: ctx} }

// Name returns "memory".
func (e *Engine) Name() string { return "memory" }

// Context returns the engine's lifecycle context.
func (e *Engine) Context() context.Context {
	if e.ctx == nil {
		return context.Background()
	}

	return e.ctx
}

// --- ColumnFactory ---

// NewFloat64Column creates a float64 column from the given slice.
func (e *Engine) NewFloat64Column(name string, data []float64) dataset.AnyColumn {
	return &float64Column{name: name, data: data}
}

// NewInt64Column creates an int64 column from the given slice.
func (e *Engine) NewInt64Column(name string, data []int64) dataset.AnyColumn {
	return &int64Column{name: name, data: data}
}

// NewStringColumn creates a string column from the given slice.
func (e *Engine) NewStringColumn(name string, data []string) dataset.AnyColumn {
	return &stringColumn{name: name, data: data}
}

// NewBoolColumn creates a bool column from the given slice.
func (e *Engine) NewBoolColumn(name string, data []bool) dataset.AnyColumn {
	return &boolColumn{name: name, data: data}
}

// NewTimestampColumn creates a timestamp column (int64-backed) from the given slice.
func (e *Engine) NewTimestampColumn(name string, data []int64) dataset.AnyColumn {
	return &int64Column{name: name, data: data, dtype: dataset.DTypeTimestamp}
}

// NewTimestampFromTime creates a timestamp column from Go time.Time values.
// Each value is converted to UnixNano (int64 nanoseconds since epoch).
func (e *Engine) NewTimestampFromTime(name string, times []time.Time) dataset.AnyColumn {
	data := make([]int64, len(times))
	for i, t := range times {
		data[i] = t.UnixNano()
	}

	return &int64Column{name: name, data: data, dtype: dataset.DTypeTimestamp}
}

// NewDateColumn creates a date-only column from int64 days since the Unix epoch
// (1970-01-01). Compatible with Arrow DATE32.
func (e *Engine) NewDateColumn(name string, days []int64) dataset.AnyColumn {
	return &int64Column{name: name, data: days, dtype: dataset.DTypeDate}
}

// NewDateFromTime creates a date column from Go time.Time values.
// Each value is truncated to midnight UTC and stored as days since epoch.
func (e *Engine) NewDateFromTime(name string, times []time.Time) dataset.AnyColumn {
	data := make([]int64, len(times))
	for i, t := range times {
		data[i] = t.Unix() / 86400 //nolint:mnd // seconds-per-day is a domain constant.
	}

	return &int64Column{name: name, data: data, dtype: dataset.DTypeDate}
}

// NewTimeColumn creates a time-of-day column from int64 nanoseconds since
// midnight (00:00:00.000000000). Compatible with Arrow TIME64(ns).
func (e *Engine) NewTimeColumn(name string, nanos []int64) dataset.AnyColumn {
	return &int64Column{name: name, data: nanos, dtype: dataset.DTypeTime}
}

// NewTimestampFromString creates a timestamp column by parsing date/time
// strings. Tries RFC3339 first, then ISO 8601 date (via rickb777/date),
// then common layouts. Returns an error if any value fails to parse.
func (e *Engine) NewTimestampFromString(name string, values []string) (dataset.AnyColumn, error) {
	data := make([]int64, len(values))

	for i, s := range values {
		ns, err := parseTimestamp(s)
		if err != nil {
			return nil, fmt.Errorf("memory: NewTimestampFromString %q index %d: %w", name, i, err)
		}

		data[i] = ns
	}

	return &int64Column{name: name, data: data, dtype: dataset.DTypeTimestamp}, nil
}

// NewDateFromString creates a date column by parsing date strings using
// rickb777/date. Supports ISO 8601 ("2006-01-02", "20060102").
// Returns an error if any value fails to parse.
func (e *Engine) NewDateFromString(name string, values []string) (dataset.AnyColumn, error) {
	data := make([]int64, len(values))

	for i, s := range values {
		d, err := date.ParseISO(s)
		if err != nil {
			return nil, fmt.Errorf("memory: NewDateFromString %q index %d: %w", name, i, err)
		}

		// Convert to days since Unix epoch (1970-01-01).
		// Date in v2 is days since year zero; MidnightUTC() gives a time.Time.
		t := d.MidnightUTC()
		data[i] = t.Unix() / 86400 //nolint:mnd // seconds-per-day is a domain constant.
	}

	return &int64Column{name: name, data: data, dtype: dataset.DTypeDate}, nil
}

// FromColumns constructs a Table from a schema and pre-built columns.
func (e *Engine) FromColumns(schema *dataset.Schema, cols ...dataset.AnyColumn) (dataset.Table, error) {
	if len(cols) == 0 {
		return nil, fmt.Errorf("FromColumns: %w", ErrEmptyColumn)
	}

	length := cols[0].Len()

	columns := make(map[string]dataset.AnyColumn, len(cols))
	for _, col := range cols {
		if col.Len() != length {
			return nil, fmt.Errorf("memory: column %q has length %d, expected %d", //nolint:err113 // error contains dynamic context values that vary per call site.
				col.Name(), col.Len(), length)
		}

		columns[col.Name()] = col
	}

	return &memDataset{schema: schema, columns: columns, length: length, eng: e}, nil
}

// --- BuilderFactory ---

// NewBuilder creates a typed row-appender for the given schema.
func (e *Engine) NewBuilder(schema *dataset.Schema) dataset.Builder {
	b := &memBuilder{eng: e, schema: schema}

	b.appenders = make(map[string]any, schema.NumFields())
	for i := range schema.NumFields() {
		f := schema.Field(i)
		switch f.Dtype { //nolint:exhaustive // intentional subset; default case handles the rest.
		case dataset.DTypeFloat64:
			b.appenders[f.Name] = &memFloat64Appender{}
		case dataset.DTypeInt64, dataset.DTypeTimestamp, dataset.DTypeDate, dataset.DTypeTime:
			b.appenders[f.Name] = &memInt64Appender{}
		case dataset.DTypeString:
			b.appenders[f.Name] = &memStringAppender{}
		case dataset.DTypeBool:
			b.appenders[f.Name] = &memBoolAppender{}
		default:
		}
	}

	return b
}

// --- Aggregator (returns AnyColumn, preserves input type) ---

// Sum returns the sum of a numeric column as a single-row column.
func (e *Engine) Sum(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *float64Column:
		s := simd.SliceSum(c.data)
		return &float64Column{name: c.name, data: []float64{s}}, nil
	case *int64Column:
		s := simd.SliceSum(c.data)
		return &int64Column{name: c.name, data: []int64{s}, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("Sum: %s: %w", col.DType(), ErrUnsupportedType)
	}
}

// Mean returns the arithmetic mean of a float64 column as a single-row column.
func (e *Engine) Mean(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	if col.Len() == 0 {
		return nil, fmt.Errorf("Mean: %w", ErrEmptyColumn)
	}

	switch c := col.(type) {
	case *float64Column:
		s := simd.SliceSum(c.data)
		return &float64Column{name: c.name, data: []float64{s / float64(len(c.data))}}, nil
	case *int64Column:
		s := simd.SliceSum(c.data)
		return &float64Column{name: c.name, data: []float64{float64(s) / float64(len(c.data))}}, nil
	default:
		return nil, fmt.Errorf("Mean: %s: %w", col.DType(), ErrUnsupportedType)
	}
}

// MinMax returns two single-row columns containing the min and max values.
func (e *Engine) MinMax(col dataset.AnyColumn) (dataset.AnyColumn, dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *float64Column:
		if len(c.data) == 0 {
			return nil, nil, fmt.Errorf("MinMax: %w", ErrEmptyColumn)
		}
		// Scalar reduction; SIMD via Vec[T] generics regresses due to heap-escape (see slice.go).
		lo, hi := simd.SliceMinMax(c.data)

		return &float64Column{name: c.name, data: []float64{lo}},
			&float64Column{name: c.name, data: []float64{hi}}, nil
	case *int64Column:
		if len(c.data) == 0 {
			return nil, nil, fmt.Errorf("MinMax: %w", ErrEmptyColumn)
		}

		lo, hi := simd.SliceMinMax(c.data)

		return &int64Column{name: c.name, data: []int64{lo}, dtype: c.dtype},
			&int64Column{name: c.name, data: []int64{hi}, dtype: c.dtype}, nil
	case *stringColumn:
		if len(c.data) == 0 {
			return nil, nil, fmt.Errorf("MinMax: %w", ErrEmptyColumn)
		}

		lo, hi := c.data[0], c.data[0]
		for _, v := range c.data[1:] {
			if v < lo {
				lo = v
			}

			if v > hi {
				hi = v
			}
		}

		return &stringColumn{name: c.name, data: []string{lo}},
			&stringColumn{name: c.name, data: []string{hi}}, nil
	default:
		return nil, nil, fmt.Errorf("MinMax: %s: %w", col.DType(), ErrUnsupportedType)
	}
}

// Count returns the row count of a column as a single-row int64 column.
func (e *Engine) Count(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return &int64Column{name: col.Name(), data: []int64{col.Len()}}, nil
}

// Median returns the median of a float64 column as a single-row column.
func (e *Engine) Median(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	c, ok := col.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("Median: got %s: %w", col.DType(), ErrRequiresFloat64)
	}

	n := len(c.data)
	if n == 0 {
		return nil, fmt.Errorf("Median: %w", ErrEmptyColumn)
	}
	// O(n) partial sort via NthElement — no full sort needed.
	tmp := make([]float64, n)
	copy(tmp, c.data)

	mid := n / 2
	dsort.NthElement(tmp, mid)

	var v float64

	if n%2 == 0 {
		// After NthElement(mid), tmp[:mid] are all ≤ tmp[mid].
		// Lower median = max(tmp[:mid]) — simple O(n/2) scan.
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

	return &float64Column{name: c.name, data: []float64{v}}, nil
}

// Variance returns the sample variance of a float64 column as a single-row column.
func (e *Engine) Variance(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	c, ok := col.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("Variance: got %s: %w", col.DType(), ErrRequiresFloat64)
	}

	n := len(c.data)
	if n < 2 {
		return &float64Column{name: c.name, data: []float64{0}}, nil
	}
	// Two-pass: 1) mean via SliceSum, 2) sum of squared deviations
	mean := simd.SliceSum(c.data) / float64(n)
	ss := 0.0

	for _, v := range c.data {
		d := v - mean
		ss += d * d
	}

	return &float64Column{name: c.name, data: []float64{ss / float64(n-1)}}, nil
}

// StdDev returns the sample standard deviation as a single-row float64 column.
func (e *Engine) StdDev(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	vCol, err := e.Variance(col)
	if err != nil {
		return nil, err
	}

	v := vCol.(*float64Column).data[0] //nolint:errcheck,forcetypeassert // Variance always returns *float64Column.

	return &float64Column{name: col.Name(), data: []float64{math.Sqrt(v)}}, nil
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
// For ties, the first value encountered wins.
func (e *Engine) Mode(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	if col.Len() == 0 {
		return nil, fmt.Errorf("Mode: %w", ErrEmptyColumn)
	}

	switch c := col.(type) {
	case *float64Column:
		return modeFloat64(c), nil
	case *int64Column:
		return modeInt64(c), nil
	case *stringColumn:
		return modeString(c), nil
	default:
		return nil, fmt.Errorf("Mode: %s: %w", col.DType(), ErrUnsupportedType)
	}
}

func modeFloat64(c *float64Column) dataset.AnyColumn {
	// Sort-based mode: copy → sort → scan for longest run.
	// Better cache locality and no map overhead for numeric data.
	tmp := make([]float64, len(c.data))
	copy(tmp, c.data)
	slices.Sort(tmp)

	bestVal := tmp[0]
	bestCount := 1
	curCount := 1

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

	return &float64Column{name: c.name, data: []float64{bestVal}}
}

func modeInt64(c *int64Column) dataset.AnyColumn {
	tmp := make([]int64, len(c.data))
	copy(tmp, c.data)
	slices.Sort(tmp)

	bestVal := tmp[0]
	bestCount := 1
	curCount := 1

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

	return &int64Column{name: c.name, data: []int64{bestVal}, dtype: c.dtype}
}

func modeString(c *stringColumn) dataset.AnyColumn {
	// Strings: map-based counting (sort order isn't meaningful for mode tie-breaking).
	counts := make(map[string]int, len(c.data)/2) //nolint:mnd // reasonable pre-alloc hint.

	var bestVal string

	bestCount := 0

	for _, v := range c.data {
		counts[v]++
		if counts[v] > bestCount {
			bestCount = counts[v]
			bestVal = v
		}
	}

	return &stringColumn{name: c.name, data: []string{bestVal}}
}

// Percentile returns the p-th quantile as a single-row float64 column.
// p ∈ [0,1]. Uses sort-based linear interpolation (R-7 method).
func (e *Engine) Percentile(col dataset.AnyColumn, p float64) (dataset.AnyColumn, error) {
	if col.Len() == 0 {
		return nil, fmt.Errorf("Percentile: %w", ErrEmptyColumn)
	}

	if p < 0 || p > 1 {
		return nil, fmt.Errorf("Percentile: p=%f out of range [0,1]: %w", p, ErrOutOfRange)
	}

	switch c := col.(type) {
	case *float64Column:
		return percentileFloat64(c, p), nil
	case *int64Column:
		return percentileInt64(c, p), nil
	default:
		return nil, fmt.Errorf("Percentile: %s: %w", col.DType(), ErrUnsupportedType)
	}
}

func percentileFloat64(c *float64Column, p float64) dataset.AnyColumn {
	tmp := make([]float64, len(c.data))
	copy(tmp, c.data)
	slices.Sort(tmp)

	val := interpolateQuantile(tmp, p)

	return &float64Column{name: c.name, data: []float64{val}}
}

func percentileInt64(c *int64Column, p float64) dataset.AnyColumn {
	tmp := make([]float64, len(c.data))
	for i, v := range c.data {
		tmp[i] = float64(v)
	}

	slices.Sort(tmp)

	val := interpolateQuantile(tmp, p)

	return &float64Column{name: c.name, data: []float64{val}}
}

// interpolateQuantile computes the p-th quantile from a sorted slice
// using R-7 linear interpolation (same as NumPy default / Excel PERCENTILE).
func interpolateQuantile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 1 {
		return sorted[0]
	}

	// R-7: index = (n-1)*p
	idx := p * float64(n-1)
	lo := int(math.Floor(idx))
	hi := lo + 1
	frac := idx - float64(lo)

	if hi >= n {
		return sorted[n-1]
	}

	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

// --- Caster ---

// Cast converts a column to the specified dtype.
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
	case *float64Column:
		return c, nil
	case *int64Column:
		data := make([]float64, len(c.data))
		for i, v := range c.data {
			data[i] = float64(v)
		}

		return &float64Column{name: c.name, data: data}, nil
	case *stringColumn:
		data := make([]float64, len(c.data))
		for i, s := range c.data {
			var f float64
			if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
				data[i] = math.NaN()
			} else {
				data[i] = f
			}
		}

		return &float64Column{name: c.name, data: data}, nil
	default:
		return nil, fmt.Errorf("memory: cannot cast %s to float64: %w", col.DType(), ErrUnsupportedType)
	}
}

func (e *Engine) castToInt64(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *int64Column:
		return c, nil
	case *float64Column:
		data := make([]int64, len(c.data))
		for i, v := range c.data {
			data[i] = int64(v)
		}

		return &int64Column{name: c.name, data: data}, nil
	default:
		return nil, fmt.Errorf("memory: cannot cast %s to int64: %w", col.DType(), ErrUnsupportedType)
	}
}

func (e *Engine) castToString(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *stringColumn:
		return c, nil
	case *float64Column:
		data := make([]string, len(c.data))
		for i, v := range c.data {
			data[i] = fmt.Sprintf("%g", v)
		}

		return &stringColumn{name: c.name, data: data}, nil
	case *int64Column:
		data := make([]string, len(c.data))
		for i, v := range c.data {
			data[i] = strconv.FormatInt(v, 10)
		}

		return &stringColumn{name: c.name, data: data}, nil
	default:
		return nil, fmt.Errorf("memory: cannot cast %s to string: %w", col.DType(), ErrUnsupportedType)
	}
}

// --- Column types (private, satisfy AnyColumn + Column[T]) ---

type float64Column struct {
	name string
	data []float64
}

func (c *float64Column) Name() string         { return c.name }
func (c *float64Column) Len() int64           { return int64(len(c.data)) }
func (c *float64Column) DType() dataset.DType { return dataset.DTypeFloat64 }
func (c *float64Column) Values() []float64    { return c.data }
func (c *float64Column) IsNull() []bool {
	// NaN is the null sentinel for float64 (see memFloat64Appender.AppendNull)
	hasNull := slices.ContainsFunc(c.data, math.IsNaN)

	if !hasNull {
		return nil
	}

	nulls := make([]bool, len(c.data))
	for i, v := range c.data {
		nulls[i] = math.IsNaN(v)
	}

	return nulls
}

type int64Column struct {
	name  string
	data  []int64
	dtype dataset.DType // DTypeInt64, DTypeTimestamp, DTypeDate, or DTypeTime
}

func (c *int64Column) Name() string { return c.name }
func (c *int64Column) Len() int64   { return int64(len(c.data)) }
func (c *int64Column) DType() dataset.DType {
	if c.dtype != 0 {
		return c.dtype
	}

	return dataset.DTypeInt64
}
func (c *int64Column) Values() []int64 { return c.data }
func (c *int64Column) IsNull() []bool  { return nil }

type stringColumn struct {
	name string
	data []string
}

func (c *stringColumn) Name() string         { return c.name }
func (c *stringColumn) Len() int64           { return int64(len(c.data)) }
func (c *stringColumn) DType() dataset.DType { return dataset.DTypeString }
func (c *stringColumn) Values() []string     { return c.data }
func (c *stringColumn) IsNull() []bool       { return nil }

type boolColumn struct {
	name string
	data []bool
}

func (c *boolColumn) Name() string         { return c.name }
func (c *boolColumn) Len() int64           { return int64(len(c.data)) }
func (c *boolColumn) DType() dataset.DType { return dataset.DTypeBool }
func (c *boolColumn) Values() []bool       { return c.data }
func (c *boolColumn) IsNull() []bool       { return nil }

// --- Dataset ---

type memDataset struct {
	schema  *dataset.Schema
	columns map[string]dataset.AnyColumn
	length  int64
	eng     *Engine
}

func (d *memDataset) Schema() *dataset.Schema { return d.schema }
func (d *memDataset) NumRows() int64          { return d.length }
func (d *memDataset) NumCols() int64          { return int64(d.schema.NumFields()) }
func (d *memDataset) Engine() dataset.Engine  { return d.eng }
func (d *memDataset) Column(name string) (dataset.AnyColumn, error) {
	col, ok := d.columns[name]
	if !ok {
		return nil, &dataset.ColumnNotFoundError{Name: name}
	}

	return col, nil
}

// --- Builder ---

type memBuilder struct {
	eng       *Engine
	schema    *dataset.Schema
	appenders map[string]any
}

func (b *memBuilder) Float64(col string) dataset.Float64Appender {
	return b.appenders[col].(*memFloat64Appender) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
}

func (b *memBuilder) Int64(col string) dataset.Int64Appender {
	return b.appenders[col].(*memInt64Appender) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
}

func (b *memBuilder) String(col string) dataset.StringAppender {
	return b.appenders[col].(*memStringAppender) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
}

func (b *memBuilder) Bool(col string) dataset.BoolAppender {
	return b.appenders[col].(*memBoolAppender) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
}

func (b *memBuilder) Build() (dataset.Table, error) {
	cols := make([]dataset.AnyColumn, b.schema.NumFields())
	for i := range b.schema.NumFields() {
		f := b.schema.Field(i)
		switch f.Dtype { //nolint:exhaustive // intentional subset; default case handles the rest.
		case dataset.DTypeFloat64:
			a, ok := b.appenders[f.Name].(*memFloat64Appender)
			if !ok {
				return nil, fmt.Errorf("expected *memFloat64Appender, got %T: %w", b.appenders[f.Name], ErrTakeTypeMismatch)
			}

			cols[i] = &float64Column{name: f.Name, data: a.data}
		case dataset.DTypeInt64, dataset.DTypeTimestamp, dataset.DTypeDate, dataset.DTypeTime:
			a, ok := b.appenders[f.Name].(*memInt64Appender)
			if !ok {
				return nil, fmt.Errorf("expected *memInt64Appender, got %T: %w", b.appenders[f.Name], ErrTakeTypeMismatch)
			}

			cols[i] = &int64Column{name: f.Name, data: a.data, dtype: f.Dtype}
		case dataset.DTypeString:
			a, ok := b.appenders[f.Name].(*memStringAppender)
			if !ok {
				return nil, fmt.Errorf("expected *memStringAppender, got %T: %w", b.appenders[f.Name], ErrTakeTypeMismatch)
			}

			cols[i] = &stringColumn{name: f.Name, data: a.data}
		case dataset.DTypeBool:
			a, ok := b.appenders[f.Name].(*memBoolAppender)
			if !ok {
				return nil, fmt.Errorf("expected *memBoolAppender, got %T: %w", b.appenders[f.Name], ErrTakeTypeMismatch)
			}

			cols[i] = &boolColumn{name: f.Name, data: a.data}
		default:
		}
	}

	return b.eng.FromColumns(b.schema, cols...)
}

// --- Typed Appenders ---

type memFloat64Appender struct{ data []float64 }

func (a *memFloat64Appender) Append(v float64)          { a.data = append(a.data, v) }
func (a *memFloat64Appender) AppendNull()               { a.data = append(a.data, math.NaN()) }
func (a *memFloat64Appender) AppendValues(vs []float64) { a.data = append(a.data, vs...) }
func (a *memFloat64Appender) Reserve(n int) {
	a.data = slices.Grow(a.data, n)
}

type memInt64Appender struct{ data []int64 }

func (a *memInt64Appender) Append(v int64)          { a.data = append(a.data, v) }
func (a *memInt64Appender) AppendNull()             { a.data = append(a.data, 0) }
func (a *memInt64Appender) AppendValues(vs []int64) { a.data = append(a.data, vs...) }
func (a *memInt64Appender) Reserve(n int) {
	a.data = slices.Grow(a.data, n)
}

type memStringAppender struct{ data []string }

func (a *memStringAppender) Append(v string)          { a.data = append(a.data, v) }
func (a *memStringAppender) AppendNull()              { a.data = append(a.data, "") }
func (a *memStringAppender) AppendValues(vs []string) { a.data = append(a.data, vs...) }
func (a *memStringAppender) Reserve(n int) {
	a.data = slices.Grow(a.data, n)
}

type memBoolAppender struct{ data []bool }

func (a *memBoolAppender) Append(v bool)          { a.data = append(a.data, v) }
func (a *memBoolAppender) AppendNull()            { a.data = append(a.data, false) }
func (a *memBoolAppender) AppendValues(vs []bool) { a.data = append(a.data, vs...) }
func (a *memBoolAppender) Reserve(n int) {
	a.data = slices.Grow(a.data, n)
}

// --- Selector ---

// Select returns a new column containing only the rows at the given indices.
func (e *Engine) Select(col dataset.AnyColumn, indices []int) (dataset.AnyColumn, error) {
	n := len(indices)

	switch c := col.(type) {
	case *float64Column:
		out := make([]float64, n)
		for i, idx := range indices {
			out[i] = c.data[idx]
		}

		return &float64Column{name: c.name, data: out}, nil
	case *int64Column:
		out := make([]int64, n)
		for i, idx := range indices {
			out[i] = c.data[idx]
		}

		return &int64Column{name: c.name, data: out, dtype: c.dtype}, nil
	case *stringColumn:
		out := make([]string, n)
		for i, idx := range indices {
			out[i] = c.data[idx]
		}

		return &stringColumn{name: c.name, data: out}, nil
	case *boolColumn:
		out := make([]bool, n)
		for i, idx := range indices {
			out[i] = c.data[idx]
		}

		return &boolColumn{name: c.name, data: out}, nil
	default:
		return nil, fmt.Errorf("take: %T: %w", col, ErrUnsupportedType)
	}
}

// Slice returns a sub-column from start (inclusive) to end (exclusive).
func (e *Engine) Slice(col dataset.AnyColumn, start, end int) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *float64Column:
		return &float64Column{name: c.name, data: c.data[start:end]}, nil
	case *int64Column:
		return &int64Column{name: c.name, data: c.data[start:end], dtype: c.dtype}, nil
	case *stringColumn:
		return &stringColumn{name: c.name, data: c.data[start:end]}, nil
	case *boolColumn:
		return &boolColumn{name: c.name, data: c.data[start:end]}, nil
	default:
		return nil, fmt.Errorf("Slice: %T: %w", col, ErrUnsupportedType)
	}
}

// SortIndices returns the permutation that sorts the column ascending.
func (e *Engine) SortIndices(col dataset.AnyColumn) ([]int, error) {
	switch c := col.(type) {
	case *float64Column:
		return dsort.IndicesFloat64(c.data), nil
	case *int64Column:
		return dsort.IndicesInt64(c.data), nil
	case *stringColumn:
		return dsort.IndicesString(c.data), nil
	case *boolColumn:
		n := len(c.data)

		indices := make([]int, n)
		for i := range indices {
			indices[i] = i
		}

		parallelSortFunc(indices, func(a, b int) int {
			if !c.data[a] && c.data[b] {
				return -1
			}

			if c.data[a] && !c.data[b] {
				return 1
			}

			return 0
		})

		return indices, nil
	default:
		return nil, fmt.Errorf("SortIndices: %T: %w", col, ErrUnsupportedType)
	}
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

// Filter returns a new Table containing only rows where mask evaluates true.
func (e *Engine) Filter(ds dataset.Table, mask dataset.Masker) (dataset.Table, error) {
	bools, err := mask.Mask(ds)
	if err != nil {
		return nil, fmt.Errorf("memory: %w", err)
	}

	indices := e.FilterIndices(bools)
	if len(indices) == 0 {
		// Return empty dataset with same schema
		schema := ds.Schema()

		cols := make([]dataset.AnyColumn, schema.NumFields())
		for i := range schema.NumFields() {
			f := schema.Field(i)
			switch f.Dtype { //nolint:exhaustive // intentional subset; default case handles the rest.
			case dataset.DTypeFloat64:
				cols[i] = e.NewFloat64Column(f.Name, nil)
			case dataset.DTypeInt64:
				cols[i] = e.NewInt64Column(f.Name, nil)
			case dataset.DTypeString:
				cols[i] = e.NewStringColumn(f.Name, nil)
			case dataset.DTypeBool:
				cols[i] = e.NewBoolColumn(f.Name, nil)
			case dataset.DTypeTimestamp:
				cols[i] = e.NewTimestampColumn(f.Name, nil)
			case dataset.DTypeDate:
				cols[i] = e.NewDateColumn(f.Name, nil)
			case dataset.DTypeTime:
				cols[i] = e.NewTimeColumn(f.Name, nil)
			default:
			}
		}

		return e.FromColumns(schema, cols...)
	}
	// Delegate to Selector.Take for scatter-gather
	schema := ds.Schema()

	cols := make([]dataset.AnyColumn, schema.NumFields())
	for i := range schema.NumFields() {
		col, err := ds.Column(schema.Field(i).Name)
		if err != nil {
			return nil, fmt.Errorf("memory: %w", err)
		}

		taken, err := e.Select(col, indices)
		if err != nil {
			return nil, err
		}

		cols[i] = taken
	}

	return e.FromColumns(schema, cols...)
}

// --- Filler ---

// Fill forward- or backward-fills null values in a column.
func (e *Engine) Fill(col dataset.AnyColumn, dir dataset.FillDirection) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *float64Column:
		return fillFloat64(e, c, dir), nil
	case *int64Column:
		return fillInt64(e, c, dir), nil
	case *stringColumn:
		return fillString(e, c, dir), nil
	default:
		return nil, fmt.Errorf("Fill: %T: %w", col, ErrUnsupportedType)
	}
}

func fillFloat64(_ *Engine, c *float64Column, dir dataset.FillDirection) dataset.AnyColumn {
	nulls := c.IsNull()
	if nulls == nil {
		return c // no nulls, nothing to fill
	}

	out := make([]float64, len(c.data))
	copy(out, c.data)

	if dir == dataset.FillDown {
		for i := 1; i < len(out); i++ {
			if nulls[i] {
				out[i] = out[i-1]
			}
		}
	} else { // FillUp
		for i := len(out) - 2; i >= 0; i-- {
			if nulls[i] {
				out[i] = out[i+1]
			}
		}
	}

	return &float64Column{name: c.name, data: out}
}

func fillInt64(_ *Engine, c *int64Column, dir dataset.FillDirection) dataset.AnyColumn {
	nulls := c.IsNull()
	if nulls == nil {
		return c
	}

	out := make([]int64, len(c.data))
	copy(out, c.data)

	if dir == dataset.FillDown {
		for i := 1; i < len(out); i++ {
			if nulls[i] {
				out[i] = out[i-1]
			}
		}
	} else {
		for i := len(out) - 2; i >= 0; i-- {
			if nulls[i] {
				out[i] = out[i+1]
			}
		}
	}

	return &int64Column{name: c.name, data: out, dtype: c.dtype}
}

func fillString(_ *Engine, c *stringColumn, dir dataset.FillDirection) dataset.AnyColumn {
	nulls := c.IsNull()
	if nulls == nil {
		return c
	}

	out := make([]string, len(c.data))
	copy(out, c.data)

	if dir == dataset.FillDown {
		for i := 1; i < len(out); i++ {
			if nulls[i] {
				out[i] = out[i-1]
			}
		}
	} else {
		for i := len(out) - 2; i >= 0; i-- {
			if nulls[i] {
				out[i] = out[i+1]
			}
		}
	}

	return &stringColumn{name: c.name, data: out}
}

// DropNA removes rows containing null values in the specified columns.
func (e *Engine) DropNA(ds dataset.Table, cols ...string) (dataset.Table, error) {
	n := int(ds.NumRows())
	if n == 0 {
		return ds, nil
	}

	// If no cols specified, check all
	if len(cols) == 0 {
		schema := ds.Schema()

		cols = make([]string, schema.NumFields())
		for i := range schema.NumFields() {
			cols[i] = schema.Field(i).Name
		}
	}

	// Build keep mask: true = keep row (no nulls in specified cols)
	keep := make([]bool, n)
	for i := range keep {
		keep[i] = true
	}

	for _, name := range cols {
		col, err := ds.Column(name)
		if err != nil {
			return nil, fmt.Errorf("memory: %w", err)
		}

		switch c := col.(type) {
		case *float64Column:
			for i, isNull := range c.IsNull() {
				if isNull {
					keep[i] = false
				}
			}
		case *int64Column:
			for i, isNull := range c.IsNull() {
				if isNull {
					keep[i] = false
				}
			}
		case *stringColumn:
			for i, isNull := range c.IsNull() {
				if isNull {
					keep[i] = false
				}
			}
		case *boolColumn:
			for i, isNull := range c.IsNull() {
				if isNull {
					keep[i] = false
				}
			}
		}
	}

	indices := e.FilterIndices(keep)
	if len(indices) == n {
		return ds, nil // no rows dropped
	}

	schema := ds.Schema()

	outCols := make([]dataset.AnyColumn, schema.NumFields())
	for i := range schema.NumFields() {
		col, err := ds.Column(schema.Field(i).Name)
		if err != nil {
			return nil, fmt.Errorf("memory: %w", err)
		}

		taken, err := e.Select(col, indices)
		if err != nil {
			return nil, err
		}

		outCols[i] = taken
	}

	return e.FromColumns(schema, outCols...)
}

// ReplaceNA replaces null (NaN) values in a float64 column with defaultVal.
func (e *Engine) ReplaceNA(col dataset.AnyColumn, defaultVal float64) (dataset.AnyColumn, error) {
	c, ok := col.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("ReplaceNA: got %T: %w", col, ErrRequiresFloat64)
	}

	nulls := c.IsNull()
	if nulls == nil {
		return c, nil
	}

	out := make([]float64, len(c.data))
	copy(out, c.data)

	for i, isNull := range nulls {
		if isNull {
			out[i] = defaultVal
		}
	}

	return &float64Column{name: c.name, data: out}, nil
}

// --- Composer ---

// Stack vertically concatenates datasets with compatible schemas.
func (e *Engine) Stack(datasets ...dataset.Table) (dataset.Table, error) {
	if len(datasets) == 0 {
		return nil, fmt.Errorf("Stack requires at least one dataset: %w", ErrUnsupportedType)
	}
	// Use first dataset's schema as reference
	schema := datasets[0].Schema()

	// Validate all datasets have compatible schemas
	for i := 1; i < len(datasets); i++ {
		s := datasets[i].Schema()
		if s.NumFields() != schema.NumFields() {
			return nil, fmt.Errorf("memory: Stack schema mismatch: expected %d fields, dataset %d has %d", //nolint:err113 // error contains dynamic context values that vary per call site.
				schema.NumFields(), i, s.NumFields())
		}

		for j := range schema.NumFields() {
			if s.Field(j).Name != schema.Field(j).Name || s.Field(j).Dtype != schema.Field(j).Dtype {
				return nil, fmt.Errorf("memory: Stack schema mismatch at field %d: %q(%s) vs %q(%s)", //nolint:err113 // error contains dynamic context values that vary per call site.
					j, schema.Field(j).Name, schema.Field(j).Dtype, s.Field(j).Name, s.Field(j).Dtype)
			}
		}
	}

	// Compute total length
	totalLen := 0
	for _, ds := range datasets {
		totalLen += int(ds.NumRows())
	}

	// Concatenate each column
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

			cols[ci] = &int64Column{name: name, data: vals, dtype: schema.Field(ci).Dtype}
		default:
		}
	}

	return e.FromColumns(schema, cols...)
}

// Combine horizontally concatenates datasets with equal row counts.
func (e *Engine) Combine(datasets ...dataset.Table) (dataset.Table, error) {
	if len(datasets) == 0 {
		return nil, fmt.Errorf("Combine requires at least one dataset: %w", ErrUnsupportedType)
	}

	// Validate all datasets have same length
	n := datasets[0].NumRows()
	for i := 1; i < len(datasets); i++ {
		if datasets[i].NumRows() != n {
			return nil, fmt.Errorf("memory: Combine length mismatch: expected %d, dataset %d has %d", //nolint:err113 // error contains dynamic context values that vary per call site.
				n, i, datasets[i].NumRows())
		}
	}

	// Merge all fields and columns
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

// Lag shifts a column's values down by n positions, filling with NaN/zero.
func (e *Engine) Lag(col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *float64Column:
		out := make([]float64, len(c.data))
		for i := range out {
			if i-n >= 0 {
				out[i] = c.data[i-n]
			}
		}

		return &float64Column{name: c.name, data: out}, nil
	case *int64Column:
		out := make([]int64, len(c.data))
		for i := range out {
			if i-n >= 0 {
				out[i] = c.data[i-n]
			}
		}

		return &int64Column{name: c.name, data: out, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("Lag: %T: %w", col, ErrUnsupportedType)
	}
}

// Lead shifts a column's values up by n positions, filling with NaN/zero.
func (e *Engine) Lead(col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *float64Column:
		out := make([]float64, len(c.data))
		for i := range out {
			if i+n < len(c.data) {
				out[i] = c.data[i+n]
			}
		}

		return &float64Column{name: c.name, data: out}, nil
	case *int64Column:
		out := make([]int64, len(c.data))
		for i := range out {
			if i+n < len(c.data) {
				out[i] = c.data[i+n]
			}
		}

		return &int64Column{name: c.name, data: out, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("Lead: %T: %w", col, ErrUnsupportedType)
	}
}

// CumSum returns the cumulative sum of a float64 column.
func (e *Engine) CumSum(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *float64Column:
		out := make([]float64, len(c.data))

		acc := 0.0
		for i, v := range c.data {
			acc += v
			out[i] = acc
		}

		return &float64Column{name: c.name, data: out}, nil
	case *int64Column:
		out := make([]int64, len(c.data))

		acc := int64(0)
		for i, v := range c.data {
			acc += v
			out[i] = acc
		}

		return &int64Column{name: c.name, data: out, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("CumSum: %T: %w", col, ErrUnsupportedType)
	}
}

// CumMax returns the cumulative maximum of a numeric column.
func (e *Engine) CumMax(col dataset.AnyColumn) (dataset.AnyColumn, error) { //nolint:dupl // type-specialized code path.
	switch c := col.(type) {
	case *float64Column:
		out := make([]float64, len(c.data))
		if len(c.data) > 0 {
			out[0] = c.data[0]
			for i := 1; i < len(c.data); i++ {
				if c.data[i] > out[i-1] {
					out[i] = c.data[i]
				} else {
					out[i] = out[i-1]
				}
			}
		}

		return &float64Column{name: c.name, data: out}, nil
	case *int64Column:
		out := make([]int64, len(c.data))
		if len(c.data) > 0 {
			out[0] = c.data[0]
			for i := 1; i < len(c.data); i++ {
				out[i] = max(c.data[i], out[i-1])
			}
		}

		return &int64Column{name: c.name, data: out, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("CumMax: %T: %w", col, ErrUnsupportedType)
	}
}

// CumMin returns the cumulative minimum of a numeric column.
func (e *Engine) CumMin(col dataset.AnyColumn) (dataset.AnyColumn, error) { //nolint:dupl // type-specialized code path.
	switch c := col.(type) {
	case *float64Column:
		out := make([]float64, len(c.data))
		if len(c.data) > 0 {
			out[0] = c.data[0]
			for i := 1; i < len(c.data); i++ {
				if c.data[i] < out[i-1] {
					out[i] = c.data[i]
				} else {
					out[i] = out[i-1]
				}
			}
		}

		return &float64Column{name: c.name, data: out}, nil
	case *int64Column:
		out := make([]int64, len(c.data))
		if len(c.data) > 0 {
			out[0] = c.data[0]
			for i := 1; i < len(c.data); i++ {
				out[i] = min(c.data[i], out[i-1])
			}
		}

		return &int64Column{name: c.name, data: out, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("CumMin: %T: %w", col, ErrUnsupportedType)
	}
}

// Rank returns competition rank (1-indexed). Ties get the same rank,
// next rank skips. E.g. [10,20,20,30] → [1,2,2,4].
func (e *Engine) Rank(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	n := col.Len()

	sorted, err := e.SortIndices(col)
	if err != nil {
		return nil, err
	}

	ranks := make([]int64, n)

	switch c := col.(type) {
	case *float64Column:
		for pos, idx := range sorted {
			if pos == 0 {
				ranks[idx] = 1
			} else {
				prevIdx := sorted[pos-1]
				if c.data[idx] == c.data[prevIdx] {
					ranks[idx] = ranks[prevIdx]
				} else {
					ranks[idx] = int64(pos + 1)
				}
			}
		}
	case *int64Column:
		for pos, idx := range sorted {
			if pos == 0 {
				ranks[idx] = 1
			} else {
				prevIdx := sorted[pos-1]
				if c.data[idx] == c.data[prevIdx] {
					ranks[idx] = ranks[prevIdx]
				} else {
					ranks[idx] = int64(pos + 1)
				}
			}
		}
	default:
		return nil, fmt.Errorf("Rank: %T: %w", col, ErrUnsupportedType)
	}

	return &int64Column{name: col.Name(), data: ranks}, nil
}

// DenseRank returns dense rank (no gaps). E.g. [10,20,20,30] → [1,2,2,3].
func (e *Engine) DenseRank(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	n := col.Len()

	sorted, err := e.SortIndices(col)
	if err != nil {
		return nil, err
	}

	ranks := make([]int64, n)

	switch c := col.(type) {
	case *float64Column:
		denseRank := int64(1)

		for pos, idx := range sorted {
			if pos > 0 {
				prevIdx := sorted[pos-1]
				if c.data[idx] != c.data[prevIdx] {
					denseRank++
				}
			}

			ranks[idx] = denseRank
		}
	case *int64Column:
		denseRank := int64(1)

		for pos, idx := range sorted {
			if pos > 0 {
				prevIdx := sorted[pos-1]
				if c.data[idx] != c.data[prevIdx] {
					denseRank++
				}
			}

			ranks[idx] = denseRank
		}
	default:
		return nil, fmt.Errorf("DenseRank: %T: %w", col, ErrUnsupportedType)
	}

	return &int64Column{name: col.Name(), data: ranks}, nil
}

// PercentRank returns (rank - 1) / (n - 1) as float64. Returns 0 for single element.
func (e *Engine) PercentRank(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	rankCol, err := e.Rank(col)
	if err != nil {
		return nil, err
	}

	ranks := rankCol.(dataset.Column[int64]).Values() //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
	n := len(ranks)

	out := make([]float64, n)
	if n <= 1 {
		return &float64Column{name: col.Name(), data: out}, nil
	}

	denom := float64(n - 1)
	for i, r := range ranks {
		out[i] = float64(r-1) / denom
	}

	return &float64Column{name: col.Name(), data: out}, nil
}

// RowNumber returns a 1-indexed sequential column of length n.
func (e *Engine) RowNumber(n int) (dataset.AnyColumn, error) {
	out := make([]int64, n)
	for i := range out {
		out[i] = int64(i + 1)
	}

	return &int64Column{name: "row_number", data: out}, nil
}

// parseTimestamp uses araddon/dateparse to automatically detect and parse
// date/time strings into nanoseconds since epoch. Supports a wide variety of
// formats including ISO 8601, RFC3339, US/EU date styles, and many more.
func parseTimestamp(s string) (int64, error) {
	t, err := dateparse.ParseAny(s)
	if err != nil {
		return 0, fmt.Errorf("cannot parse %q as timestamp: %w", s, err)
	}

	return t.UnixNano(), nil
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
	_ dataset.HasEngine      = (*memDataset)(nil)

	_ dataset.AnyColumn       = (*float64Column)(nil)
	_ dataset.AnyColumn       = (*int64Column)(nil)
	_ dataset.AnyColumn       = (*stringColumn)(nil)
	_ dataset.AnyColumn       = (*boolColumn)(nil)
	_ dataset.Column[float64] = (*float64Column)(nil)
	_ dataset.Column[int64]   = (*int64Column)(nil)
	_ dataset.Column[string]  = (*stringColumn)(nil)
	_ dataset.Column[bool]    = (*boolColumn)(nil)
)
