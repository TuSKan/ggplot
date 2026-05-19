package stat

import (
	"context"
	"fmt"
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// NormalizeOption configures NormalizeY / NormalizeX.
type NormalizeOption func(*normalizeConfig)

type normalizeConfig struct {
	total float64 // target sum; default 1.0
}

// WithTotal sets the target sum for normalization (default 1.0).
// Use 100 for percentage output.
func WithTotal(t float64) NormalizeOption { return func(c *normalizeConfig) { c.total = t } }

// NormalizeY rescales the y channel so values sum to the given total
// (default 1.0). Applied as a pipeline stage after transforms like
// BinX or Count to convert frequencies into proportions.
//
// Uses the engine's Aggregator.Sum for the column sum and
// MathKernel.MulScalar for element-wise scaling — no manual
// float64 loops. Returns a lazy Dataset.
//
// Output hint: y → HintProportion.
func NormalizeY(opts ...NormalizeOption) Transform {
	cfg := normalizeConfig{total: 1.0}
	for _, o := range opts {
		o(&cfg)
	}

	return &normalizeTransform{cfg: cfg, axis: "y"}
}

// NormalizeX rescales the x channel so values sum to the given total.
func NormalizeX(opts ...NormalizeOption) Transform {
	cfg := normalizeConfig{total: 1.0}
	for _, o := range opts {
		o(&cfg)
	}

	return &normalizeTransform{cfg: cfg, axis: "x"}
}

type normalizeTransform struct {
	cfg  normalizeConfig
	axis string
}

func (n *normalizeTransform) Name() string { return "normalize" + n.axis }

func (n *normalizeTransform) OutputSchema() []string { return nil }

func (n *normalizeTransform) OutputMapping() map[string]string { return nil }

func (n *normalizeTransform) OutputHints() map[string]ChannelHint {
	return map[string]ChannelHint{n.axis: HintProportion}
}

func (n *normalizeTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	colName := in.Mapping[n.axis]
	if colName == "" {
		return TransformResult{}, fmt.Errorf("normalize%s: missing %q aesthetic: %w", n.axis, n.axis, ErrMissingColumn)
	}

	eng := dataset.GetEngine(in.Data.Table())

	// Use Aggregator.Sum for the column total.
	agg, ok := eng.(dataset.Aggregator)
	if !ok {
		return TransformResult{}, fmt.Errorf("normalize%s: engine %q: Aggregator: %w",
			n.axis, eng.Name(), dataset.ErrUnsupportedEngine)
	}

	col, err := in.Data.Column(colName)
	if err != nil {
		return TransformResult{}, fmt.Errorf("normalize%s: %w", n.axis, err)
	}

	sumCol, err := agg.Sum(col)
	if err != nil {
		return TransformResult{}, fmt.Errorf("normalize%s: sum: %w", n.axis, err)
	}

	sum, ok := dataset.ScalarFloat64(sumCol)
	if !ok {
		return TransformResult(in), nil // zero or non-float64 sum — passthrough
	}

	scale := n.cfg.total / sum

	// Use MathKernel.MulScalar for element-wise scaling.
	mk, ok := eng.(dataset.MathKernel)
	if !ok {
		return TransformResult{}, fmt.Errorf("normalize%s: engine %q: MathKernel: %w",
			n.axis, eng.Name(), dataset.ErrUnsupportedEngine)
	}

	scaled, err := mk.MulScalar(col, scale)
	if err != nil {
		return TransformResult{}, fmt.Errorf("normalize%s: scale: %w", n.axis, err)
	}

	// Lazy: WithColumn builds a lazy op — materialized on Collect.
	outData := in.Data.WithColumn(scaled)

	outMapping := make(map[string]string, len(in.Mapping))
	maps.Copy(outMapping, in.Mapping)

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
