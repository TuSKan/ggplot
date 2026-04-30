package scale

import (
	"math"

	"github.com/TuSKan/ggplot/dataset"
)

// ScaleOpt is a functional option that configures a [ConfiguredScale].
type ScaleOpt func(*ConfiguredScale)

// WithBreaks sets explicit tick positions that override the automatic
// [Scale.Ticks] generation. The values are in data space.
func WithBreaks(breaks []float64) ScaleOpt {
	return func(c *ConfiguredScale) {
		c.breaks = make([]float64, len(breaks))
		copy(c.breaks, breaks)
	}
}

// WithLabels sets explicit tick labels corresponding to the positions
// produced by [WithBreaks] (same index order). If len(labels) < len(breaks),
// excess ticks use the inner scale's Format. If [WithBreaks] is not set,
// labels are matched to auto-generated ticks positionally.
func WithLabels(labels []string) ScaleOpt {
	return func(c *ConfiguredScale) {
		c.labels = make([]string, len(labels))
		copy(c.labels, labels)
	}
}

// WithFormatter sets a custom formatting function that replaces
// the inner scale's [Scale.Format] for every tick value.
// When both WithFormatter and WithLabels are set, WithLabels takes
// priority for positions that have a matching index.
func WithFormatter(fn func(float64) string) ScaleOpt {
	return func(c *ConfiguredScale) { c.formatter = fn }
}

// WithExpand controls the padding added to the scale domain after training.
// mult is a multiplicative fraction of the data range added to each side
// (e.g. 0.05 = 5 %); add is an absolute amount added to each side in data
// units. This replaces the renderer's default 5 % expansion for the axis.
//
// ggplot2 equivalent: scale_x_continuous(expand = expansion(mult, add))
func WithExpand(mult, add float64) ScaleOpt {
	return func(c *ConfiguredScale) {
		c.expandMult = mult
		c.expandAdd = add
		c.hasExpand = true
	}
}

// WithMinorBreaks sets explicit minor tick positions (data space).
// Minor ticks are used to draw minor grid lines between major ticks.
func WithMinorBreaks(breaks []float64) ScaleOpt {
	return func(c *ConfiguredScale) {
		c.minorBreaks = make([]float64, len(breaks))
		copy(c.minorBreaks, breaks)
	}
}

// WithClipBounds sets a visible-window clip range that is independent of
// the data domain. Data points outside this range are still present in the
// dataset (no filtering), but the axis shows only [min, max].
// Use math.NaN() for either bound to leave it auto-detected.
func WithClipBounds(min, max float64) ScaleOpt {
	return func(c *ConfiguredScale) {
		if !math.IsNaN(min) {
			c.clipMin = &min
		}
		if !math.IsNaN(max) {
			c.clipMax = &max
		}
	}
}

// ---------------------------------------------------------------------------

// Configure wraps an existing [Scale] with the given options.
// If no options are provided the inner scale is returned unchanged.
func Configure(inner Scale, opts ...ScaleOpt) Scale {
	if len(opts) == 0 {
		return inner
	}
	cs := &ConfiguredScale{inner: inner}
	for _, o := range opts {
		o(cs)
	}
	return cs
}

// ConfiguredScale is a decorator that wraps any Scale and overrides its
// behavior based on user-supplied configuration.
type ConfiguredScale struct {
	inner Scale

	breaks      []float64            // WithBreaks
	labels      []string             // WithLabels
	formatter   func(float64) string // WithFormatter
	expandMult  float64              // WithExpand multiplicative
	expandAdd   float64              // WithExpand additive
	hasExpand   bool                 // true when WithExpand was called
	minorBreaks []float64            // WithMinorBreaks
	clipMin     *float64             // WithClipBounds min
	clipMax     *float64             // WithClipBounds max
}

// Inner returns the underlying scale.
func (c *ConfiguredScale) Inner() Scale { return c.inner }

// --- Scale interface ---

// Train delegates to the inner scale.
func (c *ConfiguredScale) Train(col dataset.AnyColumn) error {
	return c.inner.Train(col)
}

// Map normalizes a data value using the effective bounds (which may
// include expand and clip adjustments).
func (c *ConfiguredScale) Map(v float64) float64 {
	mn, mx := c.Bounds()
	if mx == mn {
		return 0.5
	}
	return (v - mn) / (mx - mn)
}

// Inverse maps a [0,1] normalized value back to data space using the
// effective bounds.
func (c *ConfiguredScale) Inverse(v float64) float64 {
	mn, mx := c.Bounds()
	return mn + v*(mx-mn)
}

// Ticks returns user-supplied breaks if set, otherwise delegates.
func (c *ConfiguredScale) Ticks(n int) []float64 {
	if len(c.breaks) > 0 {
		return c.breaks
	}
	return c.inner.Ticks(n)
}

// Format returns the display string for a data value. Priority:
//  1. WithLabels (positional match against Ticks)
//  2. WithFormatter
//  3. inner.Format
func (c *ConfiguredScale) Format(v float64) string {
	if len(c.labels) > 0 {
		ticks := c.Ticks(0)
		for i, t := range ticks {
			if math.Abs(t-v) < 1e-12 {
				if i < len(c.labels) {
					return c.labels[i]
				}
				break
			}
		}
	}
	if c.formatter != nil {
		return c.formatter(v)
	}
	return c.inner.Format(v)
}

// Bounds returns the effective domain, applying clip bounds and expand.
// Priority: clipBounds > expand > inner.Bounds.
func (c *ConfiguredScale) Bounds() (float64, float64) {
	mn, mx := c.inner.Bounds()

	// Apply expand: widen by mult*range + add on each side.
	if c.hasExpand {
		rng := mx - mn
		mn -= rng*c.expandMult + c.expandAdd
		mx += rng*c.expandMult + c.expandAdd
	}

	// Apply clip bounds (overrides expand / inner).
	if c.clipMin != nil {
		mn = *c.clipMin
	}
	if c.clipMax != nil {
		mx = *c.clipMax
	}

	return mn, mx
}

// String delegates to the inner scale.
func (c *ConfiguredScale) String() string { return c.inner.String() }

// --- BoundsSetter ---

// SetBounds delegates to the inner scale (if it supports it).
func (c *ConfiguredScale) SetBounds(mn, mx float64) {
	if bs, ok := c.inner.(BoundsSetter); ok {
		bs.SetBounds(mn, mx)
	}
}

// --- MinorTicker ---

// MinorTicks returns user-supplied minor break positions if set.
// Otherwise it auto-generates midpoints between consecutive major ticks.
func (c *ConfiguredScale) MinorTicks() []float64 {
	if len(c.minorBreaks) > 0 {
		return c.minorBreaks
	}
	// Auto-generate: one midpoint between each pair of major ticks.
	major := c.Ticks(5)
	if len(major) < 2 {
		return nil
	}
	minor := make([]float64, 0, len(major)-1)
	for i := 0; i < len(major)-1; i++ {
		minor = append(minor, (major[i]+major[i+1])/2)
	}
	return minor
}

// --- Expander ---

// Expand returns the user-supplied multiplicative and additive expansion
// parameters.
func (c *ConfiguredScale) Expand() (mult, add float64) {
	return c.expandMult, c.expandAdd
}

// HasExpand reports whether the user explicitly set expand parameters.
func (c *ConfiguredScale) HasExpand() bool {
	return c.hasExpand
}

// --- Compile-time interface checks ---

var (
	_ Scale        = (*ConfiguredScale)(nil)
	_ BoundsSetter = (*ConfiguredScale)(nil)
	_ MinorTicker  = (*ConfiguredScale)(nil)
	_ Expander     = (*ConfiguredScale)(nil)
)
