package colormap

import (
	"fmt"
	"math"

	"github.com/gogpu/gg"
)

// Color is the canonical color type used throughout this package. It is a
// type alias for [gg.RGBA] (float-space components in [0,1] plus alpha), so
// user code can refer to colormap.Color without importing gg directly. The
// alias keeps the rendering pipeline in float space — no uint8 truncation
// when sampling or interpolating colormaps.
type Color = gg.RGBA

// Implementations should:
//   - Clamp t outside [0,1] to At(0)/At(1), or substitute Under/Over if set
//     via WithExtremes.
//   - Return Bad (or fully-transparent black) when t is NaN.
//   - Be safe for concurrent reads — Cmap values are immutable; mutators
//     (Reversed, Resampled, WithExtremes) return a new Cmap.
type Cmap interface {
	// At samples the colormap at t. t outside [0,1] is clamped (or replaced
	// by Under/Over). NaN inputs return Bad.
	At(t float64) gg.RGBA

	// Name returns the registered identifier.
	Name() string

	// N returns the number of distinct colors. 256 for LUT-backed cmaps,
	// len(colors) for [ListedCmap].
	N() int

	// Category returns the matplotlib taxonomy bucket.
	Category() Category

	// Reversed returns a Cmap with the gradient flipped (At(t) -> At(1-t)).
	Reversed() Cmap

	// Resampled returns a Cmap that quantizes t into n equally-sized bins.
	// Useful for legend swatches or stepped color bars.
	Resampled(n int) Cmap

	// WithExtremes returns a Cmap with explicit colors for out-of-range and
	// NaN inputs. Pass nil for any to keep the default (clamped At(0)/At(1)
	// for under/over, transparent for bad).
	WithExtremes(under, over, bad *gg.RGBA) Cmap
}

// extremes holds the optional out-of-range / NaN colors. Embedded by all
// concrete Cmap implementations so WithExtremes() shares one allocation.
type extremes struct {
	under *gg.RGBA
	over  *gg.RGBA
	bad   *gg.RGBA
}

// resolve dispatches t into the correct extreme, or returns false if t is
// in-range and the caller should sample normally.
func (e extremes) resolve(t float64) (gg.RGBA, bool) {
	switch {
	case math.IsNaN(t):
		if e.bad != nil {
			return *e.bad, true
		}
		return gg.RGBA{}, true
	case t < 0:
		if e.under != nil {
			return *e.under, true
		}
	case t > 1:
		if e.over != nil {
			return *e.over, true
		}
	}
	return gg.RGBA{}, false
}

// LinearSegmentedCmap is a continuous colormap backed by a 256-entry RGB LUT.
// It is the standard backing type for matplotlib's perceptually-uniform,
// sequential, diverging, and cyclic colormaps.
type LinearSegmentedCmap struct {
	name string
	cat  Category
	lut  [256][3]uint8
	ext  extremes
}

// NewLinearSegmented constructs a colormap from a 256-entry LUT.
func NewLinearSegmented(name string, cat Category, lut [256][3]uint8) *LinearSegmentedCmap {
	return &LinearSegmentedCmap{name: name, cat: cat, lut: lut}
}

// At samples the colormap by linearly interpolating between adjacent LUT entries.
func (c *LinearSegmentedCmap) At(t float64) gg.RGBA {
	if r, ok := c.ext.resolve(t); ok {
		return r
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	pos := t * 255
	lo := int(pos)
	if lo >= 255 {
		return rgbaFromLUT(c.lut[255])
	}
	hi := lo + 1
	frac := pos - float64(lo)
	a := rgbaFromLUT(c.lut[lo])
	b := rgbaFromLUT(c.lut[hi])
	return a.Lerp(b, frac)
}

func (c *LinearSegmentedCmap) Name() string       { return c.name }
func (c *LinearSegmentedCmap) N() int             { return 256 }
func (c *LinearSegmentedCmap) Category() Category { return c.cat }

func (c *LinearSegmentedCmap) Reversed() Cmap {
	var rev [256][3]uint8
	for i := 0; i < 256; i++ {
		rev[i] = c.lut[255-i]
	}
	return &LinearSegmentedCmap{
		name: c.name + "_r",
		cat:  c.cat,
		lut:  rev,
		ext:  c.ext,
	}
}

func (c *LinearSegmentedCmap) Resampled(n int) Cmap {
	if n < 1 {
		n = 1
	}
	colors := make([]gg.RGBA, n)
	if n == 1 {
		colors[0] = c.At(0.5)
	} else {
		for i := 0; i < n; i++ {
			colors[i] = c.At(float64(i) / float64(n-1))
		}
	}
	return &resampledCmap{
		base:   c,
		name:   fmt.Sprintf("%s_n%d", c.name, n),
		colors: colors,
		ext:    c.ext,
	}
}

func (c *LinearSegmentedCmap) WithExtremes(under, over, bad *gg.RGBA) Cmap {
	clone := *c
	clone.ext = mergeExtremes(c.ext, under, over, bad)
	return &clone
}

// ListedCmap is a discrete colormap defined by an explicit color list.
// It is the backing type for qualitative palettes (tab10, Set1, etc.) and
// also for resampled continuous cmaps.
type ListedCmap struct {
	name   string
	cat    Category
	colors []gg.RGBA
	ext    extremes
}

// NewListed constructs a discrete colormap from a slice of colors.
// The slice is copied; the caller may mutate the input afterwards safely.
func NewListed(name string, cat Category, colors []gg.RGBA) *ListedCmap {
	cp := make([]gg.RGBA, len(colors))
	copy(cp, colors)
	return &ListedCmap{name: name, cat: cat, colors: cp}
}

// At returns the i-th color where i = floor(t * N). t in [0,1] maps to
// indices 0..N-1.
func (c *ListedCmap) At(t float64) gg.RGBA {
	if r, ok := c.ext.resolve(t); ok {
		return r
	}
	n := len(c.colors)
	if n == 0 {
		return gg.RGBA{}
	}
	if t < 0 {
		return c.colors[0]
	}
	if t >= 1 {
		return c.colors[n-1]
	}
	idx := int(t * float64(n))
	if idx >= n {
		idx = n - 1
	}
	return c.colors[idx]
}

// Color returns the i-th color directly, cycling i modulo N.
// Use this when applying a qualitative palette to discrete categories.
func (c *ListedCmap) Color(i int) gg.RGBA {
	n := len(c.colors)
	if n == 0 {
		return gg.RGBA{}
	}
	idx := i % n
	if idx < 0 {
		idx += n
	}
	return c.colors[idx]
}

// Colors returns a copy of the color list.
func (c *ListedCmap) Colors() []gg.RGBA {
	cp := make([]gg.RGBA, len(c.colors))
	copy(cp, c.colors)
	return cp
}

func (c *ListedCmap) Name() string       { return c.name }
func (c *ListedCmap) N() int             { return len(c.colors) }
func (c *ListedCmap) Category() Category { return c.cat }

func (c *ListedCmap) Reversed() Cmap {
	rev := make([]gg.RGBA, len(c.colors))
	for i, col := range c.colors {
		rev[len(c.colors)-1-i] = col
	}
	return &ListedCmap{
		name:   c.name + "_r",
		cat:    c.cat,
		colors: rev,
		ext:    c.ext,
	}
}

func (c *ListedCmap) Resampled(n int) Cmap {
	if n < 1 {
		n = 1
	}
	colors := make([]gg.RGBA, n)
	if n == 1 {
		colors[0] = c.At(0.5)
	} else {
		for i := 0; i < n; i++ {
			colors[i] = c.At(float64(i) / float64(n-1))
		}
	}
	return &ListedCmap{
		name:   fmt.Sprintf("%s_n%d", c.name, n),
		cat:    c.cat,
		colors: colors,
		ext:    c.ext,
	}
}

func (c *ListedCmap) WithExtremes(under, over, bad *gg.RGBA) Cmap {
	clone := *c
	clone.colors = append([]gg.RGBA(nil), c.colors...)
	clone.ext = mergeExtremes(c.ext, under, over, bad)
	return &clone
}

// resampledCmap wraps a base cmap with a fixed-size sampling grid. We keep
// the base reference for Reversed() so reversal stays accurate.
type resampledCmap struct {
	base   Cmap
	name   string
	colors []gg.RGBA
	ext    extremes
}

func (c *resampledCmap) At(t float64) gg.RGBA {
	if r, ok := c.ext.resolve(t); ok {
		return r
	}
	n := len(c.colors)
	if n == 0 {
		return gg.RGBA{}
	}
	if t < 0 {
		return c.colors[0]
	}
	if t >= 1 {
		return c.colors[n-1]
	}
	idx := int(t * float64(n))
	if idx >= n {
		idx = n - 1
	}
	return c.colors[idx]
}

func (c *resampledCmap) Name() string       { return c.name }
func (c *resampledCmap) N() int             { return len(c.colors) }
func (c *resampledCmap) Category() Category { return c.base.Category() }
func (c *resampledCmap) Reversed() Cmap     { return c.base.Reversed().Resampled(len(c.colors)) }

func (c *resampledCmap) Resampled(n int) Cmap { return c.base.Resampled(n) }

func (c *resampledCmap) WithExtremes(under, over, bad *gg.RGBA) Cmap {
	clone := *c
	clone.ext = mergeExtremes(c.ext, under, over, bad)
	return &clone
}

// rgbaFromLUT converts a 3-byte LUT entry to a fully-opaque gg.RGBA in
// [0,1] float space.
func rgbaFromLUT(e [3]uint8) gg.RGBA {
	return gg.RGBA{
		R: float64(e[0]) / 255.0,
		G: float64(e[1]) / 255.0,
		B: float64(e[2]) / 255.0,
		A: 1,
	}
}

// mergeExtremes merges new under/over/bad overrides on top of an existing
// extremes value. nil inputs leave the existing setting unchanged.
func mergeExtremes(base extremes, under, over, bad *gg.RGBA) extremes {
	out := base
	if under != nil {
		v := *under
		out.under = &v
	}
	if over != nil {
		v := *over
		out.over = &v
	}
	if bad != nil {
		v := *bad
		out.bad = &v
	}
	return out
}
