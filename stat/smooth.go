package stat

import (
	"context"
	"fmt"
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// SmoothOption configures the SmoothXY transform.
type SmoothOption func(*smoothConfig)

type smoothConfig struct {
	Method string // "linear" or "loess" (default)
	NOut   int    // output grid size; 0 = 80
	SE     bool   // produce ymin/ymax confidence bands
}

// WithMethod sets the smoothing method: "linear" or "loess" (default).
func WithMethod(m string) SmoothOption { return func(c *smoothConfig) { c.Method = m } }

// WithNOut sets the output grid size.
func WithNOut(n int) SmoothOption { return func(c *smoothConfig) { c.NOut = n } }

// WithSmoothPoints is an alias for [WithNOut] for backward compatibility.
func WithSmoothPoints(n int) SmoothOption { return WithNOut(n) }

// WithSE enables 95% confidence band output (ymin, ymax columns).
func WithSE(se bool) SmoothOption { return func(c *smoothConfig) { c.SE = se } }

// SmoothXY returns a Transform that fits a smooth curve through (x, y) data.
// Produces x (grid) and y (fitted) columns.
// Uses engine-native StatKernel.LinearFit or LoessFit — zero materialization.
func SmoothXY(opts ...SmoothOption) Transform {
	cfg := smoothConfig{Method: "loess", NOut: 80}
	for _, o := range opts {
		o(&cfg)
	}

	return &smoothTransform{cfg: cfg}
}

type smoothTransform struct {
	cfg smoothConfig
}

func (s *smoothTransform) Name() string { return "smoothXY" }

func (s *smoothTransform) OutputSchema() []string {
	if s.cfg.SE {
		return []string{"x", "y", "ymin", "ymax"}
	}

	return []string{"x", "y"}
}

func (s *smoothTransform) OutputMapping() map[string]string { return nil }

func (s *smoothTransform) OutputHints() map[string]ChannelHint { return nil }

func (s *smoothTransform) Apply(ctx context.Context, in TransformInput) (TransformResult, error) {
	xName := in.Mapping["x"]
	yName := in.Mapping["y"]

	if xName == "" || yName == "" {
		return TransformResult{}, fmt.Errorf("smoothXY: missing 'x' or 'y' aesthetic: %w", ErrMissingColumn)
	}

	// Dispatch to engine StatKernel.
	eng := dataset.GetEngine(in.Data.Table())
	if eng == nil {
		return TransformResult{}, fmt.Errorf("smoothXY: no engine: %w", ErrUnsupportedType)
	}

	sk, ok := eng.(dataset.StatKernel)
	if !ok {
		return TransformResult{}, fmt.Errorf("smoothXY: engine %q: StatKernel: %w", eng.Name(), ErrUnsupportedType)
	}

	xCol, err := in.Data.Column(xName)
	if err != nil {
		return TransformResult{}, fmt.Errorf("smoothXY: %w", err)
	}

	yCol, err := in.Data.Column(yName)
	if err != nil {
		return TransformResult{}, fmt.Errorf("smoothXY: %w", err)
	}

	var tbl dataset.Table

	switch {
	case s.cfg.Method == "linear" && s.cfg.SE:
		tbl, err = sk.LinearFitSE(xCol, yCol, s.cfg.NOut)
	case s.cfg.Method == "linear":
		tbl, err = sk.LinearFit(xCol, yCol, s.cfg.NOut)
	case s.cfg.SE:
		tbl, err = sk.LoessFitSE(ctx, xCol, yCol, s.cfg.NOut)
	default: // "loess"
		tbl, err = sk.LoessFit(ctx, xCol, yCol, s.cfg.NOut)
	}

	if err != nil {
		return TransformResult{}, fmt.Errorf("smoothXY: %w", err)
	}

	outData := dataset.From(tbl)

	outMapping := make(map[string]string, len(in.Mapping))
	maps.Copy(outMapping, in.Mapping)

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
