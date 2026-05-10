// Package dataset provides columnar data abstractions for the Grammar of
// Graphics pipeline. Frame verbs execute eagerly via the dataset's engine
// (memory and arrow backends materialize on each verb); the BigQuery engine
// is the only backend with internal lazy SQL accumulation. Arrow IPC and
// Parquet ingest paths support zero-copy reads.
//
// # Engine-First Architecture
//
// Every data operation is delegated to an [Engine] backend. The dataset package
// defines only interfaces and contracts — no concrete column types, no fallbacks.
// Engines (Arrow, memory, SQL) implement sub-interfaces ([Aggregator], [Windower],
// [Joiner], etc.) for the operations they support.
//
// # Type System
//
// The type system is aligned with Apache Arrow:
//   - [Field] maps to arrow.Field (name, type, nullable, metadata)
//   - [Schema] maps to arrow.Schema (ordered collection of fields)
//   - [AnyColumn] is the type-erased column interface (engine-native storage)
//   - [Column] is the generic typed access layer
//   - [GetColumn] bridges untyped to typed via a single type assertion
package dataset

import "fmt"

// Table represents an immutable, columnar data source.
//
// Implementations include in-memory tables, Arrow tables, and BigQuery-backed
// remote tables. ETL verbs are exposed by wrapping a Table in a [Dataset]
// (the fluent API defined in frame.go) via [From].
type Table interface {
	// Schema returns the dataset's schema.
	Schema() *Schema

	// Column retrieves a named column. Returns [ErrColumnNotFound] if absent.
	// The returned [AnyColumn] can be type-asserted to [Column[T]] for typed
	// access, or use [GetColumn] for a safe generic retrieval.
	Column(name string) (AnyColumn, error)

	// NumRows returns the logical number of rows.
	NumRows() int64

	// NumCols returns the number of columns.
	NumCols() int64
}

// Names returns the column names from a dataset's schema.
func Names(ds Table) []string {
	fields := ds.Schema().Fields()

	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}

	return names
}

// Closer is optionally implemented by datasets that hold resources
// requiring explicit cleanup (e.g., Arrow tables, database connections).
type Closer interface {
	Close() error
}

// Close releases resources if the dataset implements [Closer]. Safe to call
// on any Dataset — returns nil for datasets without resources.
func Close(ds Table) error {
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
