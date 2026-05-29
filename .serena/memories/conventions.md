# Conventions

Hard repo rules live in `.agents/rules/` (`rules.md` always-on, `lazy.md`). These are binding, not advisory.

## Git policy (CRITICAL)
**Agent must NEVER run git mutation commands**: add, commit, tag, push, reset, clean, checkout, switch, or anything that mutates tree/branches/commits. User owns version control. Read-only OK: `git status`, `git diff`, `git log`.

## stat/ laziness (CRITICAL — `mem:lazy` topic in lazy.md)
**No materialization inside `stat/` transforms.** Forbidden in stat/: `.Values()`, `.Float64()`, `.Int64()`, `.Collect()`, manual `[]float64` loops, `ReducerFunc`/`func([]float64)float64`, getFloat64Values-style helpers.
Stat transforms compose ONLY engine interfaces (`dataset/engine.go`: Aggregator, MathKernel, Windower, Selector, Filterer, Caster, ColumnFactory, Composer) + lazy `Dataset` verbs (Filter, Arrange, Select, SelectRows, GroupBy.Summarize, WithColumn, Rename). Data materializes only at `Collect(ctx)`.
Allowed `.Collect()`/`.Values()` callers: `dataset/` internals, `dataset.ScalarFloat64`, geom `Draw()` methods, test assertions.
Missing engine op → add to `dataset/engine.go` + implement in memory/arrow/bigquery; never on raw slices in stat/.

## Go style
Idiomatic production Go. Explicit errors, stable public APIs, deterministic tests, small focused changes, table-driven tests. Concrete types when useful; interfaces only when abstraction is intentional. Preserve GoG terminology exported names: Plot, Layer, Aes, Geom, Stat, Scale, Coord, Theme, Guide, Dataset, Renderer — don't rename casually.
Short vars (`x,y,r,g,b,dx,dy,w,h`) and numeric plotting defaults (margins, DPI, alpha, tick count) are fine — don't over-abstract them (lint disables mnd/varnamelen/funlen/goconst for this reason).

## API stability
Don't churn exported names/signatures/package layout/semantics unless task requires. If changing API: document old→new, migration impact, justification.

## Determinism / cross-platform
Stable ordering, no map-iteration-order in output, consistent float formatting, no platform-specific fonts in tests. Prefer SVG string goldens over raster. Explain any tolerance relaxation or golden change (bug fix / intentional / formatting-only / incidental).

## Lint
golangci-lint v2. Don't downgrade `.golangci.yml` or remove linters to pass CI. `//nolint` only when linter is wrong for the domain case, as local as possible, with explanatory comment.

## Adding geom / stat (current layout)
- Geom: type+opts in `geom/geom.go`; `draw*` func in `drawer.go`; wire dispatch switch in `drawer.go`; tests in `ggplot_test.go` (+ golden if visual).
- Stat: implement `stat.Transform` in new `stat/<name>.go`; keep lazy; tests in `stat/`.
