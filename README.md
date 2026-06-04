# ggplot

[![Go Reference](https://pkg.go.dev/badge/github.com/TuSKan/ggplot.svg)](https://pkg.go.dev/github.com/TuSKan/ggplot)
[![Go Report Card](https://goreportcard.com/badge/github.com/TuSKan/ggplot)](https://goreportcard.com/report/github.com/TuSKan/ggplot)
[![CI](https://github.com/TuSKan/ggplot/actions/workflows/ci.yml/badge.svg)](https://github.com/TuSKan/ggplot/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/TuSKan/ggplot/branch/main/graph/badge.svg)](https://codecov.io/gh/TuSKan/ggplot)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/TuSKan/ggplot)](https://github.com/TuSKan/ggplot/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

![ggplot](assets/cover.jpg)

**Production-grade Grammar of Graphics for Go.**

A pure-Go data visualization library implementing a rigorous, declarative Grammar of Graphics pipeline. Inspired by Hadley Wickham's renowned [ggplot2](https://ggplot2-book.org/), but architected specifically for Go's type safety and interface-driven engine architecture.

## Overview

`ggplot` provides an expressive, composable API for generating complex data visualizations. It decouples data manipulation (Apache Arrow, BigQuery, Memory) from statistical transformations and final rendering, resulting in highly scalable plotting pipelines.

| Capability | Supported Features |
|---|---|
| **Geometries** | Point, Line, Step, Bar, Col, Histogram, Area, Density, Polygon, Rug, HLine, VLine, Segment, Text, BoxPlot, Smooth, Tile, ErrorBar, Crossbar, Linerange, Pointrange, Curve, Violin, Dotplot, Raster, JitterPoint |
| **Statistics** | Identity, Bin/Count, Density (KDE), Smooth (LOESS + lm), Summary, BoxPlot (Tukey/range whiskers, notch CI), ViolinY (grouped KDE), DotBin (Wilkinson stacking) |
| **Aesthetics** | Size, Alpha, Shape, Linetype — per-point and per-group mapping |
| **Scales** | Linear, Log10, Sqrt, Reverse, Discrete, DateTime, Binned, Size, Alpha, Shape, Linetype, Identity |
| **Color Palettes** | 60+ built-in palettes — Viridis, ColorBrewer, Tab10, Observable, Seaborn, and more |
| **Faceting** | Grid (row ~ col, margins, drop), Wrap (NCols, drop), Labellers (Value, Both, Context, custom) |
| **Data Backends** | Native Memory, Apache Arrow IPC/Parquet, BigQuery SQL pushdown |
| **Data Types** | Float64, Int64, String, Bool, Timestamp, Date, Time |
| **Output** | PNG, SVG 1.1 (responsive, metadata: tooltips/links/ARIA), PDF 1.4, HiDPI via `WithScale()`; interactive desktop window (GPU scene graph rendering) + headless `Session` loop |
| **Theming** | 60+ themes — Dashboard, Dark, Classic, Minimal, Observable, Seaborn, Nord, Dracula, and more |
| **Annotations** | `AnnotateText`, `AnnotateRect`, `AnnotateSegment`, `AnnotateArrow`, `AnnotateLabel` — layer-less fixed-coordinate annotations |
| **Coordinate Systems** | `CoordCartesian` (viewport zoom), `CoordFixed` (aspect ratio), `CoordFlip`, `Coord(Polar())` |

---

## What's New in v0.0.12

### Theme System Granularity

Full theme-aware legend rendering and plot margin control with physical units:

```go
// Plot margins with physical units — cm, inches, points, or lines.
ggplot.New(ds, aes.X("x"), aes.Y("y")).
    Layer(geom.Line()).
    PlotMargin(theme.PlotMargin{
        Top:    theme.Inches(0.5),
        Right:  theme.Cm(0.8),
        Bottom: theme.Pt(20),
        Left:   theme.Cm(1.0),
    }).
    Save(ctx, "margins.png", 800, 500)
```

### Block-Level Alignment

Configurable alignment for titles, labels, caption, and legend — matching ggplot2:

```go
// Left-aligned titles, right-aligned caption, center-aligned legend.
ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
    Layer(geom.Point()).
    Align(theme.BlockAlignment{
        Title:   theme.AlignLeft,
        Caption: theme.AlignRight,   // default
        XLabel:  theme.AlignCenter,  // default
        Legend:  theme.AlignCenter,
    }).
    Save(ctx, "aligned.png", 800, 500)
```

### Continuous Legends

Graduated-size circles and alpha gradient strips, populated automatically from `aes.Size` / `aes.Alpha` mappings:

```go
ggplot.New(ds, aes.X("lon"), aes.Y("lat")).
    Layer(geom.Point(), aes.Size("population"), aes.Alpha("confidence")).
    ScaleSizeArea().
    ScaleAlpha(0.15, 1.0).
    Save(ctx, "bubbles.png", 800, 500)
```

### Legend Theme Overrides

Per-plot legend styling without creating a custom theme:

```go
ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
    Layer(geom.Point(geom.WithSize(5))).
    ThemeOverride(
        theme.LegendTitleOverride(theme.ElementText{Bold: true, Size: 13}),
        theme.LegendBackgroundOverride(theme.ElementRect{Fill: color.RGBA{240, 240, 240, 255}}),
    ).
    Save(ctx, "styled_legend.png", 800, 500)
```

### ⚠️ Breaking Changes

- `Labs()` → `Labels()` — the Plot builder method for configuring labels.
- `LabOpt` → `LabelOpt` — the functional option type.
- `XLab()` → `XLabel()`, `YLab()` → `YLabel()` — axis label constructors.

<details>
<summary><strong>What was new in v0.0.11</strong></summary>

### Browser WASM Surface

Render plots in the browser via WebAssembly — CPU raster, SVG, or WebGPU modes:

```go
import "github.com/TuSKan/ggplot/output/web"

// Mounts into <div id="plot-container">.
// Drag to pan, scroll to zoom — same DataSpaceController as the desktop window.
_ = web.Mount(ctx, plot, "plot-container")
```

</details>

<details>
<summary><strong>What was new in v0.0.10</strong></summary>

### Data-Space Interactive Pan/Zoom

Pan (drag) and zoom (scroll wheel) operate on data-space scale bounds — axes stay at fixed screen positions while tick labels update dynamically:

```go
import "github.com/TuSKan/ggplot/output/window"

// Opens a GPU window. Drag to pan, scroll to zoom.
// Double-click to reset. Per-panel zoom for faceted plots.
_ = window.Show(ctx, plot, window.WithTitle("ggplot"), window.WithSize(900, 600))
```

### Output Layer

Unified destination layer separating Build → Draw → Surface:

- **`output.Figure`** / **`output.Source`** interfaces — `*Built` implements `Figure`, `*Plot` implements `Source`.
- **`output/file`** — `file.Save(ctx, plot, "out.png", 800, 500)` replaces `Plot.Save`.
- **`output/image`** — `image.Render(ctx, plot, 800, 500)` for in-memory images.
- **`output/window`** — GPU desktop window with `DataSpaceController`.
- **`output.Session`** — event loop for any `LiveSurface`. `WithRebuildDelay` makes rebuilds async/debounced.

### ⚠️ Breaking Changes

- `Plot.Save`, `Plot.Encode`, `Plot.Image`, `Plot.WriteTo` → **removed**. Use `file.Save` / `file.Encode` / `image.Render`.
- `Built.Save`, `Built.WriteTo` → **removed**. Use `output.Render` + a surface.
- `Plot.Build` returns `output.Figure` (concretely `*Built`), not `*Built` directly.

</details>

<details>
<summary><strong>What was new in v0.0.9</strong></summary>

### Advanced Faceting

Free scales, grid margins, custom labellers, and strip placement:

```go
ggplot.New(ds, aes.X("x"), aes.Y("y")).
    Layer(geom.Point()).
    FacetWrap("species", facet.FreeY()).   // independent Y range per panel
    Save(ctx, "facets.png", 900, 300)

ggplot.New(ds, aes.X("x"), aes.Y("y")).
    Layer(geom.Point()).
    FacetGrid("year", "site", facet.GridMargins(true)).  // row/col/corner "All" panels
    Save(ctx, "grid.png", 900, 600)
```

Labellers (`facet.LabelValue`, `LabelBoth`, `LabelContext`, `Label(fn)`), `facet.Drop(false)` to keep empty panels, `facet.GridSpace("free_y")` for proportional sizing, and `facet.StripBottom()` / `GridStripLeft()` for strip repositioning.

### Secondary Axes

Dual Y-axis derived from the primary via a transform pair, or a mirrored duplicate:

```go
ggplot.New(ds, aes.X("hour"), aes.Y("temp_c")).
    Layer(geom.Point(geom.WithColor("steelblue"))).
    SecondAxis(scale.SecAxis(
        func(c float64) float64 { return c*9/5 + 32 },       // °C → °F
        func(f float64) float64 { return (f - 32) * 5 / 9 }, // °F → °C
        "Temperature (°F)",
    )).
    Save(ctx, "dual_axis.png", 800, 500)
```

### Theme Overrides

Per-plot element overrides without authoring a custom theme:

```go
ggplot.New(ds, aes.X("x"), aes.Y("y")).
    Layer(geom.Point()).
    Theme("default").
    ThemeOverride(
        theme.StripTextOverride(theme.ElementText{Bold: true, Size: 13}),
    ).
    Save(ctx, "styled.png", 800, 500)
```

### New Geometries & Coordinate Systems

`geom.Violin`, `geom.Dotplot`, `geom.Raster`, `geom.Curve`, `geom.Crossbar`, `geom.Linerange`, `geom.Pointrange`, and `geom.JitterPoint`. Coordinate systems: `CoordCartesian` (viewport zoom), `CoordFixed` (aspect ratio), `CoordFlip`, and `Coord(coord.Polar(...))`. Coordinate transforms now dispatch to engine-native math kernels (Arrow compute / SQL functions) instead of scalar fallbacks.

### ⚠️ Breaking Changes

- `Plot.FacetWrap(col string, opts ...WrapOpt)` — was `(col, nCols, nRows)`. Use `facet.NCols(n)`.
- `Plot.FacetGrid(row, col string, opts ...GridOpt)` — replaces the old two-arg form.
- `Plot.SecondaryY` → **`Plot.SecondAxis`**.
- `coord.TransFunc` is now a name-only spec type (math moved to engine `MathKernel`).

</details>

<details>
<summary><strong>What was new in v0.0.8</strong></summary>

### Aesthetics Mapping

Map data columns directly to visual properties — size, transparency, shape, and line style:

```go
ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Size("magnitude"), aes.Alpha("confidence")).
    Layer(geom.Point()).
    ScaleSizeArea().              // area-proportional sizing
    ScaleAlpha(0.2, 0.9).        // opacity range
    Save(ctx, "scatter.png", 800, 500)
```

### Shape & Linetype Scales

Categorical columns map to distinct shapes or dash patterns:

```go
ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Shape("species")).
    Layer(geom.Point(geom.WithSize(5))).
    Save(ctx, "shapes.png", 800, 500)

ggplot.New(ds, aes.X("date"), aes.Y("value"), aes.Linetype("model")).
    Layer(geom.Line()).
    Save(ctx, "linetypes.png", 800, 500)
```

10 built-in shapes: `circle`, `square`, `triangle`, `diamond`, `triangleDown`, `plus`, `cross`, `star`, `pentagon`, `hexagon`.
6 built-in linetypes: `solid`, `dashed`, `dotted`, `dotdash`, `longdash`, `twodash`.

### Shape Constants

All shape names are now exported constants (`canvas.ShapeCircle`, `canvas.ShapeStar`, etc.) — no more magic strings.

</details>

<details>
<summary><strong>What was new in v0.0.7</strong></summary>

- **CIELAB gradient constructors**: `colormap.Gradient()`, `Gradient2()`, `GradientN()` with perceptually uniform interpolation
- **Theme-aware colour defaults**: Separate Color/Fill palettes per theme
- **Legend key glyphs**: Circle for points, line for smooth, rectangle for bars
- **Guide customization**: `ColorBarWidth()`, `ColorBarNBin()`, `LegendCols()`
- **NA color**: `scale.SetNAColor()` for missing value display

</details>

<details>
<summary><strong>What was new in v0.0.6</strong></summary>

- **Temporal data types**: `DTypeTimestamp`, `DTypeDate`, `DTypeTime` across all engines
- **DateTime scale**: `scale.DateTime` with auto calendar-aligned ticks
- **Binned scale**: `scale.Binned` with `WithBins(n)` and `WithBinBreaks(edges)`
- **Out-of-bounds policies**: `scale.WithOOB(OOBSquish | OOBCensor)`
- **Opt-in drivers**: GPU, CSV, Parquet as blank-import packages

</details>

> Full changelog: [CHANGELOG.md](CHANGELOG.md)

---


## Gallery

| | |
|---|---|
| ![Clifford Attractor](assets/clifford.png) | ![Butterfly Curve](assets/butterfly.png) |
| *Clifford attractor — 500k points, alpha blending, continuous color scale* | *Butterfly curve — parametric path with color interpolation* |

### Geometries

| | | |
|---|---|---|
| ![Point](assets/point.png) | ![Line](assets/line.png) | ![Step](assets/step.png) |
| `geom.Point` | `geom.Line` | `geom.Step` |
| ![Bar](assets/bar.png) | ![Col](assets/col.png) | ![Histogram](assets/histogram.png) |
| `geom.Bar` | `geom.Col` | `geom.Histogram` |
| ![Area](assets/area.png) | ![Density](assets/density.png) | ![Smooth](assets/smooth.png) |
| `geom.Area` | `geom.Density` | `geom.Smooth` |
| ![Boxplot](assets/boxplot.png) | ![ErrorBar](assets/errorbar.png) | ![Rug](assets/rug.png) |
| `geom.Boxplot` | `geom.ErrorBar` | `geom.Rug` |
| ![Text](assets/text.png) | ![HLine + VLine](assets/hline_vline.png) | ![Segment](assets/segment.png) |
| `geom.Text` | `geom.HLine` / `geom.VLine` | `geom.Segment` |
| ![Polygon](assets/polygon.png) | ![Tile](assets/tile.png) | ![Crossbar](assets/crossbar.png) |
| `geom.Polygon` | `geom.Tile` | `geom.Crossbar` |
| ![Linerange](assets/linerange.png) | ![Pointrange](assets/pointrange.png) | ![Curve](assets/curve.png) |
| `geom.Linerange` | `geom.Pointrange` | `geom.Curve` |
| ![Violin](assets/violin.png) | ![Dotplot](assets/dotplot.png) | ![Raster](assets/raster.png) |
| `geom.Violin` | `geom.Dotplot` | `geom.Raster` |
| ![JitterPoint](assets/jitter_point.png) | | |
| `geom.JitterPoint` | | |

### Annotations

| |
|---|
| ![Annotations](assets/annotate.png) |
| `Plot.Annotate()` — text, rect, segment, arrow, label |

### Coordinate Systems

| | |
|---|---|
| ![CoordFixed](assets/coord_fixed.png) | ![CoordZoom](assets/coord_zoom.png) |
| `CoordFixed(1)` — circle stays circular | `CoordCartesian` — viewport zoom |

### Faceting & Advanced Axes

| | |
|---|---|
| ![Shared Scales](examples/facet_free_scales/01_shared_scales.png) | ![Free Y](examples/facet_free_scales/02_free_y.png) |
| `FacetWrap("species")` — shared scales | `FacetWrap("species", FreeY())` — independent Y per panel |
| ![Secondary Axis](examples/secondary_axis/01_secondary_axis_temp.png) | ![Dup Axis](examples/secondary_axis/02_dup_axis.png) |
| `SecondAxis(SecAxis(°C→°F))` — dual Y-axis | `SecondAxis(DupAxis("°C"))` — mirrored axis |

### Theme & Legends

| | |
|---|---|
| ![Plot Margins](examples/theme_legends/01_plot_margin_units.png) | ![Legend Theming](examples/theme_legends/02_legend_theming.png) |
| `PlotMargin` — physical units (cm, inches, pt) | `LegendTitleOverride` — per-plot legend styling |
| ![Size Legend](examples/theme_legends/03_size_legend.png) | ![Alpha Legend](examples/theme_legends/04_alpha_legend.png) |
| `aes.Size` — graduated-circle legend | `aes.Alpha` — gradient-strip legend |
| ![Combined](examples/theme_legends/05_combined_aesthetics.png) | |
| Size + Alpha + Color — three legend types stacked | |

> Each image is generated by a self-contained example in [`examples/`](examples/).

---

## Quick Start

### Installation

```bash
go get github.com/TuSKan/ggplot
```

### Scatter Plot with Smooth

```go
package main

import (
	"context"
	"log"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
)

func main() {
	ctx := context.Background()
	eng := memory.NewEngine(ctx)
	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
		eng.NewFloat64Column("y", []float64{2, 4, 5, 4, 6, 8, 7, 9, 10, 11}),
	)
	if err != nil {
		log.Fatalln(err)
	}

	ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(5), geom.WithColor("coral"))).
		Layer(geom.Smooth()).
		Labels(ggplot.Title("Quick Start"), ggplot.XLabel("X"), ggplot.YLabel("Y")).
		Save(ctx, "scatter.png", 800, 500)
}
```

### Faceted Time Series

```go
ggplot.New(ds, aes.X("day"), aes.Y("temp")).
    Layer(geom.Line(geom.WithColor("seagreen"), geom.WithLineWidth(1.5))).
    FacetWrap("season", facet.NCols(2)).
    Labels(ggplot.Title("Temperature by Season")).
    Theme("dark").
    Save(ctx, "facets.png", 900, 600)
```

### Box Plot with Groups

```go
ggplot.New(ds, aes.X("group"), aes.Y("value")).
    Layer(geom.Boxplot(geom.WithFill("#E8E8E8"), geom.WithAlpha(0.8))).
    Labels(ggplot.Title("Distribution by Group")).
    Theme("classic").
    Save(ctx, "boxplot.png", 800, 500)
```

### Interactive output (window & session)

A `*Plot` is an `output.Source` that an `output.Session` drives onto a `LiveSurface` — build → draw → event loop. The desktop backend wires this to a GPU window with data-space interactive zoom:

```go
import "github.com/TuSKan/ggplot/output/window"

// Opens a GPU window. Drag to pan, scroll to zoom — axes stay fixed,
// tick labels update dynamically. Double-click to reset.
// Operates on data-space scale bounds (O(1)), not canvas transforms.
// Call from the main goroutine. See examples/window.
_ = window.Show(ctx, plot, window.WithTitle("ggplot"), window.WithSize(900, 600))
```

The same `Session` loop runs headless against any `LiveSurface` — for tests, servers, or a custom frontend. `WithRebuildDelay` makes rebuilds asynchronous and debounced (the last good frame keeps drawing while the next is computed):

```go
sess := output.NewSession(plot, surface,
    output.WithRebuildDelay(30*time.Millisecond),
)
_ = sess.Run(ctx) // runs until the surface's events close. See examples/session.
```

For browser output, the same `Session` loop is driven by DOM events over WebAssembly. Build with `GOOS=js GOARCH=wasm`:

```go
import "github.com/TuSKan/ggplot/output/web"

// Mounts into a <div id="plot-container">. CPU rasterizer + Canvas2D putImageData (default).
// Drag to pan, scroll to zoom — same DataSpaceController as the desktop window.
// WithSVG() for vector output with native tooltips/links/ARIA from metadata channels.
_ = web.Mount(ctx, plot, "plot-container")
```

---

## Why `ggplot`?

- **Declarative composition** — Build complex charts by layering geometries and statistics instead of drawing pixels.
- **Provider-agnostic engines** — Swap out the dataset engine (`memory` → `arrow` → `bigquery`) without changing plotting code.
- **Publication-ready output** — Anti-aliased 2D rendering powered by `gogpu/gg`, saving to PNG, SVG, or PDF at configurable DPI scales.
- **60+ built-in themes** — From minimal dashboards to editorial typography to neon cyberpunk.
- **Parallel rendering** — Panel-parallel build and draw pipelines via `errgroup` for multi-facet plots.

---

## Architecture & Data Backends

`ggplot` is built around an interface-driven `dataset.Table` engine. You are not limited to `[]float64` slices — back your plots with robust columnar frameworks. See [**DATASET.md**](docs/DATASET.md) for a deep-dive.

- **Memory Engine (`dataset/memory`)** — Lightweight native Go slices. Best for standard web-server rendering.
- **Arrow Engine (`dataset/arrow`)** — Apache Arrow backed IPC streams and Parquet datasets. Zero-copy reads. Best for datasets >1M rows.
- **BigQuery Engine (`dataset/bigquery`)** — Lazy SQL pushdown execution. Best for massive data warehouses where filtering and aggregation execute on the database before streaming visual aggregates to Go.

---

## Theming

`ggplot` ships 60+ built-in themes. Use `.Theme("name")` to switch:

```go
p.Theme("dashboard")   // clean card-style (default)
p.Theme("dark")        // dark background
p.Theme("minimal")     // minimal chrome
p.Theme("observable")  // Observable-inspired
p.Theme("nord")        // Nord palette
p.Theme("cyberpunk")   // neon-on-dark
```

See all themes with `theme.AllNames()`.

---

## Documentation

| Document | Description |
|---|---|
| [DATASET.md](docs/DATASET.md) | Deep dive into the Engine abstraction, Memory, and Arrow backends |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Package map, rendering pipeline, design decisions |
| [ROADMAP.md](docs/ROADMAP.md) | Development plan aligned with the ggplot2 book (3e) |
| [BENCHMARK.md](docs/BENCHMARK.md) | Arrow vs Memory engine performance benchmarks |

---

## Dependencies

| Package | Role |
|---|---|
| [`gogpu/gg`](https://github.com/gogpu/gg) | 2D vector rendering with anti-aliased lines, fills, and text |
| [`apache/arrow-go`](https://github.com/apache/arrow-go) | Columnar data (zero-copy for IPC/Parquet reads) |

## Contributing

Contributions are welcome!

## License

MIT — see [LICENSE](LICENSE).
