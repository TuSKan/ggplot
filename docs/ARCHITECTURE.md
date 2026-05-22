# Architecture

> Go implementation of the Grammar of Graphics — declarative, composable data visualization.

---

## Overview

`ggplot` follows an immutable builder pattern where every method returns a new `Plot` value
(via internal `clone()`). The rendering pipeline is a strict sequence:

```
PlotSpec → Facet Split → Stat Transform → Scale Training → Layout → Render
```

**Build** and **Draw** are separate phases. `Plot.Build()` produces a `Built` value containing
all resolved layers, trained scales, and layout geometry. `Built.Draw()` renders data layers
and chrome onto a canvas. Both phases execute **panel-parallel** via `errgroup` for
multi-panel faceted plots.

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
├── drawer.go            # Geometry drawing: drawPoint, drawLine, drawBar, drawBoxplot, …
├── spec.go              # PlotSpec, Labels, RenderOpt, WithCPU()
├── util.go              # Shared helpers (normalize, resolveColor, …)
│
├── aes/                 # Aesthetic mapping constructors: X(), Y(), Color(), Fill(), Size(),
│                        #   Label(), XEnd(), YEnd(), YMin(), YMax(), Group(), …
├── geom/                # Geometry layer definitions, options, position adjustments
│   ├── geom.go          #   Point, Line, Step, Bar, Col, Histogram, Area, Density, Rug,
│   │                    #   HLine, VLine, Text, BoxPlot, Smooth, Tile, Segment, ErrorBar,
│   │                    #   Polygon, Ribbon, Difference + functional options
│   ├── position.go      #   Pos interface: Identity, Dodge, Stack, Jitter, Nudge
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
│   └── summary.go       #   Summary
│
├── scale/               # Scale types + Resolve factory
│   └── scale.go         #   Linear, Log10, Sqrt, Reverse, Discrete + NiceSequence, FormatNumber
│
├── coord/               # Coordinate systems
│   └── coord.go         #   Cartesian, Polar
│
├── facet/               # Faceting strategies
│   └── facet.go         #   None, Wrap (NCols/NRows), Grid (row ~ col)
│
├── theme/               # Theme definitions (60+ built-in themes)
│   ├── theme.go         #   Name, Theme struct, Element types, inheritance resolver, registry
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
├── canvas/              # Canvas abstraction wrapping gogpu/gg
│   ├── canvas.go        #   Canvas interface: MoveTo, LineTo, Stroke, Fill, DrawImage, …
│   └── gg.go            #   GGCanvas (GPU/CPU auto), GGCanvasCPU (pure-CPU rasterizer)
│
├── output/              # Output format helpers (SVG, PDF via RecordingCanvas)
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
│   ├── geometries/      # One self-contained example per geometry type (17 geoms)
│   ├── showcase/        # Clifford attractor, butterfly curve
│   ├── phase2_geometries/
│   ├── phase2_statistics/
│   ├── phase2_scales/
│   └── phase2_features/
│
└── docs/
    ├── ARCHITECTURE.md  # (this file)
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
panels, _ := p.spec.Facet.Split(p.spec.Dataset)
rows, cols := p.spec.Facet.GridDims(len(panels))
```

The `Facet` interface produces `[]FacetPanel`, each containing a filtered `Dataset`
and a display label. For `FacetNone`, this is a single panel with the full dataset.

#### 1b. Panel-Parallel Build

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
  bars/histograms.

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

Each panel's data layers are rendered to an independent `GGCanvasCPU` sub-canvas
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

### Canvas Abstraction

The `canvas` package wraps `gogpu/gg` contexts:

- **`GGCanvas`** — default, uses GPU acceleration when available
- **`GGCanvasCPU`** — pure-CPU analytic rasterizer, deterministic output

`ggplot.WithCPU()` forces CPU rendering in `Save()` / `WriteTo()`.
`DrawImage` on `GGCanvas` enables parallel sub-canvas compositing.

---

## Dependencies

| Dependency | Role |
|---|---|
| `gogpu/gg` | 2D vector rendering (anti-aliased lines, fills, text) |
| `apache/arrow-go` | Zero-copy columnar data (optional, for `dataset/arrow`) |
| `golang.org/x/sync` | `errgroup` for panel-parallel build and draw |
| `golang.org/x/image` | Font rendering support |
| `go-text/typesetting` | Text shaping (indirect, via gg) |
