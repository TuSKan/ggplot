package arrow

import "errors"

// Sentinel errors for the arrow engine package.
var (
	// ErrUnsupportedType is returned for unsupported column types.
	ErrUnsupportedType = errors.New("arrow: unsupported column type")

	// ErrLengthMismatch is returned when column lengths don't match.
	ErrLengthMismatch = errors.New("arrow: column length mismatch")

	// ErrEmptyColumn is returned when an operation requires non-empty data.
	ErrEmptyColumn = errors.New("arrow: empty column")

	// ErrRequiresFloat64 is returned when a float64 column is required.
	ErrRequiresFloat64 = errors.New("arrow: operation requires float64 column")

	// ErrRequiresInt64 is returned when an int64 column is required.
	ErrRequiresInt64 = errors.New("arrow: operation requires int64 column")

	// ErrRequiresNumeric is returned when a numeric column is required.
	ErrRequiresNumeric = errors.New("arrow: operation requires numeric column")

	// ErrJoinKeyMismatch is returned when join key types don't match.
	ErrJoinKeyMismatch = errors.New("arrow: join key type mismatch")

	// ErrTakeTypeMismatch is returned when a Take/Slice result has unexpected type.
	ErrTakeTypeMismatch = errors.New("arrow: unexpected result type from Take/Slice")

	// ErrComputeTypeMismatch is returned when a compute kernel result has unexpected type.
	ErrComputeTypeMismatch = errors.New("arrow: unexpected result type from compute kernel")
)
