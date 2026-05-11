package memory

import "errors"

// Sentinel errors for the memory engine package.
var (
	// ErrUnsupportedType is returned for unsupported column types.
	ErrUnsupportedType = errors.New("memory: unsupported column type")

	// ErrLengthMismatch is returned when column lengths don't match.
	ErrLengthMismatch = errors.New("memory: column length mismatch")

	// ErrEmptyColumn is returned when an operation requires non-empty data.
	ErrEmptyColumn = errors.New("memory: empty column")

	// ErrRequiresFloat64 is returned when a float64 column is required.
	ErrRequiresFloat64 = errors.New("memory: operation requires float64 column")

	// ErrRequiresInt64 is returned when an int64 column is required.
	ErrRequiresInt64 = errors.New("memory: operation requires int64 column")

	// ErrRequiresNumeric is returned when a numeric column is required.
	ErrRequiresNumeric = errors.New("memory: operation requires numeric column")

	// ErrJoinKeyMismatch is returned when join key types don't match.
	ErrJoinKeyMismatch = errors.New("memory: join key type mismatch")

	// ErrTakeTypeMismatch is returned when a Take/Select result has unexpected type.
	ErrTakeTypeMismatch = errors.New("memory: unexpected result type from Take/Select")
)
