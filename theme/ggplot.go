package theme

import "image/color"

func init() {
	MustRegister(Ggplot, newGgplot)
	MustRegister(Default, newGgplot) // Default is an alias for Ggplot.
}

// newGgplot mirrors matplotlib's ggplot.mplstyle (a port of R ggplot2's
// theme_grey): gray panel, white grid lines as positive space, gray
// edges and text.
//
// Source: matplotlib/lib/matplotlib/mpl-data/stylelib/ggplot.mplstyle
func newGgplot() Theme {
	t := baseTheme("ggplot")
	bg := hex("E5E5E5")
	axisLabel := hex("555555")

	t.Background = color.White
	t.Panel.Background = bg
	t.Panel.Border = color.White
	t.Panel.BorderWidth = 1

	t.Grid.MajorColor = color.White
	t.Grid.MajorWidth = 1
	t.Grid.MinorColor = hexA("FFFFFF", 160)
	t.Grid.MinorWidth = 0.5
	t.Grid.DashPattern = nil

	t.Text.Title = FontConfig{Family: "sans-serif", Size: 14, Color: axisLabel, Bold: true}
	t.Text.Subtitle = FontConfig{Family: "sans-serif", Size: 11, Color: axisLabel}
	t.Text.AxisTitle = FontConfig{Family: "sans-serif", Size: 11, Color: axisLabel}
	t.Text.TickLabel = FontConfig{Family: "sans-serif", Size: 10, Color: axisLabel}
	t.Text.Legend = FontConfig{Family: "sans-serif", Size: 10, Color: axisLabel}
	t.Text.Annotation = FontConfig{Family: "sans-serif", Size: 10, Color: axisLabel}

	t.Ticks.Color = axisLabel

	// Mirrors matplotlib's ggplot.mplstyle: patch.edgecolor: white
	t.Geom.PatchEdgeColor = color.White
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	t.Palette = []color.Color{
		hex("E24A33"), // red
		hex("348ABD"), // blue
		hex("988ED5"), // purple
		hex("777777"), // gray
		hex("FBC15E"), // yellow
		hex("8EBA42"), // green
		hex("FFB5B8"), // pink
	}
	return t
}
