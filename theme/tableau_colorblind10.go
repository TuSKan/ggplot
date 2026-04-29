package theme

import "image/color"

func init() { MustRegister(TableauColorblind10, newTableauColorblind10) }

// newTableauColorblind10 mirrors matplotlib's
// tableau-colorblind10.mplstyle: white canvas with Tableau's 10-color
// colorblind-safe cycle.
//
// Source: matplotlib/lib/matplotlib/mpl-data/stylelib/tableau-colorblind10.mplstyle
func newTableauColorblind10() Theme {
	t := baseTheme("tableau_colorblind10")

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

	// tableau_colorblind10: white panel + gray grid → gray edge matches grid.
	t.Geom.PatchEdgeColor = gray(220)
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	t.Palette = []color.Color{
		hex("006BA4"),
		hex("FF800E"),
		hex("ABABAB"),
		hex("595959"),
		hex("5F9ED1"),
		hex("C85200"),
		hex("898989"),
		hex("A2C8EC"),
		hex("FFBC79"),
		hex("CFCFCF"),
	}
	return t
}
