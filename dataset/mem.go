package dataset

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// --- In-memory columns (used by constructors, Summarize, and Collect) ---

// Float64Column is an in-memory float64 column with optional null mask.
type Float64Column struct {
	Data  []float64
	Nulls []bool // nil means no nulls; if non-nil, true = null
}

func (c *Float64Column) Len() int     { return len(c.Data) }
func (c *Float64Column) DType() DType { return DTypeFloat64 }

func (c *Float64Column) Float64s() (Float64Iter, error) {
	return &sliceFloat64Iter{data: c.Data, nulls: c.Nulls}, nil
}

func (c *Float64Column) Int64s() (Int64Iter, error) {
	return &float64ToInt64Iter{data: c.Data, nulls: c.Nulls}, nil
}

func (c *Float64Column) Strings() (StringIter, error) {
	return &float64ToStringIter{data: c.Data, nulls: c.Nulls}, nil
}

func (c *Float64Column) Min() (float64, error) {
	if len(c.Data) == 0 {
		return 0, fmt.Errorf("dataset: empty column")
	}
	m := math.MaxFloat64
	for i, v := range c.Data {
		if c.Nulls != nil && c.Nulls[i] {
			continue
		}
		if v < m {
			m = v
		}
	}
	return m, nil
}

func (c *Float64Column) Max() (float64, error) {
	if len(c.Data) == 0 {
		return 0, fmt.Errorf("dataset: empty column")
	}
	m := -math.MaxFloat64
	for i, v := range c.Data {
		if c.Nulls != nil && c.Nulls[i] {
			continue
		}
		if v > m {
			m = v
		}
	}
	return m, nil
}

func (c *Float64Column) SliceColumn(i, j int) Column {
	nulls := c.Nulls
	if nulls != nil {
		nulls = nulls[i:j]
	}
	return &Float64Column{Data: c.Data[i:j], Nulls: nulls}
}

func (c *Float64Column) FilterColumn(mask []bool, count int) (Column, error) {
	data := make([]float64, 0, count)
	var nulls []bool
	if c.Nulls != nil {
		nulls = make([]bool, 0, count)
	}
	for i, keep := range mask {
		if i >= len(c.Data) {
			break
		}
		if keep {
			data = append(data, c.Data[i])
			if nulls != nil {
				nulls = append(nulls, c.Nulls[i])
			}
		}
	}
	return &Float64Column{Data: data, Nulls: nulls}, nil
}

// Int64Column is an in-memory int64 column.
type Int64Column struct {
	Data  []int64
	Nulls []bool
}

func (c *Int64Column) Len() int     { return len(c.Data) }
func (c *Int64Column) DType() DType { return DTypeInt64 }

func (c *Int64Column) Float64s() (Float64Iter, error) {
	return &int64ToFloat64Iter{data: c.Data, nulls: c.Nulls}, nil
}

func (c *Int64Column) Int64s() (Int64Iter, error) {
	return &sliceInt64Iter{data: c.Data, nulls: c.Nulls}, nil
}

func (c *Int64Column) Strings() (StringIter, error) {
	return &int64ToStringIter{data: c.Data, nulls: c.Nulls}, nil
}

func (c *Int64Column) Min() (float64, error) {
	if len(c.Data) == 0 {
		return 0, fmt.Errorf("dataset: empty column")
	}
	m := int64(math.MaxInt64)
	for i, v := range c.Data {
		if c.Nulls != nil && c.Nulls[i] {
			continue
		}
		if v < m {
			m = v
		}
	}
	return float64(m), nil
}

func (c *Int64Column) Max() (float64, error) {
	if len(c.Data) == 0 {
		return 0, fmt.Errorf("dataset: empty column")
	}
	m := int64(math.MinInt64)
	for i, v := range c.Data {
		if c.Nulls != nil && c.Nulls[i] {
			continue
		}
		if v > m {
			m = v
		}
	}
	return float64(m), nil
}

func (c *Int64Column) SliceColumn(i, j int) Column {
	nulls := c.Nulls
	if nulls != nil {
		nulls = nulls[i:j]
	}
	return &Int64Column{Data: c.Data[i:j], Nulls: nulls}
}

func (c *Int64Column) FilterColumn(mask []bool, count int) (Column, error) {
	data := make([]int64, 0, count)
	var nulls []bool
	if c.Nulls != nil {
		nulls = make([]bool, 0, count)
	}
	for i, keep := range mask {
		if i >= len(c.Data) {
			break
		}
		if keep {
			data = append(data, c.Data[i])
			if nulls != nil {
				nulls = append(nulls, c.Nulls[i])
			}
		}
	}
	return &Int64Column{Data: data, Nulls: nulls}, nil
}

// StringColumn is an in-memory string/categorical column.
type StringColumn struct {
	Data  []string
	Nulls []bool
}

func (c *StringColumn) Len() int     { return len(c.Data) }
func (c *StringColumn) DType() DType { return DTypeString }

func (c *StringColumn) Float64s() (Float64Iter, error) {
	return nil, fmt.Errorf("dataset: string column does not natively support Float64 iteration")
}

func (c *StringColumn) Int64s() (Int64Iter, error) {
	return nil, fmt.Errorf("dataset: string column does not natively support Int64 iteration")
}

func (c *StringColumn) Strings() (StringIter, error) {
	return &sliceStringIter{data: c.Data, nulls: c.Nulls}, nil
}

func (c *StringColumn) SliceColumn(i, j int) Column {
	nulls := c.Nulls
	if nulls != nil {
		nulls = nulls[i:j]
	}
	return &StringColumn{Data: c.Data[i:j], Nulls: nulls}
}

func (c *StringColumn) FilterColumn(mask []bool, count int) (Column, error) {
	data := make([]string, 0, count)
	var nulls []bool
	if c.Nulls != nil {
		nulls = make([]bool, 0, count)
	}
	for i, keep := range mask {
		if i >= len(c.Data) {
			break
		}
		if keep {
			data = append(data, c.Data[i])
			if nulls != nil {
				nulls = append(nulls, c.Nulls[i])
			}
		}
	}
	return &StringColumn{Data: data, Nulls: nulls}, nil
}

// --- Iterators ---

type sliceFloat64Iter struct {
	data  []float64
	nulls []bool
	pos   int
}

func (it *sliceFloat64Iter) Next() (float64, bool, bool) {
	if it.pos >= len(it.data) {
		return 0, false, false
	}
	v := it.data[it.pos]
	isNull := it.nulls != nil && it.nulls[it.pos]
	it.pos++
	return v, isNull, true
}

type sliceInt64Iter struct {
	data  []int64
	nulls []bool
	pos   int
}

func (it *sliceInt64Iter) Next() (int64, bool, bool) {
	if it.pos >= len(it.data) {
		return 0, false, false
	}
	v := it.data[it.pos]
	isNull := it.nulls != nil && it.nulls[it.pos]
	it.pos++
	return v, isNull, true
}

type sliceStringIter struct {
	data  []string
	nulls []bool
	pos   int
}

func (it *sliceStringIter) Next() (string, bool, bool) {
	if it.pos >= len(it.data) {
		return "", false, false
	}
	v := it.data[it.pos]
	isNull := it.nulls != nil && it.nulls[it.pos]
	it.pos++
	return v, isNull, true
}

// --- Cross-type iterators ---

type float64ToInt64Iter struct {
	data  []float64
	nulls []bool
	pos   int
}

func (it *float64ToInt64Iter) Next() (int64, bool, bool) {
	if it.pos >= len(it.data) {
		return 0, false, false
	}
	v := int64(it.data[it.pos])
	isNull := it.nulls != nil && it.nulls[it.pos]
	it.pos++
	return v, isNull, true
}

type float64ToStringIter struct {
	data  []float64
	nulls []bool
	pos   int
}

func (it *float64ToStringIter) Next() (string, bool, bool) {
	if it.pos >= len(it.data) {
		return "", false, false
	}
	isNull := it.nulls != nil && it.nulls[it.pos]
	v := it.data[it.pos]
	it.pos++
	if isNull {
		return "", true, true
	}
	return fmt.Sprintf("%g", v), false, true
}

type int64ToFloat64Iter struct {
	data  []int64
	nulls []bool
	pos   int
}

func (it *int64ToFloat64Iter) Next() (float64, bool, bool) {
	if it.pos >= len(it.data) {
		return 0, false, false
	}
	v := float64(it.data[it.pos])
	isNull := it.nulls != nil && it.nulls[it.pos]
	it.pos++
	return v, isNull, true
}

type int64ToStringIter struct {
	data  []int64
	nulls []bool
	pos   int
}

func (it *int64ToStringIter) Next() (string, bool, bool) {
	if it.pos >= len(it.data) {
		return "", false, false
	}
	isNull := it.nulls != nil && it.nulls[it.pos]
	v := it.data[it.pos]
	it.pos++
	if isNull {
		return "", true, true
	}
	return fmt.Sprintf("%d", v), false, true
}

// --- In-memory Dataset (memoryDataset) ---

type memoryDataset struct {
	cols    []string
	columns map[string]Column
	length  int
}

func (m *memoryDataset) Columns() []string { return m.cols }
func (m *memoryDataset) Len() int          { return m.length }
func (m *memoryDataset) Column(name string) (Column, error) {
	col, ok := m.columns[name]
	if !ok {
		return nil, &ErrColumnNotFound{Name: name}
	}
	return col, nil
}

// --- DataFrame constructors ---

// NewDataFrame creates an in-memory Dataset from named float64 slices.
// All slices must have the same length.
func NewDataFrame(data map[string][]float64) (Dataset, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("dataset: empty data map")
	}

	var length int
	var names []string
	first := true
	for name, vals := range data {
		if first {
			length = len(vals)
			first = false
		} else if len(vals) != length {
			return nil, fmt.Errorf("dataset: column %q has length %d, expected %d", name, len(vals), length)
		}
		names = append(names, name)
	}
	sort.Strings(names)

	columns := make(map[string]Column, len(data))
	for name, vals := range data {
		cp := make([]float64, len(vals))
		copy(cp, vals)
		columns[name] = &Float64Column{Data: cp}
	}

	return &memoryDataset{cols: names, columns: columns, length: length}, nil
}

// NewMixedDataFrame creates an in-memory Dataset from float64, int64, and string columns.
func NewMixedDataFrame(opts ...DataFrameOpt) (Dataset, error) {
	b := &dataFrameBuilder{columns: make(map[string]Column)}
	for _, opt := range opts {
		opt(b)
	}
	if len(b.columns) == 0 {
		return nil, fmt.Errorf("dataset: no columns provided")
	}

	var length int
	first := true
	for name, col := range b.columns {
		if first {
			length = col.Len()
			first = false
		} else if col.Len() != length {
			return nil, fmt.Errorf("dataset: column %q has length %d, expected %d", name, col.Len(), length)
		}
	}

	names := make([]string, 0, len(b.columns))
	for name := range b.columns {
		names = append(names, name)
	}
	sort.Strings(names)

	return &memoryDataset{cols: names, columns: b.columns, length: length}, nil
}

// DataFrameOpt is a functional option for NewMixedDataFrame.
type DataFrameOpt func(*dataFrameBuilder)

type dataFrameBuilder struct {
	columns map[string]Column
}

// WithFloat64s adds a float64 column.
func WithFloat64s(name string, data []float64) DataFrameOpt {
	return func(b *dataFrameBuilder) {
		cp := make([]float64, len(data))
		copy(cp, data)
		b.columns[name] = &Float64Column{Data: cp}
	}
}

// WithInt64s adds an int64 column.
func WithInt64s(name string, data []int64) DataFrameOpt {
	return func(b *dataFrameBuilder) {
		cp := make([]int64, len(data))
		copy(cp, data)
		b.columns[name] = &Int64Column{Data: cp}
	}
}

// WithStrings adds a string column.
func WithStrings(name string, data []string) DataFrameOpt {
	return func(b *dataFrameBuilder) {
		cp := make([]string, len(data))
		copy(cp, data)
		b.columns[name] = &StringColumn{Data: cp}
	}
}

// WithColumn adds a pre-built column.
func WithColumn(name string, col Column) DataFrameOpt {
	return func(b *dataFrameBuilder) {
		b.columns[name] = col
	}
}

// --- CollectFloat64s extracts all float64 from a column ---

// CollectFloat64s materializes a column's float64 values into a Go slice.
func CollectFloat64s(col Column) ([]float64, error) {
	iter, ok := col.(IterableColumn)
	if !ok {
		return nil, fmt.Errorf("dataset: column does not support iteration")
	}
	flt, err := iter.Float64s()
	if err != nil {
		return nil, err
	}
	result := make([]float64, 0, col.Len())
	for {
		v, isNull, ok := flt.Next()
		if !ok {
			break
		}
		if isNull {
			result = append(result, math.NaN())
		} else {
			result = append(result, v)
		}
	}
	return result, nil
}

// CollectStrings materializes a column's string values into a Go slice.
func CollectStrings(col Column) ([]string, error) {
	iter, ok := col.(IterableColumn)
	if !ok {
		return nil, fmt.Errorf("dataset: column does not support iteration")
	}
	sit, err := iter.Strings()
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, col.Len())
	for {
		v, _, ok := sit.Next()
		if !ok {
			break
		}
		result = append(result, v)
	}
	return result, nil
}

// Describe returns a printable schema summary of a dataset.
func Describe(ds Dataset) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Dataset: %d rows × %d columns\n", ds.Len(), len(ds.Columns())))
	for _, colName := range ds.Columns() {
		col, err := ds.Column(colName)
		if err != nil {
			sb.WriteString(fmt.Sprintf("  %-20s (error: %v)\n", colName, err))
			continue
		}
		sb.WriteString(fmt.Sprintf("  %-20s %s[%d]\n", colName, col.DType(), col.Len()))
	}
	return sb.String()
}

// ReplaceColumn builds a new dataset where the named column is replaced
// with the given float64 values. All other columns are preserved by
// materializing them through CollectFloat64s or CollectStrings.
func ReplaceColumn(ds Dataset, colName string, vals []float64) (Dataset, error) {
	names := ds.Columns()
	var opts []DataFrameOpt
	for _, name := range names {
		if name == colName {
			opts = append(opts, WithFloat64s(name, vals))
			continue
		}
		col, err := ds.Column(name)
		if err != nil {
			continue
		}

		// Materialize: try float64 first, then string.
		if fvals, err := CollectFloat64s(col); err == nil {
			opts = append(opts, WithFloat64s(name, fvals))
		} else if svals, err := CollectStrings(col); err == nil {
			opts = append(opts, WithStrings(name, svals))
		}
	}
	return NewMixedDataFrame(opts...)
}

// GroupBy splits a dataset by unique values in the given column.
// Returns sorted group labels and a corresponding sub-dataset for each group.
// Supports string, int64, and float64 columns (all are converted to string labels).
// If the column is not found or has no rows, it falls back gracefully.
func GroupBy(ds Dataset, colName string) ([]string, []Dataset) {
	col, err := ds.Column(colName)
	if err != nil {
		return []string{""}, []Dataset{ds}
	}

	n := ds.Len()
	if n == 0 {
		return nil, nil
	}

	labels := make([]string, n)
	iter, ok := col.(IterableColumn)
	if !ok {
		return []string{""}, []Dataset{ds}
	}

	strIter, err := iter.Strings()
	if err != nil {
		// Fall back to float64 representation.
		fIter, ferr := iter.Float64s()
		if ferr != nil {
			return []string{""}, []Dataset{ds}
		}
		for i := 0; i < n; i++ {
			v, isNull, ok := fIter.Next()
			if !ok {
				break
			}
			if isNull {
				labels[i] = "NA"
			} else {
				labels[i] = fmt.Sprintf("%g", v)
			}
		}
	} else {
		for i := 0; i < n; i++ {
			v, _, ok := strIter.Next()
			if !ok {
				break
			}
			labels[i] = v
		}
	}

	// Unique groups, sorted for deterministic ordering.
	seen := make(map[string]bool)
	var groups []string
	for _, l := range labels {
		if !seen[l] {
			seen[l] = true
			groups = append(groups, l)
		}
	}
	sort.Strings(groups)

	// Build sub-datasets via FilterMask.
	subsets := make([]Dataset, len(groups))
	for gi, grp := range groups {
		mask := make([]bool, n)
		for i, l := range labels {
			mask[i] = (l == grp)
		}
		subsets[gi] = FilterMask(ds, mask)
	}

	return groups, subsets
}

// --- Compile-time interface checks ---

var (
	_ IterableColumn            = (*Float64Column)(nil)
	_ Aggregator                = (*Float64Column)(nil)
	_ NativeColumnSliceProvider = (*Float64Column)(nil)
	_ NativeFilterProvider      = (*Float64Column)(nil)

	_ IterableColumn            = (*Int64Column)(nil)
	_ Aggregator                = (*Int64Column)(nil)
	_ NativeColumnSliceProvider = (*Int64Column)(nil)
	_ NativeFilterProvider      = (*Int64Column)(nil)

	_ IterableColumn            = (*StringColumn)(nil)
	_ NativeColumnSliceProvider = (*StringColumn)(nil)
	_ NativeFilterProvider      = (*StringColumn)(nil)

	_ Dataset = (*memoryDataset)(nil)
)
