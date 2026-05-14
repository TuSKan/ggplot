package theme

import "image/color"

func init() {
	MustRegister(ObservableDark, newObservableDark)
	MustRegister(Dashboard, newDashboard)
	MustRegister(Quartz, newQuartz)
	MustRegister(Air, newAir)
	MustRegister(Ink, newInk)
}

func newObservableDark() Theme {
	bg := hex("111827")

	return Theme{
		Name:    "observable_dark",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("374151"), PatchEdgeWidth: 0.5, PatchAlpha: 0.85},
		Palette: []color.Color{
			hex("4269D0"), hex("EFB118"), hex("FF725C"), hex("6CC5B0"), hex("3CA951"),
			hex("FF8AB7"), hex("A463F2"), hex("97BBF5"), hex("9C6B4E"), hex("9498A0"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "sans-serif", Size: 11, Color: hex("E5E7EB")},
			"line":             ElementLine{Color: hex("374151"), Size: 1},
			"rect":             ElementRect{Fill: bg},
			"plot.title":       ElementText{Size: 15, Color: hex("F9FAFB")},
			"plot.subtitle":    ElementText{Size: 11, Color: hex("9CA3AF")},
			"plot.background":  ElementRect{Fill: bg},
			"panel.background": ElementRect{Fill: bg},
			"panel.border":     ElementBlank{},
			"panel.grid.major": ElementLine{Color: hex("374151"), Size: 0.8},
			"panel.grid.minor": ElementLine{Color: hex("1F2937"), Size: 0.6},
			"axis.ticks":       ElementLine{Color: hex("6B7280"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("D1D5DB"), Size: 10},
			"axis.title":       ElementText{Color: hex("D1D5DB"), Size: 11},
			"legend.text":      ElementText{Color: hex("D1D5DB"), Size: 10},
			"strip.background": ElementRect{Fill: hex("1F2937")},
			"strip.text":       ElementText{Color: hex("F9FAFB"), Size: 10},
		},
	}
}

func newDashboard() Theme {
	return Theme{
		Name:    "dashboard",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("E5E7EB"), PatchEdgeWidth: 0.5, PatchAlpha: 0.9},
		Palette: []color.Color{
			hex("2563EB"), hex("F59E0B"), hex("10B981"), hex("EF4444"), hex("8B5CF6"),
			hex("06B6D4"), hex("84CC16"), hex("F97316"), hex("EC4899"), hex("64748B"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "sans-serif", Size: 11, Color: hex("1F2937")},
			"line":             ElementLine{Color: hex("CBD5E1"), Size: 0.8},
			"rect":             ElementRect{Fill: hex("FFFFFF")},
			"plot.title":       ElementText{Size: 16, Color: hex("111827"), Bold: true},
			"plot.subtitle":    ElementText{Size: 11, Color: hex("64748B")},
			"plot.background":  ElementRect{Fill: hex("F8FAFC")},
			"panel.background": ElementRect{Fill: color.White, Color: hex("E5E7EB"), Size: 1},
			"panel.border":     ElementRect{Color: hex("E5E7EB"), Size: 1},
			"panel.grid.major": ElementLine{Color: hex("E5E7EB"), Size: 0.8},
			"panel.grid.minor": ElementLine{Color: hex("F1F5F9"), Size: 0.5},
			"axis.ticks":       ElementLine{Color: hex("CBD5E1"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("475569"), Size: 10},
			"axis.title":       ElementText{Color: hex("475569"), Size: 11},
			"legend.text":      ElementText{Color: hex("334155"), Size: 10},
			"strip.background": ElementRect{Fill: hex("F1F5F9"), Color: hex("E2E8F0")},
			"strip.text":       ElementText{Color: hex("334155"), Size: 10, Bold: true},
		},
	}
}

func newQuartz() Theme {
	t := baseTheme("quartz")
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: hex("1C1C1E")}
	t.Elements["plot.title"] = ElementText{Size: 16, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 11, Color: hex("8E8E93")}
	t.Elements["plot.background"] = ElementRect{Fill: hex("F2F2F7")}
	t.Elements["panel.background"] = ElementRect{Fill: color.White}
	t.Elements["panel.border"] = ElementRect{Color: hex("D1D1D6"), Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: hex("E5E5EA"), Size: 0.5}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.ticks"] = ElementLine{Color: hex("C7C7CC"), Size: 0.8}
	t.Elements["axis.text"] = ElementText{Color: hex("636366"), Size: 10}
	t.Elements["axis.title"] = ElementText{Color: hex("48484A"), Size: 11}
	t.Geom.PatchEdgeColor = hex("D1D1D6")
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9
	t.Palette = []color.Color{
		hex("007AFF"), hex("FF9500"), hex("34C759"), hex("FF3B30"), hex("AF52DE"),
		hex("5AC8FA"), hex("FF2D55"), hex("FFCC00"), hex("5856D6"), hex("FF6482"),
	}

	return t
}

func newAir() Theme {
	t := baseTheme("air")
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: hex("374151")}
	t.Elements["plot.title"] = ElementText{Size: 15, Color: hex("111827")}
	t.Elements["plot.subtitle"] = ElementText{Size: 11, Color: hex("9CA3AF")}
	t.Elements["plot.background"] = ElementRect{Fill: color.White}
	t.Elements["panel.background"] = ElementRect{Fill: color.White}
	t.Elements["panel.border"] = ElementBlank{}
	t.Elements["panel.grid.major"] = ElementLine{Color: hex("F3F4F6"), Size: 0.8}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.line"] = ElementLine{Color: hex("E5E7EB"), Size: 0.8}
	t.Elements["axis.ticks"] = ElementBlank{}
	t.Elements["axis.text"] = ElementText{Color: hex("9CA3AF"), Size: 10}
	t.Elements["axis.title"] = ElementText{Color: hex("6B7280"), Size: 11}
	t.Spacing.TickLength = 0
	t.Geom.PatchEdgeColor = color.White
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.85
	t.Palette = []color.Color{
		hex("6366F1"), hex("EC4899"), hex("14B8A6"), hex("F59E0B"), hex("8B5CF6"),
		hex("06B6D4"), hex("F43F5E"), hex("22C55E"),
	}

	return t
}

func newInk() Theme {
	bg := hex("0F172A")

	return Theme{
		Name:    "ink",
		Spacing: Spacing{MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10, PanelSpacing: 10, TickLength: 5},
		Geom:    GeomDefaults{PatchEdgeColor: hex("1E293B"), PatchEdgeWidth: 0.5, PatchAlpha: 0.9},
		Palette: []color.Color{
			hex("38BDF8"), hex("FB923C"), hex("A78BFA"), hex("4ADE80"), hex("F472B6"),
			hex("FACC15"), hex("2DD4BF"), hex("F87171"),
		},
		Elements: map[string]Element{
			"text":             ElementText{Family: "sans-serif", Size: 11, Color: hex("CBD5E1")},
			"line":             ElementLine{Color: hex("334155"), Size: 0.8},
			"rect":             ElementRect{Fill: bg},
			"plot.title":       ElementText{Size: 15, Color: hex("F1F5F9"), Bold: true},
			"plot.subtitle":    ElementText{Size: 11, Color: hex("64748B")},
			"plot.background":  ElementRect{Fill: bg},
			"panel.background": ElementRect{Fill: bg},
			"panel.border":     ElementBlank{},
			"panel.grid.major": ElementLine{Color: hex("1E293B"), Size: 0.6},
			"panel.grid.minor": ElementBlank{},
			"axis.ticks":       ElementLine{Color: hex("475569"), Size: 0.8},
			"axis.text":        ElementText{Color: hex("94A3B8"), Size: 10},
			"axis.title":       ElementText{Color: hex("CBD5E1"), Size: 11},
			"legend.text":      ElementText{Color: hex("94A3B8"), Size: 10},
			"strip.background": ElementRect{Fill: hex("1E293B")},
			"strip.text":       ElementText{Color: hex("E2E8F0"), Size: 10},
		},
	}
}
