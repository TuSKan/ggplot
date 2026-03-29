# Performance Budget & Regression Guardrails

`ggplot` evaluates purely natively utilizing explicit zero-copy architectures mapping boundary data structs dynamically without scaling unexpected bounds. To keep things absolutely performant explicitly evaluating metrics cleanly:

## 1. Hot Paths Budgets
*   **Dataset Filtering (`Filter`)**: `0` heap allocations physically requested per item filtering natively extracting slices.
*   **Scale Bounds (`Train`)**: Linear `O(n)` scans generating discrete categorical hashes bounding dictionary allocations naturally!
*   **AST Validations (`Compile()`)**: Expect structurally completely negligible allocations functionally preparing execution plans completely under `1ms` natively verifying safely.

## 2. Regression Protections
Pull Requests explicitly modifying bounds inside `internal/dataset` or `internal/ast` execute CI validation testing. If allocations shift boundaries negatively mapping performance constraints natively explicitly, the exact assertions mapping `testutil.RequireMaxAllocs(t, 0, fn)` strictly trigger failures avoiding physical performance collapses dynamically.
