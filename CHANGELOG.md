# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Browser WASM Surface (Phase 6 of OUTPUT-SPEC.md)

#### Added
- **`output/web` package** — new browser surface for presenting ggplot figures in WebAssembly. `web.Mount(ctx, src, "container-id")` creates a `<canvas>` element inside a named container, renders via CPU rasterizer (`canvas.NewRasterCanvasCPU`), and drives an interactive event loop through `output.Session`.
- **CPU raster mode (default)** — renders to an `image.RGBA` via the CPU rasterizer, then transfers pixels to a Canvas2D context via `putImageData`. Works in all browsers, no WebGPU required.
- **SVG mode (`web.WithSVG()`)** — renders via `RecordingCanvas` → `ExportSVGWithMeta` → `innerHTML`. Preserves metadata channels: tooltips (`<title>`), clickable links (`<a href>`), and ARIA labels. Resolution-independent vector output.
- **GPU mode (`web.WithGPU()`)** — WebGPU-accelerated rendering via `gogpu.App` + `ggcanvas`. Delegates the event loop to `gogpu`, renders via `gg.Context` (GPU SDF shapes, MSDF text), and presents zero-copy to the browser's WebGPU surface. Requires Chrome 113+/Edge 113+ or Firefox with WebGPU flag.
- **Mode selector toolbar** — embedded `index.html` includes radio buttons for Raster (CPU), SVG, and GPU (WebGPU) modes. Switching modes cancels the current mount, clears the DOM container, and re-mounts with the new options. Go reads mode from `window.ggplotRenderMode` and listens for changes via `window.ggplotSetMode` callback.
- **DOM event integration** — pointer events (down/up/move with capture), wheel/scroll, double-click detection (400ms threshold), and `ResizeObserver` for container resize — all translated to `output.Event` and fed through `output.Session`/`output.Controller`.
- **Dev server (`web.Serve`)** — embedded `index.html` (via `go:embed`) + HTTP server serving `wasm_exec.js` from GOROOT and a pre-built `.wasm` binary. Native-only build (`//go:build !js`).
- **Example (`examples/web`)** — interactive scatter plot with clustered data, tooltips, and pan/zoom. Includes a native dev server (`examples/web/serve`). Supports live mode switching between all three render backends.

#### Changed
- **`gogpu` bumped to v0.41.0** — fixes `SetAppName` interface mismatch on the browser platform (`browserPlatform` now implements `PlatformManager`). This unblocks `gogpu.App` for WASM, enabling the GPU-accelerated rendering path via `wgpu` browser WebGPU backend + `ggcanvas`.

#### Fixed
- **Surface Close race** — removed `close(events)` from `rasterSurface.Close()` and `svgSurface.Close()`. Session exits via context cancellation, not channel close. Prevents send-on-closed-channel panic during mode switching.
- **ResizeObserver callback cleanup** — `releaseCallbacks` now guards against zero-value `js.Value` targets (ResizeObserver callbacks lack a DOM target for `removeEventListener`).

### SVG Output

#### Added
- **Responsive SVG**: Root `<svg>` element now includes `style="max-width:100%;height:auto"` for automatic responsive scaling in web pages.
- **SVG metadata channels**: New `aes.Title`, `aes.Href`, `aes.AriaLabel` aesthetics. In SVG output, `Title` emits `<title>` child elements for hover tooltips, `Href` wraps primitives in `<a href="...">` for clickable links, `AriaLabel` adds `aria-label` attributes for screen readers. PNG and PDF output ignore these aesthetics.
- **Pipeline-level metadata wiring**: Metadata channels are resolved in `drawLayer()` and automatically flow through `DrawContext.SetRowMetadata(i)` into every geometry drawer — no manual canvas-level calls needed. Per-row geoms (bars, points, tiles, segments, text, errorbar, rug, crossbar, linerange, pointrange, curve, dotplot) emit individual metadata per primitive. Group geoms (line, step, area, polygon, ribbon, violin) emit group-level metadata. Point batching is disabled when metadata is mapped to ensure per-point SVG elements.
- **`canvas.SetMetadata(meta)`**: New Canvas interface method for per-primitive metadata. Backends that don't support metadata (PNG) silently discard it.
- **`canvas.ExportSVGWithMeta`**: SVG export with metadata side-channel for tooltip/link/ARIA emission.

#### Fixed
- **SVG metadata applied to wrong elements**: `RecordingCanvas.Clear()` called the recorder's `Clear()` and `Fill()` directly, producing two draw ops during SVG playback without incrementing the metadata draw-op counter. This shifted all subsequent metadata indices by two, causing tooltips and links to land on the wrong SVG elements (e.g. axis labels instead of bars). Fixed by adding `consumeMetadata()` calls to keep the counter synchronized with the SVG backend.

### Discrete Scales

#### Added
- **Band padding API**: `scale.WithPaddingInner(f)` and `scale.WithPaddingOuter(f)` control bar gap and edge spacing on discrete scales. `DiscreteScale.BandWidth()` returns the derived bar width fraction.

## [0.0.10] — 2026-06-02

### Data-Space Interactive Pan/Zoom

#### Added
- **`output.DataSpaceController()`** — new default controller for `output/window`. Pan (drag) and zoom (scroll wheel) operate on data-space scale bounds instead of a canvas-level affine transform. Axes stay at fixed screen positions; tick labels update dynamically. O(1) viewport change — no rebuild, no data iteration.
- **`output.Measurable` interface** — optional `Figure` extension exposing per-panel pixel geometry and data-space bounds. Controllers use this for pixel→data coordinate conversion and per-panel hit-testing (faceted plots).
- **`output.Zoomable` interface** — optional `Figure` extension for fast viewport changes without rebuilding. `SetPanelViewport(idx, xlim, ylim)` mutates scale bounds directly. `ResetViewport()` restores original trained bounds.
- **`output.PanelInfo`** — describes one panel's pixel geometry (`Bounds`) and data-space extent (`XRange`, `YRange`) with helper methods `PixelToData` and `ContainsPixel`.
- **`output.EventDoubleClick`** — new event kind for viewport reset gesture. `output/window` detects double-clicks (two clicks within 400ms).
- **`Built` implements `Measurable` + `Zoomable`** — panel geometry is cached during `Draw()` and exposed via `PanelInfos()`. Scale bounds are mutated directly via `SetPanelViewport()` with mutex-protected thread safety.
- **Per-panel zoom for faceted plots** — zoom/pan applies only to the panel under the cursor.

#### Fixed
- **Scroll-wheel zoom fails on HiDPI displays** — gogpu's `WM_MOUSEWHEEL` handler converts screen→client coordinates but does not divide by the DPI scale factor, while pointer events do. This mismatch causes scroll X,Y to be in physical pixels while panel geometry is in logical pixels, so `ContainsPixel` hit-testing fails for most cursor positions at ≥125% scaling. `output/window` now normalises scroll coordinates to logical DIP before dispatching.

### Output Layer (Phases 1–5 of OUTPUT-SPEC.md)

#### Changed (Breaking)
- **`canvas.GGCanvas` renamed to `canvas.RasterCanvas`** (Phase 1). Constructors renamed: `NewGGCanvas` → `NewRasterCanvas`, `NewGGCanvasCPU` → `NewRasterCanvasCPU`, `FromGGContext` → `RasterFromContext`. The file `canvas/gg.go` is now `canvas/raster.go`. No behavioral change.
- **`Plot.Build` now returns `output.Figure`** (concretely a `*Built`) instead of `*Built` directly. Introspection is via a type assertion: `fig, _ := p.Build(ctx); b := fig.(*ggplot.Built)`. `*Built` stays exported with its full method set.
- **Removed `Built.Save` and `Built.WriteTo`** — these bypassed the output layer with duplicated encoding logic. Use `file.Save`, `file.Encode`, or `output.Render` + a surface instead.
- **Removed `Plot.Save`, `Plot.Encode`, `Plot.Image`, `Plot.WriteTo`** — all output façades have been moved to dedicated surface packages. Use `file.Save`/`file.Encode` (`output/file`) and `image.Render` (`output/image`) instead.

#### Added
- **`output` package — the unified destination layer** (Phase 2). One `Surface` model for file, in-memory image, desktop window, and browser canvas:
  - `Figure` (`Draw`), `Source` (`Build`), `Sizer` (`PreferredSize`) interfaces; `*Built` implements `Figure`+`Sizer`, `*Plot` implements `Source`.
  - `Surface` (`Acquire`/`Commit`/`Bounds`/`Close`) and `LiveSurface` (`+Events()`); `Imager` for image-producing surfaces.
  - `Render(ctx, Figure, Surface)` — the one-frame primitive.
  - `Event`/`EventKind` platform-neutral input events.
  - Blank-import surface registry: `Register`, `NewSurface`, `NewLiveSurface`, `SurfaceOptions`, options (`WithSize`, `WithPath`, `WithWriter`, `WithFormat`, `WithScale`, `WithCPU`).
  - Sentinels: `ErrUnknownSurface`, `ErrSurfaceConsumed`, `ErrNotLive`, `ErrUnsupportedFormat`, `ErrNoImage`.
- **`output/file` and `output/image` surfaces** (Phase 3) — single-shot surfaces registered via blank import. File encodes PNG/SVG/PDF to a path or `io.Writer`; image publishes an in-memory `image.Image` (CPU-rasterized).
- **`file.Save(ctx, src, path, w, h, opts...)`** (`output/file`) — save to a file (PNG/SVG/PDF), format inferred from extension.
- **`file.Encode(ctx, src, dst, format, w, h, opts...)`** (`output/file`) — write encoded bytes to an `io.Writer`.
- **`image.Render(ctx, src, w, h, opts...)`** (`output/image`) — render to an in-memory `image.Image` (CPU-rasterized).
- All output functions accept an `output.Source` (i.e. `*Plot`) — no façade methods on `Plot`.
- **`output.Session` / `Controller` interaction loop** (Phase 4) — drives a `Source` onto a `LiveSurface`: build once, draw, then re-render on `Event`s. `Action` (`Ignore`/`Redraw`/`Rebuild`/`Export`/`Close`) is the per-event decision; `State` carries `Bounds` and `Figure` for the controller. Fast path (`ActionRedraw`) re-renders the current figure with updated data-space viewport (via `Zoomable`); slow path (`ActionRebuild`) calls `Source.Build` again. `ControllerFunc` adapter and `DataSpaceController()` (data-space pan/zoom, per-panel hit-testing, double-click reset); `NewSession`, `WithController`, `WithExportSurface`, `Session.Run(ctx)`.
- **`output.Session` async/debounced rebuild** — `WithRebuildDelay(d)` makes `ActionRebuild` non-blocking: rapid triggers within `d` coalesce into a single background `Source.Build`, the last good figure keeps drawing while it computes, and the result swaps in when ready. Pending rebuilds flush on event-channel close and are cancelled on `Close`/context cancellation. `WithRebuildError(fn)` handles non-fatal background-build errors. Default (no delay) stays synchronous.
- **`output/window` — native desktop GPU window** (Phase 5, `//go:build !js`). `window.Show(ctx, src, opts...)` opens a `gogpu` window and presents the figure zero-copy through `gg/integration/ggcanvas` (`Render` onto the swapchain surface). Reuses the Phase-4 `Controller`/`State` policy driven from gogpu's frame and input callbacks rather than `Session.Run` (gogpu owns the run loop). Options: `WithTitle`, `WithSize`, `WithController`, `WithRebuildDelay`, `WithRebuildError`, `WithFPS`, `WithPprof`. Default controller is `DataSpaceController()`. *(`output/web` (Phase 6) is still pending.)*
- **`output/window` uses direct `ggcanvas.Canvas` integration** — draws the figure into `gg.Context` via `ggcanvas.Canvas.Draw()` and presents via `ggcanvas.Canvas.Render()` (zero-copy `RenderDirect` when GPU is available, universal CPU fallback otherwise). Replaces the previous `gogpu/ui` widget + `SceneCanvas` approach which had device-scale mismatch issues on HiDPI displays.

#### Removed
- **`canvas.SceneCanvas`** — removed. The `scene.Scene`-based canvas adapter is no longer needed; all rendering now goes through `canvas.RasterCanvas` wrapping the `gg.Context` provided by `ggcanvas.Canvas.Draw()`.
- **`gogpu/ui` dependency** — the window no longer uses the `gogpu/ui` widget system (`app`, `desktop`, `widget`, `event`, `geometry`). Instead it uses bare `gogpu.App` + `ggcanvas.Canvas`, exactly matching the official ggcanvas integration pattern.
- **`output.State.OffsetX/OffsetY/Scale`** — the affine viewport transform fields have been removed from `State`. All interaction now uses the data-space model via `Zoomable.SetPanelViewport` / `ResetViewport`. `State` now contains only `Bounds` and `Figure`.
- **`output.DefaultController()`** — the affine-based pan/wheel-zoom controller. Replaced by `DataSpaceController()` which operates on scale bounds instead of canvas transforms.
- **`output.DrawViewport()`** and `viewportFigure` — the affine viewport rendering helpers. `Session.render()` and `output/window` now call `fig.Draw()` directly.

#### Dependencies
- Added `github.com/gogpu/gogpu` (desktop window/GPU app, used by `output/window`). Bumped `github.com/gogpu/wgpu` 0.28.7 → 0.29.1 and `go-webgpu/goffi`; added `go-webgpu/webgpu`. Rendering goldens are unaffected.
- Removed `github.com/gogpu/ui` — no longer needed.
- Promoted `github.com/gogpu/gpucontext` from indirect to direct dependency (used for `ScrollEventSource` type assertion).

#### Fixed
- **SVG/PDF vector export was double-transforming geometry.** The recording recorder bakes the active transform into every coordinate, but the SVG/PDF backends *also* re-emitted it (`<g transform>` / `cm`), so the panel's data layer (points, lines) rendered shifted off the axes. The backends now use the baked world coordinates for paths/rects verbatim and never re-apply the matrix to geometry (`SetClip`/`ClearClip` stay no-ops). PNG/raster output was unaffected.
- **Vector text ignored its anchor.** `RecordingCanvas.DrawStringAnchored` now pre-applies the anchor with the same metrics the raster path uses (the recording playback drops the anchor and font face), so tick labels, the title, and axis labels are aligned in SVG/PDF instead of left-/baseline-anchored.
- **Rotated text rendered upright in SVG/PDF** (axis titles, facet strips, slanted tick labels). The recorder bakes only the anchor *position*, not the glyph *orientation* — but it does record the active transform. The backends now track the CTM (without applying it to baked geometry) and recover the rotation for text only: SVG emits `transform="rotate(deg x y)"`, PDF bakes the rotation into the text matrix `Tm`. New regression tests in `canvas/export_test.go`.
- **`output/file`: `Commit` now propagates file close errors** instead of silently swallowing them. A filesystem error on `Close()` (e.g. disk full during buffered flush) is no longer lost.
- **`output/window`: `window.Show` now supports async/debounced rebuilds** via `WithRebuildDelay` and `WithRebuildError`, preventing UI freezes when `Source.Build` is slow (e.g. BigQuery-backed datasets). Rapid rebuild requests coalesce; the build runs in a background goroutine while the last good figure keeps drawing.
- Added example `examples/output/` exercising `file.Save`/`image.Render`/`file.Encode`, `output.NewSurface`+`Render`.
- Added example `examples/window/` — a runnable `window.Show` program (interactive pan/zoom desktop window); needs a display/GPU, so it is not run in CI.
- Added example `examples/session/` — the `output.Session` loop driven headless: a frame-capturing `LiveSurface` scripts pan/zoom/rebuild events through a real `Session` (with `WithRebuildDelay`) and writes each frame to a PNG; runs without a display.

> Known limitation: vector text renders at a single default size because the recording playback hands the backend a nil font face and drops the recorded font size; fixable in the gg recording layer (forward `FontSize`/face through `Backend.DrawText`).

- **`Plot.WriteTo`** — removed. Use `file.Encode` (`output/file`).
- **`Plot.Save`** — removed. Use `file.Save` (`output/file`).
- **`Plot.Encode`** — removed. Use `file.Encode` (`output/file`).
- **`Plot.Image`** — removed. Use `image.Render` (`output/image`).
- **`Built.RenderTo`** — removed. Use `output.Render` directly.

## [0.0.9] — 2026-05-28

### Changed

#### Coordinate Transforms — Engine-Native Dispatch
- **`coord.TransFunc`** is now a pure specification type (`Name string` only). All math (Forward/Inverse closures, safe helpers) removed from the `coord` package.
- **`applyCoordTransform`** dispatches directly to named `MathKernel` operations (`mk.Log10`, `mk.Sqrt`, `mk.Neg`, etc.) — no `MapFloat64` scalar fallback. Arrow compute kernels, SIMD highway, and SQL math functions are used natively.
- **`coordTickFormatter`** resolves transform names to scalar inverses for tick labels via `scale.FormatNumber`. Tick formatting is inherently scalar (one value per label).
- **`Transformer` interface** simplified to `XTrans()`/`YTrans()` returning `TransFunc`. Removed `FormatTickX`/`FormatTickY` methods.
- Added `ErrUnsupportedTransform` sentinel error for unknown transform names.

#### Faceting API (Phase 11 — Breaking)
- **`Plot.FacetWrap(col, opts ...WrapOpt)`** — signature changed from positional `(col string, nCols, nRows int)` to variadic functional options.
- **`Plot.FacetGrid(row, col, opts ...GridOpt)`** — new method; replaces the old `(row, col string)` two-arg form with variadic functional options.
- **`facet.Grid(row, col, opts ...GridOpt)`** — constructor signature changed to accept `GridOpt` functional options.
#### Faceting API (Phase 12 — Breaking)
- **`facet.Facet`** interface — added `FreeScales() (freeX, freeY bool)`, `SpaceMode() string`, and `StripPositions() (col, row string)` methods.
- **`Plot.SecondaryY`** renamed to **`Plot.SecondAxis`** for clarity.
- **`PlotSpec.SecondaryY`** renamed to **`PlotSpec.SecondAxis`**.
- **`Built.freeX`/`freeY`** moved from `Built` struct to `Layout` struct as `Layout.FreeX`/`Layout.FreeY` — these are facet-level layout properties, not per-panel state.

### Fixed
- **`clone()` dropped spec fields** — `SecondAxis`, `ThemeOverrides`, `ColorBarWidth`, `ColorBarNBin`, `LegendNCols`, `SizeScale`, `AlphaScale`, `ShapeScale`, and `LinetypeScale` were not copied by `Plot.clone()`, causing any builder method called after setting these to silently drop them.

### Added

#### Advanced Faceting (Phase 12)
- **Secondary Y-Axis** — `Plot.SecondAxis(scale.SecAxisSpec)` adds a right-side Y-axis derived from the primary via closure-based transform pair.
  - `scale.SecAxis(trans, inverse, name)` — constructor for custom transforms (e.g. °C → °F).
  - `scale.DupAxis(name)` — identity transform for mirroring the primary axis.
  - `scale.DerivedScale` — composite scale that applies `SecAxisSpec` to a primary scale's ticks/bounds.
  - Right margin auto-measured from secondary tick labels and title.
  - `drawYAxisRight` — new axis renderer with ticks/labels on the right side.
- **Free Scales** — per-panel independent axis bounds for faceted plots.
  - `facet.FreeX()`, `facet.FreeY()`, `facet.FreeXY()` — Wrap options.
  - `facet.GridFreeX()`, `facet.GridFreeY()` — Grid options.
  - Post-build scale unification — union bounds computed across all panels for shared axes.
  - Per-panel axis labels drawn when scales are free (not just left column / bottom row).
- **Free Space** — `facet.GridSpace(mode)` option for proportional panel sizing (`"fixed"`, `"free"`, `"free_x"`, `"free_y"`).
- **Strip Placement** — `facet.StripBottom()`, `facet.GridStripBottom()`, `facet.GridStripLeft()` options for repositioning facet strip labels.
- **Strip Styling Granularity** — `strip.text.x`, `strip.text.y`, `strip.background.x`, `strip.background.y` theme inheritance paths for axis-specific strip styling.
- **Theme Overrides** — `Plot.ThemeOverride(...theme.Override)` for per-plot element overrides without creating custom themes.
  - `theme.Override{Path, Elem}` type for specifying element overrides.
  - `theme.WithOverrides(th, ...)` — functional theme copy with overrides.
  - Convenience constructors: `theme.AxisTitleXOverride`, `theme.StripTextOverride`, `theme.StripBackgroundOverride`, `theme.PlotTitleOverride`, etc.
- New example: `examples/secondary_axis/` — 2 plots (°C→°F transform, DupAxis mirror).
- New example: `examples/facet_free_scales/` — 4 plots (shared, free_y, free_xy, strip style override).
- 7 new tests in `scale/scale_test.go` covering `DerivedScale` (bounds, ticks, manual breaks, map/inverse roundtrip, format, string) and `DupAxis`.


#### Faceting Deep Dive (Phase 11)
- **Labellers** — `facet.LabelValue()`, `facet.LabelBoth()`, `facet.LabelContext()`, and `facet.Label(func(col, val string) string)` custom labeller. `LabelContext` resolves to `LabelValue` for Wrap and `LabelBoth` for Grid.
- **`facet.GridOpt`** functional options — `GridLabeller(l)`, `GridDrop(bool)`, `GridMargins(bool)`.
- **`facet.WrapOpt`** expanded — `WithLabeller(l)`, `WithDrop(bool)`, `NCols(n)`.
- **`facet.Panel`** — added `RowVal`, `ColVal` (raw facet values), `NumRows` (row count from mask, no materialization), and `IsMargin` (aggregate margin flag).
- **Grid margins** — `GridMargins(true)` produces row-margin panels (aggregate across columns), column-margin panels (aggregate across rows), and corner margin (full dataset). Margin panels use `"All"` label and are placed at the grid edges.
- **Drop control** — `GridDrop(false)` / `WithDrop(false)` preserves empty panels for missing value combinations.
- **`facet.MarginLabel`** constant — `"All"`, the display value for aggregate margin panels.
- **Grid cell placement** — `PanelLayout.Row`/`.Col` computed from `RowVal`/`ColVal` index maps instead of sequential panel index. Ensures margin panels render in correct grid cells.
- **Panel-parallel mask helpers** — `maskHasTrue(mask)` for drop check, `maskCount(mask)` for `Panel.NumRows`. Both avoid dataset materialization.
- **Strip labels** — column headers on first grid row, rotated row labels on rightmost grid column. `PanelBorder().Color` nil-guarded for themes without panel borders (e.g. "minimal").
- New example: `examples/facet_labeller/` — 4 plots demonstrating Wrap+LabelBoth, Grid+Margins, Wrap+Drop=false, Grid default labels.
- 29 new tests in `facet/facet_test.go` covering labellers, drop, margins, column types, and grid dimensions.

#### Advanced Geometries (Phase 8 — Batch A)
- **`geom.Crossbar()`** — box with median line between ymin/ymax (no whiskers). Requires `x`, `y`, `ymin`, `ymax` aesthetics.
- **`geom.Linerange()`** — vertical line from ymin to ymax without caps. Thin variant of ErrorBar.
- **`geom.Pointrange()`** — point at `(x, y)` with vertical line from ymin to ymax. Combines point + linerange.
- **`geom.Curve()`** — quadratic bezier curves from `(x, y)` to `(xend, yend)` via `Canvas.QuadraticTo`. Curvature controlled by `geom.WithCurvature()`.
- **`geom.Violin()`** — mirrored kernel density estimate per group. Uses new `stat.ViolinY` with KDE. Supports `WithViolinScale("area"|"count"|"width")`.
- **`geom.Dotplot()`** — stacked-dot plots. Uses new `stat.DotBin` with Wilkinson-style greedy binning. Also supports `"histodot"` mode.
- **`stat.ViolinY()`** — dedicated stat transform for violin plots. Groups by X, runs KDE per group, outputs `x`, `y`, `xmin`, `xmax`, `violinwidth` columns.
- **`stat.DotBin()`** — dot-stacking stat transform. Wilkinson-style greedy binning outputs one row per dot with stacked Y positions.
- **`geom.WithCurvature()`** — functional option to control bezier curvature (default 0.5).
- **`stat.WithViolinBandwidth()`**, **`stat.WithViolinPoints()`**, **`stat.WithViolinScale()`** — violin stat options.
- **`stat.WithDotBinWidth()`**, **`stat.WithDotMethod()`** — dotbin stat options.
- Six new `geom.Type` constants: `TypeCrossbar`, `TypeLinerange`, `TypePointrange`, `TypeCurve`, `TypeViolin`, `TypeDotplot`.
- `geom.OptCurvature` flag and `paramRelevance` entries for all new geom types.
- Six new geometry examples: `examples/geometries/{crossbar,linerange,pointrange,curve,violin,dotplot}/`.

#### Advanced Geometries (Phase 8 — Batch B)
- **`geom.Raster()`** — dense pixel-aligned image grid. Composites the entire grid as a single `image.RGBA` via native canvas transforms (`Save/Translate/ScaleXY/DrawImage/Restore`) for GPU-accelerated upscaling. Options: `WithInterpolate(bool)`, `WithAlpha`.
- **`geom.JitterPoint()`** — convenience constructor for jittered scatter plots (`TypePoint` + `position.Jitter`). Configurable via `WithJitterWidth`, `WithJitterHeight`, `WithJitterSeed`. Deterministic output via `math/rand/v2.PCG` seeded by `(seed, dataLen)`.
- **`position.Jitter`** enhanced with `WithSeed(uint64)` functional option for injectable PRNG seed. `WithJitterWidth`/`Height`/`Seed` options modify the position via internal type assertion — no construction state on `Layer`.
- New geometry examples: `examples/geometries/{raster,jitter_point}/`.

#### Annotations (Phase 9)
- **`Plot.Annotate()`** — layer-less annotation API. Annotations are fixed-coordinate visual elements that bypass the data/stat/position pipeline entirely. They are drawn in data space after all data layers.
- **`AnnotateText(x, y, label, ...opts)`** — text at data coordinates.
- **`AnnotateRect(xmin, ymin, xmax, ymax, ...opts)`** — filled rectangle spanning a region.
- **`AnnotateSegment(x, y, xend, yend, ...opts)`** — line segment between two points.
- **`AnnotateArrow(x, y, xend, yend, ...opts)`** — segment with arrowhead at endpoint.
- **`AnnotateLabel(x, y, label, ...opts)`** — text with filled background box. Padding configurable via `geom.WithPadding(px)`.
- **`geom.TypeLabel`** — new geometry type constant for label-with-background.
- **`geom.WithPadding(px)`** — new functional option for label background box padding (default 4px).
- New `AnnotationType` enum: `AnnotationText`, `AnnotationRect`, `AnnotationSegment`, `AnnotationArrow`, `AnnotationLabel`.
- `PlotSpec.Annotations` field for carrying annotations through the build pipeline.
- New annotation example: `examples/annotations/annotate/`.
- Seven new tests: `TestAnnotation_{Text,Rect,Segment,Arrow,Label,Combined,Save_PNG}`.

#### Coordinate Systems (Phase 10)
- **`Plot.CoordCartesian(xmin, xmax, ymin, ymax)`** — viewport zoom without data clipping. Unlike `XLim`/`YLim` which set scale bounds early, `CoordCartesian` overrides bounds after scale training — all data participates in stat computations, only the visible window changes. Pass `math.NaN()` for any endpoint to auto-detect.
- **`Plot.CoordFixed(ratio)`** — fixed aspect ratio. `ratio = 1` gives equal scaling — one unit of x occupies the same pixel length as one unit of y. The panel dimension is shrunk and centered to enforce the ratio without distorting data.
- **`coord.CartesianZoom(xlim, ylim)`** — low-level constructor. Implements `coord.Zoomer` interface.
- **`coord.Fixed(ratio)`** — low-level constructor. Implements `coord.Fixer` interface.
- **`coord.Zoomer`** interface — optional interface for coordinate systems with viewport zoom bounds.
- **`coord.Fixer`** interface — optional interface for coordinate systems with fixed aspect ratio.
- Annotation rendering split into background pass (rect before data layers) and foreground pass (text/label/segment/arrow after data layers) so filled rectangles don't cover data lines.
- Default label annotation alpha lowered from 0.9 to 0.75 for better transparency.
- New examples: `examples/coord_cartesian_zoom/`, `examples/coord_fixed/`.
- Ten new tests: `TestCoordCartesianZoom_{Build,ScaleBoundsOverridden,Save_PNG,PartialZoom,CoordInterface}`, `TestCoordFixed_{Build,Save_PNG,Render,CustomRatio,CoordInterface}`.

#### Axis Guide (Phase 5b)
- **`AxisLabelRows(n)`** — X-axis label staggering for dense categorical axes. Auto-dodge (n=0) measures label widths and staggers to 2 rows when overlap detected. Explicit n≥2 distributes labels across n rows. Overlap skipping hides colliding labels within each row (tick marks remain).
- Bottom margin now accounts for dodge rows: `(nRows−1) × (fontSize+4)` extra pixels.
- New example: `examples/axis_label_rows/`.

### Fixed
- **`PanelBorder().Color` nil dereference** — themes without a panel border (e.g. "minimal") crashed when rendering facet strip labels. Added nil guard with fallback grey.
- **Grid margin panel placement** — margin panels were placed using sequential `pi / cols` index, causing incorrect grid cell assignment. Now uses `RowVal`/`ColVal` index maps for correct placement.
- **Auto-expand for width-based geoms** — Crossbar, Violin, and Dotplot now automatically get X-axis padding (like Bar and BoxPlot), so elements at domain edges are not clipped.
- **Distinct-X padding calculation** — X-axis padding for width-based geoms now counts distinct X positions instead of raw row count. This fixes near-zero padding for stat-transformed data (e.g., ViolinY outputs 128 grid points per group, making row-based padding negligible).
- **Dotplot Y-axis rendering** — Dotplot now uses coordinate-based Y positioning through the normalize/transform pipeline. Dot radius is computed from Y domain spacing so adjacent dots touch visually. Previously dots were pixel-stacked at the baseline and ignored the Y scale entirely.
- **Dotplot Y-domain floor** — Dotplot Y axis now anchors at zero (like Bar, Histogram, Area) so the baseline aligns with the stacking origin.
- **Bottom margin clipping with dodged labels** — X-axis title (e.g. "Category") was clipped when `AxisLabelRows(n)` staggered labels across multiple rows. Margin now grows by `(n−1) × rowHeight`.
- **Auto-dodge margin prediction** — When `AxisLabelRows(0)` (auto), the margin computation now pre-measures label widths against available plot width to predict whether 2-row staggering will trigger.

### Changed
- `docs/ROADMAP.md` Phase 8 updated: 15 of 17 items shipped; remaining items: Contour, Hex.

#### Axis Label Dodge (Phase 5b)
- **`Plot.AxisLabelRows(n)`** — auto-detect overlapping X-axis labels and stagger across `n` rows. When `n=0` (default), overlap is measured and labels are auto-staggered to 2 rows when collisions are detected. `n=1` disables staggering; `n≥2` forces that many rows.
- **X-axis overlap skipping** — within each dodge row, labels that would still overlap are hidden (tick marks remain visible).
- **Rotated label margin fix** — bottom margin now accounts for projected label height when `ElementText.Angle ≠ 0`, preventing label clipping.
- **`AxisGuide` type** — new struct in `spec.go` carrying axis guide configuration.

#### Dense Pixel Grid (Phase 8)
- **`geom.Raster()`** — dense pixel-aligned image grid geometry. Composites all cells into a single `image.RGBA` and renders via `Canvas.DrawImage` with native `Save/Translate/ScaleXY/DrawImage/Restore` transform-based compositing — GPU texture sampler handles upscaling (zero Go-side pixel loops).
- **`geom.WithInterpolate(bool)`** — option for bilinear interpolation (reserved for canvas-level control; currently both paths use native canvas sampling).
- **`geom.TypeRaster`**, **`geom.OptInterpolate`** — type constant and option flag.
## [0.0.8] — 2026-05-24

### Added

#### Aesthetics Mapping (Phase 7)
- **`aes.Size(col)`** — map a continuous column to point radius.
- **`aes.Alpha(col)`** — map a continuous column to opacity.
- **`aes.Shape(col)`** — map a categorical column to point shape.
- **`aes.Linetype(col)`** — map a categorical column to line dash pattern.

#### New Scales
- **`scale.SizeScale`** — continuous size mapping with `SizeModeLinear`, `SizeModeArea`, `SizeModeRadius`. Constructors: `NewSize(min, max)`, `NewSizeDefault()`, `NewSizeArea()`.
- **`scale.AlphaScale`** — continuous opacity mapping. Constructors: `NewAlpha(min, max)`, `NewAlphaDefault()`.
- **`scale.ShapeScale`** — discrete shape mapping with 10-shape default cycle. Constructors: `NewShape()`, `NewShapeManual(map)`.
- **`scale.LinetypeScale`** — discrete linetype mapping with 6 dash patterns. Constructors: `NewLinetype()`, `NewLinetypeManual(map)`.
- **`scale.IdentityScale`** — pass-through scale for raw pixel/opacity/color values.
- **`scale.ValueMapper`** — named optional interface for scales providing `MapValue(float64) float64`.

#### Shape Constants (canvas)
- **Exported shape constants**: `canvas.ShapeCircle`, `ShapeSquare`, `ShapeTriangle`, `ShapeTriangleDown`, `ShapeDiamond`, `ShapePlus`, `ShapeCross`, `ShapeStar`, `ShapePentagon`, `ShapeHexagon`.
- **`canvas.Shapes()`** — returns the ordered default shape cycle.
- **`canvas.IsStrokeShape()`** — reports whether a shape should be stroked instead of filled.
- **`canvas.DrawShapePath()`** — path-based fallback using `drawRegularPolygonPath()` matching `gg.Context.DrawRegularPolygon` orientation.

#### Canvas Backend
- **`GGCanvas.DrawShape()`** — uses `gg.Context.DrawRegularPolygon` natively for polygon shapes (triangle, square, diamond, pentagon, hexagon), eliminating manual vertex math.

#### Builder API
- **`Plot.ScaleSize(min, max)`**, **`ScaleSizeArea()`** — configure size scale.
- **`Plot.ScaleAlpha(min, max)`** — configure alpha scale.
- **`Plot.ScaleShape()`**, **`ScaleShapeManual(map)`** — configure shape scale.
- **`Plot.ScaleLinetype()`**, **`ScaleLinetypeManual(map)`** — configure linetype scale.
- **`Plot.ScaleSizeIdentity()`**, **`ScaleAlphaIdentity()`** — identity scale overrides.

#### Examples
- **`examples/phase7_aesthetics/`** — comprehensive demo of all Phase 7 aesthetics.

### Changed
- `NewAlpha(0, 0)` now means literal zero range (not defaults). Use `NewAlphaDefault()` for `[0.1, 1.0]`.
- `NewSize(0, 0)` now means literal zero range. Use `NewSizeDefault()` for `[1.0, 6.0]`.
- `SizeScale.String()` returns `"size"` (was `"size:linear"`). Use `Mode()` to query the sizing mode.
- `NewShapeManual()` / `NewLinetypeManual()` now defensively copy the map argument (`maps.Clone`).
- `SizeScale.MapValue()` / `AlphaScale.MapValue()` clamp normalized value to `[0, 1]` before range interpolation.
- `drawPoints` / `drawLine` use `scale.ValueMapper` named interface instead of anonymous interface assertions.
- All shape string comparisons in `drawer.go` and `ggplot.go` use `canvas.Shape*` constants and `canvas.IsStrokeShape()`.

### Fixed
- **Map aliasing bug**: `NewShapeManual(m)` and `NewLinetypeManual(m)` no longer store the caller's map by reference.
- **Out-of-range MapValue**: Values outside the trained domain no longer produce negative radii or opacity > 1.0.

## [0.0.7] — 2026-05-22

### Added

#### CIELAB Gradient Constructors (Phase 6)
- **`colormap.Gradient(low, high)`** — 2-stop perceptually uniform gradient using CIELAB interpolation.
- **`colormap.Gradient2(low, mid, high)`** — 3-stop diverging gradient with midpoint at t=0.5.
- **`colormap.GradientN(colors)`** — N-stop multi-color gradient with evenly spaced breakpoints.
- Full `Cmap` interface support: `Reversed()`, `Resampled()`, `WithExtremes()`, clamping, NaN handling.
- sRGB ↔ D65 XYZ ↔ CIELAB conversion helpers for round-trip-accurate colour space transforms.

#### Color/Fill Split in Theme Defaults (Phase 6)
- **`ColorDefaults`** now has independent `ColorDiscrete`, `ColorSequential`, `FillDiscrete`, `FillSequential` fields.
- Fill fields fall back to corresponding Color fields when nil — no duplication needed.
- **`theme.DefaultCmapFor(name, aesthetic, category)`** — resolves `AesColor` vs `AesFill` for correct default palette dispatch.
- All 75 themes updated with split Color/Fill defaults (e.g., dark themes: Fill=Inferno, Color=Observable10).

#### NA Color on colormap.Scale (Phase 6)
- **`colormap.Scale.SetNAColor(c *gg.RGBA)`** — user-configurable color for missing/NaN values.
- `nil` reverts to the cmap's default bad color (transparent black).

#### Legend Key Glyphs (Phase 6)
- **`LegendGlyph`** type with three variants: `GlyphRect`, `GlyphPoint`, `GlyphLine`.
- Legend keys auto-match their geom type: circles for `geom.Point`/`Rug`, lines for `geom.Line`/`Smooth`/`Step`/`Segment`, rectangles for bars/tiles.
- **`drawGlyph()`** helper renders the appropriate shape in both vertical and horizontal legends.

#### Guide Customization API (Phase 6)
- **`Plot.ColorBarWidth(w)`** — set continuous color bar width in pixels.
- **`Plot.ColorBarNBin(n)`** — set number of discrete gradient steps in the color bar.
- **`Plot.LegendCols(n)`** — set number of columns for categorical legends.
- `ColorBarSpec.BarWidth`/`NBin` fields wired into `drawColorBar()`.

#### Examples
- **`examples/phase6_color_scales/`** — 6 example functions generating 8 PNGs: CIELAB gradient, diverging gradient, multi-stop terrain, theme-aware auto-selection (default/dark/okabe_ito), legend key glyphs, guide customization.

### Changed
- Build pipeline now uses `theme.DefaultCmapFor()` for all colour scale fallbacks — themes control default palettes, not hard-coded constants.
- Removed redundant `themePaletteCmap()` function.
- `grouped_color` golden test updated (legend key shape changed from rectangle to circle for `geom.Point` — intentional visual improvement).

## [0.0.6] — 2026-05-22

### Added

#### Temporal Data Types (Phase 1)
- **`DTypeTimestamp`**, **`DTypeDate`**, **`DTypeTime`** — first-class temporal column types stored as `int64` (nanoseconds or days-since-epoch).
- **Memory engine**: `NewTimestampColumn(name, []time.Time)`, `NewDateColumn(name, []date.Date)`, `NewTimeColumn(name, []dataset.TimeOfDay)`.
- **Arrow engine**: Maps `arrow.Timestamp`, `Date32`, `Date64`, `Time32`, `Time64` → ggplot `DType`.
- **BigQuery engine**: Maps `DateFieldType` → `DTypeDate`, `TimeFieldType` → `DTypeTime`, `TimestampFieldType` → `DTypeTimestamp`.
- **`dataset.ParseTimestamp`/`ParseDate`**: Convenience parsers using `dateparse` and `rickb777/date/v2`.

#### DateTime Scale (Phase 5)
- **`scale.DateTime`** — auto-detecting time-series scale with calendar-aligned ticks.
- Granularity detection: second → minute → hour → day → month → year.
- Intraday spans (<2 days) show time-only labels (`"15:04"`).
- Uses `time.Local` timezone for tick formatting.

#### Binned Scale (Phase 5)
- **`scale.Binned`** — discretize continuous axes into range-labeled bins.
- **`scale.WithBins(n)`** — `scale.Opt` for explicit bin count.
- **`scale.WithBinBreaks([]float64)`** — `scale.Opt` for explicit bin edges.
- Default: 7 bins via Sturges heuristic.
- `Format()` returns `[lo, hi)` range labels (last bin `[lo, hi]`).

#### Out-of-Bounds Policies (Phase 5)
- **`scale.OOBPolicy`** type: `OOBKeep` (default), `OOBSquish`, `OOBCensor`.
- **`scale.WithOOB(policy)`** — composable with `WithClipBounds`.

#### Opt-in Driver Packages
- **`canvas/gpu`** — blank-import package for GPU acceleration (moved from `canvas/gg.go`).
- **`dataset/memory/csv`** — CSV handler for memory engine.
- **`dataset/memory/parquet`** — Parquet handler for memory engine.
- **`dataset/arrow/csv`** — CSV handler for Arrow engine.
- **`dataset/arrow/parquet`** — Parquet handler for Arrow engine.

#### Axis Text Rotation
- **`theme.ElementText.Angle`** — axis label rotation angle (degrees), rendered via right-aligned anchor for overlap prevention.

#### Examples
- **`examples/temporal/`** — timestamp, date, time, and datetime scale examples.
- **`examples/scale_config/09_binned.png`** — binned scale example with explicit breaks.

### Changed
- GPU import removed from `canvas/gg.go` — default binary is CPU-only (~8 MB stripped).
- CSV/Parquet handlers extracted from `dataset/memory` and `dataset/arrow` into subpackages; engine packages focused on column construction.
- `dataset.Engine` interface extended with `RegisterCSVHandler` and `RegisterParquetHandler` for driver registration.
- Golden test PNGs updated for rendering improvements.

### Fixed
- **BinnedScale rendering**: `Bounds()` now returns data-space domain instead of index-space, fixing blank/mispositioned points in the rendering pipeline.
- **Intraday DateTime labels**: `detectGranularity` uses `"15:04"` format for spans under 2 days.

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
