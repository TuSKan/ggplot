# IMPROVEMENTS — ggplot Quality Roadmap

Comprehensive improvement plan based on a 5-part brutal-honest review.
Organized into 9 phases, ordered by priority.

## Design Decisions (Resolved)

| Question | Decision |
|----------|----------|
| Rename `Dataset` → `Frame`? | **No.** Keep `Dataset` name everywhere. |
| Backward compat for old string APIs? | **No.** Remove old methods, no deprecation aliases. |
| Snapshot test format? | **SVG structural comparison** (cross-platform stable). |
| CoordFlip approach? | **Proper pre-scale-training transform**, not render-time swap. |
| BigQuery `_temp` dataset? | **Document as prerequisite.** Write permissions required. |
| Lazy Frame in scope? | **Yes.** No backward compatibility constraints. |
| MathKernel arithmetic? | **Use real SIMD** via `compute.Vec[T]` chunked loops. |
| Color palettes? | **Copy ALL matplotlib palettes** (viridis, magma, inferno, plasma, cividis, twilight, etc.). |
| Stat/Scale/Theme selection? | **Use typed constants**, not strings. |
| MathKernel 34-method interface? | **Replace with `Compute(name, args...)` + registry.** |

---

## Phase 0 — Security: SQL Injection Fix (URGENT, Day 1)

> **CRITICAL**: The BigQuery engine has a textbook SQL injection vulnerability.

### Issue

`dataset/filter.go` `sqlVal()`:
```go
case string:
    return fmt.Sprintf("'%s'", val)   // ← no escaping
```

`Eq("name", "'; DROP TABLE users; --")` produces unescaped SQL. While BigQuery
doesn't allow multi-statement via Storage Read API, it enables:
- **Data exfiltration**: `Eq("name", "x' OR 1=1 --")` returns every row
- **Information leaks**: subquery injection in predicates
- **Tier 2 pendingSQL**: full Job SQL with wider attack surface

### Fix

**Step 1 — Immediate string escaping** in `dataset/filter.go`:
```go
case string:
    escaped := strings.ReplaceAll(val, "'", "''")
    escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
    return fmt.Sprintf("'%s'", escaped)
```

**Step 2 — Parameterized queries** for BigQuery Jobs:
```go
// New type for SQL expressions with parameters.
type SQLExpr struct {
    SQL    string         // "`col` = @p1"
    Params []bigquery.QueryParameter
}

// Masker gains a SQLExpr method.
type Masker interface {
    Mask(ds Table) ([]bool, error)
    SQLExpr() (SQLExpr, error)
}
```

**Step 3 — Column name validation**:
```go
var validIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateColumnName(name string) error {
    if !validIdentifier.MatchString(name) {
        return fmt.Errorf("invalid column name %q: must match [A-Za-z_][A-Za-z0-9_]*", name)
    }
    return nil
}
```

**Step 4 — Remove `fmt.Printf` from library code**:
- `dataset/bigquery/temp.go`: DRY RUN `fmt.Printf` leaks SQL fragments to stdout
- `dataset/bigquery/dataset.go`: `import "log"` → remove or replace with `slog.Logger`
- `dataset/bigquery/engine.go`: same treatment
- All library code: zero `fmt.Print*` or `log.Print*` calls

**Tests**:
```go
func TestSqlVal_Escaping(t *testing.T) {
    tests := []struct{ input, want string }{
        {"O'Brien", "'O''Brien'"},
        {"x'; DROP TABLE--", "'x''; DROP TABLE--'"},
        {"normal", "'normal'"},
    }
    for _, tt := range tests {
        got := sqlVal(tt.input)
        if got != tt.want {
            t.Errorf("sqlVal(%q) = %q, want %q", tt.input, got, tt.want)
        }
    }
}
```

**Deliverables**: Tag `v0.x.1` security patch. Add `SECURITY.md` with disclosure policy.

---

## Phase 1 — Documentation Honesty (Day 1–2)

Strip every false claim. Zero code logic changes — only comments and markdown.

### 1.1 — Package doc `ggplot.go` (lines 1–28)

**Current** (false):
```go
// All data flows through the [dataset.Dataset] abstraction backed by Apache Arrow
// for zero-copy performance, with lazy evaluation for ETL operations.
```

**Fixed**:
```go
// All data flows through the [dataset.Dataset] abstraction. Multiple engine
// backends are supported: memory (Go slices), Apache Arrow (columnar arrays),
// and BigQuery (SQL pushdown). Arrow IPC and Parquet ingest paths provide
// zero-copy reads; constructing from Go slices requires one copy.
```

### 1.2 — Memory engine SIMD comments

**File**: `dataset/memory/engine.go`

Replace every misleading SIMD comment:
```diff
-    // SIMD: AVX-512/AVX2/NEON via go-highway
+    // Scalar reduction via compute.SliceSum.
+    // TODO(phase4): replace with chunked Vec[float64] SIMD loop.
```

Apply to: `Sum` (line 98), `MinMax` (line 132), `Variance` (line 208).

### 1.3 — DATASET.md full sync

**File**: `docs/DATASET.md`

| Doc Says | Code Says | Fix |
|----------|-----------|-----|
| `Dataset` interface: `Schema(); Column(); Len()` | `Table` interface: `Schema(); Column(); NumRows(); NumCols()` | Update |
| `Selector.Take` | `Selector.Select` | Update |
| `Selector.SliceColumn` | `Selector.Slice` | Update |
| `Aggregator.Min(col)` / `Max(col)` | `Aggregator.MinMax(col)` | Update |
| References `factor/`, `strings/`, `datetime/`, `ipc/`, `json/` | Don't exist on disk | Remove |
| "Lazy evaluation" | Frame is eager | Rewrite truthfully |
| "go-highway SIMD primitives" | Scalar fallback for slice-level functions | Add caveat |

**Lazy evaluation section rewrite**:
> Eager evaluation — Dataset verbs execute immediately via the engine and return
> a new Dataset wrapping the result. `Collect()` returns the accumulated Table
> and error. The BigQuery engine has its own internal lazy SQL accumulation via
> `sync.Once`-gated materialization, but this is engine-specific.

### 1.4 — ARCHITECTURE.md, ROADMAP.md

- **ARCHITECTURE.md**: Remove "lazy"/"plan"/"DAG" unless describing Phase 7 future work
- **ROADMAP.md**: Change `✅` to `🔶` for phases with unmet criteria (swallowed errors, shallow clone, no context.Context)

### 1.5 — Font package doc comments

Rewrite LLM-generated word-salad in `internal/fonts/`:

```diff
-// Registry provides a centralized locator index storing mapped physical font
-// struct structs tracking states without cascade logic mappings.
+// Registry indexes system fonts discovered by scanning OS font directories.

-// NewRegistry initializes an OS-level font store generating parsing iterators
-// fetching system standard definitions out.
+// NewRegistry discovers and indexes all fonts in standard OS font directories.

-// Match executes explicit scoring maps comparing geometric elements generating
-// string weights without attempting mapping alias arrays loops locally.
+// Match finds the best font for the given family, weight, and style using
+// a scoring heuristic that prefers exact matches.

-// Resolver acts as the pure logical cascade engine intercepting queries and
-// mapping exact ties vs aliases overrides.
+// Resolver maps font requests to loaded faces via CSS-like cascade:
+// exact match → family alias → weight fallback → system default.

-// LoadFace exposes completely parallel-safe resolutions returning generic
-// mapping handles to external renderers.
+// LoadFace returns a font.Face for the given family, size, weight, and style.
+// Results are cached; concurrent calls share one Face.
```

### 1.6 — README.md claim audit

| Claim | Reality | Action |
|-------|---------|--------|
| "high-performance ML pipelines" | Scalar, single-threaded | Soften or qualify |
| "Apache Arrow for zero-copy" | Only IPC/Parquet | Add caveat |
| "Go's concurrency model" | Zero goroutines | Remove or "planned" |
| "SIMD-accelerated" | Only transcendentals | Qualify precisely |

### 1.7 — New files (partially ✅ DONE)

- **`SECURITY.md`**: Vulnerability disclosure policy (email contact)
- **`CHANGELOG.md`**: ✅ Created — Keep a Changelog format
- **`CONTRIBUTING.md`**: ✅ Created — dev workflow, architecture overview, extension points
- **`.golangci.yml`**: Enable `staticcheck`, `errcheck`, `gocritic`

---

## Phase 2 — Critical Rendering Fixes (Days 3–4)

Fix bugs that produce incorrect output or silently lose data.

### 2.1 — Plot.clone() deep copy ✅ DONE

**File**: `ggplot.go` lines 73–97

**Problem**: `copy(layers, p.spec.Layers)` shallow-copies — `LayerSpec.Mapping`
maps are shared between original and clone. Modifying a derived Plot mutates parent.

**Patch**:
```go
func (p *Plot) clone() *Plot {
    // Deep-clone layers: each LayerSpec.Mapping must be independent.
    layers := make([]grammar.LayerSpec, len(p.spec.Layers))
    for i, l := range p.spec.Layers {
        m := make(grammar.AesMap, len(l.Mapping))
        for k, v := range l.Mapping {
            m[k] = v
        }
        layers[i] = grammar.LayerSpec{Geom: l.Geom, Mapping: m}
    }

    // Deep-clone scale overrides (ScaleOverride.Params is a map).
    scales := make(map[string]grammar.ScaleOverride, len(p.spec.ScaleOverrides))
    for k, v := range p.spec.ScaleOverrides {
        params := make(map[string]string, len(v.Params))
        for pk, pv := range v.Params {
            params[pk] = pv
        }
        scales[k] = grammar.ScaleOverride{Type: v.Type, Params: params}
    }

    return &Plot{
        spec: grammar.PlotSpec{
            Dataset:        p.spec.Dataset,
            GlobalMapping:  p.spec.GlobalMapping.Merge(nil),
            Layers:         layers,
            ScaleOverrides: scales,
            Coord:          p.spec.Coord,
            Facet:          p.spec.Facet,
            ThemeName:      p.spec.ThemeName,
            Labels:         p.spec.Labels,
            XLim:           p.spec.XLim,
            YLim:           p.spec.YLim,
            LegendPosition: p.spec.LegendPosition,
        },
    }
}
```

**Test**: Render `base.Layer(geom.Line())` and `base.Layer(geom.Smooth())`
independently, verify no corruption. Run with `-race`.

### 2.2 — Error surfacing in renderTo (Issues #3, #4)

**File**: `ggplot.go` lines 336–391

**Problem**: `if err == nil && transformed.Table() != nil` silently ignores stat
failures. Users get weird-looking plots with no error.

**Patch** — replace both grouped and ungrouped paths:
```go
// Grouped path (line 344):
transformed, err := s.Compute(grpDS, statMapping)
if err != nil {
    return fmt.Errorf("ggplot: stat %q failed for group %q: %w",
        statName, grpLabel, err)
}
if transformed.Table() == nil {
    return fmt.Errorf("ggplot: stat %q produced nil table for group %q",
        statName, grpLabel)
}
grpDS = transformed
grpMerged = updateMappingForStat(statName, grpMerged)

// Ungrouped path (line 386): same pattern
```

### 2.3 — groupByColumn returns errors + optimization (Issues #4, #18)

**File**: `ggplot.go` lines 919–965

**Changes**:
1. Signature → `([]string, []dataset.Dataset, error)`
2. Return errors instead of `nil, nil`
3. Replace `map[string][]bool` (n-length mask per group) with `map[string][]int`
   (indices only) — eliminates O(n×k) memory

```go
func groupByColumn(ds dataset.Dataset, colName string) ([]string, []dataset.Dataset, error) {
    col, err := ds.Column(colName)
    if err != nil {
        return nil, nil, fmt.Errorf("groupByColumn: column %q: %w", colName, err)
    }

    var vals []string
    switch tc := col.(type) {
    case dataset.Column[string]:
        vals = tc.Values()
    case dataset.Column[float64]:
        vals = make([]string, len(tc.Values()))
        for i, v := range tc.Values() {
            vals[i] = fmt.Sprintf("%g", v)
        }
    case dataset.Column[int64]:
        vals = make([]string, len(tc.Values()))
        for i, v := range tc.Values() {
            vals[i] = fmt.Sprintf("%d", v)
        }
    default:
        return nil, nil, fmt.Errorf("groupByColumn: unsupported type %T", col)
    }

    // Build index lists instead of n-length bool masks.
    groupIdx := make(map[string][]int, len(vals)/4)
    var order []string
    for i, v := range vals {
        if _, exists := groupIdx[v]; !exists {
            order = append(order, v)
        }
        groupIdx[v] = append(groupIdx[v], i)
    }

    subsets := make([]dataset.Dataset, len(order))
    for i, label := range order {
        mask := make([]bool, len(vals))
        for _, idx := range groupIdx[label] {
            mask[idx] = true
        }
        filtered := ds.Filter(dataset.BoolMask(mask))
        if filtered.Err() != nil {
            return nil, nil, fmt.Errorf("groupByColumn: filter %q: %w", label, filtered.Err())
        }
        subsets[i] = filtered
    }
    return order, subsets, nil
}
```

Update call site (line 321) to handle error.

### 2.4 — gridFacet.GridDims uses actual cardinalities ✅ DONE

**File**: `facet/facet.go` lines 137–186

**Problem**: Uses `ceilSqrt(nPanels)` producing square layouts. FacetGrid("season",
"region") with 4×5 produces 5×4 square instead of 4 rows × 5 cols.

**Patch**: Store cardinalities in `Split()`, return them from `GridDims()`:
```go
type gridFacet struct {
    rowCol   string
    colCol   string
    nRowVals int // set by Split
    nColVals int // set by Split
}

func (g *gridFacet) Split(ds dataset.Dataset) ([]Panel, error) {
    rowVals, err := distinctStrings(ds, g.rowCol)
    if err != nil { return nil, err }
    colVals, err := distinctStrings(ds, g.colCol)
    if err != nil { return nil, err }

    g.nRowVals = len(rowVals)
    g.nColVals = len(colVals)
    // ... rest unchanged
}

func (g *gridFacet) GridDims(nPanels int) (int, int) {
    if g.nRowVals > 0 && g.nColVals > 0 {
        return g.nRowVals, g.nColVals
    }
    // Fallback
    if nPanels <= 0 { return 1, 1 }
    cols := ceilSqrt(nPanels)
    return (nPanels + cols - 1) / cols, cols
}
```

### 2.5 — WithSize decoupled from LineWidth (Issue #22)

**File**: `geom/geom.go` lines 228–234

**Problem**: `WithSize(s)` silently sets both `Size` AND `LineWidth`.

**Patch**: Only set `Size`:
```go
// WithSize sets the point radius. Use [WithLineWidth] for stroke width.
func WithSize(s float64) Opt {
    return func(l *Layer) {
        l.Params.Size = s
        l.setFlags |= optSize
    }
}
```

### 2.6 — log.Printf → stored warnings (Issue #23)

**File**: `geom/geom.go` lines 317–326

**Patch**: Store warnings on Layer struct, remove `"log"` import:
```go
type Layer struct {
    // ... existing fields ...
    warnings []string
}

func (l *Layer) Warnings() []string { return l.warnings }

func applyOpts(l *Layer, opts []Opt) {
    for _, o := range opts { o(l) }
    l.warnings = l.Validate() // store, don't log
}
```

### 2.7 — position.Stack panic (Issue #41)

**File**: `position/position.go` lines 56–64

Replace no-op with panic:
```go
func (stack) Adjust(xs, ys []float64, _ float64, groupIdx, _ int) ([]float64, []float64) {
    if groupIdx == 0 { return xs, ys }
    panic("position.Stack: not yet implemented for groupIdx > 0; " +
        "use position.Dodge() or position.Identity()")
}
```

Also: change `Bar()` default position from `Stack()` to `Dodge()`.

### 2.8 — position.Jitter proper PRNG (Issue #42)

**File**: `position/position.go` lines 76–92

Replace broken deterministic hash with `math/rand/v2`:
```go
func (j jitter) Adjust(xs, ys []float64, _ float64, _, _ int) ([]float64, []float64) {
    adjX := make([]float64, len(xs))
    adjY := make([]float64, len(ys))
    rng := rand.New(rand.NewPCG(42, uint64(len(xs))))
    for i := range xs {
        adjX[i] = xs[i] + (rng.Float64()-0.5)*j.xAmt
        adjY[i] = ys[i] + (rng.Float64()-0.5)*j.yAmt
    }
    return adjX, adjY
}
```

### 2.9 — CoordFlip as pre-scale transform (Issue from Part II)

**File**: `ggplot.go` renderTo

**Problem**: Current implementation swaps scales at render time. This breaks
stats computed against wrong axis, discrete-X handling, boxplot whiskers.

**Fix**: Apply CoordFlip as a data transform before scale training:
- After stat transforms, before scale training
- Swap X and Y column mappings in resolved layers
- Swap X and Y labels
- Remove the render-time `renderXScale, renderYScale = yScale, xScale` swap

### 2.10 — New `position/position_test.go`

Test all position adjustments: Identity passthrough, Dodge non-overlapping,
Jitter approximate uniformity, Nudge offset correctness.

---

## Phase 3 — API Hardening (Days 5–7)

Replace stringly-typed APIs with typed constants. Fix silent failures.

### 3.1 — Typed enums everywhere (partially ✅ DONE)

**Theme**: `ggplot.go` `Theme(name string)` → typed constant

```go
// theme/types.go
type ThemeType string

const (
    Minimal  ThemeType = "minimal"
    Dark     ThemeType = "dark"
    Classic  ThemeType = "classic"
    BW       ThemeType = "bw"
)

// ggplot.go
func (p *Plot) Theme(t theme.ThemeType) *Plot {
    cloned := p.clone()
    cloned.spec.ThemeName = string(t)
    return cloned
}
```

**Scale**: `ScaleX(string)` / `ScaleY(string)` → typed constant

```go
// scale/types.go
type Type string

const (
    LinearType  Type = ""
    Log10Type   Type = "log10"
    SqrtType    Type = "sqrt"
    ReverseType Type = "reverse"
)

// ggplot.go
func (p *Plot) ScaleX(t scale.Type) *Plot { ... }
func (p *Plot) ScaleY(t scale.Type) *Plot { ... }
```

**LegendPosition**: `LegendPosition(string)` → typed constant

```go
// ggplot.go
type LegendPos string

const (
    LegendRight  LegendPos = "right"
    LegendLeft   LegendPos = "left"
    LegendTop    LegendPos = "top"
    LegendBottom LegendPos = "bottom"
    LegendNone   LegendPos = "none"
)

func (p *Plot) LegendPosition(pos LegendPos) *Plot { ... }
```

**Stat selection**: `WithStat(string)` → typed constant

```go
// stat/types.go
type Type string

const (
    Identity Type = "identity"
    Bin      Type = "bin"
    Count    Type = "count"
    Density  Type = "density"
    Smooth   Type = "smooth"
    Summary  Type = "summary"
    Boxplot  Type = "boxplot"
)

// geom/geom.go
func WithStat(t stat.Type) Opt { ... }
```

No backward compat aliases — remove old string methods.

### 3.2 — Stat.OutputSchema() ✅ DONE

**Problem**: `updateMappingForStat` in `ggplot.go` is a hardcoded switch on stat
name strings. Adding a new stat requires modifying the core file.

**Fix**: Add `OutputSchema` to `Stat` interface:

```go
// stat/stat.go
type Stat interface {
    Name() string
    RequiredAes() []string
    OutputSchema(inputMapping map[string]string) map[string]string
    Compute(ds dataset.Dataset, mapping map[string]string) (dataset.Dataset, error)
}
```

Implementations:
```go
func (binStat) OutputSchema(m map[string]string) map[string]string {
    return map[string]string{"x": "x", "y": "count"}
}
func (densityStat) OutputSchema(m map[string]string) map[string]string {
    return map[string]string{"x": "x", "y": "density"}
}
func (smoothStat) OutputSchema(m map[string]string) map[string]string {
    return map[string]string{"x": "x", "y": "y"}
}
func (boxplotStat) OutputSchema(m map[string]string) map[string]string {
    return map[string]string{"x": "x", "y": "middle"}
}
```

Then in `ggplot.go` replace the switch:
```go
grpMerged = s.OutputSchema(grpMerged)
```

Delete `updateMappingForStat` entirely.

### 3.3 — stat.Lookup / scale.Resolve return errors ✅ DONE

**Problem**: `stat.Lookup("hisogram")` (typo) silently returns identity stat.
`scale.Resolve("foo")` silently returns Linear scale.

**Fix**:
```go
// stat/stat.go
func Lookup(name string) (Stat, error) {
    if s, ok := registry[name]; ok {
        return s, nil
    }
    return nil, fmt.Errorf("stat: unknown stat %q", name)
}

// scale/scale.go
func Resolve(name string) (Scale, error) {
    switch name {
    case "", "linear": return Linear(), nil
    case "log10":      return Log10(), nil
    case "sqrt":       return Sqrt(), nil
    case "reverse":    return Reverse(), nil
    default:           return nil, fmt.Errorf("scale: unknown scale %q", name)
    }
}
```

Update all call sites in `ggplot.go` to propagate errors.

### 3.4 — __bins typed StatOptions ✅ DONE

**Problem**: `__bins` is `int→string→int` via `fmt.Sprintf`/`Sscanf` round-trip.

**Fix**: Add `StatOptions` struct:
```go
// stat/stat.go
type Options struct {
    Bins     int
    BinWidth float64
    Method   string  // "lm", "loess"
    Span     float64
}

type Stat interface {
    // Compute performs the transformation.
    Compute(ctx context.Context, ds dataset.Dataset, mapping map[string]string, opts Options) (dataset.Dataset, error)
}
```

Remove `__bins` magic key. Pass options from `geom.Params` directly.

### 3.5 — Scale.Train auto-casts int64 (Issue #24)

**Problem**: `Scale.Train` rejects int64 columns with error. Forces manual cast.

**Fix** in `scale/scale.go` `domain.train`:
```go
func (d *domain) train(col dataset.AnyColumn) error {
    switch c := col.(type) {
    case dataset.Column[float64]:
        vals := c.Values()
        // ... existing float64 logic
    case dataset.Column[int64]:
        vals := c.Values()
        mn, mx := float64(math.MaxInt64), float64(math.MinInt64)
        for _, v := range vals {
            fv := float64(v)
            if fv < mn { mn = fv }
            if fv > mx { mx = fv }
        }
        return d.update(mn, mx)
    default:
        return fmt.Errorf("scale: column %q (%s) is not numeric", col.Name(), col.DType())
    }
}
```

### 3.6 — All matplotlib color palettes (Issue #43)

**File**: `internal/color/palette.go`

Replace 5-stop approximations with full 256-entry LUTs from matplotlib (CC0).

**Palettes to embed** (all from matplotlib, CC0 licensed):
- viridis, magma, inferno, plasma, cividis
- twilight, twilight_shifted
- turbo
- coolwarm, RdYlBu, Spectral
- tab10, tab20, Set1, Set2, Set3, Paired

Implementation: `//go:embed` or inline `[256][3]uint8` arrays.

```go
func Viridis(t float64) color.Color {
    t = clamp01(t)
    f := t * 255
    lo := int(f)
    if lo >= 255 { return rgbaAt(viridisLUT, 255) }
    frac := f - float64(lo)
    a := rgbaAt(viridisLUT, lo)
    b := rgbaAt(viridisLUT, lo+1)
    return lerpRGBA(a, b, frac)
}
```

### 3.7 — Extensible geom dispatch (Issue from Part V)

**Problem**: `drawLayer` in `ggplot.go` switches on `geom.Type` string. Third-party
geoms impossible without forking.

**Fix**: Add `Draw` method to `Layer`:
```go
// geom/geom.go
type Drawer interface {
    Draw(cv canvas.Canvas, ds dataset.Dataset, mapping map[string]string,
         params DrawParams) error
}

// Each geom type implements Drawer. Registered via geom.RegisterDrawer.
```

Then `drawLayer` becomes:
```go
if drawer, ok := geom.LookupDrawer(rl.geom.Geom); ok {
    return drawer.Draw(cv, rl.ds, rl.mapping, params)
}
return fmt.Errorf("unknown geom type %q", rl.geom.Geom)
```

---

## Phase 4 — Performance (Week 2)

### 4.1 — Memory MathKernel real SIMD (Issues #7, #34)

**File**: `dataset/memory/math_kernel.go`

Replace `applyBinaryFloat64` closure-per-element with chunked SIMD:

```go
import "github.com/ajroetker/go-highway/hwy"

func (e *Engine) AddCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
    ca, ok := a.(*float64Column)
    if !ok { return nil, fmt.Errorf("requires float64, got %T", a) }
    cb, ok := b.(*float64Column)
    if !ok { return nil, fmt.Errorf("requires float64, got %T", b) }
    if len(ca.data) != len(cb.data) {
        return nil, fmt.Errorf("length mismatch")
    }

    n := len(ca.data)
    out := make([]float64, n)
    lanes := hwy.NumLanes[float64]()
    i := 0
    for ; i+lanes <= n; i += lanes {
        va := hwy.Load(ca.data[i:])
        vb := hwy.Load(cb.data[i:])
        hwy.Store(hwy.Add(va, vb), out[i:])
    }
    for ; i < n; i++ { out[i] = ca.data[i] + cb.data[i] } // scalar tail

    return &float64Column{name: ca.name, data: out}, nil
}
```

Apply same pattern for `SubCols`, `MulCols`, `DivCols`, `AddScalar`, `MulScalar`.

### 4.2 — LOESS sliding window (Issue #25)

**File**: `stat/stat.go` lines 315–363

Replace O(nOut × n log n) with O(n + nOut × k):

```go
// Since pts is sorted by X and xEval advances monotonically,
// maintain a sliding [lo, hi] window of size k.
lo, hi := 0, k
for i := 0; i < nOut; i++ {
    xEval := xMin + float64(i)*step
    xs[i] = xEval

    // Advance window: grow hi rightward, shrink lo rightward
    for hi < n && math.Abs(pts[hi].x-xEval) < math.Abs(pts[lo].x-xEval) {
        lo++
        hi++
    }
    maxDist := math.Max(math.Abs(pts[lo].x-xEval), math.Abs(pts[hi-1].x-xEval))
    if maxDist < 1e-12 { maxDist = 1e-12 }

    // Weighted local regression over pts[lo:hi]
    var sw, swx, swy, swxx, swxy float64
    for j := lo; j < hi; j++ {
        u := math.Abs(pts[j].x-xEval) / maxDist
        w := (1 - u*u*u); w = w * w * w
        dx := pts[j].x - xEval
        sw += w; swx += w*dx; swy += w*pts[j].y
        swxx += w*dx*dx; swxy += w*dx*pts[j].y
    }
    // solve 2x2 WLS...
}
```

Memory: 2 ints instead of 2n floats per output point.

### 4.3 — Arrow double-copy elimination (Issue #46)

**File**: `dataset/arrow/math_kernel.go` `applySliceTransform`

Pre-allocate Arrow buffer directly, write transform into it, skip Go-slice
intermediate. Cuts memory traffic in half.

### 4.4 — Reserve → slices.Grow (Issue #12)

**File**: `dataset/memory/engine.go` lines 434–470

```diff
-func (a *memFloat64Appender) Reserve(n int) {
-    if cap(a.data)-len(a.data) < n {
-        a.data = append(make([]float64, 0, len(a.data)+n), a.data...)
-    }
-}
+func (a *memFloat64Appender) Reserve(n int) {
+    a.data = slices.Grow(a.data, n)
+}
```

Apply to all 4 appender types.

### 4.5 — rowKey → xxh3 hashing (Issue #9)

**File**: `dataset/frame.go` `rowKey`

Replace `fmt.Sprintf` per-cell per-row with `xxh3.Hash` over raw typed bytes.

### 4.6 — Distinct size hints (Issue #10)

**File**: `dataset/frame.go` `Distinct`

```diff
-seen := make(map[string]struct{})
+seen := make(map[string]struct{}, int(f.tbl.NumRows())/2)
```

### 4.7 — KDE parallelization ✅ DONE

**File**: `stat/stat.go` `densityStat.Compute`

Implemented: grid evaluation chunked across `runtime.NumCPU()` goroutines with
precomputed `bwInv` and `norm` constants. Context cancellation checked every 64
points per goroutine. Error propagation via buffered channel.

Also added:
- `stat.Options.Bandwidth` — explicit KDE bandwidth (0 = Silverman auto)
- `stat.Options.BinMethod` — histogram binning: `"sturges"`, `"scott"`, `"fd"`, `"sqrt"`
- `silvermanBandwidth()` — extracted to standalone function
- `autoBins()` — Sturges, Scott, Freedman-Diaconis, Sqrt strategies

---

## Phase 5 — context.Context Plumbing (partially ✅ DONE)

Mechanical but invasive. Touch every interface that does I/O or >1ms compute.

### 5.1 — Engine sub-interfaces

**File**: `dataset/engine.go`

Add `ctx context.Context` as first parameter to:
- `Aggregator`: `Sum(ctx, col)`, `Mean(ctx, col)`, etc.
- `Joiner`: `Join(ctx, left, right, spec)`
- `Reshaper`: all methods
- `Filterer`: `Filter(ctx, ds, mask)`
- `MathKernel`: all methods
- `Selector.SortIndices(ctx, col)`

### 5.2 — All engine implementations

- **Memory**: accept ctx, ignore for now (local compute)
- **Arrow**: pass ctx to compute kernels
- **BigQuery**: pass ctx to BQ API calls (already partially done)

### 5.3 — Stat interface ✅ DONE

Implemented: `Stat.Compute` accepts `context.Context` as first parameter.
All stat implementations check `ctx.Err()` in hot loops.

Check `ctx.Err()` in LOESS inner loop and KDE outer loop.

### 5.4 — Plot API ✅ DONE

Implemented: `Save(ctx, filename, w, h)`, `Render(ctx, w, h)`, `WriteTo(ctx, w, format, w, h)`
all accept `context.Context` as first parameter.

### 5.5 — Plot.WriteTo ✅ DONE

Implemented: `WriteTo(ctx, w, format, width, height)` supports `"png"`, `"svg"`, `"pdf"` formats.
Enables streaming into HTTP responses, S3 uploads, in-memory buffers.

---

## Phase 6 — Testing & CI (Week 3, parallel with Phase 5)

### 6.1 — SVG snapshot tests

**File**: [NEW] `ggplot_snapshot_test.go`

One test per geom type. Render to SVG canvas, compare structural elements.

```go
func TestSnapshot_Point(t *testing.T) {
    eng := memory.NewEngine()
    ds, _ := dataset.NewDataset(eng,
        eng.NewFloat64Column("x", testdata.Seq(100)),
        eng.NewFloat64Column("y", testdata.Rand(100)),
    )
    p := ggplot.New(ds, aes.X("x"), aes.Y("y")).Layer(geom.Point())
    svg, err := p.RenderSVG(400, 300)
    if err != nil { t.Fatal(err) }

    golden := filepath.Join("testdata", "golden", "point.svg")
    if *updateGolden {
        os.WriteFile(golden, svg, 0644)
        return
    }
    expected, _ := os.ReadFile(golden)
    if diff := svgDiff(expected, svg); diff != "" {
        t.Errorf("snapshot mismatch:\n%s", diff)
    }
}
```

Tests needed: Point, Line, Bar, Histogram, Area, Smooth, Boxplot, Density,
Step, Text, Rug, HLine, VLine, FacetWrap, FacetGrid (15 total).

`svgDiff` compares structural elements (paths, circles, rects, text) ignoring
whitespace and floating-point precision beyond 2 decimal places.

### 6.2 — CI improvements

**File**: `.github/workflows/ci.yml`

```yaml
    - name: Run tests with race detector
      run: go test -v -race -coverprofile coverage.out -timeout 180s ./...

    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest]
        go-version: ['1.26']
        goexperiment: ['', 'simd']
    env:
      GOEXPERIMENT: ${{ matrix.goexperiment }}
```

**Additional CI changes**:
- Build ALL examples (not just 6)
- Add `benchstat` comparison against main branch
- Add `.golangci.yml` with `staticcheck`, `errcheck`, `gocritic`
- Add coverage threshold (reject if < 40%)

### 6.3 — Rendering benchmarks ✅ DONE

**File**: `ggplot_bench_test.go` — 11 benchmark scenarios implemented.

### 6.4 — Position and stat tests

**File**: [NEW] `position/position_test.go`
- Identity passthrough (same pointers returned)
- Dodge non-overlapping groups
- Jitter approximate uniform distribution, i=0 not always -0.5
- Nudge exact offset

**File**: [NEW] `stat/stat_test.go` (or expand existing)
- Bin: correct bin centers and counts
- Density: integral ≈ 1.0
- Smooth: output contains nOut points
- Boxplot: correct IQR, whisker fences match Tukey
- Summary: mean of each group

### 6.5 — HiDPI parameter (Issue #54 from Part V)

```go
func (p *Plot) SaveAt(filename string, w, h int, scale float64) error
```

Default `scale=2.0` for retina displays. `Save()` calls `SaveAt(f, w, h, 1.0)`.

---

## Phase 7 — Output Formats: SVG & PDF ✅ DONE

### 7.1 — SVG Canvas ✅ DONE

**File**: `internal/canvas/export_svg.go` — native SVG 1.1 backend implementing `recording.Backend`.
Uses `gg.Path.Iterate()` (compatible with gg v0.43.2). `gogpu/gg-svg` v0.1.0 is incompatible
with current gg API (`path.Elements` removed).

### 7.2 — Plot.Save dispatches on extension ✅ DONE

`Save()` dispatches on file extension (`.png`, `.svg`, `.pdf`).
`WriteTo()` supports `"png"`, `"svg"`, `"pdf"` format strings.
Vector formats use `RecordingCanvas` → `recording.Backend` playback.

### 7.3 — PDF Export Backend ✅ DONE

**File**: `internal/canvas/export_pdf.go` — native PDF 1.4 backend implementing `recording.Backend`.
`gogpu/gg-pdf` v0.1.0 is incompatible with current gg API. Native backend written instead.

**File**: `internal/canvas/recording.go` — `RecordingCanvas` wraps `recording.Recorder`,
captures all draw ops for replay into SVG/PDF backends.

---

## Phase 8 — Lazy Frame Architecture (Weeks 5–6)

No backward compatibility constraints.

### 8.1 — Design

```
Dataset verbs → plan.Op nodes → Plan tree → Collect(ctx) → Optimizer → Execute
```

All Dataset verbs (`Select`, `Filter`, `Arrange`, `Distinct`, `GroupBy.Summarize`,
`Mutate`) become plan nodes instead of eager engine calls.

### 8.2 — Plan package

**File**: [NEW] `dataset/plan/op.go`
```go
type Op interface {
    Kind() string
    Children() []Op
    Schema() *dataset.Schema
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

type ScanOp struct {
    Table dataset.Table
}
```

**File**: [NEW] `dataset/plan/plan.go`
```go
type Plan struct {
    Root Op
}

func (p *Plan) Collect(ctx context.Context, eng dataset.Engine) (dataset.Table, error) {
    // Walk plan bottom-up, execute each Op via engine
    return execute(ctx, eng, p.Root)
}
```

**File**: [NEW] `dataset/plan/optimizer.go`
```go
type Optimizer func(Op) Op

// FuseFilters: consecutive FilterOps → single AND filter
func FuseFilters(op Op) Op { ... }

// PushdownSelect: push Select below Filter when possible
func PushdownSelect(op Op) Op { ... }
```

### 8.3 — Dataset becomes lazy

**File**: `dataset/frame.go`

```go
type Dataset struct {
    eng  dataset.Engine
    plan *plan.Plan  // lazy plan tree
    // cached materialized result
    tbl  dataset.Table
    err  error
    once sync.Once
}

func (f Dataset) Select(cols ...string) Dataset {
    return Dataset{
        eng:  f.eng,
        plan: &plan.Plan{Root: &plan.SelectOp{Input: f.plan.Root, Columns: cols}},
    }
}

func (f Dataset) Collect(ctx context.Context) Dataset {
    f.once.Do(func() {
        optimized := plan.Optimize(f.plan)
        f.tbl, f.err = optimized.Collect(ctx, f.eng)
    })
    return f
}
```

### 8.4 — Engine-specific optimizers

**BigQuery**: Fuse Select+Filter into Storage Read API `ReadSession.DataFormat`
and `RowRestriction`. Fuse complex ops into single SQL Job.

**Arrow**: Fuse element-wise ops into single-pass compute kernels.

**Memory**: Initially no-op. Later: chunked parallel execution.

### 8.5 — MathKernel → Compute registry

Replace the 34-method `MathKernel` interface:
```go
// dataset/engine.go
type ComputeKernel interface {
    Compute(name string, args ...AnyColumn) (AnyColumn, error)
}
```

With a registry:
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

Register all existing operations (AddCols→"add", Sin→"sin", etc.).

---

## Phase 9 — Statistical Method Parameters (After Phase 8)

### 9.1 — Smooth methods

`stat.Smooth` currently supports only LOESS. Add:
- **`lm`**: Simple linear regression (y = a + bx)
- **`glm`**: Generalized linear model
- **`loess`**: Existing implementation (renamed from default)

Configured via `stat.Options.Method`:
```go
case "lm":    return linearFit(pts, nOut, xMin, xMax)
case "loess":  return loessFit(pts, nOut, xMin, xMax, opts.Span)
```

### 9.2 — Histogram binning strategies ✅ DONE

Implemented: `stat.Options.BinMethod` supports `"sturges"` (default), `"scott"`, `"fd"`, `"sqrt"`.
`autoBins()` function implements all four strategies.

### 9.3 — Density bandwidth selection ✅ DONE

Implemented: `stat.Options.Bandwidth` for explicit bandwidth. `0` = Silverman auto-select.
`silvermanBandwidth()` extracted as standalone function.

### 9.4 — Boxplot variants

Add notched boxplots and configurable whisker percentiles:
```go
type BoxplotWhisker string
const (
    WhiskerTukey BoxplotWhisker = "tukey"    // 1.5×IQR (default)
    WhiskerRange BoxplotWhisker = "range"    // min-max
)
```

Add `WithNotch(bool)` option.

---

## Summary

| Phase | Focus | Status |
|-------|-------|--------|
| 0. Security | SQL injection fix, SECURITY.md | 🔴 Open |
| 1. Docs | Strip false claims, sync docs | 🟡 Partial (1.7 CHANGELOG/CONTRIBUTING done) |
| 2. Rendering | clone(), errors, gridFacet, positions | 🟡 Partial (2.1 deep copy, 2.4 gridFacet done) |
| 3. API | Typed enums, OutputSchema, palettes | 🟡 Partial (3.1–3.4 done: typed enums, OutputSchema, error returns, StatOptions) |
| 4. Performance | SIMD MathKernel, LOESS, Arrow, hashing | 🟡 Partial (4.7 KDE parallelization done) |
| 5. Context | context.Context everywhere | 🟡 Partial (5.3–5.5 done: Stat, Plot API, WriteTo) |
| 6. Testing | SVG snapshots, -race, CI, benchmarks | 🟡 Partial (6.3 benchmarks done) |
| 7. Formats | SVG canvas, PDF canvas, WriteTo | ✅ Done |
| 8. Lazy Frame | Plan/execute split, optimizers | 🔴 Open |
| 9. Stats | Smooth methods, binning, bandwidth, boxplot | 🟡 Partial (9.2 binning, 9.3 bandwidth done) |
