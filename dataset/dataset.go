// Package dataset provides zero-copy, lazy-evaluating columnar data abstractions
// for the Grammar of Graphics pipeline.
//
// The core [Dataset] interface represents an immutable columnar data source.
// All ETL operations (Select, Filter, Mutate, etc.) are available through [Frame],
// a dplyr-inspired fluent wrapper that chains lazy transformations.
//
// # Materialization Policy
//
// Lazy transformations (Filter, Mutate, Select, GroupBy) produce new Dataset
// wrappers that delay physical computation until data is accessed. This allows
// Arrow-backed datasets to leverage zero-copy slicing, and SQL-backed datasets
// to push predicates down to the server.
//
// Materialization is triggered only when:
//   - A rendering backend consumes column data for drawing.
//   - A statistical transform requires contiguous arrays (e.g., KDE, loess).
//   - The user explicitly calls [Frame.Collect].
package dataset

import "fmt"

// Dataset represents an immutable, columnar data source.
//
// Implementations include in-memory frames, Arrow tables, and SQL-backed
// remote tables. All ETL operations are available via [Frame].
type Dataset interface {
	// Columns returns the available column names.
	Columns() []string

	// Column retrieves a named column. Returns [ErrColumnNotFound] if absent.
	Column(name string) (Column, error)

	// Len returns the logical number of rows.
	Len() int
}

// Closer is optionally implemented by datasets that hold resources
// requiring explicit cleanup (e.g., Arrow tables, database connections).
type Closer interface {
	Close() error
}

// Close releases resources if the dataset implements [Closer]. Safe to call
// on any Dataset; non-closable datasets are silently ignored.
func Close(ds Dataset) error {
	if c, ok := ds.(Closer); ok {
		return c.Close()
	}
	return nil
}

// ErrColumnNotFound indicates a requested column does not exist.
type ErrColumnNotFound struct {
	Name string
}

func (e *ErrColumnNotFound) Error() string {
	return fmt.Sprintf("dataset: column %q not found", e.Name)
}
