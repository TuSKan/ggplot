package theme

import "image/color"

func init() {
	MustRegister(Minimal, newMinimal)
	MustRegister(Dark, newDark)
	MustRegister(BW, newBW)
}

// Library originals — themes that don't come from matplotlib or
// pyplot-themes but are kept because callers selected them by name in
// existing user code. Visual definitions are unchanged from the
// pre-port versions.

// newMinimal returns a stripped-down ggplot-style theme with no panel
// border, very light grid, and no tick marks.
func newMinimal() Theme {
	t := newGgplot()
	t.Name = "minimal"
	t.Panel.Border = color.Transparent
	t.Panel.BorderWidth = 0
	t.Grid.MajorColor = color.RGBA{R: 230, G: 230, B: 230, A: 255}
	t.Grid.MinorColor = color.Transparent
	t.Ticks.Color = color.Transparent
	t.Ticks.Length = 0
	return t
}

// newDark returns the library's blue-tinted dark theme. For a more
// matplotlib-faithful dark style use [DarkBackground] instead.
func newDark() Theme {
	bg := color.RGBA{R: 30, G: 30, B: 40, A: 255}
	textColor := color.RGBA{R: 220, G: 220, B: 230, A: 255}
	gridColor := color.RGBA{R: 55, G: 55, B: 65, A: 255}

	return Theme{
		Name:       "dark",
		Background: bg,
		Panel: PanelStyle{
			Background:  bg,
			Border:      gridColor,
			BorderWidth: 1,
		},
		Grid: GridStyle{
			MajorColor:  color.RGBA{R: 60, G: 60, B: 75, A: 160},
			MajorWidth:  0.5,
			MinorColor:  color.RGBA{R: 45, G: 45, B: 55, A: 100},
			MinorWidth:  0.3,
			DashPattern: []float64{4, 4},
		},
		Text: TextStyles{
			Title:      FontConfig{Family: "sans-serif", Size: 16, Color: textColor, Bold: true},
			Subtitle:   FontConfig{Family: "sans-serif", Size: 12, Color: color.RGBA{R: 180, G: 180, B: 200, A: 255}},
			AxisTitle:  FontConfig{Family: "sans-serif", Size: 12, Color: textColor},
			TickLabel:  FontConfig{Family: "sans-serif", Size: 10, Color: color.RGBA{R: 160, G: 160, B: 180, A: 255}},
			Legend:     FontConfig{Family: "sans-serif", Size: 10, Color: textColor},
			Annotation: FontConfig{Family: "sans-serif", Size: 10, Color: textColor},
		},
		Ticks: TickStyle{
			Length: 5, Width: 1, Color: color.RGBA{R: 100, G: 100, B: 120, A: 255},
		},
		Spacing: Spacing{
			MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10,
			PanelSpacing: 10,
		},
		// dark: near-black panel + subtle grid → white edges pop off the dark canvas.
		Geom: GeomDefaults{
			PatchEdgeColor: color.White,
			PatchEdgeWidth: 0.5,
			PatchAlpha:     0.9,
		},
	}
}

// newBW returns a black-and-white print theme with a black panel
// border and solid gray grid lines.
func newBW() Theme {
	t := baseTheme("bw")
	t.Panel.Border = color.Black
	t.Panel.BorderWidth = 1
	t.Grid.MajorColor = color.RGBA{R: 200, G: 200, B: 200, A: 255}
	t.Grid.MajorWidth = 0.5
	t.Grid.MinorColor = color.RGBA{R: 230, G: 230, B: 230, A: 255}
	t.Grid.MinorWidth = 0.3
	t.Grid.DashPattern = nil
	// bw: white panel, black border, gray grid → black edges for B&W print.
	t.Geom.PatchEdgeColor = color.Black
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0
	return t
}
