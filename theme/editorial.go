package theme

import "image/color"

func init() {
	MustRegister(Tufte, newTufte)
	MustRegister(Academic, newAcademic)
	MustRegister(Newsroom, newNewsroom)
	MustRegister(Editorial, newEditorial)
	MustRegister(Monochrome, newMonochrome)
}

func newTufte() Theme {
	return Theme{
		Name:    "tufte",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("111111"), PatchEdgeWidth: 0.3, PatchAlpha: 0.9},
		Palette: []color.Color{hex("000000"), hex("4D4D4D"), hex("737373"), hex("969696"), hex("BDBDBD"), hex("D9D9D9")},
		Elements: map[string]Element{
			"text":             ElementText{Family: "serif", Size: 11, Color: hex("111111")},
			"line":             ElementLine{Color: hex("111111"), Size: 0.8},
			"rect":             ElementRect{Fill: color.White},
			"plot.title":       ElementText{Size: 15},
			"plot.subtitle":    ElementText{Size: 10, Color: hex("555555")},
			"plot.background":  ElementRect{Fill: color.White},
			"panel.background": ElementRect{Fill: color.White},
			"panel.border":     ElementBlank{},
			"panel.grid.major": ElementBlank{},
			"panel.grid.minor": ElementBlank{},
			"axis.line":        ElementLine{Color: hex("111111"), Size: 0.8},
			"axis.ticks":       ElementLine{Color: hex("111111"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("333333"), Size: 9},
			"axis.title":       ElementText{Color: hex("222222"), Size: 10},
			"legend.text":      ElementText{Color: hex("333333"), Size: 9},
			"strip.text":       ElementText{Size: 10},
		},
	}
}

func newAcademic() Theme {
	return Theme{
		Name:    "academic",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("222222"), PatchEdgeWidth: 0.5, PatchAlpha: 1.0},
		Palette: []color.Color{
			hex("1B365D"), hex("8C1515"), hex("00629B"), hex("007C92"),
			hex("6C6F70"), hex("B26F16"), hex("53284F"), hex("2E7D32"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "serif", Size: 11, Color: hex("111111")},
			"line":             ElementLine{Color: hex("222222"), Size: 0.8},
			"rect":             ElementRect{Fill: color.White},
			"plot.title":       ElementText{Size: 14, Bold: true},
			"plot.subtitle":    ElementText{Size: 10, Color: hex("444444")},
			"plot.background":  ElementRect{Fill: color.White},
			"panel.background": ElementRect{Fill: color.White},
			"panel.border":     ElementRect{Color: hex("222222"), Size: 0.8},
			"panel.grid.major": ElementLine{Color: hex("E5E5E5"), Size: 0.6},
			"panel.grid.minor": ElementBlank{},
			"axis.line":        ElementLine{Color: hex("222222"), Size: 0.8},
			"axis.ticks":       ElementLine{Color: hex("222222"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("222222"), Size: 9},
			"axis.title":       ElementText{Size: 10},
			"legend.text":      ElementText{Color: hex("222222"), Size: 9},
			"strip.background": ElementRect{Fill: hex("F5F5F5"), Color: hex("CCCCCC")},
			"strip.text":       ElementText{Size: 10, Bold: true},
		},
	}
}

func newNewsroom() Theme {
	t := baseTheme("newsroom")
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: hex("1A1A2E")}
	t.Elements["plot.title"] = ElementText{Size: 18, Bold: true, Color: hex("1A1A2E")}
	t.Elements["plot.subtitle"] = ElementText{Size: 12, Color: hex("6B6B80")}
	t.Elements["plot.background"] = ElementRect{Fill: hex("FAFAFA")}
	t.Elements["panel.background"] = ElementRect{Fill: color.White}
	t.Elements["panel.border"] = ElementBlank{}
	t.Elements["panel.grid.major"] = ElementLine{Color: hex("EBEBEB"), Size: 0.8}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.line"] = ElementLine{Color: hex("1A1A2E"), Size: 1}
	t.Elements["axis.ticks"] = ElementBlank{}
	t.Elements["axis.text"] = ElementText{Color: hex("6B6B80"), Size: 10}
	t.Elements["axis.title"] = ElementText{Color: hex("3D3D56"), Size: 11}
	t.Spacing.TickLength = 0
	t.Geom.PatchEdgeColor = color.White
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0
	t.Palette = []color.Color{
		hex("E63946"), hex("1D3557"), hex("457B9D"), hex("F4A261"),
		hex("2A9D8F"), hex("264653"), hex("E76F51"), hex("A8DADC"),
	}

	return t
}

func newEditorial() Theme {
	t := baseTheme("editorial")
	t.Elements["text"] = ElementText{Family: "serif", Size: 11, Color: hex("2D2D2D")}
	t.Elements["plot.title"] = ElementText{Size: 16, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 11, Color: hex("757575"), Italic: true}
	t.Elements["panel.background"] = ElementRect{Fill: hex("FEFCF8")}
	t.Elements["panel.border"] = ElementRect{Color: hex("C4B998"), Size: 0.5}
	t.Elements["panel.grid.major"] = ElementLine{Color: hex("E8E0D0"), Size: 0.5}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.ticks"] = ElementLine{Color: hex("8C8272"), Size: 0.8}
	t.Elements["axis.text"] = ElementText{Color: hex("5C5344"), Size: 9}
	t.Elements["axis.title"] = ElementText{Color: hex("3D352C"), Size: 10}
	t.Geom.PatchEdgeColor = hex("C4B998")
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9
	t.Palette = []color.Color{
		hex("8B4513"), hex("2F4F4F"), hex("8B0000"), hex("006400"),
		hex("4B0082"), hex("CD853F"), hex("556B2F"), hex("800020"),
	}

	return t
}

func newMonochrome() Theme {
	t := baseTheme("monochrome")
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: color.Black}
	t.Elements["plot.title"] = ElementText{Size: 14, Bold: true}
	t.Elements["panel.border"] = ElementRect{Color: color.Black, Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: gray(210), Size: 0.5}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.ticks"] = ElementLine{Color: color.Black, Size: 1}
	t.Geom.PatchEdgeColor = color.Black
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0
	t.Palette = []color.Color{
		color.Black, gray(77), gray(128), gray(179), gray(204), gray(230),
	}

	return t
}
