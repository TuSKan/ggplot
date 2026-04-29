package theme

import "image/color"

func init() { MustRegister(Colorblind, newColorblind) }

// newColorblind implements the Wong (2011) 8-color colorblind-safe palette.
//
// Source: Wong, B. (2011). Color blindness. Nature Methods, 8(6), 441.
// https://www.nature.com/articles/nmeth.1618
//
// This is the most widely cited, scientifically validated discrete palette
// for colorblind accessibility. It is safe for deuteranopia, protanopia,
// and tritanopia. The chrome (white panel, light grid) is neutral so the
// colors carry all the visual weight.
func newColorblind() Theme {
	t := baseTheme("colorblind")

	t.Background = color.White
	t.Panel.Background = color.White
	t.Panel.Border = gray(180)
	t.Panel.BorderWidth = 1

	t.Grid.MajorColor = gray(220)
	t.Grid.MajorWidth = 0.5
	t.Grid.MinorColor = gray(240)
	t.Grid.MinorWidth = 0.3
	t.Grid.DashPattern = nil

	t.Ticks.Color = gray(60)

	// White panel + light gray grid → matching gray edge.
	t.Geom.PatchEdgeColor = gray(220)
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	// Wong 2011 palette — verbatim from pyplot-themes Colorblind class.
	t.Palette = []color.Color{
		hex("000000"), // Black
		hex("E69F00"), // Orange
		hex("56B4E9"), // Sky Blue
		hex("009E73"), // Bluish Green
		hex("F0E442"), // Yellow
		hex("0072B2"), // Blue
		hex("D55E00"), // Vermillion
		hex("CC79A7"), // Reddish Purple
	}
	return t
}
