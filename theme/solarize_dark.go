package theme

import "image/color"

func init() { MustRegister(SolarizeDark, newSolarizeDark) }

// newSolarizeDark implements Ethan Schoonover's Solarized Dark scheme.
func newSolarizeDark() Theme {
	t := baseTheme("solarize_dark")
	base03 := hex("002B36")
	base02 := hex("073642")
	base01 := hex("586E75")
	base0 := hex("839496")
	base1 := hex("93A1A1")

	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 10, Color: base0}
	t.Elements["plot.title"] = ElementText{Size: 14, Color: base1, Bold: true}
	t.Elements["plot.background"] = ElementRect{Fill: base03}
	t.Elements["panel.background"] = ElementRect{Fill: base02}
	t.Elements["panel.border"] = ElementRect{Color: base01, Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: hexA("FFFFFF", 45), Size: 0.8}
	t.Elements["panel.grid.minor"] = ElementLine{Color: hexA("FFFFFF", 20), Size: 0.4}
	t.Elements["axis.ticks"] = ElementLine{Color: base01, Size: 1}
	t.Elements["legend.text"] = ElementText{Color: base1}

	t.Geom.PatchEdgeColor = base01
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.9

	t.Palette = []color.Color{
		hex("B58900"), hex("CB4B16"), hex("DC322F"), hex("D33682"),
		hex("6C71C4"), hex("268BD2"), hex("2AA198"), hex("859900"),
	}

	return t
}
