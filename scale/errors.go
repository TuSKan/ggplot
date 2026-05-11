package scale

import "errors"

// Sentinel errors for the scale package.
var (
	// ErrUnsupportedScale is returned for unsupported scale types.
	ErrUnsupportedScale = errors.New("scale: unsupported scale type")

	// ErrInvalidBreak is returned for invalid scale breaks.
	ErrInvalidBreak = errors.New("scale: invalid break value")
)
