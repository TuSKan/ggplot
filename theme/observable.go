package theme

import "image/color"

func init() { MustRegister(Observable, newObservable) }

// newObservable creates a modern, web-optimized theme inspired by
// Observable Plot's defaults. Clean white canvas, subtle gray grid,
// Inter typeface, and the Observable10 qualitative palette.
//
// Reference: https://observablehq.com/plot/
func newObservable() Theme {
	t := baseTheme("observable")
	textColor := hex("1b1e23")   // near-black for high contrast on screen
	mutedText := hex("6e7781")   // muted gray for secondary text
	gridColor := hex("d0d7de")   // very subtle grid
	borderColor := hex("d0d7de") // matching panel edge

	t.Elements["text"] = ElementText{Family: "Inter, system-ui, sans-serif", Size: 11, Color: textColor}
	t.Elements["plot.title"] = ElementText{Size: 16, Bold: true}
	t.Elements["plot.subtitle"] = ElementText{Size: 12, Color: mutedText}
	t.Elements["plot.background"] = ElementRect{Fill: color.White}
	t.Elements["panel.background"] = ElementRect{Fill: color.White}
	t.Elements["panel.border"] = ElementRect{Color: borderColor, Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: gridColor, Size: 0.5}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.title"] = ElementText{Size: 12, Color: mutedText}
	t.Elements["axis.text"] = ElementText{Size: 10, Color: mutedText}
	t.Elements["axis.ticks"] = ElementLine{Color: borderColor, Size: 1}
	t.Elements["legend.text"] = ElementText{Size: 10, Color: mutedText}

	t.Geom.PatchEdgeColor = color.White
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 0.85

	// Observable10 palette (10 screen-optimized qualitative colors).
	t.Palette = []color.Color{
		hex("4269d0"), // blue
		hex("efb118"), // gold
		hex("ff725c"), // coral
		hex("6cc5b0"), // teal
		hex("3ca951"), // green
		hex("ff8ab7"), // pink
		hex("a463f2"), // purple
		hex("97bbf5"), // sky
		hex("9c6b4e"), // brown
		hex("9498a0"), // gray
	}

	return t
}
