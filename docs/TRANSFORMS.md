# ggplot — Composable Transform Architecture

> **Status:** Partially implemented · v0.4 · 2026-05-20
> **Target:** v0.6 (one minor release out from current v0.5)
> **Supersedes:** `options-refactor.md` v0.1 and `options-refactor-v0.2.md` v0.2 — both obsolete; they proposed infrastructure that already exists.
>
> **Grounded.** This document is written against the verified codebase: the existing `Plot`/`Built` separation, the `stat.Transform` interface, the `geom.Layer` struct with `Pipeline` field, the `buildPanel` pipeline in `ggplot.go`, the `position.Pos` interface, and the `ColorScales map[string]*colormap.Scale` integration.
>
> **Implementation status (v0.4).** The following components from this proposal are implemented and verified:
> - `stat.Transform` interface and `RunPipeline` — `stat/transform.go`
> - `ChannelHint` type and all standard hints (Count, Proportion, Probability, Interval, Cumulative, Deviation)
> - `geom.Layer.Pipeline` field — composable transform chains on every layer
> - `buildPanel` pipeline execution — reads `Layer.Pipeline`, runs `RunPipeline`, materializes lazy stages
> - Hint-aware axis formatting — `collectPipelineHints`, `hintFormatter`, `applyHintFormatters` in `ggplot.go`
> - Introspection APIs — `Built.PipelineFor(panel, layer)` and `Built.Explain()`
> - Transforms: `BinX`, `Count`, `DensityX`, `SmoothXY`, `SummaryXY`, `BoxplotY`, `IdentityTransform`, `NormalizeY`/`NormalizeX`, `FilterX`/`FilterY`, `SortBy`, `ReverseRows`, `TopN`, `SelectRow`, `StackY`/`StackX`, `GroupX`/`GroupY`
> - Extended `Aggregator` interface: `StdDev`, `First`, `Last`, `Mode` across memory, arrow, and bigquery engines
> - Extended `GroupX`/`GroupY` reducer vocabulary: sum, mean, median, min, max, count, variance, deviation, first, last, mode
> - New geom constructors: `RectY`, `RectX`, `LineY`, `LineX`, `AreaY`, `AreaX`, `PointY`, `RibbonY`, `Difference`
> - Ribbon and difference drawers registered in `drawer.go`
> - Mode implementations improved: sort-based scan for float64/int64 (memory + arrow), direct Arrow array iteration for strings (no `[]string` materialization)
> - `Canvas.Close()` added to release GPU resources deterministically (eliminates wgpu BindGroup/Buffer finalizer warnings)

---

## 1. The architectural delta with Observable Plot

Plot's signature trick is one specific structural property:

> **Marks and transforms share an options shape.** A transform consumes `(data, options)` and returns `(data, options)`. A mark consumes `(data, options)` and produces a renderable. Because both speak the same input language, transforms chain arbitrarily before any mark, and any compatible mark can consume the output of any transform chain.

This is what enables `Plot.binX({y: "count"})` to feed `rectY` (histogram), `lineY` (frequency polygon), `areaY` (filled curve), or `dot` (1D heatmap). The bin transform doesn't know which mark will consume it.

The architectural delta with this codebase, verified from `ggplot.go:buildPanel`:

- `geom.Layer` carries one `StatName` field; `stat.Lookup(name)` resolves it; `Stat.Compute(ctx, ds, mapping, opts)` runs once and returns a transformed dataset
- The stat output goes straight to position adjustment and then to the renderer
- There is no way to insert a second transform between stat and render without modifying `buildPanel`
- The result: each `geom.Histogram`-style constructor is wedded to exactly one stat (`stat.Bin`). Adding "histogram of proportions" requires either a new stat that does both at once, or post-processing the dataset, or a new geom constructor

This is the gap. Everything else I'd been writing about — ValueSpec interfaces, Options as a tagged union, building a theme element system, building a color scheme catalog — was either misdirected or already shipped. The composable-transform model is the architectural delta that's actually missing.

## 2. What composability looks like in Go

Plot's transform model translates to Go as a function type that operates on whatever shape carries data plus context through the pipeline. In your codebase, the natural type to evolve is `geom.Layer` — it already carries `Data` lineage through `Mapping`, `Params`, and stat output schema. What it doesn't carry is **a chain**.

### 2.1 The Transform type

✅ **Implemented** in `stat/transform.go`.

```go
// Package stat

// Transform is the composable data-transform contract. Transforms
// chain because their input and output shapes are the same:
// data + mapping in, data + mapping out. The bin transform, the
// smooth transform, the normalize transform, the identity transform —
// all the same shape.
//
// Apply MUST NOT mutate in; it returns a new TransformResult.
type Transform interface {
    // Name returns a stable identifier for debugging, golden tests,
    // and pipeline introspection.
    Name() string

    // Apply runs the transform. Implementations MUST NOT mutate in.
    Apply(ctx context.Context, in TransformInput) (TransformResult, error)

    // OutputMapping describes how aesthetic channels are rewritten.
    // nil means the transform preserves the mapping (identity, filter, sort).
    // A non-nil map rewrites channels: {"y": "count"} means the y channel
    // now points at the "count" column the transform produced.
    OutputMapping() map[string]string

    // OutputSchema names the columns this transform produces.
    OutputSchema() []string

    // OutputHints declares semantic hints for output channels.
    // Axis/legend formatters use these: HintProportion → "%" tick formatting,
    // HintCount → integer ticks, HintInterval → bin-edge rendering.
    // nil for transforms that don't change channel semantics.
    OutputHints() map[string]ChannelHint
}

type TransformInput struct {
    Data    dataset.Dataset
    Mapping map[string]string
}

type TransformResult struct {
    Data    dataset.Dataset
    Mapping map[string]string
}

type ChannelHint string
const (
    HintNone        ChannelHint = ""
    HintCount       ChannelHint = "count"
    HintProportion  ChannelHint = "proportion"  // axis formats as %
    HintProbability ChannelHint = "probability" // axis clamps to [0,1]
    HintInterval    ChannelHint = "interval"    // x1/x2 bin endpoints
    HintCumulative  ChannelHint = "cumulative"
    HintDeviation   ChannelHint = "deviation"
)
```

**Design note.** The implemented type uses `TransformResult` (not `TransformOutput`) and carries only `Data` + `Mapping` (no `Params`). This is simpler than the original proposal — transforms don't need to thread geom parameters. `RunPipeline` handles inter-stage materialization of lazy datasets.

### 2.2 The Pipeline model — flat chaining, not nesting

The original proposal (v0.3) showed transforms nesting inside each other: `NormalizeY(BinX(...))`. The implemented model uses **flat pipeline chaining**: transforms are independent stages in an ordered `[]stat.Transform` slice. Each stage's output becomes the next stage's input.

```go
// Flat pipeline — each transform is a standalone stage.
// RunPipeline executes them in order: BinX → NormalizeY.
pipeline := []stat.Transform{stat.BinX(), stat.NormalizeY()}
```

This is cleaner than nesting because:
- Each transform has one job and no knowledge of its neighbors
- Pipeline ordering is explicit and readable left-to-right
- Adding/removing a stage doesn't require restructuring the whole expression
- `RunPipeline` handles inter-stage materialization centrally

### 2.3 The Layer type — Pipeline field

✅ **Implemented** in `geom/geom.go`.

```go
// Package geom

// Layer is the renderable mark spec. Pipeline carries the ordered
// chain of transforms applied during Build, before position adjustment.
type Layer struct {
    Geom     Type
    Position Pos
    Params   Params
    Mapping  map[string]string

    // Pipeline is the ordered chain of transforms applied during Build.
    // An empty/nil Pipeline means "data passes through unchanged"
    // (identity behavior).
    Pipeline []stat.Transform

    // StatName is the legacy single-stat path. When Pipeline is non-empty,
    // Pipeline takes precedence.
    StatName stat.Name

    setFlags OptFlag
    warnings []string
}
```

### 2.4 Geom constructors take pipeline slices

✅ **Implemented.** New constructors take `[]stat.Transform` as the first argument:

```go
// Package geom

// RectY creates a rectangle mark anchored at y with a transform pipeline.
// With BinX, this becomes a histogram. With Count, a bar chart.
func RectY(pipeline []stat.Transform, opts ...Opt) Layer {
    l := Layer{
        Geom:     TypeBar,
        Pipeline: pipeline,
        Position: Stack(),
        Params:   Params{Width: 0.8, Alpha: 0.85},
    }
    applyOpts(&l, opts)
    return l
}

// LineY creates a connected-line mark with a transform pipeline.
func LineY(pipeline []stat.Transform, opts ...Opt) Layer { ... }

// AreaY creates a filled-area mark with a transform pipeline.
func AreaY(pipeline []stat.Transform, opts ...Opt) Layer { ... }

// PointY creates a point mark with a transform pipeline.
func PointY(pipeline []stat.Transform, opts ...Opt) Layer { ... }

// RectX, LineX, AreaX — horizontal counterparts.
func RectX(pipeline []stat.Transform, opts ...Opt) Layer { ... }
func LineX(pipeline []stat.Transform, opts ...Opt) Layer { ... }
func AreaX(pipeline []stat.Transform, opts ...Opt) Layer { ... }

// RibbonY creates a filled band between ymin and ymax columns.
func RibbonY(pipeline []stat.Transform, opts ...Opt) Layer { ... }

// Difference fills the area between two series with positive/negative coloring.
func Difference(pipeline []stat.Transform, opts ...Opt) Layer { ... }
```

Backwards-compat constructors remain — they use `geom.Stat(transforms...)` internally to build the pipeline:

```go
// Histogram is preserved as sugar. Internally uses BinX in the pipeline.
func Histogram(opts ...Opt) Layer { ... }

// Bar is preserved as sugar. Uses Count in the pipeline.
func Bar(opts ...Opt) Layer { ... }

// Col is preserved — pre-computed values, no pipeline.
func Col(opts ...Opt) Layer { ... }
```

### 2.5 Composition examples

The point of all of this:

```go
// Frequency polygon — Plot's "lineY(binX(...))" pattern:
geom.LineY([]stat.Transform{stat.BinX(stat.WithBins(40))})

// Filled-area histogram:
geom.AreaY([]stat.Transform{stat.BinX(stat.WithBins(40))})

// 1D dot plot of bin midpoints:
geom.PointY([]stat.Transform{stat.BinX(stat.WithBins(40))})

// Histogram of proportions — BinX then NormalizeY as pipeline stages:
geom.RectY([]stat.Transform{stat.BinX(stat.WithBins(40)), stat.NormalizeY()})

// Top-10 categories — Count then TopN:
geom.RectY([]stat.Transform{stat.Count(), stat.TopN(10, "count")})

// Filter before binning — FilterY then BinX:
geom.RectY([]stat.Transform{
    stat.FilterY(dataset.Gt("x", 0.0)),
    stat.BinX(stat.WithBins(40)),
})

// Cumulative stacked area:
geom.Area(geom.Stat(stat.StackY()))

// Group deviation per sensor:
geom.Col(geom.Stat(stat.GroupX("deviation")))

// Select row with max temperature, overlay as point:
geom.Point(geom.Stat(stat.SelectRow(stat.SelectMax, "temp")))
```

Each composition is one expression. The sugar constructors (`Histogram`, `Bar`, `Col`, etc.) also accept pipeline transforms via `geom.Stat(...)`:

```go
// Sugar API — same result as above, uses geom.Stat to inject transforms:
geom.Histogram(geom.Stat(stat.BinX(), stat.NormalizeY()))
geom.Col(geom.Stat(stat.GroupX("mean")))
```

## 3. The Build pipeline change

✅ **Implemented** in `ggplot.go:buildPanel`.

The pipeline execution delegates to `stat.RunPipeline`:

```go
// stat/transform.go

// RunPipeline executes an ordered chain of transforms. If the pipeline
// is nil or empty, data passes through unchanged (identity).
//
// Between stages, if a transform produces a lazy (uncollected) Dataset,
// RunPipeline materializes it before passing to the next transform.
func RunPipeline(ctx context.Context, pipeline []Transform,
    data dataset.Dataset, mapping map[string]string,
) (dataset.Dataset, map[string]string, error) {
    in := TransformInput{Data: data, Mapping: mapping}

    for i, tf := range pipeline {
        out, err := tf.Apply(ctx, in)
        if err != nil {
            return dataset.Dataset{}, nil, fmt.Errorf("transform %q: %w", tf.Name(), err)
        }

        // Materialize between stages if the output is lazy and there
        // are more transforms to run.
        if out.Data.Table() == nil && i < len(pipeline)-1 {
            out.Data, err = out.Data.Collect(ctx)
            if err != nil {
                return dataset.Dataset{}, nil, fmt.Errorf("transform %q: collect: %w", tf.Name(), err)
            }
        }

        in = TransformInput(out)
    }

    return in.Data, in.Mapping, nil
}
```

In `buildPanel`, the call site is:

```go
// buildPanel in ggplot.go:
pipeline := layer.Geom.Pipeline
if len(pipeline) == 0 && layer.Geom.StatName != stat.Identity {
    // Legacy path: stat.Name → single-transform pipeline
    pipeline = stat.LookupPipeline(layer.Geom.StatName, statOpts)
}

grpDS, grpMerged, err = stat.RunPipeline(ctx, pipeline, grpDS, grpMerged)
```

After `RunPipeline`, hint-aware formatting kicks in:

```go
// collectPipelineHints merges OutputHints() from all transforms.
// applyHintFormatters wraps scales with custom formatters (count → integer,
// proportion → percentage, etc.), respecting user-set overrides.
hints := collectPipelineHints(pipeline)
applyHintFormatters(hints, xScale, yScale)
```

The change is local to one loop body in `buildPanel`. No new types in the build pipeline. No restructured BuiltLayer. No changes to `Built`/`Built.Draw`. No changes to `Plot.ScaleColor`/`ScaleFill`. No changes to `theme.Theme`, `colormap.Scale`, `dataset.Dataset`, `coord.Coord`, `facet.Facet`, or anywhere in `canvas/`.

Introspection is available:

```go
built.PipelineFor(0, 0) // → ["binX", "normalizeY"]
built.Explain()         // → human-readable plot structure summary
```

## 4. The Stat → Transform migration

Every existing stat (verified from `stat/stat.go`) has been migrated to a `Transform` factory. Worked example for the bin stat:

### 4.1 Previous implementation (stat.Stat interface)

```go
// stat/stat.go (old interface)
type binStat struct{}

func (binStat) Name() Name                       { return Bin }
func (binStat) RequiredAes() []string            { return []string{"x"} }
func (binStat) OutputSchema() []string           { return []string{"x", "count", "xmin", "xmax"} }
func (binStat) OutputMapping() map[string]string { return map[string]string{"x": "x", "y": "count"} }

func (binStat) Compute(_ context.Context, ds dataset.Dataset, mapping map[string]string, opts Options) (dataset.Dataset, error) {
    xCol := mapping["x"]
    if xCol == "" { return dataset.Dataset{}, fmt.Errorf(...) }
    vals, err := ds.Float64(xCol, dataset.Clean)
    // ...binning math...
    return newFloat64Dataset(ds, map[string][]float64{
        "x": centers, "count": counts,
    })
}
```

### 4.2 Current implementation (stat.Transform interface)

✅ **Implemented** in `stat/bin.go`.

```go
// stat/bin.go

// BinX returns a Transform that bins the x channel into evenly-spaced
// rectangles, producing x/x1/x2 (bin edges) and count (per-bin count).
// Output channel hint for the y channel is HintCount.
//
// Options:
//   WithBins(n)            — explicit bin count (overrides BinMethod)
//   WithBinMethod(method)  — "sturges" (default), "scott", "fd", "sqrt"
func BinX(opts ...BinOption) Transform {
    cfg := defaultBinConfig
    for _, o := range opts { o(&cfg) }
    return &binTransform{cfg: cfg, axis: "x"}
}

type binTransform struct {
    cfg  binConfig
    axis string  // "x"
}

func (b *binTransform) Name() string                       { return "binX" }
func (b *binTransform) OutputSchema() []string             { return []string{"x", "x1", "x2", "count"} }
func (b *binTransform) OutputMapping() map[string]string   {
    return map[string]string{"x": "x", "y": "count", "x1": "x1", "x2": "x2"}
}
func (b *binTransform) OutputHints() map[string]ChannelHint {
    return map[string]ChannelHint{"y": HintCount, "x1": HintInterval, "x2": HintInterval}
}

func (b *binTransform) Apply(ctx context.Context, in TransformInput) (TransformResult, error) {
    xCol := in.Mapping[b.axis]
    if xCol == "" {
        return TransformResult{}, fmt.Errorf("binX: missing %q aesthetic", b.axis)
    }
    // ... engine-native StatKernel.Histogram ...
    // ... builds output dataset with x, x1, x2, count columns ...
    outMapping := make(map[string]string, len(in.Mapping)+4)
    maps.Copy(outMapping, in.Mapping)
    maps.Copy(outMapping, b.OutputMapping())

    return TransformResult{Data: outData, Mapping: outMapping}, nil
}
```

### 4.3 The functional-option pattern for transforms

✅ **Implemented.** Each transform is its own type with its own options:

```go
// stat/bin.go — implemented
type BinOption func(*binConfig)
func WithBins(n int) BinOption                { return func(c *binConfig) { c.Bins = n } }
func WithBinMethod(method string) BinOption   { return func(c *binConfig) { c.Method = method } }

// stat/normalize.go — implemented
type NormalizeOption func(*normalizeConfig)
func WithTotal(t float64) NormalizeOption     { return func(c *normalizeConfig) { c.total = t } }
```

This is more idiomatic Go than the central-struct-with-magic-field-names pattern. Each transform's options are visible at the call site and validated locally.

## 5. Composition unlocks

The transforms that unlock substantial chart variety. Items marked ✅ are implemented; ⏳ are pending.

### 5.1 NormalizeY / NormalizeX — ✅ Implemented

```go
// NormalizeY rescales the y channel so values sum to the given total
// (default 1.0). Applied as a pipeline stage after BinX or Count to
// convert frequencies into proportions.
//
// Uses Aggregator.Sum + MathKernel.MulScalar — stays lazy.
// Output hint: y → HintProportion.
func NormalizeY(opts ...NormalizeOption) Transform { ... }
func NormalizeX(opts ...NormalizeOption) Transform { ... }

// Usage — flat pipeline, BinX then NormalizeY:
geom.RectY([]stat.Transform{stat.BinX(stat.WithBins(40)), stat.NormalizeY()})
// → histogram of proportions, y-axis labeled as percentages

geom.Histogram(geom.Stat(stat.BinX(stat.WithBins(20)), stat.NormalizeY(stat.WithTotal(100))))
// → percentage histogram via sugar API
```

The hint propagation is the crucial detail. `NormalizeY` declares `OutputHints: {"y": HintProportion}`. `applyHintFormatters` reads the hint and formats ticks as `0%`/`25%`/`50%`/`100%` instead of `0.00`/`0.25`/...

### 5.2 StackY / StackX as transforms — ✅ Implemented

`StackY` and `StackX` are pipeline transforms that accumulate values cumulatively:

```go
func StackY() Transform { ... }
func StackX() Transform { ... }

// Usage — cumulative area:
geom.Area(geom.Stat(stat.StackY()))

// Output hints: y → HintCumulative
```

This isn't a deprecation of `position.Stack` — `position.Stack` stays as a position-level adjustment (it handles the X-axis dodging concern that's orthogonal). `stat.StackY` handles Y-axis accumulation. Both coexist: the position adjustment is "where to put bars side-by-side at the same X", the transform is "how to accumulate Y values across groups."

### 5.3 Filter / Sort / Select / TopN — ✅ Implemented

Data-shaping transforms that compose as pipeline stages:

```go
// FilterY / FilterX — engine-native predicate filtering (stays lazy).
func FilterY(masker dataset.Masker) Transform { ... }
func FilterX(masker dataset.Masker) Transform { ... }

// SortBy — sort rows by column, ascending by default.
func SortBy(column string, opts ...SortOption) Transform { ... }

// ReverseRows — reverse row order.
func ReverseRows() Transform { ... }

// TopN — keep top N rows by column. Default descending (largest first).
func TopN(n int, column string, opts ...SortOption) Transform { ... }

// SelectRow — keep a single row by mode (first, last, min, max).
func SelectRow(mode SelectMode, column string) Transform { ... }

// Standard select modes:
const (
    SelectFirst SelectMode = "first"
    SelectLast  SelectMode = "last"
    SelectMin   SelectMode = "min"
    SelectMax   SelectMode = "max"
)

// Usage — top-10 categories:
geom.RectY([]stat.Transform{stat.Count(), stat.TopN(10, "count")})

// Usage — highlight min/max in a scatter:
geom.Point(geom.Stat(stat.SelectRow(stat.SelectMax, "temp")), geom.WithColor("#E74C3C"))

// Usage — filtered scatter:
geom.Point(geom.Stat(stat.FilterY(dataset.Gt("y", 30.0))))
```

### 5.4 GroupX / GroupY with reducer vocabulary — ✅ Implemented (partial)

Groups data by one axis and applies a named reducer to the other:

```go
// GroupX groups by x, reduces y. GroupY is symmetric.
func GroupX(reducer string) Transform { ... }
func GroupY(reducer string) Transform { ... }

// Usage:
geom.Col(geom.Stat(stat.GroupX("mean")))      // mean per group
geom.Col(geom.Stat(stat.GroupX("deviation")))  // std dev per group
geom.Col(geom.Stat(stat.GroupX("first")))      // first value per group
```

**Reducer vocabulary — implemented (engine-native via `Aggregator`):**

| Reducer | `AggFunc` | Status |
|---------|-----------|--------|
| `"sum"` | `AggSum` | ✅ |
| `"mean"` | `AggMean` | ✅ |
| `"median"` | `AggMedian` | ✅ |
| `"min"` | `AggMin` | ✅ |
| `"max"` | `AggMax` | ✅ |
| `"count"` | `AggCount` | ✅ |
| `"variance"` | `AggVariance` | ✅ |
| `"deviation"` / `"stddev"` | `AggStdDev` | ✅ |
| `"first"` | `AggFirst` | ✅ |
| `"last"` | `AggLast` | ✅ |
| `"mode"` | `AggMode` | ✅ |

**Reducer vocabulary — not yet implemented:**

| Reducer | Notes |
|---------|-------|
| `"proportion"` | Requires normalized count per group. Needs design. |
| `"proportion-facet"` | Proportion within facet panel. Needs design. |
| `"p10"`, `"p50"`, `"p90"` | Percentile reducers. `AggPercentile` enum exists but dispatch is not wired. |

**Not yet implemented:**

| Item | Notes |
|------|-------|
| `Group(xReducer, yReducer)` | Dual-axis grouping. Only `GroupX`/`GroupY` exist. |

## 6. What stays unchanged

To bound the scope and reassure on stability:

- **`Plot` and its builder methods.** `New`, `Layer`, `Aes`, `ScaleX`, `ScaleY`, `ScaleColor`, `ScaleFill`, `ScaleColorManual`, `ScaleColorContinuous`, `FacetWrap`, `FacetGrid`, `Coord`, `CoordFlip`, `Theme`, `XLim`, `YLim`, `Labs`, `LegendPosition`, `Save`, `WriteTo`. Every public method on `*Plot` keeps the same signature.
- **`Built` and its methods.** `LayerData`, `NumPanels`, `NumLayers`, `Theme`, `Labels`, `PanelLayout`, `Save`, `WriteTo`, `DrawCanvas`, `Draw`. Untouched. New: `PipelineFor` and `Explain`.
- **`PlotSpec`** in `spec.go`. The `Layers []LayerSpec` field type doesn't change (LayerSpec still wraps a `geom.Layer`).
- **`colormap` package.** Cmap, Norm, Scale, Resolve, NewContinuous/Discrete/Manual — all untouched.
- **`theme` package.** Element, Theme, parentOf, resolveText/Line/Rect, all 25+ named themes, baseTheme. Untouched.
- **`scale` package.** Scale, ConfiguredScale, DiscreteScale, BoundsSetter, Expander, all options, Resolve, ticks algorithm. Untouched.
- **`coord`, `facet`, `aes`, `canvas`, `fonts`, `dataset/*`, `output`.** All untouched.

The blast radius is `stat/*.go`, `geom/geom.go` (Pipeline field on Layer, new constructors), `drawer.go` (ribbon/difference drawers), `canvas/canvas.go` + `canvas/gg.go` (added `Close()` to the `Canvas` interface; `GGCanvas.Close()` releases GPU resources), and one section of `buildPanel` in `ggplot.go`. Everything else is exactly where it is today.

## 7. Migration plan

Three minor releases. The deprecation policy is the same one CHANGELOG already follows for `coord.IsFlipped` and `Stat`-style parameters.

### 7.1 v0.6 — Land the new model alongside the old

| # | Package | Action | Status |
|---|---------|--------|--------|
| 1 | `stat/` | `Transform` interface in `stat/transform.go`. `RunPipeline` executor. | ✅ Done |
| 2 | `stat/` | Each existing stat migrated to Transform factory: `BinX`, `Count`, `DensityX`, `SmoothXY`, `SummaryXY`, `BoxplotY`. | ✅ Done |
| 3 | `stat/` | New transforms: `NormalizeY`, `NormalizeX`, `StackY`, `StackX`, `FilterX`, `FilterY`, `SortBy`, `ReverseRows`, `TopN`, `SelectRow`, `GroupX`, `GroupY`. | ✅ Done |
| 4 | `geom/` | `Pipeline []stat.Transform` field on `Layer`. New constructors: `RectY`, `RectX`, `LineY`, `LineX`, `AreaY`, `AreaX`, `PointY`, `RibbonY`, `Difference`. | ✅ Done |
| 5 | `geom/` | Existing constructors (`Point`, `Line`, `Bar`, `Histogram`, `BoxPlot`, `Smooth`, `Density`, `Area`, `Col`) preserved. Accept pipeline via `geom.Stat(...)`. | ✅ Done |
| 6 | `ggplot.go` | `buildPanel` reads `Layer.Pipeline`, runs `RunPipeline`. Falls back to legacy `StatName` path. | ✅ Done |
| 7 | `ggplot.go` | `Built.PipelineFor(panel, layer)` and `Built.Explain()` introspection. | ✅ Done |
| 8 | `ggplot.go` | Hint-aware axis formatting via `collectPipelineHints` + `applyHintFormatters`. | ✅ Done |
| 9 | `dataset/` | Extended `Aggregator` interface: `StdDev`, `First`, `Last`, `Mode` across all engines. | ✅ Done |
| 9b | `dataset/` | Mode: sort-based for float64/int64 (memory + arrow), direct Arrow array iteration for strings. | ✅ Done |
| 10 | `drawer.go` | Ribbon and difference drawers. | ✅ Done |
| 10b | `canvas/` | `Canvas.Close()` for GPU resource cleanup; `GGCanvas.Close()` delegates to `gg.Context.Close()`. | ✅ Done |
| 11 | tests | Composition tests in `stat/composition_test.go`. | ✅ Done |
| 12 | docs | This document, updated examples in `examples/phase5_transforms/` (12 examples: histogram, percentage, filter, group, topN, sort, stack, group-deviation, group-first, select-row, ribbon-band, difference-fill). | ✅ Done |

After v0.6 ships:
- Old API: works unchanged. `geom.Histogram(geom.WithBins(40))` produces a Layer with the new Pipeline machinery underneath.
- New API: available. `geom.RectY([]stat.Transform{stat.BinX(stat.WithBins(40))})` works.
- Both APIs interoperate. `geom.Histogram(...)` and `geom.RectY(...)` produce structurally-equivalent Layers.

### 7.2 v0.7 — Soft-warn on deprecated paths

| # | Action | Status |
|---|--------|--------|
| 1 | Calling `stat.Lookup(...)` (the old registry) emits a typed `Diagnostic{Code: "GGD002", Level: Warning}` once per process. Surfaces in `Built.Diagnostics`. | ⏳ Pending |
| 2 | All examples in `examples/` migrate to the new API. | ⏳ Pending |
| 3 | `docs/MIGRATION.md` published. | ⏳ Pending |

### 7.3 v0.8 — Remove deprecated shims

Old `stat.Stat` interface and its concrete types removed. `Layer.StatName` field removed. `stat.Lookup` removed. CHANGELOG documents the removals with the equivalent new-API call for each removed symbol.

### 7.4 Compatibility CI

Throughout v0.6–v0.8, the existing `ggplot_golden_test.go` and `ggplot_test.go` run against the new internals with the old API. Visual drift fails the build. Non-negotiable, same as the CHANGELOG implies for the current orientation-aware geoms refactor.

## 8. Risks and how to handle them

### 8.1 The interface explosion problem

Concern: every transform option becomes its own type (BinOption, NormalizeOption, ...). Today you have one `stat.Options` struct with all fields.

Response: that's the point. Today, `stat.Options.Bandwidth` is meaningful only for `stat.Density`; setting it on `stat.Bin` is silently ignored. The current design pretends one config covers all stats. The new design respects the type system. The cost is ~7 option types instead of 1 struct; the benefit is compile-time validation that you don't set `WithBandwidth` on `BinX`.

**Status:** ✅ Resolved. Each transform has its own option type.

### 8.2 Performance: the per-transform allocations

Concern: each Transform.Apply returns a new TransformResult, and chaining N transforms allocates N intermediate datasets.

Response: each step today already allocates a new dataset (look at `binStat.Compute` returning `newFloat64Dataset(...)`). The composition model adds a thin TransformInput/Result wrapper struct (2 fields, all already paid for) per step — negligible overhead. `RunPipeline` handles inter-stage materialization of lazy datasets, and the last stage stays lazy — the caller handles final `Collect`.

**Status:** ✅ Resolved. `RunPipeline` materializes lazily between stages.

### 8.3 The position/transform split

Concern: putting StackY in `stat/` parallels `position.Stack`. Two ways to stack creates confusion.

Response: they handle different concerns. `position.Stack` adjusts X positions for side-by-side bars (the "dodge axis"). `stat.StackY` accumulates Y values for stacked rectangles (the "height axis"). Plot keeps these conceptually unified inside `stack`, but Plot's mark API is also implicit about which axis is being adjusted. Your existing position package handles X-axis adjustment well; the new `stat.StackY` handles Y-axis accumulation. Both can be present in a pipeline without conflict.

For users, document the rule once: "use `position.Stack` when bars should sit side-by-side on the X axis (grouped categorical); use `stat.StackY` when bar heights should accumulate (stacked-on-Y)." Two different visual outcomes; two different tools.

**Status:** ✅ Resolved. Both coexist.

### 8.4 Channel-hint vocabulary lock-in

Concern: adding HintCount/HintProportion/HintInterval/HintProbability/HintCumulative/HintDeviation as a closed enum locks future stats out of declaring new hints.

Response: ✅ Resolved — `ChannelHint` is `type ChannelHint string` (open type). Known hints get special formatting; unknown hints get default formatting. Third-party transforms can declare arbitrary hints.

### 8.5 Discoverability vs implicitness

Concern: today `geom.Histogram(geom.WithBins(40))` is one symbol. New API requires the user to know both `geom.RectY` and `stat.BinX`. Cognitive load increase.

Response: this is real. The mitigation is keeping the sugar constructors as first-class API forever, not deprecating them. `geom.Histogram` stays. Users who don't need composition never reach for `RectY` + `BinX`. Users who do need composition discover it via doc examples. The progressive-disclosure axis is preserved.

## 9. Test strategy

### 9.1 Output-schema golden tests

✅ **Implemented** in `stat/stat_test.go`.

```go
func TestBinX_GoldenSchema(t *testing.T) {
    tf := stat.BinX(stat.WithBins(20))
    require.Equal(t, []string{"x", "x1", "x2", "count"}, tf.OutputSchema())
    require.Equal(t, map[string]string{"x": "x", "y": "count", "x1": "x1", "x2": "x2"}, tf.OutputMapping())
    require.Equal(t, map[string]ChannelHint{"y": HintCount, "x1": HintInterval, "x2": HintInterval}, tf.OutputHints())
}
```

### 9.2 Composition correctness

✅ **Implemented** in `stat/composition_test.go`.

```go
func TestNormalizeY_AfterBinX(t *testing.T) {
    ds := testFixture(t)
    pipeline := []stat.Transform{stat.BinX(stat.WithBins(10)), stat.NormalizeY()}

    outDS, outMapping, err := stat.RunPipeline(ctx, pipeline, ds, map[string]string{"x": "weight"})
    require.NoError(t, err)

    counts, _ := outDS.Float64(outMapping["y"])
    sum := 0.0
    for _, c := range counts { sum += c }
    require.InDelta(t, 1.0, sum, 1e-10, "normalize should produce proportions summing to 1")
}
```

### 9.3 Layer-data equivalence under old vs new API

For backward compat, generate the same plot two ways and check that `Built.LayerData(panel, layer)` produces structurally identical data:

```go
func TestHistogramAPIEquivalence(t *testing.T) {
    ds := testFixture(t)
    oldP := ggplot.New(ds, aes.X("weight")).Layer(geom.Histogram(geom.WithBins(40)))
    newP := ggplot.New(ds, aes.X("weight")).Layer(
        geom.RectY([]stat.Transform{stat.BinX(stat.WithBins(40))}),
    )

    oldBuilt, _ := oldP.Build(ctx)
    newBuilt, _ := newP.Build(ctx)

    assertSameLayerData(t, oldBuilt.LayerData(0, 0), newBuilt.LayerData(0, 0))
}
```

If this test passes for every existing geom×stat combination, the migration is safe. This is the test that gates the v0.7 deprecation warnings — if any combination fails equivalence, the deprecation is held back.

### 9.4 Visual golden equivalence

The existing `ggplot_golden_test.go` is the ultimate safety net: same plot specs, byte-identical PNG output between old and new pipelines. If those tests pass unchanged through v0.6, the refactor is invisibly safe.

## 10. The work plan

| Phase | Work | Status |
|-------|------|--------|
| W1 | Add `Transform` interface in `stat/transform.go`. `RunPipeline`. Update `buildPanel` to consume `Layer.Pipeline`. All existing tests pass unchanged. | ✅ Done |
| W2 | Migrate `stat.Bin` to `stat.BinX` (new transform). Side-by-side: old binStat still works. Pipeline equivalence tests. | ✅ Done |
| W3 | Migrate `stat.Count`, `stat.Density`, `stat.Smooth`, `stat.Summary`, `stat.Boxplot` to transform factories. Equivalence tests for each. | ✅ Done |
| W4 | New transforms: `NormalizeY`/`NormalizeX`, `Filter`, `Sort`, `Reverse`, `TopN`, `SelectRow`. Schema/hint tests. | ✅ Done |
| W5 | New transforms: `StackY`/`StackX`, `GroupX`/`GroupY` with extended reducer vocabulary. Composition tests for stack-after-bin. | ✅ Done |
| W6 | New geom constructors: `RectY`/`RectX`, `LineY`/`LineX`, `AreaY`/`AreaX`, `PointY`, `RibbonY`, `Difference`. | ✅ Done |
| W7 | Backward-compat shims: verify `Histogram`, `Bar`, `Col`, `Smooth`, `Density`, `Area` work with pipelines via `geom.Stat`. Visual golden tests pass unchanged. | ⏳ Pending |
| W8 | Documentation: this doc (TRANSFORMS.md), composition recipe cookbook, migration guide. Update README with composition examples. Cut v0.6. | ✅ Done |

## 11. Why this is the right v0.6 move

Three reasons:

**1. It's the only architectural delta with both reference designs.** ggplot2 doesn't have stat composition. Plot does. Adding it puts you between them — taking ggplot2's terminology and structure but matching Plot's compositional power. The roadmap doc's "this is not a port" claim becomes load-bearing.

**2. The infrastructure is already in place.** The Build/Render separation, the Layer struct with extension fields, the `stat.Stat.OutputMapping` contract, the system PANEL/group columns, the typed Names for stats/geoms/positions — every prerequisite for this refactor is already shipped. This is not building infrastructure; it's connecting infrastructure that exists into a composable shape.

**3. It unlocks chart variety with minimal new code.** Section 2.5's compositions are new chart types unlocked from existing math. With ~12 transforms × ~8 compatible geoms = ~100 chart shapes available from one infrastructure change. Each new transform after that adds another row to the matrix.

---

## 12. Remaining work

Items from the original proposal that are not yet implemented:

### Transforms

| Item | Notes |
|------|-------|
| `BinY()` — y-axis binning | Symmetric counterpart of `BinX`. Needed by `RectX` for horizontal histograms. |
| `DensityY()` — y-axis density | Symmetric counterpart of `DensityX`. Needed by `AreaX`. |
| `WithCumulative(dir)` on BinX | Cumulative histogram option (+1 forward, −1 reverse). |
| `WithSE(bool)` on SmoothXY | Standard error bands for smooth. Would output ymin/ymax for `RibbonY`. |
| `Group(xReducer, yReducer)` | Dual-axis grouping. Current API only has `GroupX`/`GroupY`. |
| Percentile reducers (`"p10"`, `"p50"`, `"p90"`) | `AggPercentile` enum exists in `dataset/frame.go` but dispatch is not wired. |
| `"proportion"` / `"proportion-facet"` reducers | Normalized count within group/facet. Needs design. |

### Infrastructure

| Item | Notes |
|------|-------|
| `Built.Diagnostics` | Typed warnings for deprecated paths (§7.2). |
| `TypeRect` replacing `TypeBar`/`TypeHistogram` | `RectY` currently uses `TypeBar` internally. Not urgent. |
| Backward-compat shim verification (W7) | Verify all sugar constructors produce structurally-equivalent Layers. |
| `docs/MIGRATION.md` | Migration guide for v0.7 deprecation. |

---

**Resolved design decisions:**

1. **Package location:** `stat.Transform` in `stat/` — ✅ Implemented as designed.
2. **Composition model:** Flat pipeline (`[]stat.Transform`) instead of Russian-doll nesting. Cleaner, more Go-idiomatic.
3. **StackY vs position.Stack split:** Both coexist. `stat.StackY`/`stat.StackX` handle within-group cumulative stacking; `position.Stack` handles across-group bar positioning.
4. **Reducer vocabulary:** Extended beyond Plot — includes: sum, mean, median, min, max, count, variance, deviation/stddev, first, last, mode. All engine-native via the `Aggregator` interface.
5. **No-transform case:** Pipeline field accepts `[]stat.Transform`, so the no-transform case is an empty/nil slice.
6. **Channel hints:** Open `string` type — extensible without package changes.
