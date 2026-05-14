package theme

import "image/color"

func init() {
	MustRegister(SeabornDarkgrid, newSeabornDarkgrid)
	MustRegister(Seaborn, newSeabornDarkgrid)
	MustRegister(SeabornWhitegrid, newSeabornWhitegrid)
	MustRegister(SeabornDark, newSeabornDark)
	MustRegister(SeabornWhite, newSeabornWhite)
	MustRegister(SeabornTicks, newSeabornTicks)
	MustRegister(SeabornDeep, newSeabornDeep)
	MustRegister(SeabornMuted, newSeabornMuted)
	MustRegister(SeabornBright, newSeabornBright)
	MustRegister(SeabornColorblind, newSeabornColorblind)
	MustRegister(SeabornPastel, newSeabornPastel)
	MustRegister(SeabornDarkPalette, newSeabornDarkPalette)
	MustRegister(SeabornPaper, newSeabornPaper)
	MustRegister(SeabornNotebook, newSeabornNotebook)
	MustRegister(SeabornTalk, newSeabornTalk)
	MustRegister(SeabornPoster, newSeabornPoster)
}

// seabornBase returns the chrome shared by all seaborn presets.
func seabornBase(name string) Theme {
	tickAndText := gray(38)

	return Theme{
		Name: name,
		Spacing: Spacing{
			MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10,
			PanelSpacing: 10,
		},
		Geom: GeomDefaults{
			PatchEdgeColor: nil,
			PatchEdgeWidth: 0.5,
			PatchAlpha:     0.8,
		},
		Palette: seabornDeepPalette(),
		Elements: map[string]Element{
			"text":       ElementText{Family: "sans-serif", Size: 10, Color: tickAndText},
			"line":       ElementLine{Color: tickAndText, Size: 1},
			"rect":       ElementRect{Fill: color.White},
			"plot.title": ElementText{Size: 12, Bold: true},
			"axis.ticks": ElementLine{Color: tickAndText, Size: 1},
		},
	}
}

func seabornDeepPalette() []color.Color {
	return []color.Color{
		hex("4C72B0"), hex("55A868"), hex("C44E52"),
		hex("8172B2"), hex("CCB974"), hex("64B5CD"),
	}
}

// --- chrome variants ---

func newSeabornDarkgrid() Theme {
	t := seabornBase("seaborn_darkgrid")
	t.Elements["panel.background"] = ElementRect{Fill: hex("EAEAF2")}
	t.Elements["panel.border"] = ElementBlank{}
	t.Elements["panel.grid.major"] = ElementLine{Color: color.White, Size: 1}
	t.Elements["panel.grid.minor"] = ElementLine{Color: hexA("FFFFFF", 160), Size: 0.5}
	t.Geom.PatchEdgeColor = color.White
	t.Geom.PatchAlpha = 0.8

	return t
}

func newSeabornWhitegrid() Theme {
	t := seabornBase("seaborn_whitegrid")
	t.Elements["panel.border"] = ElementRect{Color: gray(204), Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: gray(204), Size: 0.5}
	t.Elements["panel.grid.minor"] = ElementLine{Color: gray(230), Size: 0.3}
	t.Geom.PatchEdgeColor = gray(204)
	t.Geom.PatchAlpha = 0.8

	return t
}

func newSeabornDark() Theme {
	t := seabornBase("seaborn_dark")
	t.Elements["panel.background"] = ElementRect{Fill: hex("EAEAF2")}
	t.Elements["panel.border"] = ElementBlank{}
	t.Elements["panel.grid.major"] = ElementBlank{}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Geom.PatchEdgeColor = color.White
	t.Geom.PatchAlpha = 0.8

	return t
}

func newSeabornWhite() Theme {
	t := seabornBase("seaborn_white")
	t.Elements["panel.border"] = ElementRect{Color: gray(38), Size: 1}
	t.Elements["panel.grid.major"] = ElementBlank{}
	t.Elements["panel.grid.minor"] = ElementBlank{}

	return t
}

func newSeabornTicks() Theme {
	t := newSeabornWhite()
	t.Name = "seaborn_ticks"
	// Ticks point outward and are visible — override with larger size.
	t.Elements["axis.ticks"] = ElementLine{Color: gray(38), Size: 1}

	return t
}

// --- palette variants ---

func newSeabornDeep() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_deep"
	t.Palette = seabornDeepPalette()

	return t
}

func newSeabornMuted() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_muted"
	t.Palette = []color.Color{
		hex("4878CF"), hex("6ACC65"), hex("D65F5F"),
		hex("B47CC7"), hex("C4AD66"), hex("77BEDB"),
	}

	return t
}

func newSeabornBright() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_bright"
	t.Palette = []color.Color{
		hex("003FFF"), hex("03ED3A"), hex("E8000B"),
		hex("8A2BE2"), hex("FFC400"), hex("00D7FF"),
	}

	return t
}

func newSeabornColorblind() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_colorblind"
	t.Palette = []color.Color{
		hex("0072B2"), hex("009E73"), hex("D55E00"),
		hex("CC79A7"), hex("F0E442"), hex("56B4E9"),
	}

	return t
}

func newSeabornPastel() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_pastel"
	t.Palette = []color.Color{
		hex("92C6FF"), hex("97F0AA"), hex("FF9F9A"),
		hex("D0BBFF"), hex("FFFEA3"), hex("B0E0E6"),
	}

	return t
}

func newSeabornDarkPalette() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_dark_palette"
	t.Palette = []color.Color{
		hex("001C7F"), hex("017517"), hex("8C0900"),
		hex("7600A1"), hex("B8860B"), hex("006374"),
	}

	return t
}

// --- font-size variants ---

func scaleSeabornFonts(t Theme, title, label float64) Theme {
	t.Elements["plot.title"] = ElementText{Size: title, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: title - 2}
	t.Elements["axis.title"] = ElementText{Size: label}
	t.Elements["axis.text"] = ElementText{Size: label - 1}
	t.Elements["legend.text"] = ElementText{Size: label - 1}
	t.Elements["annotation.text"] = ElementText{Size: label - 1}

	return t
}

func newSeabornPaper() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_paper"

	return scaleSeabornFonts(t, 11, 8.4)
}

func newSeabornNotebook() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_notebook"

	return scaleSeabornFonts(t, 14, 11)
}

func newSeabornTalk() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_talk"

	return scaleSeabornFonts(t, 17, 13.5)
}

func newSeabornPoster() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_poster"

	return scaleSeabornFonts(t, 20, 16.5)
}
