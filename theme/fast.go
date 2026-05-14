package theme

func init() { MustRegister(Fast, newFast) }

// newFast mirrors matplotlib's fast.mplstyle.
func newFast() Theme {
	t := baseTheme("fast")
	t.Elements["panel.border"] = ElementRect{Color: gray(180), Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: gray(204), Size: 0.8}
	t.Elements["panel.grid.minor"] = ElementLine{Color: gray(230), Size: 0.4}
	t.Elements["axis.ticks"] = ElementLine{Color: gray(60), Size: 1}

	t.Geom.PatchEdgeColor = gray(204)
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	return t
}
