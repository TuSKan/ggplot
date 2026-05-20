package stat

import (
	"context"
	"fmt"
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// DensityOption configures the DensityX / DensityY transform.
type DensityOption func(*densityConfig)

type densityConfig struct {
	Bandwidth float64 // KDE bandwidth; 0 = Silverman auto-select
	Points    int     // number of output grid points; 0 = 512
}

// WithBandwidth sets an explicit KDE bandwidth. 0 = Silverman auto-select.
func WithBandwidth(bw float64) DensityOption {
	return func(c *densityConfig) { c.Bandwidth = bw }
}

// WithDensityPoints sets the number of output grid points.
func WithDensityPoints(n int) DensityOption {
	return func(c *densityConfig) { c.Points = n }
}

// DensityX returns a Transform that computes a kernel density estimation
// on the x channel. Produces x (grid) and density columns.
// Uses engine-native StatKernel.KDE — zero materialization in stat/.
func DensityX(opts ...DensityOption) Transform {
	cfg := densityConfig{Points: 512}
	for _, o := range opts {
		o(&cfg)
	}

	return &densityTransform{cfg: cfg, axis: "x"}
}

// DensityY returns a Transform that computes a kernel density estimation
// on the y channel. Produces y (grid) and density columns.
// This is the symmetric counterpart of [DensityX] for horizontal density plots.
func DensityY(opts ...DensityOption) Transform {
	cfg := densityConfig{Points: 512}
	for _, o := range opts {
		o(&cfg)
	}

	return &densityTransform{cfg: cfg, axis: "y"}
}

type densityTransform struct {
	cfg  densityConfig
	axis string // "x" or "y"
}

func (d *densityTransform) Name() string { return "density" + d.axis }

func (d *densityTransform) OutputSchema() []string {
	return []string{"density", d.axis}
}

func (d *densityTransform) OutputMapping() map[string]string {
	cross := "y"
	if d.axis == "y" {
		cross = "x"
	}

	return map[string]string{d.axis: d.axis, cross: "density"}
}

func (d *densityTransform) OutputHints() map[string]ChannelHint { return nil }

func (d *densityTransform) Apply(ctx context.Context, in TransformInput) (TransformResult, error) {
	label := d.Name()

	srcCol := in.Mapping[d.axis]
	if srcCol == "" {
		return TransformResult{}, fmt.Errorf("%s: missing '%s' aesthetic: %w", label, d.axis, ErrMissingColumn)
	}

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

	tbl, err := sk.KDE(ctx, col, d.cfg.Bandwidth, d.cfg.Points)
	if err != nil {
		return TransformResult{}, fmt.Errorf("%s: %w", label, err)
	}

	outData := dataset.From(tbl)

	// KDE produces "x" (grid) + "density". For DensityY, rename "x" → "y".
	if d.axis == "y" {
		outData = outData.Rename("x", "y")
	}

	outMapping := make(map[string]string, len(in.Mapping)+len(d.OutputMapping()))
	maps.Copy(outMapping, in.Mapping)
	maps.Copy(outMapping, d.OutputMapping())

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
