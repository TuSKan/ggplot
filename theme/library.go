package theme

import "image/color"

func init() {
	MustRegister(Minimal, newMinimal)
	MustRegister(Dark, newDark)
	MustRegister(BW, newBW)
}

// newMinimal returns a stripped-down ggplot-style theme with no panel
// border, very light grid, and no tick marks.
func newMinimal() Theme {
	t := newGgplot()
	t.Name = "minimal"
	t.Elements["panel.border"] = ElementBlank{}
	t.Elements["panel.grid.major"] = ElementLine{Color: color.RGBA{R: 230, G: 230, B: 230, A: 255}, Size: 0.5}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.ticks"] = ElementBlank{}

	return t
}

// newDark returns the library's blue-tinted dark theme.
func newDark() Theme {
	bg := color.RGBA{R: 30, G: 30, B: 40, A: 255}
	textColor := color.RGBA{R: 220, G: 220, B: 230, A: 255}
	gridColor := color.RGBA{R: 55, G: 55, B: 65, A: 255}

	return Theme{
		Name: "dark",
		Spacing: Spacing{
			MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10,
			PanelSpacing: 10,
		},
		Geom: GeomDefaults{
			PatchEdgeColor: color.White,
			PatchEdgeWidth: 0.5,
			PatchAlpha:     0.9,
		},
		Elements: map[string]Element{
			"text": ElementText{Family: "sans-serif", Size: 10, Color: textColor},
			"line": ElementLine{Color: color.RGBA{R: 100, G: 100, B: 120, A: 255}, Size: 1},
			"rect": ElementRect{Fill: bg},

			"plot.title":    ElementText{Size: 16, Bold: true},
			"plot.subtitle": ElementText{Size: 12, Color: color.RGBA{R: 180, G: 180, B: 200, A: 255}},

			"plot.background":  ElementRect{Fill: bg},
			"panel.background": ElementRect{Fill: bg},
			"panel.border":     ElementRect{Color: gridColor, Size: 1},
			"panel.grid.major": ElementLine{Color: color.RGBA{R: 60, G: 60, B: 75, A: 160}, Size: 0.5, Linetype: []float64{4, 4}},
			"panel.grid.minor": ElementLine{Color: color.RGBA{R: 45, G: 45, B: 55, A: 100}, Size: 0.3},

			"axis.title": ElementText{Size: 12},
			"axis.text":  ElementText{Color: color.RGBA{R: 160, G: 160, B: 180, A: 255}},
			"axis.ticks": ElementLine{Color: color.RGBA{R: 100, G: 100, B: 120, A: 255}, Size: 1},

			"legend.text": ElementText{},
		},
	}
}

// newBW returns a black-and-white print theme.
func newBW() Theme {
	t := baseTheme("bw")
	t.Elements["panel.border"] = ElementRect{Color: color.Black, Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: color.RGBA{R: 200, G: 200, B: 200, A: 255}, Size: 0.5}
	t.Elements["panel.grid.minor"] = ElementLine{Color: color.RGBA{R: 230, G: 230, B: 230, A: 255}, Size: 0.3}
	t.Geom.PatchEdgeColor = color.Black
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	return t
}
