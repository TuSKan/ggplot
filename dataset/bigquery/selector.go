package bigquery

import (
	"fmt"

	"github.com/TuSKan/ggplot/dataset"
)

// --- Selector ---

// SortIndices is not supported on remote data.
// Use Frame.Arrange which should generate ORDER BY SQL.
func (e *Engine) SortIndices(col dataset.AnyColumn) ([]int, error) {
	return nil, fmt.Errorf("bigquery: SortIndices not supported on remote data; use Frame.Arrange")
}

// FilterIndices evaluates a boolean mask to indices (local only).
func (e *Engine) FilterIndices(mask []bool) []int {
	indices := make([]int, 0)
	for i, v := range mask {
		if v {
			indices = append(indices, i)
		}
	}
	return indices
}

// Select applies Take (row gather by indices).
// For bqColumns: materializes first, then downloads and operates locally.
// This is inherently non-lazy — indices come from local computation.
func (e *Engine) Select(col dataset.AnyColumn, indices []int) (dataset.AnyColumn, error) {
	if bqCol, ok := col.(*bqColumn); ok {
		localDS, err := bqCol.ds.download()
		if err != nil {
			return nil, err
		}
		localCol, err := localDS.Column(bqCol.name)
		if err != nil {
			return nil, err
		}
		return e.localEngine().Select(localCol, indices)
	}
	return e.localEngine().Select(col, indices)
}

// Slice applies a range restriction on a column.
// For bqColumns: uses RowRestriction "col BETWEEN start AND end" — fully lazy.
func (e *Engine) Slice(col dataset.AnyColumn, start, end int) (dataset.AnyColumn, error) {
	if bqCol, ok := col.(*bqColumn); ok {
		restriction := fmt.Sprintf("`%s` BETWEEN %d AND %d", bqCol.Name(), start, end)
		newDS := bqCol.ds.withRestriction(restriction)
		return &bqColumn{
			ds:    newDS,
			name:  bqCol.name,
			dtype: bqCol.dtype,
		}, nil
	}
	return e.localEngine().Slice(col, start, end)
}

// errNotBQDataset returns a typed error for wrong dataset type.
func errNotBQDataset(op string) error {
	return fmt.Errorf("bigquery: %s requires a BigQuery dataset", op)
}
