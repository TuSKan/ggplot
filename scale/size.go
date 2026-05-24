package scale

import (
	"math"

	"github.com/TuSKan/ggplot/dataset"
)

// SizeMode controls how values map to sizes.
type SizeMode string

const (
	// SizeModeLinear maps values linearly to radius.
	SizeModeLinear SizeMode = "linear"
	// SizeModeArea maps values to area (radius is proportional to sqrt(value)).
	SizeModeArea SizeMode = "area"
	// SizeModeRadius maps values to radius direct proportionally (value of 0 maps to 0).
	SizeModeRadius SizeMode = "radius"
)

// SizeScale maps continuous data values to a point-radius range.
type SizeScale struct {
	domain

	mode     SizeMode
	rangeMin float64
	rangeMax float64
}

// NewSize returns a new SizeScale with the specified radius range.
func NewSize(rangeMin, rangeMax float64) *SizeScale {
	return &SizeScale{
		mode:     SizeModeLinear,
		rangeMin: rangeMin,
		rangeMax: rangeMax,
	}
}

// NewSizeDefault returns a new linear SizeScale using the default range [1.0, 6.0].
func NewSizeDefault() *SizeScale {
	return &SizeScale{
		mode:     SizeModeLinear,
		rangeMin: 1.0,
		rangeMax: 6.0,
	}
}

// NewSizeArea returns a new SizeScale configured for area-proportional mapping.
func NewSizeArea() *SizeScale {
	return &SizeScale{
		mode:     SizeModeArea,
		rangeMin: 1.0,
		rangeMax: 6.0,
	}
}

// Train trains the scale's domain on a data column.
func (s *SizeScale) Train(col dataset.AnyColumn) error {
	return s.train(col)
}

// Map maps a value v to a normalized fraction in [0, 1].
func (s *SizeScale) Map(v float64) float64 {
	if s.max == s.min {
		return 0.5
	}

	switch s.mode {
	case SizeModeArea:
		mn := s.min
		if mn < 0 {
			mn = 0
		}

		if v < 0 {
			v = 0
		}

		sqrtMin := math.Sqrt(mn)
		sqrtMax := math.Sqrt(s.max)

		if sqrtMax == sqrtMin {
			return 0.5
		}

		return (math.Sqrt(v) - sqrtMin) / (sqrtMax - sqrtMin)

	case SizeModeRadius:
		if s.max <= 0 {
			return 0
		}

		if v < 0 {
			v = 0
		}

		return v / s.max

	case SizeModeLinear:
		fallthrough
	default:
		return (v - s.min) / (s.max - s.min)
	}
}

// Inverse maps a normalized fraction back to data space.
func (s *SizeScale) Inverse(v float64) float64 {
	switch s.mode {
	case SizeModeArea:
		mn := s.min
		if mn < 0 {
			mn = 0
		}

		sqrtMin := math.Sqrt(mn)
		sqrtMax := math.Sqrt(s.max)
		st := sqrtMin + v*(sqrtMax-sqrtMin)

		return st * st

	case SizeModeRadius:
		return v * s.max

	case SizeModeLinear:
		fallthrough
	default:
		return s.min + v*(s.max-s.min)
	}
}

// MapValue maps a data value to its actual pixel radius,
// clamped to the configured range.
func (s *SizeScale) MapValue(v float64) float64 {
	t := max(0, min(1, s.Map(v))) //nolint:mnd // clamp normalized value to [0, 1]

	return s.rangeMin + t*(s.rangeMax-s.rangeMin)
}

// Ticks generates nice tick values.
func (s *SizeScale) Ticks(n int) []float64 {
	if s.mode == SizeModeArea {
		mn := s.min
		if mn < 0 {
			mn = 0
		}

		sqrtTicks := NiceSequence(math.Sqrt(mn), math.Sqrt(s.max), n)
		ticks := make([]float64, len(sqrtTicks))

		for i, st := range sqrtTicks {
			ticks[i] = st * st
		}

		return ticks
	}

	return NiceSequence(s.min, s.max, n)
}

// Format formats values as strings.
func (s *SizeScale) Format(v float64) string {
	return FormatNumber(v)
}

// Bounds returns the scale bounds.
func (s *SizeScale) Bounds() (float64, float64) {
	return s.min, s.max
}

// Range returns the scale range.
func (s *SizeScale) Range() (float64, float64) {
	return s.rangeMin, s.rangeMax
}

// SetBounds manually overrides scale bounds.
func (s *SizeScale) SetBounds(mn, mx float64) {
	s.min = mn
	s.max = mx
	s.trained = true
}

// String returns "size".
func (s *SizeScale) String() string {
	return "size"
}

// Mode returns the sizing mode (linear, area, or radius).
func (s *SizeScale) Mode() SizeMode {
	return s.mode
}

// Verify interface compliance
var _ Scale = (*SizeScale)(nil)
var _ BoundsSetter = (*SizeScale)(nil)
var _ ValueMapper = (*SizeScale)(nil)
