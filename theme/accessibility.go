package theme

import "image/color"

func init() {
	MustRegister(HighContrast, newHighContrast)
	MustRegister(OkabeIto, newOkabeIto)
	MustRegister(Viridis, newViridis)
	MustRegister(Cividis, newCividis)
}

func newHighContrast() Theme {
	return Theme{
		Name:    "high_contrast",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 6},
		Geom:    GeomDefaults{PatchEdgeColor: color.Black, PatchEdgeWidth: 1, PatchAlpha: 1.0},
		Palette: []color.Color{
			color.Black, hex("E69F00"), hex("56B4E9"), hex("009E73"),
			hex("F0E442"), hex("0072B2"), hex("D55E00"), hex("CC79A7"),
		},
		Elements: map[string]Element{
			"text":              ElementText{Family: "sans-serif", Size: 12, Color: color.Black},
			"line":              ElementLine{Color: color.Black, Size: 1.2},
			"rect":              ElementRect{Fill: color.White, Color: color.Black, Size: 1},
			"plot.title":        ElementText{Size: 16, Bold: true},
			"plot.subtitle":     ElementText{Size: 12},
			"plot.background":   ElementRect{Fill: color.White},
			"panel.background":  ElementRect{Fill: color.White, Color: color.Black, Size: 1},
			"panel.border":      ElementRect{Color: color.Black, Size: 1.2},
			"panel.grid.major":  ElementLine{Color: hex("BDBDBD"), Size: 0.8},
			"panel.grid.minor":  ElementBlank{},
			"axis.line":         ElementLine{Color: color.Black, Size: 1.2},
			"axis.ticks":        ElementLine{Color: color.Black, Size: 1.2},
			"axis.text":         ElementText{Size: 11},
			"axis.title":        ElementText{Size: 12, Bold: true},
			"legend.background": ElementRect{Fill: color.White, Color: color.Black, Size: 1},
			"legend.text":       ElementText{Size: 11},
			"strip.background":  ElementRect{Fill: color.Black},
			"strip.text":        ElementText{Color: color.White, Size: 11, Bold: true},
		},
	}
}

func newOkabeIto() Theme {
	t := baseTheme("okabe_ito")
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: hex("222222")}
	t.Elements["plot.title"] = ElementText{Size: 15, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 11, Color: hex("666666")}
	t.Elements["panel.border"] = ElementBlank{}
	t.Elements["panel.grid.major"] = ElementLine{Color: hex("DDDDDD"), Size: 0.8}
	t.Elements["panel.grid.minor"] = ElementLine{Color: hex("EEEEEE"), Size: 0.5}
	t.Elements["axis.ticks"] = ElementLine{Color: hex("999999"), Size: 0.8}
	t.Elements["axis.text"] = ElementText{Color: hex("444444"), Size: 10}
	t.Geom.PatchEdgeColor = hex("DDDDDD")
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9
	// Canonical Okabe-Ito order (orange first — avoids black-on-white confusion).
	t.Palette = []color.Color{
		hex("E69F00"), hex("56B4E9"), hex("009E73"), hex("F0E442"),
		hex("0072B2"), hex("D55E00"), hex("CC79A7"), hex("000000"),
	}

	return t
}

// colormapTheme builds a neutral chrome theme whose palette is sampled
// from a perceptually-uniform colormap. Shared by Viridis and Cividis.
func colormapTheme(name string, palette []color.Color) Theme {
	t := baseTheme(name)
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: hex("222222")}
	t.Elements["plot.title"] = ElementText{Size: 15, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 11, Color: hex("666666")}
	t.Elements["panel.border"] = ElementRect{Color: hex("AAAAAA"), Size: 0.8}
	t.Elements["panel.grid.major"] = ElementLine{Color: hex("E0E0E0"), Size: 0.5}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.ticks"] = ElementLine{Color: hex("888888"), Size: 0.8}
	t.Elements["axis.text"] = ElementText{Color: hex("444444"), Size: 10}
	t.Geom.PatchEdgeColor = hex("CCCCCC")
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9
	t.Palette = palette

	return t
}

func newViridis() Theme {
	// 8 evenly-spaced stops from the viridis colormap.
	return colormapTheme("viridis", []color.Color{
		hex("440154"), hex("46327E"), hex("365C8D"), hex("277F8E"),
		hex("1FA187"), hex("4AC16D"), hex("9FDA3A"), hex("FDE725"),
	})
}

func newCividis() Theme {
	// 8 evenly-spaced stops from the cividis colormap (colorblind-safe).
	return colormapTheme("cividis", []color.Color{
		hex("00204D"), hex("1A3A5C"), hex("42526B"), hex("6B6A76"),
		hex("98875C"), hex("C0A53A"), hex("E3C40D"), hex("FFE945"),
	})
}
