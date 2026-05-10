package theme

import "image/color"

func init() {
	MustRegister(FewLight, newFewLight)
	MustRegister(FewDark, newFewDark)
}

// fewChrome returns the shared white-panel chrome for all Few variants.
// (Same as the existing `few` theme; only the palette differs per variant.)
func fewChrome(name string) Theme {
	t := baseTheme(name)
	t.Background = color.White
	t.Panel.Background = color.White
	t.Panel.Border = color.Black
	t.Panel.BorderWidth = 1
	t.Grid.MajorColor = color.Transparent
	t.Grid.MajorWidth = 0
	t.Grid.MinorColor = color.Transparent
	t.Grid.MinorWidth = 0
	t.Ticks.Color = color.Black
	t.Text.Title.Color = color.Black
	t.Text.Subtitle.Color = color.Black
	t.Text.AxisTitle.Color = color.Black
	t.Text.TickLabel.Color = color.Black
	t.Text.Legend.Color = color.Black
	// white panel, no grid, black border → black edges match axes style.
	t.Geom.PatchEdgeColor = color.Black
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	return t
}

// newFewLight uses Stephen Few's "light" 9-color qualitative palette.
// Source: raybuhr/pyplot-themes palettes.Few.light
func newFewLight() Theme {
	t := fewChrome("few_light")
	t.Palette = []color.Color{
		hex("8C8C8C"), // gray
		hex("88BDE6"), // light blue
		hex("FBB258"), // light orange
		hex("90CD97"), // light green
		hex("F6AAC9"), // light pink
		hex("BFA554"), // khaki
		hex("BC99C7"), // light purple
		hex("EDDD46"), // yellow
		hex("F07E6E"), // salmon
	}

	return t
}

// newFewDark uses Stephen Few's "dark" 9-color qualitative palette.
// Source: raybuhr/pyplot-themes palettes.Few.dark
func newFewDark() Theme {
	t := fewChrome("few_dark")
	t.Palette = []color.Color{
		hex("000000"), // black
		hex("265DAB"), // dark blue
		hex("DF5C24"), // dark orange
		hex("059748"), // dark green
		hex("E5126F"), // dark pink
		hex("9D722A"), // dark brown
		hex("7B3A96"), // dark purple
		hex("C7B42E"), // dark yellow
		hex("CB2027"), // dark red
	}

	return t
}
