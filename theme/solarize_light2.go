package theme

import "image/color"

func init() { MustRegister(SolarizeLight2, newSolarizeLight2) }

// newSolarizeLight2 mirrors matplotlib's Solarize_Light2.mplstyle.
func newSolarizeLight2() Theme {
	t := baseTheme("solarize_light2")
	base00 := hex("657B83")
	base2 := hex("EEE8D5")
	base3 := hex("FDF6E3")

	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 10, Color: base00}
	t.Elements["plot.title"] = ElementText{Size: 16, Bold: true}
	t.Elements["plot.background"] = ElementRect{Fill: base3}
	t.Elements["panel.background"] = ElementRect{Fill: base2}
	t.Elements["panel.border"] = ElementRect{Color: base00, Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: base3, Size: 1}
	t.Elements["panel.grid.minor"] = ElementLine{Color: hexA("FDF6E3", 160), Size: 0.5}
	t.Elements["axis.ticks"] = ElementLine{Color: base00, Size: 1}

	t.Geom.PatchEdgeColor = base3
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	t.Palette = []color.Color{
		hex("268BD2"), hex("2AA198"), hex("859900"), hex("CB4B16"),
		hex("D33682"), hex("6C71C4"), hex("657B83"), hex("93A1A1"),
	}

	return t
}
