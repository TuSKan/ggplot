package theme

import "image/color"

func init() { MustRegister(Fast, newFast) }

// newFast mirrors matplotlib's fast.mplstyle. Upstream this style only
// toggles path-simplification rcParams (path.simplify, agg.path.chunksize)
// for performance; it inherits the matplotlib *default* chrome (white panel,
// gray grid), not the ggplot gray panel.
//
// Source: matplotlib/lib/matplotlib/mpl-data/stylelib/fast.mplstyle
func newFast() Theme {
	t := baseTheme("fast")

	// matplotlib default: white panel, light gray grid, black axes text.
	t.Background = color.White
	t.Panel.Background = color.White
	t.Panel.Border = gray(180)
	t.Panel.BorderWidth = 1

	t.Grid.MajorColor = gray(204) // matplotlib default axes.grid color
	t.Grid.MajorWidth = 0.8
	t.Grid.MinorColor = gray(230)
	t.Grid.MinorWidth = 0.4
	t.Grid.DashPattern = nil

	t.Ticks.Color = gray(60)

	// white panel + gray grid → light gray edge.
	t.Geom.PatchEdgeColor = gray(204)
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	return t
}
