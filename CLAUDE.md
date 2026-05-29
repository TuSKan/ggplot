# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`ggplot` (`github.com/TuSKan/ggplot`) is a pure-Go implementation of the Grammar of Graphics — a declarative, composable data-visualization library inspired by ggplot2. Requires **Go 1.26+**.

## Binding repository rules (`.agents/rules/`)

These are hard constraints, not suggestions:

- **Never run git mutation commands.** `git add`, `commit`, `tag`, `push`, `reset`, `clean`, `checkout`, `switch` — all forbidden. The user owns version control. Read-only inspection (`git status`, `git diff`, `git log`) is fine.
- **No materialization inside `stat/` transforms** (see `.agents/rules/lazy.md`). Stat transforms compose engine interfaces and lazy `Dataset` verbs only. `.Values()`, `.Float64()`, `.Collect()`, manual `[]float64` loops, and `ReducerFunc` are strictly forbidden in `stat/`. If a transform needs an operation the engine lacks, add it to `dataset/engine.go` and implement it in each engine backend — never on raw slices. Allowed `.Collect()` callers: `dataset/` internals, geom `Draw()` methods, test assertions.
- **Verification gate before claiming any task complete** (run all four, all must pass):
  ```bash
  go test ./...
  go mod tidy
  go fmt ./...
  golangci-lint run
  ```
  `go test -short`, `--fast-only`, or single-package tests are fine for iteration but are NOT the acceptance gate. If a command fails, report it, fix it if related to your change, and re-run the full gate.
- **Update docs before reporting complete:** `CHANGELOG.md`, `README.md`, `docs/ROADMAP.md`, and `docs/ARCHITECTURE.md` when behavior/API changes.
- **Determinism:** rendering output must be deterministic unless randomness is documented (e.g. `Jitter(geom.WithSeed(...))`). Avoid map-iteration order affecting output; keep float formatting stable; SVG golden comparisons are preferred over raster.
- **API stability:** don't churn exported names, signatures, or package layout unless the task requires it. Preserve grammar-of-graphics terminology (`Plot`, `Layer`, `Aes`, `Geom`, `Stat`, `Scale`, `Coord`, `Theme`, `Guide`, `Dataset`).

## Common commands

```bash
go build ./...                              # build all packages
go test ./...                               # run all tests
go test ./stat/...                          # test a single package
go test -run TestName ./...                 # run a single test by name
go test -race -short -count=1 ./...         # race detector (CI parity)
go vet ./...
golangci-lint run                           # golangci-lint v2 (config in .golangci.yml)

# Golden image tests (root package)
go test -run TestGolden -update-goldens .   # regenerate golden files after an intentional visual change

# Benchmarks
go test -run=^$ -bench=. -benchmem -count=3 ./...
```

CI (`.github/workflows/ci.yml`) runs lint + tests on Linux/macOS/Windows, checks `go mod tidy` is clean (`git diff --exit-code` on go.mod/go.sum), and runs the race detector. **Tests must pass on all three OSes** — watch for path separators, line endings, float formatting, and font availability.

## Architecture

The authoritative architecture reference is [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). (Note: `CONTRIBUTING.md` has some stale package paths — e.g. it mentions `internal/canvas`, `internal/grammar`, `draw.go`, `stat/stat.go`; the real layout is `canvas/`, `drawer.go`, `stat/transform.go`. Trust ARCHITECTURE.md and the actual tree.)

### Pipeline

```
PlotSpec → Facet Split → Stat Transform → Scale Training → Layout → Draw → Surface
```

`Plot` is an **immutable builder** — every method (`Layer`, `Labs`, `Theme`, `FacetWrap`, …) calls `clone()` and returns a new `Plot`. The pipeline has three phases:

1. **Build** (`Plot.Build` → `output.Figure`/`*Built`): facet split → PANEL column injection → per-panel stat transforms + group splitting → scale training. Multi-panel builds run **panel-parallel** via `errgroup`; single-panel takes a goroutine-free fast path.
2. **Draw** (`Built.Draw` onto a `canvas.Canvas`): chrome (axes, grid, strips, legends) drawn sequentially; data layers drawn **panel-parallel** onto independent CPU sub-canvases then composited via `DrawImage`. Each resolved layer dispatches to a `draw*` function in `drawer.go` by `geom.Type`.
3. **Output** (`output` package): carries a `Figure` to a `Surface` (file / in-memory image / desktop GPU window / browser canvas). Platform surfaces register by blank import. `Plot.Save`/`Encode`/`Image` are façades over `output.Render`.

All pipeline errors wrap `*ggplot.Error{Phase, Layer, Stage, Cause}` (supports `errors.Is`/`As`/`Unwrap`).

### Key packages

- **Root** (`ggplot.go`, `drawer.go`, `annotate.go`, `spec.go`, `errors.go`) — the public `Plot` builder, geometry drawing, annotation API, `PlotSpec`, typed errors.
- **`aes/`** — aesthetic mapping constructors: `X()`, `Y()`, `Color()`, `Fill()`, `Size()`, `Shape()`, `Linetype()`, `Group()`, etc.
- **`geom/`** — geometry definitions + functional options + position adjustments (`position.go`: Identity, Dodge, Stack, Fill, Jitter, Nudge). Sugar constructors (`geom.Histogram()`) wrap full stat pipelines.
- **`stat/`** — `stat.Transform` interface (`Apply`, `OutputMapping`, `OutputSchema`, `OutputHints`). Bin, Count, Density (KDE), Smooth (LOESS+lm), Boxplot, Violin, DotBin, etc. **Must stay lazy** (see binding rules).
- **`scale/`** — Scale interface (`Train`/`Map`/`Inverse`/`Ticks`/`Format`/`Bounds`). `scale.Resolve(name)` maps `"log10"`/`"sqrt"`/`"reverse"` to types. Secondary-axis support via `SecAxisSpec`/`DerivedScale`.
- **`coord/`** — coordinate systems as pure specification (zero math): Cartesian, CartesianZoom, Fixed, Polar, Trans. Math is dispatched to engine `MathKernel`.
- **`facet/`** — `Facet` interface (`Split`, `GridDims`, `FreeScales`, …): None, Wrap, Grid. Mask-based lazy splitting — `Split()` never collects.
- **`theme/`** — 60+ themes with ggplot2-style element inheritance (`axis.title.x` ← `axis.title` ← `text`). Default is **Dashboard**. Registered at init; `theme.Resolve(name)` instantiates.
- **`canvas/`** — the drawing seam. `Canvas` interface; `RasterCanvas` (gg-backed, GPU or CPU); `RecordingCanvas` (records ops, replayed to SVG/PDF vector backends in `export_*.go`). `ggplot.WithCPU()` forces the deterministic CPU rasterizer.
- **`colormap/`** — 60+ palettes (Viridis, ColorBrewer, Tab10, …); CIELAB gradient constructors.
- **`dataset/`** — engine-agnostic columnar `Table`/`Dataset` abstraction with lazy frame verbs. See [`docs/DATASET.md`](docs/DATASET.md). Three engine backends implement the same interfaces:
  - `dataset/memory` — native Go slices (default).
  - `dataset/arrow` — Apache Arrow IPC/Parquet, zero-copy reads.
  - `dataset/bigquery` — lazy SQL pushdown.
  - Engine interfaces live in `dataset/engine.go` (`Aggregator`, `MathKernel`, `Windower`, `Selector`, `Filterer`, `Caster`, `ColumnFactory`, `Composer`). Data materializes only at `Collect(ctx)`.

### Adding a geometry / stat (current layout)

- **Geometry:** define the type + options in `geom/geom.go`; add a `draw*` function in `drawer.go`; wire the dispatch switch in `drawer.go`; add tests in `ggplot_test.go` (and a golden if visual).
- **Stat:** implement `stat.Transform` in a new `stat/<name>.go`; keep it lazy (engine ops only); add tests in `stat/`.

## Examples

`examples/` contains self-contained `main` programs (one dir per geometry/feature) that generate the README/docs images. Examples are part of the public API — keep them compiling, idiomatic, and close to real workflows. They may use `fmt.Print*` to show output.
