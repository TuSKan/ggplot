// Package theme provides visual styling configurations for plots.
// Themes control non-data visual elements: background colors, fonts,
// grid lines, tick marks, and spacing.
package theme

import (
	"fmt"
	"image/color"
)

// Name identifies a built-in theme.
type Name string

const (
	Default Name = "default"
	Classic Name = "classic"
	Minimal Name = "minimal"
	Dark    Name = "dark"
	BW      Name = "bw"
)

// Theme encapsulates the complete visual styling for a plot.
type Theme struct {
	Name       string
	Background color.Color
	Panel      PanelStyle
	Grid       GridStyle
	Text       TextStyles
	Ticks      TickStyle
	Spacing    Spacing
}

// PanelStyle controls the data panel appearance.
type PanelStyle struct {
	Background  color.Color
	Border      color.Color
	BorderWidth float64
}

// GridStyle controls major and minor grid lines.
type GridStyle struct {
	MajorColor     color.Color
	MajorWidth     float64
	MinorColor     color.Color
	MinorWidth     float64
	MajorLineCount int       // 0 = auto
	DashPattern    []float64 // nil = solid, e.g. {4,4} = dashed, {2,3} = dotted
}

// TextStyles holds font configurations for different text roles.
type TextStyles struct {
	Title      FontConfig
	Subtitle   FontConfig
	AxisTitle  FontConfig
	TickLabel  FontConfig
	Legend     FontConfig
	Annotation FontConfig
}

// FontConfig encapsulates text rendering parameters.
type FontConfig struct {
	Family string
	Size   float64
	Color  color.Color
	Bold   bool
	Italic bool
}

// TickStyle controls axis tick mark appearance.
type TickStyle struct {
	Length float64
	Width  float64
	Color  color.Color
}

// Spacing controls margins and inter-element spacing.
type Spacing struct {
	MarginTop    float64
	MarginRight  float64
	MarginBottom float64
	MarginLeft   float64
	PanelSpacing float64
}

// --- Built-in themes ---

// newDefault returns the standard light theme.
func newDefault() Theme {
	return Theme{
		Name:       "default",
		Background: color.White,
		Panel: PanelStyle{
			Background:  color.White,
			Border:      color.RGBA{R: 200, G: 200, B: 200, A: 255},
			BorderWidth: 1,
		},
		Grid: GridStyle{
			MajorColor:  color.RGBA{R: 200, G: 200, B: 200, A: 180},
			MajorWidth:  0.5,
			MinorColor:  color.RGBA{R: 230, G: 230, B: 230, A: 120},
			MinorWidth:  0.3,
			DashPattern: []float64{4, 4},
		},
		Text: TextStyles{
			Title:      FontConfig{Family: "sans-serif", Size: 16, Color: color.Black, Bold: true},
			Subtitle:   FontConfig{Family: "sans-serif", Size: 12, Color: gray(80)},
			AxisTitle:  FontConfig{Family: "sans-serif", Size: 12, Color: gray(40)},
			TickLabel:  FontConfig{Family: "sans-serif", Size: 10, Color: gray(60)},
			Legend:     FontConfig{Family: "sans-serif", Size: 10, Color: gray(40)},
			Annotation: FontConfig{Family: "sans-serif", Size: 10, Color: gray(60)},
		},
		Ticks: TickStyle{
			Length: 5, Width: 1, Color: gray(60),
		},
		Spacing: Spacing{
			MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10,
			PanelSpacing: 10,
		},
	}
}

// newClassic returns the ggplot2 classic theme (white background, grid lines).
func newClassic() Theme {
	t := newDefault()
	t.Name = "classic"
	return t
}

// newMinimal returns a stripped-down theme with minimal visual clutter.
func newMinimal() Theme {
	t := newDefault()
	t.Name = "minimal"
	t.Panel.Border = color.Transparent
	t.Panel.BorderWidth = 0
	t.Grid.MajorColor = color.RGBA{R: 230, G: 230, B: 230, A: 255}
	t.Grid.MinorColor = color.Transparent
	t.Ticks.Color = color.Transparent
	t.Ticks.Length = 0
	return t
}

// newDark returns a dark-background theme suitable for presentations.
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
	}
}

// newBW returns a black-and-white theme for print publications.
func newBW() Theme {
	t := newDefault()
	t.Name = "bw"
	t.Panel.Border = color.Black
	t.Panel.BorderWidth = 1
	t.Grid.MajorColor = color.RGBA{R: 200, G: 200, B: 200, A: 255}
	t.Grid.MinorColor = color.RGBA{R: 230, G: 230, B: 230, A: 255}
	return t
}

// Resolve returns a Theme for the given name.
// Returns an error for unknown names.
func Resolve(name Name) (Theme, error) {
	switch name {
	case Default, "":
		return newDefault(), nil
	case Classic:
		return newClassic(), nil
	case Minimal:
		return newMinimal(), nil
	case Dark:
		return newDark(), nil
	case BW:
		return newBW(), nil
	default:
		return Theme{}, fmt.Errorf("theme: unknown name %q", name)
	}
}

func gray(v uint8) color.Color {
	return color.RGBA{R: v, G: v, B: v, A: 255}
}
