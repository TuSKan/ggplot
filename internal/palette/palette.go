package palette

import "image/color"

// Viridis returns the RGB approximation mapped across domain [0, 1].
// Extrapolates standard color boundaries gracefully clamping limits safely.
func Viridis(v float64) color.Color {
	if v < 0.0 {
		v = 0.0
	} else if v > 1.0 {
		v = 1.0
	}

	// 5-Color Stop simplification bounds.
	stops := []struct {
		point float64
		col   color.RGBA
	}{
		{0.00, color.RGBA{68, 1, 84, 255}},
		{0.25, color.RGBA{59, 82, 139, 255}},
		{0.50, color.RGBA{33, 145, 140, 255}},
		{0.75, color.RGBA{94, 201, 98, 255}},
		{1.00, color.RGBA{253, 231, 37, 255}},
	}

	for i := 0; i < len(stops)-1; i++ {
		s1 := stops[i]
		s2 := stops[i+1]
		if v >= s1.point && v <= s2.point {
			t := (v - s1.point) / (s2.point - s1.point)
			return color.RGBA{
				R: uint8(float64(s1.col.R) + t*float64(s2.col.R-s1.col.R)),
				G: uint8(float64(s1.col.G) + t*float64(s2.col.G-s1.col.G)),
				B: uint8(float64(s1.col.B) + t*float64(s2.col.B-s1.col.B)),
				A: 255,
			}
		}
	}
	return stops[len(stops)-1].col
}

// OkabeIto resolves an 8-color cyclic colorblind-safe sequential discrete sequence mapping mapped indexes statically.
func OkabeIto(idx int) color.Color {
	palette := []color.RGBA{
		{230, 159, 0, 255},   // Orange
		{86, 180, 233, 255},  // Sky Blue
		{0, 158, 115, 255},   // Green
		{240, 228, 66, 255},  // Yellow
		{0, 114, 178, 255},   // Blue
		{213, 94, 0, 255},    // Vermillion
		{204, 121, 167, 255}, // Reddish Purple
		{0, 0, 0, 255},       // Black
	}
	if idx < 0 {
		return color.RGBA{150, 150, 150, 255} // Fallback Gray missing mappings visually.
	}
	return palette[idx%len(palette)]
}
