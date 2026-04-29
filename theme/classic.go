package theme

import "image/color"

func init() { MustRegister(Classic, newClassic) }

// newClassic mirrors matplotlib's classic.mplstyle (the matplotlib 1.x
// defaults): white panel, black axes, no grid, primary-color cycle.
//
// Source: matplotlib/lib/matplotlib/mpl-data/stylelib/classic.mplstyle
func newClassic() Theme {
	t := baseTheme("classic")

	// figure.facecolor: 0.75 (light gray)
	t.Background = gray(192)

	t.Panel.Background = color.White
	t.Panel.Border = color.Black
	t.Panel.BorderWidth = 1

	// axes.grid: False — keep dashed, very subtle so users that turn the
	// grid on still get something readable.
	t.Grid.MajorColor = color.Transparent
	t.Grid.MajorWidth = 0
	t.Grid.MinorColor = color.Transparent
	t.Grid.MinorWidth = 0
	t.Grid.DashPattern = []float64{1, 3}

	t.Text.Title = FontConfig{Family: "sans-serif", Size: 14, Color: color.Black, Bold: true}
	t.Text.Subtitle = FontConfig{Family: "sans-serif", Size: 12, Color: color.Black}
	t.Text.AxisTitle = FontConfig{Family: "sans-serif", Size: 12, Color: color.Black}
	t.Text.TickLabel = FontConfig{Family: "sans-serif", Size: 12, Color: color.Black}
	t.Text.Legend = FontConfig{Family: "sans-serif", Size: 12, Color: color.Black}
	t.Text.Annotation = FontConfig{Family: "sans-serif", Size: 12, Color: color.Black}

	t.Ticks.Color = color.Black

	// axes.prop_cycle: 'bgrcmyk'
	t.Palette = []color.Color{
		hex("0000FF"), // b — blue
		hex("008000"), // g — green
		hex("FF0000"), // r — red
		hex("00BFBF"), // c — cyan
		hex("BF00BF"), // m — magenta
		hex("BFBF00"), // y — yellow
		hex("000000"), // k — black
	}
	return t
}
