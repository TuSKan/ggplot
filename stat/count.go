package stat

import (
	"context"
	"fmt"
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// Count returns a Transform that counts occurrences of each unique
// x value. Produces x (unique values, sorted) and count columns.
// Uses engine-native GroupBy + Summarize(AggCount) — zero materialization.
func Count() Transform {
	return &countTransform{}
}

type countTransform struct{}

func (c *countTransform) Name() string { return "count" }

func (c *countTransform) OutputSchema() []string {
	return []string{"count", "x"}
}

func (c *countTransform) OutputMapping() map[string]string {
	return map[string]string{"x": "x", "y": "count"}
}

func (c *countTransform) OutputHints() map[string]ChannelHint {
	return map[string]ChannelHint{"y": HintCount}
}

func (c *countTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	xCol := in.Mapping["x"]
	if xCol == "" {
		return TransformResult{}, fmt.Errorf("count: missing 'x' aesthetic: %w", ErrMissingColumn)
	}

	// Engine-native: GroupBy(x) → Count → Arrange(x).
	outData := in.Data.
		GroupBy(xCol).
		Summarize(dataset.Count("count", xCol)).
		Arrange(xCol)

	outMapping := make(map[string]string, len(in.Mapping)+len(c.OutputMapping()))
	maps.Copy(outMapping, in.Mapping)
	maps.Copy(outMapping, c.OutputMapping())

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
