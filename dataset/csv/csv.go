// Package csv provides CSV reading and writing for the dataset package.
//
// This is a pure API facade — it contains zero heavy imports. The actual
// CSV parsing/writing is delegated to the engine via the [dataset.CSVReader]
// and [dataset.CSVWriter] interfaces:
//
//   - Memory engine: uses go-simdcsv + schema inference (dataset/memory/csv.go)
//   - Arrow engine: uses arrow/csv for zero-copy ingest (dataset/arrow/csv.go)
//
// Usage:
//
//	ds, err := csv.Read(file, eng, csv.WithHeader(true))
//	err = csv.Write(file, ds, eng)
package csv

import (
	"fmt"
	"io"

	"github.com/TuSKan/ggplot/dataset"
)

// Option is a functional option for CSV read/write.
type Option func(*dataset.CSVConfig)

// WithHeader enables CSV header parsing (first row = column names).
func WithHeader(v bool) Option {
	return func(c *dataset.CSVConfig) { c.HasHeader = v }
}

// WithComma sets the field delimiter. Default is ','.
func WithComma(r rune) Option {
	return func(c *dataset.CSVConfig) { c.Comma = r }
}

// WithComment sets the comment character.
func WithComment(r rune) Option {
	return func(c *dataset.CSVConfig) { c.Comment = r }
}

// WithNullValues sets strings to treat as null.
func WithNullValues(vals ...string) Option {
	return func(c *dataset.CSVConfig) { c.NullValues = vals }
}

// WithChunk sets the number of rows per batch during reading.
// 0 means engine default (arrow: 65536, memory: unlimited).
func WithChunk(n int) Option {
	return func(c *dataset.CSVConfig) { c.ChunkSize = n }
}

func defaults() dataset.CSVConfig {
	return dataset.CSVConfig{
		HasHeader:  true,
		Comma:      ',',
		NullValues: []string{"", "NA", "NULL", "null", "NaN"},
	}
}

func buildConfig(opts []Option) dataset.CSVConfig {
	cfg := defaults()
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

// Read reads a CSV from r using the given engine.
// The engine must implement [dataset.CSVReader].
func Read(r io.Reader, eng dataset.Engine, opts ...Option) (dataset.Dataset, error) {
	reader, ok := eng.(dataset.CSVReader)
	if !ok {
		return nil, fmt.Errorf("csv: engine %q does not implement CSVReader", eng.Name())
	}
	return reader.ReadCSV(r, buildConfig(opts))
}

// Write writes a Dataset as CSV to w using the given engine.
// The engine must implement [dataset.CSVWriter].
func Write(w io.Writer, ds dataset.Dataset, eng dataset.Engine, opts ...Option) error {
	writer, ok := eng.(dataset.CSVWriter)
	if !ok {
		return fmt.Errorf("csv: engine %q does not implement CSVWriter", eng.Name())
	}
	return writer.WriteCSV(w, ds, buildConfig(opts))
}
