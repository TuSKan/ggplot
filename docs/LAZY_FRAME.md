# Lazy Frame Architecture

> Spec for converting the eager `dataset.Dataset` into a deferred plan/execute model.  
> This is a breaking change to the `Dataset` type.

---

## Motivation

Today every `Dataset` verb (Select, Filter, Arrange, Distinct, GroupBy.Summarize,
Mutate) executes immediately via the engine. This:

1. **Prevents cross-verb optimisation** — consecutive Filter+Filter can't be fused.
2. **Forces materialisation** — each verb allocates a new Table, even if the
   downstream consumer only needs a subset of columns.
3. **Blocks SQL pushdown** — BigQuery engine can't fuse a Select+Filter+Arrange
   chain into one SQL query because each verb fires independently.

## Current State

```go
// dataset/frame.go
type Dataset struct {
    tbl Table
    err error
}
```

Every verb calls `requireEngine()` → type-asserts the sub-interface → executes
immediately → wraps result in a new `Dataset{tbl: result}`.

## Target State

```
Dataset verbs → plan.Op nodes → Plan tree → Collect(ctx) → Optimizer → Execute
```

All verbs become plan nodes. Materialisation only happens on explicit `Collect(ctx)`,
or when a consuming API (ggplot rendering, stat computation) forces it.

---

## Design

### 1. Plan Package

**[NEW]** `dataset/plan/op.go`

```go
package plan

import "github.com/TuSKan/ggplot/dataset"

// Op is a single node in a logical query plan.
type Op interface {
    Kind() string
    Children() []Op
    Schema() *dataset.Schema
}

// Leaf: scans an already-materialised Table.
type ScanOp struct {
    Table dataset.Table
}

type SelectOp struct {
    Input   Op
    Columns []string
}

type FilterOp struct {
    Input Op
    Mask  dataset.Masker
}

type ArrangeOp struct {
    Input   Op
    Columns []string
    Desc    []bool
}

type DistinctOp struct {
    Input   Op
    Columns []string
}

type MutateOp struct {
    Input Op
    Name  string
    Expr  Expr // column expression (future: AST)
}

type LimitOp struct {
    Input  Op
    Offset int
    N      int
}

type GroupByOp struct {
    Input    Op
    GroupCols []string
    AggExprs  []AggExpr // (name, func, source_col)
}
```

**[NEW]** `dataset/plan/plan.go`

```go
type Plan struct {
    Root Op
}

func (p *Plan) Collect(ctx context.Context, eng dataset.Engine) (dataset.Table, error) {
    optimized := Optimize(p)
    return execute(ctx, eng, optimized.Root)
}

func Optimize(p *Plan) *Plan {
    root := p.Root
    for _, opt := range defaultOptimizers {
        root = opt(root)
    }
    return &Plan{Root: root}
}
```

**[NEW]** `dataset/plan/optimizer.go`

```go
type Optimizer func(Op) Op

var defaultOptimizers = []Optimizer{
    FuseFilters,
    PushdownSelect,
    EliminateIdentityOps,
}

// FuseFilters: consecutive FilterOps → single AND filter.
func FuseFilters(op Op) Op { ... }

// PushdownSelect: push Select below Filter when columns allow.
func PushdownSelect(op Op) Op { ... }

// EliminateIdentityOps: remove Select("*") or Limit(0, MaxInt).
func EliminateIdentityOps(op Op) Op { ... }
```

### 2. Dataset Becomes Lazy

**[MODIFY]** `dataset/frame.go`

```go
type Dataset struct {
    eng  dataset.Engine
    plan *plan.Plan   // lazy plan tree (nil if already materialised)

    // cached materialised result
    tbl  dataset.Table
    err  error
    once sync.Once
}

// Eager path: wraps an already-materialised Table.
func From(ds Table) Dataset {
    return Dataset{
        eng:  GetEngine(ds),
        tbl:  ds,
    }
}

// Lazy path: every verb appends a plan node.
func (f Dataset) Select(cols ...string) Dataset {
    return Dataset{
        eng:  f.eng,
        plan: &plan.Plan{Root: &plan.SelectOp{Input: f.root(), Columns: cols}},
    }
}

func (f Dataset) Filter(mask Masker) Dataset {
    return Dataset{
        eng:  f.eng,
        plan: &plan.Plan{Root: &plan.FilterOp{Input: f.root(), Mask: mask}},
    }
}

// root returns the plan root, wrapping tbl in ScanOp if needed.
func (f Dataset) root() plan.Op {
    if f.plan != nil {
        return f.plan.Root
    }
    return &plan.ScanOp{Table: f.tbl}
}

// Collect forces materialisation.
func (f Dataset) Collect(ctx context.Context) Dataset {
    f.once.Do(func() {
        if f.plan == nil {
            return // already materialised
        }
        f.tbl, f.err = f.plan.Collect(ctx, f.eng)
        f.plan = nil
    })
    return f
}

// Table returns the materialised table, panicking if not yet collected.
func (f Dataset) Table() Table {
    if f.plan != nil {
        panic("dataset: Table() called on uncollected lazy Dataset — call Collect(ctx) first")
    }
    return f.tbl
}
```

### 3. Engine-Specific Optimisers

Each engine can register optimisers that run after the default passes:

**BigQuery**: Fuse Select+Filter+Arrange into a single SQL query with
`ReadSession.RowRestriction`. Fuse GroupBy+Summarize into `SELECT agg() ... GROUP BY`.

**Arrow**: Fuse element-wise MutateOps into single-pass compute kernels.

**Memory**: Initially no-op. Later: chunked parallel execution over goroutine pools.

```go
// dataset/plan/engine_optimizer.go
type EngineOptimizer interface {
    Optimize(Op) Op
}

// Called by Plan.Collect before execute:
if eo, ok := eng.(EngineOptimizer); ok {
    root = eo.Optimize(root)
}
```

### 4. MathKernel → Compute Registry

Replace the 34-method `MathKernel` interface with a registry:

```go
// dataset/compute/registry.go
type KernelFunc func(args ...AnyColumn) (AnyColumn, error)

var kernels = map[string]KernelFunc{}

func Register(name string, fn KernelFunc) { kernels[name] = fn }

func Compute(name string, args ...AnyColumn) (AnyColumn, error) {
    fn, ok := kernels[name]
    if !ok { return nil, fmt.Errorf("unknown kernel %q", name) }
    return fn(args...)
}
```

Register all existing operations:
- `AddCols` → `"add"`, `SubCols` → `"sub"`, `MulCols` → `"mul"`, `DivCols` → `"div"`
- `Sin` → `"sin"`, `Cos` → `"cos"`, ..., `Erf` → `"erf"`
- `BitAnd` → `"bit_and"`, etc.

Memory engine registers stdlib math functions. Arrow engine registers compute
kernel wrappers. BigQuery engine registers SQL function generators.

### 5. Context Propagation (5.1 / 5.2)

With the lazy model, context flows through `Collect(ctx)` into `execute()` which
passes it down to every engine call. This eliminates the need to add `ctx` to
every sub-interface method — the engine already has `Context()` for lifecycle,
and the plan executor handles per-operation cancellation.

---

## Migration Strategy

### Phase A — Plan Package (no breaking changes)
1. Create `dataset/plan/` with Op types, Plan, and optimisers
2. Add `plan.Execute(ctx, eng, op)` that walks the tree bottom-up
3. Unit-test plan construction and optimisers independently

### Phase B — Dual-Mode Dataset
1. Add `plan *plan.Plan` field to Dataset
2. Keep all existing eager code paths — if `plan == nil`, behave as today
3. Add `Collect(ctx)` method
4. Existing callers still work (they use `Table()` which returns the eager result)

### Phase C — Lazy Verbs
1. Change verbs to build plan nodes instead of calling engine
2. Add `Collect(ctx)` calls at all materialisation points:
   - `ggplot.renderTo()` — before stat transforms
   - `stat.Compute()` — before accessing column data
   - `Dataset.Table()` — explicit materialisation
3. Remove eager code paths

### Phase D — Engine Optimisers
1. Implement BigQuery SQL fusion
2. Implement Arrow kernel fusion
3. Benchmark against eager baseline

---

## Impact on ggplot

The rendering pipeline (`ggplot.renderTo`) will need to call `Collect(ctx)` on
every Dataset before passing it to stat transforms and `drawLayer`. This is a
single-point change since the pipeline already has `context.Context`:

```go
// Before stat transform:
ds = ds.Collect(ctx)
if ds.Err() != nil {
    return fmt.Errorf("ggplot: collect dataset: %w", ds.Err())
}
```

Stat transforms (`stat.Compute`) will also call `Collect(ctx)` on their input
before accessing column data.

---

## Estimated Timeline

| Phase | Duration | Scope |
|-------|----------|-------|
| A. Plan package | 3 days | New package, no breaking changes |
| B. Dual-mode Dataset | 3 days | Add plan field, Collect method |
| C. Lazy verbs | 4 days | Convert all verbs, update callers |
| D. Engine optimisers | 4 days | BigQuery SQL fusion, Arrow fusion |
| E. Benchmarks + cleanup | 2 days | Verify perf, remove dead code |

**Total: ~2.5 weeks**

**Tag `v1.0.0`** after Phase D stabilises.
