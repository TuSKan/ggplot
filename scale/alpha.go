package scale

// AlphaScale maps continuous data values to an opacity range [0.1, 1.0] by default.
type AlphaScale struct {
	LinearScale

	rangeMin float64
	rangeMax float64
}

// NewAlpha returns a new AlphaScale mapping to the specified opacity range.
func NewAlpha(rangeMin, rangeMax float64) *AlphaScale {
	return &AlphaScale{
		rangeMin: rangeMin,
		rangeMax: rangeMax,
	}
}

// NewAlphaDefault returns a new AlphaScale using the default range [0.1, 1.0].
func NewAlphaDefault() *AlphaScale {
	return &AlphaScale{
		rangeMin: 0.1,
		rangeMax: 1.0,
	}
}

// MapValue transforms a data value into the actual visual opacity,
// clamped to the configured range.
func (s *AlphaScale) MapValue(v float64) float64 {
	t := max(0, min(1, s.Map(v))) //nolint:mnd // clamp normalized value to [0, 1]

	return s.rangeMin + t*(s.rangeMax-s.rangeMin)
}

// Range returns the configured opacity range.
func (s *AlphaScale) Range() (float64, float64) {
	return s.rangeMin, s.rangeMax
}

// String returns "alpha".
func (s *AlphaScale) String() string {
	return "alpha"
}

// Verify interface compliance
var _ Scale = (*AlphaScale)(nil)
var _ BoundsSetter = (*AlphaScale)(nil)
var _ ValueMapper = (*AlphaScale)(nil)
