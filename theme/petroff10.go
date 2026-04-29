package theme

import "image/color"

func init() { MustRegister(Petroff10, newPetroff10) }

// newPetroff10 mirrors matplotlib's petroff10.mplstyle, introduced in
// matplotlib 3.10. The palette was designed by Matthew Petroff for high
// perceptual distinctiveness across the full 10-color cycle.
//
// Source: matplotlib/lib/matplotlib/mpl-data/stylelib/petroff10.mplstyle
func newPetroff10() Theme {
	t := baseTheme("petroff10")

	t.Background = color.White
	t.Panel.Background = color.White
	t.Panel.Border = gray(180)
	t.Panel.BorderWidth = 1

	t.Grid.MajorColor = gray(220)
	t.Grid.MajorWidth = 0.5
	t.Grid.MinorColor = gray(240)
	t.Grid.MinorWidth = 0.3
	t.Grid.DashPattern = nil

	t.Ticks.Color = gray(60)

	// petroff10: white panel + light gray grid → matching gray edge.
	t.Geom.PatchEdgeColor = gray(220)
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	// Verbatim from petroff10.mplstyle axes.prop_cycle.
	t.Palette = []color.Color{
		hex("3f90da"), // blue
		hex("ffa90e"), // orange
		hex("bd1f01"), // red
		hex("94a4a2"), // gray-teal
		hex("832db6"), // purple
		hex("a96b59"), // brown
		hex("e76300"), // burnt orange
		hex("b9ac70"), // khaki
		hex("717581"), // slate
		hex("92dadd"), // sky cyan
	}
	return t
}
