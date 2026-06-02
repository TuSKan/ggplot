# Architecture

> Go implementation of the Grammar of Graphics — declarative, composable data visualization.

---

## Overview

`ggplot` follows an immutable builder pattern where every method returns a new `Plot` value
(via internal `clone()`). The rendering pipeline is a strict sequence:

```
PlotSpec → Facet Split → Stat Transform → Scale Training → Layout → Draw → Surface
```

**Build**, **Draw**, and **Output** are separate phases. `Plot.Build()` produces an
`output.Figure` (concretely a `*Built`) containing all resolved layers, trained scales, and
layout geometry. `Figure.Draw()` — implemented by `Built.Draw` — renders data layers and
chrome onto a `canvas.Canvas`. The **output layer** then carries a `Figure` to a destination
`Surface`: a file, an in-memory image, a desktop GPU window, or a browser canvas. Build and
Draw execute **panel-parallel** via `errgroup` for multi-panel faceted plots. The output
layer is specified in [**OUTPUT-SPEC.md**](OUTPUT-SPEC.md).

All data flows through the `dataset.Table` abstraction, which provides a columnar,
iterator-based interface. Three engine backends implement it: native Go slices
(`dataset/memory`), Apache Arrow arrays (`dataset/arrow`), and BigQuery SQL
pushdown (`dataset/bigquery`). Frame verbs (defined on `dataset.Dataset`,
the fluent wrapper) build a lazy execution chain across all engines. Data is only
materialized when explicitly requested via `Collect(ctx)`.

---

## Package Map

```
github.com/TuSKan/ggplot
│
├── ggplot.go            # Plot builder API, Build/Draw pipeline orchestrator
├── errors.go            # Typed error envelope: *Error{Phase, Layer, Stage, Cause}
├── drawer.go            # Geometry drawing: drawPoint, drawLine, drawBar, drawBoxplot, drawRaster, drawAnnotation*, …
├── annotate.go          # Annotation API: AnnotateText, AnnotateRect, AnnotateSegment, AnnotateArrow, AnnotateLabel
├── spec.go              # PlotSpec, Labels, AxisGuide, RenderOpt, WithCPU()
├── util.go              # Shared helpers (normalize, resolveColor, …)
│
├── aes/                 # Aesthetic mapping constructors: X(), Y(), Color(), Fill(), Size(),
│                        #   Label(), XEnd(), YEnd(), YMin(), YMax(), Group(), …
├── geom/                # Geometry layer definitions, options, position adjustments
│   ├── geom.go          #   Point, Line, Step, Bar, Col, Histogram, Area, Density, Rug,
│   │                    #   HLine, VLine, Text, BoxPlot, Smooth, Tile, Segment, ErrorBar,
│   │                    #   Polygon, Ribbon, Difference, Crossbar, Linerange, Pointrange,
│   │                    #   Curve, Violin, Dotplot, Raster, JitterPoint + functional options
│   ├── position.go      #   Pos interface: Identity, Dodge, Stack, Fill, Jitter (with WithSeed), Nudge
│   └── shim_test.go     #   Sugar ≡ pipeline equivalence tests
│
├── stat/                # Statistical transformations (stat.Transform interface)
│   ├── transform.go     #   Transform interface: Name(), Apply(), OutputMapping(), OutputSchema()
│   ├── bin.go           #   BinX, BinY (Sturges/Scott/FD/Sqrt)
│   ├── count.go         #   Count
│   ├── density.go       #   DensityX (KDE, parallel kernel evaluation)
│   ├── smooth.go        #   SmoothXY (LOESS + lm, with SE bands)
│   ├── boxplot.go       #   BoxplotY (Tukey/range whiskers, notch CI)
│   ├── normalize.go     #   NormalizeY (proportion, percent)
│   ├── filter.go        #   Filter (predicate-based row filtering)
│   ├── group.go         #   GroupX, GroupY, Group (reducers: mean, median, sum, min, max, p10–p90)
│   ├── select.go        #   SelectX
│   ├── sort.go          #   SortX
│   ├── stack.go         #   StackY
│   ├── violin.go        #   ViolinY (grouped KDE for violin plots)
│   ├── dotbin.go        #   DotBin (Wilkinson-style greedy dot stacking)
│   └── summary.go       #   Summary
│
├── scale/               # Scale types + Resolve factory
│   └── scale.go         #   Linear, Log10, Sqrt, Reverse, Discrete + NiceSequence, FormatNumber
│                        #   SecAxisSpec, DerivedScale — secondary Y-axis via closure transforms
│                        #   BoundsSetter interface for post-build scale unification
│
├── coord/               # Coordinate systems (pure specification — zero math)
│   └── coord.go         #   Cartesian, CartesianZoom (Zoomer), Fixed (Fixer), Polar, Trans
│                        #   TransFunc is name-only; pipeline dispatches to MathKernel
│
├── facet/               # Faceting strategies
│   └── facet.go         #   None, Wrap (NCols, Labeller, Drop, FreeX/Y/XY, StripBottom),
│                        #   Grid (row ~ col, Labeller, Drop, Margins, FreeX/Y, Space, StripBottom/Left)
│                        #   Facet interface: Split, Labels, GridDims, FreeScales, SpaceMode, StripPositions
│                        #   Panel carries RowVal/ColVal/NumRows/IsMargin; mask-based lazy splitting
│
├── theme/               # Theme definitions (60+ built-in themes)
│   ├── theme.go         #   Name, Theme struct, Element types, inheritance resolver, registry
│                        #   Override type, WithOverrides — per-plot element overrides
│                        #   strip.text.x/y, strip.background.x/y inheritance paths
│   ├── elements.go      #   ElementText, ElementLine, ElementRect, ElementBlank, Merge functions
│   ├── color_defaults.go#   Per-theme colormap defaults (discrete, sequential, diverging, cyclic)
│   ├── ggplot.go        #   Ggplot (matplotlib) + Default alias → Dashboard
│   ├── modern.go        #   ObservableDark, Dashboard, Quartz, Air, Ink
│   ├── seaborn.go       #   Seaborn family (darkgrid, whitegrid, dark, white, ticks, palettes)
│   └── …               #   Classic, BW, Nord, Dracula, Gruvbox, GitHub, Cyberpunk, …
│
├── guide/               # Axis, legend, grid, and color bar rendering
│   └── guide.go         #   DrawXAxis, DrawYAxis, DrawGrid, DrawLegend, DrawColorBar
│
├── colormap/            # Color palettes: Viridis, ColorBrewer, Tab10, Observable10,
│                        #   discrete/continuous/manual/cyclic scales
│
├── canvas/              # Drawing seam — the path-level Canvas API and its backends
│   ├── canvas.go        #   Canvas interface: MoveTo, LineTo, Stroke, Fill, DrawImage, …
│   ├── raster.go        #   RasterCanvas (gg-backed; GPU or CPU rasterizer)   [was gg.go]
│   ├── recording.go     #   RecordingCanvas (records ops; replayed to vector)
│   └── export_*.go      #   SVG / PDF vector backends (replay a Recording)
│
├── output/              # Output layer — destinations for a Figure. See docs/OUTPUT-SPEC.md
│   ├── output.go        #   Figure, Source, Sizer, Surface, LiveSurface, Event, Render
│   ├── viewport.go      #   PanelInfo, Measurable, Zoomable — data-space viewport interfaces
│   ├── controller_dataspace.go # DataSpaceController — data-space pan/zoom (default for window)
│   ├── registry.go      #   blank-import surface registry: NewSurface / NewLiveSurface
│   ├── session.go       #   Session + Controller — interaction loop for live surfaces
│   ├── file/            #   FileSurface — PNG / SVG / PDF on disk
│   ├── image/           #   BufferSurface — in-memory image.Image
│   ├── window/          #   WindowSurface — desktop GPU window      (//go:build !js)
│   └── web/             #   CanvasSurface — browser <canvas>        (//go:build js && wasm)
│
├── dataset/             # Columnar data abstraction, see docs/DATASET.md
│   ├── memory/          #   Native Go slices engine + stat kernels (LOESS, KDE, bin, boxplot)
│   ├── arrow/           #   Apache Arrow IPC/Parquet engine + stat kernels
│   └── bigquery/        #   BigQuery SQL pushdown engine + native SQL stat kernels
│
├── internal/
│   ├── color/           # Color palette helpers
│   └── fonts/           # Embedded fonts (Go Noto Sans), system font registry
│
├── examples/
│   ├── geometries/      # One self-contained example per geometry type (23 geoms)
│   ├── showcase/        # Clifford attractor, butterfly curve
│   ├── phase2_geometries/
│   ├── phase2_statistics/
│   ├── phase2_scales/
│   └── phase2_features/
│
└── docs/
    ├── ARCHITECTURE.md  # (this file)
    ├── OUTPUT-SPEC.md   # Output layer specification (Figure, Surface, Session)
    ├── ROADMAP.md       # Development plan aligned with ggplot2-book (3e)
    ├── DATASET.md       # Deep dive on dataset/engine architecture
    └── BENCHMARK.md     # Benchmark results (Arrow vs Memory, SIMD)
```

---

## Rendering Pipeline

The pipeline runs across `Plot.Build()` and `Built.Draw()`:

### 1. Build Phase (`Plot.Build`)

#### 1a. Facet Split

```go
panels, _ := p.spec.Facet.Split(ctx, p.spec.Dataset)
rows, cols := p.spec.Facet.GridDims(len(panels))
```

The `Facet` interface produces `[]facet.Panel`, each containing a lazy-filtered `Dataset`,
a display label, and grid metadata (`RowVal`, `ColVal`, `NumRows`, `IsMargin`).
For `FacetNone`, this is a single panel with the full dataset.

**Labellers** (`LabelValue`, `LabelBoth`, `LabelContext`, custom `Label(fn)`) control
strip text formatting. `LabelContext` is the sentinel default — resolves to `LabelValue`
for Wrap and `LabelBoth` for Grid.

**Drop** (`Drop(false)`) preserves empty panels for missing value combinations.
Emptiness is checked via `maskHasTrue(mask)` without materializing data.

**Margins** (`GridMargins(true)`) produces extra panels: row margins (across all columns),
column margins (across all rows), and the corner margin (full dataset). Margin panels
have `IsMargin=true` and use `MarginLabel` ("All") as their `RowVal`/`ColVal`.

**Lazy pipeline**: `Split()` never calls `Collect`. Datasets are filtered via
`dataset.Filter(mask)` which is lazy. `Panel.NumRows` is computed from `maskCount(mask)`.
Materialization happens once per panel in `buildPanel` (step 1c below).

#### 1b. Grid Placement

```go
rowIndex[panel.RowVal] → grid row
colIndex[panel.ColVal] → grid column
```

For Grid facets, `PanelLayout.Row`/`.Col` are computed from `RowVal`/`ColVal` index maps
(preserving insertion order, with `"All"` margins at the end). For Wrap/None, fallback
to sequential `pi / cols`, `pi % cols`.

#### 1c. PANEL Column Injection

A `ColPANEL` system column (constant int64 = panel index) is lazily chained via
`WithColumn`, then the entire filter + PANEL chain is materialized with a single `Collect`.

#### 1d. Panel-Parallel Build

For multi-panel plots, each panel's data pipeline runs concurrently via `errgroup.Group`.
A single-panel fast path avoids goroutine overhead for simple plots.

Per panel:

- **Stat Transform + Group Splitting** — each layer's `stat.Transform` pipeline is applied
  via `Apply(ctx, TransformInput)`. If a colour/group aesthetic is present, the dataset is
  split by distinct values before stat computation, producing one resolved layer per group.

- **Scale Training** — X and Y scales are trained on resolved data columns. For discrete X
  data, a `DiscreteScale` is used with automatic string→float64 position mapping. For boxplots,
  the Y scale additionally trains on `lower`, `q1`, `middle`, `q3`, `upper`, `notch_lower`,
  `notch_upper`. Padding is applied: 5% of range for continuous, plus extra half-bin for
  bars/histograms/crossbars/violins/dotplots (computed from distinct X positions, not raw row count).

### 2. Draw Phase (`Built.Draw`)

#### 2a. Layout (Margins)

Margins are computed from:
- Y-axis tick label widths (measured with `cv.MeasureString`)
- Title / subtitle / axis labels / caption heights
- Legend width (for right/left) or height (for top/bottom)

**For multi-panel facets**, margins are cached from the first panel and reused
for all subsequent panels to ensure consistent cell alignment.

#### 2b. Chrome Rendering (Sequential)

Grid lines, panel background, borders, axes, strip labels, and legends are drawn
sequentially in absolute coordinates using the theme's element hierarchy.

Axes are drawn conditionally:
- **Y axis**: only on the leftmost column (`col == 0`)
- **X axis**: only on the bottom row (`row == rows-1`)

#### 2c. Data Layer Rendering (Parallel)

Each panel's data layers are rendered to an independent CPU `RasterCanvas` sub-canvas
concurrently via `errgroup.Group`. The sub-canvas is then composited onto the
main canvas via `DrawImage`.

Each resolved layer is dispatched to the appropriate `draw*` function in `drawer.go`
based on `geom.Type`.

#### 2d. Title, Subtitle, Caption

Drawn **once** after the panel loop, centered on the full canvas width.

---

## Concurrency Model

```
Plot.Build()
├── Single-panel fast path (no goroutines)
└── Multi-panel:
    └── errgroup.Group
        ├── panel[0]: stat → scale → resolve
        ├── panel[1]: stat → scale → resolve
        └── panel[N]: stat → scale → resolve

Built.Draw()
├── Chrome: sequential (axes, grid, labels, legend)
└── Data layers:
    └── errgroup.Group
        ├── panel[0]: sub-canvas → draw layers → DrawImage
        ├── panel[1]: sub-canvas → draw layers → DrawImage
        └── panel[N]: sub-canvas → draw layers → DrawImage
```

All errors are wrapped in `*ggplot.Error{Phase, Layer, Stage, Cause}` with
`errors.Is` / `errors.As` / `Unwrap` support.

---

## Key Design Decisions

### Immutable Builder Pattern

Every `Plot` method (`Layer`, `Labs`, `Theme`, `FacetWrap`, etc.) calls `clone()`
which deep-copies the `PlotSpec`. This prevents mutation surprises when building
multiple plots from a shared base.

### Dataset Abstraction

Data storage, memory layout, and SIMD computation are isolated inside the `dataset` package.
`ggplot` interacts exclusively with the engine-agnostic `dataset.Table` interface.

For an in-depth look at how the execution engines (Memory, Arrow, BigQuery) operate,
see [**DATASET.md**](DATASET.md).

### Scale Training + Resolution

Scales implement:
```go
type Scale interface {
    Train(col AnyColumn) error // expand domain to include data
    Map(v float64) float64     // data → [0,1]
    Inverse(v float64) float64 // [0,1] → data
    Ticks(n int) []float64     // nice tick positions in data-space
    Format(v float64) string   // tick label text
    Bounds() (float64, float64)
}
```

`scale.Resolve(name)` maps string names (`"log10"`, `"sqrt"`, `"reverse"`) to
concrete scale types, used by the `.ScaleX()` / `.ScaleY()` builder API.

### Stat Transform Pipeline

Stats implement the composable `stat.Transform` interface:

```go
type Transform interface {
    Name() string
    Apply(ctx context.Context, input TransformInput) (*TransformOutput, error)
    OutputMapping() map[string]string
    OutputSchema() map[string]ColumnType
    OutputHints() map[string]ChannelHint
}
```

Pipeline constructors like `geom.RectY([]stat.Transform{stat.BinX(), stat.Count()}, opts...)`
compose transforms into a full grammar-of-graphics data pipeline. Sugar constructors
like `geom.Histogram()` wrap common pipelines for convenience.

### Theme System

Themes use sealed element types with a ggplot2-style inheritance hierarchy:

```go
type Theme struct {
    Name     string
    Palette  []color.Color
    Spacing  Spacing
    Geom     GeomDefaults
    Elements map[string]Element  // "axis.title.x" → ElementText{...}
}
```

Elements inherit through a tree: `axis.title.x` ← `axis.title` ← `text`.
Zero-value fields merge with parent values via `MergeText()` / `MergeLine()` / `MergeRect()`.

60+ themes are registered at init-time. `theme.Resolve(name)` instantiates on demand.
Default theme is **Dashboard** (clean card-style with Tab10/Blues palette).

### Typed Error Envelope

All pipeline errors wrap `*ggplot.Error`:

```go
type Error struct {
    Phase Phase   // PhaseBuild, PhaseDraw, PhaseRender
    Layer int     // 0-based layer index (-1 if not layer-specific)
    Stage string  // "transform", "scale", "theme", …
    Cause error   // underlying error
}
```

### Canvas Abstraction (the drawing seam)

The `canvas` package is the **front of the rendering pipe** — the path-level `Canvas`
interface every `Figure` draws through, decoupled from any specific backend:

- **`RasterCanvas`** — gg-backed pixel backend. `NewRasterCanvas` uses GPU acceleration
  when available; `NewRasterCanvasCPU` is a pure-CPU analytic rasterizer with deterministic
  output. `RasterFromContext` borrows an externally-owned `gg.Context` — used by the desktop
  window surface for a zero-copy handoff; the borrower must not `Close()` it.
- **`RecordingCanvas`** — records draw operations for replay into true vector SVG/PDF.

`ggplot.WithCPU()` forces the CPU rasterizer. `DrawImage` enables parallel sub-canvas
compositing. `RasterCanvas` was previously named `GGCanvas` (and its file `gg.go` → `raster.go`).

### Output Layer

> **Status:** Phases 1–5 of [OUTPUT-SPEC.md](OUTPUT-SPEC.md) §11 are implemented (unreleased).
> Done: the `GGCanvas → RasterCanvas` rename (Phase 1); the `output` core —
> `Figure`, `Source`, `Sizer`, `Surface`, `LiveSurface`, `Imager`, `Event`, `Render`, and the
> blank-import registry (Phase 2); the `output/file` and `output/image` surfaces with
> `Plot.Save`/`Encode`/`Image` and `Built.RenderTo` as façades over `Render` (Phase 3); the
> `Session`/`Controller` interaction loop with fast-path viewport redraw and slow-path rebuild,
> tested headless with a scripted `LiveSurface`, plus async/debounced rebuild via
> `WithRebuildDelay`/`WithRebuildError` and a runnable headless example (`examples/session/`)
> (Phase 4); `output/window` (`//go:build !js`) — `window.Show` opening a `gogpu` window with
> zero-copy `ggcanvas` presentation, reusing the Phase-4 `Controller`/`State`, with a runnable
> example (`examples/window/`); runtime-verified on a desktop GPU (Phase 5). `Plot.Build` now
> returns `output.Figure` (concretely `*Built`). Still pending: `output/web` (Phase 6, wasm).
> Note `window.Show` drives gogpu's callback loop directly (gogpu owns the run loop) rather than
> calling `Session.Run`.

The `output` package is the **back of the rendering pipe** — it carries a `Figure` to a
destination. One `Surface` abstraction serves four destinations:

| Surface | Package | Destination |
|---|---|---|
| `FileSurface` | `output/file` | PNG / SVG / PDF on disk |
| `BufferSurface` | `output/image` | in-memory `image.Image` |
| `WindowSurface` | `output/window` (`!js`) | desktop GPU window — zero-copy via `gg/integration/ggcanvas` |
| `CanvasSurface` | `output/web` (`js && wasm`) | browser `<canvas>` over the `wgpu` browser backend |

`Surface` uses an `Acquire`/`Commit` frame model; `LiveSurface` adds an event stream for the
interactive (window, web) targets. `output.Render(ctx, fig, surf)` presents one frame — the
entire static story. `Session` + `Controller` drive the interactive loop with a **fast path**
(redraw the current `Figure` with updated data-space viewport via `Zoomable` — O(1), no rebuild)
and a **slow path** (rebuild from the `Source` when an interaction crosses the trained data
extent — scales retrain, stats recompute). The default `DataSpaceController` provides data-space
pan (drag) and zoom (scroll wheel) that mutates scale bounds directly on the built figure via
`Measurable` + `Zoomable` interfaces; axes stay at fixed screen positions.

Platform selection is by **blank import**: each platform subpackage's `init()` registers its
surface, and `output.NewSurface(ctx, "window", …)` resolves it — no build tags appear in user
code. Build constraints are confined to the platform leaf files. `Plot.Save` / `Encode` /
`Image` are façades over `Render`; `window.Show` / `web.Mount` drive a `Session`. The full
design — interfaces, the four surfaces, migration phasing — is in
[**OUTPUT-SPEC.md**](OUTPUT-SPEC.md).

---

## Dependencies

| Dependency | Role |
|---|---|
| `gogpu/gg` | 2D vector rendering (anti-aliased lines, fills, text) |
| `apache/arrow-go` | Zero-copy columnar data (optional, for `dataset/arrow`) |
| `golang.org/x/sync` | `errgroup` for panel-parallel build and draw |
| `golang.org/x/image` | Font rendering support |
| `go-text/typesetting` | Text shaping (indirect, via gg) |
| `gg/integration/ggcanvas` | Zero-copy gg → GPU-surface presentation (optional, `output/window`) |
| `gogpu/gogpu`, `gogpu/gpucontext` | Desktop window + GPU device provider (optional, `output/window`) |
| `gogpu/wgpu` | WebGPU — browser backend for `output/web` (optional, `js/wasm`) |
