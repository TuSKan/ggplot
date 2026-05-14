package theme

import "image/color"

func init() { MustRegister(Fivethirtyeight, newFivethirtyeight) }

// newFivethirtyeight mirrors matplotlib's fivethirtyeight.mplstyle.
func newFivethirtyeight() Theme {
	t := baseTheme("fivethirtyeight")
	bg := hex("F0F0F0")

	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 12, Color: gray(60)}
	t.Elements["plot.title"] = ElementText{Size: 18, Color: gray(30), Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 14}
	t.Elements["plot.background"] = ElementRect{Fill: bg}
	t.Elements["panel.background"] = ElementRect{Fill: bg}
	t.Elements["panel.border"] = ElementRect{Color: bg, Size: 3}
	t.Elements["panel.grid.major"] = ElementLine{Color: hex("CBCBCB"), Size: 1}
	t.Elements["panel.grid.minor"] = ElementLine{Color: hexA("CBCBCB", 120), Size: 0.5}
	t.Elements["axis.ticks"] = ElementBlank{}

	t.Geom.PatchEdgeColor = hex("CBCBCB")
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	t.Palette = []color.Color{
		hex("008FD5"), hex("FC4F30"), hex("E5AE38"),
		hex("6D904F"), hex("8B8B8B"), hex("810F7C"),
	}

	return t
}
