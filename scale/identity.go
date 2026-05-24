package scale

import (
	"github.com/TuSKan/ggplot/dataset"
)

// IdentityScale passes raw data values through directly as visual values.
// Useful when data columns already represent physical pixels, opacities, or colors.
type IdentityScale struct {
	domain
}

// NewIdentity returns a new IdentityScale.
func NewIdentity() *IdentityScale {
	return &IdentityScale{}
}

// Train trains the identity scale domain.
func (s *IdentityScale) Train(col dataset.AnyColumn) error {
	return s.train(col)
}

// Map maps a value to itself (no normalization transformation).
func (s *IdentityScale) Map(v float64) float64 {
	return v
}

// MapValue returns the value itself.
func (s *IdentityScale) MapValue(v float64) float64 {
	return v
}

// Inverse returns the value itself.
func (s *IdentityScale) Inverse(v float64) float64 {
	return v
}

// Ticks returns nicely spaced ticks for the scale domain.
func (s *IdentityScale) Ticks(n int) []float64 {
	return NiceSequence(s.min, s.max, n)
}

// Format formats the value.
func (s *IdentityScale) Format(v float64) string {
	return FormatNumber(v)
}

// Bounds returns the scale bounds.
func (s *IdentityScale) Bounds() (float64, float64) {
	return s.min, s.max
}

// String returns "identity".
func (s *IdentityScale) String() string {
	return "identity"
}

// SetBounds manually overrides the scale domain.
func (s *IdentityScale) SetBounds(mn, mx float64) {
	s.min = mn
	s.max = mx
	s.trained = true
}

// Verify interface compliance
var _ Scale = (*IdentityScale)(nil)
var _ BoundsSetter = (*IdentityScale)(nil)
var _ ValueMapper = (*IdentityScale)(nil)
