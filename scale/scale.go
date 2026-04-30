// Package scale provides scale transformations that map data values to
// visual aesthetic values. Scales control axis ranges, color mappings,
// size mappings, and produce the guides (axes and legends) that make
// plots interpretable.
//
// Continuous scales map a numeric domain [min, max] to a visual range.
// Discrete scales map categorical values to a set of visual values.
package scale

import (
	"fmt"
	"math"

	"github.com/TuSKan/ggplot/dataset"
)

// Type identifies a scale transformation.
type Type string

const (
	Linear  Type = "linear"
	Log10   Type = "log10"
	Sqrt    Type = "sqrt"
	Reverse Type = "reverse"
)

// Scale defines the interface for all scale types.
type Scale interface {
	// Train updates the scale's domain from the given column data.
	Train(col dataset.AnyColumn) error

	// Map transforms a raw value to the [0,1] normalized scale.
	Map(v float64) float64

	// Inverse maps a [0,1] normalized value back to data space.
	Inverse(v float64) float64

	// Ticks generates n nicely spaced tick positions in data space.
	Ticks(n int) []float64

	// Format converts a data value to a display string.
	Format(v float64) string

	// Bounds returns the trained domain [min, max].
	Bounds() (float64, float64)

	// String returns a description.
	String() string
}

// BoundsSetter is an optional interface for scales that support manual
// bounds override. Both [LinearScale] and [DiscreteScale] implement this.
type BoundsSetter interface {
	SetBounds(mn, mx float64)
}

// MinorTicker is an optional interface for scales that provide minor tick
// positions between major ticks. Used by the guide system to draw minor
// grid lines. Implement this by using [WithMinorBreaks] on a [ConfiguredScale].
type MinorTicker interface {
	MinorTicks() []float64
}

// Expander is an optional interface for scales that carry user-specified
// expand parameters. The rendering pipeline queries this to decide whether
// to apply its default 5 % padding or the user's explicit expansion.
type Expander interface {
	Expand() (mult, add float64)
	HasExpand() bool
}

// --- Shared training state ---

// domain holds the trained min/max and provides reusable Train logic.
type domain struct {
	min, max float64
	trained  bool
}

// train updates the domain from a column's numeric values.
func (d *domain) train(col dataset.AnyColumn) error {
	switch c := col.(type) {
	case dataset.Column[float64]:
		vals := c.Values()
		if len(vals) == 0 {
			return nil
		}
		mn, mx := math.Inf(1), math.Inf(-1)
		for _, v := range vals {
			if math.IsNaN(v) {
				continue
			}
			if v < mn {
				mn = v
			}
			if v > mx {
				mx = v
			}
		}
		if math.IsInf(mn, 1) {
			return nil // all NaN
		}
		return d.update(mn, mx)

	case dataset.Column[int64]:
		vals := c.Values()
		if len(vals) == 0 {
			return nil
		}
		mn, mx := float64(vals[0]), float64(vals[0])
		for _, v := range vals[1:] {
			fv := float64(v)
			if fv < mn {
				mn = fv
			}
			if fv > mx {
				mx = fv
			}
		}
		return d.update(mn, mx)

	default:
		return fmt.Errorf("scale: column %q (%s) is not numeric", col.Name(), col.DType())
	}
}

func (d *domain) update(mn, mx float64) error {
	if !d.trained {
		d.min, d.max = mn, mx
		d.trained = true
	} else {
		if mn < d.min {
			d.min = mn
		}
		if mx > d.max {
			d.max = mx
		}
	}
	return nil
}

// --- Continuous scales ---

// NewLinear returns a standard linear scale.
func NewLinear() Scale {
	return &LinearScale{}
}

// LinearScale is a standard linear continuous scale.
type LinearScale struct {
	domain
}

func (s *LinearScale) Train(col dataset.AnyColumn) error {
	return s.domain.train(col)
}

func (s *LinearScale) Map(v float64) float64 {
	if s.max == s.min {
		return 0.5
	}
	return (v - s.min) / (s.max - s.min)
}

func (s *LinearScale) Inverse(v float64) float64 {
	return s.min + v*(s.max-s.min)
}

func (s *LinearScale) Ticks(n int) []float64 {
	return NiceSequence(s.min, s.max, n)
}

func (s *LinearScale) Format(v float64) string    { return FormatNumber(v) }
func (s *LinearScale) Bounds() (float64, float64) { return s.min, s.max }
func (s *LinearScale) String() string             { return "linear" }

// SetBounds manually overrides the scale domain. Used by the rendering
// pipeline for padding and forcing Y=0.
func (s *LinearScale) SetBounds(mn, mx float64) {
	s.min = mn
	s.max = mx
	s.trained = true
}

// NewLog10 returns a base-10 logarithmic scale.
func NewLog10() Scale {
	return &logScale{base: 10}
}

type logScale struct {
	base float64
	domain
}

func (s *logScale) Train(col dataset.AnyColumn) error {
	if err := s.domain.train(col); err != nil {
		return err
	}
	// Clamp min to avoid log(0).
	if s.min <= 0 {
		s.min = 1e-6
	}
	return nil
}

func (s *logScale) Map(v float64) float64 {
	if v <= 0 {
		v = 1e-6
	}
	logMin := math.Log10(s.min)
	logMax := math.Log10(s.max)
	if logMax == logMin {
		return 0.5
	}
	return (math.Log10(v) - logMin) / (logMax - logMin)
}

func (s *logScale) Inverse(v float64) float64 {
	logMin := math.Log10(s.min)
	logMax := math.Log10(s.max)
	return math.Pow(10, logMin+v*(logMax-logMin))
}

func (s *logScale) Ticks(n int) []float64 {
	if s.min <= 0 || s.max <= 0 {
		return nil
	}
	logMin := math.Floor(math.Log10(s.min))
	logMax := math.Ceil(math.Log10(s.max))
	var ticks []float64
	for e := logMin; e <= logMax; e++ {
		ticks = append(ticks, math.Pow(10, e))
	}
	return ticks
}

func (s *logScale) Format(v float64) string    { return FormatNumber(v) }
func (s *logScale) Bounds() (float64, float64) { return s.min, s.max }
func (s *logScale) String() string             { return "log10" }
func (s *logScale) SetBounds(mn, mx float64) {
	if mn <= 0 {
		mn = 1e-6
	}
	s.min = mn
	s.max = mx
	s.trained = true
}

// NewSqrt returns a square-root scale.
func NewSqrt() Scale {
	return &sqrtScale{}
}

type sqrtScale struct {
	domain
}

func (s *sqrtScale) Train(col dataset.AnyColumn) error {
	if err := s.domain.train(col); err != nil {
		return err
	}
	if s.min < 0 {
		s.min = 0
	}
	return nil
}

func (s *sqrtScale) Map(v float64) float64 {
	if v < 0 {
		v = 0
	}
	sqrtMin := math.Sqrt(s.min)
	sqrtMax := math.Sqrt(s.max)
	if sqrtMax == sqrtMin {
		return 0.5
	}
	return (math.Sqrt(v) - sqrtMin) / (sqrtMax - sqrtMin)
}

func (s *sqrtScale) Inverse(v float64) float64 {
	sqrtMin := math.Sqrt(s.min)
	sqrtMax := math.Sqrt(s.max)
	val := sqrtMin + v*(sqrtMax-sqrtMin)
	return val * val
}

func (s *sqrtScale) Ticks(n int) []float64 {
	// Generate nice tick values in sqrt-space, then square back to data-space.
	sqrtTicks := NiceSequence(math.Sqrt(s.min), math.Sqrt(s.max), n)
	ticks := make([]float64, len(sqrtTicks))
	for i, st := range sqrtTicks {
		ticks[i] = st * st
	}
	return ticks
}

func (s *sqrtScale) Format(v float64) string    { return FormatNumber(v) }
func (s *sqrtScale) Bounds() (float64, float64) { return s.min, s.max }
func (s *sqrtScale) String() string             { return "sqrt" }
func (s *sqrtScale) SetBounds(mn, mx float64) {
	if mn < 0 {
		mn = 0
	}
	s.min = mn
	s.max = mx
	s.trained = true
}

// NewReverse returns a scale that inverts the axis direction.
func NewReverse() Scale {
	return &reverseScale{}
}

type reverseScale struct {
	domain
}

func (s *reverseScale) Train(col dataset.AnyColumn) error { return s.domain.train(col) }
func (s *reverseScale) Map(v float64) float64 {
	if s.max == s.min {
		return 0.5
	}
	return 1.0 - (v-s.min)/(s.max-s.min)
}
func (s *reverseScale) Inverse(v float64) float64 {
	return s.max - v*(s.max-s.min)
}
func (s *reverseScale) Ticks(n int) []float64 {
	ticks := NiceSequence(s.min, s.max, n)
	// Reverse the order.
	for i, j := 0, len(ticks)-1; i < j; i, j = i+1, j-1 {
		ticks[i], ticks[j] = ticks[j], ticks[i]
	}
	return ticks
}
func (s *reverseScale) Format(v float64) string    { return FormatNumber(v) }
func (s *reverseScale) Bounds() (float64, float64) { return s.min, s.max }
func (s *reverseScale) String() string             { return "reverse" }
func (s *reverseScale) SetBounds(mn, mx float64) {
	s.min = mn
	s.max = mx
	s.trained = true
}

// Resolve returns a Scale for the given type.
// Returns an error for unknown types.
func Resolve(t Type) (Scale, error) {
	switch t {
	case Linear, "":
		return NewLinear(), nil
	case Log10:
		return NewLog10(), nil
	case Sqrt:
		return NewSqrt(), nil
	case Reverse:
		return NewReverse(), nil
	default:
		return nil, fmt.Errorf("scale: unknown type %q", t)
	}
}

// --- Manager ---

// Manager holds trained scales for all aesthetic channels.
type Manager struct {
	scales map[string]Scale
}

// NewManager creates a Manager with default scales.
func NewManager() *Manager {
	return &Manager{scales: make(map[string]Scale)}
}

// Set registers a scale for an aesthetic channel.
func (m *Manager) Set(channel string, s Scale) {
	m.scales[channel] = s
}

// Get retrieves the scale for a channel. Returns a linear scale as default.
func (m *Manager) Get(channel string) Scale {
	if s, ok := m.scales[channel]; ok {
		return s
	}
	s := NewLinear()
	m.scales[channel] = s
	return s
}

// --- Tick helpers ---

// NiceSequence produces approximately n optimally placed tick positions
// between lo and hi using the Talbot-Lin-Hanrahan (2010) extended
// Wilkinson algorithm. See ticks.go for the full implementation.
func NiceSequence(lo, hi float64, n int) []float64 {
	if n <= 0 {
		n = 5
	}
	if lo == hi {
		return []float64{lo}
	}
	return extendedWilkinson(lo, hi, n)
}

// niceNum finds a "nice" number approximately equal to x.
// If round is true, rounds to nearest; otherwise, ceiling.
//
// Reference: Heckbert, P. (1990) "Nice Numbers for Graph Labels",
// Graphics Gems, pp. 61-63.
//
// Used as a fallback by the extended Wilkinson algorithm (see ticks.go).
func niceNum(x float64, round bool) float64 {
	if x <= 0 {
		return 1
	}
	exp := math.Floor(math.Log10(x))
	frac := x / math.Pow(10, exp)

	var nice float64
	if round {
		switch {
		case frac < 1.5:
			nice = 1
		case frac < 3:
			nice = 2
		case frac < 7:
			nice = 5
		default:
			nice = 10
		}
	} else {
		switch {
		case frac <= 1:
			nice = 1
		case frac <= 2:
			nice = 2
		case frac <= 5:
			nice = 5
		default:
			nice = 10
		}
	}
	return nice * math.Pow(10, exp)
}

func roundTo(v float64, digits int) float64 {
	factor := math.Pow(10, float64(digits))
	return math.Round(v*factor) / factor
}

// FormatNumber formats a tick value for display.
// Integers are shown without decimals, floats use compact notation.
func FormatNumber(v float64) string {
	if v == math.Floor(v) && math.Abs(v) < 1e12 {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%.4g", v)
}

// --- Compile-time interface checks ---

var (
	_ Scale        = (*LinearScale)(nil)
	_ Scale        = (*logScale)(nil)
	_ Scale        = (*sqrtScale)(nil)
	_ Scale        = (*reverseScale)(nil)
	_ BoundsSetter = (*LinearScale)(nil)
	_ BoundsSetter = (*logScale)(nil)
	_ BoundsSetter = (*sqrtScale)(nil)
	_ BoundsSetter = (*reverseScale)(nil)
)
