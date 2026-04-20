// Package color provides color palettes and utilities for data visualization.
package color

import "image/color"

// Viridis maps a scalar in [0,1] to the Viridis perceptually-uniform colormap.
func Viridis(t float64) color.Color {
	t = clamp01(t)
	return interpolateStops(t, viridisStops)
}

// Magma maps a scalar in [0,1] to the Magma perceptually-uniform colormap.
func Magma(t float64) color.Color {
	t = clamp01(t)
	return interpolateStops(t, magmaStops)
}

// Inferno maps a scalar in [0,1] to the Inferno perceptually-uniform colormap.
func Inferno(t float64) color.Color {
	t = clamp01(t)
	return interpolateStops(t, infernoStops)
}

// OkabeIto returns the i-th color from the Okabe-Ito colorblind-safe palette.
// The palette cycles through 8 colors.
func OkabeIto(i int) color.Color {
	if i < 0 {
		return color.RGBA{R: 150, G: 150, B: 150, A: 255}
	}
	return okabeItoPalette[i%len(okabeItoPalette)]
}

// OkabeItoPalette returns the full 8-color Okabe-Ito palette.
func OkabeItoPalette() []color.Color {
	cp := make([]color.Color, len(okabeItoPalette))
	for i, c := range okabeItoPalette {
		cp[i] = c
	}
	return cp
}

// Category10 returns the i-th color from the D3 Category10 palette.
func Category10(i int) color.Color {
	if i < 0 {
		return color.RGBA{R: 150, G: 150, B: 150, A: 255}
	}
	return category10Palette[i%len(category10Palette)]
}

// --- Color stops ---

type stop struct {
	t float64
	c color.RGBA
}

var viridisStops = []stop{
	{0.00, color.RGBA{68, 1, 84, 255}},
	{0.25, color.RGBA{59, 82, 139, 255}},
	{0.50, color.RGBA{33, 145, 140, 255}},
	{0.75, color.RGBA{94, 201, 98, 255}},
	{1.00, color.RGBA{253, 231, 37, 255}},
}

var magmaStops = []stop{
	{0.00, color.RGBA{0, 0, 4, 255}},
	{0.25, color.RGBA{81, 18, 124, 255}},
	{0.50, color.RGBA{183, 55, 121, 255}},
	{0.75, color.RGBA{252, 135, 97, 255}},
	{1.00, color.RGBA{252, 253, 191, 255}},
}

var infernoStops = []stop{
	{0.00, color.RGBA{0, 0, 4, 255}},
	{0.25, color.RGBA{87, 16, 110, 255}},
	{0.50, color.RGBA{188, 55, 84, 255}},
	{0.75, color.RGBA{249, 142, 9, 255}},
	{1.00, color.RGBA{252, 255, 164, 255}},
}

var okabeItoPalette = []color.RGBA{
	{230, 159, 0, 255},   // Orange
	{86, 180, 233, 255},  // Sky blue
	{0, 158, 115, 255},   // Green
	{240, 228, 66, 255},  // Yellow
	{0, 114, 178, 255},   // Blue
	{213, 94, 0, 255},    // Vermillion
	{204, 121, 167, 255}, // Reddish purple
	{0, 0, 0, 255},       // Black
}

var category10Palette = []color.RGBA{
	{31, 119, 180, 255},  // Blue
	{255, 127, 14, 255},  // Orange
	{44, 160, 44, 255},   // Green
	{214, 39, 40, 255},   // Red
	{148, 103, 189, 255}, // Purple
	{140, 86, 75, 255},   // Brown
	{227, 119, 194, 255}, // Pink
	{127, 127, 127, 255}, // Gray
	{188, 189, 34, 255},  // Yellow-green
	{23, 190, 207, 255},  // Cyan
}

func interpolateStops(t float64, stops []stop) color.Color {
	for i := 0; i < len(stops)-1; i++ {
		if t >= stops[i].t && t <= stops[i+1].t {
			f := (t - stops[i].t) / (stops[i+1].t - stops[i].t)
			return color.RGBA{
				R: lerp8(stops[i].c.R, stops[i+1].c.R, f),
				G: lerp8(stops[i].c.G, stops[i+1].c.G, f),
				B: lerp8(stops[i].c.B, stops[i+1].c.B, f),
				A: 255,
			}
		}
	}
	return stops[len(stops)-1].c
}

func lerp8(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + t*float64(int(b)-int(a)))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
