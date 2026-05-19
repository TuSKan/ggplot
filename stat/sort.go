package stat

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/TuSKan/ggplot/dataset"
)

// SortOption configures SortBy and TopN.
type SortOption func(*sortConfig)

type sortConfig struct {
	descending bool
}

// Desc makes SortBy sort in descending order.
func Desc() SortOption { return func(c *sortConfig) { c.descending = true } }

// Ascending makes SortBy sort in ascending order (the default).
func Ascending() SortOption { return func(c *sortConfig) { c.descending = false } }

// SortBy returns a Transform that sorts all rows by the named column.
// Default order is ascending; use Desc() for descending.
// Uses the engine's Selector.SortIndices + Dataset.SelectRows.
func SortBy(column string, opts ...SortOption) Transform {
	cfg := sortConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	return &sortTransform{column: column, cfg: cfg}
}

// ReverseRows returns a Transform that reverses the row order.
// Uses Dataset.SelectRows for engine-native scatter-gather.
func ReverseRows() Transform {
	return &reverseTransform{}
}

// --- sortTransform ---

type sortTransform struct {
	column string
	cfg    sortConfig
}

func (s *sortTransform) Name() string                        { return "sortBy" }
func (s *sortTransform) OutputSchema() []string              { return nil }
func (s *sortTransform) OutputMapping() map[string]string    { return nil }
func (s *sortTransform) OutputHints() map[string]ChannelHint { return nil }

func (s *sortTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	// Resolve through mapping: "y" → actual column name (e.g. "sales").
	col := s.column
	if mapped, ok := in.Mapping[col]; ok {
		col = mapped
	}

	if !s.cfg.descending {
		// Ascending: lazy Arrange.
		outMapping := make(map[string]string, len(in.Mapping))
		maps.Copy(outMapping, in.Mapping)

		return TransformResult{Data: in.Data.Arrange(col), Mapping: outMapping}, nil
	}

	// Descending: use engine Selector.SortIndices, reverse, then SelectRows.
	eng := dataset.GetEngine(in.Data.Table())

	sel, ok := eng.(dataset.Selector)
	if !ok {
		return TransformResult{}, fmt.Errorf("sortBy: engine %q: Selector: %w",
			eng.Name(), dataset.ErrUnsupportedEngine)
	}

	sortCol, err := in.Data.Column(col)
	if err != nil {
		return TransformResult{}, fmt.Errorf("sortBy: %w", err)
	}

	indices, err := sel.SortIndices(sortCol)
	if err != nil {
		return TransformResult{}, fmt.Errorf("sortBy: %w", err)
	}

	slices.Reverse(indices)

	outData, err := in.Data.SelectRows(indices)
	if err != nil {
		return TransformResult{}, fmt.Errorf("sortBy: %w", err)
	}

	outMapping := make(map[string]string, len(in.Mapping))
	maps.Copy(outMapping, in.Mapping)

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}

// --- reverseTransform ---

type reverseTransform struct{}

func (r *reverseTransform) Name() string                        { return "reverse" }
func (r *reverseTransform) OutputSchema() []string              { return nil }
func (r *reverseTransform) OutputMapping() map[string]string    { return nil }
func (r *reverseTransform) OutputHints() map[string]ChannelHint { return nil }

func (r *reverseTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	n := int(in.Data.NumRows())
	if n <= 1 {
		return TransformResult(in), nil
	}

	indices := make([]int, n)
	for i := range indices {
		indices[i] = n - 1 - i
	}

	outData, err := in.Data.SelectRows(indices)
	if err != nil {
		return TransformResult{}, fmt.Errorf("reverse: %w", err)
	}

	outMapping := make(map[string]string, len(in.Mapping))
	maps.Copy(outMapping, in.Mapping)

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
