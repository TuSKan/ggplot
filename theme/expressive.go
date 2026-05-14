package theme

import "image/color"

func init() {
	MustRegister(Cyberpunk, newCyberpunk)
	MustRegister(Blueprint, newBlueprint)
	MustRegister(Terminal, newTerminal)
	MustRegister(Retro, newRetro)
}

func newCyberpunk() Theme {
	bg := hex("090A1A")

	return Theme{
		Name:    "cyberpunk",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("00F5FF"), PatchEdgeWidth: 0.8, PatchAlpha: 0.85},
		Palette: []color.Color{
			hex("00F5FF"), hex("FF00A8"), hex("FFE600"), hex("39FF14"),
			hex("9D4EDD"), hex("FF6B00"), hex("00FF9F"), hex("FF3131"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "sans-serif", Size: 11, Color: hex("E6F1FF")},
			"line":             ElementLine{Color: hex("00F5FF"), Size: 1},
			"rect":             ElementRect{Fill: bg},
			"plot.title":       ElementText{Size: 16, Color: hex("00F5FF"), Bold: true},
			"plot.subtitle":    ElementText{Size: 11, Color: hex("FF00A8")},
			"plot.background":  ElementRect{Fill: bg},
			"panel.background": ElementRect{Fill: hex("0B1026"), Color: hex("1F2A60"), Size: 1},
			"panel.border":     ElementRect{Color: hex("00F5FF"), Size: 1},
			"panel.grid.major": ElementLine{Color: hex("1F2A60"), Size: 0.8},
			"panel.grid.minor": ElementLine{Color: hex("121936"), Size: 0.5},
			"axis.line":        ElementLine{Color: hex("00F5FF"), Size: 1},
			"axis.ticks":       ElementLine{Color: hex("00F5FF"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("C7D2FE"), Size: 10},
			"axis.title":       ElementText{Color: hex("E6F1FF"), Size: 11},
			"legend.text":      ElementText{Color: hex("E6F1FF"), Size: 10},
			"strip.background": ElementRect{Fill: hex("111827"), Color: hex("00F5FF")},
			"strip.text":       ElementText{Color: hex("00F5FF"), Size: 10, Bold: true},
		},
	}
}

func newBlueprint() Theme {
	bg := hex("0B3D91")

	return Theme{
		Name:    "blueprint",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("93C5FD"), PatchEdgeWidth: 0.5, PatchAlpha: 0.85},
		Palette: []color.Color{
			hex("FFFFFF"), hex("FDE68A"), hex("93C5FD"), hex("A7F3D0"),
			hex("FCA5A5"), hex("DDD6FE"), hex("FDBA74"), hex("67E8F9"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "sans-serif", Size: 11, Color: hex("E0F2FE")},
			"line":             ElementLine{Color: hex("BFDBFE"), Size: 1},
			"rect":             ElementRect{Fill: bg},
			"plot.title":       ElementText{Size: 16, Color: color.White, Bold: true},
			"plot.subtitle":    ElementText{Size: 11, Color: hex("BFDBFE")},
			"plot.background":  ElementRect{Fill: bg},
			"panel.background": ElementRect{Fill: bg, Color: hex("93C5FD"), Size: 1},
			"panel.border":     ElementRect{Color: hex("93C5FD"), Size: 1},
			"panel.grid.major": ElementLine{Color: hex("3B82F6"), Size: 0.7},
			"panel.grid.minor": ElementLine{Color: hex("2563EB"), Size: 0.4},
			"axis.line":        ElementLine{Color: hex("BFDBFE"), Size: 1},
			"axis.ticks":       ElementLine{Color: hex("BFDBFE"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("E0F2FE"), Size: 10},
			"axis.title":       ElementText{Color: color.White, Size: 11},
			"legend.text":      ElementText{Color: hex("E0F2FE"), Size: 10},
			"strip.background": ElementRect{Fill: hex("1D4ED8"), Color: hex("93C5FD")},
			"strip.text":       ElementText{Color: color.White, Size: 10, Bold: true},
		},
	}
}

func newTerminal() Theme {
	bg := hex("1E1E1E")

	return Theme{
		Name:    "terminal",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("4EC9B0"), PatchEdgeWidth: 0.5, PatchAlpha: 0.85},
		Palette: []color.Color{
			hex("4EC9B0"), hex("569CD6"), hex("CE9178"), hex("DCDCAA"),
			hex("C586C0"), hex("9CDCFE"), hex("D7BA7D"), hex("6A9955"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "monospace, sans-serif", Size: 11, Color: hex("CCCCCC")},
			"line":             ElementLine{Color: hex("3C3C3C"), Size: 0.8},
			"rect":             ElementRect{Fill: bg},
			"plot.title":       ElementText{Size: 14, Color: hex("4EC9B0"), Bold: true},
			"plot.subtitle":    ElementText{Size: 11, Color: hex("808080")},
			"plot.background":  ElementRect{Fill: bg},
			"panel.background": ElementRect{Fill: hex("252526")},
			"panel.border":     ElementRect{Color: hex("3C3C3C"), Size: 1},
			"panel.grid.major": ElementLine{Color: hex("2D2D2D"), Size: 0.5},
			"panel.grid.minor": ElementBlank{},
			"axis.ticks":       ElementLine{Color: hex("4EC9B0"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("808080"), Size: 10},
			"axis.title":       ElementText{Color: hex("CCCCCC"), Size: 11},
			"legend.text":      ElementText{Color: hex("808080"), Size: 10},
			"strip.background": ElementRect{Fill: hex("2D2D2D")},
			"strip.text":       ElementText{Color: hex("4EC9B0"), Size: 10},
		},
	}
}

func newRetro() Theme {
	return Theme{
		Name:    "retro",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("5C4033"), PatchEdgeWidth: 0.5, PatchAlpha: 0.9},
		Palette: []color.Color{
			hex("E74C3C"), hex("3498DB"), hex("2ECC71"), hex("F39C12"),
			hex("9B59B6"), hex("1ABC9C"), hex("E67E22"), hex("34495E"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "serif", Size: 11, Color: hex("3C2415")},
			"line":             ElementLine{Color: hex("8B7355"), Size: 0.8},
			"rect":             ElementRect{Fill: hex("FFF8DC")},
			"plot.title":       ElementText{Size: 16, Bold: true},
			"plot.subtitle":    ElementText{Size: 11, Color: hex("8B7355"), Italic: true},
			"plot.background":  ElementRect{Fill: hex("FFF8DC")},
			"panel.background": ElementRect{Fill: hex("FFFDF5")},
			"panel.border":     ElementRect{Color: hex("8B7355"), Size: 1},
			"panel.grid.major": ElementLine{Color: hex("E8DCC8"), Size: 0.5},
			"panel.grid.minor": ElementBlank{},
			"axis.ticks":       ElementLine{Color: hex("8B7355"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("5C4033"), Size: 10},
			"axis.title":       ElementText{Color: hex("3C2415"), Size: 11},
			"legend.text":      ElementText{Color: hex("5C4033"), Size: 10},
			"strip.background": ElementRect{Fill: hex("D4B896"), Color: hex("8B7355")},
			"strip.text":       ElementText{Color: hex("3C2415"), Size: 10, Bold: true},
		},
	}
}
