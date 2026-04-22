# ggplot — Grammar of Graphics for Go

![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-blue)

A pure-Go implementation of the **Grammar of Graphics** for declarative, composable
data visualization. Inspired by R's [ggplot2](https://ggplot2-book.org/) by Hadley Wickham.

```go
ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
    Layer(geom.Point(geom.WithSize(4), geom.WithAlpha(0.7))).
    Layer(geom.Smooth()).
    Labs(ggplot.Title("My Plot"), ggplot.XLab("X"), ggplot.YLab("Y")).
    Theme("dark").
    Save("plot.png", 800, 500)
```

---

## Gallery

| | |
|---|---|
| ![Clifford Attractor](assets/clifford.png) | ![Butterfly Curve](assets/butterfly.png) |
| *Clifford attractor — 500 k points, alpha blending, continuous color scale* | *Butterfly curve — parametric path with color interpolation* |

| | | |
|---|---|---|
| ![Scatter](examples/geometries/point/point.png) | ![Line](examples/geometries/line/line.png) | ![Area](examples/geometries/area/area.png) |
| ![Bar](examples/geometries/bar/bar.png) | ![Histogram](examples/geometries/histogram/histogram.png) | ![Smooth](examples/geometries/smooth/smooth.png) |

---

## Features

### Geometries
Point, Line, Path, Step, Bar, Histogram, Area, Density, Rug,
HLine, VLine, Text, BoxPlot, Smooth

### Statistics
Identity, Bin/Count, Density (KDE), Smooth (LOESS with tri-cube kernel),
Summary, BoxPlot (Tukey 1.5×IQR fences)

### Scales
Linear, Log10, Sqrt, Reverse, Discrete — with `scale.Resolve(name)` factory
and `.ScaleX("log10")` / `.ScaleY("sqrt")` builder API

### Coordinates
Cartesian, Flipped (`CoordFlip`)

### Faceting
None, Wrap (NCols/NRows), Grid (row ~ col) — with strip labels and
consistent panel alignment

### Themes
Default, Classic, Minimal, Dark, BW — or compose your own `theme.Theme`

### Aesthetics
X, Y, Color, Group, Fill, Label, Size, Alpha — with automatic
categorical legend and continuous color bar

### Data Backends
- **Native**: `dataset.DataFrame` with `[]float64` / `[]string` columns
- **Apache Arrow**: Zero-copy `TableDataset` / `TableColumn` via `dataset/arrow`
- **SQL**: Lazy `Table` dataset with predicate pushdown (experimental)

---

## Quick Start

### Install

```bash
go get github.com/TuSKan/ggplot
```

### Scatter Plot

```go
package main

import (
    "github.com/TuSKan/ggplot"
    "github.com/TuSKan/ggplot/aes"
    "github.com/TuSKan/ggplot/dataset"
    "github.com/TuSKan/ggplot/geom"
)

func main() {
    ds, _ := dataset.NewDataFrame(map[string][]float64{
        "x": {1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
        "y": {2, 4, 5, 4, 6, 8, 7, 9, 10, 11},
    })

    ggplot.New(ds, aes.X("x"), aes.Y("y")).
        Layer(geom.Point(geom.WithSize(5), geom.WithColor("coral"))).
        Layer(geom.Smooth()).
        Labs(ggplot.Title("Quick Start"), ggplot.XLab("X"), ggplot.YLab("Y")).
        Theme("minimal").
        Save("scatter.png", 800, 500)
}
```

### Faceted Time Series

```go
ggplot.New(ds, aes.X("day"), aes.Y("temp")).
    Layer(geom.Line(geom.WithColor("seagreen"), geom.WithLineWidth(1.5))).
    FacetWrap("season", 2, 0).
    Labs(ggplot.Title("Temperature by Season")).
    Theme("dark").
    Save("facets.png", 900, 600)
```

### Box Plot with Groups

```go
ggplot.New(ds, aes.X("group"), aes.Y("value")).
    Layer(geom.BoxPlot(geom.WithFill("lightyellow"), geom.WithAlpha(0.8))).
    Labs(ggplot.Title("Distribution by Group")).
    Theme("classic").
    Save("boxplot.png", 800, 500)
```

---

## Builder API Reference

| Method | Description |
|---|---|
| `New(ds, aes...)` | Create a plot with dataset and global aesthetics |
| `.Layer(geom)` | Add a geometry layer (point, line, bar, …) |
| `.Labs(opts...)` | Set title, subtitle, axis labels, caption |
| `.Theme(name)` | Apply a named theme (default, classic, minimal, dark, bw) |
| `.ScaleX(name)` | Override X scale (log10, sqrt, reverse) |
| `.ScaleY(name)` | Override Y scale (log10, sqrt, reverse) |
| `.CoordFlip()` | Swap X and Y axes |
| `.FacetWrap(col, ncol, nrow)` | Wrap facet panels by column values |
| `.FacetGrid(row, col)` | Grid facet by two columns |
| `.LegendPosition(pos)` | Legend placement: right, left, top, bottom, none |
| `.XLim(min, max)` | Set X axis limits |
| `.YLim(min, max)` | Set Y axis limits |
| `.Save(path, w, h)` | Render and save as PNG |
| `.Render(w, h)` | Render and return canvas |

---

## Examples

Organized by roadmap phase under `examples/`:

| Directory | Content |
|---|---|
| `phase2_geometries/` | All 14 geometry types — point, line, step, bar, histogram, area, density, rug, hline/vline, text, boxplot, smooth |
| `phase2_statistics/` | Stat transforms — identity, bin, density, smooth (LOESS), boxplot |
| `phase2_scales/` | Scale types — linear, log10, sqrt, reverse, discrete |
| `phase2_features/` | Coordinates (cartesian, flip), facets (wrap, grid), 5 themes, legend positions, aesthetics |
| `showcase/` | Combined multi-feature showcase |
| `clifford/` | High-density attractor rendering (500 k points) |
| `butterfly/` | Parametric curve with continuous color |
| `color_mapping/` | Continuous color scale demonstration |

Run any example:
```bash
go run ./examples/phase2_geometries/
go run ./examples/clifford/
```

---

## Documentation

| Document | Description |
|---|---|
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Package map, rendering pipeline, design decisions |
| [ROADMAP.md](docs/ROADMAP.md) | 19-phase development plan aligned with the ggplot2 book (3e) |

---

## Roadmap

The [full roadmap](docs/ROADMAP.md) tracks 19 development phases aligned with
the [ggplot2 book (3e)](https://ggplot2-book.org/):

- ✅ **Phase 1–4** — Core architecture, grammar primitives, data backends, production hardening
- 🔲 **Phase 5–8** — Position/colour/other scales, faceting controls
- 🔲 **Phase 9–12** — Annotations, composition (patchwork), maps, networks
- 🔲 **Phase 13–19** — Themes deep-dive, guides, output backends, programming/extensibility

---

## Dependencies

| Package | Role |
|---|---|
| [`gogpu/gg`](https://github.com/gogpu/gg) | 2D vector rendering with anti-aliased lines, fills, and text |
| [`apache/arrow-go`](https://github.com/apache/arrow-go) | Zero-copy columnar data (optional, for `dataset/arrow`) |
| [`golang.org/x/image`](https://pkg.go.dev/golang.org/x/image) | Font rendering support |

---

## License

MIT — see [LICENSE](LICENSE).
