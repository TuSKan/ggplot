# CoordFlip — Pre-Scale Data Transform Refactor

## Problem

CoordFlip currently works via **render-time swaps** scattered across 5 locations:

| File | Line | What it does |
|------|------|-------------|
| `ggplot.go` L851 | Swaps which scale drives the left-axis tick measurement |
| `ggplot.go` L1000 | Swaps `renderXScale`/`renderYScale` and labels for grid + axes |
| `draw.go` L351 | Bar: switches `barAxisLen` from `w` to `h` |
| `draw.go` L378 | Bar: swaps rectangle construction (horizontal vs vertical) |
| `coord/coord.go` L38 | `flippedCoord.Transform` swaps `x↔y` in pixel mapping |

### Why this is fragile

1. **Stats run on un-flipped data** — `stat_smooth` regresses Y on X, but with `CoordFlip` the user intends X on Y. The smooth curve ends up wrong for flipped axes.
2. **Every new geometry must add `IsFlipped()` branches** — bar has 2, histogram would need the same, boxplot would need it too.
3. **Scale training happens on the wrong axis** — X data trains the X scale, but after flip it's displayed as Y. Limits, breaks, and formatting are applied to the wrong axis.

### Current flow (broken for stats)

```
Data(x,y) → Stat(x,y) → ScaleTrain(x→xScale, y→yScale) → Render(swap at paint time)
                ↑                                                    ↑
         stat sees original x,y                           coord.Transform swaps x↔y
         smooth regresses y~x                             5 scattered IsFlipped branches
```

## Target Architecture

Swap the aesthetic mapping **once, early**, before stat transforms:

```
Data(x,y) → MappingSwap(x↔y) → Stat(x,y) → ScaleTrain(x→xScale, y→yScale) → Render(no branches)
                  ↑                                                                   ↑
           if CoordFlip:                                                   coord.Transform is
           aes x↔y, labels x↔y                                            always cartesian
           done ONCE here                                                  (no special cases)
```

### Key insight

`CoordFlip` is not a coordinate system — it's a **mapping transform**. The coordinate system stays Cartesian; we just swap which data column maps to which axis.

## Proposed Changes

### Phase 1: Mapping swap in `renderTo` (before stats)

#### [MODIFY] [ggplot.go](file:///d:/Developments/ggplot/ggplot.go)

At the top of `renderTo`, before the facet/stat loop:

```go
// If flipped, swap the X/Y aesthetic mapping and labels ONCE.
// This ensures stats (smooth, density) operate on the correct axis.
if p.spec.Coord.IsFlipped() {
    // Swap global mapping
    p.spec.GlobalMapping["x"], p.spec.GlobalMapping["y"] =
        p.spec.GlobalMapping["y"], p.spec.GlobalMapping["x"]
    // Swap labels
    p.spec.Labels.X, p.spec.Labels.Y = p.spec.Labels.Y, p.spec.Labels.X
    // Replace coord with cartesian — the swap is already done
    p.spec.Coord = coord.Cartesian()
}
```

> [!WARNING]
> `renderTo` operates on a **cloned** `p.spec` (via `p.clone()`). Verify that `clone()` is called before `renderTo` so the original Plot is not mutated. If `renderTo` mutates `p.spec` directly, the swap must happen on a local copy.

#### Remove all render-time IsFlipped branches

| Location | Action |
|----------|--------|
| `ggplot.go` L851 | Remove — left-axis always uses yScale after swap |
| `ggplot.go` L1000 | Remove — renderXScale/renderYScale always match after swap |
| `draw.go` L351 | Remove — barAxisLen always uses w |
| `draw.go` L378 | Remove — rectangle is always vertical orientation |

---

### Phase 2: Simplify `coord.Coord` interface

#### [MODIFY] [coord/coord.go](file:///d:/Developments/ggplot/coord/coord.go)

```go
// IsFlipped is no longer needed on Coord after the mapping swap.
// Keep it for backward compat but deprecate:

// Deprecated: IsFlipped is handled by mapping swap before stats.
// This method always returns false after the refactor.
IsFlipped() bool
```

Or remove `IsFlipped()` entirely and keep `flippedCoord` only as a marker that triggers the mapping swap.

#### Option: Remove `flippedCoord` entirely

`coord.Flip()` would return `cartesianCoord{}` after swapping is done at the Plot level. The Coord interface reduces to just `Transform` + `String`.

---

### Phase 3: Verify stat correctness

The main correctness win is `stat_smooth` with CoordFlip:

```go
// Before refactor: smooth regresses y~x (wrong — user wants x~y)
// After refactor:  mappings are swapped, so stat sees y as the new x
//                  smooth now regresses on the correct axis
```

Add a test:

```go
func TestCoordFlip_Smooth(t *testing.T) {
    // Data: x=[1,2,3,4,5], y=[2,4,6,8,10]
    // With CoordFlip, the smooth should regress the original X on original Y
    // (because mappings are swapped before stat)
    ds := ... // create dataset
    p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
        Layer(geom.Smooth()).
        CoordFlip()
    _, err := p.Render(ctx, 400, 300)
    if err != nil {
        t.Fatalf("CoordFlip+Smooth failed: %v", err)
    }
}
```

## Risk Analysis

| Risk | Mitigation |
|------|-----------|
| `renderTo` mutates `p.spec` | Verify clone() is called first; if not, swap on a local copy |
| Per-layer mappings override global | Swap per-layer mappings too (iterate `p.spec.Layers[i].Mapping`) |
| Scale limits `XLim`/`YLim` need swap | Swap `p.spec.XLim ↔ p.spec.YLim` in the same block |
| Histogram bins (stat_bin) — binning on wrong axis | After swap, stat_bin sees correct "x" column |
| Boxplot stat — groups on wrong axis | After swap, boxplot groups correctly |
| CoordFlip + Facet interaction | Facet split happens before swap — should be fine since facet uses column names |

## Verification Plan

1. `go test ./...` — all existing CoordFlip tests must pass
2. New test: `TestCoordFlip_Smooth` — stat operates on correct axis
3. New test: `TestCoordFlip_Histogram` — bins on correct axis
4. Visual: run `examples/showcase` and `examples/phase2_features` to confirm horizontal bars render correctly
5. `go vet ./...` — no issues
6. Grep for `IsFlipped` — should have zero hits in `ggplot.go` and `draw.go` after cleanup

## Files Changed

| File | Change |
|------|--------|
| `ggplot.go` | **[MODIFY]** Add mapping swap at top of `renderTo`; remove 2 `IsFlipped` branches |
| `draw.go` | **[MODIFY]** Remove 2 `IsFlipped` branches from bar drawing |
| `coord/coord.go` | **[MODIFY]** Deprecate or remove `IsFlipped()` from interface |
| `ggplot_test.go` | **[MODIFY]** Add `TestCoordFlip_Smooth`, `TestCoordFlip_Histogram` |
