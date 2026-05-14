package theme

import "image/color"

func init() {
	MustRegister(FewLight, newFewLight)
	MustRegister(FewDark, newFewDark)
}

// fewChrome returns the shared white-panel chrome for all Few variants.
func fewChrome(name string) Theme {
	t := baseTheme(name)
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: color.Black}
	t.Elements["panel.border"] = ElementRect{Color: color.Black, Size: 1}
	t.Elements["panel.grid.major"] = ElementBlank{}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.ticks"] = ElementLine{Color: color.Black, Size: 1}
	t.Geom.PatchEdgeColor = color.Black
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	return t
}

func newFewLight() Theme {
	t := fewChrome("few_light")
	t.Palette = []color.Color{
		hex("8C8C8C"), hex("88BDE6"), hex("FBB258"), hex("90CD97"), hex("F6AAC9"),
		hex("BFA554"), hex("BC99C7"), hex("EDDD46"), hex("F07E6E"),
	}

	return t
}

func newFewDark() Theme {
	t := fewChrome("few_dark")
	t.Palette = []color.Color{
		hex("000000"), hex("265DAB"), hex("DF5C24"), hex("059748"), hex("E5126F"),
		hex("9D722A"), hex("7B3A96"), hex("C7B42E"), hex("CB2027"),
	}

	return t
}
