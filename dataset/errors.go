package dataset

import "errors"

// Sentinel errors for the dataset package.
var (
	// ErrUncollected is returned when an operation requires a collected Dataset.
	ErrUncollected = errors.New("dataset: operation on uncollected Dataset — call Collect(ctx) first")

	// ErrUnsupportedEngine is returned when an engine lacks a required capability.
	ErrUnsupportedEngine = errors.New("dataset: unsupported engine capability")

	// ErrNoEngine is returned when a Dataset has no engine.
	ErrNoEngine = errors.New("dataset: Dataset requires an engine")

	// ErrInvalidSlice is returned when a slice range is invalid.
	ErrInvalidSlice = errors.New("dataset: invalid slice range")

	// ErrNoAggResults is returned when there are no aggregation results.
	ErrNoAggResults = errors.New("dataset: no aggregation results to merge")

	// ErrUnsupportedAggFunc is returned for unknown aggregation functions.
	ErrUnsupportedAggFunc = errors.New("dataset: unknown AggFunc")

	// ErrUnsupportedDType is returned for unsupported data types.
	ErrUnsupportedDType = errors.New("dataset: unsupported DType")

	// ErrTypeMismatch is returned when column types don't match.
	ErrTypeMismatch = errors.New("dataset: column type mismatch")

	// ErrColumnNotNumeric is returned when a numeric column is required.
	ErrColumnNotNumeric = errors.New("dataset: column is not numeric")

	// ErrUnsupportedPredicate is returned for unsupported filter predicates.
	ErrUnsupportedPredicate = errors.New("dataset: unsupported predicate operator")
)
