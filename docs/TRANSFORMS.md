# ggplot — Composable Transform Architecture

> **Status:** Design proposal · v0.3 · 2026-05-14
> **Target:** v0.6 (one minor release out from current v0.5)
> **Supersedes:** `options-refactor.md` v0.1 and `options-refactor-v0.2.md` v0.2 — both obsolete; they proposed infrastructure that already exists.
>
> **Grounded.** This document is written against the verified codebase in this conversation: the existing `Plot`/`Built` separation, the `stat.Stat` interface, the `geom.Layer` struct, the `buildPanel` pipeline in `ggplot.go`, the `position.Pos` interface, and the `ColorScales map[string]*colormap.Scale` integration. Nothing in this doc invents an API that doesn't acknowledge what's already shipped.

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

```go
// Package stat (or a new package: pipeline)

// Transform is the contract for any function that operates on a layer's
// data, mapping, and parameters and produces a new (data, mapping, params)
// triple. Transforms compose because their input and output shape are the
// same.
//
// The bin transform, the smooth transform, the normalize transform, the
// stack transform, the dodge transform, the filter transform, the sort
// transform, the identity transform — all the same shape. They differ
// only in what they do to the data and how they rewrite the mapping.
//
// Replaces the single-purpose Stat.Compute path. The existing Stat
// interface is preserved one minor release for compatibility (§5.2).
type Transform interface {
    // Name returns a stable identifier used for Mark.Pipeline.Names(),
    // golden tests, and Explain() output.
    Name() string

    // Apply runs the transform. Implementations MUST NOT mutate in;
    // they return a new TransformResult.
    Apply(ctx context.Context, in TransformInput) (TransformOutput, error)

    // OutputMapping describes how aesthetic channels are rewritten by
    // this transform. nil means the transform preserves the mapping
    // (filter, sort, identity). A non-nil map indicates rewriting:
    // {"y": "count"} means the y channel now points at the "count"
    // column the transform produced. This is exactly the contract the
    // existing stat.Stat.OutputMapping() defines; the semantics carry over.
    OutputMapping() map[string]string

    // OutputSchema names the columns this transform produces. Used by
    // Built.LayerData() for introspection and by golden tests.
    OutputSchema() []string

    // OutputHints declares the semantic hint for each output channel.
    // Used by axis/legend formatters: HintProportion → "%" tick formatting,
    // HintCount → integer ticks, HintInterval → bin-edge rendering.
    // nil for transforms that don't change channel semantics.
    OutputHints() map[string]ChannelHint
}

type TransformInput struct {
    Data    dataset.Dataset
    Mapping ggplot.AesMap      // exists today in ggplot/spec.go
    Params  geom.Params        // exists today in geom/geom.go
}

type TransformOutput struct {
    Data    dataset.Dataset
    Mapping ggplot.AesMap
    Params  geom.Params
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

The shape is deliberately close to the existing `stat.Stat` interface. Three of the four methods (`Name`, `OutputMapping`, `OutputSchema`) already exist on `stat.Stat`. The new method is `OutputHints`. The signature of `Apply` differs from `Compute` in that it carries `Mapping` and `Params` through, not just the dataset — but the dataset transform itself is identical.

This is the smallest possible interface that lets you compose. Anything larger is over-engineered for the use case.

### 2.2 The Mark type — geom.Layer extended

The existing `geom.Layer` becomes a Mark by adding a Pipeline slice:

```go
// Package geom

// Layer is the renderable mark spec. New field: Pipeline.
type Layer struct {
    Geom     Type
    Position position.Pos
    Params   Params
    Mapping  map[string]string

    // NEW: ordered chain of transforms applied during Build, before
    // position adjustment. An empty Pipeline means "data passes through
    // unchanged" (matches the current Identity stat).
    //
    // The pipeline is what makes this Layer a composable "Mark" in
    // Observable Plot's sense: stat.BinX(stat.WithBins(40)) returns
    // a transform that, when this Layer is built, runs against the
    // panel's data.
    Pipeline []stat.Transform

    // DEPRECATED: kept for compatibility with v0.5 stat.Name lookup.
    // In v0.6, Pipeline is the authoritative path; StatName populates
    // Pipeline at constructor time via stat.Lookup(StatName).AsTransform().
    // In v0.7, this field is removed.
    StatName stat.Name

    setFlags OptFlag
    warnings []string
}
```

The renaming "Layer → Mark" is *not* proposed. The type stays `geom.Layer` because that's the name throughout the codebase and renaming would churn every file. The semantic content — "this is a renderable spec that carries a pipeline" — is what changes, not the name.

### 2.3 Geoms become functions over transforms

Today, `geom.Histogram(opts...)` returns a Layer with `StatName: stat.Bin`. Under the new model, geom constructors take an optional transform argument:

```go
// Package geom

// RectY creates a rectangle mark anchored at y. With no transform argument,
// it renders pre-computed x1/x2/y rectangles (ggplot2's geom_rect). With a
// transform argument, it renders the output of that transform — so RectY(BinX(...))
// is a histogram, RectY(StackY(BinX(...))) is a stacked histogram, etc.
//
// This replaces the current geom.Bar / geom.Histogram / geom.Col triplet,
// which exist today only because each binds a different default stat.
func RectY(t stat.Transform, opts ...Opt) Layer {
    l := Layer{
        Geom:     TypeRect,  // NEW type, replaces TypeBar/TypeHistogram
        Position: position.Identity(),
        Pipeline: nonNilPipeline(t),
        Params:   Params{Width: 0.8, Alpha: 0.85},
    }
    applyOpts(&l, opts)
    return l
}

// LineY creates a connected-line mark consuming x and y channels.
// With BinX, this becomes a frequency polygon.
func LineY(t stat.Transform, opts ...Opt) Layer {
    l := Layer{
        Geom:     TypeLine,
        Position: position.Identity(),
        Pipeline: nonNilPipeline(t),
        Params:   Params{LineWidth: 2, Alpha: 1.0},
    }
    applyOpts(&l, opts)
    return l
}

// AreaY creates a filled-area mark consuming x and y channels.
// With BinX, this becomes a filled histogram outline. With Density, a KDE area.
func AreaY(t stat.Transform, opts ...Opt) Layer { ... }

// PointY consumes x and y. With BinX, this becomes a 1D dot plot.
func PointY(t stat.Transform, opts ...Opt) Layer { ... }

// nonNilPipeline returns [t] if t != nil, else nil. The identity case
// is "no transform" — the data flows through unchanged.
func nonNilPipeline(t stat.Transform) []stat.Transform {
    if t == nil {
        return nil
    }
    return []stat.Transform{t}
}
```

Backwards-compat constructors stay as one-line shims:

```go
// Histogram is preserved as v0.4-compatible sugar: equivalent to RectY(BinX(...)).
func Histogram(opts ...Opt) Layer {
    bins := defaultBins
    for _, o := range opts {
        // intercept WithBins to extract the count
        ...
    }
    return RectY(stat.BinX(stat.WithBins(bins)), opts...)
}

// Bar is preserved: equivalent to RectY(Count(...)) with stack position.
func Bar(opts ...Opt) Layer {
    return RectY(stat.Count(), WithPosition(position.Stack()))
}

// Col is preserved: equivalent to RectY(nil) — pre-computed values, no stat.
func Col(opts ...Opt) Layer {
    return RectY(nil, opts...)
}
```

The user's hands don't move. The internals are now composable.

### 2.4 Composition examples

The point of all of this:

```go
// Frequency polygon — Plot's "lineY(binX(...))" pattern, impossible today:
geom.LineY(stat.BinX(stat.WithBins(40)))

// Filled-area histogram — impossible today:
geom.AreaY(stat.BinX(stat.WithBins(40)))

// 1D dot plot of bin midpoints — impossible today:
geom.PointY(stat.BinX(stat.WithBins(40)))

// Histogram of proportions — today requires writing a custom stat:
geom.RectY(stat.NormalizeY("sum", stat.BinX(stat.WithBins(40))))

// Smooth + ribbon for standard error — today requires two layers
// with custom stats; under the new model:
geom.RibbonY(stat.Smooth(stat.WithMethod("loess"), stat.WithSE(true)))
geom.LineY(stat.Smooth(stat.WithMethod("loess"))) // overlay

// Filter before binning — impossible today without preprocessing:
geom.RectY(stat.BinX(
    stat.WithBins(40),
    stat.WithFilter(func(row int, t dataset.Table) bool {
        return /* predicate */
    }),
))

// Top-10 categories — impossible today:
geom.RectY(stat.Select(stat.TopN(10, "count"), stat.Count()))
```

Each line was previously a custom-stat or custom-geom task. Under the new model, each is one composition. That's the architectural payoff.

## 3. The Build pipeline change

This is the smallest change in the whole proposal. Current `buildPanel` (read from `ggplot.go`):

```go
// CURRENT:
for _, layer := range p.spec.Layers {
    // ...mapping merge, group split, color scale training...

    if s.Name() != stat.Identity {
        transformed, err := s.Compute(ctx, grpDS, statMapping, opts)
        // ...error handling, mapping rewrite via updateMappingForStat...
        grpDS = transformed
    }

    // ...bake group color, inject group column, append to resolved...
}
// ...position adjustment via applyPositionAdjust...
```

New `buildPanel` body for the stat phase:

```go
// NEW:
for _, layer := range p.spec.Layers {
    // ...mapping merge, group split, color scale training (unchanged)...

    // Resolve pipeline: prefer Layer.Pipeline; fall back to stat.Lookup(StatName)
    // for v0.5 layers that haven't migrated.
    pipeline := layer.Geom.Pipeline
    if len(pipeline) == 0 && layer.Geom.StatName != stat.Identity {
        s, _ := stat.Lookup(layer.Geom.StatName)
        pipeline = []stat.Transform{stat.AsTransform(s, statOpts(layer))}
    }

    // Run the pipeline.
    spec := stat.TransformInput{Data: grpDS, Mapping: grpMerged, Params: layer.Geom.Params}
    for _, tf := range pipeline {
        out, err := tf.Apply(ctx, spec)
        if err != nil {
            return BuiltPanel{}, fmt.Errorf("ggplot: transform %q failed: %w", tf.Name(), err)
        }
        spec = stat.TransformInput(out)
    }

    grpDS = spec.Data
    grpMerged = spec.Mapping
    layer.Geom.Params = spec.Params

    // ...rest unchanged: bake group color, inject group column, append to resolved...
}
// ...position adjustment unchanged...
```

The change is local to one loop body in `buildPanel`. No new types in the build pipeline. No restructured BuiltLayer. No changes to `Built`/`Built.Draw`. No changes to `Plot.ScaleColor`/`ScaleFill`. No changes to `theme.Theme`, `colormap.Scale`, `dataset.Dataset`, `coord.Coord`, `facet.Facet`, or anywhere in `canvas/`.

The introspection contract gets stronger: each transform's `Name()` and `OutputSchema()` are queryable via `Layer.Pipeline`, so `Built.LayerData(panel, layer)` can be paired with a future `Built.PipelineFor(panel, layer) []string` for golden-test-friendly serialization.

## 4. The Stat → Transform migration

Every existing stat (verified from `stat/stat.go`) becomes a `Transform` factory with near-identical body. Worked example for the bin stat:

### 4.1 Current implementation

```go
// stat/stat.go today
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

### 4.2 New shape

```go
// stat/bin.go — new file under the new model

// BinX returns a Transform that bins the x channel into evenly-spaced
// rectangles, producing x/x1/x2 (bin edges) and count (per-bin count).
// Output channel hint for the y channel is HintCount.
//
// Options:
//   WithBins(n)            — explicit bin count (overrides BinMethod)
//   WithBinMethod(method)  — "sturges" (default), "scott", "fd", "sqrt"
//   WithFilter(predicate)  — row predicate applied before binning
//   WithCumulative(dir)    — +1 cumulative, -1 reverse cumulative, 0 default
func BinX(opts ...BinOption) Transform {
    cfg := defaultBinConfig
    for _, o := range opts { o(&cfg) }
    return &binTransform{cfg: cfg, axis: "x"}
}

// BinY is symmetric: binning on the y channel.
func BinY(opts ...BinOption) Transform { ... }

type binTransform struct {
    cfg  binConfig
    axis string  // "x" or "y"
}

func (b *binTransform) Name() string                       { return "binX" /* or binY */ }
func (b *binTransform) OutputSchema() []string             { return []string{"x", "x1", "x2", "count"} }
func (b *binTransform) OutputMapping() map[string]string   {
    return map[string]string{"x": "x", "y": "count", "x1": "x1", "x2": "x2"}
}
func (b *binTransform) OutputHints() map[string]ChannelHint {
    return map[string]ChannelHint{"y": HintCount, "x1": HintInterval, "x2": HintInterval}
}

func (b *binTransform) Apply(ctx context.Context, in TransformInput) (TransformOutput, error) {
    // The binning math is exactly what's in binStat.Compute today —
    // copied verbatim, just operating on TransformInput.Data and writing
    // through to TransformOutput.Data, TransformOutput.Mapping (rewritten
    // via b.OutputMapping()), and TransformOutput.Params (unchanged for bin).
    xCol := in.Mapping[b.axis]
    if xCol == "" {
        return TransformOutput{}, fmt.Errorf("binX: missing %q aesthetic", b.axis)
    }
    vals, err := in.Data.Float64(xCol, dataset.Clean)
    if err != nil { return TransformOutput{}, fmt.Errorf("binX: %w", err) }

    // ... existing binning math from binStat.Compute, unchanged ...

    outData := newFloat64Dataset(in.Data, map[string][]float64{
        "x": centers, "x1": lowerEdges, "x2": upperEdges, "count": counts,
    })

    // Apply OutputMapping over input mapping to produce the output mapping.
    outMapping := make(ggplot.AesMap, len(in.Mapping)+4)
    maps.Copy(outMapping, in.Mapping)
    maps.Copy(outMapping, b.OutputMapping())

    return TransformOutput{
        Data:    outData,
        Mapping: outMapping,
        Params:  in.Params,
    }, nil
}
```

The binning math is identical. The interface differs only in `Apply` taking and returning `TransformInput`/`TransformOutput` instead of `(ctx, ds, mapping, opts)` and `(ds, error)`. Every existing stat translates this way.

### 4.3 The functional-option pattern for transforms

Today, `stat.Options` is one big struct passed via `Stat.Compute` — fields like `Bins`, `Method`, `Whisker`, `Notch`, `BinMethod`, etc. used by different stats. Under the new model, each transform is its own type with its own options:

```go
// stat/bin.go
type BinOption func(*binConfig)
func WithBins(n int) BinOption                { return func(c *binConfig) { c.Bins = n } }
func WithBinMethod(method string) BinOption   { return func(c *binConfig) { c.Method = method } }
func WithCumulative(dir int) BinOption        { return func(c *binConfig) { c.Cumulative = dir } }
func WithFilter(p Predicate) BinOption        { return func(c *binConfig) { c.Filter = p } }

// stat/smooth.go
type SmoothOption func(*smoothConfig)
func WithMethod(m string) SmoothOption        { return func(c *smoothConfig) { c.Method = m } }
func WithSpan(s float64) SmoothOption         { return func(c *smoothConfig) { c.Span = s } }
func WithPoints(n int) SmoothOption           { return func(c *smoothConfig) { c.Points = n } }
func WithSE(enabled bool) SmoothOption        { return func(c *smoothConfig) { c.SE = enabled } }

// stat/density.go
type DensityOption func(*densityConfig)
func WithBandwidth(bw float64) DensityOption  { return func(c *densityConfig) { c.Bandwidth = bw } }
// ...

// stat/boxplot.go
type BoxplotOption func(*boxplotConfig)
func WithWhisker(rule string) BoxplotOption   { return func(c *boxplotConfig) { c.Whisker = rule } }
func WithNotch(enabled bool) BoxplotOption    { return func(c *boxplotConfig) { c.Notch = enabled } }
```

This is more idiomatic Go than the central-struct-with-magic-field-names pattern. Each transform's options are visible at the call site and validated locally.

## 5. The four real composition unlocks

To make the value proposition concrete — the four transforms that don't exist today and would unlock substantial chart variety:

### 5.1 NormalizeY / NormalizeX

```go
// NormalizeY rescales the y channel of its input so each group sums to
// the given total (default 1.0). Used after BinX/Count to convert
// frequencies into proportions.
//
// Inner is the transform whose output should be normalized. Pass nil
// to normalize raw input data.
//
// Output hint: y becomes HintProportion (axis renders as %).
func NormalizeY(inner Transform, opts ...NormalizeOption) Transform { ... }

// NormalizeX is the x-axis symmetric counterpart.
func NormalizeX(inner Transform, opts ...NormalizeOption) Transform { ... }

// Usage:
geom.RectY(stat.NormalizeY(stat.BinX(stat.WithBins(40))))
// → histogram of proportions, y-axis labeled as percentages
```

The hint propagation is the crucial detail. `NormalizeY` declares `OutputHints: {"y": HintProportion}`. Axis training reads the hint and formats ticks as `0%`/`25%`/`50%`/`100%` instead of `0.00`/`0.25`/...

### 5.2 Stack and StackY as transforms

Today, stacking happens in `applyPositionAdjust` inside `ggplot.go` — it's a *position adjustment*, not a transform. The result is that you can't stack the output of a transform; stacking happens to the final geom data.

Under the new model, `StackY` becomes a `Transform`:

```go
func StackY(inner Transform, opts ...StackOption) Transform { ... }

// Usage:
geom.RectY(stat.StackY(stat.BinX(stat.WithBins(40))))
// → stacked histogram across groups

geom.AreaY(stat.StackY(stat.Density()))
// → stacked density plot (currently impossible — Density doesn't compose)
```

This isn't a deprecation of `position.Stack` — `position.Stack` stays as a position-level adjustment (it handles the X-axis dodging concern that's orthogonal). But the Y-stacking math moves to a Transform so it can sit inside a pipeline. Both coexist; the position adjustment is "where to put bars side-by-side at the same X", the transform is "how to accumulate Y values across groups."

### 5.3 Filter / Sort / Select / TopN

These are trivial transforms that ggplot2 doesn't have at all and Plot uses constantly:

```go
func Filter(inner Transform, p Predicate) Transform { ... }
func Sort(inner Transform, by SortBy) Transform { ... }
func Reverse(inner Transform) Transform { ... }
func TopN(inner Transform, n int, byChannel string) Transform { ... }
func Select(inner Transform, mode SelectMode) Transform { ... } // first/last/min/max

// Usage:
geom.RectY(stat.TopN(stat.Count(), 10, "count"))
// → top-10 categories by count, currently requires preprocessing

geom.PointY(stat.Filter(nil, func(r int, t dataset.Table) bool {
    return /* condition */
}))
// → filtered scatter without preprocessing
```

Inner-transform composition reads naturally: `TopN(Count())` is "count, then take top 10." `Sort(BinX(...), Desc("count"))` is "bin, then sort bins by count descending." Each composition is one Go expression, no intermediate dataset.

### 5.4 GroupX with reducer vocabulary

The bridge to Plot's `proportion-facet` / `deviation` / `mode` reducer story:

```go
// GroupX groups data by the x channel and applies a named reducer to the
// y channel within each group. Reducers: "count", "sum", "mean", "median",
// "min", "max", "first", "last", "mode", "deviation", "p10", "p50", "p90",
// "proportion", "proportion-facet".
func GroupX(reducer string, inner Transform) Transform { ... }
func GroupY(reducer string, inner Transform) Transform { ... }
func Group(xReducer, yReducer string, inner Transform) Transform { ... }

// Usage:
geom.PointY(stat.GroupX("mean", nil), aes.Y("price"))
// → scatter of mean price per x category
// Replaces today's stat.Summary which is closed-vocabulary
```

The reducer registry is a separate package (`stat/reducer/`) but its design is straightforward and doesn't deserve its own document section — it's a lookup table from string to `func([]float64) float64` with output hints. ~200 lines total including all the named reducers.

## 6. What stays unchanged

To bound the scope and reassure on stability:

- **`Plot` and its builder methods.** `New`, `Layer`, `Aes`, `ScaleX`, `ScaleY`, `ScaleColor`, `ScaleFill`, `ScaleColorManual`, `ScaleColorContinuous`, `FacetWrap`, `FacetGrid`, `Coord`, `CoordFlip`, `Theme`, `XLim`, `YLim`, `Labs`, `LegendPosition`, `Save`, `WriteTo`. Every public method on `*Plot` keeps the same signature.
- **`Built` and its methods.** `LayerData`, `NumPanels`, `NumLayers`, `Theme`, `Labels`, `PanelLayout`, `Save`, `WriteTo`, `DrawCanvas`, `Draw`. Untouched.
- **`PlotSpec`** in `spec.go`. The `Layers []LayerSpec` field type doesn't change (LayerSpec still wraps a `geom.Layer`).
- **`colormap` package.** Cmap, Norm, Scale, Resolve, NewContinuous/Discrete/Manual — all untouched.
- **`theme` package.** Element, Theme, parentOf, resolveText/Line/Rect, all 25+ named themes, baseTheme. Untouched.
- **`scale` package.** Scale, ConfiguredScale, DiscreteScale, BoundsSetter, Expander, all options, Resolve, ticks algorithm. Untouched.
- **`coord`, `facet`, `aes`, `canvas`, `fonts`, `dataset/*`, `output`.** All untouched.

The blast radius is `stat/*.go`, `geom/geom.go` (one new field on Layer, new constructors RectY/LineY/AreaY/PointY, etc.), and one section of `buildPanel` in `ggplot.go`. Everything else is exactly where it is today.

## 7. Migration plan

Three minor releases. The deprecation policy is the same one CHANGELOG already follows for `coord.IsFlipped` and `Stat`-style parameters.

### 7.1 v0.6 — Land the new model alongside the old

| # | Package | Action |
|---|---------|--------|
| 1 | `stat/` | New `Transform` interface in `stat/transform.go`. Existing `Stat` interface preserved. |
| 2 | `stat/` | Each existing stat gets a sibling `Transform` factory: `BinX`, `Count`, `Density`, `Smooth`, `Summary`, `Boxplot`. The original `binStat`/`countStat`/etc. types implement an `AsTransform()` shim so `stat.Lookup(Name).AsTransform()` works in `buildPanel`. |
| 3 | `stat/` | New transforms: `NormalizeY`, `NormalizeX`, `StackY`, `Filter`, `Sort`, `Reverse`, `TopN`, `Select`, `GroupX`, `GroupY`. New reducer registry in `stat/reducer/`. |
| 4 | `geom/` | New `Pipeline []stat.Transform` field on `Layer`. New constructors: `RectY`, `RectX`, `LineY`, `LineX`, `AreaY`, `AreaX`, `PointY`, `RibbonY`, `Difference`. |
| 5 | `geom/` | Existing constructors (`Point`, `Line`, `Bar`, `Histogram`, `BoxPlot`, `Smooth`, `Density`, `Area`, `Col`) become one-line shims over the new constructors. Public signatures unchanged. |
| 6 | `ggplot.go` | `buildPanel` reads `Layer.Pipeline` when non-empty, falls back to `stat.Lookup(StatName).AsTransform()` for v0.5 callers. The change is one loop body, ~20 lines. |
| 7 | `ggplot.go` | New `Built.PipelineFor(panel, layer) []string` returns transform names for introspection. |
| 8 | tests | Golden tests for each new transform's output schema and channel hints. Composition tests for the four unlocks (§5). |
| 9 | docs | New `docs/TRANSFORMS.md` documenting the model and composition examples. |

After v0.6 ships:
- Old API: works unchanged. `geom.Histogram(geom.WithBins(40))` produces a Layer with the new Pipeline machinery underneath.
- New API: available. `geom.RectY(stat.BinX(stat.WithBins(40)))` works.
- Both APIs interoperate. `geom.Histogram(...)` and `geom.RectY(...)` produce structurally-equivalent Layers.

Deprecation markers on:

```go
// Deprecated: use stat.BinX(stat.WithBins(n)) inside a geom constructor instead.
// Example: geom.RectY(stat.BinX(stat.WithBins(40))) replaces
// geom.Histogram(geom.WithBins(40)). Will be removed in v0.8.
type binStat struct{}
```

### 7.2 v0.7 — Soft-warn on deprecated paths

| # | Action |
|---|--------|
| 1 | Calling `stat.Lookup(...)` (the old registry) emits a typed `Diagnostic{Code: "GGD002", Level: Warning}` once per process. Surfaces in `Built.Diagnostics`. |
| 2 | `Plot.Explain()` introspection method added — returns the resolved pipeline per layer as a string slice (useful for debugging composition). |
| 3 | All examples in `examples/` migrate to the new API. |
| 4 | `docs/MIGRATION.md` published. |

### 7.3 v0.8 — Remove deprecated shims

Old `stat.Stat` interface and its concrete types removed. `Layer.StatName` field removed. `stat.Lookup` removed. CHANGELOG documents the removals with the equivalent new-API call for each removed symbol.

### 7.4 Compatibility CI

Throughout v0.6–v0.8, the existing `ggplot_golden_test.go` and `ggplot_test.go` run against the new internals with the old API. Visual drift fails the build. Non-negotiable, same as the CHANGELOG implies for the current orientation-aware geoms refactor.

## 8. Risks and how to handle them

### 8.1 The interface explosion problem

Concern: every transform option becomes its own type (BinOption, SmoothOption, DensityOption, ...). Today you have one `stat.Options` struct with all fields.

Response: that's the point. Today, `stat.Options.Bandwidth` is meaningful only for `stat.Density`; setting it on `stat.Bin` is silently ignored. The current design pretends one config covers all stats. The new design respects the type system. The cost is ~7 option types instead of 1 struct; the benefit is compile-time validation that you don't set `WithBandwidth` on `BinX`.

### 8.2 Performance: the per-transform allocations

Concern: each Transform.Apply returns a new TransformOutput, and chaining N transforms allocates N intermediate datasets.

Response: each step today already allocates a new dataset (look at `binStat.Compute` returning `newFloat64Dataset(...)`). The composition model adds a thin TransformInput/Output wrapper struct (3 fields, all already paid for) per step — negligible overhead. The deeper performance question is whether transforms can share intermediate buffers, which is a Dataset-engine concern. Your existing `dataset.Dataset.Collect(ctx)` is the materialization point; composed transforms can stay lazy through `Apply` chains and only materialize at the end, matching today's behavior.

### 8.3 The position/transform split

Concern: putting StackY in `stat/` parallels `position.Stack`. Two ways to stack creates confusion.

Response: they handle different concerns. `position.Stack` adjusts X positions for side-by-side bars (the "dodge axis"). `stat.StackY` accumulates Y values for stacked rectangles (the "height axis"). Plot keeps these conceptually unified inside `stack`, but Plot's mark API is also implicit about which axis is being adjusted. Your existing position package handles X-axis adjustment well; the new `stat.StackY` handles Y-axis accumulation. Both can be present in a pipeline without conflict.

For users, document the rule once: "use `position.Stack` when bars should sit side-by-side on the X axis (grouped categorical); use `stat.StackY` when bar heights should accumulate (stacked-on-Y)." Two different visual outcomes; two different tools.

### 8.4 Channel-hint vocabulary lock-in

Concern: adding HintCount/HintProportion/HintInterval/HintProbability/HintCumulative/HintDeviation as a closed enum locks future stats out of declaring new hints.

Response: make it an open string type (`type ChannelHint string`). Consumers (axis formatters, color bar formatters) do best-effort matching: known hints get special formatting; unknown hints get default formatting. Third-party transforms can declare arbitrary hints; first-party axis/legend code recognizes the documented vocabulary; extensions add to it without package changes.

### 8.5 Discoverability vs implicitness

Concern: today `geom.Histogram(geom.WithBins(40))` is one symbol. New API requires the user to know both `geom.RectY` and `stat.BinX`. Cognitive load increase.

Response: this is real. The mitigation is keeping the sugar constructors as first-class API forever, not deprecating them. `geom.Histogram` stays. Users who don't need composition never reach for `RectY` + `BinX`. Users who do need composition discover it via doc examples. The progressive-disclosure axis is preserved.

Plot has the same tension and resolves it the same way: `Plot.dot(data, {x, y})` is the simple form; `Plot.dot(data, Plot.binX(...))` is the composition form. Both exist; the user picks.

### 8.6 The "stat.AsTransform" adapter

Concern: the migration path requires existing `stat.Stat` types to implement an `AsTransform() Transform` method. That's a behavior change for anyone who has implemented a custom Stat.

Response: ship a default adapter:

```go
// stat/adapter.go

// AsTransform wraps an existing Stat implementation as a Transform.
// Used internally during v0.6/v0.7 to bridge old-API geoms; available
// publicly for third parties migrating custom stats.
func AsTransform(s Stat, opts Options) Transform {
    return &statAdapter{stat: s, opts: opts}
}

type statAdapter struct {
    stat Stat
    opts Options
}

func (a *statAdapter) Name() string                       { return string(a.stat.Name()) }
func (a *statAdapter) OutputSchema() []string             { return a.stat.OutputSchema() }
func (a *statAdapter) OutputMapping() map[string]string   { return a.stat.OutputMapping() }
func (a *statAdapter) OutputHints() map[string]ChannelHint { return nil } // legacy stats don't declare hints

func (a *statAdapter) Apply(ctx context.Context, in TransformInput) (TransformOutput, error) {
    out, err := a.stat.Compute(ctx, in.Data, in.Mapping, a.opts)
    if err != nil { return TransformOutput{}, err }
    newMapping := make(ggplot.AesMap, len(in.Mapping)+len(a.stat.OutputMapping()))
    maps.Copy(newMapping, in.Mapping)
    maps.Copy(newMapping, a.stat.OutputMapping())
    return TransformOutput{Data: out, Mapping: newMapping, Params: in.Params}, nil
}
```

No third-party Stat author has to change anything to keep working through v0.7.

## 9. Test strategy

### 9.1 Output-schema golden tests

```go
// stat/bin_test.go
func TestBinX_GoldenSchema(t *testing.T) {
    tf := stat.BinX(stat.WithBins(20))
    require.Equal(t, []string{"x", "x1", "x2", "count"}, tf.OutputSchema())
    require.Equal(t, map[string]string{"x": "x", "y": "count", "x1": "x1", "x2": "x2"}, tf.OutputMapping())
    require.Equal(t, map[string]ChannelHint{"y": HintCount, "x1": HintInterval, "x2": HintInterval}, tf.OutputHints())
}
```

### 9.2 Composition correctness

```go
// stat/composition_test.go
func TestNormalizeY_AfterBinX(t *testing.T) {
    ds := testFixture(t)
    inner := stat.BinX(stat.WithBins(10))
    outer := stat.NormalizeY(inner)

    out, err := outer.Apply(ctx, stat.TransformInput{Data: ds, Mapping: AesMap{"x": "weight"}})
    require.NoError(t, err)

    counts, _ := out.Data.Float64("count")
    sum := 0.0
    for _, c := range counts { sum += c }
    require.InDelta(t, 1.0, sum, 1e-10, "normalize should produce proportions summing to 1")
    require.Equal(t, HintProportion, out.Hints["y"])
}
```

### 9.3 Layer-data equivalence under old vs new API

For backward compat, generate the same plot two ways and check that `Built.LayerData(panel, layer)` produces structurally identical data:

```go
func TestHistogramAPIEquivalence(t *testing.T) {
    ds := testFixture(t)
    oldP := ggplot.New(ds, aes.X("weight")).Layer(geom.Histogram(geom.WithBins(40)))
    newP := ggplot.New(ds, aes.X("weight")).Layer(geom.RectY(stat.BinX(stat.WithBins(40))))

    oldBuilt, _ := oldP.Build(ctx)
    newBuilt, _ := newP.Build(ctx)

    assertSameLayerData(t, oldBuilt.LayerData(0, 0), newBuilt.LayerData(0, 0))
}
```

If this test passes for every existing geom×stat combination, the migration is safe. This is the test that gates the v0.7 deprecation warnings — if any combination fails equivalence, the deprecation is held back.

### 9.4 Visual golden equivalence

The existing `ggplot_golden_test.go` is the ultimate safety net: same plot specs, byte-identical PNG output between old and new pipelines. If those tests pass unchanged through v0.6, the refactor is invisibly safe.

## 10. The work plan

Realistic engineering time, based on the codebase complexity I've now seen:

| Phase | Work | Time |
|-------|------|------|
| W1 | Add `Transform` interface in `stat/transform.go`. Write `AsTransform()` adapter. Update `buildPanel` to consume `Layer.Pipeline` with adapter fallback. All existing tests pass unchanged. | 2 days |
| W2 | Migrate `stat.Bin` to `stat.BinX` (new transform). Side-by-side: old binStat still works. Pipeline equivalence tests. | 1 day |
| W3 | Migrate `stat.Count`, `stat.Density`, `stat.Smooth`, `stat.Summary`, `stat.Boxplot` to transform factories. Equivalence tests for each. | 3 days |
| W4 | New transforms: `NormalizeY`/`NormalizeX`, `Filter`, `Sort`, `Reverse`, `TopN`, `Select`. Schema/hint tests. | 2 days |
| W5 | New transforms: `StackY`/`StackX`, `GroupX`/`GroupY` with reducer registry. Composition tests for stack-after-bin. | 3 days |
| W6 | New geom constructors: `RectY`/`RectX`, `LineY`/`LineX`, `AreaY`/`AreaX`, `PointY`, `RibbonY`, `Difference`. | 3 days |
| W7 | Backward-compat shims: rewrite `Histogram`, `Bar`, `Col`, `Smooth`, `Density`, `Area` as one-liners over new constructors. Visual golden tests pass unchanged. | 2 days |
| W8 | Documentation: `docs/TRANSFORMS.md`, composition recipe cookbook, migration guide. Update README with composition examples. Cut v0.6. | 2 days |

**Total: 18 days of focused work, or ~4 weeks at a sustainable pace.** Less than the 12-week estimate in `options-refactor-v0.2.md` because most of the infrastructure that doc proposed building (ValueSpec, Channel system, theme Element refactor) already exists. The new work is genuinely additive — new types, new constructors, new transforms — not restructuring existing code.

## 11. Why this is the right v0.6 move

Three reasons:

**1. It's the only architectural delta with both reference designs.** ggplot2 doesn't have stat composition. Plot does. Adding it puts you between them — taking ggplot2's terminology and structure but matching Plot's compositional power. The roadmap doc's "this is not a port" claim becomes load-bearing.

**2. The infrastructure is already in place.** The Build/Render separation, the Layer struct with extension fields, the `stat.Stat.OutputMapping` contract, the system PANEL/group columns, the typed Names for stats/geoms/positions — every prerequisite for this refactor is already shipped. This is not building infrastructure; it's connecting infrastructure that exists into a composable shape. The 4-week estimate is realistic because the substrate is mature.

**3. It unlocks chart variety with minimal new code.** Section 2.4's eight compositions are eight new chart types unlocked from existing math. With ~12 transforms × ~8 compatible geoms = ~100 chart shapes available from one infrastructure change. Each new transform after that adds another row to the matrix.

The polish items (S1–S5 from `analysis.md`: tabular nums, bar insets, the four placeholder geoms) ship in v0.5 and represent ~2 days of work. This refactor lands in v0.6 and represents 4 weeks. Both are visible, both are valuable, and they're sequenced correctly — polish what works now, refactor for the next architectural step in the cycle after.

---

**Open questions for you:**

1. Is `stat.Transform` the right package, or should this be `pipeline.Transform` (new package)? My recommendation: stay in `stat/` because that's where the migration is most natural for existing users. New `stat/transform.go` file. But a separate `pipeline/` package is defensible if you want to signal "this is a new thing." answer: keep stat

2. The `StackY` vs `position.Stack` split — comfortable with both coexisting, or should `position.Stack` get a deprecation cycle of its own and become an alias for `stat.StackY` in pipeline context? answer: eliminate position.Stack 

3. Reducer vocabulary scope — match Plot's exact vocabulary (~15 names), or extend with statistically-richer reducers (Welford variance, quantile-based outlier rejection) that Plot lacks? answer: extend

4. `geom.RectY(nil, ...)` for the no-transform case — does `nil` read clearly, or should it be `geom.RectY(stat.Identity(), ...)` for explicitness? Plot uses an explicit empty-options form.

If you want me to draft the actual interface code as a PR-ready diff against the current `stat/stat.go` and `geom/geom.go`, that's the natural next step.
