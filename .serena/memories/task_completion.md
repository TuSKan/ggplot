# Task Completion

Mandatory acceptance gate (`.agents/rules/rules.md`). A task is complete ONLY when all four pass:

```
go test ./...
go mod tidy
go fmt ./...
golangci-lint run
```

- `-short`, `--fast-only`, or single-package runs are fine for iteration but are NOT the acceptance gate.
- If any fails: report failing command, summarize failure, fix if related to the change, re-run full gate.
- Do not weaken/loosen tests to pass. Don't claim verified if you couldn't run the commands — say so explicitly.

## Docs to update before declaring done (when behavior/API changes)
`CHANGELOG.md`, `README.md`, `docs/ROADMAP.md`, `docs/ARCHITECTURE.md`.

## Final report should include
Changed files, verification commands run + result of each, remaining risks/skipped checks.

## Regression tests
Bug fixes need a regression test unless impossible. Dataset changes need tests for: empty, nil, mixed types, missing values, stable output order. Visual changes: update goldens only when intentional + explained.
