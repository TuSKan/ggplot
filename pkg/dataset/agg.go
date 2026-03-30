package dataset

import "fmt"

// Aggregator exposes Min and Max for optimized boundary training.
// Concrete columns (like Arrow) implement this to provide fast-path.
type Aggregator interface {
	Min() (float64, error)
	Max() (float64, error)
}

// Min computes the minimum value in a column.
// Returns an error if the column does not support numerical aggregation or is empty.
func Min(col Column) (float64, error) {
	if col.Len() == 0 {
		return 0, fmt.Errorf("dataset: cannot calculate Min of empty column")
	}

	if agg, ok := col.(Aggregator); ok {
		return agg.Min()
	}
	// A generic fallback could use typed accessors, but for now we expect implemented Aggregators
	return 0, fmt.Errorf("dataset: column does not implement Min")
}

// Max computes the maximum value in a column.
func Max(col Column) (float64, error) {
	if col.Len() == 0 {
		return 0, fmt.Errorf("dataset: cannot calculate Max of empty column")
	}

	if agg, ok := col.(Aggregator); ok {
		return agg.Max()
	}
	return 0, fmt.Errorf("dataset: column does not implement Max")
}
