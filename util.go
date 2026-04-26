// util.go contains internal utility functions used across the ggplot rendering pipeline.
package ggplot

import (
	"image/color"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/internal/canvas"
	icolor "github.com/TuSKan/ggplot/internal/color"
)

// setColorFromTheme sets the canvas colour from a theme color.Color,
// handling nil (defaults to black).
func setColorFromTheme(cv canvas.Canvas, c color.Color) {
	if c == nil {
		cv.SetRGBA(0, 0, 0, 1)
		return
	}
	r, g, b, a := c.RGBA()
	cv.SetRGBA(
		float64(r)/65535.0,
		float64(g)/65535.0,
		float64(b)/65535.0,
		float64(a)/65535.0,
	)
}

// getFloat64Values returns the float64 values for the named column, or nil on any error.
func getFloat64Values(ds dataset.Dataset, col string) []float64 {
	if col == "" {
		return nil
	}
	c, err := ds.Column(col)
	if err != nil {
		return nil
	}
	fc, ok := c.(dataset.Column[float64])
	if !ok {
		return nil
	}
	return fc.Values()
}

// normalize maps a value to [0, 1] within [min, max].
func normalize(v, min, max float64) float64 {
	if max == min {
		return 0.5
	}
	return (v - min) / (max - min)
}

// resolveColor parses a hex color string into normalized [0,1] RGB.
// Falls back to defaults if hex is empty or invalid.
func resolveColor(hex string, defR, defG, defB float64) (float64, float64, float64) {
	if hex == "" {
		return defR, defG, defB
	}
	c := icolor.Hex(hex)
	if c.A == 0 {
		// Hex returns transparent on invalid input.
		return defR, defG, defB
	}
	return float64(c.R) / 255.0, float64(c.G) / 255.0, float64(c.B) / 255.0
}
