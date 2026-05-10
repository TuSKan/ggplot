package theme

import "image/color"

func init() { MustRegister(SolarizeDark, newSolarizeDark) }

// newSolarizeDark implements Ethan Schoonover's Solarized Dark scheme.
//
// Color reference: https://ethanschoonover.com/solarized/
//
//   - base03 (#002B36): darkest background (figure canvas)
//   - base02 (#073642): dark background highlights (panel)
//   - base01 (#586E75): optional content / subtle grid
//   - base00 (#657B83): body text (light theme) / content
//   - base0  (#839496): body text (dark theme)
//   - base1  (#93A1A1): emphasized text (dark theme)
//
// Palette uses the 8 Solarized accent colors in their canonical order:
// yellow, orange, red, magenta, violet, blue, cyan, green.
func newSolarizeDark() Theme {
	t := baseTheme("solarize_dark")

	// ── Solarized base colors ──────────────────────────────────────────────
	base03 := hex("002B36") // darkest bg
	base02 := hex("073642") // bg highlight / panel
	base01 := hex("586E75") // optional content / subtle lines
	base0 := hex("839496")  // dark-theme body text
	base1 := hex("93A1A1")  // dark-theme emphasized text

	// ── Canvas & panel ────────────────────────────────────────────────────
	t.Background = base03
	t.Panel.Background = base02
	t.Panel.Border = base01
	t.Panel.BorderWidth = 1

	// ── Grid ─────────────────────────────────────────────────────────────
	// White lines with low alpha on the dark panel, dashed for depth.
	t.Grid.MajorColor = hexA("FFFFFF", 45) // ~18% white
	t.Grid.MajorWidth = 0.8
	t.Grid.MinorColor = hexA("FFFFFF", 20)
	t.Grid.MinorWidth = 0.4
	t.Grid.DashPattern = nil // solid; user can switch to {4,4}

	// ── Typography ───────────────────────────────────────────────────────
	t.Text.Title = FontConfig{Family: "sans-serif", Size: 14, Color: base1, Bold: true}
	t.Text.Subtitle = FontConfig{Family: "sans-serif", Size: 11, Color: base0}
	t.Text.AxisTitle = FontConfig{Family: "sans-serif", Size: 11, Color: base0}
	t.Text.TickLabel = FontConfig{Family: "sans-serif", Size: 10, Color: base0}
	t.Text.Legend = FontConfig{Family: "sans-serif", Size: 10, Color: base1}
	t.Text.Annotation = FontConfig{Family: "sans-serif", Size: 10, Color: base0}

	// ── Ticks ─────────────────────────────────────────────────────────────
	t.Ticks.Color = base01
	t.Ticks.Length = 5
	t.Ticks.Width = 1

	// ── Geom defaults ─────────────────────────────────────────────────────
	// Dark panel → use base01 as a subtle edge (visible but not harsh).
	t.Geom.PatchEdgeColor = base01
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9

	// ── Palette: 8 Solarized accent colors ───────────────────────────────
	t.Palette = []color.Color{
		hex("B58900"), // yellow
		hex("CB4B16"), // orange
		hex("DC322F"), // red
		hex("D33682"), // magenta
		hex("6C71C4"), // violet
		hex("268BD2"), // blue
		hex("2AA198"), // cyan
		hex("859900"), // green
	}

	return t
}
