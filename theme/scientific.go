package theme

import "image/color"

func init() {
	MustRegister(AstronomyDark, newAstronomyDark)
	MustRegister(NASA, newNASA)
	MustRegister(Ocean, newOcean)
	MustRegister(Earth, newEarth)
	MustRegister(Forest, newForest)
	MustRegister(Desert, newDesert)
}

func newAstronomyDark() Theme {
	bg := hex("020617")

	return Theme{
		Name:    "astronomy_dark",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("1E293B"), PatchEdgeWidth: 0.5, PatchAlpha: 0.85},
		Palette: []color.Color{
			hex("7DD3FC"), hex("FDE68A"), hex("FCA5A5"), hex("C4B5FD"), hex("86EFAC"),
			hex("FDBA74"), hex("F0ABFC"), hex("A5B4FC"), hex("67E8F9"), hex("D1D5DB"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "sans-serif", Size: 11, Color: hex("E5E7EB")},
			"line":             ElementLine{Color: hex("334155"), Size: 0.8},
			"rect":             ElementRect{Fill: bg},
			"plot.title":       ElementText{Size: 15, Color: hex("F8FAFC")},
			"plot.subtitle":    ElementText{Size: 11, Color: hex("94A3B8")},
			"plot.background":  ElementRect{Fill: bg},
			"panel.background": ElementRect{Fill: bg},
			"panel.border":     ElementRect{Color: hex("1E293B"), Size: 1},
			"panel.grid.major": ElementLine{Color: hex("1E293B"), Size: 0.8},
			"panel.grid.minor": ElementLine{Color: hex("0F172A"), Size: 0.5},
			"axis.line":        ElementLine{Color: hex("334155"), Size: 0.8},
			"axis.ticks":       ElementLine{Color: hex("475569"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("CBD5E1"), Size: 10},
			"axis.title":       ElementText{Color: hex("E2E8F0"), Size: 11},
			"legend.text":      ElementText{Color: hex("CBD5E1"), Size: 10},
			"strip.background": ElementRect{Fill: hex("0F172A"), Color: hex("1E293B")},
			"strip.text":       ElementText{Color: hex("E2E8F0"), Size: 10, Bold: true},
		},
	}
}

func newNASA() Theme {
	t := baseTheme("nasa")
	nasaBlue := hex("0B3D91")

	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: hex("1F2937")}
	t.Elements["plot.title"] = ElementText{Size: 16, Color: nasaBlue, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 11, Color: hex("4B5563")}
	t.Elements["panel.border"] = ElementBlank{}
	t.Elements["panel.grid.major"] = ElementLine{Color: hex("E5E7EB"), Size: 0.8}
	t.Elements["panel.grid.minor"] = ElementLine{Color: hex("F3F4F6"), Size: 0.5}
	t.Elements["axis.line"] = ElementLine{Color: nasaBlue, Size: 1}
	t.Elements["axis.ticks"] = ElementLine{Color: nasaBlue, Size: 0.8}
	t.Elements["axis.text"] = ElementText{Color: hex("374151"), Size: 10}
	t.Elements["strip.background"] = ElementRect{Fill: hex("EFF6FF"), Color: hex("DBEAFE")}
	t.Elements["strip.text"] = ElementText{Color: nasaBlue, Size: 10, Bold: true}
	t.Geom.PatchEdgeColor = nasaBlue
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9
	t.Palette = []color.Color{
		nasaBlue, hex("FC3D21"), hex("6CB4EE"), hex("FDB515"), hex("00A676"),
		hex("7B2CBF"), hex("E63946"), hex("457B9D"), hex("2A9D8F"), hex("6C757D"),
	}

	return t
}

func newOcean() Theme {
	t := baseTheme("ocean")
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: hex("1E3A5F")}
	t.Elements["plot.title"] = ElementText{Size: 15, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 11, Color: hex("5B7FA0")}
	t.Elements["plot.background"] = ElementRect{Fill: hex("F0F7FF")}
	t.Elements["panel.background"] = ElementRect{Fill: color.White}
	t.Elements["panel.border"] = ElementRect{Color: hex("A0C4E8"), Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: hex("D4E8F7"), Size: 0.6}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.ticks"] = ElementLine{Color: hex("7FAED0"), Size: 0.8}
	t.Elements["axis.text"] = ElementText{Color: hex("3D6E8E"), Size: 10}
	t.Geom.PatchEdgeColor = hex("A0C4E8")
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.85
	t.Palette = []color.Color{
		hex("023E8A"), hex("0077B6"), hex("00B4D8"), hex("48CAE4"),
		hex("90E0EF"), hex("0096C7"), hex("003459"), hex("5E60CE"),
	}

	return t
}

// natureTheme builds a tinted-panel nature theme. Each variant passes
// its own palette and tint colors to avoid structural duplication.
type natureTint struct {
	name                           string
	text, subtitle, axisText, tick color.Color
	plotBg, panelBg                color.Color
	border, grid                   color.Color
	palette                        []color.Color
}

func natureTheme(nt natureTint) Theme {
	t := baseTheme(nt.name)
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: nt.text}
	t.Elements["plot.title"] = ElementText{Size: 15, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 11, Color: nt.subtitle}
	t.Elements["plot.background"] = ElementRect{Fill: nt.plotBg}
	t.Elements["panel.background"] = ElementRect{Fill: nt.panelBg}
	t.Elements["panel.border"] = ElementRect{Color: nt.border, Size: 0.8}
	t.Elements["panel.grid.major"] = ElementLine{Color: nt.grid, Size: 0.5}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.ticks"] = ElementLine{Color: nt.tick, Size: 0.8}
	t.Elements["axis.text"] = ElementText{Color: nt.axisText, Size: 10}
	t.Geom.PatchEdgeColor = nt.border
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9
	t.Palette = nt.palette

	return t
}

func newEarth() Theme {
	return natureTheme(natureTint{
		name: "earth", text: hex("3D3027"), subtitle: hex("7D6E5D"),
		axisText: hex("6B5D4E"), tick: hex("A89882"),
		plotBg: hex("F5F0EB"), panelBg: hex("FBF8F5"),
		border: hex("C4B8A8"), grid: hex("E5DDD3"),
		palette: []color.Color{
			hex("5B7553"), hex("8B4513"), hex("DAA520"), hex("2E6B4F"),
			hex("A0522D"), hex("6B8E23"), hex("CD853F"), hex("556B2F"),
		},
	})
}

func newForest() Theme {
	return natureTheme(natureTint{
		name: "forest", text: hex("2D3B2D"), subtitle: hex("5E7A5E"),
		axisText: hex("4A6640"), tick: hex("7FA870"),
		plotBg: hex("F2F5F0"), panelBg: hex("FAFCF8"),
		border: hex("A8C4A0"), grid: hex("D8E8D0"),
		palette: []color.Color{
			hex("2D6A4F"), hex("40916C"), hex("52B788"), hex("95D5B2"),
			hex("1B4332"), hex("74C69D"), hex("B7E4C7"), hex("8B6914"),
		},
	})
}

func newDesert() Theme {
	return natureTheme(natureTint{
		name: "desert", text: hex("4A3728"), subtitle: hex("8B7355"),
		axisText: hex("7A6240"), tick: hex("C19A6B"),
		plotBg: hex("FDF5E6"), panelBg: hex("FFF8EE"),
		border: hex("D2B48C"), grid: hex("EDE0CC"),
		palette: []color.Color{
			hex("C2452D"), hex("E08E45"), hex("DEB841"), hex("A67B5B"),
			hex("6B3A2A"), hex("E6A756"), hex("8B6914"), hex("C19552"),
		},
	})
}
