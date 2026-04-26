# Dataset — Grammar of Data Manipulation

> A unified backend for dataframe operations, inspired by R's [tidyverse](https://r4ds.hadley.nz/).
> Built on a pluggable **Engine** architecture with Apache Arrow, Go-slice, and SQL backends.

---

## Design Philosophy

### Engine First

Every data operation — storage, aggregation, windowing, joining, reshaping —
is **delegated to an Engine backend**. The `dataset/` package defines only
interfaces and contracts. There are **no fallbacks**, no concrete column types
in the core package, and no implementation-specific logic leaking upward.

```
User Code → Frame (fluent API) → Engine (Arrow / Memory / SQL)
```

### Key Principles

| Principle | Implementation |
|---|---|
| Engine owns all computation | Sub-interfaces: `Aggregator`, `Windower`, `Joiner`, `Reshaper`, `Filterer`, `Filler`, `Composer` |
| No fallbacks | If an engine doesn't implement a sub-interface, it's an error — not a slow path |
| Two-tier column design | `AnyColumn` (type-erased) + `Column[T]` (generic typed access) |
| Arrow-aligned schema | `Field` → `arrow.Field`, `Schema` → `arrow.Schema` |
| Eager evaluation | Frame verbs execute immediately via the engine; `Collect()` returns the accumulated result. BigQuery has engine-specific lazy SQL accumulation. |
| Zero-copy when possible | Arrow backend uses zero-copy slicing, mmap IPC, SIMD kernels |
| Billion-row ready | Streaming `Builder` construction — no boxing, no intermediate slices |

---

## Package Map

```
github.com/TuSKan/ggplot/
│
├── dataset/                     # Core interfaces & contracts (NO concrete types)
│   ├── types.go                 #   DType enum (Float64, Int64, String, Bool, Timestamp)
│   ├── column.go                #   Field, Schema, AnyColumn, Column[T], GetColumn[T]
│   ├── dataset.go               #   Dataset interface, Names(), Close()
│   ├── engine.go                #   Engine + sub-interfaces (13 total, incl. MathKernel)
│   ├── frame.go                 #   Frame fluent API (all verbs)
│   ├── filter.go                #   Masker interface
│   ├── fill.go                  #   FillDirection enum
│   ├── join.go                  #   JoinType, JoinSpec
│   ├── pivot.go                 #   PivotLongerSpec, PivotWiderSpec
│   │
│   ├── compute/                 #   go-highway SIMD primitives
│   │   ├── compute.go           #     Vec[T] type alias, Lanes constraint, NumLanes
│   │   ├── reduction.go         #     SliceSum, SliceMin, SliceMax, SliceMinMax
│   │   ├── arithmetic.go        #     Add, Sub, Mul, Div, FMA, Neg (vector ops)
│   │   ├── math.go              #     Sqrt, Abs, Ceil, Floor, Round (vector ops)
│   │   ├── bitwise.go           #     And, Or, Xor, Not, ShiftLeft, ShiftRight
│   │   ├── comparison.go        #     Equal, Less, LessEqual, Greater, GreaterEqual
│   │   ├── conditional.go       #     Select, Blend (mask-based)
│   │   ├── shuffle.go           #     Reverse, Broadcast
│   │   ├── memory.go            #     Load, Store, MaskedLoad, MaskedStore
│   │   ├── convert.go           #     ConvertTo (lane type conversion)
│   │   ├── float.go             #     ApproxReciprocal, ApproxReciprocalSqrt
│   │   └── util.go              #     AllTrue, AnyTrue, FirstTrue, CountTrue
│   │
│   ├── math/                    #   go-highway transcendental transforms
│   │   ├── transforms.go        #     Exp, Log, Tanh, Sigmoid, Erf (dst,src pattern)
│   │   └── vec.go               #     Sin, Cos (SIMD vector transforms)
│   │
│   ├── sort/                    #   go-highway sort primitives
│   │   ├── sort.go              #     Sort (SIMD radix), NthElement (O(n)), SortIndices
│   │   └── sort_test.go         #     Correctness + benchmarks
│   │
│   ├── memory/                  #   Go-slice engine backend
│   │   ├── engine.go            #     All sub-interfaces implemented
│   │   ├── join.go              #     Joiner: hash-join (6 types)
│   │   ├── reshape.go           #     Reshaper: PivotLonger/Wider, Separate, Concatenate, Complete
│   │   ├── math_kernel.go       #     MathKernel: 36 ops via highway + stdlib
│   │   ├── sort.go              #     ParallelSortFunc (parallel merge-sort)
│   │   ├── engine_test.go       #     Full engine tests
│   │   ├── math_kernel_test.go  #     MathKernel tests (14 tests)
│   │   └── bench_test.go        #     Benchmarks: all ops at 1K–10M
│   │
│   ├── arrow/                   #   Apache Arrow SIMD engine backend
│   │   ├── engine.go            #     All sub-interfaces implemented
│   │   ├── join.go              #     Joiner: hash-join (6 types)
│   │   ├── reshape.go           #     Reshaper: PivotLonger/Wider, Separate, Concatenate, Complete
│   │   ├── csv.go               #     CSVReader/Writer via arrow/csv (chunked, 64K batch)
│   │   ├── parquet.go           #     ParquetReader/Writer via pqarrow (zero-copy)
│   │   ├── math_kernel.go       #     MathKernel: Arrow compute + highway fallback
│   │   ├── engine_test.go       #     Full engine tests
│   │   ├── math_kernel_test.go  #     MathKernel tests (17 tests)
│   │   └── bench_test.go        #     Benchmarks: all ops at 1K–10M
│   │
│   ├── sql/                     #   SQL pushdown engine backend (planned)
│   │
│   ├── csv/                     #   CSV facade (pure API, no heavy imports)
│   │   ├── csv.go               #     Read/Write dispatch via CSVReader/CSVWriter interfaces
│   │   └── csv_test.go          #     Tests for both engines
│   │
│   ├── parquet/                 #   Parquet facade (pure API, no heavy imports)
│   │   ├── parquet.go           #     Read/Write dispatch via ParquetReader/ParquetWriter
│   │   └── parquet_test.go      #     Tests: round-trip, nulls, cross-engine, compression
│   │
│   ├── ipc/                     #   Arrow IPC reader (planned)
│   ├── json/                    #   JSON reader (planned)
│   ├── factor/                  #   Factor/categorical ops (planned)
│   ├── strings/                 #   String column operations (planned)
│   └── datetime/                #   Date/time ops on DTypeTimestamp columns (planned)
│
├── docs/
│   ├── DATASET.md               #   Architecture blueprint (this file)
│   └── BENCHMARK.md             #   Performance results at 1K–10M rows
│
├── ggplot.go                    #   Visualization engine
└── ...
```

---

## Layer Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        User Code                             │
│                                                              │
│  dataset.From(ds).Filter(...).PivotLonger(...).              │
│    LeftJoin(other, "id").Mutate(...).Collect()               │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│                    Frame (Fluent API)                        │
│                                                              │
│  Select       Filter       Mutate       Arrange              │
│  GroupBy      Summarize    PivotLonger  PivotWider           │
│  Separate     Fill         DropNA       Complete             │
│  LeftJoin     InnerJoin    FullJoin     SemiJoin             │
│  AntiJoin     Lag          Lead         CumSum               │
│  Rank         DenseRank    RowNumber    Stack                │
│  Combine      Distinct     Head         Tail                 │
│  Slice                                                       │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│               Engine Dispatch (type assertion)               │
│                                                              │
│  Frame checks: does engine implement Aggregator? Joiner?     │
│  If yes → delegate. If no → error (no fallback).             │
│                                                              │
├──────────────┬──────────────────┬────────────────────────────┤
│  memory/     │   arrow/         │   sql/                     │
│              │                  │                            │
│  []float64   │  Arrow arrays    │   Query builder            │
│  []string    │  Arrow compute   │   Predicate pushdown       │
│  []int64     │  go-highway SIMD │   JOIN pushdown            │
│  Go iterate  │  zero-copy slice │   GROUP BY pushdown        │
└──────────────┴──────────────────┴────────────────────────────┘
```

---

## Core Types

### DType (`types.go`)

Logical type identifier — analogous to `arrow.Type` but limited to the
types this ecosystem supports.

```go
type DType int

const (
    DTypeFloat64    DType = iota  // 64-bit floating point
    DTypeInt64                    // 64-bit integer
    DTypeString                   // string / categorical
    DTypeBool                     // boolean
    DTypeTimestamp                 // int64 nanoseconds since Unix epoch
    DTypeUnknown                  // unrecognized
)

func (d DType) String() string   // "float64", "int64", "string", "bool", "timestamp[ns]"
```

### Field (`column.go`)

Describes a single column — maps directly to `arrow.Field`.
Metadata carries type-specific parameters that DType alone cannot express.

```go
type Field struct {
    Name     string
    Dtype    DType
    Nullable bool
    Metadata map[string]string  // e.g. {"tz": "America/Sao_Paulo"}
}

// Builder-style API:
field := dataset.TimestampCol("created_at").
    WithNullable().
    WithMetadata(map[string]string{"tz": "UTC"})

// Convenience constructors:
FloatCol(name)           → Field{Dtype: DTypeFloat64}
IntCol(name)             → Field{Dtype: DTypeInt64}
StringCol(name)          → Field{Dtype: DTypeString}
BoolCol(name)            → Field{Dtype: DTypeBool}
TimestampCol(name)       → Field{Dtype: DTypeTimestamp}
NullableFloatCol(name)   → Field{Dtype: DTypeFloat64, Nullable: true}
NullableIntCol(name)     → Field{Dtype: DTypeInt64, Nullable: true}
NullableStringCol(name)  → Field{Dtype: DTypeString, Nullable: true}
```

### Schema (`column.go`)

Describes the complete structure of a dataset — an ordered collection of
Fields with a name-to-index lookup. Maps directly to `arrow.Schema`.

```go
type Schema struct {
    fields []Field           // ordered
    index  map[string]int    // name → position
}

schema := dataset.NewSchema(
    dataset.FloatCol("x"),
    dataset.StringCol("label"),
    dataset.IntCol("id"),
)

schema.NumFields()           // 3
schema.Field(0)              // Field{Name: "x", Dtype: DTypeFloat64}
schema.FieldIndex("label")   // 1
schema.HasField("missing")   // false
```

### Two-Tier Column: AnyColumn + Column[T] (`column.go`)

Go's type system doesn't allow generic methods on interfaces. This creates
a fundamental tension: engines need type-erased columns for storage, but
users need typed access for safety. The **Two-Tier pattern** solves this:

```
AnyColumn (type-erased)         Column[T] (generic typed access)
┌────────────────────┐          ┌──────────────────────────┐
│ Name() string      │          │ AnyColumn                │
│ Len() int          │◄─────────│ Values() []T             │
│ DType() DType      │          │ IsNull() []bool          │
└────────────────────┘          └──────────────────────────┘
        ▲                                   ▲
        │                                   │
  engines store this               users cast to this
  in maps/slices                   via GetColumn[T]()
```

```go
// AnyColumn — type-erased, stored in Dataset, passed to Engine methods
type AnyColumn interface {
    Name() string
    Len() int
    DType() DType
}

// Column[T] — typed access layer, extends AnyColumn
type Column[T any] interface {
    AnyColumn
    Values() []T       // zero-copy for Arrow, direct slice for memory
    IsNull() []bool    // nil = no nulls (common case, zero alloc)
}

// GetColumn[T] — the single type-assertion bridge
func GetColumn[T any](ds Dataset, name string) (Column[T], error)
```

**Every engine column type implements both**:
```go
// memory engine
var _ dataset.AnyColumn       = (*float64Column)(nil)
var _ dataset.Column[float64] = (*float64Column)(nil)

// arrow engine
var _ dataset.AnyColumn       = (*arrowFloat64Column)(nil)
var _ dataset.Column[float64] = (*arrowFloat64Column)(nil)
```

### Dataset (`dataset.go`)

```go
type Dataset interface {
    Schema() *Schema
    Column(name string) (AnyColumn, error)
    Len() int
}

// Free functions:
func Names(ds Dataset) []string     // extract column names from schema
func Close(ds Dataset) error        // release resources if Closer
func GetEngine(ds Dataset) Engine   // extract engine if HasEngine
```

---

## Engine Architecture (`engine.go`)

### Engine (required)

Every backend must implement `Engine`. This is the only required interface.

```go
type Engine interface {
    Name() string    // "memory", "arrow", "sql"
}
```

### HasEngine — Engine Propagation

```go
type HasEngine interface {
    Dataset
    Engine() Engine
}

// When transformations produce new datasets, they carry the engine forward.
// stat packages and ggplot can produce new datasets using the same engine
// without importing engine-specific packages.
func GetEngine(ds Dataset) Engine   // nil if dataset has no engine
```

### Data Construction

Two patterns for creating engine-native columns:

#### ColumnFactory — wrap existing slices

```go
type ColumnFactory interface {
    NewFloat64Column(name string, data []float64) AnyColumn
    NewInt64Column(name string, data []int64) AnyColumn
    NewStringColumn(name string, data []string) AnyColumn
    NewBoolColumn(name string, data []bool) AnyColumn
    NewTimestampColumn(name string, data []int64) AnyColumn
    FromColumns(schema *Schema, cols ...AnyColumn) (Dataset, error)
}
```

- **Memory engine**: wraps the slice directly (zero-copy)
- **Arrow engine**: builds an Arrow array (one allocation)

#### BuilderFactory — streaming construction (billion-row scale)

```go
type BuilderFactory interface {
    NewBuilder(schema *Schema) Builder
}

type Builder interface {
    Float64(col string) Float64Appender
    Int64(col string) Int64Appender
    String(col string) StringAppender
    Bool(col string) BoolAppender
    Build() (Dataset, error)
}

// Each appender is typed — no boxing, no interface{} per row:
type Float64Appender interface {
    Append(v float64)
    AppendNull()
    AppendValues(vs []float64)
    Reserve(n int)
}
// Int64Appender, StringAppender, BoolAppender follow the same pattern.
```

### Sub-Interfaces (optional, no fallbacks)

The Frame layer checks via type assertion whether an engine supports each
capability. If not, the operation returns an error — **never a slow fallback**.

#### Selector

Engine-native row manipulation — scatter-gather, slicing, sort permutation.

```go
type Selector interface {
    Take(col AnyColumn, indices []int) (AnyColumn, error)   // scatter-gather
    SliceColumn(col AnyColumn, start, end int) (AnyColumn, error) // sub-range
    SortIndices(col AnyColumn) ([]int, error)               // sort permutation
    FilterIndices(mask []bool) []int                        // mask → indices
}

// memory: direct slice ops, slices.SortFunc
// arrow: array.NewSlice (zero-copy), builder-based Take
```

#### Aggregator

Returns `AnyColumn` (single-element), preserving input type — aligned with
Arrow compute kernel type rules:

```go
type Aggregator interface {
    Sum(col AnyColumn) (AnyColumn, error)       // numeric → same type
    Mean(col AnyColumn) (AnyColumn, error)      // numeric → float64
    Min(col AnyColumn) (AnyColumn, error)       // any ordered type
    Max(col AnyColumn) (AnyColumn, error)       // any ordered type
    Count(col AnyColumn) (AnyColumn, error)     // any → int64
    Median(col AnyColumn) (AnyColumn, error)    // numeric → float64
    Variance(col AnyColumn) (AnyColumn, error)  // numeric → float64
}

// Type preservation examples:
//   Sum(float64 col) → AnyColumn wrapping float64
//   Sum(int64 col)   → AnyColumn wrapping int64
//   Min(string col)  → AnyColumn wrapping string (lexicographic)
//   Min(timestamp)   → AnyColumn wrapping int64 (earliest)
//   Count(any col)   → AnyColumn wrapping int64
```

#### Caster

```go
type Caster interface {
    Cast(col AnyColumn, target DType) (AnyColumn, error)
}
```

#### Windower

```go
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
```

#### Joiner

```go
type Joiner interface {
    Join(left, right Dataset, spec JoinSpec) (Dataset, error)
}
```

#### Reshaper

```go
type Reshaper interface {
    PivotLonger(ds Dataset, spec PivotLongerSpec) (Dataset, error)
    PivotWider(ds Dataset, spec PivotWiderSpec) (Dataset, error)
    Separate(ds Dataset, col string, into []string, sep string) (Dataset, error)
    Concatenate(ds Dataset, col string, from []string, sep string) (Dataset, error)
    Complete(ds Dataset, cols ...string) (Dataset, error)
}
```

#### Filterer, Filler, Composer

```go
type Filterer interface {
    Filter(ds Dataset, mask Masker) (Dataset, error)
}

type Filler interface {
    Fill(col AnyColumn, dir FillDirection) (AnyColumn, error)
    DropNA(ds Dataset, cols ...string) (Dataset, error)
    ReplaceNA(col AnyColumn, defaultVal float64) (AnyColumn, error)
}

type Composer interface {
    Stack(datasets ...Dataset) (Dataset, error)
    Combine(datasets ...Dataset) (Dataset, error)
}
```

#### MathKernel

Element-wise mathematical transforms — 36 operations across arithmetic,
transcendental, rounding, and bitwise categories.

```go
type MathKernel interface {
    // Arithmetic (binary column ops)
    AddCols(a, b AnyColumn) (AnyColumn, error)
    SubCols(a, b AnyColumn) (AnyColumn, error)
    MulCols(a, b AnyColumn) (AnyColumn, error)
    DivCols(a, b AnyColumn) (AnyColumn, error)
    AddScalar(col AnyColumn, val float64) (AnyColumn, error)
    MulScalar(col AnyColumn, val float64) (AnyColumn, error)

    // Unary math
    Abs(col AnyColumn) (AnyColumn, error)
    Neg(col AnyColumn) (AnyColumn, error)
    Sign(col AnyColumn) (AnyColumn, error)
    Sqrt(col AnyColumn) (AnyColumn, error)
    Pow(col AnyColumn, exp float64) (AnyColumn, error)

    // Logarithmic / Exponential
    Exp(col AnyColumn) (AnyColumn, error)
    Ln(col AnyColumn) (AnyColumn, error)
    Log2(col AnyColumn) (AnyColumn, error)
    Log10(col AnyColumn) (AnyColumn, error)

    // Trigonometric
    Sin(col AnyColumn) (AnyColumn, error)
    Cos(col AnyColumn) (AnyColumn, error)
    Tan(col AnyColumn) (AnyColumn, error)
    Asin(col AnyColumn) (AnyColumn, error)
    Acos(col AnyColumn) (AnyColumn, error)
    Atan(col AnyColumn) (AnyColumn, error)
    Atan2(y, x AnyColumn) (AnyColumn, error)

    // Special / Activation
    Tanh(col AnyColumn) (AnyColumn, error)
    Sigmoid(col AnyColumn) (AnyColumn, error)
    Erf(col AnyColumn) (AnyColumn, error)

    // Rounding
    Round(col AnyColumn) (AnyColumn, error)
    Floor(col AnyColumn) (AnyColumn, error)
    Ceil(col AnyColumn) (AnyColumn, error)

    // Bitwise (int64 columns)
    BitAnd(a, b AnyColumn) (AnyColumn, error)
    BitOr(a, b AnyColumn) (AnyColumn, error)
    BitXor(a, b AnyColumn) (AnyColumn, error)
    BitNot(col AnyColumn) (AnyColumn, error)
    BitShiftLeft(col AnyColumn, n int) (AnyColumn, error)
    BitShiftRight(col AnyColumn, n int) (AnyColumn, error)
}

// Arrow engine: Arrow compute first → highway fallback → stdlib
// Memory engine: highway transforms → stdlib math
```

### Engine Dispatch Pattern

```go
// Inside Frame methods:
func (f Frame) Summarize(...) Frame {
    eng := GetEngine(f.ds)
    agg, ok := eng.(Aggregator)
    if !ok {
        // ERROR — no fallback
        return f.withError(fmt.Errorf("engine %q does not support aggregation", eng.Name()))
    }
    // delegate to agg.Sum(), agg.Mean(), etc.
}
```

---

## Engine Backends

### memory/ — Go-Slice Engine

Lightweight backend for tests, prototyping, and small–medium data.
Uses Go slices with go-highway SIMD acceleration.

```go
eng := memory.NewEngine()
schema := dataset.NewSchema(
    dataset.FloatCol("x"),
    dataset.StringCol("label"),
)
ds, _ := eng.FromColumns(schema,
    eng.NewFloat64Column("x", []float64{1, 2, 3}),
    eng.NewStringColumn("label", []string{"a", "b", "c"}),
)
```

**Implements**: `Engine`, `ColumnFactory`, `BuilderFactory`, `Aggregator`, `Caster`, `Selector`, `Windower`, `Filler`, `Filterer`, `Composer`, `MathKernel`

**MathKernel dispatch**: highway SIMD transforms → stdlib `math.*` fallback.

### arrow/ — Apache Arrow SIMD Engine

High-performance backend using Arrow arrays, Arrow compute kernels,
go-highway SIMD for gaps, and Arrow's zero-copy slicing.

```go
eng := arrow.NewEngine(memory.DefaultAllocator)
schema := dataset.NewSchema(
    dataset.FloatCol("x"),
    dataset.StringCol("label"),
)
ds, _ := eng.FromColumns(schema,
    eng.NewFloat64Column("x", []float64{1, 2, 3}),
    eng.NewStringColumn("label", []string{"a", "b", "c"}),
)
```

**Implements**: `Engine`, `ColumnFactory`, `BuilderFactory`, `Aggregator`, `Caster`, `Selector`, `Windower`, `Filler`, `Filterer`, `Composer`, `MathKernel`

**MathKernel dispatch**: Arrow compute (23 ops) → go-highway (4 ops) → stdlib fallback.

| Category | Arrow Compute | Highway (gap fill) | Stdlib |
|---|---|---|---|
| Arithmetic | Add, Sub, Mul, Div | — | — |
| Unary | Abs, Neg, Sign, Power | — | Sqrt |
| Logarithmic | Ln, Log2, Log10 | Exp | — |
| Trigonometric | Sin, Cos, Tan, Asin, Acos, Atan, Atan2 | — | — |
| Special | — | Tanh, Sigmoid, Erf | — |
| Rounding | — | — | Round, Floor, Ceil |
| Bitwise | — | — | &, \|, ^, ~, <<, >> |

**Aggregator**: Arrow `math.Float64.Sum` + go-highway `NthElement` (O(n) Median).

**Selector**:
- `Slice` → `array.NewSlice` (zero-copy, O(1))
- `Take` → `compute.TakeArray` (Arrow native)
- `SortIndices` → `compute.SortIndicesArray` (Arrow native)
- `Filter` → `compute.FilterArray` (Arrow native)

### bigquery/ — Google BigQuery Engine

High-performance SQL pushdown engine that executes all computations in BigQuery.
Data only reaches local memory when explicitly evaluated.

```go
eng, _ := bigquery.NewEngine(ctx, "my-project")
defer eng.Close()

ds := eng.Table("analytics", "events")

result, _ := dataset.From(ds).
    Select("region", "revenue").
    Filter(dataset.Gt("revenue", 1000)).
    Collect()
```

**Features**:
- **Storage Read API**: Downloads data via Apache Arrow IPC for ultra-fast materialization.
- **Lazy Evaluation**: Accumulates `SelectedFields` and `RowRestriction` without executing.
- **SQL Translation**: Complex operations (GROUP BY, JOIN) generate SQL Jobs that write to temporary tables.
- **Local Fallback**: Seamlessly bridges to the local `arrow` engine for operations not supported by SQL.

---

## Frame API (Fluent Verbs)

Frame is the user-facing fluent API. All verbs return a new Frame (immutable chain).
Every verb delegates to the dataset's engine.

### Data Manipulation (dplyr)

```go
dataset.From(ds).
    Select("x", "y", "group").
    Filter(dataset.Gt("x", 0)).
    Mutate("z", dataset.Transform("x", fn)).
    Rename("old", "new").
    Arrange("x").
    Distinct("x", "y").
    Head(100).
    Tail(50).
    Slice(10, 20).
    GroupBy("group").
        Summarize(
            dataset.Mean("avg", "x"),
            dataset.Sum("total", "x"),
            dataset.Count("n", "x"),
            dataset.Min("lo", "x"),
            dataset.Max("hi", "x"),
            dataset.Variance("var", "x"),
        ).
    Collect()
```

### Joins

```go
dataset.From(orders).
    LeftJoin(customers, dataset.On("customer_id")).
    Select("order_id", "customer_name", "total")

dataset.From(a).InnerJoin(b, dataset.On("year", "month"))
dataset.From(orders).SemiJoin(vipCustomers, dataset.On("customer_id"))
dataset.From(orders).AntiJoin(customers, dataset.On("customer_id"))
```

### Reshape (tidyr)

```go
// PivotLonger — wide → long
dataset.From(wide).PivotLonger(
    dataset.Cols("Q1", "Q2", "Q3", "Q4"),
    dataset.NamesTo("quarter"),
    dataset.ValuesTo("revenue"),
)

// PivotWider — long → wide
dataset.From(long).PivotWider(
    dataset.NamesFrom("quarter"),
    dataset.ValuesFrom("revenue"),
)

// Separate — split column by delimiter
dataset.From(ds).Separate("date", []string{"year", "month", "day"}, "-")

// Fill / DropNA
dataset.From(ds).Fill("value", dataset.FillDown)
dataset.From(ds).DropNA("x", "y")
```

### Window Functions

```go
dataset.From(sales).
    Mutate("prev", dataset.Lag("revenue", 1)).
    Mutate("next", dataset.Lead("revenue", 1)).
    Mutate("running_total", dataset.CumSum("revenue")).
    Mutate("rank", dataset.Rank("score")).
    Mutate("dense", dataset.DenseRank("score")).
    Mutate("pct", dataset.PercentRank("score")).
    Mutate("row", dataset.RowNumber())
```

### Composing

```go
// Stack — vertical concatenation (like R's bind_rows)
dataset.From(q1).Stack(q2, q3, q4)

// Combine — horizontal concatenation (like R's bind_cols)
dataset.From(names).Combine(scores, ranks)
```

---

## Utility Packages

### Data Import/Export

```go
import (
    "github.com/TuSKan/ggplot/dataset/csv"
    "github.com/TuSKan/ggplot/dataset/parquet"
    "github.com/TuSKan/ggplot/dataset/ipc"
    "github.com/TuSKan/ggplot/dataset/json"
)

ds, _ := csv.Read("flights.csv", csv.WithEngine(eng))
ds, _ := parquet.Read("big_table.parquet", parquet.WithEngine(eng))
ds, _ := ipc.Read("data.arrow", ipc.WithEngine(eng))
ds, _ := json.Read("records.jsonl", json.WithEngine(eng))

csv.Write(ds, "output.csv")
parquet.Write(ds, "output.parquet")
```

### apply — Functional Programming (purrr)

```go
import "github.com/TuSKan/ggplot/dataset/apply"

doubled := apply.Map(xCol, func(v float64) float64 { return v * 2 })
total := apply.Reduce(xCol, 0.0, func(acc, v float64) float64 { return acc + v })
positive := apply.Keep(xCol, func(v float64) bool { return v > 0 })
transform := apply.Pipe(fn1, fn2, fn3)
```

### strings — String Operations (stringr)

```go
import "github.com/TuSKan/ggplot/dataset/strings"

col := strings.Detect(nameCol, "Smith")
col := strings.ToUpper(nameCol)
col := strings.Replace(textCol, "old", "new")
col := strings.PadLeft(idCol, 5, '0')
```

### factor — Factor Operations (forcats)

```go
import "github.com/TuSKan/ggplot/dataset/factor"

col := factor.InFreq(speciesCol)
col := factor.Reorder(speciesCol, massCol, factor.MedianFn)
col := factor.Lump(speciesCol, 5)
col := factor.Recode(speciesCol, map[string]string{"setosa": "Iris Setosa"})
```

### datetime — Date/Time Operations (lubridate)

Operates on `DTypeTimestamp` columns (int64 nanoseconds since Unix epoch).

```go
import "github.com/TuSKan/ggplot/dataset/datetime"

col := datetime.Parse(dateStrCol, "2006-01-02")
col := datetime.Year(dateCol)
col := datetime.AddMonths(dateCol, -3)
col := datetime.FloorDate(dateCol, datetime.Month)
col := datetime.Format(dateCol, "Jan 2, 2006")
```

---

## End-to-End Example

```go
package main

import (
    "fmt"

    "github.com/TuSKan/ggplot"
    "github.com/TuSKan/ggplot/aes"
    ds "github.com/TuSKan/ggplot/dataset"
    "github.com/TuSKan/ggplot/dataset/arrow"
    "github.com/TuSKan/ggplot/dataset/csv"
    dt "github.com/TuSKan/ggplot/dataset/datetime"
    fct "github.com/TuSKan/ggplot/dataset/factor"
    str "github.com/TuSKan/ggplot/dataset/strings"
    "github.com/TuSKan/ggplot/geom"
    "github.com/apache/arrow-go/v18/arrow/memory"
)

func main() {
    // ── Engine: Arrow SIMD backend ──
    eng := arrow.NewEngine(memory.DefaultAllocator)

    // ── Import CSV with engine ──
    flights, _ := csv.Read("flights.csv", csv.WithEngine(eng))
    defer ds.Close(flights)

    // ── Transform pipeline ──
    byCarrier := ds.From(flights).
        Filter(ds.And(
            str.PredicateContains("origin", "JFK"),
            dt.PredicateAfter("date", "2024-06-01"),
        )).
        Mutate("month", dt.MutateMonth("date")).
        Mutate("delay_hrs", ds.MapFloat64("dep_delay",
            func(v float64) float64 { return v / 60.0 },
        )).
        GroupBy("month", "carrier").
        Summarize(
            ds.Mean("avg_delay", "delay_hrs"),
            ds.Count("n_flights", "dep_delay"),
        )

    // ── Reorder for plotting ──
    ordered := ds.From(byCarrier.Dataset).
        Mutate("carrier", fct.MutateReorder("carrier", "avg_delay", fct.MeanFn))

    fmt.Println(ordered.String())

    // ── Visualize ──
    ggplot.New(ordered.Dataset,
        aes.X("month"), aes.Y("avg_delay"), aes.Color("carrier"),
    ).
        Layer(geom.Line(geom.WithLineWidth(2))).
        Layer(geom.Point(geom.WithSize(4))).
        Save("delays.png", 900, 500)
}
```

---

## Implementation Phases

Always ask for review before starting each task.

### Phase 1 — Engine Core ✅

Core `dataset/` interfaces compiling with both engines satisfying them.

| Status | File | Contents |
|---|---|---|
| ✅ | `types.go` | `DType` enum + `String()` |
| ✅ | `column.go` | `Field`, `Schema`, `AnyColumn`, `Column[T]`, `GetColumn[T]()` |
| ✅ | `dataset.go` | `Dataset{Schema(), Column(), Len()}`, `Names()`, `Close()` |
| ✅ | `engine.go` | `Engine` + 13 sub-interfaces incl. `MathKernel` |
| ✅ | `memory/engine.go` | All sub-interfaces implemented |
| ✅ | `arrow/engine.go` | All sub-interfaces implemented |
| ✅ | Tests | Full engine + frame tests for both engines |

### Phase 2 — Frame + SIMD ✅

Fluent API on engine dispatch + SIMD compute kernels.

| Status | Deliverable |
|---|---|
| ✅ | `frame.go` — All verbs: Select, Filter, Mutate, Arrange, Head, Tail, Slice, Distinct |
| ✅ | `frame.go` — GroupBy, Summarize (Sum/Mean/Min/Max/Count/Median/Variance) |
| ✅ | `frame.go` — Join/Reshape/Fill/Compose dispatch stubs |
| ✅ | `compute/` — go-highway SIMD primitives (Load, Store, Add, Mul, Reduce, MinMax) |
| ✅ | `math/` — go-highway transcendental transforms (Exp, Log, Tanh, Sigmoid, Erf) |
| ✅ | `sort/` — go-highway Sort (SIMD radix), NthElement (O(n)), SortIndices |
| ✅ | `docs/BENCHMARK.md` — Full benchmark results at 1K–10M rows |

### Phase 3 — Engine Operations ✅

| Status | Deliverable |
|---|---|
| ✅ | Windower: Lag, Lead, CumSum, CumMax, CumMin, Rank, DenseRank, PercentRank, RowNumber |
| ✅ | Filler: Fill (FillDown/FillUp), DropNA, ReplaceNA |
| ✅ | Filterer: Filter (mask-based row filtering) |
| ✅ | Composer: Stack, Combine |
| ✅ | Caster: Cast between Float64, Int64, String |
| ✅ | Selector: Take, Slice, SortIndices, FilterIndices |

### Phase 4 — MathKernel ✅

| Status | Deliverable |
|---|---|
| ✅ | `MathKernel` interface — 36 element-wise mathematical operations |
| ✅ | `arrow/math_kernel.go` — Arrow compute first, highway/stdlib fallback |
| ✅ | `memory/math_kernel.go` — Highway + stdlib |
| ✅ | Tests — 17 arrow + 14 memory = 31 MathKernel tests |
| ✅ | Benchmarks — All ops at 1K–10M with comparison tables |

### Phase 5 — Reduction Optimization ✅

| Status | Deliverable |
|---|---|
| ✅ | Replaced SIMD Load/Store loops with pure Go scalar loops (zero-alloc) |
| ✅ | Optimized Median: single NthElement + O(n/2) max-scan |
| ✅ | Arrow MinMax: 129ms → 10.3ms (12.5× speedup, 375K → 16 allocs) |
| ✅ | Memory Mean: 84ms → 9.7ms (8.7× speedup, 2.5M → 2 allocs) |

### Phase 6 — Remaining Work

| Task | Status | Est. LOC |
|---|---|---|
| ~~Joiner (hash-join: Left, Right, Inner, Full, Semi, Anti)~~ | ✅ | done |
| ~~Reshaper (PivotLonger, PivotWider, Separate, Concatenate, Complete)~~ | ✅ | done |
| ~~`csv` — CSV reader/writer (go-simdcsv + arrow/csv)~~ | ✅ | done |
| ~~`parquet` — Parquet reader/writer (parquet-go + pqarrow)~~ | ✅ | done |
| `avro` — Avro reader/writer | 🔲 | ~100 |
| `ipc` — Arrow IPC reader/writer | 🔲 | ~100 |
| `json` — JSON reader/writer | 🔲 | ~100 |
| `apply` — Map, Reduce, Keep, Compose, Pipe | 🔲 | ~250 |
| `strings` — Detect, Replace, Upper, Trim, Pad | 🔲 | ~400 |
| `factor` — InFreq, Reorder, Lump, Recode | 🔲 | ~300 |
| `datetime` — Parse, Year, AddDays, Floor, Format | 🔲 | ~450 |
| SQL engine + query builder | 🔲 | ~750 |
