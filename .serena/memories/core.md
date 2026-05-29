# Core

`github.com/TuSKan/ggplot` — pure-Go Grammar of Graphics plotting library (ggplot2-inspired). Go 1.26+.

## Authoritative docs (read these before deep work)
- `docs/ARCHITECTURE.md` — canonical package map + rendering pipeline. Trust over `CONTRIBUTING.md` (latter has stale paths: mentions `internal/canvas`, `internal/grammar`, `draw.go`, `stat/stat.go` — real layout is `canvas/`, `drawer.go`, `stat/transform.go`).
- `docs/DATASET.md` — engine abstraction deep dive.
- `docs/OUTPUT-SPEC.md`, `docs/ROADMAP.md`, `docs/BENCHMARK.md`.
- `CLAUDE.md` (repo root) — condensed agent guide.

## Pipeline (memorize)
`PlotSpec → Facet Split → Stat Transform → Scale Training → Layout → Draw → Surface`
Three phases: **Build** (`Plot.Build`→`output.Figure`/`*Built`), **Draw** (`Built.Draw` onto `canvas.Canvas`), **Output** (`output` pkg → `Surface`). Multi-panel Build and data-layer Draw run panel-parallel via `errgroup`; chrome drawn sequentially. Layer dispatch by `geom.Type` → `draw*` funcs in `drawer.go`.

## Source map (root + packages)
- Root: `ggplot.go` (Plot builder + Build/Draw orchestration), `drawer.go` (geometry drawing), `annotate.go`, `spec.go` (PlotSpec), `errors.go` (`*Error{Phase,Layer,Stage,Cause}`).
- `aes/` mapping ctors; `geom/` geoms+options+`position.go`; `stat/` transforms; `scale/`; `coord/` (pure spec, no math); `facet/`; `theme/` (60+, inheritance); `canvas/` (drawing seam); `colormap/`; `guide/`; `output/` (+ file/image/window/web surfaces); `dataset/` (memory|arrow|bigquery engines, `engine.go` interfaces); `examples/` (one dir per geom/feature, generate README images).

## Invariants
- Immutable builder: every `Plot` method `clone()`s and returns new `Plot`.
- Errors wrap `*ggplot.Error` (`errors.Is/As/Unwrap`).
- Rendering must be deterministic unless randomness documented (e.g. Jitter seed). No map-order-dependent output.

## Domain & process memories
- Tech/build/versions: `mem:tech_stack`
- Commands (Windows shell specifics): `mem:suggested_commands`
- Code style + hard repo rules (git ban, stat/ laziness, API stability): `mem:conventions`
- Acceptance gate before declaring done: `mem:task_completion`
