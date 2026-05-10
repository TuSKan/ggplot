package theme

import "image/color"

func init() { MustRegister(DarkBackground, newDarkBackground) }

// newDarkBackground mirrors matplotlib's dark_background.mplstyle:
// black canvas with white axes, ticks, and text; pastel color cycle.
//
// Source: matplotlib/lib/matplotlib/mpl-data/stylelib/dark_background.mplstyle
func newDarkBackground() Theme {
	t := baseTheme("dark_background")

	t.Background = color.Black
	t.Panel.Background = color.Black
	t.Panel.Border = color.White
	t.Panel.BorderWidth = 1

	t.Grid.MajorColor = hexA("FFFFFF", 60)
	t.Grid.MajorWidth = 0.5
	t.Grid.MinorColor = hexA("FFFFFF", 30)
	t.Grid.MinorWidth = 0.3
	t.Grid.DashPattern = []float64{4, 4}

	t.Text.Title = FontConfig{Family: "sans-serif", Size: 14, Color: color.White, Bold: true}
	t.Text.Subtitle = FontConfig{Family: "sans-serif", Size: 11, Color: gray(220)}
	t.Text.AxisTitle = FontConfig{Family: "sans-serif", Size: 11, Color: color.White}
	t.Text.TickLabel = FontConfig{Family: "sans-serif", Size: 10, Color: gray(200)}
	t.Text.Legend = FontConfig{Family: "sans-serif", Size: 10, Color: color.White}
	t.Text.Annotation = FontConfig{Family: "sans-serif", Size: 10, Color: color.White}

	t.Ticks.Color = color.White

	// dark_background: black panel + white grid → white edges.
	t.Geom.PatchEdgeColor = color.White
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9

	t.Palette = []color.Color{
		hex("8DD3C7"),
		hex("FEFFB3"),
		hex("BFBBD9"),
		hex("FA8174"),
		hex("81B1D2"),
		hex("FDB462"),
		hex("B3DE69"),
		hex("BC82BD"),
		hex("CCEBC4"),
		hex("FFED6F"),
	}

	return t
}
