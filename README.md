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
| **Geometries** | Point, Line, Step, Bar, Col, Histogram, Area, Density, Polygon, Rug, HLine, VLine, Segment, Text, BoxPlot, Smooth, Tile, ErrorBar |
| **Statistics** | Identity, Bin/Count, Density (KDE), Smooth (LOESS + lm), Summary, BoxPlot (Tukey/range whiskers, notch CI) |
| **Scales** | Linear, Log10, Sqrt, Reverse, Discrete |
| **Color Palettes** | 60+ built-in palettes — Viridis, ColorBrewer, Tab10, Observable, Seaborn, and more |
| **Faceting** | Grid (row ~ col), Wrap (NCols/NRows) |
| **Data Backends** | Native Memory, Apache Arrow IPC/Parquet, BigQuery SQL pushdown |
| **Output** | PNG, SVG 1.1, PDF 1.4, HiDPI via `WithScale()` |
| **Theming** | 60+ themes — Dashboard, Dark, Classic, Minimal, Observable, Seaborn, Nord, Dracula, and more |

---

## What's New in v0.0.5

### Concurrent Build & Draw

`Plot.Build` and `Built.Draw` now execute **panel-parallel** rendering via [`errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup). Multi-panel faceted plots build and render each panel concurrently, with automatic single-panel fast path (zero overhead for simple plots).

- **Parallel Build** — each facet panel's data pipeline (stat → scale → position) runs in its own goroutine
- **Parallel Draw** — each panel's data layers render to an independent sub-canvas, then composite onto the main surface via `DrawImage`
- Chrome (grid lines, axes, titles, legend) remains sequential for correctness

### Typed Error Envelope

All errors now carry structured context via `*ggplot.Error{Phase, Layer, Stage, Cause}`:

```
ggplot [build/layer 2/transform]: pipeline failed for group "A": column x not found
```

Full `errors.Is` / `errors.As` / `Unwrap` support with phase-aware sentinels (`PhaseBuild`, `PhaseDraw`, `PhaseRender`).

### Stat Transform Pipeline

Composable `stat.Transform` interface for full grammar-of-graphics data pipelines. 15+ stat transforms including `BinX`, `Count`, `DensityX`, `SmoothXY`, `BoxplotY`, `NormalizeY`, `Filter`, `StackY`, and `Summary`. Pipeline constructors like `geom.RectY(pipeline, opts...)` and `geom.LineY(pipeline, opts...)`.

### New Geometries

Tile, Segment, ErrorBar, Polygon, Ribbon, and Difference — bringing the total to **18 geometry types**.

### 60+ Themes with Dashboard Default

New default theme: **Dashboard** — a clean card-style theme with the Tab10/Blues palette. 60+ themes including Observable, Nord, Dracula, Gruvbox, GitHub, Cyberpunk, and more. Sealed element types with full ggplot2-style inheritance hierarchy.

### GPU or CPU Rendering

`canvas.NewGGCanvasCPU()` and `ggplot.WithCPU()` for deterministic, GPU-independent rasterization — essential for CI/golden tests and environments without GPU access.

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
| ![Polygon](assets/polygon.png) | ![Tile](assets/tile.png) | |
| `geom.Polygon` | `geom.Tile` | |

> Each image is generated by a self-contained example in [`examples/geometries/`](examples/geometries/).

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
    FacetWrap("season", 2, 0).
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
