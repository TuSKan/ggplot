# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.0.5] — 2026-05-20

### Added

#### Stat Transform Pipeline (Phase 4)
- **`stat.Transform` interface** — composable data-transform pipeline: `Name()`, `Apply(ctx, TransformInput)`, `OutputMapping()`, `OutputSchema()`, `OutputHints()`.
- **Pipeline constructors**: `geom.RectY()`, `geom.RectX()`, `geom.LineY()`, `geom.AreaY()` accept `[]stat.Transform` pipelines for full grammar-of-graphics composition.
- **Stat transforms**: `stat.BinX()`, `stat.BinY()`, `stat.Count()`, `stat.DensityX()`, `stat.SmoothXY()`, `stat.BoxplotY()`, `stat.NormalizeY()`, `stat.Filter()`, `stat.SelectX()`, `stat.GroupX()`, `stat.GroupY()`, `stat.Group()`, `stat.SortX()`, `stat.StackY()`, `stat.Summary()`.
- **Channel hints**: `HintCount`, `HintProportion`, `HintInterval`, `HintDensity` — axis formatters use these for semantic tick formatting (e.g., `%` for proportions).
- **Percentile reducers**: `"p10"`, `"p25"`, `"p50"`, `"p75"`, `"p90"` in `GroupX`/`GroupY`/`Group` via engine-native `AggPercentile`.

#### TypeRect Geometry
- **`geom.TypeRect`** — unified rectangle mark replacing `TypeBar`/`TypeHistogram` for pipeline constructors.
- **`geom.Params.Inset`** — pixel inset per side between adjacent rects (0.5 for continuous bins, 0 for discrete bars).
- `Histogram()`, `RectY()`, `RectX()` now use `TypeRect`; `Bar()`/`Col()` remain `TypeBar` (discrete bars).

#### Theme Element System (Phase 4.4)
- **Sealed element types**: `theme.ElementText`, `theme.ElementLine`, `theme.ElementRect`, `theme.ElementBlank` — replaces all legacy `TextStyles`/`GridStyle`/`PanelStyle`/`TickStyle`/`FontConfig`.
- **Inheritance hierarchy**: `Elements map[string]Element` + `parentOf` tree (e.g., `axis.title.x` ← `axis.title` ← `text`).
- **Merge functions**: `MergeText()`, `MergeLine()`, `MergeRect()` — zero-value-aware field merge.
- **45 themes** migrated to Element compositions.
- **New themes**: `Observable` (Observable10 palette), `Tableau` (Tab10 palette), plus `editorial`, `codeeditor`, `scientific`, `expressive`, `modern`, `accessibility` theme families.

#### New Geometries
- **`geom.Tile`** — heatmap rectangles via x/y/fill.
- **`geom.Segment`** — line segments via `aes.XEnd`/`aes.YEnd` endpoint aesthetics.
- **`geom.ErrorBar`** — vertical error bars with ymin/ymax + caps.
- **`geom.Polygon`** — closed polygon paths.
- **`geom.Ribbon`** — filled band between ymin/ymax.
- **`geom.Difference`** — difference between two series.

#### Engine Stat Kernels
- **`dataset/memory/stat_kernel.go`** — LOESS, linear regression, KDE, boxplot, binning with full SE support.
- **`dataset/arrow/stat_kernel.go`** — same kernels for Arrow engine.
- **`dataset/bigquery/stat_kernel.go`** — native SQL stat kernels for BigQuery: LOESS, KDE, bin, boxplot, smooth. No local materialization.

#### Rendering & Canvas
- **`canvas.NewGGCanvasCPU()`** — pure-CPU analytic rasterizer, bypassing GPU accelerator for deterministic output.
- **`ggplot.WithCPU()`** — `RenderOpt` to force CPU rendering in `Save()`/`WriteTo()`.
- **Cyclic color palettes**: `colormap/data_cyclic.go`.

#### Testing
- **Backward-compat shim verification**: 11 tests in `geom/shim_test.go` validating sugar ≡ pipeline equivalence.
- **Stat composition tests**: `stat/composition_test.go` — 556 lines of pipeline composition tests.
- **Golden tests**: CPU-only rendering for deterministic cross-run stability.

### Changed
- `geom.Histogram()` now produces `TypeRect` (was `TypeHistogram`) with `Inset: 0.5`.
- `geom.RectY()`/`geom.RectX()` now produce `TypeRect` (was `TypeBar`).
- `position` package merged into `geom` package as `geom.Pos` interface.
- `stat.go` split into focused files: `bin.go`, `count.go`, `density.go`, `smooth.go`, `boxplot.go`, `filter.go`, `group.go`, `normalize.go`, `select.go`, `sort.go`, `stack.go`, `summary.go`, `transform.go`.
- Golden tests run sequentially with `WithCPU()` for deterministic PNG output.
- `docs/TRANSFORMS.md` consolidated into `docs/ROADMAP.md`.

### Fixed
- **GPU canvas blank renders**: Multiple `gg.Context` instances in one process produced blank images due to GPU accelerator contention. Fixed by CPU-only rendering for golden tests.
- **BigQuery LOESS row count**: Replaced `SELECT COUNT(*)` with metadata `column.Len()` (API `NumRows`).
- **`loessTopK` dead code**: Removed unused `nOut` argument.

## [0.0.4]

### Added
- **SVG/PDF output**: `Save()` now dispatches on file extension (`.svg`, `.pdf`, `.png`).
  - `WriteTo()` supports `"svg"`, `"pdf"`, and `"png"` format strings.
  - Native SVG 1.1 and PDF 1.4 backends via `recording.Backend` interface.
- **HiDPI rendering**: `Save()` and `WriteTo()` accept `WithScale(s)` option for retina output.
- **Recording canvas**: `RecordingCanvas` wraps `recording.Recorder` for vector-format rendering.
- **KDE parallelization**: Density stat uses `runtime.NumCPU()` goroutines for kernel evaluation.
- **Bandwidth selection**: `stat.Options.Bandwidth` allows explicit KDE bandwidth; `0` auto-selects via Silverman's rule.
- **Histogram binning strategies**: `stat.Options.BinMethod` supports `"sturges"` (default), `"scott"`, `"fd"` (Freedman-Diaconis), `"sqrt"`.
- **Smooth lm method**: `stat.Options.Method = "lm"` for simple linear regression alongside LOESS.
- **Boxplot variants**: `stat.Options.Whisker` (`"tukey"` / `"range"`), `stat.Options.Notch` (95% CI). New output columns: `notch_lower`, `notch_upper`.
- **Boxplot geom options**: `geom.WithWhisker(rule)`, `geom.WithNotch(enabled)` option constructors.
- **Binary hashing**: FNV-1a row key generation in `dataset.frame` for zero-allocation GroupBy/Distinct.
- **Map capacity hints**: Pre-sized maps in `Distinct()` and `GroupBy()` to reduce rehash pressure.
- **Benchmark suite**: `ggplot_bench_test.go` covering 11 rendering scenarios.
- **Context propagation**: `Save`, `Render`, `WriteTo` accept `context.Context` for cancellation.
- **Engine-based Dataset**: Decoupled `Engine` interface with `memory`, `arrow`, `bigquery`, `csv`, `parquet` implementations.
- **ColorBrewer palettes**: Programmatic access via `colormap` package.
- **Typed constants**: `theme.Name`, `scale.Name`, `stat.Name` replace magic strings.

### Changed
- `Plot.clone()` performs full deep-copy of `Mapping`, `ScaleOverrides`, and `ColorScales` maps.
- `gridFacet.GridDims()` uses actual row/col cardinalities from `Split()`, not `ceilSqrt`.
- `coord.Transform` returns `(x, y float64)` instead of mutating in place.
- Font loading uses `internal/fonts.Resolver` with system font registry and embedded fallback.
- **Orientation-aware geoms**: `geom.Orientation` (`Vertical`/`Horizontal`) replaces `coord.Flip()`. Geoms now know their axis direction; the rendering pipeline swaps scales/labels automatically.
- `CoordFlip()` is retained as sugar — it sets `Horizontal` on all layers and swaps labels.
- Removed `coord.IsFlipped()` from `Coord` interface and `flippedCoord` type.

### Fixed
- Shallow-copy corruption in `Plot.clone()` when deriving plots.
- Facet grid layout miscalculation for non-square cardinalities.
- Silent nil-table drops in stat transforms now return proper errors.
- Unused `rowKey`/`joinParts` dead code removed (`dataset/frame.go`).
- All `errcheck` lint warnings resolved in benchmark suite.

## [0.1.0] — Initial Release

### Added
- Grammar of Graphics pipeline: Data → Stat → Scale → Coord → Geom → Guide.
- Geometries: Point, Line, Bar, Area, Histogram, Density, Smooth, Boxplot.
- Statistics: Identity, Bin, Count, Density, Smooth, Boxplot.
- Scales: Linear, Log, Sqrt, Discrete, Time, Reverse.
- Coordinate systems: Cartesian, CoordFlip.
- Faceting: None, Wrap, Grid.
- Themes: Default, Minimal, Dark.
- Annotations: HLine, VLine, Text.
- Legends: Categorical, Continuous color bar.
