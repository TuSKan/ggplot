# Contributing to ggplot

Thank you for your interest in contributing! This document explains how to set up your development environment and submit changes.

## Prerequisites

- **Go 1.26+** (module-aware) GOEXPERIMENT=simd`
- **Git** for version control
- **golangci-lint** (recommended) for lint checks

## Getting Started

```bash
git clone https://github.com/TuSKan/ggplot.git
cd ggplot
go build ./...
go test ./...
```

## Development Workflow

1. **Fork** the repository and create a feature branch.
2. **Write code** following the existing patterns.
3. **Add tests** — every new feature or bug fix needs a test.
4. **Run checks** before submitting:
   ```bash
   go build ./...
   go test ./...
   go vet ./...
   golangci-lint run
   ```
5. **Submit a pull request** against `main`.

## Code Style

- Follow standard Go formatting (`gofmt`).
- Use meaningful variable names; avoid single-letter names outside of loop indices.
- All exported types and functions must have doc comments.
- Keep packages focused: one responsibility per package.

## Architecture Overview

The library follows the Grammar of Graphics pipeline:

```
Data → Stat Transform → Scale Training → Coord Transform → Geom Rendering → Guide/Axis
```

Key packages:

| Package | Purpose |
|---------|---------|
| `dataset` | Engine-agnostic data abstraction |
| `aes` | Aesthetic mapping definitions |
| `stat` | Statistical transformations |
| `scale` | Scale training and mapping |
| `coord` | Coordinate system transforms |
| `geom` | Geometry definitions |
| `guide` | Axis and legend rendering |
| `theme` | Visual theme configuration |
| `internal/canvas` | Rendering backend abstraction |
| `internal/grammar` | Internal pipeline types |

## Adding a New Geometry

1. Define the geometry type in `geom/geom.go` with appropriate `StatName` and register option flags in `paramRelevance`.
2. Add a `draw*` function in `draw.go` that handles the geometry rendering.
3. Wire the dispatch in the `switch` statement inside `drawLayer` in `draw.go`.
4. Add tests in `ggplot_test.go`.

## Adding a New Stat

1. Implement the `stat.Stat` interface in `stat/stat.go`.
2. Register via `stat.Register()` in `init()`.
3. Add tests in `stat/stat_test.go`.

## Output Formats

The library supports multiple output backends via `recording.Backend`:

- **PNG** — Raster output via `gogpu/gg` (default)
- **SVG** — Vector output via native SVG 1.1 backend
- **PDF** — Vector output via native PDF 1.4 backend

To add a new format:
1. Implement `recording.Backend` in `internal/canvas/export_<format>.go`.
2. Wire it into `Save()` and `WriteTo()` in `ggplot.go`.

## Running Benchmarks

```bash
go test -bench=. -benchmem -count=3 .
```

## License

By contributing, you agree that your contributions will be licensed under the project's existing license.
