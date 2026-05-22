package colormap

import (
	"fmt"
	"maps"
	"math"
	"strconv"

	"github.com/gogpu/gg"

	"github.com/TuSKan/ggplot/dataset"
)

// Scale composes a [Norm] and a [Cmap] into a dataset-aware color mapper.
// It is the primary user-facing type produced by [NewContinuous],
// [NewDiscrete], and [NewManual] — and the value attached to a Plot via
// .ScaleColor / .ScaleFill / .ScaleColorManual.
//
// Scale supports three usage modes:
//
//   - Continuous: a Norm transforms numeric values into [0,1]; the Cmap
//     samples a color at that t.
//   - Discrete:   string category labels are encoded into the index space
//     of a [ListedCmap]. Cycling reuses colors when categories outnumber
//     palette entries.
//   - Manual:     specific labels are pinned to specific gg.RGBA values via
//     [Scale.SetOverride]. Overrides take precedence over both modes above
//     in [Scale.At].
type Scale struct {
	cmap       Cmap
	norm       Norm
	discrete   bool
	cycle      bool
	categories []string
	labelIdx   map[string]int
	overrides  map[string]gg.RGBA
	naColor    *gg.RGBA // nil = use cmap.At(NaN) default
}

// NewContinuous returns a Scale that maps numeric data through n into the
// unit interval, then samples c at the resulting t. If n is nil, a default
// LinearNorm is created.
func NewContinuous(c Cmap, n Norm) *Scale {
	if c == nil {
		c = Viridis
	}

	if n == nil {
		n = &LinearNorm{}
	}

	return &Scale{cmap: c, norm: n}
}

// NewDiscrete returns a Scale that maps category labels through c. Labels
// are encoded by their first-seen position; positions modulo c.N() select a
// color from c. Use [Scale.SetCycle] to control whether the palette cycles
// (default: cycles).
func NewDiscrete(c Cmap) *Scale {
	if c == nil {
		c = Tab10
	}

	return &Scale{
		cmap:     c,
		discrete: true,
		cycle:    true,
		labelIdx: make(map[string]int),
	}
}

// NewManual returns a Scale where every category label has an explicit color.
// Labels not in m fall back to a default Tab10 palette in the order they are
// trained.
func NewManual(m map[string]gg.RGBA) *Scale {
	overrides := make(map[string]gg.RGBA, len(m))
	maps.Copy(overrides, m)

	return &Scale{
		cmap:      Tab10,
		discrete:  true,
		cycle:     true,
		labelIdx:  make(map[string]int),
		overrides: overrides,
	}
}

// SetCmap replaces the underlying colormap.
func (s *Scale) SetCmap(c Cmap) {
	if c != nil {
		s.cmap = c
	}
}

// SetNorm replaces the underlying Norm. Only meaningful for continuous scales.
func (s *Scale) SetNorm(n Norm) {
	if n != nil {
		s.norm = n
	}
}

// SetCycle controls whether discrete category index values wrap modulo the
// palette size. When false, indices beyond the palette return the last color.
func (s *Scale) SetCycle(cycle bool) { s.cycle = cycle }

// SetOverride pins label to a specific color, overriding any computed value.
// Overrides apply in both continuous and discrete modes.
func (s *Scale) SetOverride(label string, c gg.RGBA) {
	if s.overrides == nil {
		s.overrides = make(map[string]gg.RGBA)
	}

	s.overrides[label] = c
}

// Cmap returns the underlying colormap.
func (s *Scale) Cmap() Cmap { return s.cmap }

// Norm returns the underlying Norm. Returns nil for purely discrete scales
// that were never given one.
func (s *Scale) Norm() Norm { return s.norm }

// Discrete reports whether this Scale operates on category labels.
func (s *Scale) Discrete() bool { return s.discrete }

// SetNAColor sets the color to use for missing/null/NaN values.
// Pass nil to revert to the cmap's default NaN color.
func (s *Scale) SetNAColor(c *gg.RGBA) { s.naColor = c }

// Categories returns the list of trained category labels in encounter order.
// Empty for continuous scales.
func (s *Scale) Categories() []string {
	out := make([]string, len(s.categories))
	copy(out, s.categories)

	return out
}

// Train adapts the Scale to the given column. For continuous Scales this
// expands the Norm bounds; for discrete Scales it appends new category
// labels to the encoding table. Calling Train multiple times accumulates.
func (s *Scale) Train(col dataset.AnyColumn) error {
	if s.discrete {
		return s.trainDiscrete(col)
	}

	if s.norm == nil {
		s.norm = &LinearNorm{}
	}

	if err := s.norm.Train(col); err != nil {
		return fmt.Errorf("colormap: %w", err)
	}

	return nil
}

func (s *Scale) trainDiscrete(col dataset.AnyColumn) error {
	if s.labelIdx == nil {
		s.labelIdx = make(map[string]int)
	}

	switch c := col.(type) {
	case dataset.Column[string]:
		for _, v := range c.Values() {
			s.observeLabel(v)
		}
	case dataset.Column[float64]:
		for _, v := range c.Values() {
			if math.IsNaN(v) {
				continue
			}

			s.observeLabel(strconv.FormatFloat(v, 'g', -1, 64))
		}
	case dataset.Column[int64]:
		for _, v := range c.Values() {
			s.observeLabel(strconv.FormatInt(v, 10))
		}
	case dataset.Column[bool]:
		for _, v := range c.Values() {
			if v {
				s.observeLabel("true")
			} else {
				s.observeLabel("false")
			}
		}
	default:
		return fmt.Errorf("colormap: discrete Scale cannot train on column %q (%s): %w", col.Name(), col.DType(), ErrParseColor)
	}

	return nil
}

func (s *Scale) observeLabel(label string) {
	if _, ok := s.labelIdx[label]; ok {
		return
	}

	s.labelIdx[label] = len(s.categories)
	s.categories = append(s.categories, label)
}

// At maps a value to a color. Accepts string, float64, int64, int, bool,
// or nil. The dispatch is:
//
//   - if v is a string and an override exists, return the override;
//   - else if Discrete, encode as a label (registering it on first sight)
//     and look up via the underlying ListedCmap or falling back to
//     cmap.At(idx/N);
//   - else (continuous), pass v through Norm and sample Cmap.At.
//
// NaN inputs sample the colormap's Bad slot.
func (s *Scale) At(v any) gg.RGBA {
	if v == nil {
		return s.bad()
	}

	if label, ok := asLabel(v); ok {
		if s.overrides != nil {
			if c, ok := s.overrides[label]; ok {
				return c
			}
		}

		if s.discrete {
			return s.atLabel(label)
		}
	}

	switch x := v.(type) {
	case float64:
		if math.IsNaN(x) {
			return s.bad()
		}

		return s.atNumeric(x)
	case float32:
		return s.atNumeric(float64(x))
	case int:
		return s.atNumeric(float64(x))
	case int32:
		return s.atNumeric(float64(x))
	case int64:
		return s.atNumeric(float64(x))
	case bool:
		if x {
			return s.atLabel("true")
		}

		return s.atLabel("false")
	case string:
		// Continuous Scale fed a string: try to parse it as a float.
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			return s.atNumeric(f)
		}

		return s.atLabel(x)
	default:
		return s.bad()
	}
}

// atNumeric routes a numeric value through Norm and Cmap.
func (s *Scale) atNumeric(v float64) gg.RGBA {
	if s.norm == nil {
		s.norm = &LinearNorm{}
	}

	t := s.norm.Norm(v)

	return s.cmap.At(t)
}

// atLabel encodes a category label and returns the corresponding color.
func (s *Scale) atLabel(label string) gg.RGBA {
	idx, ok := s.labelIdx[label]
	if !ok {
		// Lazy registration so callers that skipped Train still work.
		if s.labelIdx == nil {
			s.labelIdx = make(map[string]int)
		}

		idx = len(s.categories)
		s.labelIdx[label] = idx
		s.categories = append(s.categories, label)
	}

	return s.colorAtIndex(idx)
}

// colorAtIndex returns the cmap color for category index i.
func (s *Scale) colorAtIndex(i int) gg.RGBA {
	if listed, ok := s.cmap.(*ListedCmap); ok {
		if s.cycle {
			return listed.Color(i)
		}
		// non-cycling: clamp to last color
		n := listed.N()
		if n == 0 {
			return gg.RGBA{}
		}

		if i >= n {
			i = n - 1
		}

		return listed.Color(i)
	}
	// Continuous cmap used for discrete data: sample evenly.
	n := len(s.categories)
	if n <= 1 {
		return s.cmap.At(0)
	}

	t := float64(i) / float64(n-1)

	return s.cmap.At(t)
}

// bad returns the configured NA color, or the cmap's NaN/bad color
// (transparent black if neither is set).
func (s *Scale) bad() gg.RGBA {
	if s.naColor != nil {
		return *s.naColor
	}

	return s.cmap.At(math.NaN())
}

// asLabel returns a string label and true if v is naturally a string-like
// value (string, []byte, fmt.Stringer). Used by [Scale.At] to detect
// override candidates without touching numeric values.
func asLabel(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case []byte:
		return string(x), true
	case fmt.Stringer:
		return x.String(), true
	}

	return "", false
}
