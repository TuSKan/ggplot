# Architecture

> Go implementation of the Grammar of Graphics — declarative, composable data visualization.

---

## Overview

`ggplot` follows an immutable builder pattern where every method returns a new `Plot` value
(via internal `clone()`). The rendering pipeline is a strict sequence:

```
PlotSpec → Facet Split → Stat Transform → Scale Training → Layout → Render
```

All data flows through the `dataset.Dataset` abstraction, which provides a columnar,
iterator-based interface backed by either native Go slices or Apache Arrow arrays.

---

## Package Map

```
github.com/TuSKan/ggplot
│
├── ggplot.go            # Plot builder API, rendering pipeline orchestrator
├── draw.go              # Geometry drawing (drawPoint, drawLine, drawBar, drawBoxplot, …)
├── util.go              # Shared helpers (normalize, resolveColor, …)
│
├── aes/                 # Aesthetic mapping constructors: X(), Y(), Color(), Fill(), Size(), …
├── geom/                # Geometry layer definitions and option types
│   └── geom.go          #   Point, Line, Step, Bar, Histogram, Area, Density, Rug,
│                        #   HLine, VLine, Text, BoxPlot, Smooth + functional options
├── stat/                # Statistical transformations
│   └── stat.go          #   Identity, Bin, Density (KDE), Smooth (LOESS), Summary, BoxPlot
├── scale/               # Scale types + Resolve factory
│   └── scale.go         #   Linear, Log10, Sqrt, Reverse, Discrete + NiceSequence, FormatNumber
├── coord/               # Coordinate systems
│   └── coord.go         #   Cartesian, Flipped (CoordFlip)
├── facet/               # Faceting strategies
│   └── facet.go         #   None, Wrap (NCols/NRows), Grid (row ~ col)
├── theme/               # Theme definitions
│   └── theme.go         #   Default, Classic, Minimal, Dark, BW
├── guide/               # Axis, legend, grid, and color bar rendering
│   └── guide.go         #   DrawXAxis, DrawYAxis, DrawGrid, DrawLegend, DrawColorBar
├── position/            # Position adjustments (dodge, stack — scaffolded)
├── output/              # Output format helpers (SVG — scaffolded)
│
├── dataset/             # Columnar data abstraction, see docs/DATASET.md
│
├── internal/
│   ├── canvas/          # Canvas abstraction wrapping gogpu/gg
│   ├── color/           # Color palette helpers
│   ├── fonts/           # Embedded fonts (Go Noto Sans)
│   └── grammar/         # PlotSpec, Labels, AesMap, ScaleOverride — internal data types
│
├── examples/            # Runnable examples organized by roadmap phase
│   ├── phase2_geometries/
│   ├── phase2_statistics/
│   ├── phase2_scales/
│   ├── phase2_features/
│   ├── showcase/
│   ├── clifford/
│   ├── butterfly/
│   └── …
│
└── docs/
    ├── ARCHITECTURE.md  # (this file)
    └── ROADMAP.md       # 19-phase development plan aligned with ggplot2-book (3e)
```

---

## Rendering Pipeline

The pipeline runs inside `renderTo()` in `ggplot.go`:

### 1. Facet Split

```go
panels, _ := p.spec.Facet.Split(p.spec.Dataset)
rows, cols := p.spec.Facet.GridDims(len(panels))
```

The `Facet` interface produces `[]FacetPanel`, each containing a filtered `Dataset`
and a display label. For `FacetNone`, this is a single panel with the full dataset.

### 2. Per-Panel Processing (inside the panel loop)

For each panel:

#### 2a. Stat Transform + Group Splitting

Each layer's stat is applied via `stat.Lookup(name).Compute(ds, mapping)`.
If a colour/group aesthetic is present, the dataset is split by distinct values
before stat computation, producing one resolved layer per group.

#### 2b. Scale Training

X and Y scales are trained on resolved data columns. For discrete X data,
a `DiscreteScale` is used with automatic string→float64 position mapping.
For boxplots, the Y scale additionally trains on `lower`, `q1`, `middle`, `q3`, `upper`.

Padding is applied: 5% of range for continuous, plus extra half-bin for bars/histograms.

#### 2c. Layout (Margins)

Margins are computed from:
- Y-axis tick label widths (measured with `cv.MeasureString`)
- Title / subtitle / axis labels / caption heights
- Legend width (for right/left) or height (for top/bottom)

**For multi-panel facets**, margins are cached from the first panel and reused
for all subsequent panels to ensure consistent cell alignment.

#### 2d. Grid, Panel Background, and Borders

Grid lines, panel background, and borders are drawn in absolute coordinates
using the theme's styling (dash patterns, colors, widths).

#### 2e. Data Layer Rendering

The canvas is translated and clipped to the panel bounds. Each resolved layer
is dispatched to the appropriate `draw*` function in `draw.go` based on `geom.Type`.

Axes are drawn conditionally:
- **Y axis**: only on the leftmost column (`col == 0`)
- **X axis**: only on the bottom row (`row == rows-1`)

### 3. Title, Subtitle, Caption

Drawn **once** after the panel loop, centered on the full canvas width.

---

## Key Design Decisions

### Immutable Builder Pattern

Every `Plot` method (`Layer`, `Labs`, `Theme`, `FacetWrap`, etc.) calls `clone()`
which deep-copies the `PlotSpec`. This prevents mutation surprises when building
multiple plots from a shared base.

### Dataset Abstraction

Data storage, memory layout, and SIMD computation are isolated inside the `dataset` package. `ggplot` interacts exclusively with the engine-agnostic `dataset.Table` interface. 

For an in-depth look at how the execution engines (Memory, Arrow, BigQuery) operate, see [**DATASET.md**](DATASET.md).

### Scale Training + Resolution

Scales implement:
```go
type Scale interface {
    Train(col Column) error    // expand domain to include data
    Map(v float64) float64     // data → [0,1]
    Inverse(v float64) float64 // [0,1] → data
    Ticks(n int) []float64     // nice tick positions in data-space
    Format(v float64) string   // tick label text
    Bounds() (float64, float64)
}
```

`scale.Resolve(name)` maps string names (`"log10"`, `"sqrt"`, `"reverse"`) to
concrete scale types, used by the `.ScaleX()` / `.ScaleY()` builder API.

### Stat Registry

Stats are registered at init-time and looked up by name:
```go
stat.Register(binStat{})      // "bin"
stat.Register(smoothStat{})   // "smooth" — LOESS with tri-cube kernel
stat.Register(boxplotStat{})  // "boxplot" — 5-number summary with 1.5×IQR fences
```

Each geom declares its default stat via `geom.Layer.StatName`.

### Theme System

Themes are pure data (`theme.Theme` struct) with no behavior — colors, font sizes,
line widths, dash patterns. `theme.Resolve(name)` returns one of 5 built-in themes.

---

## Dependencies

| Dependency | Role |
|---|---|
| `gogpu/gg` | 2D vector rendering (anti-aliased lines, fills, text) |
| `apache/arrow-go` | Zero-copy columnar data (optional, for `dataset/arrow`) |
| `golang.org/x/image` | Font rendering support |
| `go-text/typesetting` | Text shaping (indirect, via gg) |
