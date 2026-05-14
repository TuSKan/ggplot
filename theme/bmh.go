package theme

import "image/color"

func init() { MustRegister(Bmh, newBmh) }

// newBmh mirrors matplotlib's bmh.mplstyle.
func newBmh() Theme {
	t := baseTheme("bmh")
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 10, Color: gray(40)}
	t.Elements["plot.title"] = ElementText{Size: 14, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 11}
	t.Elements["plot.background"] = ElementRect{Fill: color.White}
	t.Elements["panel.background"] = ElementRect{Fill: hex("EEEEEE")}
	t.Elements["panel.border"] = ElementRect{Color: color.White, Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: color.White, Size: 0.8, Linetype: []float64{4, 4}}
	t.Elements["panel.grid.minor"] = ElementLine{Color: hexA("FFFFFF", 160), Size: 0.4}
	t.Elements["axis.text"] = ElementText{Color: gray(60)}
	t.Elements["axis.ticks"] = ElementLine{Color: gray(60), Size: 1}

	t.Geom.PatchEdgeColor = color.White
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9

	t.Palette = []color.Color{
		hex("348ABD"), hex("A60628"), hex("7A68A6"), hex("467821"), hex("D55E00"),
		hex("CC79A7"), hex("56B4E9"), hex("009E73"), hex("F0E442"), hex("0072B2"),
	}

	return t
}
