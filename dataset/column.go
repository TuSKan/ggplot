package dataset

import (
	"fmt"
	"maps"
	"math"
)

// DType represents the logical data type of a column.
type DType int

const (
	// DTypeFloat64 is a 64-bit floating point column.
	DTypeFloat64 DType = iota
	// DTypeInt64 is a 64-bit integer column.
	DTypeInt64
	// DTypeString is a string/categorical column.
	DTypeString
	// DTypeBool is a boolean column.
	DTypeBool
	// DTypeUnknown is an unrecognized type.
	DTypeUnknown
)

// String returns the human-readable name of the data type.
func (d DType) String() string {
	switch d {
	case DTypeFloat64:
		return "float64"
	case DTypeInt64:
		return "int64"
	case DTypeString:
		return "string"
	case DTypeBool:
		return "bool"
	default:
		return "unknown"
	}
}

// Column represents an immutable 1-D data vector with type metadata.
type Column interface {
	// Len returns the number of elements in the column.
	Len() int

	// DType returns the column's logical data type.
	DType() DType
}

// IterableColumn extends Column with typed iteration. Concrete backends
// (Arrow, in-memory) implement this to provide streaming access to data
// without requiring full materialization.
type IterableColumn interface {
	Column
	// Float64s returns an iterator over numeric data. Returns an error
	// if the column cannot be interpreted as float64.
	Float64s() (Float64Iter, error)
	// Int64s returns an iterator over integer data.
	Int64s() (Int64Iter, error)
	// Strings returns an iterator over string/categorical data.
	Strings() (StringIter, error)
}

// Float64Iter streams float64 values one at a time.
type Float64Iter interface {
	// Next returns the next value, a null flag, and false at EOF.
	Next() (val float64, isNull bool, ok bool)
}

// Int64Iter streams int64 values one at a time.
type Int64Iter interface {
	// Next returns the next value, a null flag, and false at EOF.
	Next() (val int64, isNull bool, ok bool)
}

// StringIter streams string values one at a time.
type StringIter interface {
	// Next returns the next value, a null flag, and false at EOF.
	Next() (val string, isNull bool, ok bool)
}

// Aggregator provides fast-path Min/Max for column types that support
// it natively (e.g., Arrow arrays with SIMD-optimized scans).
type Aggregator interface {
	Min() (float64, error)
	Max() (float64, error)
}

// Min computes the minimum value of a column. If the column implements
// [Aggregator], the fast-path is used. Otherwise, it falls back to
// iterating through all float64 values.
func Min(col Column) (float64, error) {
	if col.Len() == 0 {
		return 0, fmt.Errorf("dataset: cannot compute Min of empty column")
	}
	if agg, ok := col.(Aggregator); ok {
		return agg.Min()
	}
	// Fallback: iterate float64s.
	iter, ok := col.(IterableColumn)
	if !ok {
		return 0, fmt.Errorf("dataset: column does not support iteration or Aggregator")
	}
	flt, err := iter.Float64s()
	if err != nil {
		return 0, err
	}
	result := math.MaxFloat64
	found := false
	for {
		v, isNull, ok := flt.Next()
		if !ok {
			break
		}
		if !isNull && v < result {
			result = v
			found = true
		}
	}
	if !found {
		return 0, fmt.Errorf("dataset: all values are null")
	}
	return result, nil
}

// Max computes the maximum value of a column. See [Min] for details.
func Max(col Column) (float64, error) {
	if col.Len() == 0 {
		return 0, fmt.Errorf("dataset: cannot compute Max of empty column")
	}
	if agg, ok := col.(Aggregator); ok {
		return agg.Max()
	}
	iter, ok := col.(IterableColumn)
	if !ok {
		return 0, fmt.Errorf("dataset: column does not support iteration or Aggregator")
	}
	flt, err := iter.Float64s()
	if err != nil {
		return 0, err
	}
	result := -math.MaxFloat64
	found := false
	for {
		v, isNull, ok := flt.Next()
		if !ok {
			break
		}
		if !isNull && v > result {
			result = v
			found = true
		}
	}
	if !found {
		return 0, fmt.Errorf("dataset: all values are null")
	}
	return result, nil
}

// Schema returns a map of column name → DType for the given dataset.
func Schema(ds Dataset) map[string]DType {
	m := make(map[string]DType)
	for _, name := range ds.Columns() {
		col, err := ds.Column(name)
		if err != nil {
			m[name] = DTypeUnknown
			continue
		}
		m[name] = col.DType()
	}
	return m
}

// NativeSliceProvider allows datasets to provide zero-copy slices
// via their native backend (e.g., Arrow chunked array slicing).
type NativeSliceProvider interface {
	SliceDataset(i, j int) Dataset
}

// NativeColumnSliceProvider allows columns to produce zero-copy sub-views.
type NativeColumnSliceProvider interface {
	SliceColumn(i, j int) Column
}

// NativeFilterProvider allows columns to apply boolean masks using
// their native representation (e.g., Arrow dictionary encoding).
type NativeFilterProvider interface {
	FilterColumn(mask []bool, count int) (Column, error)
}

// NativePushdownFilter is implemented by remote-backed datasets (SQL, etc.)
// that can push filter predicates to the server instead of evaluating locally.
type NativePushdownFilter interface {
	PushFilter(pred Predicate) (Dataset, error)
}

// NativePushdownGroupBy is implemented by remote-backed datasets that can
// push GROUP BY operations to the server.
type NativePushdownGroupBy interface {
	PushGroupBy(cols []string) (Dataset, error)
}

// MergeSchemas merges two column-name→DType maps, with b overriding a.
func MergeSchemas(a, b map[string]DType) map[string]DType {
	result := make(map[string]DType, len(a)+len(b))
	maps.Copy(result, a)
	maps.Copy(result, b)
	return result
}
