# Suggested Commands

System: **Windows** (PowerShell 7+ default shell). Go commands identical to unix; shell utils differ.

## Go workflow
- `go build ./...`
- `go test ./...` — all tests
- `go test ./stat/...` — single package
- `go test -run TestName ./...` — single test by name
- `go test -race -short -count=1 ./...` — race (CI parity)
- `go vet ./...`
- `golangci-lint run`
- Benchmarks: `go test -run=^$ -bench=. -benchmem -count=3 ./...`

## Golden image tests (root package)
- Regenerate after intentional visual change: `go test -run TestGolden -update-goldens .`
- Flag defined in `ggplot_golden_test.go` (`-update-goldens`). Goldens compared by sha256.

## CI (.github/workflows/ci.yml)
Runs on ubuntu+macos+windows: `go mod verify`, `go mod tidy` + `git diff --exit-code` on go.mod/go.sum, tests, golangci-lint (ubuntu), race detector. **Tests must pass on all 3 OSes** — mind path seps, line endings, float formatting, font availability.

## Windows shell specifics (PowerShell, not unix)
- List: `Get-ChildItem` (`ls`); read: `Get-Content`; `head -n` → `Get-Content -TotalCount N`.
- Null redirect: `2>$null` not `2>/dev/null`. Env: `$env:VAR` not `$VAR`. Line continuation: backtick.
- `&&`/`||`/ternary available in pwsh 7+.
- Prefer Serena symbolic tools + Grep/Glob over shell `find`/`grep`/`cat`.
