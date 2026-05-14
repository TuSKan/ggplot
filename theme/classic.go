package theme

import "image/color"

func init() { MustRegister(Classic, newClassic) }

// newClassic mirrors matplotlib's classic.mplstyle (the matplotlib 1.x defaults).
func newClassic() Theme {
	t := baseTheme("classic")
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 12, Color: color.Black}
	t.Elements["plot.title"] = ElementText{Size: 14, Bold: true}
	t.Elements["plot.background"] = ElementRect{Fill: gray(192)}
	t.Elements["panel.background"] = ElementRect{Fill: color.White}
	t.Elements["panel.border"] = ElementRect{Color: color.Black, Size: 1}
	t.Elements["panel.grid.major"] = ElementBlank{}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.ticks"] = ElementLine{Color: color.Black, Size: 1}

	t.Geom.PatchEdgeColor = color.Black
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	t.Palette = []color.Color{
		hex("0000FF"), // blue
		hex("008000"), // green
		hex("FF0000"), // red
		hex("00BFBF"), // cyan
		hex("BF00BF"), // magenta
		hex("BFBF00"), // yellow
		hex("000000"), // black
	}

	return t
}
