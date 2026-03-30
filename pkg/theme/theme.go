package theme

import (
	"image/color"
	"strconv"

	"github.com/TuSKan/ggplot/internal/fonts"
)

// FontConfig encapsulates text definitions.
type FontConfig struct {
	Family          string
	Size            float64
	Color           color.Color
	Weight          fonts.Weight
	Style           fonts.Style
	PreferMonospace bool
}

// ToFaceRequest bridges the theme explicit configurations externally directly bounding layout metrics exactly.
func (c FontConfig) ToFaceRequest(dpi float64) fonts.FaceRequest {
	return fonts.FaceRequest{
		Family:          c.Family,
		Weight:          c.Weight,
		Style:           c.Style,
		PreferMonospace: c.PreferMonospace,
		AllowFallback:   true,
		Size:            c.Size,
		DPI:             dpi,
	}
}

type Typography struct {
	Title      FontConfig
	Subtitle   FontConfig
	AxisTitle  FontConfig
	TickLabel  FontConfig
	Legend     FontConfig
	Annotation FontConfig
}

type TickStyle struct {
	Length    float64
	Thickness float64
	Color     color.Color
}

type GridLines struct {
	MajorColor     color.Color
	MajorThickness float64
	MinorColor     color.Color
	MinorThickness float64
}

type Spacing struct {
	MarginTop    float64
	MarginRight  float64
	MarginBottom float64
	MarginLeft   float64
	PanelSpacing float64
}

// Theme encapsulates styling configurations like fonts and stroke widths.
// This is the public API where users can configure all properties.
type Theme struct {
	Background color.Color
	GridLines  GridLines
	Typography Typography
	Ticks      TickStyle
	Spacing    Spacing
}

// IsTheme fulfills ast.ThemeConfig.
func (t Theme) IsTheme() {}

// Default produces a standard perceptually clean aesthetic.
// It is aliased to Classic for convenience.
func Default() Theme {
	return Theme{
		Background: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		GridLines: GridLines{
			MajorColor:     color.RGBA{R: 235, G: 235, B: 235, A: 255},
			MajorThickness: 1.0,
		},
		Typography: Typography{
			Title:      FontConfig{Family: "sans-serif", Size: 16, Color: color.Black, Weight: fonts.WeightBold},
			Subtitle:   FontConfig{Family: "sans-serif", Size: 12, Color: color.RGBA{R: 50, G: 50, B: 50, A: 255}, Weight: fonts.WeightNormal},
			AxisTitle:  FontConfig{Family: "sans-serif", Size: 12, Color: color.Black, Weight: fonts.WeightNormal},
			TickLabel:  FontConfig{Family: "sans-serif", Size: 10, Color: color.Black, Weight: fonts.WeightNormal},
			Legend:     FontConfig{Family: "sans-serif", Size: 10, Color: color.Black, Weight: fonts.WeightNormal},
			Annotation: FontConfig{Family: "sans-serif", Size: 10, Color: color.Black, Weight: fonts.WeightNormal},
		},
		Ticks: TickStyle{
			Length:    4.0,
			Thickness: 1.0,
			Color:     color.Black,
		},
		Spacing: Spacing{
			MarginTop:    10.0,
			MarginRight:  10.0,
			MarginBottom: 10.0,
			MarginLeft:   10.0,
			PanelSpacing: 10.0,
		},
	}
}

// Classic returns a clean structural default mapping.
func Classic() Theme {
	return Default()
}

// ParseHexColor parses hex strings into normalized color.Color (color.RGBA).
// Falls back to transparent on invalid input.
func ParseHexColor(hexStr string) color.Color {
	if len(hexStr) == 0 {
		return color.Transparent
	}
	if hexStr[0] == '#' {
		hexStr = hexStr[1:]
	}
	if len(hexStr) == 3 {
		hexStr = string([]byte{hexStr[0], hexStr[0], hexStr[1], hexStr[1], hexStr[2], hexStr[2]})
	}

	if len(hexStr) == 6 {
		values, err := strconv.ParseUint(hexStr, 16, 32)
		if err != nil {
			return color.Transparent
		}
		return color.RGBA{
			R: uint8(values >> 16),
			G: uint8((values >> 8) & 0xFF),
			B: uint8(values & 0xFF),
			A: 255,
		}
	} else if len(hexStr) == 8 {
		values, err := strconv.ParseUint(hexStr, 16, 32)
		if err != nil {
			return color.Transparent
		}
		return color.RGBA{
			R: uint8(values >> 24),
			G: uint8((values >> 16) & 0xFF),
			B: uint8((values >> 8) & 0xFF),
			A: uint8(values & 0xFF),
		}
	}

	return color.Transparent
}
