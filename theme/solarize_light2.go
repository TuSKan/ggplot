package theme

import "image/color"

func init() { MustRegister(SolarizeLight2, newSolarizeLight2) }

// newSolarizeLight2 mirrors matplotlib's Solarize_Light2.mplstyle, based
// on Ethan Schoonover's Solarized palette.
//
// Source: matplotlib/lib/matplotlib/mpl-data/stylelib/Solarize_Light2.mplstyle
func newSolarizeLight2() Theme {
	t := baseTheme("solarize_light2")

	base00 := hex("657B83") // body text
	base2 := hex("EEE8D5")  // axes face
	base3 := hex("FDF6E3")  // figure face

	t.Background = base3
	t.Panel.Background = base2
	t.Panel.Border = base00
	t.Panel.BorderWidth = 1

	t.Grid.MajorColor = base3
	t.Grid.MajorWidth = 1
	t.Grid.MinorColor = hexA("FDF6E3", 160)
	t.Grid.MinorWidth = 0.5
	t.Grid.DashPattern = nil

	t.Text.Title = FontConfig{Family: "sans-serif", Size: 16, Color: base00, Bold: true}
	t.Text.Subtitle = FontConfig{Family: "sans-serif", Size: 12, Color: base00}
	t.Text.AxisTitle = FontConfig{Family: "sans-serif", Size: 12, Color: base00}
	t.Text.TickLabel = FontConfig{Family: "sans-serif", Size: 10, Color: base00}
	t.Text.Legend = FontConfig{Family: "sans-serif", Size: 10, Color: base00}
	t.Text.Annotation = FontConfig{Family: "sans-serif", Size: 10, Color: base00}

	t.Ticks.Color = base00

	// Solarize_Light2: warm beige panel (#EEE8D5), base3 (#FDF6E3) grid →
	// use base3 as edge (matches grid, harmonious with Solarized palette).
	t.Geom.PatchEdgeColor = base3
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	t.Palette = []color.Color{
		hex("268BD2"), // blue
		hex("2AA198"), // cyan
		hex("859900"), // green
		hex("CB4B16"), // orange
		hex("D33682"), // magenta
		hex("6C71C4"), // violet
		hex("657B83"), // base00
		hex("93A1A1"), // base1
	}

	return t
}
