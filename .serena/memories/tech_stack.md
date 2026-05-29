# Tech Stack

- Language: Go **1.26.0** (`go.mod`). Module `github.com/TuSKan/ggplot`.
- Lint: **golangci-lint v2** (CI pins v2.12.2), config `.golangci.yml` (`default: all` minus a curated disable list; `gomodguard_v2` enabled).
- No Makefile/Taskfile — plain `go` toolchain.

## Key deps
- `github.com/gogpu/gg` — 2D vector rendering (anti-aliased). **Replaced** in go.mod with `github.com/TuSKan/gogpugg` fork.
- `github.com/apache/arrow-go/v18` + `parquet-go` — Arrow/Parquet engine.
- `cloud.google.com/go/bigquery` + `google.golang.org/api` — BigQuery engine.
- `golang.org/x/sync` (errgroup, panel-parallel), `golang.org/x/image`, `go-text/typesetting` (via gg).
- gogpu/wgpu/gpucontext — optional GPU window + browser (js/wasm) surfaces.

## SIMD note
`GOEXPERIMENT=simd` appears in CONTRIBUTING but is NOT required: `dataset/compute` deliberately uses scalar loops (go-highway `Vec[T]` heap-escapes → ~8x slower). Standard `go build`/`go test` works without it.

## Output formats
PNG (raster, default), SVG 1.1, PDF 1.4 (vector via `canvas/export_*.go` replaying a RecordingCanvas).
