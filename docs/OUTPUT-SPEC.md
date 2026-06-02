# ggplot — Output Layer Specification

> Production specification for the unified output layer: one model that serves
> file export (PNG/SVG/PDF), in-memory images, an interactive desktop window,
> and an interactive browser canvas — built on the existing
> `Plot → Build → Figure.Draw` pipeline.

Status: Phases 1–5 implemented. Phase 6 (`output/web`, WASM) is remaining
work. Target: `github.com/TuSKan/ggplot`, `go 1.26.2`.

---

## 1. Scope and goals

The output layer answers one question — *where do rendered pixels/vectors go* —
with a single abstraction. Four destinations, one model:

| Destination | Lifecycle | Interactive |
|---|---|---|
| File (`.png`, `.svg`, `.pdf`) | static — one frame | no |
| In-memory `image.Image` | static — one frame | no |
| Desktop GPU window | live — many frames | yes |
| Browser `<canvas>` (WASM) | live — many frames | yes |

Goals: (1) one `Surface` abstraction for all four; (2) `*Built` and `*Plot`
satisfy the new interfaces with near-zero change; (3) core `output` is pure Go
— WebGPU and windowing reachable only through subpackages; (4) platform
selection by **blank import**, not by build tags in user code; (5) the existing
public API (`Plot.Save`, `WriteTo`) is preserved as façades.

---

## 2. Locked design decisions

These are settled; the spec below is their consequence.

1. **`Acquire` / `Commit`** frame model on `Surface`.
2. **`Figure`** is the name of the drawable abstraction.
3. **`LiveSurface`** is the name of the interactive `Surface` extension.
4. **One `Surface` interface** — static vs. live distinguished by interface
   composition (`LiveSurface` embeds `Surface`), not by separate hierarchies.
5. **`window.Show` / `web.Mount` are package-level functions**, not methods on
   `*Plot` — `*Plot` stays platform-neutral with one method set everywhere.
6. **`Plot.Build` returns `output.Figure`** (concretely a `*Built`). `Figure` is
   minimal (`Draw` only). Introspection is via `*ggplot.Built` type assertion
   (option B). `*Built` stays exported.
7. **Package name is `output`.**
8. **`canvas.GGCanvas` is renamed to `canvas.RasterCanvas`** — committed change
   (§9).
9. **Platform selection = build-tagged leaves + a blank-import registry** (§7).
   Build tags are confined to the files that touch a platform; users select a
   platform by importing a subpackage for its `init()` side effect.

---

## 3. Architecture: three layers + interaction

```
  ggplot.Plot ──Build──▶ output.Figure ( = *ggplot.Built )
                              │  Draw(ctx, canvas.Canvas, w, h)
                              ▼
  ┌───────────────────────────────────────────────────────────────┐
  │ LAYER A — canvas : the drawing seam                            │
  │   Canvas (path-level API) ── RasterCanvas | RecordingCanvas    │
  └───────────────────────────────────────────────────────────────┘
                              │
  ┌───────────────────────────────────────────────────────────────┐
  │ LAYER B — output : destinations                                │
  │   Figure, Source, Sizer                                        │
  │   Surface  ── Acquire / Commit / Bounds / Close                │
  │   LiveSurface  ── Surface + Events()                           │
  │   Render(ctx, Figure, Surface)         one frame, any surface  │
  │   registry: NewSurface / NewLiveSurface by name                │
  └───────────────────────────────────────────────────────────────┘
                              │
  ┌───────────────────────────────────────────────────────────────┐
  │ LAYER C — output (interaction, live surfaces only)             │
  │   Session  ── build, draw, event loop, fast/slow path          │
  │   Controller ── per-event policy (pan/zoom/hover, brushing)    │
  └───────────────────────────────────────────────────────────────┘

  Platform leaves (build-tagged, register into the Layer-B registry):
    output/file  output/image  output/window(!js)  output/web(js,wasm)
```

Front of pipe = `RasterCanvas`/`RecordingCanvas` (drawing). Back of pipe =
`Surface` (destination). `gg.Context` is the shared currency between them; the
window surface exploits that for a zero-copy handoff (§6.3).

---

## 4. Layer A — `canvas` (the drawing seam)

Mostly exists today. One rename, one new constructor.

```go
package canvas

// Canvas — path-level drawing API. Every Figure renders through this.
// UNCHANGED from today (MoveTo, CubicTo, Fill, Stroke, Clip, DrawStringAnchored,
// MeasureString, Clear, Width, Height, DrawImage, Close, ...).
type Canvas interface { /* unchanged */ }

// RasterCanvas — the pixel backend, gg-powered. RENAMED from GGCanvas.
type RasterCanvas struct { /* unchanged internals */ }

func NewRasterCanvas(width, height int) *RasterCanvas      // was NewGGCanvas    — GPU
func NewRasterCanvasCPU(width, height int) *RasterCanvas   // was NewGGCanvasCPU — CPU
func RasterFromContext(ctx *gg.Context) *RasterCanvas      // was FromGGContext  — borrow

func (c *RasterCanvas) Context() *gg.Context
func (c *RasterCanvas) Image() image.Image
func (c *RasterCanvas) EncodePNG(w io.Writer) error
func (c *RasterCanvas) SavePNG(path string) error
func (c *RasterCanvas) Close() error
// ... all other methods unchanged.

// RecordingCanvas — vector backend. UNCHANGED. Records ops; replayed into
// SVG/PDF via canvas.ExportSVG / canvas.ExportPDF.
type RecordingCanvas struct { /* unchanged */ }
```

`RasterFromContext` is a **borrow** — it wraps a `gg.Context` owned by something
else (the window surface, §6.3). A `RasterCanvas` obtained via `RasterFromContext`
MUST NOT be `Close()`d by the caller; the context owner controls lifecycle.

---

## 5. Layer B — `output` (the generic destination)

### 5.1 The drawable

```go
package output

// Figure is something that can paint itself onto a Canvas at a given size.
// *ggplot.Built satisfies Figure with NO change — Built.Draw already has
// exactly this signature.
type Figure interface {
    Draw(ctx context.Context, dst canvas.Canvas, width, height int) error
}

// Source yields a fresh Figure. *ggplot.Plot is a Source. Live surfaces hold a
// Source so they can rebuild when an interaction crosses the trained data
// extent (scales retrain, stats recompute — the slow path, §8).
type Source interface {
    Build(ctx context.Context) (Figure, error)
}

// Sizer is an optional Figure extension: given a width, propose a height.
// *ggplot.Built implements Sizer (wrapping its existing autoHeight rules).
// Façades depend on Figure + optional Sizer — never on the concrete *Built.
type Sizer interface {
    PreferredSize(width int) (w, h int)
}

// Measurable is an optional Figure extension that exposes per-panel pixel
// geometry and data-space bounds after a Draw call. Controllers use it to
// convert pixel mouse positions to data coordinates for data-space pan/zoom.
type Measurable interface {
    PanelInfos() []PanelInfo
}

// PanelInfo describes the pixel geometry and data-space bounds of one panel.
type PanelInfo struct {
    Index      int              // zero-based panel index
    Bounds     image.Rectangle  // data-area rectangle in figure pixel coords
    XRange     [2]float64       // current (possibly zoomed) X data-space bounds
    YRange     [2]float64       // current (possibly zoomed) Y data-space bounds
    OrigXRange [2]float64       // full trained X bounds (for reset)
    OrigYRange [2]float64       // full trained Y bounds (for reset)
}

// Zoomable is an optional Figure extension that supports O(1) viewport changes
// without rebuilding the figure from source. The controller mutates scale
// bounds directly on the built figure, then triggers ActionRedraw — no data
// iteration, no stat recomputation, no memory allocation.
type Zoomable interface {
    SetPanelViewport(panelIndex int, xlim, ylim [2]*float64)
    ResetViewport()
}

// Imager is implemented by surfaces that publish an in-memory image (the
// "image" surface). Image returns the frame published by the last Commit.
type Imager interface {
    Image() image.Image
}
```

### 5.2 The destination

```go
// Surface is a destination a Figure is drawn to. ONE interface for file,
// in-memory image, desktop GPU window, and browser canvas.
type Surface interface {
    // Acquire returns the drawing Canvas for the next frame, sized to Bounds().
    // The returned Canvas is valid only until the next Commit.
    Acquire(ctx context.Context) (canvas.Canvas, error)

    // Commit finalizes the frame acquired by Acquire:
    //   file   -> encode + flush bytes to disk
    //   image  -> publish the in-memory image
    //   window -> submit GPU work + present (zero-copy, §6.3)
    //   web    -> submit + swapchain present
    Commit(ctx context.Context) error

    // Bounds is the current logical drawing size (device scale handled
    // internally). Fixed for file/image; tracks the window/canvas for live.
    Bounds() image.Rectangle

    Close() error
}

// LiveSurface is a Surface with an interactive lifecycle. Desktop windows and
// browser canvases implement it; file and image surfaces do not.
type LiveSurface interface {
    Surface
    Events() <-chan Event
}
```

**Single-shot vs. multi-shot (decision #4).** One interface. Static surfaces
(`file`, `image`) accept exactly one `Acquire`/`Commit` cycle and return
`ErrSurfaceConsumed` on a second `Acquire`. Live surfaces accept repeated
cycles. The contract is documented per implementation; the type is shared.

### 5.3 The one-frame primitive

```go
// Render draws a Figure onto a Surface and presents exactly ONE frame.
// This is the entire static story — PNG, SVG, PDF, in-memory image — and is
// also the per-frame primitive the Session loop calls for live surfaces.
func Render(ctx context.Context, fig Figure, surf Surface) error {
    cv, err := surf.Acquire(ctx)
    if err != nil {
        return err
    }
    b := surf.Bounds()
    if err := fig.Draw(ctx, cv, b.Dx(), b.Dy()); err != nil {
        return err
    }
    return surf.Commit(ctx)
}
```

### 5.4 Events

```go
type EventKind uint8

const (
    EventResize EventKind = iota
    EventPointerMove
    EventPointerDown
    EventPointerUp
    EventScroll
    EventKey
    EventClose
    EventDoubleClick // double-click / double-tap for viewport reset
)

// Event is the platform-neutral input event. output/window translates OS
// events into it; output/web translates DOM events into it. The Session and
// Controller see only this type — never a platform event.
type Event struct {
    Kind      EventKind
    X, Y      float64   // logical coordinates
    DX, DY    float64   // scroll / drag deltas
    Buttons   uint8
    Key       string
    Modifiers uint8
}
```

---

## 6. Layer B — the four concrete surfaces

Each lives in its own subpackage and registers itself (§7). `output` core
imports none of them.

| Surface | Package | Build constraint | `Acquire` returns | `Commit` does |
|---|---|---|---|---|
| `fileSurface` | `output/file` | none | `RasterCanvas` or `RecordingCanvas` by extension | encode PNG/SVG/PDF, write file |
| `imageSurface` | `output/image` | none | `NewRasterCanvasCPU` | publish `image.Image` |
| `windowSurface` | `output/window` | `//go:build !js` | `RasterFromContext(ggcanvas.Context())` | `ggcanvas.RenderDirect` — zero-copy |
| `canvasSurface` | `output/web` | `//go:build js && wasm` | `RasterFromContext(...)` over wgpu browser surface | swapchain present |

### 6.1 `output/file`

Resolves the file extension to an encoder. `.png` → `RasterCanvas` (CPU path is
headless-safe; GPU optional via `RenderOpt`). `.svg`/`.pdf` → `RecordingCanvas`
→ `canvas.ExportSVG` / `canvas.ExportPDF`. `Commit` flushes to disk.

### 6.2 `output/image`

`Acquire` → `NewRasterCanvasCPU`. `Commit` is a no-op that publishes the result;
the surface exposes `Image() image.Image`. No GPU, no window — pure Go.

### 6.3 `output/window` — desktop, zero-copy

`//go:build !js`. Wraps a `gogpu` window and a `gg/integration/ggcanvas.Canvas`.

- `window.Show` is a standalone package-level function (not a `Surface`
  registered in the blank-import registry). gogpu owns the run loop via
  callbacks, which is incompatible with the `Surface.Acquire`/`Commit` pull
  model, so the window backend integrates directly.
- The plot draws into `canvas.RasterFromContext(ggcanvas.Context())` — a
  borrowed `gg.Context`. The borrowed `RasterCanvas` is never `Close()`d;
  `ggcanvas` owns the context.
- Presentation is zero-copy via `ggcanvas.Render` onto the swapchain texture.
- Interaction reuses the platform-neutral `Controller` / `State` policy from
  the `output` core. gogpu input callbacks are translated to `output.Event`s
  and dispatched through the same `DataSpaceController` (§8).
- **Default controller:** `window.Show` defaults to `DataSpaceController()` —
  data-space pan/zoom with per-panel hit-testing and double-click reset.
- `WithRebuildDelay` enables async, debounced rebuilds: rapid rebuild requests
  coalesce, and the build runs in a background goroutine while the last good
  figure keeps drawing. `WithRebuildError` sets a handler for non-fatal
  background build errors.
- **Diagnostic options:**
  - `WithFPS()` — enables an FPS counter overlay (EMA-smoothed) in the
    top-right corner.
  - `WithPprof()` — starts a pprof HTTP server on `localhost:6060` for live
    CPU/memory profiling during interaction.
- **HiDPI workaround:** On Windows HiDPI, gogpu's scroll event coordinates
  are in physical pixels while panel geometry is in logical pixels. The window
  normalizes scroll coordinates by dividing by the scale factor.
- **Double-click detection:** Two clicks within 400ms are coalesced into an
  `EventDoubleClick` (viewport reset via `Zoomable.ResetViewport`).
- Requires `gogpu` ≥ v0.40.

### 6.4 `output/web` — browser, WASM

`//go:build js && wasm`. Wraps an HTML `<canvas>` and the `wgpu` browser
backend (`navigator.gpu`). Requires `wgpu` ≥ v0.28. `Acquire` returns a
`RasterCanvas` over the browser GPU context; `Commit` presents to the swapchain;
`Events()` is fed by `syscall/js` DOM listeners. Drives the same `Session` as
the desktop window.

---

## 7. Platform selection: build-tagged leaves + blank-import registry

Build tags are **mandatory and unavoidable** for a tree that cross-compiles to
both native and `js/wasm` — they are confined to the files that touch a
platform (`output/window/*.go` is `!js`; `output/web/*.go` is `js && wasm`).
On top of that, a registry turns *platform selection* into a blank import, so
no build tag ever appears in ggplot user code.

```go
// output/registry.go — pure Go, no build tags.
type SurfaceFactory func(ctx context.Context, opt SurfaceOptions) (Surface, error)

var surfaceRegistry = map[string]SurfaceFactory{}

// Register is called from a platform subpackage's init().
func Register(name string, f SurfaceFactory) { surfaceRegistry[name] = f }

// NewSurface constructs a registered surface by name. Returns ErrUnknownSurface
// if the corresponding subpackage was not blank-imported.
func NewSurface(ctx context.Context, name string, opt ...SurfaceOpt) (Surface, error)

// NewLiveSurface is NewSurface + a runtime assertion to LiveSurface.
func NewLiveSurface(ctx context.Context, name string, opt ...SurfaceOpt) (LiveSurface, error)
```

Each subpackage that uses the registry registers in `init()`:

```go
// output/file/file.go       (no build tag)
func init() { output.Register("file", newFileSurface) }

// output/image/image.go     (no build tag)
func init() { output.Register("image", newImageSurface) }

// output/web/web.go         //go:build js && wasm
func init() { output.Register("web", newCanvasSurface) }
```

**Note:** `output/window` does **not** register into the registry. gogpu owns
the run loop via callbacks (callback-driven), which is incompatible with the
pull-model `Acquire`/`Commit` contract. Instead, `window.Show` is a standalone
package-level function. User code imports `output/window` and calls `Show`
directly — no `NewSurface` / `NewLiveSurface` indirection.

User code selects a platform purely by which subpackage it imports — no
`//go:build` in user code:

```go
import "github.com/TuSKan/ggplot/output/window"

window.Show(ctx, plot, window.WithSize(900, 600))
```

A `js/wasm` build blank-imports `output/web` instead; `output/window`'s `!js`
files are simply excluded from that build by their own tag. The build tags are
invisible to ggplot's users — selection is by import. This is the agreed hybrid.

---

## 8. Layer C — `Session` and `Controller` (live surfaces only)

### 8.1 Core types

```go
// State is the mutable interaction state shared between the Session and its
// Controller. The controller reads events and mutates the viewport via
// Zoomable; the session reads the state to render.
type State struct {
    Bounds image.Rectangle  // surface's current logical drawing size
    Figure Figure           // current figure (read-only for controllers)
}

// Controller decides, per event, what the Session does next. The default
// is DataSpaceController (data-space pan/zoom); swap it via WithController
// for linked brushing, custom gestures, etc.
type Controller interface {
    OnEvent(ev Event, st *State) Action
}

// ControllerFunc adapts a plain function to the Controller interface.
type ControllerFunc func(ev Event, st *State) Action

type Action uint8

const (
    ActionIgnore Action = iota
    ActionRedraw   // fast path — redraw current Figure (viewport updated via Zoomable)
    ActionRebuild  // slow path — Source.Build again (data extent changed)
    ActionExport   // run a static Surface against the current frame
    ActionClose
)

// Session drives a Source onto a LiveSurface: build once, draw, then re-render
// on events. It owns the fast-path / slow-path policy.
type Session struct { /* unexported */ }

func NewSession(src Source, surf LiveSurface, opt ...SessionOpt) *Session
func (s *Session) Run(ctx context.Context) error  // event loop until surface closes
```

### 8.2 Data-space interaction model

Interaction operates in **data space**, not canvas space. The controller
mutates scale bounds directly on the built `*Figure` via the `Zoomable`
interface — O(1) per event, no data iteration, no stat recomputation.
Axes remain at fixed screen positions; only the data visible within the
panel changes. Tick labels regenerate on the next `Draw` call.

```go
// DataSpaceController provides data-space pan (drag) and zoom (scroll wheel)
// that operates on scale bounds rather than a canvas-level affine transform.
// For faceted plots, zoom/pan applies only to the panel under the cursor.
// Double-click resets the viewport via Zoomable.ResetViewport.
func DataSpaceController() Controller
```

The controller queries `Measurable.PanelInfos()` for per-panel pixel→data
coordinate mapping, then calls `Zoomable.SetPanelViewport()` to shift the
scale bounds. For non-`Measurable` figures, it assumes a single panel at
index 0. For non-`Zoomable` figures, pan/zoom is silently a no-op.

### 8.3 Fast path vs. slow path

A `*Built` is valid only for the data extent it was built against — `Build`
trains scales and runs stats. So the `Session` classifies every interaction:

- `ActionRedraw` — the `DataSpaceController` has updated the viewport via
  `Zoomable.SetPanelViewport`. Call `output.Render` once. GPU-bound.
- `ActionRebuild` — viewport crosses the trained data bounds, or a brush changes
  a stat input. `Session` calls `Source.Build(ctx)` again, off the event
  goroutine, debounced; the last good `Figure` keeps drawing until the new one
  is ready. For a BigQuery-backed dataset this pushes a new `WHERE` window down
  to SQL.

This policy lives in `Session` (pure Go), so desktop and browser inherit it
identically and it is testable headless with a scripted fake `LiveSurface`.

### 8.4 Async rebuild

`WithRebuildDelay(d)` makes rebuilds asynchronous and debounced:

1. Rapid `ActionRebuild` triggers within `d` coalesce into a single background
   `Source.Build`.
2. The last good figure stays on screen while the build runs.
3. If a new rebuild request arrives during an in-flight build, it sets a `dirty`
   flag and is replayed when the current build finishes.
4. `WithRebuildError(fn)` receives non-fatal errors from background builds; the
   session continues with the last good figure.
5. On session shutdown, pending async builds are cancelled via `context`
   cancellation.

Interaction is deliberately absent from the `Surface` contract — a PNG export
never links an event loop.

---

## 9. Committed rename: `GGCanvas` → `RasterCanvas`

A committed change in this spec. `canvas.GGCanvas` and
`gg/integration/ggcanvas.Canvas` are different layers (drawing vs. presentation);
the near-identical names invite confusion. The drawing-side type is renamed to
state its role — a raster backend of `canvas.Canvas`, peer to `RecordingCanvas`.

| Before | After |
|---|---|
| `canvas.GGCanvas` (type) | `canvas.RasterCanvas` |
| `canvas.NewGGCanvas` | `canvas.NewRasterCanvas` |
| `canvas.NewGGCanvasCPU` | `canvas.NewRasterCanvasCPU` |
| `canvas.FromGGContext` | `canvas.RasterFromContext` |

Call sites to update (verified against current source): `canvas/gg.go`
(definition); `ggplot.go` — `Built.DrawCanvas` (return type + doc),
and the panel sub-canvas in `Built.Draw` (~line 2098). No behavioral change.
The file `canvas/gg.go` may be renamed to `canvas/raster.go` for consistency.

`Built.Save` and `Built.WriteTo` have been removed — all output goes through
`Plot.Save` / `Plot.Encode` / `output.Render`.

---

## 10. ggplot-facing API

```go
// --- static, on *Plot ---
func (p *Plot) Save(ctx context.Context, filename string, w, h int, opt ...RenderOpt) error
func (p *Plot) Encode(ctx context.Context, dst io.Writer, format string, w, h int, opt ...RenderOpt) (int64, error)
func (p *Plot) Image(ctx context.Context, w, h int, opt ...RenderOpt) (image.Image, error)

// --- escape hatch: any custom Surface, on *Built ---
func (b *Built) RenderTo(ctx context.Context, surf output.Surface) error

// --- build now returns the Figure interface (decision #6, option B) ---
func (p *Plot) Build(ctx context.Context) (output.Figure, error)  // concretely a *Built

// --- live, package-level functions (decision #5) ---
func window.Show(ctx context.Context, src output.Source, opt ...window.Opt) error  // //go:build !js
func web.Mount(ctx context.Context, src output.Source, containerID string, opt ...MountOpt) error // //go:build js
```

**Removed:** `Built.Save` and `Built.WriteTo` — these bypassed the output layer
with duplicated encoding logic. Use `Plot.Save`, `Plot.Encode`, or
`output.Render` + a surface instead.

`Save`, `Encode`, `Image`, `RenderTo`, `window.Show`, `web.Mount` all depend on
`Figure` + optional `Sizer` — never on the concrete `*Built`. Only user-facing
introspection asserts to `*ggplot.Built`:

```go
fig, _ := plot.Build(ctx)
if b, ok := fig.(*ggplot.Built); ok {
    fmt.Println(b.Explain())          // also: NumPanels, LayerData, Theme,
}                                     // Labels, PanelLayout, PipelineFor
```

`*ggplot.Built` stays exported. `Build`'s return type widens to `Figure`; the
concrete type and its full introspection API are unchanged.

### Three call flows, one model

```go
// PNG
surf, _ := output.NewSurface(ctx, "file", output.WithPath("plot.png"), output.WithSize(800, 600))
fig, _ := plot.Build(ctx)
output.Render(ctx, fig, surf)

// Desktop UI            (import _ ".../output/window")
window.Show(ctx, plot)   // builds, opens a window, runs Session.Run

// Browser               (import _ ".../output/web", GOOS=js GOARCH=wasm)
web.Mount(ctx, plot, "plot-container")
```

---

## 11. Migration phasing

Each phase ships independently; the existing public API is preserved throughout.

> **Status:** Phases 1–5 are implemented (unreleased). Phase 4 also has async/debounced rebuild (`WithRebuildDelay`/`WithRebuildError`), data-space interaction (`DataSpaceController`, `Measurable`, `Zoomable`, `PanelInfo`), and a runnable headless example (`examples/session/`). Phase 5 ships `window.Show` with `WithRebuildDelay`/`WithRebuildError` for async builds, `WithFPS`/`WithPprof` diagnostic options, HiDPI scroll normalization, double-click viewport reset, and `examples/window/`. The affine viewport model (`State.OffsetX/OffsetY/Scale`, `DefaultController`, `DrawViewport`) has been removed — all interaction uses the data-space model via `Zoomable`. `Built.Save` and `Built.WriteTo` have been removed — all output goes through the `output` layer. Phase 6 (`output/web`, WASM) is the remaining work.

| Phase | Deliverable | Risk |
|---|---|---|
| **1** ✅ | Rename `GGCanvas → RasterCanvas` (§9). Mechanical; gated by golden tests. | Low |
| **2** ✅ | `output` core: `Figure`, `Source`, `Sizer`, `Measurable`, `Zoomable`, `PanelInfo`, `Imager`, `Surface`, `LiveSurface`, `Event`, `Render`, registry. `Built` implements `Figure`+`Sizer`+`Measurable`+`Zoomable`; `Build` returns `Figure`. | Low |
| **3** ✅ | `output/file` + `output/image`. `Plot.Save`/`Encode`/`Image` become façades over `Render`. | Low |
| **4** ✅ | `output/session`: `Session`, `Controller`, `DataSpaceController`, `ControllerFunc`, fast/slow path, async rebuild, headless `LiveSurface` tests. | Med |
| **5** ✅ | `output/window` + `window.Show` — `ggcanvas` zero-copy desktop, `WithFPS`, `WithPprof`, HiDPI workaround, double-click reset. | Med |
| **6** | Bump `wgpu` ≥ 0.28; `output/web` + `web.Mount`; `cmd/ggplot-wasm`. Requires the `gg` fork to compile for `js/wasm`. | Med — see §12. |

---

## 12. Prerequisites and risks

- **`wgpu` bump to ≥ v0.28.0** — the pinned `v0.27.5` predates the browser
  WebGPU backend. Adopting `gogpu/ui`/`gogpu` for the window path pulls a
  ≥0.28 `wgpu` anyway, so Phases 5–6 align.
- **The `gg` fork (`TuSKan/gogpugg`) must compile for `GOOS=js GOARCH=wasm`** —
  gates Phase 6. The `replace` directive also redirects any transitive `gg`
  dependency to the fork; keep it API-compatible with upstream `gg v0.48.x`.
- **`gg/gpu` accelerator** is required for the window zero-copy path
  (`import _`). Whether it builds for `js/wasm` is the Phase-6 open question.
- **Determinism** — `output/file` encoders produce byte-stable output; the
  existing `testdata/golden` suite gates the rename and the façade refactor.
- **Resource lifecycle** — every live `Surface` and `Session` owns its GPU
  resources and releases them on `Close()`; `context` cancellation tears down.
  A borrowed `RasterCanvas` (`RasterFromContext`) is never closed by its borrower.

---

## 13. Rendering performance optimizations

Interactive rendering at high point counts (100K–1M) required several
optimizations in the drawing layer (`drawer.go`). These are transparent to
the output layer but were enabled by the interactive window use case.

| Optimization | Technique | Effect |
|---|---|---|
| **Batched Fill** | Accumulate same-color shapes into one `Fill()` call | Eliminates per-shape GPU submit overhead |
| **Pixel occupancy decimation** | Bitmap tracks which screen cells are occupied; skip duplicate points | O(N) → O(visible pixels) for dense scatters |
| **Adaptive grid coarsening** | Bin sizes 2×2, 3×3, 4×4 scale with oversampling ratio | Caps total GPU geometry at ~screen resolution |
| **Min/max line decimation** | Per-pixel-column min/max Y envelope | 100K LineTo → ~2K for dense time series |
| **Rectangle for tiny points** | `DrawRectangle` for radius ≤ 1.5px instead of `DrawCircle` | 16× fewer GPU triangles (4 `LineTo` vs 4 `CubicTo`) |
| **Filled rug rectangles** | `Fill` instead of `Stroke` for rug marks | Avoids `StrokeExpander` CPU cost |
| **Viewport culling** | Skip off-screen geometry during pan/zoom | Only visible points reach the GPU |

Stress test benchmarks (`examples/stress/`):

| Scenario | Points | Target FPS |
|---|---|---|
| scatter | 50K × 4 groups | ≥ 25 |
| line | 10 × 100K | ≥ 25 |
| composite | 100K multi-layer | ≥ 20 |
| dense | 500K cluster | ~ 15 |
| million | 1M spiral | ~ 11 |

