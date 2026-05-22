package scale

import "math"

// OOBPolicy defines behavior for data points that fall outside the
// trained scale domain after normalization.
type OOBPolicy int

const (
	// OOBKeep passes out-of-bounds values through unchanged.
	// This is the default behavior — points outside the domain are drawn
	// at their extrapolated positions.
	OOBKeep OOBPolicy = iota

	// OOBCensor replaces out-of-bounds normalized values with NaN,
	// effectively hiding those data points. This matches ggplot2's
	// default oob behavior for continuous scales.
	OOBCensor

	// OOBSquish clamps out-of-bounds normalized values to the [0, 1]
	// range, pinning them to the domain boundary.
	OOBSquish
)

// applyOOB applies the OOB policy to a normalized value in [0, 1] space.
// Returns the adjusted value (possibly NaN for censor).
func applyOOB(v float64, policy OOBPolicy) float64 {
	if v >= 0 && v <= 1 {
		return v // in-bounds — no action needed
	}

	switch policy { //nolint:exhaustive // OOBKeep is handled by default.
	case OOBCensor:
		return math.NaN()
	case OOBSquish:
		return math.Max(0, math.Min(1, v))
	default: // OOBKeep
		return v
	}
}
