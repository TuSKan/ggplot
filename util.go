// util.go contains internal utility functions used across the ggplot rendering pipeline.
package ggplot

import (
	"image/color"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/internal/canvas"
	"github.com/TuSKan/ggplot/theme"
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

// getFloat64Iter returns a Float64Iter for the named column, or nil on any error.
func getFloat64Iter(ds dataset.Dataset, col string) dataset.Float64Iter {
	if col == "" {
		return nil
	}
	c, err := ds.Column(col)
	if err != nil {
		return nil
	}
	iter, ok := c.(dataset.IterableColumn)
	if !ok {
		return nil
	}
	flt, err := iter.Float64s()
	if err != nil {
		return nil
	}
	return flt
}

// normalize maps a value to [0, 1] within [min, max].
func normalize(v, min, max float64) float64 {
	if max == min {
		return 0.5
	}
	return (v - min) / (max - min)
}

// resolveColor uses theme.ParseHexColor to parse hex strings,
// returning normalized [0,1] RGB. Falls back to defaults if hex is empty.
func resolveColor(hex string, defR, defG, defB float64) (float64, float64, float64) {
	if hex == "" {
		return defR, defG, defB
	}
	c := theme.ParseHexColor(hex)
	r, g, b, _ := c.RGBA()
	if r == 0 && g == 0 && b == 0 {
		// ParseHexColor returns transparent on invalid input.
		return defR, defG, defB
	}
	return float64(r) / 65535.0, float64(g) / 65535.0, float64(b) / 65535.0
}
