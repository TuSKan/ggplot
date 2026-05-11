package dataset

// DType represents the logical data type of a column.
// This is the type ID — analogous to arrow.Type.
type DType int

const (
	// DTypeFloat64 is a 64-bit floating point column.
	DTypeFloat64 DType = iota
	// DTypeInt64 is a 64-bit integer column.
	DTypeInt64
	// DTypeString is a string/categorical column.
	DTypeString
	// DTypeBool is a boolean column.
	DTypeBool
	// DTypeTimestamp is a timestamp column stored as int64 nanoseconds
	// since the Unix epoch (1970-01-01T00:00:00Z). This representation
	// is zero-copy compatible with Arrow's TIMESTAMP(ns) type.
	DTypeTimestamp
	// DTypeUnknown is an unrecognized type.
	DTypeUnknown
)

// String returns the human-readable name of the DType.
func (d DType) String() string {
	switch d { //nolint:exhaustive // intentional subset; default case handles the rest.
	case DTypeFloat64:
		return "float64"
	case DTypeInt64:
		return "int64"
	case DTypeString:
		return "string"
	case DTypeBool:
		return "bool"
	case DTypeTimestamp:
		return "timestamp[ns]"
	default:
		return "unknown"
	}
}
