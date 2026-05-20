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
	Bins       int    // explicit bin count; 0 = auto
	BinMethod  string // "sturges" (default), "scott", "fd", "sqrt"
	Cumulative int    // 0=off, +1=forward cumulative, -1=reverse cumulative
}

// WithBins sets an explicit bin count.
func WithBins(n int) BinOption { return func(c *binConfig) { c.Bins = n } }

// WithBinMethod selects the automatic binning strategy.
// Supported: "sturges" (default), "scott", "fd" (Freedman-Diaconis), "sqrt".
func WithBinMethod(m string) BinOption { return func(c *binConfig) { c.BinMethod = m } }

// WithCumulative enables cumulative histogram mode.
// +1 = forward cumulative (left to right), -1 = reverse (right to left).
// 0 = off (default, standard histogram).
func WithCumulative(dir int) BinOption { return func(c *binConfig) { c.Cumulative = dir } }

// BinX returns a Transform that bins the x channel into evenly-spaced
// bins, producing x (centers) and count (per-bin frequency).
// Uses engine-native StatKernel.Histogram — zero materialization in stat/.
func BinX(opts ...BinOption) Transform {
	cfg := binConfig{BinMethod: "sturges"}
	for _, o := range opts {
		o(&cfg)
	}

	return &binTransform{cfg: cfg, axis: "x"}
}

// BinY returns a Transform that bins the y channel into evenly-spaced
// bins, producing y (centers) and count (per-bin frequency).
// This is the symmetric counterpart of [BinX] for horizontal histograms.
func BinY(opts ...BinOption) Transform {
	cfg := binConfig{BinMethod: "sturges"}
	for _, o := range opts {
		o(&cfg)
	}

	return &binTransform{cfg: cfg, axis: "y"}
}

type binTransform struct {
	cfg  binConfig
	axis string // "x" or "y"
}

func (b *binTransform) Name() string { return "bin" + b.axis }

func (b *binTransform) OutputSchema() []string {
	return []string{"count", b.axis}
}

func (b *binTransform) OutputMapping() map[string]string {
	// The binned axis keeps its name; the cross axis maps to "count".
	cross := "y"
	if b.axis == "y" {
		cross = "x"
	}

	return map[string]string{b.axis: b.axis, cross: "count"}
}

func (b *binTransform) OutputHints() map[string]ChannelHint {
	cross := "y"
	if b.axis == "y" {
		cross = "x"
	}

	if b.cfg.Cumulative != 0 {
		return map[string]ChannelHint{cross: HintCumulative}
	}

	return map[string]ChannelHint{cross: HintCount}
}

func (b *binTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	label := b.Name()

	srcCol := in.Mapping[b.axis]
	if srcCol == "" {
		return TransformResult{}, fmt.Errorf("%s: missing '%s' aesthetic: %w", label, b.axis, ErrMissingColumn)
	}

	// Dispatch to engine StatKernel.
	eng := dataset.GetEngine(in.Data.Table())
	if eng == nil {
		return TransformResult{}, fmt.Errorf("%s: no engine: %w", label, ErrUnsupportedType)
	}

	sk, ok := eng.(dataset.StatKernel)
	if !ok {
		return TransformResult{}, fmt.Errorf("%s: engine %q: StatKernel: %w", label, eng.Name(), ErrUnsupportedType)
	}

	col, err := in.Data.Column(srcCol)
	if err != nil {
		return TransformResult{}, fmt.Errorf("%s: %w", label, err)
	}

	tbl, err := sk.Histogram(col, b.cfg.Bins)
	if err != nil {
		return TransformResult{}, fmt.Errorf("%s: %w", label, err)
	}

	outData := dataset.From(tbl)

	// Histogram produces "x" (centers) + "count". For BinY, rename "x" → "y".
	if b.axis == "y" {
		outData = outData.Rename("x", "y")
	}

	// Apply cumulative sum to count column if requested.
	if b.cfg.Cumulative != 0 {
		outData, err = applyCumulative(outData, "count", b.cfg.Cumulative)
		if err != nil {
			return TransformResult{}, fmt.Errorf("%s: cumulative: %w", label, err)
		}
	}

	outMapping := make(map[string]string, len(in.Mapping)+len(b.OutputMapping()))
	maps.Copy(outMapping, in.Mapping)
	maps.Copy(outMapping, b.OutputMapping())

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}

// applyCumulative replaces a column with its forward (+1) or reverse (-1)
// cumulative sum. Uses engine-native Windower.CumSum — stays fully lazy.
func applyCumulative(ds dataset.Dataset, colName string, dir int) (dataset.Dataset, error) {
	if dir >= 0 {
		// Forward: input is collected, read column directly.
		eng := dataset.GetEngine(ds.Table())

		win, ok := eng.(dataset.Windower)
		if !ok {
			return dataset.Dataset{}, fmt.Errorf("cumulative: engine %q: Windower: %w", eng.Name(), dataset.ErrUnsupportedEngine)
		}

		col, err := ds.Column(colName)
		if err != nil {
			return dataset.Dataset{}, fmt.Errorf("cumulative: %w", err)
		}

		cumCol, err := win.CumSum(col)
		if err != nil {
			return dataset.Dataset{}, fmt.Errorf("cumulative: %w", err)
		}

		return ds.WithColumn(cumCol), nil // lazy
	}

	// Reverse: entire reverse→cumsum→reverse runs at Collect time via Mutate.
	return ds.Mutate(colName, &cumSumMutator{col: colName, reverse: true}), nil
}

// cumSumMutator implements dataset.MutateFunc.
// At Collect time it applies Windower.CumSum to the named column,
// optionally reversing row order before/after for reverse cumulative.
type cumSumMutator struct {
	col     string
	reverse bool
}

func (m *cumSumMutator) Apply(tbl dataset.Table) (dataset.AnyColumn, error) {
	eng := dataset.GetEngine(tbl)

	win, ok := eng.(dataset.Windower)
	if !ok {
		return nil, fmt.Errorf("cumulative: engine %q: Windower: %w", eng.Name(), dataset.ErrUnsupportedEngine)
	}

	col, err := tbl.Column(m.col)
	if err != nil {
		return nil, fmt.Errorf("cumulative: %w", err)
	}

	if !m.reverse {
		result, cumErr := win.CumSum(col)
		if cumErr != nil {
			return nil, fmt.Errorf("cumulative: %w", cumErr)
		}

		return result, nil
	}

	// Reverse: Select(reverse) → CumSum → Select(reverse).
	sel, ok := eng.(dataset.Selector)
	if !ok {
		return nil, fmt.Errorf("cumulative: engine %q: Selector: %w", eng.Name(), dataset.ErrUnsupportedEngine)
	}

	n := int(col.Len())
	revIdx := make([]int, n)

	for i := range revIdx {
		revIdx[i] = n - 1 - i
	}

	reversed, err := sel.Select(col, revIdx)
	if err != nil {
		return nil, fmt.Errorf("cumulative: reverse: %w", err)
	}

	cumCol, err := win.CumSum(reversed)
	if err != nil {
		return nil, fmt.Errorf("cumulative: %w", err)
	}

	result, selErr := sel.Select(cumCol, revIdx)
	if selErr != nil {
		return nil, fmt.Errorf("cumulative: reverse: %w", selErr)
	}

	return result, nil
}
