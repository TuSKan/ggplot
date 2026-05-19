package stat

import (
	"context"
	"fmt"
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// BoxplotOption configures the BoxplotY transform.
type BoxplotOption func(*boxplotConfig)

type boxplotConfig struct {
	Whisker string // "tukey" (default, 1.5×IQR) or "range" (min-max)
	Notch   bool   // compute notch confidence interval around median
}

// WithWhisker sets the whisker rule: "tukey" (1.5×IQR, default) or "range" (min-max).
func WithWhisker(rule string) BoxplotOption {
	return func(c *boxplotConfig) { c.Whisker = rule }
}

// WithNotch enables notched boxplots (95% CI around median).
func WithNotch(enabled bool) BoxplotOption {
	return func(c *boxplotConfig) { c.Notch = enabled }
}

// BoxplotY returns a Transform that computes the five-number summary
// for each unique X group. Produces x, lower, q1, middle, q3, upper,
// notch_lower, notch_upper columns.
// Uses engine-native StatKernel.Boxplot — zero materialization in stat/.
func BoxplotY(opts ...BoxplotOption) Transform {
	cfg := boxplotConfig{Whisker: "tukey"}
	for _, o := range opts {
		o(&cfg)
	}

	return &boxplotTransform{cfg: cfg}
}

type boxplotTransform struct {
	cfg boxplotConfig
}

func (b *boxplotTransform) Name() string { return "boxplotY" }

func (b *boxplotTransform) OutputSchema() []string {
	return []string{"lower", "middle", "notch_lower", "notch_upper", "q1", "q3", "upper", "x"}
}

func (b *boxplotTransform) OutputMapping() map[string]string {
	return map[string]string{"x": "x", "y": "middle"}
}

func (b *boxplotTransform) OutputHints() map[string]ChannelHint { return nil }

func (b *boxplotTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	yName := in.Mapping["y"]
	if yName == "" {
		return TransformResult{}, fmt.Errorf("boxplotY: missing 'y' aesthetic: %w", ErrMissingColumn)
	}

	// Dispatch to engine StatKernel.
	eng := dataset.GetEngine(in.Data.Table())
	if eng == nil {
		return TransformResult{}, fmt.Errorf("boxplotY: no engine: %w", ErrUnsupportedType)
	}

	sk, ok := eng.(dataset.StatKernel)
	if !ok {
		return TransformResult{}, fmt.Errorf("boxplotY: engine %q: StatKernel: %w", eng.Name(), ErrUnsupportedType)
	}

	yCol, err := in.Data.Column(yName)
	if err != nil {
		return TransformResult{}, fmt.Errorf("boxplotY: %w", err)
	}

	// Optional X column for grouping.
	var groupCol dataset.AnyColumn

	xName := in.Mapping["x"]
	if xName != "" {
		groupCol, _ = in.Data.Column(xName)
	}

	tbl, err := sk.Boxplot(yCol, groupCol, b.cfg.Whisker, b.cfg.Notch)
	if err != nil {
		return TransformResult{}, fmt.Errorf("boxplotY: %w", err)
	}

	outData := dataset.From(tbl)

	outMapping := make(map[string]string, len(in.Mapping)+len(b.OutputMapping()))
	maps.Copy(outMapping, in.Mapping)
	maps.Copy(outMapping, b.OutputMapping())

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
