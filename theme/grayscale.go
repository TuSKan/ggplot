package theme

import "image/color"

func init() { MustRegister(Grayscale, newGrayscale) }

// newGrayscale mirrors matplotlib's grayscale.mplstyle.
func newGrayscale() Theme {
	t := baseTheme("grayscale")
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 10, Color: color.Black}
	t.Elements["plot.title"] = ElementText{Size: 14, Bold: true}
	t.Elements["plot.background"] = ElementRect{Fill: gray(192)}
	t.Elements["panel.background"] = ElementRect{Fill: color.White}
	t.Elements["panel.border"] = ElementRect{Color: color.Black, Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: gray(180), Size: 0.5}
	t.Elements["panel.grid.minor"] = ElementLine{Color: gray(220), Size: 0.3}
	t.Elements["axis.ticks"] = ElementLine{Color: color.Black, Size: 1}

	t.Geom.PatchEdgeColor = gray(60)
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	t.Palette = []color.Color{gray(0), gray(102), gray(153), gray(178)}

	return t
}
