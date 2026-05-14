# TODO — Colormap & Theme Integration

## 1. Add Cyclic Colormaps

Add `Twilight` and `Phase` as `NewLinearSegmented("twilight", Cyclic, ...)`.

**Why:** Circular variables (phase angle, azimuth, RA mod 24h, hour of day, orbital phase) need 0° and 360° to be visually identical. Sequential/diverging maps are mathematically wrong for this.

**Colormaps to add:**
- `Twilight` — perceptually-uniform cyclic (matplotlib, primary)
- `Phase` — HSL-based cyclic (complement)

The `Cyclic` category already exists in `registry.go`.

## 2. Add Turbo and JetLegacy

```go
Turbo = NewLinearSegmented("turbo", Miscellaneous, turboLUT)
```

Turbo (Mikhailov 2019) is a "better Jet" — high visual dynamic range, good for heatmaps and engineering dashboards, but **not** colorblind-safe. Never a default.

```go
// JetLegacy is the classic MATLAB jet colormap. It has known perceptual
// defects: non-monotonic luminance, colorblind-unsafe, and phantom
// banding artifacts. Prefer Turbo, Viridis, or Inferno for new work.
JetLegacy = NewLinearSegmented("jet", Miscellaneous, jetLUT)
```

Never a default anywhere.

## 3. Theme → ColorDefaults Mapping

### Type

```go
// ColorDefaults maps a theme to its recommended colormaps by usage category.
type ColorDefaults struct {
    Discrete   colormap.Cmap // qualitative / categorical
    Sequential colormap.Cmap // ordered continuous data
    Diverging  colormap.Cmap // departure from midpoint
    Cyclic     colormap.Cmap // circular variables (phase, angle)
}
```

### Lookup function

```go
func DefaultCmapFor(theme Name, cat colormap.Category) colormap.Cmap
```

Called by the scale pipeline when user hasn't explicitly set a colormap.

### Default assignments

#### Light-background themes

| Theme | Discrete | Sequential | Diverging | Cyclic |
|-------|----------|------------|-----------|--------|
| Default/Ggplot | Tab10 | Viridis | RdBu | Twilight |
| Observable | Observable10 | Viridis | RdBu | Twilight |
| Classic | Tab10 | Blues | RdBu | Twilight |
| Minimal | Tab10 | Viridis | RdBu | Twilight |
| BW | Tab10 | Greys | RdGy | Twilight |
| Dashboard | Tab10 | Blues | RdBu | Twilight |
| Quartz | Tab10 | Viridis | RdBu | Twilight |
| Air | Tab10 | Viridis | RdBu | Twilight |
| Tableau | Tab10 | Blues | RdBu | Twilight |
| TableauColorblind10 | Tab10 | Blues | RdBu | Twilight |
| NASA | Tab10 | Blues | RdBu | Twilight |
| GitHubLight | Tab10 | Blues | RdBu | Twilight |

#### Dark-background themes

| Theme | Discrete | Sequential | Diverging | Cyclic |
|-------|----------|------------|-----------|--------|
| Dark | Observable10 | Plasma | Coolwarm | Twilight |
| DarkBackground | Observable10 | Inferno | Coolwarm | Twilight |
| ObservableDark | Observable10 | Plasma | RdBu | Twilight |
| AstronomyDark | Observable10 | Magma | Spectral | Twilight |
| Ink | Observable10 | Plasma | Coolwarm | Twilight |
| Nord | Observable10 | Magma | Coolwarm | Twilight |
| Dracula | Observable10 | Magma | Coolwarm | Twilight |
| GruvboxDark | Observable10 | Inferno | RdBu | Twilight |
| GitHubDark | Observable10 | Plasma | RdBu | Twilight |
| SolarizeDark | Observable10 | Plasma | Coolwarm | Twilight |
| Cyberpunk | Observable10 | Inferno | Spectral | Twilight |
| Blueprint | Observable10 | Plasma | Coolwarm | Twilight |
| Terminal | Observable10 | Plasma | Coolwarm | Twilight |

#### Editorial / publication themes

| Theme | Discrete | Sequential | Diverging | Cyclic |
|-------|----------|------------|-----------|--------|
| Tufte | Greys | Greys | RdGy | Twilight |
| Academic | OkabeIto | Cividis | RdBu | Twilight |
| Monochrome | Greys | Greys | RdGy | Twilight |
| Newsroom | Tab10 | Blues | RdBu | Twilight |
| Editorial | OkabeIto | Greys | RdGy | Twilight |

#### Accessibility themes

| Theme | Discrete | Sequential | Diverging | Cyclic |
|-------|----------|------------|-----------|--------|
| HighContrast | OkabeIto | Cividis | RdBu | Twilight |
| OkabeIto | OkabeIto | Cividis | BrBG | Twilight |
| Colorblind | OkabeIto | Cividis | BrBG | Twilight |
| Viridis | OkabeIto | Viridis | RdBu | Twilight |
| Cividis | OkabeIto | Cividis | BrBG | Twilight |

### Selection logic

- **Dark backgrounds:** Magma/Inferno/Plasma — bright high-end pops on dark canvas
- **Light backgrounds:** Viridis/Blues — dark low-end visible on white
- **Accessibility:** Cividis — designed for deuteranopia + protanopia
- **Editorial/print:** Greys — B&W safe, no color interpretation needed
- **Cyclic:** Always Twilight — only perceptually-uniform cyclic map

### Future: Color vs Fill split

If needed later, extend without breaking API:

```go
type ColorDefaults struct {
    ColorDiscrete   colormap.Cmap
    ColorContinuous colormap.Cmap
    FillDiscrete    colormap.Cmap
    FillContinuous  colormap.Cmap
    Diverging       colormap.Cmap
    Cyclic          colormap.Cmap
}
```

Zero-value fields fall back to Discrete/Sequential.

## Anti-patterns to avoid

- **No theme-specific colormap variants** (ObservableViridis, NordMagma, etc.)
- **No Jet as default** anywhere
- **No Turbo as default** — useful but not conservative enough for scientific work
