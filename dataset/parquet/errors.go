package parquet

import "errors"

// ErrUnsupportedType is returned for unsupported column types.
var ErrUnsupportedType = errors.New("parquet: unsupported column type")
