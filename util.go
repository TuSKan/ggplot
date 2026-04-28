// util.go contains internal utility functions used across the ggplot rendering pipeline.
package ggplot

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/TuSKan/ggplot/colormap"
	"github.com/TuSKan/ggplot/internal/canvas"
)

// fileExt returns the lowercased file extension including the dot.
func fileExt(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

// createFile creates or truncates a file for writing.
func createFile(filename string) (*os.File, error) {
	return os.Create(filename)
}

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

// normalize maps a value to [0, 1] within [min, max].
func normalize(v, min, max float64) float64 {
	if max == min {
		return 0.5
	}
	return (v - min) / (max - min)
}

// resolveColor parses a color literal into normalized [0,1] RGB. Accepts hex
// strings, CSS named colors, "tab:*" aliases, and rgb()/hsl() functional
// forms via [colormap.Parse]. Falls back to defaults on empty or invalid
// input.
func resolveColor(spec string, defR, defG, defB float64) (float64, float64, float64) {
	if spec == "" {
		return defR, defG, defB
	}
	c, err := colormap.Parse(spec)
	if err != nil {
		return defR, defG, defB
	}
	return c.R, c.G, c.B
}
