package stat

import (
	"context"
	"fmt"
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// BinOption configures the BinX / BinY transform.
type BinOption func(*binConfig)

type binConfig struct {
	Bins      int    // explicit bin count; 0 = auto
	BinMethod string // "sturges" (default), "scott", "fd", "sqrt"
}

// WithBins sets an explicit bin count.
func WithBins(n int) BinOption { return func(c *binConfig) { c.Bins = n } }

// WithBinMethod selects the automatic binning strategy.
// Supported: "sturges" (default), "scott", "fd" (Freedman-Diaconis), "sqrt".
func WithBinMethod(m string) BinOption { return func(c *binConfig) { c.BinMethod = m } }

// BinX returns a Transform that bins the x channel into evenly-spaced
// bins, producing x (centers) and count (per-bin frequency).
// Uses engine-native StatKernel.Histogram — zero materialization in stat/.
func BinX(opts ...BinOption) Transform {
	cfg := binConfig{BinMethod: "sturges"}
	for _, o := range opts {
		o(&cfg)
	}

	return &binTransform{cfg: cfg}
}

type binTransform struct {
	cfg binConfig
}

func (b *binTransform) Name() string { return "binX" }

func (b *binTransform) OutputSchema() []string {
	return []string{"count", "x"}
}

func (b *binTransform) OutputMapping() map[string]string {
	return map[string]string{"x": "x", "y": "count"}
}

func (b *binTransform) OutputHints() map[string]ChannelHint {
	return map[string]ChannelHint{"y": HintCount}
}

func (b *binTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	xCol := in.Mapping["x"]
	if xCol == "" {
		return TransformResult{}, fmt.Errorf("binX: missing 'x' aesthetic: %w", ErrMissingColumn)
	}

	// Dispatch to engine StatKernel.
	eng := dataset.GetEngine(in.Data.Table())
	if eng == nil {
		return TransformResult{}, fmt.Errorf("binX: no engine: %w", ErrUnsupportedType)
	}

	sk, ok := eng.(dataset.StatKernel)
	if !ok {
		return TransformResult{}, fmt.Errorf("binX: engine %q: StatKernel: %w", eng.Name(), ErrUnsupportedType)
	}

	col, err := in.Data.Column(xCol)
	if err != nil {
		return TransformResult{}, fmt.Errorf("binX: %w", err)
	}

	tbl, err := sk.Histogram(col, b.cfg.Bins)
	if err != nil {
		return TransformResult{}, fmt.Errorf("binX: %w", err)
	}

	outData := dataset.From(tbl)

	outMapping := make(map[string]string, len(in.Mapping)+len(b.OutputMapping()))
	maps.Copy(outMapping, in.Mapping)
	maps.Copy(outMapping, b.OutputMapping())

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
