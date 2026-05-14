package theme

import "image/color"

func init() {
	MustRegister(GitHubLight, newGitHubLight)
	MustRegister(GitHubDark, newGitHubDark)
	MustRegister(Nord, newNord)
	MustRegister(Dracula, newDracula)
	MustRegister(GruvboxLight, newGruvboxLight)
	MustRegister(GruvboxDark, newGruvboxDark)
}

func newGitHubLight() Theme {
	t := baseTheme("github_light")
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: hex("24292F")}
	t.Elements["plot.title"] = ElementText{Size: 15, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 11, Color: hex("57606A")}
	t.Elements["plot.background"] = ElementRect{Fill: hex("FFFFFF")}
	t.Elements["panel.background"] = ElementRect{Fill: hex("FFFFFF")}
	t.Elements["panel.border"] = ElementRect{Color: hex("D0D7DE"), Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: hex("D8DEE4"), Size: 0.5}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.ticks"] = ElementLine{Color: hex("D0D7DE"), Size: 0.8}
	t.Elements["axis.text"] = ElementText{Color: hex("57606A"), Size: 10}
	t.Elements["axis.title"] = ElementText{Color: hex("24292F"), Size: 11}
	t.Geom.PatchEdgeColor = hex("D0D7DE")
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9
	t.Palette = []color.Color{
		hex("0969DA"), hex("CF222E"), hex("1A7F37"), hex("8250DF"),
		hex("BF8700"), hex("0550AE"), hex("E16F24"), hex("6E7781"),
	}

	return t
}

func newGitHubDark() Theme {
	bg := hex("0D1117")

	return Theme{
		Name:    "github_dark",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("30363D"), PatchEdgeWidth: 0.5, PatchAlpha: 0.9},
		Palette: []color.Color{
			hex("58A6FF"), hex("F85149"), hex("3FB950"), hex("D2A8FF"),
			hex("D29922"), hex("79C0FF"), hex("FFA657"), hex("8B949E"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "sans-serif", Size: 11, Color: hex("C9D1D9")},
			"line":             ElementLine{Color: hex("30363D"), Size: 0.8},
			"rect":             ElementRect{Fill: bg},
			"plot.title":       ElementText{Size: 15, Color: hex("F0F6FC"), Bold: true},
			"plot.subtitle":    ElementText{Size: 11, Color: hex("8B949E")},
			"plot.background":  ElementRect{Fill: bg},
			"panel.background": ElementRect{Fill: bg},
			"panel.border":     ElementRect{Color: hex("30363D"), Size: 1},
			"panel.grid.major": ElementLine{Color: hex("21262D"), Size: 0.5},
			"panel.grid.minor": ElementBlank{},
			"axis.ticks":       ElementLine{Color: hex("30363D"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("8B949E"), Size: 10},
			"axis.title":       ElementText{Color: hex("C9D1D9"), Size: 11},
			"legend.text":      ElementText{Color: hex("8B949E"), Size: 10},
			"strip.background": ElementRect{Fill: hex("161B22")},
			"strip.text":       ElementText{Color: hex("C9D1D9"), Size: 10},
		},
	}
}

func newNord() Theme {
	bg := hex("2E3440")

	return Theme{
		Name:    "nord",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("3B4252"), PatchEdgeWidth: 0.5, PatchAlpha: 0.85},
		Palette: []color.Color{
			hex("5E81AC"), hex("D08770"), hex("A3BE8C"), hex("BF616A"), hex("B48EAD"),
			hex("88C0D0"), hex("EBCB8B"), hex("81A1C1"), hex("8FBCBB"), hex("4C566A"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "sans-serif", Size: 11, Color: hex("D8DEE9")},
			"line":             ElementLine{Color: hex("4C566A"), Size: 0.8},
			"rect":             ElementRect{Fill: bg},
			"plot.title":       ElementText{Size: 15, Color: hex("ECEFF4"), Bold: true},
			"plot.subtitle":    ElementText{Size: 11, Color: hex("D8DEE9")},
			"plot.background":  ElementRect{Fill: bg},
			"panel.background": ElementRect{Fill: bg},
			"panel.border":     ElementBlank{},
			"panel.grid.major": ElementLine{Color: hex("3B4252"), Size: 0.8},
			"panel.grid.minor": ElementLine{Color: hex("343A46"), Size: 0.5},
			"axis.ticks":       ElementLine{Color: hex("4C566A"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("D8DEE9"), Size: 10},
			"axis.title":       ElementText{Color: hex("E5E9F0"), Size: 11},
			"legend.text":      ElementText{Color: hex("D8DEE9"), Size: 10},
			"strip.background": ElementRect{Fill: hex("3B4252")},
			"strip.text":       ElementText{Color: hex("ECEFF4"), Size: 10, Bold: true},
		},
	}
}

func newDracula() Theme {
	bg := hex("282A36")

	return Theme{
		Name:    "dracula",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("44475A"), PatchEdgeWidth: 0.5, PatchAlpha: 0.85},
		Palette: []color.Color{
			hex("BD93F9"), hex("FF79C6"), hex("50FA7B"), hex("FFB86C"),
			hex("8BE9FD"), hex("FF5555"), hex("F1FA8C"), hex("6272A4"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "sans-serif", Size: 11, Color: hex("F8F8F2")},
			"line":             ElementLine{Color: hex("6272A4"), Size: 0.8},
			"rect":             ElementRect{Fill: bg},
			"plot.title":       ElementText{Size: 15, Bold: true},
			"plot.subtitle":    ElementText{Size: 11, Color: hex("C6C6C2")},
			"plot.background":  ElementRect{Fill: bg},
			"panel.background": ElementRect{Fill: bg},
			"panel.border":     ElementBlank{},
			"panel.grid.major": ElementLine{Color: hex("44475A"), Size: 0.8},
			"panel.grid.minor": ElementLine{Color: hex("343746"), Size: 0.5},
			"axis.ticks":       ElementLine{Color: hex("6272A4"), Size: 0.8},
			"axis.text":        ElementText{Size: 10},
			"axis.title":       ElementText{Size: 11},
			"legend.text":      ElementText{Size: 10},
			"strip.background": ElementRect{Fill: hex("44475A")},
			"strip.text":       ElementText{Size: 10, Bold: true},
		},
	}
}

func newGruvboxLight() Theme {
	t := baseTheme("gruvbox_light")
	bg := hex("FBF1C7")
	fg := hex("3C3836")

	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: fg}
	t.Elements["plot.title"] = ElementText{Size: 15, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 11, Color: hex("665C54")}
	t.Elements["plot.background"] = ElementRect{Fill: bg}
	t.Elements["panel.background"] = ElementRect{Fill: hex("EBDBB2")}
	t.Elements["panel.border"] = ElementRect{Color: hex("928374"), Size: 0.8}
	t.Elements["panel.grid.major"] = ElementLine{Color: hex("D5C4A1"), Size: 0.6}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.ticks"] = ElementLine{Color: hex("928374"), Size: 0.8}
	t.Elements["axis.text"] = ElementText{Color: hex("504945"), Size: 10}
	t.Elements["axis.title"] = ElementText{Color: fg, Size: 11}
	t.Geom.PatchEdgeColor = hex("928374")
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9
	t.Palette = []color.Color{
		hex("CC241D"), hex("98971A"), hex("D79921"), hex("458588"),
		hex("B16286"), hex("689D6A"), hex("D65D0E"), hex("928374"),
	}

	return t
}

func newGruvboxDark() Theme {
	bg := hex("282828")
	fg := hex("EBDBB2")

	return Theme{
		Name:    "gruvbox_dark",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("504945"), PatchEdgeWidth: 0.5, PatchAlpha: 0.85},
		Palette: []color.Color{
			hex("FB4934"), hex("B8BB26"), hex("FABD2F"), hex("83A598"),
			hex("D3869B"), hex("8EC07C"), hex("FE8019"), hex("A89984"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "sans-serif", Size: 11, Color: fg},
			"line":             ElementLine{Color: hex("665C54"), Size: 0.8},
			"rect":             ElementRect{Fill: bg},
			"plot.title":       ElementText{Size: 15, Bold: true},
			"plot.subtitle":    ElementText{Size: 11, Color: hex("A89984")},
			"plot.background":  ElementRect{Fill: bg},
			"panel.background": ElementRect{Fill: hex("1D2021")},
			"panel.border":     ElementBlank{},
			"panel.grid.major": ElementLine{Color: hex("3C3836"), Size: 0.8},
			"panel.grid.minor": ElementLine{Color: hex("32302F"), Size: 0.5},
			"axis.ticks":       ElementLine{Color: hex("665C54"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("BDAE93"), Size: 10},
			"axis.title":       ElementText{Color: fg, Size: 11},
			"legend.text":      ElementText{Color: hex("BDAE93"), Size: 10},
			"strip.background": ElementRect{Fill: hex("3C3836")},
			"strip.text":       ElementText{Size: 10, Bold: true},
		},
	}
}
