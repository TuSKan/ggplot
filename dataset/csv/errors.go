package csv

import "errors"

// ErrUnsupportedType is returned for unsupported column types.
var ErrUnsupportedType = errors.New("csv: unsupported column type")
