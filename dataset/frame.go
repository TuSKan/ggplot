package dataset

import "fmt"

// Frame provides a dplyr-inspired fluent API for lazy dataset transformations.
// All verbs return a new Frame backed by a lazy Dataset wrapper; no data is
// copied until a consumer reads columns.
//
// Usage:
//
//	result := dataset.From(ds).
//	    Select("x", "y", "group").
//	    Filter(dataset.Gt("x", 0)).
//	    Mutate("z", dataset.MapFloat64("x", func(v float64) float64 { return v * 2 })).
//	    Arrange("x").
//	    Head(100)
//
//	// Use result.Dataset in a plot:
//	p := ggplot.New(result.Dataset, ...)
type Frame struct {
	// Dataset is the underlying lazy dataset. Exported so it can be passed
	// directly to consumers that accept Dataset.
	Dataset Dataset
}

// From wraps a Dataset in a Frame for fluent ETL operations.
func From(ds Dataset) Frame { return Frame{Dataset: ds} }

// Select keeps only the named columns, lazily.
func (f Frame) Select(cols ...string) Frame {
	return Frame{Dataset: &selectDataset{parent: f.Dataset, cols: cols}}
}

// Filter keeps rows matching the predicate, lazily.
// If the underlying dataset supports [NativePushdownFilter] (e.g., SQL),
// the predicate is pushed down to the backend.
func (f Frame) Filter(pred Predicate) Frame {
	if pd, ok := f.Dataset.(NativePushdownFilter); ok {
		pushed, err := pd.PushFilter(pred)
		if err == nil {
			return Frame{Dataset: pushed}
		}
	}
	return Frame{Dataset: &lazyFilterDataset{parent: f.Dataset, pred: pred}}
}

// Mutate adds or replaces a column with a lazily-computed derivation.
func (f Frame) Mutate(name string, fn ColumnFunc) Frame {
	return Frame{Dataset: &mutateDataset{
		parent:  f.Dataset,
		colName: name,
		fn:      fn,
	}}
}

// Rename lazily renames a column from oldName to newName.
func (f Frame) Rename(oldName, newName string) Frame {
	return Frame{Dataset: &renameDataset{parent: f.Dataset, old: oldName, new: newName}}
}

// Head returns the first n rows, lazily.
func (f Frame) Head(n int) Frame {
	length := f.Dataset.Len()
	if n > length {
		n = length
	}
	return f.Slice(0, n)
}

// Tail returns the last n rows, lazily.
func (f Frame) Tail(n int) Frame {
	length := f.Dataset.Len()
	if n > length {
		n = length
	}
	return f.Slice(length-n, length)
}

// Slice returns rows [i, j), lazily with zero-copy if supported.
func (f Frame) Slice(i, j int) Frame {
	return Frame{Dataset: sliceDatasetFrom(f.Dataset, i, j)}
}

// Distinct returns unique rows based on the given columns.
// If no columns are specified, all columns are used.
func (f Frame) Distinct(cols ...string) Frame {
	return Frame{Dataset: &distinctDataset{parent: f.Dataset, cols: cols}}
}

// Arrange sorts by a column. Use desc=true for descending order.
func (f Frame) Arrange(col string, desc ...bool) Frame {
	d := false
	if len(desc) > 0 {
		d = desc[0]
	}
	return Frame{Dataset: &arrangeDataset{parent: f.Dataset, col: col, desc: d}}
}

// GroupBy groups the dataset by one or more columns, returning a
// [GroupedFrame] that supports Summarize and Ungroup.
func (f Frame) GroupBy(cols ...string) GroupedFrame {
	return GroupedFrame{parent: f.Dataset, groupCols: cols}
}

// Collect forces materialization of the entire lazy pipeline into an
// in-memory dataset. This is useful when you need to iterate multiple
// times or when the underlying source is expensive to re-evaluate.
func (f Frame) Collect() (Frame, error) {
	return f.collect()
}

// Len is a convenience accessor for f.Dataset.Len().
func (f Frame) Len() int { return f.Dataset.Len() }

// Columns is a convenience accessor for f.Dataset.Columns().
func (f Frame) Columns() []string { return f.Dataset.Columns() }

// Column is a convenience accessor for f.Dataset.Column(name).
func (f Frame) Column(name string) (Column, error) { return f.Dataset.Column(name) }

// --- Column derivation functions ---

// ColumnFunc is a lazy column derivation function used by [Frame.Mutate].
type ColumnFunc func(ds Dataset) (Column, error)

// MapFloat64 creates a [ColumnFunc] that transforms a float64 column element-wise.
func MapFloat64(srcCol string, fn func(float64) float64) ColumnFunc {
	return func(ds Dataset) (Column, error) {
		col, err := ds.Column(srcCol)
		if err != nil {
			return nil, err
		}
		iter, ok := col.(IterableColumn)
		if !ok {
			return nil, fmt.Errorf("dataset: column %q does not support iteration", srcCol)
		}
		return &transformedFloat64Column{parent: iter, mapper: fn}, nil
	}
}

// MapString creates a [ColumnFunc] that transforms a string column to float64.
func MapString(srcCol string, fn func(string) float64) ColumnFunc {
	return func(ds Dataset) (Column, error) {
		col, err := ds.Column(srcCol)
		if err != nil {
			return nil, err
		}
		iter, ok := col.(IterableColumn)
		if !ok {
			return nil, fmt.Errorf("dataset: column %q does not support string iteration", srcCol)
		}
		return &transformedStringColumn{parent: iter, mapper: fn}, nil
	}
}

// ConstFloat64 creates a [ColumnFunc] that returns a column filled with a constant.
func ConstFloat64(val float64) ColumnFunc {
	return func(ds Dataset) (Column, error) {
		return &constFloat64Column{val: val, length: ds.Len()}, nil
	}
}

// Expr creates a [ColumnFunc] from a multi-column expression.
// The function receives the full dataset for arbitrary column access.
func Expr(fn func(ds Dataset) (Column, error)) ColumnFunc {
	return fn
}

// --- GroupedFrame ---

// GroupedFrame represents a dataset partitioned by one or more grouping columns.
type GroupedFrame struct {
	parent    Dataset
	groupCols []string
}

// Summarize aggregates each group using the given aggregation functions,
// producing a new Frame with one row per group.
func (g GroupedFrame) Summarize(aggs ...Aggregation) Frame {
	return Frame{Dataset: &summarizeDataset{
		parent:    g.parent,
		groupCols: g.groupCols,
		aggs:      aggs,
	}}
}

// Ungroup returns the underlying dataset as a flat Frame.
func (g GroupedFrame) Ungroup() Frame {
	return Frame{Dataset: g.parent}
}

// Aggregation describes a named aggregation operation for [GroupedFrame.Summarize].
type Aggregation struct {
	Name   string        // output column name
	Source string        // source column name
	Fn     AggregationFn // aggregation function
}

// AggregationFn is a function that reduces a slice of float64 values to a scalar.
type AggregationFn func(vals []float64) float64

// --- Aggregation constructors ---

// Mean creates an aggregation that computes the arithmetic mean.
func Mean(name, col string) Aggregation {
	return Aggregation{Name: name, Source: col, Fn: func(vals []float64) float64 {
		if len(vals) == 0 {
			return 0
		}
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		return sum / float64(len(vals))
	}}
}

// Sum creates an aggregation that computes the sum.
func Sum(name, col string) Aggregation {
	return Aggregation{Name: name, Source: col, Fn: func(vals []float64) float64 {
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s
	}}
}

// Count creates an aggregation that counts non-null values.
func Count(name, col string) Aggregation {
	return Aggregation{Name: name, Source: col, Fn: func(vals []float64) float64 {
		return float64(len(vals))
	}}
}

// Min aggregation computes the minimum.
func MinAgg(name, col string) Aggregation {
	return Aggregation{Name: name, Source: col, Fn: func(vals []float64) float64 {
		if len(vals) == 0 {
			return 0
		}
		m := vals[0]
		for _, v := range vals[1:] {
			if v < m {
				m = v
			}
		}
		return m
	}}
}

// Max aggregation computes the maximum.
func MaxAgg(name, col string) Aggregation {
	return Aggregation{Name: name, Source: col, Fn: func(vals []float64) float64 {
		if len(vals) == 0 {
			return 0
		}
		m := vals[0]
		for _, v := range vals[1:] {
			if v > m {
				m = v
			}
		}
		return m
	}}
}

// Median creates an aggregation that computes the median.
func Median(name, col string) Aggregation {
	return Aggregation{Name: name, Source: col, Fn: func(vals []float64) float64 {
		if len(vals) == 0 {
			return 0
		}
		sorted := make([]float64, len(vals))
		copy(sorted, vals)
		sortFloats(sorted)
		mid := len(sorted) / 2
		if len(sorted)%2 == 0 {
			return (sorted[mid-1] + sorted[mid]) / 2
		}
		return sorted[mid]
	}}
}

// Variance creates an aggregation that computes the sample variance.
func Variance(name, col string) Aggregation {
	return Aggregation{Name: name, Source: col, Fn: func(vals []float64) float64 {
		if len(vals) < 2 {
			return 0
		}
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		mean := sum / float64(len(vals))
		ss := 0.0
		for _, v := range vals {
			d := v - mean
			ss += d * d
		}
		return ss / float64(len(vals)-1)
	}}
}

// Agg creates a custom aggregation.
func Agg(name, col string, fn AggregationFn) Aggregation {
	return Aggregation{Name: name, Source: col, Fn: fn}
}

func sortFloats(s []float64) {
	// Simple insertion sort for typical small group sizes.
	for i := 1; i < len(s); i++ {
		j := i
		for j > 0 && s[j] < s[j-1] {
			s[j], s[j-1] = s[j-1], s[j]
			j--
		}
	}
}
