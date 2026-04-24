// Package memory provides a lightweight Go-slice-backed compute engine
// for the dataset package. It implements [dataset.ColumnFactory],
// [dataset.BuilderFactory], [dataset.Aggregator], and [dataset.Caster].
//
// Usage:
//
//	eng := memory.NewEngine()
//	f := eng.(dataset.ColumnFactory)
//	ds, _ := f.FromColumns(
//	    dataset.NewSchema(dataset.FloatCol("x"), dataset.StringCol("label")),
//	    f.NewFloat64Column("x", []float64{1, 2, 3}),
//	    f.NewStringColumn("label", []string{"a", "b", "c"}),
//	)
package memory

import (
	"fmt"
	"math"

	"github.com/TuSKan/ggplot/dataset"
	simd "github.com/TuSKan/ggplot/dataset/compute"
	dsort "github.com/TuSKan/ggplot/dataset/sort"
)

// Engine is the Go-slice compute backend.
type Engine struct{}

// NewEngine creates a memory engine.
func NewEngine() *Engine { return &Engine{} }

// Name returns "memory".
func (e *Engine) Name() string { return "memory" }

// --- ColumnFactory ---

func (e *Engine) NewFloat64Column(name string, data []float64) dataset.AnyColumn {
	return &float64Column{name: name, data: data}
}

func (e *Engine) NewInt64Column(name string, data []int64) dataset.AnyColumn {
	return &int64Column{name: name, data: data}
}

func (e *Engine) NewStringColumn(name string, data []string) dataset.AnyColumn {
	return &stringColumn{name: name, data: data}
}

func (e *Engine) NewBoolColumn(name string, data []bool) dataset.AnyColumn {
	return &boolColumn{name: name, data: data}
}

func (e *Engine) NewTimestampColumn(name string, data []int64) dataset.AnyColumn {
	return &int64Column{name: name, data: data, dtype: dataset.DTypeTimestamp}
}

func (e *Engine) FromColumns(schema *dataset.Schema, cols ...dataset.AnyColumn) (dataset.Table, error) {
	if len(cols) == 0 {
		return nil, fmt.Errorf("memory: FromColumns requires at least one column")
	}
	length := cols[0].Len()
	columns := make(map[string]dataset.AnyColumn, len(cols))
	for _, col := range cols {
		if col.Len() != length {
			return nil, fmt.Errorf("memory: column %q has length %d, expected %d",
				col.Name(), col.Len(), length)
		}
		columns[col.Name()] = col
	}
	return &memDataset{schema: schema, columns: columns, length: length, eng: e}, nil
}

// --- BuilderFactory ---

func (e *Engine) NewBuilder(schema *dataset.Schema) dataset.Builder {
	b := &memBuilder{eng: e, schema: schema}
	b.appenders = make(map[string]any, schema.NumFields())
	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		switch f.Dtype {
		case dataset.DTypeFloat64:
			b.appenders[f.Name] = &memFloat64Appender{}
		case dataset.DTypeInt64, dataset.DTypeTimestamp:
			b.appenders[f.Name] = &memInt64Appender{}
		case dataset.DTypeString:
			b.appenders[f.Name] = &memStringAppender{}
		case dataset.DTypeBool:
			b.appenders[f.Name] = &memBoolAppender{}
		}
	}
	return b
}

// --- Aggregator (returns AnyColumn, preserves input type) ---

func (e *Engine) Sum(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *float64Column:
		// SIMD: AVX-512/AVX2/NEON via go-highway
		s := simd.SliceSum(c.data)
		return &float64Column{name: c.name, data: []float64{s}}, nil
	case *int64Column:
		s := simd.SliceSum(c.data)
		return &int64Column{name: c.name, data: []int64{s}, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("memory: Sum not supported for %s", col.DType())
	}
}

func (e *Engine) Mean(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	if col.Len() == 0 {
		return nil, fmt.Errorf("memory: Mean of empty column")
	}
	switch c := col.(type) {
	case *float64Column:
		s := simd.SliceSum(c.data)
		return &float64Column{name: c.name, data: []float64{s / float64(len(c.data))}}, nil
	case *int64Column:
		s := simd.SliceSum(c.data)
		return &float64Column{name: c.name, data: []float64{float64(s) / float64(len(c.data))}}, nil
	default:
		return nil, fmt.Errorf("memory: Mean not supported for %s", col.DType())
	}
}

func (e *Engine) MinMax(col dataset.AnyColumn) (dataset.AnyColumn, dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *float64Column:
		if len(c.data) == 0 {
			return nil, nil, fmt.Errorf("memory: MinMax of empty column")
		}
		// SIMD: single-pass min+max via go-highway
		lo, hi := simd.SliceMinMax(c.data)
		return &float64Column{name: c.name, data: []float64{lo}},
			&float64Column{name: c.name, data: []float64{hi}}, nil
	case *int64Column:
		if len(c.data) == 0 {
			return nil, nil, fmt.Errorf("memory: MinMax of empty column")
		}
		lo, hi := simd.SliceMinMax(c.data)
		return &int64Column{name: c.name, data: []int64{lo}, dtype: c.dtype},
			&int64Column{name: c.name, data: []int64{hi}, dtype: c.dtype}, nil
	case *stringColumn:
		if len(c.data) == 0 {
			return nil, nil, fmt.Errorf("memory: MinMax of empty column")
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
		return nil, nil, fmt.Errorf("memory: MinMax not supported for %s", col.DType())
	}
}

func (e *Engine) Count(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return &int64Column{name: col.Name(), data: []int64{int64(col.Len())}}, nil
}

func (e *Engine) Median(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	c, ok := col.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("memory: Median requires float64 column, got %s", col.DType())
	}
	n := len(c.data)
	if n == 0 {
		return nil, fmt.Errorf("memory: Median of empty column")
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

func (e *Engine) Variance(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	c, ok := col.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("memory: Variance requires float64 column, got %s", col.DType())
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
		return nil, fmt.Errorf("memory: unsupported cast to %s", target)
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
		return nil, fmt.Errorf("memory: cannot cast %s to float64", col.DType())
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
		return nil, fmt.Errorf("memory: cannot cast %s to int64", col.DType())
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
			data[i] = fmt.Sprintf("%d", v)
		}
		return &stringColumn{name: c.name, data: data}, nil
	default:
		return nil, fmt.Errorf("memory: cannot cast %s to string", col.DType())
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
	hasNull := false
	for _, v := range c.data {
		if math.IsNaN(v) {
			hasNull = true
			break
		}
	}
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
	dtype dataset.DType // DTypeInt64 or DTypeTimestamp
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
		return nil, &dataset.ErrColumnNotFound{Name: name}
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
	return b.appenders[col].(*memFloat64Appender)
}
func (b *memBuilder) Int64(col string) dataset.Int64Appender {
	return b.appenders[col].(*memInt64Appender)
}
func (b *memBuilder) String(col string) dataset.StringAppender {
	return b.appenders[col].(*memStringAppender)
}
func (b *memBuilder) Bool(col string) dataset.BoolAppender {
	return b.appenders[col].(*memBoolAppender)
}

func (b *memBuilder) Build() (dataset.Table, error) {
	cols := make([]dataset.AnyColumn, b.schema.NumFields())
	for i := 0; i < b.schema.NumFields(); i++ {
		f := b.schema.Field(i)
		switch f.Dtype {
		case dataset.DTypeFloat64:
			a := b.appenders[f.Name].(*memFloat64Appender)
			cols[i] = &float64Column{name: f.Name, data: a.data}
		case dataset.DTypeInt64, dataset.DTypeTimestamp:
			a := b.appenders[f.Name].(*memInt64Appender)
			cols[i] = &int64Column{name: f.Name, data: a.data, dtype: f.Dtype}
		case dataset.DTypeString:
			a := b.appenders[f.Name].(*memStringAppender)
			cols[i] = &stringColumn{name: f.Name, data: a.data}
		case dataset.DTypeBool:
			a := b.appenders[f.Name].(*memBoolAppender)
			cols[i] = &boolColumn{name: f.Name, data: a.data}
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
	if cap(a.data)-len(a.data) < n {
		a.data = append(make([]float64, 0, len(a.data)+n), a.data...)
	}
}

type memInt64Appender struct{ data []int64 }

func (a *memInt64Appender) Append(v int64)          { a.data = append(a.data, v) }
func (a *memInt64Appender) AppendNull()             { a.data = append(a.data, 0) }
func (a *memInt64Appender) AppendValues(vs []int64) { a.data = append(a.data, vs...) }
func (a *memInt64Appender) Reserve(n int) {
	if cap(a.data)-len(a.data) < n {
		a.data = append(make([]int64, 0, len(a.data)+n), a.data...)
	}
}

type memStringAppender struct{ data []string }

func (a *memStringAppender) Append(v string)          { a.data = append(a.data, v) }
func (a *memStringAppender) AppendNull()              { a.data = append(a.data, "") }
func (a *memStringAppender) AppendValues(vs []string) { a.data = append(a.data, vs...) }
func (a *memStringAppender) Reserve(n int) {
	if cap(a.data)-len(a.data) < n {
		a.data = append(make([]string, 0, len(a.data)+n), a.data...)
	}
}

type memBoolAppender struct{ data []bool }

func (a *memBoolAppender) Append(v bool)          { a.data = append(a.data, v) }
func (a *memBoolAppender) AppendNull()            { a.data = append(a.data, false) }
func (a *memBoolAppender) AppendValues(vs []bool) { a.data = append(a.data, vs...) }
func (a *memBoolAppender) Reserve(n int) {
	if cap(a.data)-len(a.data) < n {
		a.data = append(make([]bool, 0, len(a.data)+n), a.data...)
	}
}

// --- Selector ---

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
		return nil, fmt.Errorf("memory: Take not supported for %T", col)
	}
}

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
		return nil, fmt.Errorf("memory: Slice not supported for %T", col)
	}
}

func (e *Engine) SortIndices(col dataset.AnyColumn) ([]int, error) {
	switch c := col.(type) {
	case *float64Column:
		return dsort.SortIndicesFloat64(c.data), nil
	case *int64Column:
		return dsort.SortIndicesInt64(c.data), nil
	case *stringColumn:
		return dsort.SortIndicesString(c.data), nil
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
		return nil, fmt.Errorf("memory: SortIndices not supported for %T", col)
	}
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
	indices := e.FilterIndices(bools)
	if len(indices) == 0 {
		// Return empty dataset with same schema
		schema := ds.Schema()
		cols := make([]dataset.AnyColumn, schema.NumFields())
		for i := 0; i < schema.NumFields(); i++ {
			f := schema.Field(i)
			switch f.Dtype {
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
			}
		}
		return e.FromColumns(schema, cols...)
	}
	// Delegate to Selector.Take for scatter-gather
	schema := ds.Schema()
	cols := make([]dataset.AnyColumn, schema.NumFields())
	for i := 0; i < schema.NumFields(); i++ {
		col, err := ds.Column(schema.Field(i).Name)
		if err != nil {
			return nil, err
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

func (e *Engine) Fill(col dataset.AnyColumn, dir dataset.FillDirection) (dataset.AnyColumn, error) {
	switch c := col.(type) {
	case *float64Column:
		return fillFloat64(e, c, dir), nil
	case *int64Column:
		return fillInt64(e, c, dir), nil
	case *stringColumn:
		return fillString(e, c, dir), nil
	default:
		return nil, fmt.Errorf("memory: Fill not supported for %T", col)
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

func (e *Engine) DropNA(ds dataset.Table, cols ...string) (dataset.Table, error) {
	n := int(ds.NumRows())
	if n == 0 {
		return ds, nil
	}

	// If no cols specified, check all
	if len(cols) == 0 {
		schema := ds.Schema()
		cols = make([]string, schema.NumFields())
		for i := 0; i < schema.NumFields(); i++ {
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
			return nil, err
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
	c, ok := col.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("memory: ReplaceNA requires float64 column, got %T", col)
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

func (e *Engine) Stack(datasets ...dataset.Table) (dataset.Table, error) {
	if len(datasets) == 0 {
		return nil, fmt.Errorf("memory: Stack requires at least one dataset")
	}
	// Use first dataset's schema as reference
	schema := datasets[0].Schema()

	// Validate all datasets have compatible schemas
	for i := 1; i < len(datasets); i++ {
		s := datasets[i].Schema()
		if s.NumFields() != schema.NumFields() {
			return nil, fmt.Errorf("memory: Stack schema mismatch: expected %d fields, dataset %d has %d",
				schema.NumFields(), i, s.NumFields())
		}
		for j := 0; j < schema.NumFields(); j++ {
			if s.Field(j).Name != schema.Field(j).Name || s.Field(j).Dtype != schema.Field(j).Dtype {
				return nil, fmt.Errorf("memory: Stack schema mismatch at field %d: %q(%s) vs %q(%s)",
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
		return nil, fmt.Errorf("memory: Combine requires at least one dataset")
	}

	// Validate all datasets have same length
	n := datasets[0].NumRows()
	for i := 1; i < len(datasets); i++ {
		if datasets[i].NumRows() != n {
			return nil, fmt.Errorf("memory: Combine length mismatch: expected %d, dataset %d has %d",
				n, i, datasets[i].NumRows())
		}
	}

	// Merge all fields and columns
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
		return nil, fmt.Errorf("memory: Lag not supported for %T", col)
	}
}

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
		return nil, fmt.Errorf("memory: Lead not supported for %T", col)
	}
}

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
		return nil, fmt.Errorf("memory: CumSum not supported for %T", col)
	}
}

func (e *Engine) CumMax(col dataset.AnyColumn) (dataset.AnyColumn, error) {
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
				if c.data[i] > out[i-1] {
					out[i] = c.data[i]
				} else {
					out[i] = out[i-1]
				}
			}
		}
		return &int64Column{name: c.name, data: out, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("memory: CumMax not supported for %T", col)
	}
}

func (e *Engine) CumMin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
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
				if c.data[i] < out[i-1] {
					out[i] = c.data[i]
				} else {
					out[i] = out[i-1]
				}
			}
		}
		return &int64Column{name: c.name, data: out, dtype: c.dtype}, nil
	default:
		return nil, fmt.Errorf("memory: CumMin not supported for %T", col)
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
		return nil, fmt.Errorf("memory: Rank not supported for %T", col)
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
		return nil, fmt.Errorf("memory: DenseRank not supported for %T", col)
	}
	return &int64Column{name: col.Name(), data: ranks}, nil
}

// PercentRank returns (rank - 1) / (n - 1) as float64. Returns 0 for single element.
func (e *Engine) PercentRank(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	rankCol, err := e.Rank(col)
	if err != nil {
		return nil, err
	}
	ranks := rankCol.(dataset.Column[int64]).Values()
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
