# ggplot — Revised Roadmap

> **Pure-Go implementation of the Grammar of Graphics.**
> Inspired by [ggplot2](https://ggplot2-book.org/) (R, Wickham et al.), but architected for Go: static typing, composable interfaces, explicit pipelines, panel-parallel rendering, and zero-copy columnar data.
>
> This is **not a port**. It is a Grammar of Graphics implementation that takes ggplot2's *concepts* and re-expresses them through Go's design vocabulary — interfaces over `ggproto`, generics over R's S3 dispatch, `context.Context` over global state, `io.Writer` over device handles, errgroups over implicit serialization.

---

## Guiding Principles

1. **Spec, Build, Render are three distinct phases with public boundaries.**
   `Plot` is a declarative spec. `Build(ctx)` resolves data through the grammar pipeline and returns a `*Built`. `Built.Draw(ctx, canvas)` produces graphical output. Each boundary is testable in isolation.

2. **Data flows through a typed columnar abstraction (`dataset.Table`) with three engines: memory, Arrow, BigQuery.** Lazy where possible. Materialized only at `Collect(ctx)`.

3. **Every layer's data carries `PANEL` and `group` columns** from data-prep onward. These are first-class system columns, not internal annotations.

4. **Position scales are trained twice** (pre-stat for transform application, post-stat for range). Non-position scales are trained once, after position adjustment. This is the correct grammar order, not an optimization.

5. **The Theme `Element` system is foundational, not cosmetic.** Every guide drawer (axis, legend, strip) consumes `ElementText`/`ElementLine`/`ElementRect`/`ElementBlank` with inheritance. Named themes are instances; the system is the spec.

6. **Extension contracts (`Geom`, `Stat`, `Scale`, `Position`, `Coord`) are public and stable from v0.x onward.** Third-party packages pin to them. Breaking changes cost a major version bump.

7. **No global state.** No `theme_set()`. No package-level mutable defaults. Everything is on the `Plot` value or passed explicitly.

8. **Concurrency is panel-parallel, not pixel-parallel.** Once `Build` is done, panels are independent. `errgroup` fans out drawing.

9. **Errors are typed and phase-aware.** `*ggplot.Error{Phase, Layer, Stage, Cause}`. Build-time errors (user spec) are distinguishable from engine errors (data/SQL/Arrow) and render errors (canvas).

10. **Performance is a tracked invariant.** Allocs/op and ns/op are gated in CI for canonical plots. Regressions block PRs.

---

## The Pipeline (Reference)

```
                  ┌──────────────────────────────────┐
                  │  Plot (immutable spec)           │
                  │  data, layers, scales, coord,    │
                  │  facet, theme, labels            │
                  └────────────────┬─────────────────┘
                                   │ Build(ctx)
                                   ▼
  ┌────────────────────────────────────────────────────────────────────┐
  │ BUILD PHASE                                                        │
  │  1. Per-layer data resolution (data ← layer or plot default)       │
  │  2. Coord pre-pass + Facet inspection → assign PANEL column        │
  │  3. Aesthetic evaluation → group column from non-continuous aes    │
  │  4. Scale-transform pass (log/sqrt applied to raw data)            │
  │  5. Position-scale 1st training + oob handling + NA filtering      │
  │  6. Stat compute (split by PANEL × group, then reassemble)         │
  │  7. after_stat() aesthetic re-binding                              │
  │  8. Geom data setup (xmin/xmax reparameterisation, etc.)           │
  │  9. Position adjustment (stack/dodge/fill/jitter)                  │
  │ 10. Position-scale 2nd training (range may have changed)           │
  │ 11. Non-position scale training (color, size, alpha, shape, …)     │
  │ 12. after_scale() aesthetic re-binding                             │
  │ 13. finish_data() hook (stat + facet)                              │
  └─────────────────────────────┬──────────────────────────────────────┘
                                │ → *Built{Layers, Layout, Coord, Theme}
                                │   (data-only, fully resolved)
                                │
                                │ Built.Draw(ctx, canvas)
                                ▼
  ┌────────────────────────────────────────────────────────────────────┐
  │ RENDER PHASE                                                       │
  │  1. Layout solve (panel grid, strip placement, legend pack)        │
  │  2. Per-panel coord transform (cartesian → polar, etc.)            │
  │  3. Per-panel geom render → Grob list  ← errgroup parallel         │
  │  4. Guide render (axes, legends, color bars) using Theme Elements  │
  │  5. Adornment (title, subtitle, caption, plot margin)              │
  │  6. Composite onto canvas (PNG / SVG / PDF / writer)               │
  └────────────────────────────────────────────────────────────────────┘
```

This is the contract. Every phase, geom, and refactor is judged against whether it preserves or violates these stages.

---

## Roadmap Phases

> Legend: ✅ shipped · 🔶 partial · 🔲 planned · ⚠️ blocked/needs design

---

### Phase 1 — Core Grammar & Data Plane 🔶

> Book Ch.1–2 (Introduction, First Steps), Ch.13 (Layers), Ch.19 (Internals)

**Public API:**
- ✅ Typed `Dataset`/`Frame` with `DType` system (Float64, Int64, String, Bool, Timestamp, Date, Time) and null masks
- ✅ Fluent ETL: `Select`, `Filter`, `Mutate`, `Arrange`, `Distinct`, `Summarize` (eager; lazy planned)
- ✅ Cross-type column iterators
- ✅ Canvas abstraction over `gg` with full 2D primitives
- ✅ Aesthetic constructors (`X`, `Y`, `Color`, `Group`, `Fill`, `Label`, `Size`, `Alpha`)

**Pipeline scaffolding:**
- ✅ `PlotSpec` declarative composition
- ✅ `Plot.Build(ctx)` / `Built.Draw(ctx, canvas)` pipeline — see Phase 4.1
- ✅ `PANEL` and `group` as first-class system columns — see Phase 4.2

**Engine extensibility:**
- ✅ `NativeFilterProvider`, `IterableColumn` interfaces
- 🔲 Document the column extension surface (custom dtypes: decimal, geo, fixed-point)

---

### Phase 2 — Grammar Primitives ✅ (with gaps)

> Book Ch.3 (Individual Geoms), Ch.4 (Collective Geoms), Ch.5 (Statistical Summaries)

**Geometries:** ✅ Point, Line, Path, Step, Bar, Histogram, Area, Density, Rug, HLine, VLine, Text, BoxPlot, Smooth

**Statistics:** ✅ Identity, Bin/Count, Density (KDE), Smooth (LOESS + LM), Summary, BoxPlot
- ✅ **Percentile reducers** (`"p10"`, `"p25"`, `"p50"`, `"p75"`, `"p90"`) in `GroupX`/`GroupY` and `Group` — dispatched via engine-native `AggPercentile`

**Scales:** ✅ Linear, Log10, Sqrt, Reverse, Discrete + `scale.Resolve(name)` factory

**Coordinates:** ✅ Cartesian, Polar + geom-level Orientation

**Faceting:** ✅ None, Wrap (NCols/NRows), Grid (row ~ col), Strip Labels

**Themes:** ✅ Default, Classic, Minimal, Dark, BW *(hard-coded — to be refactored onto Element system in Phase 4)*

**Guides:** ✅ X/Y Axes, Grid, Categorical Legend, Continuous Color Bar (vert + horiz)

**Aesthetics:** ✅ X, Y, Color, Group, Fill, Label, Size, Alpha

**Builder shortcuts:** ✅ `LegendPosition()`, `ScaleX("log10")`, `ScaleY("sqrt")`

> ✅ **Resolved:** `geom.Bar` now defaults to `position.Stack`. Position adjustments (stack, dodge, fill) are wired into the build pipeline as of Phase 4.3.

---

### Phase 3 — Data Backends 🔶

- ✅ **Arrow adapter** — zero-copy `TableDataset` / `TableColumn`, chunked iterators, `Buffer` pre-allocator
- ✅ **SQL/BigQuery adapter** — lazy `Table` with predicate pushdown, `FilterSQL` / `GroupBySQL`, auto type detection, escaped filters, `errors.Is(err, io.EOF)` correctness
- ✅ Backend extensibility: `NativeFilterProvider`, `IterableColumn`
- 🔲 **Iceberg/S3Tables read** *(natural alignment with existing Beam-Iceberg work)* — read-side adapter mirroring the BQ engine
- 🔲 **DuckDB engine** — for local Parquet/CSV without spinning Arrow IPC, useful for notebooks/CLI
- 🔲 **`GroupAggregator` interface** — current `dispatchAgg` allocates per group; Arrow `hash_aggregate` for single-pass grouped agg. **Required before any new Phase 9 stat.**
- 🔲 **FNV-64 collision guard** in `execDistinct` (frame.go:802) — fix: xxh3 + equality fallback at >10M rows
- 🔲 **`context.Context` plumbed through engine sub-interfaces** — currently only at `executeOps` boundary; `AddCols`, `Sum`, `Join`, `PivotLonger` need it for cancellation across long BQ scans
- 🔲 **OpenTelemetry spans** in BQ engine — surface bytes-billed and slot-millis as metrics

---

### Phase 4 — Pipeline Refactor & Extension Contracts ⚠️ (Architectural Foundation)

> **This phase is the gate for everything after.** No new geoms, scales, or themes ship until these contracts are public and frozen.

#### 4.1 — Build / Render Separation ✅

- ✅ Public `Built` type with panels, layout, coord, theme, labels
- ✅ `Plot.Build(ctx context.Context) (*Built, error)`
- ✅ `Built.Draw(ctx context.Context, c canvas.Canvas) error`
- ✅ `Built.LayerData(panel, layer int) Dataset` + `NumPanels()` / `NumLayers(panel)` — introspection for testing & debugging
- ✅ Decomposed rendering pipeline: `buildPanel`, `trainPanelScales`, `applyPositionAdjust`, `drawLayer`, `drawXAxis`, `drawYAxis`, etc.

#### 4.2 — System Columns (PANEL, group) ✅

- ✅ `ColPANEL` and `ColGroup` as reserved system column constants in `spec.go`
- ✅ `Dataset.WithColumn(col)` verb + `ConstInt64Column` / `Int64ColumnFromStrings` helpers
- ✅ Facet inspection assigns `PANEL` int64 column to every panel's data
- ✅ Group splitting injects `group` int64 column per group subset
- ✅ Group colors baked into `Geom.Params.Color` at build time (no runtime override)

#### 4.3 — Position Adjustment Interface ✅

- ✅ `position.Pos` interface with `Adjust(xs, ys, width, groupIdx, nGroups)` + `String()`
- ✅ `position.Identity` (default for points/lines)
- ✅ `position.Stack` (stateful, accumulates Y offsets across groups)
- ✅ `position.Dodge` (side-by-side shifting)
- ✅ `position.Fill` (100% stacked, two-phase with `FillSetup` interface)
- ✅ `position.New(name)` factory for pipeline to create fresh per-panel instances
- ✅ `applyPositionAdjust` wired into `buildPanel()` after stat computation
- ✅ `geom.Bar`/`geom.Histogram` default to `position.Stack()`

> Jitter, Nudge, JitterDodge remain in Phase 13 — they're refinements, not core grammar.

#### 4.4 — Theme Element System ✅ (foundation, formerly Phase 12)

- ✅ `theme.Element` sealed interface (`element()` unexported marker)
- ✅ `theme.ElementText{Family, Size, Color, Bold, Italic, Hjust, Vjust, Angle, Margin, LineHeight}`
- ✅ `theme.ElementLine{Color, Size, Linetype, Lineend}`
- ✅ `theme.ElementRect{Fill, Color, Size, Linetype}`
- ✅ `theme.ElementBlank{}` — suppresses drawing
- ✅ `MergeText`, `MergeLine`, `MergeRect` — zero-value-aware field merge for inheritance
- ✅ `IsBlank(Element) bool` helper
- ✅ Inheritance hierarchy via `Elements map[string]Element` + `parentOf` tree (`axis.title.x` ← `axis.title` ← `text` ← root)
- ✅ Typed resolver methods: `PlotTitle()`, `AxisTextElem()`, `PanelBackground()`, `PanelGridMajor()`, `AxisTicks()`, etc.
- ✅ All 45 themes migrated to Element compositions (no backward compat — old `TextStyles`/`GridStyle`/`PanelStyle`/`TickStyle`/`FontConfig` removed)
- ✅ Rendering pipeline (`ggplot.go`) consumes Elements via typed accessors (70+ call sites migrated)
- ✅ `neutralPaletteTheme()` helper for palette-only themes
- ✅ New themes: `Observable` (Observable10 palette), `Tableau` (Tab10 palette, previously unregistered)

> Granular `theme()` overrides and `theme.New(base, ...)` user composition stay in Phase 12; this phase establishes the substrate.

#### 4.5 — Extension Contracts ✅ (formerly Phase 17 interfaces)

The public extension API — implemented via Go interfaces + registration, not ggplot2-style class hierarchies:

- ✅ `stat.Transform` — `Name()`, `Apply(ctx, TransformInput) (TransformResult, error)`, `OutputMapping()`, `OutputSchema()`, `OutputHints()`; composable pipeline contract
- ✅ `scale.Scale` — `Train()`, `Map()`, `Inverse()`, `Ticks()`, `Format()`, `Bounds()` + optional `BoundsSetter`
- ✅ `geom.Pos` — `Adjust()`, `String()` + optional `Stacker`, `FillSetup`; extensible via `NewPos()` factory
- ✅ `coord.Coord` — `Transform(x, y, w, h)`, `String()`; Cartesian + Polar implementations
- ✅ Geometry extensibility via `RegisterDrawer(Type, Drawer)` + `RegisterGeomType(Type, OptFlag)` — no `geom.Geom` interface needed; registration is more idiomatic Go than a class hierarchy
- ✅ `TypeRect` — unified rectangle mark replacing `TypeBar`/`TypeHistogram` for pipeline constructors (`RectY`, `RectX`, `Histogram`); inset controlled by `Params.Inset`
- ✅ Backward-compat shim verification — 11 table-driven tests in `geom/shim_test.go` verify all sugar constructors produce structurally-equivalent Layers to pipeline counterparts

> `docs/MIGRATION.md` and stability-guarantee policy deferred until an actual deprecation is planned.

#### 4.6 — Production Hardening (continued)

- ✅ Lint matrix (`errcheck`, `staticcheck`, `gocritic`, `errorlint`), `-race` CI, `GOEXPERIMENT` matrix
- ✅ 6 PNG goldens (SHA-256, platform-tagged)
- ✅ Clone independence tests, deep-clone safety
- ✅ Hot-path optimizations (`strconv` over `Sprintf`, `SelectRows` over `BoolMask`, `engine()` parent walk eliminated, `flatten()` exact capacity)
- ✅ **Typed error envelope** — `*ggplot.Error{Phase, Layer, Stage, Cause}` with `errors.Is`/`Unwrap` support; 38 call sites migrated
- ✅ **Panel-parallel rendering** — `errgroup.Group` over facet panels in `Build` + data layers in `Draw`; single-panel fast path
- 🔲 **`Built`-level golden tests** — JSON snapshots of `Built` are platform-independent and orders of magnitude more sensitive than PNG goldens; PNG goldens stay as smoke tests. *Deferred: requires exposing a public `Plot.Build() → Built` API first.*
- 🔲 **Performance CI gates** — track allocs/op and ns/op for ~10 canonical plots; fail PR on >10% regression. *Benchmarks exist (11 render + 40+ engine); needs CI workflow with `benchstat` comparison. Infrastructure-only, no code change.*
- ~~🔲~~ ~~**`Built.Diagnostics`** — typed warnings for deprecated stat paths (legacy `StatName` usage)~~ *Removed: `StatName` field was fully replaced by `stat.Transform` pipeline; no deprecated paths remain.*
- 🔲 **Real SIMD execution** — pending Go SIMD intrinsics that don't heap-escape `Vec[T]` through generic call sites ([golang/go#65592](https://github.com/golang/go/issues/65592)); current scalar loops + `dmath`. *Blocked by Go upstream.*
- 🔲 **KDE inner-loop SIMD** — `stat.Density` Gaussian kernel via `compute.Exp`/`compute.Mul`/`compute.ReduceSum`; currently scalar + NumCPU parallelism. *Blocked by Real SIMD above.*

#### 4.7 — Visual Polish (analysis.md §6)

- ✅ **Histogram bar inset (S4)** — 0.5px inset per side between continuous-mode bars so adjacent bins never visually merge; matches Observable Plot's default
- ✅ **Placeholder geom drawers (S5)** — `geom.Tile` (heatmap), `geom.Segment` (x,y→xend,yend), `geom.ErrorBar` (ymin/ymax + caps), `geom.Polygon` (closed paths); all four had declared types but no drawer
- ✅ **`aes.XEnd`/`aes.YEnd` channels** — endpoint aesthetics for Segment geom
- ✅ **Tabular figures on quantitative axes (S1)** — monospaced digit rendering for aligned tick labels via `SetTabularNums(true)` + OpenType `tnum` feature; enabled for X-axis labels, Y-axis labels, and Y-axis margin measurement

---

### Phase 5 — Position Scales & Axes ✅

> Book Ch.10 (Position scales and axes), Ch.14 (Scales and guides)

#### 5a — Scale Configuration ✅
- ✅ `scale.WithBreaks([]float64)` — user-supplied tick positions
- ✅ `scale.WithLabels([]string)` — user-supplied tick labels
- ✅ `scale.WithFormatter(func(float64) string)` — currency, percent, scientific, custom
- ✅ `scale.WithExpand(mult, add)` — axis padding
- ✅ `scale.WithMinorBreaks([]float64)` — minor grid line positions
- ✅ `scale.WithClipBounds(min, max)` — coord_cartesian-style zoom-without-filter

#### 5b — Axes & Time Scales ✅
- ✅ **Date/Time scale** — `scale.DateTime` with auto tick formatting (second/minute/hour/day/month/year); training on `DTypeTimestamp`/`DTypeDate`/`DTypeTime` columns; `time.Local` timezone; intraday spans show time-only labels
- ✅ **`scale.Binned`** — discretize continuous axes into range-labeled bins; auto (Sturges) or explicit bin count via `scale.WithBins(n)`; explicit edges via `scale.WithBinBreaks([]float64)`; `Format()` shows `[lo, hi)` range labels
- ✅ **Out-of-bounds (oob) policies** — `scale.WithOOB(OOBKeep | OOBSquish | OOBCensor)` per ggplot2 Ch.14.4; composable with `WithClipBounds`
- 🔲 **Secondary axes** — `SecAxis()` / `DupAxis()` for dual Y. Requires dual-axis layout (right margin, second tick column) — straightforward once `Built.Layout` exists
- ✅ **`AxisLabelRows(n)`** — auto-detect overlapping X-axis labels and stagger across n rows; rotated label support via `ElementText.Angle`; overlap skipping within dodge rows

---

### Phase 6 — Colour Scales & Legends ✅

> Book Ch.11 (Colour scales and legends)

- ✅ **Colormap registry** — `colormap.Cmap` interface, 40+ built-in colormaps (sequential, diverging, qualitative, perceptually-uniform, cyclic, miscellaneous)
- ✅ **Cyclic colormaps** — `Twilight` (perceptually-uniform) + `Phase` (HSL wrap) for circular variables
- ✅ **Turbo / JetLegacy** — high-dynamic-range + legacy MATLAB (with doc warning; never a default)
- ✅ **Theme → ColorDefaults** — `ColorDefaults{ColorDiscrete, ColorSequential, FillDiscrete, FillSequential, Diverging, Cyclic}` mapped for all 75 themes; `DefaultCmapFor(name, aesthetic, category)` lookup with Color/Fill split
- ✅ **Color/Fill split** — `ColorDefaults` has independent `ColorDiscrete`/`ColorSequential`/`FillDiscrete`/`FillSequential` fields; zero-value Fill* fields fall back to Color*
- ✅ **Wire into scale pipeline** — discrete fallback uses `theme.DefaultCmapFor(AesColor, Qualitative)`, continuous fallback uses `theme.DefaultCmapFor(AesColor, Sequential)`; removed redundant `themePaletteCmap()`
- ✅ **CIELAB gradient constructors** — `colormap.Gradient(low, high)`, `Gradient2(low, mid, high)`, `GradientN([]Color)` with perceptually-uniform CIELAB interpolation; sRGB↔XYZ↔Lab conversion
- ✅ **Discrete color** — `ScaleColorManual(map[string]color)` already shipped
- ✅ **Palette system** — viridis (A–E), ColorBrewer (sequential/diverging/qualitative), 60+ registered colormaps
- ✅ **NA color** — `colormap.Scale.SetNAColor(c *gg.RGBA)` for user-configurable missing-value color
- ✅ **Guide customization** — `ColorBarWidth(w)`, `ColorBarNBin(n)`, `LegendCols(n)` fluent API; `ColorBarSpec.BarWidth`/`NBin` wired into rendering
- ✅ **Legend key glyphs** — `LegendGlyph` type with `GlyphRect` (bars/histogram/tile), `GlyphPoint` (point/rug), `GlyphLine` (line/smooth/step/segment); `drawGlyph` renders appropriate shapes; auto-selected based on `geom.Type`

> Anti-patterns: no theme-specific colormap variants (ObservableViridis, NordMagma); no Jet/Turbo as defaults.

---

### Phase 7 — Other Aesthetic Scales ✅

> Book Ch.12 (Other aesthetics)

- ✅ **Size** — `scale.Size(range)`, `scale.SizeArea()` (proportional to value), `scale.Radius()`
- ✅ **Shape** — `scale.ShapeManual()` with 25+ built-in point shapes
- ✅ **Linetype** — `scale.Linetype()` — solid, dashed, dotted, dotdash, longdash, twodash + custom dash arrays
- ✅ **Alpha** — `scale.Alpha(range)` for opacity mapping
- ✅ **Identity scales** — `scale.*Identity()` to use raw column values directly as aesthetic values

---

### Phase 8 — Advanced Geometries 🔶

> Book Ch.3–5 (revisited with full position/coord/scale support)

**Already shipped (Phase 4.5 / 4.7):**
- ✅ `geom.Tile` — heatmap cells at (x, y) with continuous fill
- ✅ `geom.Segment` — line segments from (x, y) to (xend, yend) via `aes.XEnd`/`aes.YEnd`
- ✅ `geom.ErrorBar` — vertical/horizontal error bars with caps
- ✅ `geom.Polygon` — arbitrary filled/stroked closed polygons
- ✅ `geom.Ribbon` / `RibbonY` — filled bands between ymin/ymax
- ✅ `geom.Rect` / `TypeRect` — parameterised rectangles (xmin/ymin/xmax/ymax)
- ✅ `geom.Difference` — difference-fill between two series

**Shipped — Error bar family (Batch A):**
- ✅ `geom.Crossbar` — box with median line between ymin/ymax (no whiskers)
- ✅ `geom.Linerange` — vertical/horizontal line without caps (thin ErrorBar)
- ✅ `geom.Pointrange` — point + linerange (mean ± SE / CI)

**Shipped — Distribution geoms (Batch A):**
- ✅ `geom.Violin` — mirrored density per group (symmetric KDE around category axis); uses dedicated `stat.ViolinY`
- ✅ `geom.Dotplot` — dot plots for small datasets (stacked dots at binned positions); uses dedicated `stat.DotBin` (Wilkinson)

**Remaining — 2D density:**
- 🔲 `geom.Contour` / `ContourFilled` — 2D density contours from XYZ (requires `stat.Density2D` and marching-squares contouring)

**Shipped — Curve/Bezier (Batch A):**
- ✅ `geom.Curve` — quadratic bezier curves between endpoints via `Canvas.QuadraticTo`; curvature controlled by `geom.WithCurvature()`

**Remaining — Binning/Aggregation:**
- 🔲 `geom.Hex` — hexagonal binning (server-side aggregation hook for >1M points)

**Shipped — Position:**
- ✅ `geom.JitterPoint` — convenience constructor (`TypePoint` + `position.Jitter`); injectable seed via `WithJitterSeed(uint64)`; configurable displacement via `WithJitterWidth`/`WithJitterHeight`; deterministic via `math/rand/v2.PCG` seeded by `(seed, dataLen)`

**Remaining — Pixel grid:**
- ✅ `geom.Raster` — dense pixel-aligned image grid; native canvas transform compositing (`Save/Translate/ScaleXY/DrawImage/Restore`) for GPU-accelerated upscaling; bilinear interpolation via `WithInterpolate(true)`; half-cell padding for correct grid alignment

---

### Phase 9 — Annotations 🟡

> Book Ch.8 (Annotations)

- ✅ Reference lines: `geom.HLine(WithIntercept)`, `geom.VLine(WithIntercept)`, `geom.ABLine(WithSlope, WithIntercept)`
- ✅ `annotate()` — layer-less annotations: `AnnotateText`, `AnnotateRect`, `AnnotateSegment`, `AnnotateArrow`, `AnnotateLabel`
- ✅ `geom.Label` — text with background box (`AnnotateLabel` with `geom.WithPadding`)
- 🔲 **Direct labelling / anti-collision** — `ggrepel`-style force-directed text placement
- 🔲 **Marginal annotations** — custom axis marks, distribution overlays

---

### Phase 10 — Coordinate Systems 🟡

> Book Ch.15 (Coordinate systems)

**Shipped — Viewport zoom:**
- ✅ `coord.CartesianZoom(xlim, ylim)` / `Plot.CoordCartesian(xmin, xmax, ymin, ymax)` — viewport zoom without data clipping; overrides scale bounds after training so all data participates in stat computations; implements `coord.Zoomer` interface

**Shipped — Fixed aspect ratio:**
- ✅ `coord.Fixed(ratio)` / `Plot.CoordFixed(ratio)` — fixed aspect ratio; `ratio=1` gives equal scaling (1 unit x = 1 unit y in pixels); panel dimension shrunk and centred to enforce ratio; implements `coord.Fixer` interface

**Shipped — Polar:**
- ✅ `coord.Polar(theta, start, direction)` — already shipped; verify pie/rose/radar interactions with `position.Stack`

**Remaining:**
- 🔲 `coord.Trans(xtrans, ytrans)` — separate transformations per axis (post-stat, vs. scale transforms which are pre-stat)
- 🔲 `coord.Map(projection)` — map projections (Mercator, Lambert, equal-area)


---

### Phase 11 — Faceting Deep Dive 🔲

> Book Ch.16 (Faceting). Unblocked by Phase 4.1 (per-panel scale state).

- 🔲 **Free scales** — `facet.Wrap(col, FreeX(), FreeY())` per-panel independent axes (requires per-panel scale clones in `Built.Layout`)
- 🔲 **Free space** — `facet.Grid(row, col, Space("free"))` proportional panel sizing
- 🔲 **Labellers** — `labeller.Both()`, `labeller.Parsed()`, custom label formatters
- 🔲 **Facet margins** — `Margins(true)` for aggregate panels alongside facets
- 🔲 **Drop control** — `Drop(false)` to show empty panels for missing combinations
- 🔲 **Strip placement** — `StripPosition("bottom" | "left")` for axis-adjacent strips

---

### Phase 12 — Theme System Granularity 🔲

> Book Ch.17 (Themes). Foundation already built in Phase 4.4.

- 🔲 **Granular `theme()` overrides** — `Theme(WithAxisTitleX(ElementText{Face: "bold"}))` etc. (every `axis.title.x`, `legend.key.size`, …)
- 🔲 **User-composable themes** — `theme.New(base, overrides...)` returning a composed `Theme`
- 🔲 **Plot margin** with unit support — `cm`, `inches`, `lines`, `pt`
- 🔲 **Legend layout** — `legend.box`, `legend.key`, `legend.background`, `legend.margin`
- 🔲 **Strip styling** — `strip.text`, `strip.background`, `strip.clip`
- 🔲 **Theme inheritance proofs** — golden tests that `axis.title.x` correctly inherits from `axis.title` from `text` from root
- 🔲 **Continuous legends** — graduated-size and continuous-alpha legend rendering (deferred from Phase 7)

---

### Phase 13 — Position Adjustment Refinements 🔲

> Phase 4.3 shipped Identity, Stack, Dodge, Fill. This phase adds the rest.

- 🔲 `position.Jitter(width, height)` — random displacement; injectable `rand.Source` for determinism
- 🔲 `position.JitterDodge` — combined within dodged groups
- 🔲 `position.Nudge(x, y)` — fixed offset for labels relative to anchor

---

### Phase 14 — Maps & Spatial 🔲

> Book Ch.6 (Maps)

- 🔲 `geom.SF` — Simple Features rendering from GeoJSON / Shapefile (via `paulmach/orb` or similar)
- 🔲 `coord.SF` — CRS-aware coordinate system with proper map projection
- 🔲 `borders()` — convenience for country/state outlines
- 🔲 **Choropleth** — fill polygons by data values

---

### Phase 15 — Networks 🔲

> Book Ch.7 (Networks)

- 🔲 **Graph layouts** — force-directed (Fruchterman-Reingold), tree, circle, grid
- 🔲 `geom.Edge` / `geom.Node` — edge bundling, node glyphs
- 🔲 **Network aesthetics** — edge weight, node size mapped via standard scales

---

### Phase 16 — Composition (Patchwork) 🔲

> Book Ch.9 (Arranging plots). May be a separate `ggplot/compose` subpackage.

- 🔲 `compose.Arrange(p1, p2, p3, layout)` — multi-plot grids
- 🔲 `compose.Layout(areas)` — custom grid arrangements
- 🔲 `compose.Inset(p, x, y, width, height)` — overlaid mini-plots
- 🔲 **Shared axes** — linked scales across composed plots
- 🔲 **Collected guides** — single legend for multiple composed plots

---

### Phase 17 — Programming Surface 🔲

> Book Ch.18, Ch.20, Ch.21 (Programming, Extensions, Case study). Interfaces shipped in Phase 4.5; this phase adds higher-level programming sugar.

- 🔲 **Programmatic aesthetics** — `aes.String("colname")` for dynamic column names from variables
- 🔲 **Custom Geom protocol docs** — examples + cookbook for third-party `geom.Geom` implementations
- 🔲 **Custom Stat protocol docs** — examples for third-party `stat.Stat`
- 🔲 **Custom Scale protocol docs** — examples for third-party `scale.Scale`
- 🔲 **Smooth method expansion** — `stat.Smooth` dispatch for GAM, cubic spline, LOWESS in addition to LM/LOESS
- 🔲 `after_stat()` / `after_scale()` — computed aesthetics from stat/scale output (already in pipeline at steps 7 and 12, this exposes the user API)
- 🔲 **`ggplot/extra` module** — example third-party geoms (springs, marquee, sina) demonstrating the extension contract

---

### Phase 18 — Output & Interactivity 🔶

> Beyond book scope — Go-specific extensions.

- ✅ SVG export via `RecordingCanvas`
- ✅ PDF export via `RecordingCanvas`
- ✅ HiDPI / scaled output (`Save(... WithScale(2.0))`, `WriteTo` with `RenderOpt`)
- 🔲 **`io.Writer` outputs** — `WriteTo(w io.Writer, opt RenderOpt) error` for streaming to HTTP, gzip, blob stores
- 🔲 **HTML output** — interactive plots with hover tooltips via embedded SVG + minimal JS (no React/heavyweight runtime)
- 🔲 **Animated GIF / APNG** — frame-by-frame rendering for time-series; `Plot.Animate(frames, fn)` builds a sequence of `Built`s
- 🔲 **Live preview** — `p.Show()` opens a native window with hot-reload on data changes (development tool)
- 🔲 **Server-side aggregation hooks** — datashader-style auto-aggregation for >1M points (hexbin, gridding) before geom hits renderer

---

### Phase 19 — Documentation, Ecosystem, Stability 🔲

- 🔲 **API reference** — `godoc` with runnable examples for every public type
- 🔲 **Gallery** — 30+ curated examples with source + rendered output
- 🔲 **Cookbook** — task-oriented guides (time series, distributions, maps, composition, Beam→ggplot pipelines)
- 🔲 **Benchmarks** — `BENCHMARK.md` with comparison vs R/ggplot2 for equivalent plots; tracked over time
- 🔲 **Performance regression CI** — golden numbers for 10+ canonical plots
- 🔲 **CI/CD** — GitHub Actions: lint, test, race, benchmark, example rendering, doc build on every PR
- 🔲 **Stability policy** — public surfaces semver-guaranteed; deprecation cycle of 2 minor releases minimum; `internal/` is unpublished
- 🔲 **NOTICES** — font licensing (Go Noto Sans Apache 2.0), dependency attribution
- 🔲 **`v1.0` release** — frozen public API, CHANGELOG, migration guide from v0.x

---

## Cross-Cutting Concerns

These are not phases; they thread through every phase.

### Concurrency
- `Plot` is immutable (deep-cloned on every method) — already shipped, goroutine-safe by construction
- `Build` is single-threaded but `Stat` compute is panel-parallel via `errgroup`
- `Draw` is panel-parallel after Phase 4.1
- Engines (memory/Arrow/BQ) do their own internal parallelism

### Context propagation
- `ctx` flows through `Build`, `Draw`, every engine op
- Cancellation honored at every IO and per-row hot loop boundary
- Phase 4 hardens this end-to-end

### Determinism
- Stats with randomness (`stat.Density` with bootstrap, `position.Jitter`) accept injectable `rand.Source`
- Float-ordering invariants: stable sorts, deterministic group iteration
- Documented per-stat in godoc

### Observability
- Optional `slog.Logger` injection at `Build` boundary
- OpenTelemetry spans in BQ engine (bytes-billed, slot-millis)
- Phase counters in `Built` (rows-in, rows-out per stage) for debugging

### Error model
- `*ggplot.Error{Phase, Layer, Stage, Cause}` with `errors.Is`/`Unwrap`
- Build errors (user spec) vs Engine errors (data) vs Render errors (canvas)
- No panics in public API for non-bug conditions

### Performance gates
- `make bench` runs canonical suite
- CI compares against `benchmarks/golden.json` and fails PR on >10% regression
- Allocs/op tracked separately from ns/op

### Stability and versioning
- Public packages (`ggplot`, `aes`, `geom`, `stat`, `scale`, `coord`, `facet`, `theme`, `guide`, `position`, `dataset`, `dataset/memory`, `dataset/arrow`, `dataset/bigquery`) have stability guarantees from v0.5 onward
- `internal/*` is unpublished and may break at any time
- 2-minor-release deprecation cycle minimum

---

## Why This Roadmap (vs. ggplot2 Port)

This roadmap deliberately does not ship as a ggplot2 clone. Specifically:

| ggplot2 (R) | This package (Go) | Rationale |
|---|---|---|
| `ggproto` OOP | Go interfaces | Native, simpler, type-checked |
| S3 dispatch | Type switches and generics | Compile-time correctness |
| Lazy data via promises | `dataset.Table` lazy chain via engines | Explicit, cancellable, distributable |
| Global `theme_set()` | No global state | Concurrency-safe, embeddable |
| `aes_string()` | `aes.String("col")` | Programmatic aesthetics without quasiquotation |
| `+` operator overloading | Fluent builder methods | Idiomatic Go |
| `print(p)` triggers render | Explicit `Build()` then `Draw()` | Inspectable, testable, parallelizable |
| Tied to grid graphics | Pluggable canvas (gg, SVG, PDF, HTML, custom) | Backend-agnostic |
| Single-row dataframe | Typed `Frame` with `Column[T]` generics | Compile-time dtype, zero-copy Arrow |
| `data.frame` only | Memory, Arrow, BigQuery, Iceberg engines | Production data plane |

The grammar is the same. The implementation is Go.
