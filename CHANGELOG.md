# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

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
