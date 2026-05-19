package stat

import (
	"context"
	"fmt"
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// DensityOption configures the DensityX transform.
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

	return &densityTransform{cfg: cfg}
}

type densityTransform struct {
	cfg densityConfig
}

func (d *densityTransform) Name() string { return "densityX" }

func (d *densityTransform) OutputSchema() []string {
	return []string{"density", "x"}
}

func (d *densityTransform) OutputMapping() map[string]string {
	return map[string]string{"x": "x", "y": "density"}
}

func (d *densityTransform) OutputHints() map[string]ChannelHint { return nil }

func (d *densityTransform) Apply(ctx context.Context, in TransformInput) (TransformResult, error) {
	xCol := in.Mapping["x"]
	if xCol == "" {
		return TransformResult{}, fmt.Errorf("densityX: missing 'x' aesthetic: %w", ErrMissingColumn)
	}

	// Dispatch to engine StatKernel.
	eng := dataset.GetEngine(in.Data.Table())
	if eng == nil {
		return TransformResult{}, fmt.Errorf("densityX: no engine: %w", ErrUnsupportedType)
	}

	sk, ok := eng.(dataset.StatKernel)
	if !ok {
		return TransformResult{}, fmt.Errorf("densityX: engine %q: StatKernel: %w", eng.Name(), ErrUnsupportedType)
	}

	col, err := in.Data.Column(xCol)
	if err != nil {
		return TransformResult{}, fmt.Errorf("densityX: %w", err)
	}

	tbl, err := sk.KDE(ctx, col, d.cfg.Bandwidth, d.cfg.Points)
	if err != nil {
		return TransformResult{}, fmt.Errorf("densityX: %w", err)
	}

	outData := dataset.From(tbl)

	outMapping := make(map[string]string, len(in.Mapping)+len(d.OutputMapping()))
	maps.Copy(outMapping, in.Mapping)
	maps.Copy(outMapping, d.OutputMapping())

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
