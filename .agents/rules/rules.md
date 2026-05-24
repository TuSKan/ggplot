---
trigger: always_on
---

# ggplot Engineering Rules

This repository is `github.com/TuSKan/ggplot`, a Go grammar-of-graphics plotting library.

The agent must treat API stability, rendering correctness, deterministic output, and idiomatic Go design as higher priority than cosmetic style changes.

## Git policy

**The agent must never run git mutation commands.**

Forbidden commands include:

- `git add`
- `git commit`
- `git tag`
- `git push`
- `git reset`
- `git clean`
- `git checkout`
- `git switch`
- any command that mutates branches, commits, tags, or the working tree unexpectedly

The user manages version control.

The agent may run read-only Git commands for inspection:

```bash
git status
git diff
git log
```

The agent must not stage, commit, discard, rewrite, or push changes.

## Mandatory verification

Before reporting any task as complete, the agent must run the full verification gate:

```bash
go test ./...
go mod tidy
go fmt ./...
golangci-lint run
```

The agent must not say that a task is complete unless all commands pass.

If any command fails, the agent must:

1. Report the failing command.
2. Summarize the failure.
3. Fix the issue if it is related to the current changes.
4. Re-run the full verification gate.

Do not replace the final verification gate with weaker alternatives such as:

```bash
go test -short ./...
golangci-lint run --fast-only
go test ./pkg/only/changed
```

Those are acceptable for quick local checks, but they are not the final acceptance gate.

## Preferred workflow

For every non-trivial code change:

1. Inspect the relevant package and tests.
2. Understand the intended grammar-of-graphics behavior before editing.
3. Make the smallest correct change.
4. Run focused tests for the changed package.
5. Run the full verification gate:

```bash
go test ./...
go mod tidy
go fmt ./...
golangci-lint run
```

6. Only then summarize the result.

## Go style policy

Use idiomatic, production-grade Go.

Prefer:

- explicit errors
- stable public APIs
- deterministic tests
- small focused changes
- table-driven tests where useful
- clear package boundaries
- exported names that match grammar-of-graphics terminology
- concrete types when the concrete type is the useful API
- interfaces only when abstraction is intentional and useful

Avoid:

- accidental public API churn
- broad rewrites unrelated to the task
- hidden global mutation
- nondeterministic rendering behavior
- weakening tests to make them pass
- disabling linters without a precise documented reason
- over-abstracting simple plotting concepts
- adding dependencies without a strong reason

## Grammar-of-graphics policy

This is a plotting and grammar-of-graphics codebase.

The agent must preserve conceptual clarity around:

- `Plot`
- `Layer`
- `Aes`
- `Geom`
- `Stat`
- `Scale`
- `Coord`
- `Theme`
- `Guide`
- `Dataset`
- `Renderer`

Do not rename or reshape these concepts casually.

The code should remain understandable to users familiar with grammar-of-graphics systems such as ggplot2, Vega-Lite, Plot, or similar plotting libraries.

Acceptable domain patterns include:

- short variables such as `x`, `y`, `r`, `g`, `b`, `dx`, `dy`, `w`, `h`
- numeric plotting defaults such as margins, DPI, alpha, tick count, stroke width, padding, radius, and line width
- package-level immutable defaults such as default themes, palettes, scales, geoms, stats, and renderers
- renderer/layout functions that are longer than ordinary business logic when they preserve pipeline clarity
- examples that use `fmt.Print` or `fmt.Println` to show generated output

Do not "clean up" plotting logic by abstracting numeric constants if the abstraction makes the rendering code harder to read.

Do not split rendering, scale-training, layout, or guide-building algorithms into many tiny helpers only to satisfy generic complexity rules if doing so makes the graphics pipeline harder to understand.

## Rendering correctness policy

Rendering output must be deterministic unless the API explicitly documents randomness.

For SVG, PNG, or other renderer output:

- preserve stable ordering where possible
- avoid map iteration order affecting output
- keep floating-point formatting consistent
- avoid platform-specific output differences
- make snapshot/golden tests deterministic
- document any intentional nondeterminism

If a renderer changes output, the agent must explain whether the change is:

1. a bug fix;
2. an intentional visual/API change;
3. a formatting-only change;
4. an incidental change that should be avoided.

## Dataset policy

Dataset handling must preserve column semantics, row counts, type information, and missing-value behavior.

The agent must be careful with:

- nil values
- empty datasets
- heterogeneous column types
- categorical values
- numeric conversion
- time values
- pointer values
- map iteration order
- stable column ordering

Do not silently coerce data unless the API explicitly owns that coercion.

If changing dataset behavior, add or update tests covering:

- empty input
- nil input
- mixed types
- missing values
- stable output order

## Lint policy

The repository uses `golangci-lint` v2.

The agent must not:

- downgrade `.golangci.yml`
- remove linters only to make CI pass
- add broad `//nolint` comments without explanation
- use `//nolint` when a small code fix is better
- replace strict linting with `--fast-only` as the final verification gate

A `//nolint` is acceptable only when:

1. the linter is wrong for this plotting/rendering/domain-specific case;
2. the suppression is as local as possible;
3. the comment explains why.

Good example:

```go
value := 0.5 //nolint:mnd // Half-pixel alignment is intentional for crisp stroke rendering.
```

Good example for examples:

```go
fmt.Println(svg) //nolint:forbidigo // Example intentionally prints rendered SVG output.
```

Bad example:

```go
//nolint
```

## Cross-platform policy

Tests must pass on Linux, macOS, and Windows.

The agent must consider cross-platform differences in:

- file paths
- line endings
- floating-point formatting
- image encoding
- font availability
- locale-sensitive formatting
- map iteration order
- filesystem permissions

Do not write tests that depend on platform-specific fonts unless the font is explicitly bundled or mocked.

For visual/golden tests, prefer renderer output that is stable in CI. SVG string comparisons are usually more deterministic than raster image comparisons.

When relaxing a tolerance, explain why.

## Examples policy

Examples are part of the public API.

The agent must keep examples:

- compiling
- simple
- idiomatic
- close to real user workflows
- useful as documentation

Examples may use `fmt.Print`, `fmt.Println`, or `fmt.Printf` when demonstrating output.

Do not make examples artificially complex to satisfy linters.

Do not break example output unless the API behavior intentionally changed.

## Tests

The agent must preserve or improve test coverage for changed behavior.

For bug fixes, add or update a regression test unless impossible.

For plotting behavior, tests should prefer:

- deterministic output
- stable serialized SVG when possible
- explicit expected values
- table-driven tests
- small datasets
- edge cases around empty/missing/nil data
- layout/scale/coordinate invariants

Do not loosen tests just to make implementation easier.

If changing renderer output, update golden/snapshot files only when the visual or serialized change is intentional and explained.

## Benchmarks

Benchmarks should measure meaningful plotting-library operations, such as:

- dataset construction
- aesthetic mapping
- scale training
- layer building
- SVG rendering
- PNG/raster rendering, if applicable
- large dataset rendering
- repeated plot construction

Do not add benchmarks that only measure trivial wrappers.

Benchmark results on GitHub-hosted runners are noisy. Treat CI benchmark artifacts as trend hints, not absolute performance truth.

## Generated files

If the agent runs:

```bash
go generate ./...
```

it must check whether generated files changed.

If generated files changed, the agent must explain:

- what generated them;
- why they changed;
- whether the changes should be committed.

The agent must not silently modify generated files.

## Dependency policy

Avoid adding dependencies unless they clearly improve the library.

Before adding a dependency, the agent must consider:

- API stability
- transitive dependency cost
- maintenance health
- license compatibility
- cross-platform behavior
- whether a small internal implementation is more appropriate

Do not add heavy dependencies for simple tasks.

## Public API policy

Do not change exported names, function signatures, package layout, or documented semantics unless the task explicitly requires it.

If an API change is necessary, the agent must explain:

- the old API;
- the new API;
- migration impact;
- why the change is worth it.

Prefer API names that match grammar-of-graphics concepts.

Avoid leaking renderer internals into the high-level plotting API.

## Completion contract

A task is complete only when:

1. The requested code or documentation change is implemented.
2. The full verification gate passed:

```bash
go test ./...
go mod tidy
go fmt ./...
golangci-lint run
```

3. The final response includes:
   - changed files;
   - verification commands run;
   - result of each command;
   - any remaining risks or skipped checks.

Always update CHANGELOG.md and README.md before reporting a task as complete.

If the agent cannot run the verification commands, it must explicitly say so and must not claim the task is fully verified.