package theme

import "image/color"

func init() { MustRegister(DarkBackground, newDarkBackground) }

// newDarkBackground mirrors matplotlib's dark_background.mplstyle.
func newDarkBackground() Theme {
	t := baseTheme("dark_background")
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 10, Color: color.White}
	t.Elements["plot.title"] = ElementText{Size: 14, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 11, Color: gray(220)}
	t.Elements["plot.background"] = ElementRect{Fill: color.Black}
	t.Elements["panel.background"] = ElementRect{Fill: color.Black}
	t.Elements["panel.border"] = ElementRect{Color: color.White, Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: hexA("FFFFFF", 60), Size: 0.5, Linetype: []float64{4, 4}}
	t.Elements["panel.grid.minor"] = ElementLine{Color: hexA("FFFFFF", 30), Size: 0.3}
	t.Elements["axis.text"] = ElementText{Color: gray(200)}
	t.Elements["axis.ticks"] = ElementLine{Color: color.White, Size: 1}

	t.Geom.PatchEdgeColor = color.White
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9

	t.Palette = []color.Color{
		hex("8DD3C7"), hex("FEFFB3"), hex("BFBBD9"), hex("FA8174"), hex("81B1D2"),
		hex("FDB462"), hex("B3DE69"), hex("BC82BD"), hex("CCEBC4"), hex("FFED6F"),
	}

	return t
}
