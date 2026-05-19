# Lazy Evaluation — Stat Transform Rules

> Reference: [`docs/DATASET.md`](../../docs/DATASET.md)

## Absolute Rule

**No materialization in `stat/` transforms.** The `stat` package composes engine operations. It never touches row data.

## Forbidden in `stat/`

The following are **strictly forbidden** in any stat transform:

| Forbidden pattern | Why |
|---|---|
| `.Values()` | Materializes column data into a Go slice |
| `.Float64(col, ...)` | Materializes + type-asserts column data |
| `.Int64(col, ...)` | Materializes + type-asserts column data |
| `.Collect(ctx)` | Forces lazy chain materialization |
| `[]float64` iteration | Manual loop over row data |
| `ReducerFunc` / `func([]float64) float64` | Scalar aggregation on materialized slices |
| `getFloat64Values` or equivalent helpers | Wrapper around materialization |

## Only Allowed Callers

| Caller | May call `.Values()` / `.Float64()` | Reason |
|---|---|---|
| `dataset/` package internals | ✅ | Engine owns computation |
| `dataset.ScalarFloat64()` | ✅ | Reads 1-element aggregate result |
| Geom `Draw()` methods | ✅ | Terminal render — data needed for GPU/SVG |
| Test assertion helpers | ✅ | After `Collect(ctx)` at assertion time |
| `stat/` transforms | ❌ **NEVER** | Must stay lazy |

## How Stat Transforms Must Work

Every stat transform must compose **only** engine interfaces and lazy `Dataset` verbs:

### Engine Interfaces (from `dataset/engine.go`)

| Interface | Operations | Use case |
|---|---|---|
| `Aggregator` | `Sum`, `Mean`, `MinMax`, `Count`, `Median`, `Variance` | GroupBy summaries, normalization totals |
| `MathKernel` | `MulScalar`, `DivCols`, `SubCols`, `AddCols`, ... | Element-wise scaling, differencing |
| `Windower` | `CumSum`, `CumMax`, `Lag`, `Lead`, `Rank`, ... | Cumulative transforms, stacking |
| `Selector` | `SortIndices`, `Select`, `Slice`, `FilterIndices` | Sorting, scatter-gather, filtering |
| `Filterer` | `Filter(Table, Masker)` | Row-level predicate filtering |
| `Caster` | `Cast(col, DType)` | Type conversion (int64→float64) |
| `ColumnFactory` | `NewFloat64Column`, `FromColumns` | Creating new columns from engine data |
| `Composer` | `Stack`, `Combine` | Vertical/horizontal concatenation |

### Lazy Dataset Verbs (from `dataset/frame.go`)

```go
// All return Dataset — no computation until Collect(ctx)
ds.Filter(masker)           // row-level filtering
ds.Arrange(col)             // sort ascending
ds.Head(n) / ds.Tail(n)     // first/last N rows
ds.Select(cols...)          // column selection
ds.SelectRows(indices)      // scatter-gather
ds.GroupBy(col).Summarize()  // grouped aggregation
ds.WithColumn(col)          // add/replace column
ds.Rename(old, new)         // rename column (lazy)
ds.Collect(ctx)             // ONLY at pipeline end / Draw
```

## Correct Patterns

### ✅ Normalize (engine Aggregator + MathKernel)

```go
func (n *normalizeTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
    col, _ := in.Data.Column(colName)
    sumCol, _ := agg.Sum(col)                        // engine-native
    sum, ok := dataset.ScalarFloat64(sumCol)          // 1-element scalar read
    scaled, _ := mk.MulScalar(col, n.cfg.total/sum)  // engine-native
    outData := in.Data.WithColumn(scaled)             // lazy
    return TransformResult{Data: outData, ...}, nil
}
```

### ✅ Filter (dataset Masker)

```go
func (f *filterTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
    outData := in.Data.Filter(f.masker)  // lazy — masker evaluated by engine
    return TransformResult{Data: outData, ...}, nil
}
```

### ✅ StackY (Windower + MathKernel + lazy Rename)

```go
func (s *stackYTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
    cumSumCol, _ := win.CumSum(col)           // engine-native
    yminCol, _ := mk.SubCols(cumSumCol, col)  // engine-native
    outData := in.Data.
        WithColumn(yminCol).                   // lazy
        Rename(yCol, "ymin").                  // lazy
        WithColumn(cumSumCol)                  // lazy
    return TransformResult{Data: outData, ...}, nil
}
```

### ✅ Sort Descending (Selector.SortIndices)

```go
func (s *sortTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
    col, _ := in.Data.Column(s.column)
    indices, _ := sel.SortIndices(col)      // engine-native permutation
    slices.Reverse(indices)                 // reverse in-place
    outData, _ := in.Data.SelectRows(indices)  // engine-native scatter-gather
    return TransformResult{Data: outData, ...}, nil
}
```

## Anti-Patterns

### ❌ Manual slice iteration

```go
// WRONG — materializes entire column
vals := fc.Values()
for i, v := range vals {
    mask[i] = predicate(v)
}
```

### ❌ Collect inside transform

```go
// WRONG — forces eager evaluation
collected, _ := sorted.Collect(ctx)
n := collected.NumRows()
```

### ❌ ReducerFunc on []float64

```go
// WRONG — scalar aggregation outside engine
type ReducerFunc func([]float64) float64
```

## Missing Engine Operations

If a transform needs an operation the engine doesn't provide:

1. **Add the interface** to `dataset/engine.go`
2. **Implement** in `dataset/memory/`, `dataset/arrow/`, `dataset/bigquery/`
3. **Use** from the stat transform via engine dispatch
4. **Never** implement the operation inside `stat/` on raw slices

## Why

Imagine a billion-row dataset in BigQuery. A `stat.FilterY` that calls `.Values()` downloads a billion floats to iterate locally. The engine's `Filterer.Filter` pushes the predicate to SQL and returns only matching rows — zero local memory. This is why **all computation must go through the engine**.
