// Package colormap provides a matplotlib-style colormap and color-scale API
// for the ggplot grammar of graphics pipeline.
//
// # Core abstractions
//
//   - [Cmap]   — maps a normalized t in [0,1] to an [gg.RGBA] color. Implementations:
//     [LinearSegmentedCmap] (256-entry LUT, continuous) and [ListedCmap] (discrete).
//   - [Norm]   — maps an arbitrary scalar value into [0,1]. Implementations include
//     [LinearNorm], [LogNorm], [PowerNorm], [TwoSlopeNorm], [BoundaryNorm], [AsinhNorm].
//   - [Scale]  — composes a Norm and a Cmap and adds dataset training. This is the
//     ggplot analogue of matplotlib's ScalarMappable and is what user code actually
//     attaches to a [Plot] via .ScaleColor / .ScaleFill / .ScaleColorManual.
//
// # Categories
//
// Cmaps are grouped into matplotlib's standard categories:
//
//   - [PerceptuallyUniform]: viridis, plasma, inferno, magma, cividis, turbo
//   - [Sequential]:          greys, blues, greens, oranges, reds, purples, ylgn,
//     ylgnbu, ylorbr, ylorrd, bugn, gnbu, bupu, orrd, pubu, pubugn, purd, rdpu
//   - [Diverging]:           rdbu, rdylbu, spectral, coolwarm, brbg, piyg, prgn,
//     puor, rdgy, rdylgn, bwr
//   - [Cyclic]:              twilight, twilight_shifted, hsv
//   - [Qualitative]:         tab10, tab20, set1, set2, set3, paired, pastel1,
//     pastel2, accent, dark2, okabe_ito
//
// # Registry
//
// Built-in cmaps register themselves at init time. Look them up by name with
// [Resolve] (the suffix "_r" returns the reversed colormap, e.g. "viridis_r"):
//
//	cm, err := colormap.Resolve("viridis")
//	cm := colormap.MustResolve("plasma_r")
//
// or use the typed exported variables directly:
//
//	cm := colormap.Viridis
//	cm := colormap.Plasma.Reversed()
//
// # Color literal parsing
//
// [Parse] accepts hex strings (delegates to gg.ParseHex), CSS / X11 named
// colors, the matplotlib "tab:blue" / "tab:orange" aliases, and rgb()/rgba()/
// hsl() functional forms. It returns [gg.RGBA] (float-space color) so the
// result interoperates directly with the renderer without uint8 truncation.
//
// # Design conventions
//
// All Cmap methods return [gg.RGBA] (not image/color.Color) so the rendering
// pipeline can stay in float space. gg.RGBA itself satisfies the standard
// color.Color interface, so values pass through canvas.SetColor unchanged.
//
// LUT data is vendored from matplotlib's BSD/CC0 reference data. The package
// does not reimplement hex parsing or RGB lerping — those come from gg.
package colormap
