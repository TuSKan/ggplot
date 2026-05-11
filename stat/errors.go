package stat

import "errors"

// Sentinel errors for the stat package.
var (
	// ErrInsufficientData is returned when there's not enough data.
	ErrInsufficientData = errors.New("stat: insufficient data")

	// ErrUnsupportedType is returned for unsupported column types.
	ErrUnsupportedType = errors.New("stat: unsupported column type")

	// ErrMissingColumn is returned when a required column is missing.
	ErrMissingColumn = errors.New("stat: missing required column")

	// ErrInvalidParam is returned for invalid parameter values.
	ErrInvalidParam = errors.New("stat: invalid parameter")
)
