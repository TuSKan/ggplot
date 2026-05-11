package bigquery

import (
	"fmt"

	"github.com/TuSKan/ggplot/dataset"
)

// --- Filterer: mask.Expr() → RowRestriction ---

// Filter applies a predicate as a RowRestriction on the bqDataset.
// No execution happens — just appends to the restriction string.
func (e *Engine) Filter(ds dataset.Table, mask dataset.Masker) (dataset.Table, error) {
	bq, ok := ds.(*bqDataset)
	if !ok {
		return nil, errNotBQDataset("Filter")
	}

	type expr interface{ Expr() string }
	if ex, ok := mask.(expr); ok {
		return bq.withRestriction(ex.Expr()), nil
	}

	return nil, fmt.Errorf("filter mask does not implement Expr(): %w", ErrUnsupportedType)
}
