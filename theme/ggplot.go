package theme

import "image/color"

func init() {
	MustRegister(Ggplot, newGgplot)
}

// newGgplot mirrors matplotlib's ggplot.mplstyle.
func newGgplot() Theme {
	t := baseTheme("ggplot")
	bg := hex("E5E5E5")
	axisLabel := hex("555555")

	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 10, Color: axisLabel}
	t.Elements["plot.title"] = ElementText{Size: 14, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 11}
	t.Elements["plot.background"] = ElementRect{Fill: color.White}
	t.Elements["panel.background"] = ElementRect{Fill: bg}
	t.Elements["panel.border"] = ElementRect{Color: color.White, Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: color.White, Size: 1}
	t.Elements["panel.grid.minor"] = ElementLine{Color: hexA("FFFFFF", 160), Size: 0.5}
	t.Elements["axis.title"] = ElementText{Size: 11}
	t.Elements["axis.ticks"] = ElementLine{Color: axisLabel, Size: 1}
	t.Elements["legend.text"] = ElementText{Size: 10}
	t.Elements["annotation.text"] = ElementText{Size: 10}

	t.Geom.PatchEdgeColor = color.White
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	t.Palette = []color.Color{
		hex("E24A33"), hex("348ABD"), hex("988ED5"), hex("777777"),
		hex("FBC15E"), hex("8EBA42"), hex("FFB5B8"),
	}

	return t
}
