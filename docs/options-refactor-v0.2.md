# Design Doc — Options-Passing Transform Architecture (v0.2)

> **Status:** Proposal · v0.2 · 2026-05-13
> **Target:** `github.com/TuSKan/ggplot` v0.5
> **Supersedes:** v0.1 of this document.
> **Changes from v0.1:** ValueSpec is now an interface (not a tagged union); ComputedValue dispatches via `dataset.MathKernel` (not a parallel interpreter); Filter and Sort are first-class transforms in the chain (not inline Options fields); the `ValueExpr` string-parser variant is removed.

---

## 1. Motivation

Today, `geom.Histogram(geom.WithBins(40), geom.WithFill("species"))` carries its statistical configuration *inside* the geom. The stat is bound to the visual representation: `stat.Bin` exists only because `geom.Histogram` calls it. This couples concerns that should be orthogonal.

Three concrete consequences:

1. **You cannot compose stats.** `stat.Normalize(stat.Bin(...))` is not expressible because `stat.Bin` returns a configured stat object, not options. A user cannot say "bin first, then normalize the bin counts to sum to 1" without writing custom code.

2. **You cannot retarget a stat to another geom.** Observable Plot's `Plot.binX({y: "count"}, ...)` feeds `rectY` (histogram), `lineY` (frequency polygon), `dot` (1D heatmap), or `areaY` (filled curve). Yours cannot — `geom.Histogram` is the only consumer of `stat.Bin`.

3. **The transform pipeline cannot be inspected, serialized, or golden-tested at intermediate steps.** Without a public spec object between stages, `Built.MarshalJSON()` (roadmap Phase 4.7) can only serialize the final state, not the chain.

This design replaces the geom-owns-stat model with an **options-passing chain**: every transform is a pure function `Options → Options`, every mark is a passive consumer of `Options`, and composition is left-to-right Go-idiomatic.

The change is contained: `aes`, `geom`, `stat`, and `position` touch it. The `dataset`, `scale`, `coord`, `facet`, `theme`, `guide`, and `output` packages do not.

## 2. Goals

- **G1.** Decouple transforms from marks. A `stat.BinX` produces a value that any mark capable of consuming the resulting channels can render.
- **G2.** Inspectable, serializable spec at every stage. `opts.MarshalJSON()` returns the full state after each transform.
- **G3.** Reuse the `dataset.MathKernel` interface for all derived-column math. No parallel interpreter, no string-parsing path.
- **G4.** Preserve the existing user-facing API for one minor release. `geom.Histogram(geom.WithBins(40))` keeps working, implemented as a thin shim.
- **G5.** Custom transforms are trivial to write: one function, three inputs, one return value.
- **G6.** Facet-awareness is a type invariant — transforms cannot ignore facet partitioning.

## 3. Non-goals

- **NG1.** Renaming `geom` to `mark`. ggplot2-aligned vocabulary stays.
- **NG2.** Renaming aesthetics to channels in the public API. `aes.X("col")` stays; internally it constructs a `ChannelSpec`.
- **NG3.** Removing the `Plot` builder. `ggplot.New(ds, ...).Layer(geom.Point()).Save(...)` stays.
- **NG4.** A general DAG of transforms. Chain is linear per layer. Parallel branches live in composite marks.
- **NG5.** Reactive re-rendering. Plot punts on this; we do too. `Build` is recomputed when inputs change.
- **NG6.** A new expression parser. **The `dataset.MathKernel` interface already provides 36 element-wise ops with per-engine dispatch (memory→highway+stdlib; Arrow→compute kernels+highway+stdlib; BigQuery→SQL pushdown). The plot layer reuses it without reinvention.**

## 4. Glossary

| Term | Meaning |
|---|---|
| **Channel** | A named visual encoding slot: `x`, `y`, `fill`, `stroke`, `r`, `fx`, `fy`, `title`, etc. |
| **ChannelSpec** | The per-channel configuration: value source, scale type, label, hint, provenance. |
| **ValueSpec** | **An interface** describing how a channel's data is resolved. Concrete impls: `FieldValue`, `ConstValue`, `AccessorValue`, `ArrayValue`, `DeferredValue`, `ComputedValue`, `SQLValue`. |
| **Channel hint** | A semantic annotation on a derived channel (e.g., "this is a length") that downstream consumers use to format correctly. |
| **Options** | The spec object that flows through the transform chain. Encapsulates data, channel specs, transform chain, initializer chain, style, facet mode, layout hints. |
| **Transform** | A pure function `Options → Options` applied in data space, before scales are constructed. Includes filter, sort, reverse, select, bin, group, stack, normalize, window — all uniformly. |
| **Initializer** | A pure function applied in screen space, after scales are constructed (e.g., dodge, hexbin, pointer). |
| **Reducer** | A named summary function. `count`, `sum`, `mean`, `median`, `p95`, `deviation`, `proportion-facet`, etc. |
| **Mark** | A passive renderer consuming `Options` and producing drawable primitives. |
| **Frame / Table** | The columnar data abstraction (existing `dataset.Table`). |
| **FacetIndex** | A `[]int` of row indexes belonging to one panel. |
| **MathKernel** | The existing `dataset.MathKernel` sub-interface with 36 element-wise ops. The single source of math truth — `ComputedValue` dispatches through it. |

## 5. Package Layout

```
github.com/TuSKan/ggplot/
├── grammar/                  # NEW — public, stable from v0.5
│   ├── options.go            # Options struct + builder methods + clone
│   ├── channel.go            # ChannelName constants + ChannelSpec + ChannelSource
│   ├── value.go              # ValueSpec interface + concrete impls + registry
│   ├── value_helpers.go      # value.Field, value.Const, value.Log10, value.Add, etc.
│   ├── hint.go               # ChannelHint constants + Hints map
│   ├── transform.go          # TransformFunc, TransformChain, run + composition
│   ├── initializer.go        # InitializerFunc, InitializerChain
│   ├── reducer.go            # Reducer interface + registry + built-ins
│   ├── facet.go              # FacetIndex, FacetMode
│   ├── style.go              # Style struct (non-channel visual settings)
│   ├── resolve.go            # ResolveInput, ResolvedChannels, channel materialization
│   └── options_json.go       # Marshal/Unmarshal for Options and ValueSpec
│
├── aes/                      # REFACTORED — same public API, builds ChannelSpec
│   └── aes.go
│
├── stat/                     # REFACTORED — each stat is a TransformFunc factory
│   ├── stat.go               # registry + helpers
│   ├── filter.go             # Filter (NEW: now a transform)
│   ├── sort.go               # Sort, Reverse (NEW: now transforms)
│   ├── select.go             # SelectFirst, SelectLast, SelectMin, SelectMax
│   ├── bin.go                # BinX, BinY, Bin (2D)
│   ├── group.go              # GroupX, GroupY, Group
│   ├── stack.go              # StackX, StackY (also exposed in position/)
│   ├── normalize.go          # NormalizeX, NormalizeY
│   ├── window.go             # WindowX, WindowY (rolling reductions)
│   ├── density.go            # KDE
│   ├── smooth.go             # LOESS, LM
│   ├── summary.go            # Per-group summary reducers
│   └── identity.go           # Identity (passthrough)
│
├── geom/                     # REFACTORED — geoms become Mark + Options consumers
│   ├── point.go
│   ├── line.go               # + WithOrientation
│   ├── bar.go                # + WithOrientation
│   ├── histogram.go          # shim: stat.BinX + geom.RectY
│   ├── rect.go               # NEW — ranged x1/x2/y1/y2 mark
│   ├── ribbon.go             # NEW — y1/y2 area
│   ├── errorbar.go           # NEW
│   ├── cell.go               # NEW — discrete heatmap primitive
│   ├── ...
│   └── geom.go               # Mark interface, shared infrastructure
│
├── position/                 # REFACTORED — Position is a TransformFunc
│   ├── identity.go
│   ├── stack.go              # re-exports stat/stack.go
│   ├── dodge.go              # data-space dodge
│   ├── fill.go
│   └── jitter.go
│
├── init/                     # NEW — screen-space transforms
│   ├── init.go               # InitializerFunc constructors
│   ├── dodge.go              # screen-space dodge (beeswarm)
│   ├── hexbin.go             # pixel-grid hex bucketing
│   └── pointer.go            # interaction filter
│
├── internal/
│   └── spec/                 # RENAMED from internal/grammar
│       ├── plotspec.go       # plot-level wiring (size, title, theme, coord, scales)
│       ├── built.go          # the Built type
│       ├── layout.go
│       └── pipeline.go       # Build / Draw orchestrator
│
└── (dataset/, scale/, coord/, facet/, theme/, guide/, output/, internal/canvas/, internal/color/, internal/fonts/ unchanged)
```

## 6. The Core Types

### 6.1 `ChannelName` and `ChannelSpec`

```go
package grammar

type ChannelName string

const (
    // Position channels — bound to x/y scales.
    ChX  ChannelName = "x"
    ChY  ChannelName = "y"
    ChZ  ChannelName = "z"  // grouping channel; defaults to fill ?? stroke

    // Paired channels — for ranged marks (rect, ribbon, errorbar, difference).
    ChX1 ChannelName = "x1"
    ChX2 ChannelName = "x2"
    ChY1 ChannelName = "y1"
    ChY2 ChannelName = "y2"

    // Visual channels — bound to color/opacity/r/symbol scales.
    ChFill          ChannelName = "fill"
    ChStroke        ChannelName = "stroke"
    ChFillOpacity   ChannelName = "fillOpacity"
    ChStrokeOpacity ChannelName = "strokeOpacity"
    ChOpacity       ChannelName = "opacity"
    ChStrokeWidth   ChannelName = "strokeWidth"
    ChR             ChannelName = "r"          // radius (sqrt scale by default)
    ChLength        ChannelName = "length"
    ChRotate        ChannelName = "rotate"
    ChSymbol        ChannelName = "symbol"

    // Facet channels — bound to band scales.
    ChFX ChannelName = "fx"
    ChFY ChannelName = "fy"

    // Metadata channels — not scaled; consumed by tip/SVG output.
    ChTitle     ChannelName = "title"
    ChHref      ChannelName = "href"
    ChAriaLabel ChannelName = "ariaLabel"

    // Text mark channel.
    ChText ChannelName = "text"
)

// ChannelSpec describes how a single channel is sourced and presented.
type ChannelSpec struct {
    Value  ValueSpec     `json:"value"`           // interface — see §6.2
    Type   scale.Type    `json:"type,omitempty"`  // optional explicit scale type
    Label  string        `json:"label,omitempty"` // axis/legend label
    Hint   ChannelHint   `json:"hint,omitempty"`  // semantic hint for downstream
    Source ChannelSource `json:"source,omitempty"` // provenance
}

// ChannelSource records the lineage of a derived channel.
type ChannelSource struct {
    Origin    string `json:"origin,omitempty"`    // original column name, if any
    Transform string `json:"transform,omitempty"` // last transform that touched it
    Reducer   string `json:"reducer,omitempty"`   // reducer name, if applicable
}
```

### 6.2 `ValueSpec` — Interface

```go
// ValueSpec resolves a channel's value to a column at Build time.
//
// Implementations: FieldValue, ConstValue, AccessorValue, ArrayValue,
// DeferredValue, ComputedValue, SQLValue. Custom implementations may be
// registered via RegisterValueSpec for round-tripping through JSON.
type ValueSpec interface {
    // Kind returns the canonical name for serialization. Must match an
    // entry in the valueSpecRegistry. Built-in kinds:
    //   "field" "const" "accessor" "array" "deferred" "computed" "sql"
    Kind() string

    // Label is the human-readable label for axes and legends. Defaults
    // depend on the implementation (e.g., FieldValue.Label() == Name).
    Label() string

    // Resolve materializes the value as an AnyColumn aligned to the
    // input frame's row indexing. Engine-specific dispatch lives here.
    Resolve(ctx context.Context, in ResolveInput) (dataset.AnyColumn, error)

    // MarshalJSON emits {"kind": "...", ...kind-specific fields...}.
    MarshalJSON() ([]byte, error)
}

// ResolveInput carries everything needed to materialize a ValueSpec.
type ResolveInput struct {
    Engine dataset.Engine    // for sub-interface dispatch (especially MathKernel)
    Table  dataset.Table     // for field lookups
    Facets []FacetIndex      // for facet-aware resolution (used by some transforms)
}

// --- Concrete implementations ---

// FieldValue references a column by name.
type FieldValue struct {
    Name string `json:"field"`
    Lbl  string `json:"label,omitempty"`
}

func (f FieldValue) Kind() string  { return "field" }
func (f FieldValue) Label() string { if f.Lbl != "" { return f.Lbl }; return f.Name }
func (f FieldValue) Resolve(ctx context.Context, in ResolveInput) (dataset.AnyColumn, error) {
    return in.Table.Column(f.Name)
}

// ConstValue resolves to a column of a single repeated value.
type ConstValue struct {
    V   any    `json:"const"`
    Lbl string `json:"label,omitempty"`
}

// AccessorValue evaluates a Go function per row. Not serializable; round-trips
// as {"kind":"accessor","dtype":"...","label":"..."} with nil Fn.
type AccessorValue struct {
    Fn    func(row int, t dataset.Table) any `json:"-"`
    DType dataset.DType                      `json:"dtype"`
    Lbl   string                             `json:"label,omitempty"`
}

// ArrayValue wraps a pre-computed column. Serialized via column metadata,
// not values (values would defeat the spec's role as a contract, not data).
type ArrayValue struct {
    Col dataset.AnyColumn `json:"-"`
    Lbl string            `json:"label,omitempty"`
}

// DeferredValue is a placeholder filled by an upstream transform. Used by
// stats that produce derived channels (e.g., BinX produces x1, x2, y).
type DeferredValue struct {
    Name   string                                                            `json:"deferred"`
    DType  dataset.DType                                                     `json:"dtype"`
    Filler func(ctx context.Context, in ResolveInput) (dataset.AnyColumn, error) `json:"-"`
    Lbl    string                                                            `json:"label,omitempty"`
}

// ComputedValue produces a derived column by composing dataset.MathKernel ops.
// Op names match MathKernel method names exactly. This is the single math
// vocabulary across the package.
type ComputedValue struct {
    Op     string      `json:"op"`     // "addCols", "log10", "mulScalar", ...
    Args   []ValueSpec `json:"args,omitempty"`
    Scalar any         `json:"scalar,omitempty"` // for *Scalar variants
    Lbl    string      `json:"label,omitempty"`
}

func (c ComputedValue) Kind() string  { return "computed" }
func (c ComputedValue) Label() string {
    if c.Lbl != "" { return c.Lbl }
    return defaultLabelFor(c.Op, c.Args, c.Scalar)
}
func (c ComputedValue) Resolve(ctx context.Context, in ResolveInput) (dataset.AnyColumn, error) {
    mk, ok := in.Engine.(dataset.MathKernel)
    if !ok {
        return nil, fmt.Errorf("engine %q does not implement MathKernel (required for computed value %q)", in.Engine.Name(), c.Op)
    }
    // Resolve args recursively.
    cols := make([]dataset.AnyColumn, len(c.Args))
    for i, arg := range c.Args {
        col, err := arg.Resolve(ctx, in)
        if err != nil {
            return nil, fmt.Errorf("computed %q arg %d: %w", c.Op, i, err)
        }
        cols[i] = col
    }
    return dispatchMathKernel(mk, c.Op, cols, c.Scalar)
}

// dispatchMathKernel switches over the MathKernel op vocabulary.
func dispatchMathKernel(mk dataset.MathKernel, op string, cols []dataset.AnyColumn, scalar any) (dataset.AnyColumn, error) {
    switch op {
    // Arithmetic
    case "addCols":   return mk.AddCols(cols[0], cols[1])
    case "subCols":   return mk.SubCols(cols[0], cols[1])
    case "mulCols":   return mk.MulCols(cols[0], cols[1])
    case "divCols":   return mk.DivCols(cols[0], cols[1])
    case "addScalar": s, _ := scalar.(float64); return mk.AddScalar(cols[0], s)
    case "mulScalar": s, _ := scalar.(float64); return mk.MulScalar(cols[0], s)

    // Unary
    case "abs":   return mk.Abs(cols[0])
    case "neg":   return mk.Neg(cols[0])
    case "sign":  return mk.Sign(cols[0])
    case "sqrt":  return mk.Sqrt(cols[0])
    case "pow":   s, _ := scalar.(float64); return mk.Pow(cols[0], s)

    // Log / exp
    case "exp":   return mk.Exp(cols[0])
    case "ln":    return mk.Ln(cols[0])
    case "log2":  return mk.Log2(cols[0])
    case "log10": return mk.Log10(cols[0])

    // Trig
    case "sin":   return mk.Sin(cols[0])
    case "cos":   return mk.Cos(cols[0])
    case "tan":   return mk.Tan(cols[0])
    case "asin":  return mk.Asin(cols[0])
    case "acos":  return mk.Acos(cols[0])
    case "atan":  return mk.Atan(cols[0])
    case "atan2": return mk.Atan2(cols[0], cols[1])

    // Special
    case "tanh":    return mk.Tanh(cols[0])
    case "sigmoid": return mk.Sigmoid(cols[0])
    case "erf":     return mk.Erf(cols[0])

    // Rounding
    case "round": return mk.Round(cols[0])
    case "floor": return mk.Floor(cols[0])
    case "ceil":  return mk.Ceil(cols[0])

    // Bitwise (int64)
    case "bitAnd":        return mk.BitAnd(cols[0], cols[1])
    case "bitOr":         return mk.BitOr(cols[0], cols[1])
    case "bitXor":        return mk.BitXor(cols[0], cols[1])
    case "bitNot":        return mk.BitNot(cols[0])
    case "bitShiftLeft":  n, _ := scalar.(int); return mk.BitShiftLeft(cols[0], n)
    case "bitShiftRight": n, _ := scalar.(int); return mk.BitShiftRight(cols[0], n)

    default:
        return nil, fmt.Errorf("computed value: unknown op %q (must be a MathKernel method name)", op)
    }
}

// SQLValue is a raw SQL escape hatch for BigQuery engine only.
// Use for things MathKernel cannot express (DATE_TRUNC, REGEXP_CONTAINS, etc).
type SQLValue struct {
    SQL string `json:"sql"`
    Lbl string `json:"label,omitempty"`
}

func (s SQLValue) Resolve(ctx context.Context, in ResolveInput) (dataset.AnyColumn, error) {
    bq, ok := in.Engine.(interface {
        ResolveSQL(ctx context.Context, t dataset.Table, sql string) (dataset.AnyColumn, error)
    })
    if !ok {
        return nil, fmt.Errorf("engine %q does not support raw SQL value specs (BigQuery only)", in.Engine.Name())
    }
    return bq.ResolveSQL(ctx, in.Table, s.SQL)
}
```

### 6.3 Value-spec helpers (`grammar/value_helpers.go`)

Users should not type `ComputedValue{Op: "log10", Args: ...}` directly. Helper constructors mirror the MathKernel vocabulary:

```go
// In package grammar (or a value/ sub-package for cleaner imports):

func Field(name string) ValueSpec  { return FieldValue{Name: name, Lbl: name} }
func Const(v any) ValueSpec        { return ConstValue{V: v} }

// Unary
func Log10(a ValueSpec) ValueSpec  { return computed("log10", a) }
func Ln(a ValueSpec) ValueSpec     { return computed("ln", a) }
func Log2(a ValueSpec) ValueSpec   { return computed("log2", a) }
func Sqrt(a ValueSpec) ValueSpec   { return computed("sqrt", a) }
func Abs(a ValueSpec) ValueSpec    { return computed("abs", a) }
func Neg(a ValueSpec) ValueSpec    { return computed("neg", a) }
func Exp(a ValueSpec) ValueSpec    { return computed("exp", a) }
func Sin(a ValueSpec) ValueSpec    { return computed("sin", a) }
func Cos(a ValueSpec) ValueSpec    { return computed("cos", a) }
func Floor(a ValueSpec) ValueSpec  { return computed("floor", a) }
func Round(a ValueSpec) ValueSpec  { return computed("round", a) }
func Ceil(a ValueSpec) ValueSpec   { return computed("ceil", a) }
func Tanh(a ValueSpec) ValueSpec   { return computed("tanh", a) }
func Sigmoid(a ValueSpec) ValueSpec { return computed("sigmoid", a) }

// Binary
func Add(a, b ValueSpec) ValueSpec { return computed("addCols", a, b) }
func Sub(a, b ValueSpec) ValueSpec { return computed("subCols", a, b) }
func Mul(a, b ValueSpec) ValueSpec { return computed("mulCols", a, b) }
func Div(a, b ValueSpec) ValueSpec { return computed("divCols", a, b) }
func Atan2(y, x ValueSpec) ValueSpec { return computed("atan2", y, x) }

// Scalar variants
func AddK(a ValueSpec, k float64) ValueSpec { return computedScalar("addScalar", a, k) }
func MulK(a ValueSpec, k float64) ValueSpec { return computedScalar("mulScalar", a, k) }
func Pow(a ValueSpec, k float64) ValueSpec  { return computedScalar("pow", a, k) }

// SQL escape hatch
func SQL(expr string) ValueSpec    { return SQLValue{SQL: expr} }

func computed(op string, args ...ValueSpec) ValueSpec {
    return ComputedValue{Op: op, Args: args, Lbl: defaultLabelFor(op, args, nil)}
}
func computedScalar(op string, a ValueSpec, k float64) ValueSpec {
    return ComputedValue{Op: op, Args: []ValueSpec{a}, Scalar: k, Lbl: defaultLabelFor(op, []ValueSpec{a}, k)}
}
```

Usage:

```go
opts := grammar.NewOptions(ds).
    With(grammar.ChX, grammar.ChannelSpec{Value: grammar.Field("date")}).
    With(grammar.ChY, grammar.ChannelSpec{Value: grammar.Log10(grammar.Field("price"))})
```

The serialized JSON for that y-channel:

```json
{
  "value": {
    "kind": "computed",
    "op": "log10",
    "args": [{"kind": "field", "field": "price"}],
    "label": "log10(price)"
  },
  "label": "log10(price)"
}
```

### 6.4 ValueSpec registry and JSON round-trip

```go
// grammar/options_json.go

type valueSpecHeader struct {
    Kind string `json:"kind"`
}

var valueSpecRegistry = map[string]func([]byte) (ValueSpec, error){
    "field":    unmarshalField,
    "const":    unmarshalConst,
    "accessor": unmarshalAccessor,  // round-trips with nil Fn; tests that depend on identity must compare structurally
    "array":    unmarshalArray,
    "deferred": unmarshalDeferred,
    "computed": unmarshalComputed,
    "sql":      unmarshalSQL,
}

// RegisterValueSpec lets third parties add their own ValueSpec kinds.
// Returns an error if kind is already registered (prevents accidental override).
func RegisterValueSpec(kind string, unmarshal func([]byte) (ValueSpec, error)) error {
    if _, exists := valueSpecRegistry[kind]; exists {
        return fmt.Errorf("value spec kind %q already registered", kind)
    }
    valueSpecRegistry[kind] = unmarshal
    return nil
}

func UnmarshalValueSpec(data []byte) (ValueSpec, error) {
    var h valueSpecHeader
    if err := json.Unmarshal(data, &h); err != nil { return nil, err }
    if h.Kind == "" { return nil, errors.New("value spec missing 'kind' field") }
    unmarshal, ok := valueSpecRegistry[h.Kind]
    if !ok {
        return nil, fmt.Errorf("unknown value spec kind %q", h.Kind)
    }
    return unmarshal(data)
}

// ChannelSpec.UnmarshalJSON uses UnmarshalValueSpec for the Value field.
func (c *ChannelSpec) UnmarshalJSON(data []byte) error {
    var aux struct {
        Value  json.RawMessage `json:"value"`
        Type   scale.Type      `json:"type,omitempty"`
        Label  string          `json:"label,omitempty"`
        Hint   ChannelHint     `json:"hint,omitempty"`
        Source ChannelSource   `json:"source,omitempty"`
    }
    if err := json.Unmarshal(data, &aux); err != nil { return err }
    val, err := UnmarshalValueSpec(aux.Value)
    if err != nil { return err }
    *c = ChannelSpec{Value: val, Type: aux.Type, Label: aux.Label, Hint: aux.Hint, Source: aux.Source}
    return nil
}
```

### 6.5 `ChannelHint`

```go
type ChannelHint string

const (
    HintNone        ChannelHint = ""
    HintLength      ChannelHint = "length"      // y2-y1 is a length (from stack)
    HintInterval    ChannelHint = "interval"    // x1/x2 are bin endpoints
    HintProbability ChannelHint = "probability" // value in [0,1]
    HintCount       ChannelHint = "count"
    HintProportion  ChannelHint = "proportion"
    HintRank        ChannelHint = "rank"
    HintCumulative  ChannelHint = "cumulative"
)

// Hints can apply across paired channels jointly; stored on Options.Hints
// keyed by primary channel name.
```

### 6.6 `Reducer`

```go
type Reducer interface {
    Name() string
    Reduce(col dataset.AnyColumn) (any, error)
    ReduceFacet(col dataset.AnyColumn, facet FacetIndex) (any, error)
    OutputDType(in dataset.DType) dataset.DType
}

// Built-in reducers — names match Observable Plot's vocabulary.
var (
    ReducerCount             Reducer = &countReducer{}
    ReducerSum               Reducer = &sumReducer{}
    ReducerMean              Reducer = &meanReducer{}
    ReducerMedian            Reducer = &medianReducer{}
    ReducerMin               Reducer = &minReducer{}
    ReducerMax               Reducer = &maxReducer{}
    ReducerMinIndex          Reducer = &minIndexReducer{}
    ReducerMaxIndex          Reducer = &maxIndexReducer{}
    ReducerFirst             Reducer = &firstReducer{}
    ReducerLast              Reducer = &lastReducer{}
    ReducerMode              Reducer = &modeReducer{}
    ReducerDistinct          Reducer = &distinctReducer{}
    ReducerDeviation         Reducer = &welfordStddevReducer{} // Welford
    ReducerVariance          Reducer = &welfordVarianceReducer{}
    ReducerProportion        Reducer = &proportionReducer{}
    ReducerProportionFacet   Reducer = &proportionFacetReducer{}
    ReducerIdentity          Reducer = &identityReducer{}
)

func PercentileReducer(p float64) Reducer { ... }  // satisfies "p95", "p99", etc.

var Reducers = newReducerRegistry()

func (r *ReducerRegistry) Register(name string, red Reducer) error
func (r *ReducerRegistry) Lookup(name string) (Reducer, error)
```

Many reducers in `Aggregator` are already implemented in `dataset/{memory,arrow}/engine.go`. The Reducer registry wraps them when applicable so the plot-layer math stays consistent with the data-layer math.

### 6.7 `Options` — final shape

```go
// Options is the central spec object that flows through the transform chain.
// Value type — every method returns a new Options. JSON-serializable.
//
// Notable absence: no Filter, Sort, or Reverse fields. These are transforms
// in the Transform chain (see §6.8). Inline ergonomic builders WithFilter,
// WithSort, WithReverse delegate to stat.Filter, stat.Sort, stat.Reverse.
type Options struct {
    Data dataset.Table `json:"-"` // serialized via summary { engine, rows, cols }

    Channels map[ChannelName]ChannelSpec `json:"channels,omitempty"`

    // Transform is the composed chain of data-space transforms. Includes
    // filter, sort, reverse, select, bin, group, stack, normalize, window,
    // density, smooth, summary, identity — all uniformly composed.
    Transform TransformChain `json:"transform,omitempty"`

    // Initializer is the composed chain of screen-space initializers.
    Initializer InitializerChain `json:"initializer,omitempty"`

    Hints map[ChannelName]ChannelHint `json:"hints,omitempty"`

    Style Style `json:"style,omitempty"`

    Facet FacetMode `json:"facet,omitempty"`

    Layout LayoutHints `json:"layout,omitempty"`
}

// --- Constructors and builder methods ---

func NewOptions(data dataset.Table) Options {
    return Options{Data: data, Channels: make(map[ChannelName]ChannelSpec)}
}

func (o Options) With(name ChannelName, spec ChannelSpec) Options {
    out := o.clone()
    out.Channels[name] = spec
    return out
}

func (o Options) WithField(name ChannelName, field string) Options {
    return o.With(name, ChannelSpec{Value: FieldValue{Name: field, Lbl: field}, Label: field})
}

func (o Options) WithTransform(t TransformFunc, name string) Options {
    out := o.clone()
    out.Transform = out.Transform.Append(t, name)
    return out
}

func (o Options) WithInitializer(i InitializerFunc, name string) Options {
    out := o.clone()
    out.Initializer = out.Initializer.Append(i, name)
    return out
}

func (o Options) WithHint(name ChannelName, h ChannelHint) Options {
    out := o.clone()
    if out.Hints == nil { out.Hints = make(map[ChannelName]ChannelHint) }
    out.Hints[name] = h
    return out
}

// Inline ergonomic builders for filter/sort/reverse — delegate to transforms.
func (o Options) WithFilter(pred func(row int, t dataset.Table) bool) Options {
    return stat.Filter(o, pred)
}

func (o Options) WithSort(by SortBy) Options { return stat.Sort(o, by) }

func (o Options) WithReverse() Options       { return stat.Reverse(o) }

func (o Options) clone() Options { ... }  // deep-copies Channels, Hints, chains
```

### 6.8 `TransformFunc` and `TransformChain`

```go
// TransformFunc is the contract for a data-space transform.
//
// Invariants:
//   1. MUST NOT mutate in.Data, in.Facets, or in.Options.
//   2. MUST preserve PANEL and group system columns (or reproduce them).
//   3. MUST honor in.Ctx for cancellation.
//   4. MUST be facet-aware: when in.Facets has multiple partitions, output is
//      partitioned consistently.
//   5. MUST set ChannelSpec.Source on every channel it derives.
//   6. MUST set ChannelSpec.Hint where semantics differ from input.
type TransformFunc func(ctx context.Context, in TransformInput) (TransformOutput, error)

type TransformInput struct {
    Data    dataset.Table
    Facets  []FacetIndex
    Options Options
}

type TransformOutput struct {
    Data    dataset.Table
    Facets  []FacetIndex
    Options Options
}

// TransformChain is the composed sequence applied to an Options.
type TransformChain struct {
    funcs []TransformFunc
    names []string  // for JSON / Explain()
}

func (c TransformChain) Append(f TransformFunc, name string) TransformChain { ... }
func (c TransformChain) Names() []string                                    { ... }
func (c TransformChain) Run(ctx context.Context, in TransformInput) (TransformOutput, error) { ... }

// JSON shape — names only; functions aren't serializable.
func (c TransformChain) MarshalJSON() ([]byte, error) {
    return json.Marshal(c.names)
}
```

### 6.9 `InitializerFunc` and `InitializerChain`

```go
// InitializerFunc runs AFTER scales are constructed. Receives resolved
// channels (post-scale pixel/color/etc. values) and may modify them.
type InitializerFunc func(ctx context.Context, in InitializerInput) (InitializerOutput, error)

type InitializerInput struct {
    Data       dataset.Table
    Facets     []FacetIndex
    Options    Options
    Channels   ResolvedChannels
    Scales     ScaleMap
    Dimensions Dimensions
}

type InitializerOutput struct {
    Data     dataset.Table
    Facets   []FacetIndex
    Channels ResolvedChannels
}

type ResolvedChannels map[ChannelName]dataset.AnyColumn

type InitializerChain struct { /* same shape as TransformChain */ }
```

### 6.10 `Mark` interface (geom package)

```go
package geom

import "github.com/TuSKan/ggplot/grammar"

type Mark interface {
    Name() string
    RequiredChannels() []grammar.ChannelName
    OptionalChannels() []grammar.ChannelName
    PairedChannels() [][]grammar.ChannelName
    DefaultTransforms() []grammar.TransformFunc
    RequiredScaleTypes() map[grammar.ChannelName]scale.Type
    Draw(ctx context.Context, in DrawInput) ([]canvas.Primitive, error)
}

type DrawInput struct {
    Channels   grammar.ResolvedChannels
    Facets     []grammar.FacetIndex
    Style      grammar.Style
    Dimensions Dimensions
    Theme      theme.Theme
}

type Layer struct {
    Mark    Mark
    Options grammar.Options
    Data    dataset.Table  // may differ from plot's primary data
}
```

## 7. The Pipeline

```
User code:
    p := ggplot.New(ds, aes.X("date"), aes.Y("close")).
        Layer(geom.LineY()).
        Layer(geom.Point(geom.WithR(2)))
    p.Save("out.svg", 800, 500)

Internally:

  ┌─────────────────────────────────────────────────────────────────────┐
  │ BUILD                                                               │
  │                                                                     │
  │  For each layer:                                                    │
  │   1. Resolve effective Options (inherit plot-level + layer overrides)│
  │   2. Run TransformChain in order:                                   │
  │      in := {Data, Facets, Options}                                  │
  │      for t in chain (filter, sort, bin, normalize, ...):            │
  │        out := t(ctx, in)                                            │
  │        in = out                                                     │
  │   3. Resolve each ChannelSpec.Value to a Column:                    │
  │      - FieldValue:    table.Column(name)                            │
  │      - ConstValue:    repeat to length                              │
  │      - AccessorValue: invoke per-row                                │
  │      - ArrayValue:    direct                                        │
  │      - DeferredValue: invoke Filler                                 │
  │      - ComputedValue: recurse args, dispatch via engine.MathKernel  │
  │      - SQLValue:      engine.ResolveSQL (BQ only; reject elsewhere) │
  │   4. Train position scales (1st pass, with scale transforms)        │
  │   5. Train position scales (2nd pass, post-stack/dodge range)       │
  │   6. Train non-position scales (color, size, alpha, ...)            │
  │   7. Run InitializerChain (screen-space)                            │
  │   → *Built{Layers, Scales, Coord, Theme, Layout, Diagnostics}       │
  └─────────────────────────────────────────────────────────────────────┘

  ┌─────────────────────────────────────────────────────────────────────┐
  │ DRAW (panel-parallel via errgroup)                                  │
  │                                                                     │
  │  For each panel:                                                    │
  │    1. For each layer: layer.Mark.Draw(ctx, DrawInput{...})          │
  │    2. Render axes, grid, strips (consume Theme.Elements)            │
  │  Render legends                                                     │
  │  Render adornment (title, subtitle, caption, margins)               │
  └─────────────────────────────────────────────────────────────────────┘
```

## 8. Three Worked Examples

### 8.1 Filter + Bin + Normalize, Composed

```go
import (
    "github.com/TuSKan/ggplot/grammar"
    "github.com/TuSKan/ggplot/stat"
    "github.com/TuSKan/ggplot/geom"
    "github.com/TuSKan/ggplot/ggplot"
)

opts := grammar.NewOptions(ds).
    With(grammar.ChX, grammar.ChannelSpec{Value: grammar.Field("weight")}).
    With(grammar.ChFill, grammar.ChannelSpec{Value: grammar.Field("sex")})

opts = stat.Filter(opts, func(row int, t dataset.Table) bool {
    // Keep adults only.
    col, _ := dataset.GetColumn[int64](t, "age")
    return col.Values()[row] >= 18
})

opts = stat.BinX(opts, stat.Out{grammar.ChY2: "count"}, stat.WithBins(40))
opts = stat.NormalizeY(opts, "sum")  // proportions sum to 1 within each sex

ggplot.New().Layer(geom.RectY(opts)).Save("normhist.png", 800, 500)
```

The `Options.Transform` chain after these three calls has names: `["filter", "binX", "normalizeY"]`. The JSON of `opts.MarshalJSON()` shows the full state and is golden-testable.

### 8.2 Log-scaled Y from a Computed Value

```go
opts := grammar.NewOptions(ds).
    With(grammar.ChX, grammar.ChannelSpec{Value: grammar.Field("date")}).
    With(grammar.ChY, grammar.ChannelSpec{
        Value: grammar.Log10(grammar.Field("price")),
        Label: "log₁₀(price)",
    })

ggplot.New().Layer(geom.Line(opts)).Save("logprice.png", 800, 500)
```

At Build time, the y-channel's `Resolve` invokes `dataset.MathKernel.Log10` on the price column. On the Arrow engine this hits `compute.Log10`; on the memory engine it hits go-highway; on BigQuery it generates `LOG10(price)` in the pushed-down query. No expression parser, no string interpreting, no duplicated math.

### 8.3 Backward-compat: existing user code unchanged

```go
// v0.4 code, runs unchanged on v0.5:
ggplot.New(ds, aes.X("weight"), aes.Fill("sex")).
    Layer(geom.Histogram(geom.WithBins(40))).
    Save("hist.png", 800, 500)
```

Internally, `geom.Histogram(geom.WithBins(40))` becomes:

```go
func Histogram(opts ...HistogramOption) func(grammar.Options) Layer {
    cfg := histogramDefaults
    for _, o := range opts { o(&cfg) }
    return func(plotOpts grammar.Options) Layer {
        o := stat.BinX(plotOpts, stat.Out{grammar.ChY: "count"}, stat.WithBins(cfg.Bins))
        return geom.RectY(o)
    }
}
```

The user's hands don't move. The internals are now composable.

## 9. Filter and Sort as Transforms — Worked

```go
// stat/filter.go
package stat

import (
    "context"
    "fmt"

    "github.com/TuSKan/ggplot/dataset"
    "github.com/TuSKan/ggplot/grammar"
)

// Filter appends a filter transform to the chain.
//
// The predicate is evaluated against each row of the input frame. Rows for
// which the predicate returns false are excluded from downstream transforms,
// scale training, and rendering. Facet indexes are adjusted accordingly.
func Filter(opts grammar.Options, pred func(row int, t dataset.Table) bool) grammar.Options {
    return opts.WithTransform(func(ctx context.Context, in grammar.TransformInput) (grammar.TransformOutput, error) {
        eng, ok := dataset.GetEngine(in.Data).(dataset.Filterer)
        if !ok {
            return grammar.TransformOutput{}, fmt.Errorf("filter: engine %q does not implement Filterer", in.Data.GetEngine().Name())
        }
        // Build a boolean mask via the predicate, then delegate to engine.Filter.
        mask := make([]bool, in.Data.NumRows())
        for i := range mask {
            if err := ctx.Err(); err != nil {
                return grammar.TransformOutput{}, err
            }
            mask[i] = pred(i, in.Data)
        }
        filtered, err := eng.Filter(in.Data, &boolMasker{mask: mask})
        if err != nil {
            return grammar.TransformOutput{}, fmt.Errorf("filter: %w", err)
        }
        // Rebuild facets relative to the filtered indexing.
        newFacets := adjustFacets(in.Facets, mask)
        return grammar.TransformOutput{
            Data:    filtered,
            Facets:  newFacets,
            Options: in.Options,  // channels unchanged
        }, nil
    }, "filter")
}

// stat/sort.go
type SortBy struct {
    Channel grammar.ChannelName  // sort by a channel's value
    By      grammar.ValueSpec    // ...or by an arbitrary ValueSpec (computed, field, ...)
    Desc    bool
    Reduce  string               // for impute-ordinal-domain semantics: "median", "max", etc.
}

func Sort(opts grammar.Options, by SortBy) grammar.Options { ... }
func Reverse(opts grammar.Options) grammar.Options          { ... }
func Select(opts grammar.Options, mode SelectMode) grammar.Options { ... }  // first, last, min, max
```

Effect on the pipeline: the single `Transform` chain processes everything in order. No "basic transforms" branch, no special phase. `Built.Diagnostics` and `Explain()` report the chain uniformly.

## 10. Engine Awareness — Reused, Not Reinvented

Resolution rules at Build time. The behavior is determined by which sub-interfaces the engine implements:

| ValueSpec      | Required engine sub-interface         | memory | arrow | bigquery |
|----------------|---------------------------------------|--------|-------|----------|
| FieldValue     | `dataset.Table` (always)              | ✓ (slice)    | ✓ (zero-copy) | ✓ (column ref) |
| ConstValue     | none                                  | ✓     | ✓    | ✓ (constant in SELECT) |
| AccessorValue  | none                                  | ✓     | ✓    | ✓ (after stream) |
| ArrayValue     | none                                  | ✓     | ✓    | ✓ |
| DeferredValue  | depends on filler implementation      | varies | varies | varies |
| **ComputedValue** | **`dataset.MathKernel`**           | ✓ (highway+stdlib) | ✓ (Arrow compute → highway → stdlib) | ✓ (SQL pushdown) |
| SQLValue       | engine implements `ResolveSQL`        | rejected | rejected | ✓ |

When an engine lacks a required sub-interface, Build returns a typed error: `*grammar.Error{Phase: PhaseBuild, Reason: ReasonEngineMismatch, Channel: ..., Spec: ...}`. The user sees exactly which channel's value spec failed and why. This matches the `dataset` package's "no fallback" doctrine — failure is explicit, not a slow path.

**Critical reuse:** `ComputedValue` does not parse anything, does not allocate anything beyond arg resolution, and does not maintain a parallel math implementation. It dispatches strings to method calls. The 36 MathKernel methods *are* the expression vocabulary.

## 11. Migration Plan

Three minor releases, honoring the roadmap's 2-minor-deprecation policy.

### v0.5 — Land the new model alongside the old

| Step | Package | Action |
|------|---------|--------|
| 1 | `grammar/` | NEW package. Land types from §6. Public. |
| 2 | `internal/spec/` | RENAME from `internal/grammar`. |
| 3 | `aes/` | Internal: `aes.X("col")` returns a function `func(grammar.Options) grammar.Options` that calls `opts.WithField(grammar.ChX, "col")`. Public API unchanged. |
| 4 | `stat/` | REWRITE each stat as a `TransformFunc` factory. Add `Filter`, `Sort`, `Reverse`, `Select`. Old `stat.Bin`-style symbols → deprecated aliases. |
| 5 | `geom/` | NEW geoms taking `grammar.Options` directly: `geom.RectY`, `geom.Ribbon`, `geom.ErrorBar`, `geom.Cell`, `geom.Difference`. |
| 6 | `geom/` | Existing geoms (`Point`, `Line`, `Bar`, `Histogram`, `BoxPlot`, `Smooth`, `Density`) get internal rewiring + shims. Public API unchanged. |
| 7 | `position/` | Convert `Stack`, `Dodge`, `Fill`, `Identity` to `TransformFunc`. Re-export from `stat/`. |
| 8 | `init/` | NEW package: `init.Dodge`, `init.Hexbin`, `init.Pointer`. |
| 9 | `internal/spec/pipeline.go` | Replace `renderTo()` with Build/Draw split consuming the new model. |
| 10 | `docs/MIGRATION.md` | Side-by-side examples. |

**v0.5 deprecation markers:**

```go
// Deprecated: use stat.BinX returning grammar.Options instead. Old signature
// returns a configured stat object that cannot compose. Removed in v0.7.
// See docs/MIGRATION.md.
func Bin(opts ...BinOption) Stat { ... }
```

### v0.6 — Soft-warn on deprecated paths

| Step | Action |
|------|--------|
| 1 | Deprecated symbols emit a typed `Diagnostic{Code: "GGD001", Level: Warning}` first time invoked per process. |
| 2 | `Diagnostic` surfaces on `Built.Diagnostics`, visible via `Plot.Explain()`. |
| 3 | All `examples/` migrate to new API. |
| 4 | `docs/MIGRATION.md` featured in README. |

### v0.7 — Remove deprecated shims

Old symbols removed. CHANGELOG documents the breakage with one-liner equivalents.

### Compatibility CI

Throughout v0.5–v0.7, a CI matrix runs the v0.4 examples *unchanged* against new internals and snapshot-compares output. Visual drift fails the build. Non-negotiable.

## 12. Test Strategy

Six layers, ordered by cost-to-write.

### 12.1 Options golden tests (new gold standard)

```go
// stat/bin_test.go
func TestBinX_golden(t *testing.T) {
    opts := grammar.NewOptions(testDataset).
        With(grammar.ChX, grammar.ChannelSpec{Value: grammar.Field("weight")}).
        With(grammar.ChFill, grammar.ChannelSpec{Value: grammar.Field("sex")})
    opts = stat.BinX(opts, stat.Out{grammar.ChY: "count"}, stat.WithBins(20))

    goldenAssert(t, "testdata/bin_x.golden.json", opts)
}
```

`goldenAssert` marshals to JSON, compares to golden file, supports `-update`. Golden JSON contains: every ChannelSpec with value source, label, hint, source provenance; the transform chain as named entries; style, facet mode, layout hints.

Platform-independent. Font-independent. Antialiasing-independent. A regression in `stat.BinX` shows up as a precise JSON diff.

### 12.2 ValueSpec round-trip tests

```go
// grammar/value_test.go
func TestValueSpec_RoundTrip(t *testing.T) {
    cases := []grammar.ValueSpec{
        grammar.Field("price"),
        grammar.Const(42.0),
        grammar.Log10(grammar.Field("price")),
        grammar.Add(grammar.Field("a"), grammar.Mul(grammar.Field("b"), grammar.Const(2.0))),
        grammar.SQL("DATE_TRUNC(ts, HOUR)"),
    }
    for _, original := range cases {
        data, err := json.Marshal(original)
        require.NoError(t, err)
        roundtripped, err := grammar.UnmarshalValueSpec(data)
        require.NoError(t, err)
        require.Equal(t, original.Kind(), roundtripped.Kind())
        require.Equal(t, original.Label(), roundtripped.Label())
    }
}

func TestComputedValue_Dispatch_MemoryEngine(t *testing.T) {
    eng := memory.NewEngine()
    ds := /* fixture with column "x" = [1, 4, 16, 64] */
    spec := grammar.Log2(grammar.Field("x"))
    col, err := spec.Resolve(ctx, grammar.ResolveInput{Engine: eng, Table: ds})
    require.NoError(t, err)
    require.Equal(t, []float64{0, 2, 4, 6}, col.(dataset.Column[float64]).Values())
}

func TestComputedValue_Dispatch_ArrowEngine(t *testing.T) { /* same fixture, arrow engine */ }
func TestComputedValue_Dispatch_BQEngine(t *testing.T)     { /* requires BIGQUERY_EMULATOR_HOST */ }
```

### 12.3 Transform invariant tests (property-based)

```go
func TestTransform_NoMutation(t *testing.T) {
    for name, factory := range transformFactories {
        t.Run(name, func(t *testing.T) {
            in := canonicalInput(t)
            inSnapshot := deepClone(in)
            _, _ = factory()(ctx, in)
            require.Equal(t, inSnapshot, in, "transform mutated input")
        })
    }
}

func TestTransform_FacetAware(t *testing.T) { /* ... */ }
func TestTransform_DerivedChannelsHaveSource(t *testing.T) { /* ... */ }
func TestTransform_DerivedChannelsHaveHint(t *testing.T)   { /* ... */ }
```

### 12.4 Reducer numerical tests

```go
func TestWelfordVariance_StreamingPrecision(t *testing.T) {
    // 10M samples from N(1e6, 1). Naive sum-of-squares loses precision;
    // Welford holds to <1e-10 relative error.
}

func TestPercentile_BoundaryCases(t *testing.T) {
    // p0, p100, exactly-on-element, exactly-between-elements.
}
```

### 12.5 Engine-equivalence tests

```go
func TestEngineEquivalence_BinX(t *testing.T) {
    for _, c := range []struct{ name string; eng func() dataset.Engine }{
        {"memory", func() dataset.Engine { return memory.NewEngine() }},
        {"arrow",  func() dataset.Engine { return arrow.NewEngine(memory.DefaultAllocator) }},
        {"bigquery", func() dataset.Engine { return bqEmulator(t) }},
    } {
        t.Run(c.name, func(t *testing.T) {
            opts := /* same Options across engines */
            built := mustBuild(t, opts)
            goldenAssertLayerData(t, "testdata/bin_x_layer.golden.json", built.LayerData(0))
        })
    }
}
```

### 12.6 Fuzz tests + benchmark suite

```go
func FuzzOptionsRoundTrip(f *testing.F)  { /* marshal-unmarshal-marshal idempotency */ }
func FuzzTransformChain(f *testing.F)    { /* random valid transform sequences */ }
func FuzzComputedValue(f *testing.F)     { /* random valid op trees over MathKernel vocabulary */ }
```

```go
func BenchmarkBinX_10k(b *testing.B)        { benchBinX(b, 10_000) }
func BenchmarkBinX_1M(b *testing.B)         { benchBinX(b, 1_000_000) }
func BenchmarkBinX_Arrow_1M(b *testing.B)   { benchBinXArrow(b, 1_000_000) }
func BenchmarkComputedLog10_1M(b *testing.B) { /* via MathKernel path */ }
```

CI golden: `benchmarks/golden.json` tracks ns/op and allocs/op for ~15 canonical chains. PR blocked on >10% regression.

## 13. Versioning and Stability

| Surface | Stable from | Guarantees |
|---------|-------------|------------|
| `grammar/` exported types | v0.5 | Additive only until v1.0. ChannelName constants are wire format. |
| `grammar.ValueSpec` interface | v0.5 | Methods frozen until v1.0. New built-in kinds additive. Third-party kinds registered via `RegisterValueSpec`. |
| `grammar.ComputedValue.Op` vocabulary | v0.5 | Names match `dataset.MathKernel` method names. New ops added in lockstep with MathKernel additions. |
| `grammar.Reducer` registry names | v0.5 | Names match Observable Plot's vocabulary. New reducers additive. |
| `grammar.ChannelHint` constants | v0.5 | Open vocabulary; consumers must tolerate unknown hints. |
| `grammar.TransformFunc` signature | v0.5 | Frozen until v1.0. Adding fields to `TransformInput`/`TransformOutput` is breaking. |
| `geom.*` constructor signatures | v0.5 (existing) | Preserved through v0.7 via shims. |
| `stat.*` constructor signatures | v0.5 (new) | Stable. Old signatures deprecated v0.5, removed v0.7. |
| `init/` package | v0.5 | New package, additive. |
| JSON shape of `Options.MarshalJSON()` | v0.5 | Stable. New fields `omitempty`. Schema version bumped via `"_v": N` for breaking changes. |
| `internal/spec/` | not stable | Internal pipeline can change freely between minors. |

## 14. Open Questions

**Q1.** Should `Plot.marks([...])` equivalent — composite marks as functions returning `[]Layer` — need a dedicated type, or is `[]Layer` enough? Tentative: `[]Layer` with `ggplot.Layers(...)` as a variadic constructor.

**Q2.** Mark-specific extra channels (Plot's `channels: {name: "name", sport: "sport"}` for the tip mark): expressed via `Options.Channels` with a namespace prefix (`"x-name"`, `"x-sport"`)? Tentative: yes, with a documented `x-` prefix reserved for mark-specific extras the tip mark reads.

**Q3.** Channel hints registry: open vocabulary or closed? Tentative: open string, with closed enum at major consumers (axis, tip, legend). Future hints from extensions don't require a package change.

**Q4.** Streaming/incremental rendering for very large datasets — `ValueDeferred` filler that streams from BQ Storage Read API. Tentative: defer to a separate design doc; the spec layer is agnostic about materialization timing.

**Q5.** Should the `ComputedValue.Op` vocabulary be extensible (third-party math ops) or strictly the MathKernel surface? Tentative: strictly MathKernel for now. Extensions land via MathKernel-implementing engines or via custom `ValueSpec` registrations.

**Q6.** `geom.Histogram` and similar shorthand geoms past v0.7 — keep as compositors or move to `compat/`? Tentative: keep in `geom/` permanently; they encode useful defaults (binX→rectY with insets), not just deprecation wrapping.

## 15. v0.5 Work Plan (12 weeks)

| Week | Work |
|------|------|
| 1 | `grammar/` skeleton: types only, no behavior. JSON marshal contract. |
| 2 | `ValueSpec` registry + concrete impls + helpers. `ComputedValue.Resolve` dispatcher over `MathKernel`. Round-trip tests. |
| 3 | `Reducer` registry with all built-ins, Welford for variance. Percentile parser. |
| 4 | `TransformFunc` / `InitializerFunc` infrastructure + chain composition + JSON. |
| 5 | Migrate `stat.Bin` (BinX, BinY, Bin). First golden tests + engine-equivalence tests. |
| 6 | Migrate `stat.Filter`, `stat.Sort`, `stat.Reverse`, `stat.Select` — the unified-chain proof. |
| 7 | Migrate `stat.Group`, `stat.Stack`, `stat.Normalize`, `stat.Window`. |
| 8 | Migrate `stat.Smooth`, `stat.Density`, `stat.Identity`, `stat.Summary`. |
| 9 | Land `geom.RectY`, `geom.Ribbon`, `geom.ErrorBar`, `geom.Cell`. Existing geoms get shims. |
| 10 | Land `init/`: `Dodge`, `Hexbin`, `Pointer`. |
| 11 | Rewrite `internal/spec/pipeline.go`. Build/Draw separation. Compat-CI runs v0.4 examples. |
| 12 | Migration docs, examples migration, benchmarks. Cut v0.5. |

---

## Appendix A — Worked stat: `BinX` under the new model

```go
// stat/bin.go
package stat

import (
    "context"
    "fmt"

    "github.com/TuSKan/ggplot/dataset"
    "github.com/TuSKan/ggplot/grammar"
)

type Out map[grammar.ChannelName]string  // channel → reducer name

type BinOption func(*binConfig)

type binConfig struct {
    Bins       int
    Thresholds []float64
    Cumulative int  // +1, -1, 0
    Filter     bool
    Interval   float64
}

func WithBins(n int) BinOption             { return func(c *binConfig) { c.Bins = n } }
func WithThresholds(t []float64) BinOption { return func(c *binConfig) { c.Thresholds = t } }
func WithCumulative(dir int) BinOption     { return func(c *binConfig) { c.Cumulative = dir } }
func WithBinInterval(w float64) BinOption  { return func(c *binConfig) { c.Interval = w } }

// BinX bins on the X channel; outputs x1, x2 (HintInterval) and one channel
// per (output, reducer) pair in out.
func BinX(opts grammar.Options, out Out, options ...BinOption) grammar.Options {
    cfg := binDefaults
    for _, o := range options { o(&cfg) }

    return opts.WithTransform(func(ctx context.Context, in grammar.TransformInput) (grammar.TransformOutput, error) {
        xSpec, ok := in.Options.Channels[grammar.ChX]
        if !ok {
            return grammar.TransformOutput{}, fmt.Errorf("binX: missing required channel x")
        }

        // Resolve x via whichever ValueSpec it is. This is the unified
        // resolution path — FieldValue, ComputedValue, SQLValue, etc all
        // hit the engine's appropriate sub-interface.
        xCol, err := xSpec.Value.Resolve(ctx, grammar.ResolveInput{
            Engine: in.Data.GetEngine(),
            Table:  in.Data,
            Facets: in.Facets,
        })
        if err != nil {
            return grammar.TransformOutput{}, fmt.Errorf("binX: resolve x: %w", err)
        }

        zCol, _ := resolveZ(ctx, in.Data, in.Options)
        thresholds := computeThresholds(xCol, cfg)

        outOpts := in.Options

        outOpts = outOpts.With(grammar.ChX1, grammar.ChannelSpec{
            Value: grammar.DeferredValue{
                Name:  "binX_x1",
                DType: dataset.DTypeFloat64,
                Filler: func(ctx context.Context, _ grammar.ResolveInput) (dataset.AnyColumn, error) {
                    return materializeBinEdges(thresholds, in.Facets, zCol, edgeLower)
                },
            },
            Type:  xSpec.Type,
            Label: xSpec.Label,
            Hint:  grammar.HintInterval,
            Source: grammar.ChannelSource{Origin: xSpec.Value.Label(), Transform: "binX"},
        })
        outOpts = outOpts.With(grammar.ChX2, grammar.ChannelSpec{
            Value: grammar.DeferredValue{
                Name:  "binX_x2",
                DType: dataset.DTypeFloat64,
                Filler: func(ctx context.Context, _ grammar.ResolveInput) (dataset.AnyColumn, error) {
                    return materializeBinEdges(thresholds, in.Facets, zCol, edgeUpper)
                },
            },
            Type:  xSpec.Type,
            Label: xSpec.Label,
            Hint:  grammar.HintInterval,
            Source: grammar.ChannelSource{Origin: xSpec.Value.Label(), Transform: "binX"},
        })

        for outCh, reducerName := range out {
            red, err := grammar.Reducers.Lookup(reducerName)
            if err != nil {
                return grammar.TransformOutput{}, fmt.Errorf("binX: unknown reducer %q", reducerName)
            }
            outOpts = outOpts.With(outCh, grammar.ChannelSpec{
                Value: grammar.DeferredValue{
                    Name:  fmt.Sprintf("binX_%s_%s", outCh, reducerName),
                    DType: red.OutputDType(xCol.DType()),
                    Filler: func(ctx context.Context, _ grammar.ResolveInput) (dataset.AnyColumn, error) {
                        return reduceBinsX(ctx, xCol, zCol, in.Facets, thresholds, red, cfg)
                    },
                },
                Hint:   hintForReducer(reducerName),
                Source: grammar.ChannelSource{Transform: "binX", Reducer: reducerName},
            })
        }

        return grammar.TransformOutput{
            Data: in.Data, Facets: in.Facets, Options: outOpts,
        }, nil
    }, "binX")
}
```

Key things to notice:

1. The transform is a closure; it doesn't mutate `in.Options`.
2. `xSpec.Value.Resolve(...)` is the unified resolution path. It dispatches on the ValueSpec implementation. If x is a `FieldValue`, it's a column lookup; if x is `grammar.Log10(grammar.Field("price"))`, it goes through MathKernel. Same code path.
3. Heavy work runs inside `DeferredValue.Filler` closures. The data plane chooses materialization time. For the BigQuery engine, fillers can emit SQL fragments rather than streaming rows.
4. Every derived channel has `Source` and `Hint` set.
5. Facet handling is implicit: `in.Facets` is closed over by the fillers.

## Appendix B — Backward-compat shim

```go
// v0.4 user code, unchanged:
p := ggplot.New(ds, aes.X("date"), aes.Y("close")).
    Layer(geom.Line(geom.WithColor("seagreen"))).
    Layer(geom.Smooth(stat.WithMethod("loess"))).
    FacetWrap("season", 2, 0).
    Theme("minimal").
    Save("plot.png", 800, 500)
```

Internals:

1. `aes.X("date")` returns an `aes.Mapping` (existing type) whose effect is `opts.With(grammar.ChX, ChannelSpec{Value: grammar.Field("date")})` when consumed.
2. `geom.Line(...)` is a function `func(opts grammar.Options) Layer` that returns `Layer{Mark: lineMark{}, Options: opts}`.
3. `geom.Smooth(stat.WithMethod("loess"))` appends `stat.Smooth` (now `TransformFunc`) to the chain, then returns `geom.Line{}` as consuming mark (or `geom.Ribbon{}` if SE is requested).
4. `FacetWrap` populates `Plot.Facet` (plot-level), translated to per-layer `fx`/`fy` channels at Build.
5. `Theme("minimal")` and `Save(...)` unaffected.

Users do not learn the new model to keep working. The new model is opt-in for users who need composition (normalize-after-bin, retarget bin to a different geom, math via `grammar.Log10`/`Add`/`Mul`, etc.).

---

**End of design doc v0.2.**

**Next artifact (if approved):** line-level PR plan for `stat/bin.go` + `stat/filter.go` + `grammar/value.go` — the three files that prove out the composition story end-to-end on a single weekend.
