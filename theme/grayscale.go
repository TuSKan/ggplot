package theme

import "image/color"

func init() { MustRegister(Grayscale, newGrayscale) }

// newGrayscale mirrors matplotlib's grayscale.mplstyle: monochrome
// output suited for B&W print or accessibility.
//
// Source: matplotlib/lib/matplotlib/mpl-data/stylelib/grayscale.mplstyle
func newGrayscale() Theme {
	t := baseTheme("grayscale")

	// figure.facecolor: 0.75
	t.Background = gray(192)

	t.Panel.Background = color.White
	t.Panel.Border = color.Black
	t.Panel.BorderWidth = 1

	t.Grid.MajorColor = gray(180)
	t.Grid.MajorWidth = 0.5
	t.Grid.MinorColor = gray(220)
	t.Grid.MinorWidth = 0.3
	t.Grid.DashPattern = nil

	t.Text.Title = FontConfig{Family: "sans-serif", Size: 14, Color: color.Black, Bold: true}
	t.Text.Subtitle = FontConfig{Family: "sans-serif", Size: 11, Color: color.Black}
	t.Text.AxisTitle = FontConfig{Family: "sans-serif", Size: 11, Color: color.Black}
	t.Text.TickLabel = FontConfig{Family: "sans-serif", Size: 10, Color: color.Black}
	t.Text.Legend = FontConfig{Family: "sans-serif", Size: 10, Color: color.Black}
	t.Text.Annotation = FontConfig{Family: "sans-serif", Size: 10, Color: color.Black}

	t.Ticks.Color = color.Black

	// axes.prop_cycle: ['0.00', '0.40', '0.60', '0.70']
	t.Palette = []color.Color{
		gray(0),
		gray(102),
		gray(153),
		gray(178),
	}
	return t
}
