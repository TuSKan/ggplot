package stat

import (
	"context"
	"fmt"
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// FilterY returns a Transform that keeps only rows where the y channel
// satisfies the given masker. The masker is evaluated by the engine
// (supports BigQuery SQL pushdown, Arrow compute kernels, etc.).
//
// Use dataset predicates to build maskers:
//
//	stat.FilterY(dataset.Gt("y", 25.0))
func FilterY(masker dataset.Masker) Transform {
	return &filterTransform{axis: "y", masker: masker}
}

// FilterX returns a Transform that keeps only rows where the x channel
// satisfies the given masker.
func FilterX(masker dataset.Masker) Transform {
	return &filterTransform{axis: "x", masker: masker}
}

type filterTransform struct {
	axis   string
	masker dataset.Masker
}

func (f *filterTransform) Name() string                        { return "filter" + f.axis }
func (f *filterTransform) OutputSchema() []string              { return nil }
func (f *filterTransform) OutputMapping() map[string]string    { return nil }
func (f *filterTransform) OutputHints() map[string]ChannelHint { return nil }

func (f *filterTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	col := in.Mapping[f.axis]
	if col == "" {
		return TransformResult{}, fmt.Errorf("filter%s: missing %q aesthetic: %w", f.axis, f.axis, ErrMissingColumn)
	}

	// Lazy: engine-native Dataset.Filter — materialized on Collect.
	outData := in.Data.Filter(f.masker)

	outMapping := make(map[string]string, len(in.Mapping))
	maps.Copy(outMapping, in.Mapping)

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
