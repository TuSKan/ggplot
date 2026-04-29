# Port matplotlib + pyplot-themes presets into `theme/` and default to ggplot

## Context

The library currently ships 5 themes (`Default`, `Classic`, `Minimal`, `Dark`, `BW`) defined in a single [theme/theme.go](theme/theme.go) file as hand-tuned ad-hoc styles. The user wants a comprehensive preset catalog drawn from two well-known references:

- matplotlib's stylelib (https://matplotlib.org/stable/gallery/style_sheets/style_sheets_reference.html) — the reference set Python users know
- raybuhr's pyplot-themes (https://github.com/raybuhr/pyplot-themes) — additional curated palettes (Paul Tol, Few, UC Berkeley, Tableau)

…and the new default is to be matplotlib's `ggplot` style (gray panel `#E5E5E5`, white grid, gray text `#555555`, ggplot color cycle starting with `#E24A33`), since that is what most plotting users immediately recognize as "the ggplot look." Today's default is a generic light theme that doesn't visually match any well-known preset.

Two side effects fall out of porting these styles faithfully:

1. **Themes need to carry a discrete color cycle.** Almost every matplotlib style sets `axes.prop_cycle`. Without porting the cycle, multi-series plots under `Ggplot`, `Fivethirtyeight`, `Bmh`, etc. would all look identical (Tab10 over a tinted background) — the themes would feel broken. So `Theme` gets a `Palette` field, and the two existing default-color-scale fallback sites in [ggplot.go](ggplot.go) consult it before falling back to `colormap.Tab10`.
2. **The existing `Classic` constant is wrong.** Today it's an alias for `Default`. Matplotlib's `classic` is a real, distinct style (white background, black axes, no grid, mpl1.x defaults). Per CLAUDE.md ("No backward-compatibility aliases when renaming methods or replacing string APIs"), `Classic` is repurposed to the matplotlib definition with no shim.

## Scope decisions (already confirmed with user)

- **Both sources, deduplicated.** Where matplotlib and pyplot-themes overlap (`ggplot`/`theme_ggplot2`, `fivethirtyeight`, `bmh`, `solarized_light2`, `dark_background`), prefer matplotlib's exact rcParams values from its `.mplstyle` files.
- **Default = matplotlib `ggplot` style.** `Default` constant becomes an alias that resolves to `Ggplot`.
- **Themes drive the discrete color cycle** via a new `Theme.Palette` field, wired into the two existing default-color-scale fallback points.
- **One file per theme** under `theme/`, mirroring matplotlib's stylelib layout. Keep types and `Resolve()` in [theme/theme.go](theme/theme.go).

## Files modified

- [theme/theme.go](theme/theme.go) — add `Palette []color.Color` field to `Theme`; add ~30 new `Name` constants; rewrite `Resolve()` switch.
- [theme/ggplot.go](theme/ggplot.go) (new) — and one file per preset listed below.
- [ggplot.go](ggplot.go) — at lines 567-571 and 677-680, consult resolved theme's `Palette` before falling back to `colormap.Tab10` / `colormap.Viridis`. Requires resolving the theme once earlier (today it's resolved at line 501 inside `renderTo`; the color-scale resolution at line 567 is in a different function — pass `th` through, or resolve at the spec-clone boundary so it's available at both sites).
- [examples/themes/main.go](examples/themes/main.go) — extend the `themes := []theme.Name{...}` slice to render all new presets so the visual diff is reviewable; rename outputs to `theme_<name>.png`.

## Theme catalog (30 total)

### Matplotlib base styles (9)

| Constant | mplstyle source | Distinguishing features |
|---|---|---|
| `Ggplot` | `ggplot.mplstyle` | bg `#E5E5E5`, white grid+axes, text `#555555`, palette `[E24A33, 348ABD, 988ED5, 777777, FBC15E, 8EBA42, FFB5B8]` |
| `Classic` | `classic.mplstyle` | white bg, black axes, no grid, mpl1.x palette `[blue, green, red, cyan, magenta, yellow, black]` (**REPLACES current alias-of-Default**) |
| `Grayscale` | `grayscale.mplstyle` | white bg, black axes, palette `[0.00, 0.40, 0.60, 0.70]` grayscale ramp, fig bg `0.75` |
| `Bmh` | `bmh.mplstyle` | bg `#EEEEEE`, dashed grid, palette `[348ABD, A60628, 7A68A6, 467821, D55E00, CC79A7, 56B4E9, 009E73, F0E442, 0072B2]` |
| `Fivethirtyeight` | `fivethirtyeight.mplstyle` | bg `#F0F0F0`, grid `#CBCBCB`, no ticks, line width 4, palette `[008FD5, FC4F30, E5AE38, 6D904F, 8B8B8B, 810F7C]` |
| `DarkBackground` | `dark_background.mplstyle` | black bg, white axes/text/ticks, palette `[8DD3C7, FEFFB3, BFBBD9, FA8174, 81B1D2, FDB462, B3DE69, BC82BD, CCEBC4, FFED6F]` |
| `SolarizeLight2` | `Solarize_Light2.mplstyle` | fig bg `#FDF6E3`, panel `#EEE8D5`, palette `[268BD2, 2AA198, 859900, CB4B16, D33682, 6C71C4, 657B83, 93A1A1]` |
| `TableauColorblind10` | `tableau-colorblind10.mplstyle` | white bg, palette `[006BA4, FF800E, ABABAB, 595959, 5F9ED1, C85200, 898989, A2C8EC, FFBC79, CFCFCF]` |
| `Fast` | `fast.mplstyle` | inherits Ggplot visuals; chiefly a perf hint upstream — in our port it's an alias for `Default` (we have no path-simplification analog) and exists so users can `Theme("fast")` without error |

### Seaborn family (16) — sourced from `seaborn-v0_8*.mplstyle`

Base + grid variants: `Seaborn`, `SeabornDarkgrid`, `SeabornWhitegrid`, `SeabornDark`, `SeabornWhite`, `SeabornTicks`. Base palette `[4C72B0, 55A868, C44E52, 8172B2, CCB974, 64B5CD]`, panel `#EAEAF2`, white grid+axes.

Palette variants (same chrome as base, different cycle): `SeabornDeep`, `SeabornMuted`, `SeabornBright`, `SeabornColorblind`, `SeabornPastel`, `SeabornDarkPalette`.

Font-size variants (same chrome, larger fonts): `SeabornPaper` (8/9pt), `SeabornNotebook` (10/12pt), `SeabornTalk` (13/15pt), `SeabornPoster` (16/19pt). Implement as one-line factories that take the base seaborn theme and override `Text.*.Size`.

### pyplot-themes additions (4 — non-overlapping with matplotlib)

| Constant | Source | Palette |
|---|---|---|
| `PaulTol` | `pyplot_themes/palettes.py` Paul Tol scheme | `[332288, 6699CC, 88CCEE, 44AA99, 117733, 999933, DDCC77, 661100, CC6677, AA4466, 882255, AA4499]` |
| `Few` | `pyplot_themes/palettes.py` Few medium | `[4D4D4D, 5DA5DA, FAA43A, 60BD68, F17CB0, B2912F, B276B2, DECF3F, F15854]`, no grid by default |
| `UCBerkeley` | `pyplot_themes/palettes.py` Berkeley | white bg, axes `#EEEEEE`, palette starts with Berkeley Blue / Founder's Rock / California Gold |
| `Tableau` | pyplot-themes `theme_tableau` | matplotlib's classic Tableau10 palette `[1F77B4, FF7F0E, 2CA02C, D62728, 9467BD, 8C564B, E377C2, 7F7F7F, BCBD22, 17BECF]` |

### Library originals (kept; visuals unchanged)

`Default` (now alias → `Ggplot`), `Minimal`, `Dark`, `BW`. Note: `Default` keeps its constant value `"default"` so existing user code keeps compiling, but `Resolve("default")` now returns the ggplot preset, not the old hand-tuned light theme. `Classic` is the only existing constant whose visual definition changes (former alias-of-default → matplotlib classic), per CLAUDE.md's no-shim rule.

## Implementation outline

1. **Extend the type** in [theme/theme.go](theme/theme.go): add `Palette []color.Color` to `Theme` struct (after `Spacing`). No other field changes.
2. **Add helpers** in [theme/theme.go](theme/theme.go): a `hex(s string) color.Color` helper that parses `"E24A33"` / `"#E24A33"` into `color.RGBA{A:255}` so per-theme files read like the source mplstyle. (Existing `gray()` helper stays.)
3. **One file per theme** — e.g. [theme/ggplot.go](theme/ggplot.go) defines `func newGgplot() Theme { ... }` with values lifted directly from the corresponding `.mplstyle`. Each file is ~30 lines. Group seaborn variants into `theme/seaborn.go` since they share a base.
4. **Rewrite `Resolve()`** to dispatch all new names. Keep the empty-string fallthrough pointing at `newGgplot()` so plots without an explicit `.Theme(...)` call get the new default.
5. **Update name constants** in [theme/theme.go](theme/theme.go): keep `Default`, `Classic`, `Minimal`, `Dark`, `BW`; add the 25 new constants (`Ggplot`, `Grayscale`, `Bmh`, `Fivethirtyeight`, `DarkBackground`, `SolarizeLight2`, `TableauColorblind10`, `Fast`, `Seaborn`, `SeabornDarkgrid`, `SeabornWhitegrid`, `SeabornDark`, `SeabornWhite`, `SeabornTicks`, `SeabornDeep`, `SeabornMuted`, `SeabornBright`, `SeabornColorblind`, `SeabornPastel`, `SeabornDarkPalette`, `SeabornPaper`, `SeabornNotebook`, `SeabornTalk`, `SeabornPoster`, `PaulTol`, `Few`, `UCBerkeley`, `Tableau`).
6. **Wire `Theme.Palette` into the color-scale defaults** in [ggplot.go](ggplot.go):
   - At line 567-571 (discrete fallback): `if colorScale == nil { if len(th.Palette) > 0 { colorScale = colormap.NewDiscrete(colormap.NewListed(th.Name, colormap.Qualitative, paletteToData(th.Palette))) } else { colorScale = colormap.NewDiscrete(colormap.Tab10) } }`. The `paletteToData` adapter converts `[]color.Color` to whatever shape `NewListed` takes (see [colormap/builtin.go:117](colormap/builtin.go#L117) for `tab10Data`'s shape). Continuous fallback at line 677-680 stays Viridis since matplotlib styles don't set a continuous default we can honor uniformly.
   - The theme `th` is currently resolved at [ggplot.go:501](ggplot.go#L501) inside `renderTo`. Lines 567 and 677 are in a different function (looks like `resolveLayers` or similar). Resolve once at the top of `renderTo` and pass `th` (or just `th.Palette`) down to that helper. If the helper signature would balloon, attach it to a small per-render context struct rather than threading individual params.
7. **Update [examples/themes/main.go](examples/themes/main.go)** to iterate over the full new constant list. Use a multi-series dataset so palette differences are visible. Confirm rendered PNGs visually match each source style.

## Reuse / things not to reinvent

- `colormap.NewListed(name, Qualitative, data)` already exists ([colormap/builtin.go:117](colormap/builtin.go#L117)) — use it to convert a theme palette into a discrete `Cmap` rather than introducing a new colormap type.
- `gray(uint8) color.Color` already in [theme/theme.go:211](theme/theme.go#L211).
- `theme.Resolve()` ([theme/theme.go:194](theme/theme.go#L194)) is already plumbed through [ggplot.go:501](ggplot.go#L501); no new entry point needed.
- `Plot.Theme(name)` builder ([ggplot.go:255](ggplot.go#L255)) is already typed against `theme.Name`; new constants slot in without API change.

## Verification

- `gofmt -s -l .` produces no output (CI gate).
- `go vet ./...` clean.
- `go build ./...` clean.
- `go test ./...` passes; add a `theme/theme_test.go` (currently absent — see exploration report) that calls `Resolve` for every public `Name` constant and asserts (a) no error, (b) `Theme.Name` matches the constant string, (c) `len(Theme.Palette) > 0` for themes that should carry a palette.
- `go run ./examples/themes/` regenerates one PNG per theme into `examples/themes/`. Manually compare `theme_ggplot.png`, `theme_fivethirtyeight.png`, `theme_solarize_light2.png`, `theme_dark_background.png` against the matplotlib reference gallery screenshots — they should be visually unmistakable.
- Run `go run ./examples/geometries/point/` (and one or two other unaltered examples) to confirm the new default renders without surprises and that pre-existing examples that don't call `.Theme(...)` now look like the matplotlib `ggplot` style.
