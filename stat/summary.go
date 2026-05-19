package stat

import (
	"context"
	"fmt"
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// SummaryXY returns a Transform that computes mean(y) for each
// distinct x value. Produces x (sorted unique) and y (mean) columns.
// Uses engine-native GroupBy + Summarize(AggMean) — zero materialization.
func SummaryXY() Transform {
	return &summaryTransform{}
}

type summaryTransform struct{}

func (s *summaryTransform) Name() string { return "summaryXY" }

func (s *summaryTransform) OutputSchema() []string {
	return []string{"x", "y"}
}

func (s *summaryTransform) OutputMapping() map[string]string { return nil }

func (s *summaryTransform) OutputHints() map[string]ChannelHint { return nil }

func (s *summaryTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	xCol := in.Mapping["x"]
	yCol := in.Mapping["y"]

	if xCol == "" || yCol == "" {
		return TransformResult{}, fmt.Errorf("summaryXY: missing 'x' or 'y' aesthetic: %w", ErrMissingColumn)
	}

	// Engine-native: GroupBy(x) → Mean(y) → Arrange(x).
	outData := in.Data.
		GroupBy(xCol).
		Summarize(dataset.Mean(yCol, yCol)).
		Arrange(xCol)

	outMapping := make(map[string]string, len(in.Mapping))
	maps.Copy(outMapping, in.Mapping)

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
