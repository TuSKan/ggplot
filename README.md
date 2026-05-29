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
| **Output** | PNG, SVG 1.1, PDF 1.4, HiDPI via `WithScale()` |
| **Theming** | 60+ themes — Dashboard, Dark, Classic, Minimal, Observable, Seaborn, Nord, Dracula, and more |
| **Annotations** | `AnnotateText`, `AnnotateRect`, `AnnotateSegment`, `AnnotateArrow`, `AnnotateLabel` — layer-less fixed-coordinate annotations |
| **Coordinate Systems** | `CoordCartesian` (viewport zoom), `CoordFixed` (aspect ratio), `CoordFlip`, `Coord(Polar())` |

---

## What's New in v0.0.8

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
		Labs(ggplot.Title("Quick Start"), ggplot.XLab("X"), ggplot.YLab("Y")).
		Save(ctx, "scatter.png", 800, 500)
}
```

### Faceted Time Series

```go
ggplot.New(ds, aes.X("day"), aes.Y("temp")).
    Layer(geom.Line(geom.WithColor("seagreen"), geom.WithLineWidth(1.5))).
    FacetWrap("season", facet.NCols(2)).
    Labs(ggplot.Title("Temperature by Season")).
    Theme("dark").
    Save(ctx, "facets.png", 900, 600)
```

### Box Plot with Groups

```go
ggplot.New(ds, aes.X("group"), aes.Y("value")).
    Layer(geom.Boxplot(geom.WithFill("#E8E8E8"), geom.WithAlpha(0.8))).
    Labs(ggplot.Title("Distribution by Group")).
    Theme("classic").
    Save(ctx, "boxplot.png", 800, 500)
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
