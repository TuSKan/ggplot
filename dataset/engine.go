package dataset

import (
	"context"
	"io"
)

// Engine-First Architecture
//
// Every data operation is delegated to an Engine backend.
// A backend implements whichever sub-interfaces it supports.
// The Frame layer dispatches via type assertion — if an engine does not
// implement a sub-interface, the operation returns an error.
// There are NO fallbacks.
//
// Sub-interfaces:
//
//   - [ColumnFactory]   — wrap typed slices into engine-native columns
//   - [BuilderFactory]  — streaming typed construction (billion-row scale)
//   - [Aggregator]      — Sum, Mean, Min, Max, Count, Median, Variance
//   - [Caster]          — type casting (engine-controlled)
//   - [Windower]        — Lag, Lead, CumSum, CumMax, CumMin, Rank, etc.
//   - [Joiner]          — hash-join for Left, Right, Inner, Full, Semi, Anti
//   - [Reshaper]        — PivotLonger, PivotWider, Separate, Concatenate
//   - [Filterer]        — mask-based row filtering
//   - [Filler]          — Fill, DropNA, ReplaceNA
//   - [Composer]        — Stack, Combine
//   - [CSVReader]       — read CSV from io.Reader
//   - [CSVWriter]       — write CSV to io.Writer
//   - [ParquetReader]   — read Parquet from io.ReaderAt
//   - [ParquetWriter]   — write Parquet to io.Writer

// Engine is the marker interface that all compute backends implement.
// Every engine carries a context.Context that governs its lifecycle.
// Long-running operations should check Context().Err() for cancellation.
type Engine interface {
	// Name returns a human-readable identifier (e.g., "arrow", "memory", "sql").
	Name() string
	// Context returns the engine's lifecycle context.
	Context() context.Context
}

// Optimizer is optionally implemented by engines that can fuse or reorder
// operations for efficiency. BigQuery uses this to fuse verb chains into
// a single SQL query.
type Optimizer interface {
	Optimize(ops []op) []op
}

// HasEngine is implemented by datasets that carry an engine reference.
// This enables engine propagation through transformations — stat packages
// and ggplot internals can produce new datasets using the same engine
// without importing engine-specific packages.
type HasEngine interface {
	Table
	Engine() Engine
}

// GetEngine extracts the engine from a dataset.
// Returns nil if the dataset does not carry an engine.
func GetEngine(ds Table) Engine {
	if ed, ok := ds.(HasEngine); ok {
		return ed.Engine()
	}

	return nil
}

// --- Data Construction ---

// ColumnFactory wraps existing typed slices into engine-native columns.
// Memory engine: wraps the slice (zero-copy).
// Arrow engine: builds an Arrow array (one allocation).
type ColumnFactory interface {
	NewFloat64Column(name string, data []float64) AnyColumn
	NewInt64Column(name string, data []int64) AnyColumn
	NewStringColumn(name string, data []string) AnyColumn
	NewBoolColumn(name string, data []bool) AnyColumn
	NewTimestampColumn(name string, data []int64) AnyColumn

	// FromColumns assembles columns into a Dataset with the given schema.
	// All columns must have the same length.
	FromColumns(schema *Schema, cols ...AnyColumn) (Table, error)
}

// BuilderFactory creates schema-aware builders for streaming construction.
type BuilderFactory interface {
	NewBuilder(schema *Schema) Builder
}

// Builder provides streaming, typed, zero-boxing construction.
// Each column has its own typed appender — no any boxing, no
// allocations per row.
type Builder interface {
	Float64(col string) Float64Appender
	Int64(col string) Int64Appender
	String(col string) StringAppender
	Bool(col string) BoolAppender

	Build() (Table, error)
}

// Float64Appender streams float64 values into a column.
type Float64Appender interface {
	Append(v float64)
	AppendNull()
	AppendValues(vs []float64)
	Reserve(n int)
}

// Int64Appender streams int64 values into a column.
type Int64Appender interface {
	Append(v int64)
	AppendNull()
	AppendValues(vs []int64)
	Reserve(n int)
}

// StringAppender streams string values into a column.
type StringAppender interface {
	Append(v string)
	AppendNull()
	AppendValues(vs []string)
	Reserve(n int)
}

// BoolAppender streams bool values into a column.
type BoolAppender interface {
	Append(v bool)
	AppendNull()
	AppendValues(vs []bool)
	Reserve(n int)
}

// --- Computation ---

// Aggregator provides vectorized aggregation kernels.
// All methods return AnyColumn (single-element column) preserving the
// input type — aligned with Arrow compute kernel type rules:
//
//   - Sum:      numeric → same type (int64→int64, float64→float64)
//   - Mean:     numeric → float64 (always widens)
//   - MinMax:   any ordered type → (min, max) of same type
//   - Count:    any → int64
//   - Median:   numeric → float64
//   - Variance: numeric → float64
//
// For Arrow: delegates to arrow/math SIMD operations.
// For SQL: generates SELECT SUM/AVG/MIN/MAX/COUNT queries.
type Aggregator interface {
	Sum(col AnyColumn) (AnyColumn, error)
	Mean(col AnyColumn) (AnyColumn, error)
	MinMax(col AnyColumn) (mnCol AnyColumn, mxCol AnyColumn, err error)
	Count(col AnyColumn) (AnyColumn, error)
	Median(col AnyColumn) (AnyColumn, error)
	Variance(col AnyColumn) (AnyColumn, error)
}

// Caster provides engine-controlled type casting.
// Casting is an engine operation — the engine knows its native column
// types and how to convert between them.
type Caster interface {
	Cast(col AnyColumn, target DType) (AnyColumn, error)
}

// Selector provides engine-native column/row manipulation primitives.
// These are the building blocks for Frame verbs (Select, Arrange, Head, etc.).
//
// For Arrow: zero-copy slicing, compute Take kernel, sort-indices kernel.
// For Memory: direct slice operations.
// For SQL: generates ORDER BY, LIMIT/OFFSET, WHERE rowid IN (...).
type Selector interface {
	// Select reorders/selects rows by index (scatter-gather).
	// This is the Arrow "Take" kernel.
	Select(col AnyColumn, indices []int) (AnyColumn, error)

	// Slice returns rows [start, end) from a column.
	// For Arrow: zero-copy via array.NewSlice.
	Slice(col AnyColumn, start, end int) (AnyColumn, error)

	// SortIndices returns the permutation that sorts the column ascending.
	// Returns an int slice, not a column — it's metadata for Take().
	SortIndices(col AnyColumn) ([]int, error)

	// FilterIndices returns the row indices where mask[i] == true.
	// Returns an int slice for use with Take().
	FilterIndices(mask []bool) []int
}

// Windower provides window function kernels.
// For Arrow: streaming accumulators over Arrow arrays.
// For SQL: generates OVER() / WINDOW clauses.
type Windower interface {
	Lag(col AnyColumn, n int) (AnyColumn, error)
	Lead(col AnyColumn, n int) (AnyColumn, error)
	CumSum(col AnyColumn) (AnyColumn, error)
	CumMax(col AnyColumn) (AnyColumn, error)
	CumMin(col AnyColumn) (AnyColumn, error)
	Rank(col AnyColumn) (AnyColumn, error)
	DenseRank(col AnyColumn) (AnyColumn, error)
	PercentRank(col AnyColumn) (AnyColumn, error)
	RowNumber(n int) (AnyColumn, error)
}

// Joiner provides join operations across datasets.
// For Arrow: hash-join with lazy indexed column views.
// For SQL: generates JOIN ... ON ... clauses.
type Joiner interface {
	Join(left, right Table, spec JoinSpec) (Table, error)
}

// Reshaper provides reshape/pivot operations.
// For Arrow: lazy column views (repeatedView, interleavedView).
// For SQL: generates CASE WHEN / UNPIVOT / CROSSTAB.
type Reshaper interface {
	PivotLonger(ds Table, spec PivotLongerSpec) (Table, error)
	PivotWider(ds Table, spec PivotWiderSpec) (Table, error)
	Separate(ds Table, col string, into []string, sep string) (Table, error)
	Concatenate(ds Table, col string, from []string, sep string) (Table, error)
	Complete(ds Table, cols ...string) (Table, error)
}

// Filterer provides mask-based row filtering.
// For Arrow: boolean mask filtering with zero-copy.
// For SQL: generates WHERE clauses.
type Filterer interface {
	Filter(ds Table, mask Masker) (Table, error)
}

// Filler provides missing-value handling operations.
// For Arrow: streaming fill with zero allocation.
// For SQL: generates COALESCE / window-based fill.
type Filler interface {
	Fill(col AnyColumn, dir FillDirection) (AnyColumn, error)
	DropNA(ds Table, cols ...string) (Table, error)
	ReplaceNA(col AnyColumn, defaultVal float64) (AnyColumn, error)
}

// Composer provides row/column binding operations.
// For Arrow: zero-copy concatenation of Arrow arrays.
// For SQL: UNION ALL / lateral join.
type Composer interface {
	Stack(datasets ...Table) (Table, error)
	Combine(datasets ...Table) (Table, error)
}

// MathKernel provides element-wise mathematical transforms on numeric columns.
//
// Arrow engine: uses Arrow compute Datum API when available, highway SIMD for gaps.
// Memory engine: uses highway SIMD on raw slices, falls back to math stdlib.
// SQL engine: generates MATH functions (EXP, LOG, SIN, etc.)
//
// All methods require float64 columns unless noted (bitwise requires int64).
type MathKernel interface {
	// Binary arithmetic (column × column, same length required)
	AddCols(a, b AnyColumn) (AnyColumn, error)
	SubCols(a, b AnyColumn) (AnyColumn, error)
	MulCols(a, b AnyColumn) (AnyColumn, error)
	DivCols(a, b AnyColumn) (AnyColumn, error)

	// Scalar arithmetic (column × scalar)
	AddScalar(col AnyColumn, val float64) (AnyColumn, error)
	MulScalar(col AnyColumn, val float64) (AnyColumn, error)

	// Unary numeric
	Abs(col AnyColumn) (AnyColumn, error)
	Neg(col AnyColumn) (AnyColumn, error)
	Sign(col AnyColumn) (AnyColumn, error)
	Sqrt(col AnyColumn) (AnyColumn, error)
	Pow(col AnyColumn, exp float64) (AnyColumn, error)

	// Transcendental — logarithmic
	Exp(col AnyColumn) (AnyColumn, error)
	Ln(col AnyColumn) (AnyColumn, error)
	Log2(col AnyColumn) (AnyColumn, error)
	Log10(col AnyColumn) (AnyColumn, error)

	// Transcendental — trigonometric
	Sin(col AnyColumn) (AnyColumn, error)
	Cos(col AnyColumn) (AnyColumn, error)
	Tan(col AnyColumn) (AnyColumn, error)
	Asin(col AnyColumn) (AnyColumn, error)
	Acos(col AnyColumn) (AnyColumn, error)
	Atan(col AnyColumn) (AnyColumn, error)
	Atan2(y, x AnyColumn) (AnyColumn, error)

	// Transcendental — hyperbolic / special
	Tanh(col AnyColumn) (AnyColumn, error)
	Sigmoid(col AnyColumn) (AnyColumn, error)
	Erf(col AnyColumn) (AnyColumn, error)

	// Rounding
	Round(col AnyColumn) (AnyColumn, error)
	Floor(col AnyColumn) (AnyColumn, error)
	Ceil(col AnyColumn) (AnyColumn, error)

	// Bitwise (int64 columns only)
	BitAnd(a, b AnyColumn) (AnyColumn, error)
	BitOr(a, b AnyColumn) (AnyColumn, error)
	BitXor(a, b AnyColumn) (AnyColumn, error)
	BitNot(col AnyColumn) (AnyColumn, error)
	BitShiftLeft(col AnyColumn, n int) (AnyColumn, error)
	BitShiftRight(col AnyColumn, n int) (AnyColumn, error)
}

// StatKernel provides statistical compute kernels that produce new Tables.
// These are higher-level operations that consume one or more columns and
// produce a complete result table.
//
// For Memory/Arrow: implemented via go-highway SIMD + stdlib math.
// For SQL: could generate UDFs or client-side fallback.
type StatKernel interface {
	// Histogram bins a numeric column into equal-width bins.
	// Returns a Table with columns: "x" (bin centers) and "count" (frequencies).
	// nBins <= 0 means auto-select using Sturges' rule.
	Histogram(col AnyColumn, nBins int) (Table, error)

	// KDE computes kernel density estimation over a numeric column.
	// Returns a Table with columns: "x" (grid points) and "density".
	// bandwidth <= 0 means Silverman auto-select. points is the output grid size.
	KDE(ctx context.Context, col AnyColumn, bandwidth float64, points int) (Table, error)

	// LinearFit computes OLS linear regression y = a + b*x.
	// Returns a Table with columns: "x" (grid) and "y" (fitted values).
	// nOut is the number of output grid points.
	LinearFit(xCol, yCol AnyColumn, nOut int) (Table, error)

	// LoessFit computes locally weighted regression (LOESS).
	// Returns a Table with columns: "x" (grid) and "y" (fitted values).
	// nOut is the number of output grid points.
	LoessFit(ctx context.Context, xCol, yCol AnyColumn, nOut int) (Table, error)

	// Boxplot computes the five-number summary for a numeric column,
	// optionally grouped by a categorical column.
	// Returns a Table with columns: "x", "lower", "q1", "middle", "q3",
	// "upper", "notch_lower", "notch_upper".
	// groupCol may be nil for a single-group boxplot.
	// whisker is "tukey" (1.5*IQR) or "range" (min-max).
	Boxplot(yCol, groupCol AnyColumn, whisker string, notch bool) (Table, error)
}

// --- IO ---

// CSVConfig holds engine-agnostic CSV configuration.
// The dataset/csv facade constructs this from functional options and
// passes it to the engine's CSVReader/CSVWriter implementation.
type CSVConfig struct {
	HasHeader  bool
	Comma      rune
	Comment    rune
	NullValues []string
	// ChunkSize is the number of rows per batch. 0 means engine default.
	// Arrow default: 65536, Memory default: unlimited.
	ChunkSize int
}

// CSVReader reads CSV data into an engine-native Dataset.
// Memory engine: uses go-simdcsv + schema inference.
// Arrow engine: uses arrow/csv.NewInferringReader for zero-copy ingest.
type CSVReader interface {
	ReadCSV(ctx context.Context, r io.Reader, cfg CSVConfig) (Table, error)
}

// CSVWriter writes a Dataset to CSV.
// Memory engine: uses go-simdcsv Writer.
// Arrow engine: uses go-simdcsv Writer (generic — CSV output is string-based).
type CSVWriter interface {
	WriteCSV(ctx context.Context, w io.Writer, ds Table, cfg CSVConfig) error
}

// ParquetConfig holds engine-agnostic Parquet configuration.
type ParquetConfig struct {
	// Compression codec: "snappy", "gzip", "zstd", "lz4", "none".
	Compression string
}

// ParquetReader reads Parquet data into an engine-native Dataset.
// Memory engine: uses parquet-go for struct-based row reading.
// Arrow engine: uses pqarrow.ReadTable for zero-copy columnar ingest.
type ParquetReader interface {
	ReadParquet(ctx context.Context, r io.ReaderAt, size int64, cfg ParquetConfig) (Table, error)
}

// ParquetWriter writes a Dataset to Parquet format.
// Memory engine: uses parquet-go GenericWriter.
// Arrow engine: uses pqarrow.WriteTable.
type ParquetWriter interface {
	WriteParquet(ctx context.Context, w io.Writer, ds Table, cfg ParquetConfig) error
}
