# Dataset Typed Accessor Methods with Functional Options

## Goal

Add typed accessor methods to `Dataset` that eliminate the `GetColumn[T]` + `.Values()` two-step pattern used across `draw.go`, `stat/stat.go`, and tests. Use functional options for post-processing (NaN filter, fill, abs, etc.).

## API Design

### Core methods on `Dataset`

```go
func (d Dataset) Float64(name string, opts ...Float64Opt) ([]float64, error)
func (d Dataset) Int64(name string, opts ...Int64Opt) ([]int64, error)
func (d Dataset) Strings(name string, opts ...StringOpt) ([]string, error)
func (d Dataset) Bools(name string) ([]bool, error)
```

### Float64 options

```go
type Float64Opt func([]float64) []float64

// Clean removes NaN and ±Inf values.
func Clean() Float64Opt

// FillNaN replaces NaN values with the given constant.
func FillNaN(v float64) Float64Opt

// Abs applies math.Abs to every element.
func Abs() Float64Opt

// Clamp restricts values to [min, max].
func Clamp(min, max float64) Float64Opt

// Sorted returns the values sorted ascending.
func Sorted() Float64Opt
```

### Implementation sketch (in `dataset/column.go` or `dataset/accessors.go`)

```go
func (d Dataset) Float64(name string, opts ...Float64Opt) ([]float64, error) {
    if d.err != nil {
        return nil, d.err
    }
    col, err := GetColumn[float64](d.tbl, name)
    if err != nil {
        return nil, err
    }
    vals := col.Values()
    // Apply options in order.
    for _, opt := range opts {
        vals = opt(vals)
    }
    return vals, nil
}
```

> [!IMPORTANT]
> **Copy semantics**: When no options are passed, return the original slice (zero-copy). When any option is applied, copy first to avoid mutating the underlying column data. The first option in the chain should trigger the copy; subsequent options operate on the copy.

### Usage after migration

```go
// draw.go — raw values for rendering (zero-copy, no opts)
xVals, _ := ds.Float64("x")

// stat/stat.go — clean values for computation
vals, err := ds.Float64("x", dataset.Clean())

// user code — fill + abs
vals, _ := ds.Float64("temperature", dataset.FillNaN(0), dataset.Abs())
```

### Migration checklist

1. **Create** `dataset/accessors.go` with `Float64`, `Int64`, `Strings`, `Bools` methods and option types.
2. **Replace** `getFloat64Values()` in `util.go` → callers in `draw.go` use `ds.Float64(col)`.
3. **Replace** `collectFloat64Column()` in `stat/stat.go` → callers use `ds.Float64(col, dataset.Clean())`.
4. **Replace** `collectFloat64()` in `stat/stat.go` (takes AnyColumn) → keep as internal or add `AnyColumn.Float64()`.
5. **Test helpers** in `memory/join_test.go`, `stat/stat_test.go` remain as test-local (they take `*testing.T` and fatal — different concern).
6. **Tests**: Add `dataset/accessors_test.go` covering each option and combination.
7. **Verify**: `go build ./...`, `go test ./...`, `go vet ./...`.

### Files changed

| File | Change |
|------|--------|
| `dataset/accessors.go` | **[NEW]** Typed methods + option types |
| `dataset/accessors_test.go` | **[NEW]** Tests |
| `draw.go` | Replace `getFloat64Values(ds, col)` → `ds.Float64(col)` |
| `util.go` | Remove `getFloat64Values` |
| `stat/stat.go` | Replace `collectFloat64Column(ds, col)` → `ds.Float64(col, dataset.Clean())` |
