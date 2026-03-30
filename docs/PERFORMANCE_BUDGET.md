# Performance Budget & Regression Guardrails

`ggplot` evaluates utilizing explicit zero-copy architectures boundary data structs without scaling unexpected. To keep things absolutely performant evaluating metrics :

## 1. Hot Paths Budgets
* **Dataset Filtering (`Filter`)**: `0` heap allocations requested per item filtering extracting slices.
* **Scale (`Train`)**: Linear `O(n)` scans generating discrete categorical hashes dictionary allocations!
* **AST Validations (`Compile()`)**: Expect negligible allocations preparing execution plans under `1ms` verifying.

## 2. Regression Protections
Pull Requests modifying inside `internal/dataset` or `internal/ast` execute CI validation testing. If allocations shift boundaries negatively performance constraints, the exact assertions `testutil.RequireMaxAllocs(t, 0, fn)` trigger failures avoiding physical performance collapses.
