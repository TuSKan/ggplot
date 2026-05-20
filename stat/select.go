package stat

import (
	"context"
	"fmt"
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

// --- SelectRow ---

// SelectMode identifies which row to keep from a sorted column.
type SelectMode string

// Standard select modes.
const (
	SelectFirst SelectMode = "first" // keep first row (smallest value)
	SelectLast  SelectMode = "last"  // keep last row (largest value)
	SelectMin   SelectMode = "min"   // keep row with minimum value
	SelectMax   SelectMode = "max"   // keep row with maximum value
)

// SelectRow returns a Transform that selects a single row from the dataset
// based on the given mode and column. The mode determines which row is kept:
//   - [SelectFirst]: first row in natural order
//   - [SelectLast]:  last row in natural order
//   - [SelectMin]:   row with the smallest value in column
//   - [SelectMax]:   row with the largest value in column
//
// Uses engine-native Arrange + Head/Tail — stays lazy.
func SelectRow(mode SelectMode, column string) Transform {
	return &selectRowTransform{mode: mode, column: column}
}

type selectRowTransform struct {
	mode   SelectMode
	column string
}

func (s *selectRowTransform) Name() string                        { return "select" }
func (s *selectRowTransform) OutputSchema() []string              { return nil }
func (s *selectRowTransform) OutputMapping() map[string]string    { return nil }
func (s *selectRowTransform) OutputHints() map[string]ChannelHint { return nil }

func (s *selectRowTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	col := s.column
	if col == "" {
		return TransformResult{}, fmt.Errorf("select: missing column name: %w", ErrMissingColumn)
	}

	// Validate column exists.
	if _, err := in.Data.Column(col); err != nil {
		return TransformResult{}, fmt.Errorf("select: column %q: %w", col, err)
	}

	var outData dataset.Dataset

	switch s.mode {
	case SelectFirst:
		outData = in.Data.Head(1)
	case SelectLast:
		outData = in.Data.Tail(1)
	case SelectMin:
		outData = in.Data.Arrange(col).Head(1)
	case SelectMax:
		outData = in.Data.Arrange(col).Tail(1)
	default:
		return TransformResult{}, fmt.Errorf("select: unknown mode %q: %w", s.mode, ErrUnsupportedType)
	}

	outMapping := make(map[string]string, len(in.Mapping))
	maps.Copy(outMapping, in.Mapping)

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
