# ggplot

[![Go Reference](https://pkg.go.dev/badge/github.com/TuSKan/ggplot.svg)](https://pkg.go.dev/github.com/TuSKan/ggplot)
[![Go Report Card](https://goreportcard.com/badge/github.com/TuSKan/ggplot)](https://goreportcard.com/report/github.com/TuSKan/ggplot)
[![CI](https://github.com/TuSKan/ggplot/actions/workflows/ci.yml/badge.svg)](https://github.com/TuSKan/ggplot/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/TuSKan/ggplot/branch/main/graph/badge.svg)](https://codecov.io/gh/TuSKan/ggplot)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/TuSKan/ggplot)](https://github.com/TuSKan/ggplot/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

![ggplot](assets/cover.jpg)

**Production-grade Grammar of Graphics for Go.**

A pure-Go data visualization library implementing a rigorous, declarative Grammar of Graphics pipeline. Inspired by Hadley Wickham's renowned [ggplot2](https://ggplot2-book.org/), but architected specifically for Go's performance characteristics, static typing, and concurrency model.

## Overview

`ggplot` provides an expressive, composable API for generating complex data visualizations. It decouples the data manipulation (Apache Arrow, BigQuery, Memory) from the statistical transformations and the final vector rendering, resulting in highly scalable plotting pipelines.

| Capability | Supported Features |
|---|---|
| **Geometries** | Point, Line, Path, Step, Bar, Histogram, Area, Density, Rug, HLine, VLine, Text, BoxPlot, Smooth |
| **Statistics** | Identity, Bin/Count, Density (KDE), Smooth (LOESS), Summary, BoxPlot (Tukey 1.5×IQR) |
| **Scales** | Linear, Log10, Sqrt, Reverse, Discrete |
| **Faceting** | Grid (row ~ col), Wrap (NCols/NRows) |
| **Data Backends** | Native Memory, Apache Arrow IPC/Parquet, BigQuery |
| **Theming** | Default, Classic, Minimal, Dark, BW |

---

## Gallery

| | |
|---|---|
| ![Clifford Attractor](assets/clifford.png) | ![Butterfly Curve](assets/butterfly.png) |
| *Clifford attractor — 500 k points, alpha blending, continuous color scale* | *Butterfly curve — parametric path with color interpolation* |

| | | |
|---|---|---|
| ![Scatter](assets/point.png) | ![Line](assets/line.png) | ![Area](assets/area.png) |
| ![Bar](assets/bar.png) | ![Histogram](assets/histogram.png) | ![Smooth](assets/smooth.png) |

---

## Why `ggplot`?

Data science in Go often suffers from fragmented or overly imperative plotting APIs. `ggplot` solves this by introducing:
- **Declarative Compositions** — Build complex charts by layering geometries and statistics instead of drawing pixels.
- **Provider-Agnostic Engines** — Swap out the underlying dataset execution engine (`memory` vs `arrow`) without changing a single line of your plotting code.
- **Publication-Ready Outputs** — Anti-aliased 2D vector rendering powered by `gogpu/gg`, saving to high-DPI PNGs or in-memory canvases.

## Quick Start

### Installation

```bash
go get github.com/TuSKan/ggplot
```

### 1. Scatter Plot

```go
package main

import (
	"log"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
)

func main() {
	// Initialize a memory engine and construct columns explicitly.
	eng := memory.NewEngine()
	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
		eng.NewFloat64Column("y", []float64{2, 4, 5, 4, 6, 8, 7, 9, 10, 11}),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// Build the plot using declarative Grammar of Graphics.
	ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(5), geom.WithColor("coral"))).
		Layer(geom.Smooth()).
		Labs(ggplot.Title("Quick Start"), ggplot.XLab("X"), ggplot.YLab("Y")).
		Theme("minimal").
		Save("scatter.png", 800, 500)
}
```

### 2. Faceted Time Series

```go
ggplot.New(ds, aes.X("day"), aes.Y("temp")).
    Layer(geom.Line(geom.WithColor("seagreen"), geom.WithLineWidth(1.5))).
    FacetWrap("season", 2, 0).
    Labs(ggplot.Title("Temperature by Season")).
    Theme("dark").
    Save("facets.png", 900, 600)
```

### 3. Box Plot with Groups

```go
ggplot.New(ds, aes.X("group"), aes.Y("value")).
    Layer(geom.BoxPlot(geom.WithFill("lightyellow"), geom.WithAlpha(0.8))).
    Labs(ggplot.Title("Distribution by Group")).
    Theme("classic").
    Save("boxplot.png", 800, 500)
```

---

## Architecture & Data Backends

`ggplot` is built around a rigorous, interface-driven `dataset.Table` engine. This means you are not limited to `[]float64` slices. You can back your plots with robust columnar frameworks. See [**DATASET.md**](docs/DATASET.md) for a deep-dive into the backend engine architecture.

- **Memory Engine (`dataset/memory`)**: Lightweight, native Go slices. Best for standard web-server rendering.
- **Arrow Engine (`dataset/arrow`)**: Apache Arrow backed IPC streams and Parquet datasets. Best for high-performance ML pipelines or datasets >1M rows.
- **BigQuery Engine (`dataset/bigquery`)**: Lazy SQL pushdown execution. Best for massive data warehouses where filtering and statistics must be executed on the database before streaming the visual aggregate to Go.

---

## Documentation

| Document | Description |
|---|---|
| [DATASET.md](docs/DATASET.md) | Deep dive into the Engine abstraction, Memory, and Arrow backends |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Package map, rendering pipeline, design decisions |
| [ROADMAP.md](docs/ROADMAP.md) | 19-phase development plan aligned with the ggplot2 book (3e) |

---

## Project Roadmap

We actively track our development pipeline across multiple capability tiers focusing on Grammar Primitives, Scaling Functions, and Advanced Geometries.

Please see our full [**Project Roadmap**](docs/ROADMAP.md) to understand current milestones and architectural expansion goals.

- ✅ **Phases 1–4** — Core architecture, grammar primitives, data backends, production hardening
- 🔲 **Phases 5–8** — Position/colour/other scales, faceting controls
- 🔲 **Phases 9–12** — Annotations, composition (patchwork), maps, networks
- 🔲 **Phases 13–19** — Themes deep-dive, guides, output backends, programming/extensibility

---

## Dependencies

| Package | Role |
|---|---|
| [`gogpu/gg`](https://github.com/gogpu/gg) | 2D vector rendering with anti-aliased lines, fills, and text |
| [`apache/arrow-go`](https://github.com/apache/arrow-go) | Zero-copy columnar data |

## Contributing

Contributions are welcome!

## License

MIT — see [LICENSE](LICENSE).
