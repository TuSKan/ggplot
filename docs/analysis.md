# ggplot — Deep Analysis Against Observable Plot

> **Verified from source.** Read against the full repository (42k lines, 16 packages) extracted from your upload. Every gap below is something I actually grep'd for and didn't find; every strength is something I read in the code, not inferred from the README.
>
> **What this document is not.** Not another "options-refactor v0.3." My prior v0.1/v0.2 design docs proposed building things you already shipped — Build/Render separation, system columns, the Element inheritance system, ColorScale integration, the Observable theme, the colormap+Norm split, the Stat OutputMapping contract. Treat those documents as obsolete.
>
> **Audience.** You, the maintainer. So I'm direct about both strengths and gaps without ceremony.

---

## 1. What this project actually is

Before suggesting improvements, I have to recalibrate my own picture of the project. Reading the code (not just the README's terse capability table), this is a more developed package than I'd been treating it as.

### Architectural choices that are already correct

The roadmap doc states ten guiding principles. Eight of them are visibly implemented:

| Principle | Where it lives in the code |
|---|---|
| Spec / Build / Render as three distinct phases | `Plot` (spec), `Plot.Build(ctx) → *Built`, `Built.Draw(ctx, cv, w, h)` — exact ggplot2-style separation |
| Three-engine typed columnar data | `dataset/memory`, `dataset/arrow`, `dataset/bigquery`, with `MathKernel` and reshape sub-interfaces per engine |
| `PANEL` and `group` as first-class system columns | `spec.go:ColPANEL = "PANEL"`, `ColGroup = "group"` — injected during Build into every layer's data |
| Theme Element system with inheritance | `theme/element.go`: sealed `Element` interface, `ElementText`/`ElementLine`/`ElementRect`/`ElementBlank`; `theme/theme.go:parentOf` map; `resolveText`/`resolveLine`/`resolveRect` walk the chain |
| Public extension contracts | `geom.RegisterGeomType`, `ggplot.RegisterDrawer`, `stat.Register`, `position.New(Name)`, `colormap.Register`, `theme.MustRegister` — every grammar layer has a public registry |
| No global state | `Plot` is immutable, every mutator calls `clone()` first; no package-level mutable theme state |
| Typed phase identifiers | `stat.Name`, `theme.Name`, `scale.Type`, `position.Name`, `geom.Type` — magic strings replaced |
| Per-panel parallelism intent | The `Built.Draw` loop is structured for it, even if errgroup isn't wired yet |

### The colormap package is more sophisticated than Plot's color system

This was my biggest miscalibration earlier in the conversation. Verified from `colormap/`:

- **Cmap/Norm separation.** Plot does not have this. Observable Plot's `scale.type` and `scale.transform` are entangled inside the scale builder. Matplotlib has the separation; ggplot2 doesn't. Your package does, with `LinearSegmentedCmap`/`ListedCmap` for Cmap, and `LinearNorm`/`LogNorm`/`PowerNorm`/`TwoSlopeNorm`/`BoundaryNorm`/`AsinhNorm` for Norm. The `Scale` type composes them.
- **49 registered colormaps** across the matplotlib taxonomy: 6 perceptual, 18 sequential, 11 diverging, 3 cyclic, 11 qualitative (including `okabe_ito` — Plot doesn't ship this).
- **Immutable Cmap transformations.** `Reversed()`, `Resampled(n)`, `WithExtremes(under, over, bad)` return new Cmaps, never mutate. Matches matplotlib's contract.
- **Float-space color throughout.** `colormap.Color = gg.RGBA` type alias enforces no uint8 truncation in the render pipeline. Plot truncates to 8-bit at the SVG serialization point; your package keeps precision until the rasterizer.
- **`Scale.At(v any)` polymorphic dispatch.** A single API consumes `string` / `float64` / `int` / `bool` / `fmt.Stringer` and routes to the right path (label override → discrete index → numeric Norm). Plot has separate paths.
- **Per-engine training.** `Norm.Train(col dataset.AnyColumn)` reuses the dataset abstraction, so a BigQuery-backed dataset trains the same Norm as a memory dataset.

This is genuinely better than what Plot has. If you wrote a comparison blog post, this is the section that would matter.

### The theme catalog is enormous

`theme/` has ~25 files. Themes registered (verified from `init()` calls):

```
Default, Minimal, Dark, BW, Ggplot, Classic, Grayscale,
Bmh, Fivethirtyeight, DarkBackground,
SolarizeLight, SolarizeLight2, SolarizeDark,
TableauColorblind10, Fast, PaulTol,
Seaborn (×12 variants),
Few, FewLight, FewDark, UCBerkeley,
Tableau, Observable, Colorblind,
Autumn1, Autumn2, Canyon, Chili, Tomato,
Petroff10, ObservableDark, Dashboard, Quartz, Air, Ink,
Tufte, Academic, Newsroom, Editorial, Monochrome,
GitHubLight, GitHubDark, Nord, Dracula,
GruvboxLight, GruvboxDark,
AstronomyDark, NASA, Ocean, Earth, Forest, Desert,
HighContrast, OkabeIto, Viridis, Cividis,
Cyberpunk, Blueprint, Terminal, Retro
```

The Observable theme already uses the exact Observable10 palette and Inter font I'd proposed adding. The Dashboard, Quartz, Air, Modern, ObservableDark themes are full Modern-style alternatives to ggplot2's gray-panel default. The matplotlib lineage themes (BMH, FiveThirtyEight, Petroff10, Solarize, Seaborn family) cover the Python ecosystem.

Almost everything I was about to propose adding in "§4.4b — color/theme work" already exists.

### Engine choices that are deliberate

Reading `dataset/DATASET.md` and the three engine implementations:

- **No fallback paths.** When an Arrow engine doesn't implement a sub-interface, you get a typed error, not a silent demotion to a slow path. This is the matplotlib `no-fallback` doctrine and it's the right call for a production system.
- **MathKernel as a sub-interface.** 36 element-wise ops, each with an explicit dispatch per engine. memory → highway+stdlib, Arrow → Arrow compute → highway → stdlib, BigQuery → SQL pushdown.
- **Lazy until Collect.** Datasets are lazy chains; `Collect(ctx)` materializes. This is how the Arrow engine achieves zero-copy and the BigQuery engine achieves pushdown.

These are structural choices most plotting libraries don't make. Plot is a JavaScript library running in a browser — it has no equivalent abstraction. ggplot2 in R has one engine (in-memory data frames). Your three-engine story is genuinely differentiated.

---

## 2. Where Observable Plot is ahead

Plot has been iterated on by a small focused team since 2021 and has absorbed a lot of "you don't realize this matters until you don't have it" details. Cross-referencing your code against Plot's behavior, here's what Plot does better today.

### 2.1 Beauty: typography and numeric formatting

**Tabular figures on quantitative axes.** Verified absence: I `grep`'d the entire repo for `tabular`, `FontVariant`, `font-variant`, `tnum`, `FontFeature` — zero matches. Your `canvas.Canvas` interface has no font-feature method. The result is that quantitative axis labels (like `100`, `1,000`, `10,000`) have visually mis-aligned digit columns when rendered. Plot enables `font-variant-numeric: tabular-nums` on every quantitative axis since 0.2.2 (October 2021). This is the single biggest "looks amateur vs looks publication-ready" difference.

**Default font stack.** Your `baseTheme` uses `Family: "sans-serif"` — a CSS generic that resolves differently on every platform. Plot uses `system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif`. The Observable theme you already ship hardcodes `"Inter, system-ui, sans-serif"` — better, but Inter isn't guaranteed installed. There's no font fallback chain; if Inter isn't present, the actual font is unpredictable. Your `fonts/` package has a resolver and registry — it could fix this, but I don't see it wired through the theme.

**Year formatting.** Plot detects year-like integers (1500–2500 range) and omits the thousands separator, rendering `2025` instead of `2,025`. Your `scale.FormatNumber` always uses the standard format — so a time series with year labels shows commas, which is a giveaway. ~20 lines to fix in `scale/scale.go`.

### 2.2 Beauty: defaults and density

**Inset between adjacent bars.** Verified from `drawer.go`: when bars are drawn, `halfBarPx = (minSpacing * pixPerUnit * relW) / 2`, with `relW = 0.8` default. That gives a 20% gap *for categorical* bars, but for histograms (continuous x with no spacing), `minSpacing` is the bin width and bars touch edge-to-edge. Plot adds a 1px inset between every adjacent rectangle (the bin transform sets `inset = 1` by default) so bars never visually merge even at zoom. Your edge-stroke partially compensates via `Geom.PatchEdgeColor` (the Observable theme sets it to white) but the geometry still touches.

**Default radius for dots.** Your `geom.WithSize` default is 3px when unset. Plot defaults to `r = 3` too — same. No gap here.

**Plot dimensions auto-compute.** Your `Save(ctx, "out.png", 800, 500)` requires both width and height. Plot lets you specify width and infer height from the y-scale type (linear y → ~62% aspect ratio for golden proportions; band y → height proportional to category count × `band(0.1)`). For a Go API this would be `Save(ctx, "out.png", 800, 0)` or a `ggplot.AutoHeight` constant.

**Categorical "padding".** Verified from `drawer.go:trainPanelScales`: discrete scales don't add visible padding between categories; bars span their full slot width minus the relative-width factor. Plot adds `paddingInner` and `paddingOuter` defaults of 0.1 (band scale padding) so categorical plots have breathing room.

### 2.3 Architecture: geoms not yet shipped

`geom/geom.go` declares Types that the `drawer.go` `init()` doesn't register:

```
TypePolygon  — declared, no drawer
TypeTile     — declared, no drawer    (the heatmap primitive)
TypeSegment  — declared, no drawer
TypeErrorBar — declared, no drawer
```

These are placeholder declarations. Plot's mark catalog has `cell` (heatmap), `vector`/`arrow`, `link`, `geo`, `tick`, `tip`, `pointer`, `crosshair`, `tree`. The four declared-but-undrawn geoms above are the minimum visible gap. The bigger architectural gap is interactivity — see §2.5.

### 2.4 Architecture: transform composition

Reading `buildPanel` in `ggplot.go` carefully, the stat pipeline is: one stat per layer, applied once during `Compute`. There's no way to chain stats. `stat.NormalizeAfterBin` doesn't exist because you can't compose `stat.Bin` with another transform without writing a custom Stat.

Plot's distinguishing architectural feature is that transforms are functions `(data, options) → (data, options)` and they compose. `Plot.binX({y: "proportion-facet"}, ...)` is `bin` composed with the `proportion-facet` reducer; `Plot.stackY({y: stat.bin})` retargets the bin output to stacked rects. Your single-stat-per-layer model is closer to ggplot2's `stat_*`/`geom_*` pairing — fine, but it costs you Plot's "retarget any stat to any compatible mark" trick.

This was the topic of my `options-refactor.md`. It's a legitimate architectural delta with Plot. But it's a bigger refactor than I'd realized — it touches the `buildPanel` pipeline, the per-geom Layer struct, and the AesMap propagation. Worth doing but as a v0.6 effort, not v0.5.

### 2.5 Architecture: interactivity

Plot's killer feature versus matplotlib is its interaction model: `tip`, `pointer`, `crosshair`, `frame`, and the `interval` transform that quantizes data into hover-discoverable cells. None of this exists in your package (verified absence — no `interact`, `tip`, `pointer`, `hover` strings anywhere). For PNG output it doesn't matter. For SVG output it would matter — your current SVG export is static.

This is the largest single feature gap, and the one that gives Plot its in-browser identity. The good news is that your output abstraction (`canvas/recording.go` writing through `recording.Backend` to `ExportSVG` / `ExportPDF`) is the right substrate for it. Adding SVG-specific data attributes (`data-x`, `data-y`, `aria-label`) and emitting `<title>` elements per drawn primitive would let SVG consumers add JS interaction. The `tip` mark itself could be a server-rendered fallback: an inline annotation that prints values next to nearest data point.

### 2.6 Architecture: faceting capabilities

Your `facet/` package has `None`, `Wrap(col, opts)`, `Grid(rowCol, colCol)`. Plot also has these, plus `fx`/`fy` as first-class channels — meaning *any* mark can declare its facet, not just the plot-level facet. This enables `Plot.dot(everything, {x, y, fx: yearBucket})` where one layer faces by year bucket and another layer is global (overlay). Your model has plot-level faceting only; `geom.Point(aes.FX("year"))` isn't expressible.

### 2.7 The `channel.hint` and `metadata channel` story

Plot's `title`, `href`, `ariaLabel` channels don't affect rendering — they affect what comes out in the SVG markup. `aes.Title("name")` adds `<title>` tooltips. Plot's `channels` option (specific to the tip mark) lets you carry additional columns through the pipeline without affecting rendering at all. Your `aes.Mapping` has `x`, `y`, `color`, `fill`, `size`, `shape`, `alpha`, `group`, `label`, `weight`, `xmin`/`xmax`/`ymin`/`ymax`. No metadata channels.

### 2.8 Output: SVG fidelity

`canvas/export_svg.go` exports SVG, but I'd want to verify:

- Does it emit `<g class="...">` groups per layer for CSS targeting?
- Does it embed text as `<text>` elements or rasterize to paths?
- Does it support viewBox + responsive `max-width: 100%`?
- Does it suppress crisp-edges raster ops that don't apply to vector backends?

Plot's SVG output is hand-crafted for embedding in web pages: every mark is a `<g>` with a class, text is real `<text>` (so search engines can index it, screen readers can read it), and the root has `width="..." height="..." viewBox="..."` with `style="max-width: 100%; height: auto"`.

I haven't read `canvas/export_svg.go` in full, so this is a question, not a verified gap. Worth auditing.

---

## 3. Where ggplot is ahead of Plot

I went looking for gaps but found capabilities Plot doesn't have. Worth being explicit about these — they're marketing material for the README:

### 3.1 Multi-engine data plane

Plot operates on JavaScript arrays. Your pipeline accepts memory slices, Arrow record batches, *or* BigQuery query results, with appropriate operations pushed to the engine (Arrow compute kernels for math; SQL for BigQuery filtering/aggregation). For a Go data team plotting from BigQuery, this is a structural advantage — Plot would require streaming the entire result set into the browser; yours can keep it server-side and only stream the visual aggregate.

### 3.2 Norm/Cmap separation

As covered in §1. Plot has no equivalent abstraction.

### 3.3 The Theme Element inheritance tree

Plot has no theme system at all. It has one inlined CSS stylesheet plus CSS variables. Your `parentOf` inheritance map with `resolveText`/`resolveLine`/`resolveRect` is a faithful ggplot2 port of the Element system, which is a more powerful abstraction than Plot's CSS-only approach for non-CSS outputs (PNG, PDF).

### 3.4 PDF output

Plot doesn't ship PDF. Your `canvas/export_pdf.go` does.

### 3.5 Theme catalog breadth

~25 named themes covering matplotlib, ggplot2, seaborn, observable, and original designs (Dashboard, Quartz, Air, Ink, Tufte, Academic, etc.). Plot has zero named themes.

### 3.6 Engine-agnostic SIMD math

Your `dataset/memory/math_kernel.go` uses `gogpu/highway` SIMD; Arrow uses Arrow compute → highway → stdlib; BigQuery uses SQL. Plot operates on un-SIMDed JS arrays.

These six items are the differentiation story.

---

## 4. Ranked, concrete suggestions

Ordered by impact-to-effort ratio. Each item is something I read against your code (gap is real, not speculative), with concrete implementation sketches.

### S1. Tabular figures on quantitative axes

**Why it matters most.** This single change crosses the "looks amateur" → "looks publication-ready" line. Every quantitative axis benefits. Zero downside.

**Implementation.** The `gogpu/gg` library uses `golang.org/x/image/font` for text. Modern OpenType fonts expose the `tnum` feature via `sfnt`. The path:

1. Add `FontFeatures []string` to `canvas.Canvas`:
   ```go
   // In canvas/canvas.go
   SetFontFeatures(features []string) // e.g., []string{"tnum"}
   ```
2. Implement in `canvas/gg.go` using `font.Drawer` with `font.Face` opened from an OpenType file that supports `tnum`. The Go standard library's `font` package doesn't expose OT features directly; you'd need either:
   - A vendored OT feature applier (the `tnum` feature shifts the advance of digit glyphs to a uniform width — about 80 lines of code to do manually given a font's `cmap` and `hmtx` tables), or
   - Switch from `gogpu/gg`'s text rendering to a more capable library that respects OT features (probably overkill).
3. Far simpler interim approach: in `scale.FormatNumber`, detect digit-only output and use a custom font face where the digit glyphs are pre-spaced to the digit "0" advance. Bake this into the renderer's tick-drawing path.
4. **Cheapest of all:** ship a font (Go Noto Sans Mono, IBM Plex Mono Digits, or just embed digit glyphs from Inter Tabular) as the *default tick font* in quantitative axes, and use a different font for everything else. The visual difference matters; the implementation is "use a different font here."

**Files touched:** `canvas/canvas.go`, `canvas/gg.go`, `drawer.go` (tick rendering), `fonts/registry.go`.

### S2. Year-aware integer formatting

**Why.** Time-series plots with year labels look unprofessional with thousands separators.

**Implementation.** ~10 lines in `scale/scale.go:FormatNumber`:

```go
// FormatNumber formats a tick value for display.
// Integers in [1500, 2500] are treated as years and shown without separators.
func FormatNumber(v float64) string {
    if v == math.Floor(v) && math.Abs(v) < 1e12 {
        iv := int64(v)
        if iv >= 1500 && iv <= 2500 {
            return strconv.FormatInt(iv, 10) // year — no separator
        }
        return strconv.FormatInt(iv, 10) // currently — also no separator,
                                          // but adding thousands separator below
                                          // would be wrong for years.
    }
    return fmt.Sprintf("%.4g", v)
}
```

Actually, reading your current `FormatNumber`: integers don't get thousand separators at all. So the year case is fine *if you never add separators for integers*. The risk is if a future refactor adds locale-aware number formatting. Add the year detector now as a forward-looking guard.

For non-integer continuous tick values, current `%.4g` is fine. The year case is a guard for when you add localized formatting later.

**Files touched:** `scale/scale.go`. Effort: 10 minutes.

### S3. Auto-compute plot height from width and y-scale type

**Why.** Plot's `width: 640` (height inferred) is the single most cited "feels designed" trait. Forcing the user to pick a height is friction and produces wrong aspect ratios.

**Implementation.** Add to `Built.Save` and `Built.WriteTo`:

```go
// Save renders the built plot to a file. If height ≤ 0, height is inferred
// from width and the y-scale type:
//   - linear y: width / golden_ratio (1.618)
//   - discrete y: 18px per category, clamped to [200, width]
//   - log y: width * 0.65
func (b *Built) Save(ctx context.Context, filename string, width, height int, opts ...RenderOpt) error {
    if height <= 0 {
        height = b.autoHeight(width)
    }
    // ... existing body
}

func (b *Built) autoHeight(width int) int {
    if len(b.panels) == 0 {
        return int(float64(width) / 1.618)
    }
    p := b.panels[0]
    if p.XIsDiscrete && b.layout.Rows == 1 && b.layout.Cols == 1 {
        // Discrete x, continuous y: tall enough to read y values.
        return int(float64(width) * 0.7)
    }
    if _, isDiscrete := p.YScale.(*scale.DiscreteScale); isDiscrete {
        n := len(p.YScale.(*scale.DiscreteScale).Categories())
        h := 18*n + 100  // 18px per category + axis padding
        if h < 240 { h = 240 }
        if h > width { h = width }
        return h
    }
    return int(float64(width) / 1.618)
}
```

**Files touched:** `ggplot.go` (Save/WriteTo). Effort: 1 hour including a few golden tests.

### S4. Inset between adjacent bars in continuous mode

**Why.** Bars currently touch in histograms. Plot's default 1px inset reads as "those are separate bars" even at small sizes.

**Implementation.** In `drawer.go:drawBars` (or wherever the rectangle path is built), inset the rect:

```go
const continuousBarInsetPx = 0.5  // half-pixel each side = 1px total gap

// When drawing a histogram or continuous-x bar:
if p.Orientation == geom.Vertical {
    rx += continuousBarInsetPx
    rw -= 2 * continuousBarInsetPx
} else {
    ry += continuousBarInsetPx
    rh -= 2 * continuousBarInsetPx
}
if rw < 0.5 { rw = 0.5 } // never collapse fully
```

Gate this on whether the geom is `TypeHistogram` *and* there are no group bars (since group bars use the existing `width/nGroups` narrowing). For very narrow bars (e.g., 200 bins in a 600px-wide plot), the inset can be relaxed.

**Files touched:** `drawer.go` (drawBars function). Effort: 30 minutes.

### S5. Implement the four placeholder geoms

**Why.** They're already declared in `geom.Type` with option-relevance masks; the drawer init just doesn't register them. Users hitting `geom.Tile` or `geom.ErrorBar` get silent no-render today.

**Tile (heatmap).** Maps to `cv.DrawRectangle` per row, filled with a color from `ContColScale.At(zValue)`. ~50 lines. Required for any heatmap.

**Segment.** Maps `x`/`y`/`xend`/`yend` aesthetics to four cv.MoveTo+LineTo+Stroke calls. ~30 lines. Required for any "from-to" visualization (arrows, dumbbell plots, linked points).

**ErrorBar.** Maps `x`/`ymin`/`ymax` to a vertical line with horizontal caps. ~40 lines. Required for any plot with uncertainty.

**Polygon.** Closed path through `aes.X`/`aes.Y` points within a `group`. ~25 lines. Required for any custom shape (state outlines, density contours, convex hulls).

**Files touched:** new `geom_*.go` files in the root package or `drawer.go`. Effort: 2-3 hours total. These are the lowest-hanging architectural gaps because the geom-Type registry infrastructure already exists.

### S6. Add metadata channels (`title`, `href`, `aria_label`)

**Why.** These cost almost nothing to add (no rendering changes needed) but unlock SVG hover tooltips, hyperlinks in PDFs (via PDF link annotations), and accessibility.

**Implementation.**

1. Add to `aes/aes.go`:
   ```go
   func Title(col string) Mapping     { return Mapping{Channel: "title", Column: col} }
   func Href(col string) Mapping      { return Mapping{Channel: "href", Column: col} }
   func AriaLabel(col string) Mapping { return Mapping{Channel: "aria_label", Column: col} }
   ```

2. In `drawer.go`, when drawing each primitive (point, bar, line segment), look up these aesthetics from the mapping and emit:
   - For SVG output: `<title>{value}</title>` child element on the primitive, `xlink:href={value}` for href, `aria-label={value}` on `<g>` wrapper.
   - For PNG/PDF: discard or add as PDF annotation (PDF supports link annotations natively).

3. The `canvas.Canvas` interface would need to grow:
   ```go
   // SetMetadata attaches per-primitive metadata to the next drawing call.
   // Backends that don't support metadata (PNG) discard it.
   SetMetadata(meta map[string]string)
   ```

**Files touched:** `aes/aes.go`, `canvas/canvas.go`, `canvas/recording.go` and `canvas/export_svg.go` for SVG emission. Effort: 1 day.

### S7. Stat composition (the bigger refactor)

**Why.** This is the architectural delta I documented at length earlier. It's a real gap, but not urgent — most ggplot2 users never compose stats. Treat as v0.6 work.

**Implementation strategy.** Instead of the big refactor I'd proposed:

1. Add a `stat.Pipeline []stat.Stat` field to `geom.Layer`. Run them in order in `buildPanel`.
2. After the first stat, the dataset has the output schema. The second stat's `RequiredAes()` must match what's available.
3. Validate at Build time; emit a typed error if not.
4. Add a `stat.Compose(s1, s2 ...) stat.Stat` helper for ergonomic composition.

This is far less invasive than the `Options` refactor I'd been pushing. ~3 days work including tests. Punts on the bigger "transforms are pure functions" rewrite because the existing single-stat model is acceptable for ggplot2 compatibility; composition is the only must-have addition.

**Files touched:** `stat/stat.go`, `geom/geom.go`, `ggplot.go` (buildPanel). Effort: ~3 days.

### S8. SVG output audit and improvements

**Why.** SVG is your interactive-output path. If it's bake-once static markup, you have no story for embedded web visualization.

**Audit checklist** (I haven't read `canvas/export_svg.go` carefully, so this is a question list, not findings):

- Does the root SVG have `viewBox` + `preserveAspectRatio="xMidYMid meet"`?
- Is there a `style="max-width: 100%; height: auto"` for responsive embedding?
- Are layers wrapped in `<g class="ggplot-layer ggplot-layer-{n} ggplot-{geom-type}">`?
- Is text emitted as `<text>` (selectable, searchable) or paths (rasterized)?
- Are colors emitted as inline `style="fill: #abc"` or as `fill="#abc"` attributes? (CSS-style allows external override.)
- Is there a single `<defs>` block with reusable patterns/gradients?

If any of these is "no," fixing it is a high-impact change for the SVG export specifically. The colormap.Color = gg.RGBA float-space guarantee means your color fidelity is excellent in SVG — but only if you serialize as `rgb(r,g,b)` or hex, not as the truncated values that PNG would store.

**Files touched:** `canvas/export_svg.go`, `canvas/recording.go`. Effort: 1-3 days depending on what audit finds.

### S9. Channel hints and richer continuous color stories

**Why.** Plot's `Plot.binX({y: "proportion-facet"}, ...)` produces a histogram showing per-facet proportions. The reducer name (`"proportion-facet"`) tells downstream consumers it's a proportion, so the axis formats as `0%` to `100%`. This is "channel hints" — small annotations that propagate semantic meaning.

For ggplot, this would mean:

```go
// In stat.Stat, extend the contract:
type Stat interface {
    // existing methods...
    OutputHints() map[string]ChannelHint
}

type ChannelHint string
const (
    HintCount       ChannelHint = "count"
    HintProportion  ChannelHint = "proportion"
    HintProbability ChannelHint = "probability"
    HintCumulative  ChannelHint = "cumulative"
    HintInterval    ChannelHint = "interval"
)
```

Then `scale.Scale` checks the hint via the Build pipeline and formats accordingly. Proportions get `0%` / `25%` / `50%` ticks; counts get integer ticks; cumulative gets a clamped `[0, max]` range.

**Effort:** 1 day for the contract + 1 day per consumer (axis formatting, color bar formatting). 2-3 days total.

### S10. Stat reducer vocabulary

**Why.** Plot's reducer vocabulary — `count`, `sum`, `mean`, `median`, `min`, `max`, `proportion`, `proportion-facet`, `mode`, `deviation`, `variance`, `first`, `last`, `p10`, `p50`, `p90` — is used everywhere. Your stats are single-purpose: `Bin`, `Count`, `Density`, `Smooth`, `Summary`, `Boxplot`. There's no `mean` stat or `quantile` stat.

A separate `reducer/` subpackage with these named reducers would:

1. Let `stat.Summary` accept named reducers as options.
2. Provide the substrate for the channel-hint story above (each reducer declares its output hint).
3. Mirror what your `dataset` package's `Aggregator` interface likely already does — so this is mostly a rebinding of existing aggregation kernels to a public registry.

**Effort:** 2 days if `dataset` aggregation kernels exist (I think they do — I saw references in memory/engine.go); 4-5 days if starting from scratch.

---

## 5. Where I was wrong, and why

This is for my own accountability and so you can calibrate trust in the rest of this document.

| What I claimed earlier | Actual reality |
|---|---|
| "No documented color scheme catalog." | `colormap/` ships 49 schemes across 5 matplotlib categories, with Cmap/Norm separation more advanced than Plot's. |
| "Add a Theme Element system." | Already exists at `theme/element.go` with full inheritance, sealed interface, merge helpers, and parent-of map. |
| "Add an Observable theme." | Already exists at `theme/observable.go` with the exact Observable10 palette and Inter font. |
| "Add Build/Render separation." | Already exists: `Plot.Build(ctx) → *Built`, `Built.Draw(ctx, cv, w, h)`. |
| "Add PANEL and group as first-class system columns." | Already exists: `spec.go:ColPANEL`, `ColGroup`. |
| "Add a `ColorScale` field to channels." | Already exists: `PlotSpec.ColorScales map[string]*colormap.Scale`. |
| "Add `stat.OutputMapping()` for stat→geom channel rewiring." | Already exists in `stat/stat.go`. |
| "Add Modern, Dashboard, Quartz themes." | All exist in `theme/modern.go`. |

The pattern is consistent: I was reading the README's terse capability table (`Themes: Default, Classic, Minimal, Dark, BW`) and inferring absence from omission. The README understates what ships. The roadmap doc (which I should have read sooner) explicitly states each of these principles as already-implemented.

The honest takeaway is that **your project is past the point where high-level "add these features" suggestions are useful.** The features are mostly there. What's left are:

- Polish details (tabular nums, insets, year formatting) — S1–S4
- The four declared-but-undrawn geoms — S5
- Metadata channels for SVG interactivity — S6
- Stat composition — S7 (optional)
- SVG output audit — S8
- Channel hints + reducer vocabulary — S9, S10 (optional)

Almost everything else I was about to propose, you've already built.

---

## 6. What to prioritize

If I had to pick three things to ship for v0.5 visual polish, I'd pick:

1. **S1 (tabular nums)** — biggest beauty payoff, ~1 day for the minimal fix.
2. **S4 (continuous-bar inset)** — looks-amateur fix, 30 minutes.
3. **S5 (the four placeholder geoms)** — fills a real capability gap, ~3 hours.

That's 1.5-2 days of work for the most visible upgrades.

For v0.6 if you want to address an architectural gap, **S7 (stat composition)** is the one that matters most. Stat-mark composition is Plot's signature trick and the one capability ggplot2 doesn't have natively either. Adding it would give you a story Plot users can recognize.

For v0.7 or a separate sub-project, **S8 (SVG audit)** and **S6 (metadata channels)** together set up an interactive-SVG story, which is the only path to feature-parity with Plot for in-browser visualization.

Beyond that, the package is in good shape. The README and the marketing positioning are the things that undersell it — Observable Plot has a much smaller feature surface but a much better story for "what you get when you install it." A README rewrite that surfaces the colormap.Norm system, the three-engine data plane, the 25+ themes, and the multi-format output story would do more for adoption than another year of feature work.

---

## 7. Appendix: things I didn't verify

For accountability, these are things I'd flagged or considered but didn't actually read the code for:

- **`canvas/export_svg.go` in full.** S8 is structured as questions because of this.
- **`canvas/export_pdf.go`.** I confirmed it exists but didn't verify what features (link annotations, etc.) it supports.
- **`fonts/resolver.go` integration with themes.** The theme uses `Family: "Inter, system-ui, sans-serif"` but I don't know whether the resolver falls through that chain or just uses the first.
- **`dataset/bigquery` engine.** I read enough of `engine.go` to confirm structure, but didn't trace a complete query lifecycle.
- **`ggplot_test.go` and `ggplot_golden_test.go`.** Reading the tests would tell me what's covered by golden assertions and what isn't. I prioritized reading production code.
- **The `examples/` directory.** I saw the directory listing but didn't read individual examples to verify user-facing API ergonomics.
- **Bench results.** I saw `ggplot_bench_test.go` exists but didn't read `docs/BENCHMARK.md`.

If you want me to dig into any of these for a follow-up, point me at the file. Otherwise this is the analysis.
