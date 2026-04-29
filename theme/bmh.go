package theme

import "image/color"

func init() { MustRegister(Bmh, newBmh) }

// newBmh mirrors matplotlib's bmh.mplstyle (Bayesian Methods for Hackers
// by Cameron Davidson-Pilon): light-gray panel, dashed grid, 10-color
// colorblind-friendly cycle.
//
// Source: matplotlib/lib/matplotlib/mpl-data/stylelib/bmh.mplstyle
func newBmh() Theme {
	t := baseTheme("bmh")

	t.Background = color.White
	t.Panel.Background = hex("EEEEEE")
	t.Panel.Border = color.White
	t.Panel.BorderWidth = 1

	t.Grid.MajorColor = color.White
	t.Grid.MajorWidth = 0.8
	t.Grid.MinorColor = hexA("FFFFFF", 160)
	t.Grid.MinorWidth = 0.4
	t.Grid.DashPattern = []float64{4, 4}

	t.Text.Title = FontConfig{Family: "sans-serif", Size: 14, Color: gray(40), Bold: true}
	t.Text.Subtitle = FontConfig{Family: "sans-serif", Size: 11, Color: gray(40)}
	t.Text.AxisTitle = FontConfig{Family: "sans-serif", Size: 11, Color: gray(40)}
	t.Text.TickLabel = FontConfig{Family: "sans-serif", Size: 10, Color: gray(60)}
	t.Text.Legend = FontConfig{Family: "sans-serif", Size: 10, Color: gray(40)}
	t.Text.Annotation = FontConfig{Family: "sans-serif", Size: 10, Color: gray(60)}

	t.Ticks.Color = gray(60)

	// BMH: light gray panel (#EEEEEE) + white grid → white edges.
	t.Geom.PatchEdgeColor = color.White
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9

	t.Palette = []color.Color{
		hex("348ABD"),
		hex("A60628"),
		hex("7A68A6"),
		hex("467821"),
		hex("D55E00"),
		hex("CC79A7"),
		hex("56B4E9"),
		hex("009E73"),
		hex("F0E442"),
		hex("0072B2"),
	}
	return t
}
