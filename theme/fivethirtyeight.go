package theme

import "image/color"

func init() { MustRegister(Fivethirtyeight, newFivethirtyeight) }

// newFivethirtyeight mirrors matplotlib's fivethirtyeight.mplstyle:
// light gray panel that bleeds into the figure background, thick lines,
// no tick marks, FiveThirtyEight color cycle.
//
// Source: matplotlib/lib/matplotlib/mpl-data/stylelib/fivethirtyeight.mplstyle
func newFivethirtyeight() Theme {
	t := baseTheme("fivethirtyeight")

	bg := hex("F0F0F0")
	t.Background = bg
	t.Panel.Background = bg
	t.Panel.Border = bg
	t.Panel.BorderWidth = 3

	t.Grid.MajorColor = hex("CBCBCB")
	t.Grid.MajorWidth = 1
	t.Grid.MinorColor = hexA("CBCBCB", 120)
	t.Grid.MinorWidth = 0.5
	t.Grid.DashPattern = nil

	t.Text.Title = FontConfig{Family: "sans-serif", Size: 18, Color: gray(30), Bold: true}
	t.Text.Subtitle = FontConfig{Family: "sans-serif", Size: 14, Color: gray(60)}
	t.Text.AxisTitle = FontConfig{Family: "sans-serif", Size: 14, Color: gray(60)}
	t.Text.TickLabel = FontConfig{Family: "sans-serif", Size: 12, Color: gray(60)}
	t.Text.Legend = FontConfig{Family: "sans-serif", Size: 12, Color: gray(60)}
	t.Text.Annotation = FontConfig{Family: "sans-serif", Size: 12, Color: gray(60)}

	// xtick.major.size / ytick.major.size: 0
	t.Ticks.Length = 0
	t.Ticks.Width = 0
	t.Ticks.Color = color.Transparent

	// fivethirtyeight: uniform #F0F0F0 background + #CBCBCB grid →
	// use grid color as edge for a clean, publication-style outline.
	t.Geom.PatchEdgeColor = hex("CBCBCB")
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	t.Palette = []color.Color{
		hex("008FD5"),
		hex("FC4F30"),
		hex("E5AE38"),
		hex("6D904F"),
		hex("8B8B8B"),
		hex("810F7C"),
	}
	return t
}
