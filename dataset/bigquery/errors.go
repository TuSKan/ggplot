package bigquery

import "errors"

// Sentinel errors for the BigQuery engine package.
var (
	// ErrUnsupportedType is returned for unsupported column types.
	ErrUnsupportedType = errors.New("bigquery: unsupported column type")

	// ErrUnsupportedOp is returned for unsupported operations.
	ErrUnsupportedOp = errors.New("bigquery: unsupported operation")

	// ErrSchemaMismatch is returned when schemas don't match.
	ErrSchemaMismatch = errors.New("bigquery: schema mismatch")

	// ErrNilDataset is returned when a nil dataset is provided.
	ErrNilDataset = errors.New("bigquery: nil dataset")

	// ErrEmptyDataset is returned when an empty dataset is provided.
	ErrEmptyDataset = errors.New("bigquery: empty dataset")
)
