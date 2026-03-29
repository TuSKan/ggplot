package theme

import (
	"image/color"
)

// Theme encapsulates styling configurations like fonts and stroke widths.
type Theme struct {
	Background color.Color
	GridColor  color.Color
}
