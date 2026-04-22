# ggplot Roadmap

> Go implementation of the Grammar of Graphics for declarative, composable data visualization.
> inspired by [ggplot2](https://ggplot2-book.org/) (R package)

---

## ✅ Phase 1 — Core Architecture (Complete)

> Book Ch.1–2 (Introduction, First Steps), Ch.13 (Layers), Ch.19 (Internals)

- [x] Typed `Dataset` / `Frame` API with `DType` system (Float64, Int64, String, Bool)
- [x] Lazy, fluent ETL pipeline: `Select`, `Filter`, `Mutate`, `Arrange`, `Distinct`, `Summarize`
- [x] In-memory column types with null masks and cross-type iteration
- [x] Canvas abstraction (`gg`-backed) with full 2D drawing primitives
- [x] Declarative `PlotSpec` → rendering pipeline: Facet → Scale Training → Layer Rendering

## ✅ Phase 2 — Grammar Primitives (Complete)

> Book Ch.3 (Individual Geoms), Ch.4 (Collective Geoms), Ch.5 (Statistical Summaries)

- [x] **Geometries**: Point, Line, Path, Step, Bar, Histogram, Area, Density, Rug, HLine, VLine, Text, BoxPlot, Smooth
- [x] **Statistics**: Identity, Bin/Count, Density (KDE), Smooth (LOESS), Summary, BoxPlot
- [x] **Scales**: Linear, Log10, Sqrt, Reverse, Discrete + `scale.Resolve(name)` factory
- [x] **Coordinates**: Cartesian, Flipped (`CoordFlip`)
- [x] **Faceting**: None, Wrap (with NCols/NRows), Grid (row ~ col), Strip Labels
- [x] **Themes**: Default, Classic, Minimal, Dark, BW
- [x] **Guides**: X/Y Axes, Grid, Categorical Legend, Continuous Color Bar (vertical + horizontal)
- [x] **Aesthetics**: X, Y, Color, Group, Fill, Label, Size, Alpha
- [x] **Legend Position**: right, left, top, bottom, none via `.LegendPosition()`
- [x] **Scale Override**: `.ScaleX("log10")`, `.ScaleY("sqrt")` via builder API

## ✅ Phase 3 — Data Backends (Complete)

- [x] Arrow adapter: zero-copy `TableDataset` / `TableColumn`, chunked iterators, `Buffer` pre-allocator
- [x] SQL adapter: lazy `Table` dataset with predicate pushdown, `FilterSQL` / `GroupBySQL`, auto type detection
- [x] `NativeFilterProvider` / `IterableColumn` interfaces for backend extensibility

## ✅ Phase 4 — Production Hardening (Complete)

- [x] All `errcheck` / `go vet` / linter errors resolved
- [x] `clone()` correctly copies all spec fields (XLim, YLim, LegendPosition, ScaleOverrides)
- [x] `BoundsSetter` implemented on all scale types (Linear, Log10, Sqrt, Reverse, Discrete)
- [x] Compile-time interface checks for all scale types
- [x] 70+ tests passing across all packages

---

## 🔲 Phase 5 — Position Scales & Axes (Book Ch.10, Ch.14)

> Book: `scale_x_continuous`, `scale_x_log10`, `scale_x_date`, `sec_axis`, `guide_axis`

- [ ] **Date/Time Scale** — `scale.DateTime()` with automatic tick formatting (year, month, day, hour)
- [ ] **Scale Limits** — `scale.WithLimits(min, max)` for zoom without data filtering (`coord_cartesian` equivalent)
- [ ] **Scale Breaks** — `scale.WithBreaks(...)` for explicit tick positions
- [ ] **Scale Labels** — `scale.WithLabels(...)` for custom tick label text
- [ ] **Minor Breaks** — `scale.WithMinorBreaks(...)` for minor grid line positions
- [ ] **Scale Expand** — `scale.WithExpand(mult, add)` to control axis padding (ggplot2 `expand`)
- [ ] **Secondary Axes** — `SecAxis()` / `DupAxis()` for dual Y-axis support
- [ ] **Axis Formatting** — `FormatX(func(float64) string)` for custom formatters (currency, percent, scientific)
- [ ] **guide_axis(n.dodge)** — rotated or dodged axis labels for dense categorical axes
- [ ] **Binned Scales** — `scale_x_binned()` for discretizing continuous axes

## 🔲 Phase 6 — Colour Scales & Legends (Book Ch.11, Ch.14)

> Book: `scale_colour_gradient`, `scale_colour_brewer`, `guide_colourbar`, `guide_legend`

- [ ] **Continuous Colour Scales** — `scale_colour_gradient()`, `gradient2()` (diverging), `gradientn()` (multi-stop)
- [ ] **Discrete Colour Scales** — `scale_colour_brewer()`, `scale_colour_manual(values=...)`, `scale_colour_grey()`
- [ ] **Palette System** — viridis (A–E), ColorBrewer (sequential, diverging, qualitative), hue/chroma/luminance palettes
- [ ] **Colour Blindness** — `scale_colour_viridis_d()` / `_c()` with built-in accessibility palettes
- [ ] **Fill vs Colour** — Independent `scale_fill_*` that mirror colour scales
- [ ] **Missing Value Colour** — `na.value` parameter for controlling how NAs are drawn
- [ ] **Guide Customization** — `guide_colourbar(barwidth, barheight, nbin, direction)`, `guide_legend(ncol, nrow, byrow)`
- [ ] **Legend Key Glyphs** — custom key shapes per geom (e.g., line key for `geom_smooth`)

## 🔲 Phase 7 — Other Aesthetic Scales (Book Ch.12)

> Book: `scale_size`, `scale_shape`, `scale_linetype`, `scale_alpha`

- [ ] **Size Scale** — `scale_size(range)`, `scale_size_area()` (proportional to value), `scale_radius()`
- [ ] **Shape Scale** — `scale_shape_manual()` with 25+ built-in point shapes
- [ ] **Linetype Scale** — `scale_linetype()` mapping groups to dash patterns (solid, dashed, dotted, etc.)
- [ ] **Alpha Scale** — `scale_alpha(range)` for opacity mapping
- [ ] **Identity Scales** — `scale_*_identity()` to use raw column values as aesthetic values

## 🔲 Phase 8 — Advanced Geometries (Book Ch.3–5)

> Book: `geom_violin`, `geom_dotplot`, `geom_ribbon`, `geom_tile`, `geom_contour`, `geom_sf`

- [ ] **geom.Tile / Raster** — raster grid for 2D density and correlation matrices (Ch.3)
- [ ] **geom.Violin** — mirrored density estimation per group (Ch.5)
- [ ] **geom.Ribbon** — filled band with `ymin`/`ymax` for confidence intervals (Ch.3)
- [ ] **geom.Segment / Curve** — directed line segments and bezier curves (Ch.8)
- [ ] **geom.ErrorBar / Crossbar / Linerange / Pointrange** — full error bar family (Ch.5)
- [ ] **geom.Contour / Contour_filled** — 2D density contour lines from XYZ data (Ch.3)
- [ ] **geom.Polygon / Path** — arbitrary filled/stroked polygons (Ch.3, Ch.6)
- [ ] **geom.Rect** — parameterised by xmin/ymin/xmax/ymax (Ch.3)
- [ ] **geom.Jitter** — jittered points for overplotting mitigation (Ch.3)
- [ ] **geom.Dotplot** — dot plots for small datasets (Ch.3)

## 🔲 Phase 9 — Annotations (Book Ch.8)

> Book: `annotate`, `geom_hline/vline/abline`, `geom_rect`, `geom_text/label`, reference lines

- [ ] **annotate()** — layer-less annotations: rect, text, segment, arrow, curve, pointrange
- [ ] **geom.Label** — `geom_label()` with background box and connector lines
- [ ] **Direct Labelling** — `ggrepel`-style anti-collision text placement
- [ ] **Reference Lines** — `geom.ABLine(intercept, slope)` for regression lines
- [ ] **Marginal Annotations** — custom axis marks, rug ticks, distribution overlays

## 🔲 Phase 10 — Coordinate Systems (Book Ch.15)

> Book: `coord_cartesian`, `coord_fixed`, `coord_polar`, `coord_map`, `coord_trans`

- [ ] **coord.Cartesian(xlim, ylim)** — zoom without data clipping (vs `XLim`/`YLim` which filter)
- [ ] **coord.Fixed(ratio)** — fixed aspect ratio (e.g., 1:1 for maps)
- [ ] **coord.Polar(theta, start, direction)** — polar coordinates for pie/rose/radar charts
- [ ] **coord.Trans(x, y)** — apply separate transformations to each axis
- [ ] **coord.Map(projection)** — map projections (Mercator, Lambert, etc.)

## 🔲 Phase 11 — Faceting Deep Dive (Book Ch.16)

> Book: `facet_wrap`, `facet_grid`, `labeller`, `scales = "free"`, `space = "free"`

- [ ] **Free Scales** — `facet.Wrap(col, FreeX(), FreeY())` per-panel independent axes
- [ ] **Free Space** — `facet.Grid(row, col, Space("free"))` proportional panel sizing
- [ ] **Labeller Functions** — `labeller.Both()`, `labeller.Parsed()`, custom label formatters
- [ ] **Facet Margins** — `margins = TRUE` for aggregate panels alongside facets
- [ ] **Missing Facet Combinations** — `drop = FALSE` to show empty panels
- [ ] **Strip Placement** — `strip.position = "bottom"`, `"left"` for axis-adjacent strips

## 🔲 Phase 12 — Theme System Deep Dive (Book Ch.17)

> Book: `theme()`, `element_text`, `element_line`, `element_rect`, `element_blank`

- [ ] **Element System** — `theme.Element{Text,Line,Rect,Blank}` with inheritance hierarchy
- [ ] **theme() Granularity** — individual theme element overrides (e.g., `axis.title.x`, `legend.key.size`)
- [ ] **theme Inheritance** — child elements inherit from parent (e.g., `axis.title.x` inherits from `axis.title`)
- [ ] **Custom Themes** — `theme.New(base, ...)` for composable user themes
- [ ] **Plot Margin** — `plot.margin` with unit support (cm, inches, lines)
- [ ] **Legend Layout** — `legend.box`, `legend.key`, `legend.background`, `legend.margin`
- [ ] **Strip Styling** — `strip.text`, `strip.background`, `strip.clip`

## 🔲 Phase 13 — Position Adjustments (Book Ch.4, Ch.13)

> Book: `position_dodge`, `position_stack`, `position_fill`, `position_jitter`, `position_nudge`

- [ ] **position.Stack** — stack bars/areas vertically (default for bar/area)
- [ ] **position.Dodge** — side-by-side grouped bars/points
- [ ] **position.Fill** — normalized stacking to 100% (proportional bars)
- [ ] **position.Jitter** — random displacement to avoid overplotting
- [ ] **position.JitterDodge** — combined jitter within dodged groups
- [ ] **position.Nudge** — fixed offset for labels relative to points

## 🔲 Phase 14 — Maps & Spatial (Book Ch.6)

> Book: `geom_sf`, `coord_sf`, map borders, choropleth

- [ ] **geom.SF** — Simple Features rendering from GeoJSON / Shapefile data
- [ ] **coord.SF** — CRS-aware coordinate system with proper map projection
- [ ] **Map Borders** — `borders()` convenience for country/state outlines
- [ ] **Choropleth** — fill polygons by data values for thematic maps

## 🔲 Phase 15 — Networks (Book Ch.7)

> Book: `ggraph`, `geom_edge_link`, `geom_node_point`

- [ ] **Graph Layouts** — force-directed, tree, circle, grid layout algorithms
- [ ] **geom.Edge / geom.Node** — edge rendering with bundling, node glyphs
- [ ] **ggraph Integration** — network-aware aesthetics (edge weight, node size)

## 🔲 Phase 16 — Arranging & Composition (Book Ch.9)

> Book: `patchwork`, `plot_layout`, `inset_element`

- [ ] **Patchwork** — `patchwork.Arrange(p1, p2, p3, layout="2x1")` for multi-plot grids
- [ ] **Plot Layout** — `layout.Design(areas)` for custom grid arrangements
- [ ] **Inset Plots** — `inset.Element(p, left, bottom, right, top)` for overlaid mini-plots
- [ ] **Shared Axes** — linked scales across composed plots
- [ ] **Collected Guides** — single legend for multiple composed plots

## 🔲 Phase 17 — Programming & Extensibility (Book Ch.18, Ch.20, Ch.21)

> Book: `aes_string`, `ggproto`, custom Stat/Geom/Scale, `layer()`

- [ ] **Programmatic Aesthetics** — `aes.String("colname")` for dynamic column names
- [ ] **Custom Geom Protocol** — user-defined `Geom` implementations via interface
- [ ] **Custom Stat Protocol** — user-defined `Stat` implementations via interface
- [ ] **Custom Scale Protocol** — user-defined `Scale` implementations via interface
- [ ] **after_stat() / after_scale()** — computed aesthetics from stat/scale output

## 🔲 Phase 18 — Output & Interactivity

> Beyond book scope — Go-specific extensions

- [ ] **SVG Output** — `output.SVG()` backend for web-embeddable vector graphics
- [ ] **PDF Output** — `output.PDF()` for print-quality publication figures
- [ ] **HTML Output** — interactive plots with hover tooltips via embedded SVG + JavaScript
- [ ] **Animated GIF** — frame-by-frame rendering for time-series animations
- [ ] **Live Preview** — `p.Show()` opens a native window with hot-reload on data changes

## 🔲 Phase 19 — Documentation & Ecosystem

- [ ] **API Reference** — auto-generated Go doc with runnable examples for every public type
- [ ] **Gallery** — 30+ curated examples with source code and rendered output
- [ ] **Cookbook** — task-oriented guides (time series, distributions, maps, composition)
- [ ] **Benchmarks** — performance comparison against R/ggplot2 for equivalent plots
- [ ] **CI/CD** — GitHub Actions with lint, test, benchmark, and example rendering on every PR
- [ ] **v1.0 Release** — semantic versioning, stable API guarantee, CHANGELOG
