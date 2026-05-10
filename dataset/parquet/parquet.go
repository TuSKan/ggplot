// Package parquet provides Parquet reading and writing for the dataset package.
//
// This is a pure API facade — it contains zero heavy imports. The actual
// Parquet processing is delegated to the engine via the [dataset.ParquetReader]
// and [dataset.ParquetWriter] interfaces:
//
//   - Memory engine: uses parquet-go for struct-based row I/O (dataset/memory/parquet.go)
//   - Arrow engine: uses pqarrow for zero-copy columnar I/O (dataset/arrow/parquet.go)
//
// Usage:
//
//	ds, err := parquet.Read(ctx, file, size, eng)
//	err = parquet.Write(ctx, file, ds, eng)
package parquet

import (
	"context"
	"fmt"
	"io"

	"github.com/TuSKan/ggplot/dataset"
)

// Option is a functional option for Parquet read/write.
type Option func(*dataset.ParquetConfig)

// WithCompression sets the compression codec ("snappy", "gzip", "zstd", "lz4", "none").
// Default is "snappy".
func WithCompression(codec string) Option {
	return func(c *dataset.ParquetConfig) { c.Compression = codec }
}

func defaults() dataset.ParquetConfig {
	return dataset.ParquetConfig{
		Compression: "snappy",
	}
}

func buildConfig(opts []Option) dataset.ParquetConfig {
	cfg := defaults()
	for _, o := range opts {
		o(&cfg)
	}

	return cfg
}

// Read reads a Parquet file from r (which must support random access) using
// the given engine. The engine must implement [dataset.ParquetReader].
func Read(ctx context.Context, r io.ReaderAt, size int64, eng dataset.Engine, opts ...Option) (dataset.Dataset, error) {
	reader, ok := eng.(dataset.ParquetReader)
	if !ok {
		return dataset.Dataset{}, fmt.Errorf("parquet: engine %q does not implement ParquetReader", eng.Name())
	}

	tbl, err := reader.ReadParquet(ctx, r, size, buildConfig(opts))
	if err != nil {
		return dataset.Dataset{}, err
	}

	return dataset.From(tbl), nil
}

// Write writes a Dataset as Parquet to w using the given engine.
// The engine must implement [dataset.ParquetWriter].
func Write(ctx context.Context, w io.Writer, ds dataset.Dataset, eng dataset.Engine, opts ...Option) error {
	writer, ok := eng.(dataset.ParquetWriter)
	if !ok {
		return fmt.Errorf("parquet: engine %q does not implement ParquetWriter", eng.Name())
	}

	return writer.WriteParquet(ctx, w, ds.Table(), buildConfig(opts))
}
