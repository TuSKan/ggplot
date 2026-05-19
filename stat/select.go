package stat

import (
	"context"
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// TopN returns a Transform that keeps the top N rows by the named
// column. Default order is descending (largest first); use Ascending()
// for smallest first.
// Uses Dataset.Arrange + Dataset.Head/Tail — stays lazy.
func TopN(n int, column string, opts ...SortOption) Transform {
	cfg := sortConfig{descending: true} // default: largest first
	for _, o := range opts {
		o(&cfg)
	}

	return &topNTransform{n: n, column: column, cfg: cfg}
}

type topNTransform struct {
	n      int
	column string
	cfg    sortConfig
}

func (t *topNTransform) Name() string                        { return "topN" }
func (t *topNTransform) OutputSchema() []string              { return nil }
func (t *topNTransform) OutputMapping() map[string]string    { return nil }
func (t *topNTransform) OutputHints() map[string]ChannelHint { return nil }

func (t *topNTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	// Lazy: sort ascending, then take head or tail.
	// Head/Tail handle the case where n >= row count.
	sorted := in.Data.Arrange(t.column)

	var outData dataset.Dataset
	if t.cfg.descending {
		outData = sorted.Tail(t.n)
	} else {
		outData = sorted.Head(t.n)
	}

	outMapping := make(map[string]string, len(in.Mapping))
	maps.Copy(outMapping, in.Mapping)

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
