package guide

import (
	"image/color"

	"github.com/TuSKan/ggplot/pkg/theme"
)

// TextMeasurer defines an externalized callback allowing specific renderers
// to provide bounding font boxes deterministically.
type TextMeasurer interface {
	MeasureText(text string, f theme.FontConfig) (width, height float64)
}

// ContinuousScale projects original data onto a [0,1] plane and formats ticks.
type ContinuousScale interface {
	Ticks(count int) []float64
	Format(v float64) string
	Project(v float64) float64
}

// Painter allows guides to abstractly draw lines, text, and rectangles
// without coupling to a specific rendering backend (SVG, Canvas, Ebiten).
type Painter interface {
	// DrawText draws a string aligned by anchorX ("left", "center", "right") and anchorY ("top", "middle", "bottom").
	// Rotation is specified in radians.
	DrawText(text string, x, y float64, anchorX, anchorY string, f theme.FontConfig, c color.Color, rotation float64)
	
	// DrawLine draws a stroke between two points. Dash array specifies stroke patterns (nil for solid).
	DrawLine(x1, y1, x2, y2 float64, c color.Color, width float64, dash []float64)
	
	// DrawRect draws a filled and optional outlined rectangle.
	DrawRect(x, y, w, h float64, fill, stroke color.Color, strokeWidth float64)
}

// Element is a drawn guide component which computes its size then draws itself based on the allotted bounds.
type Element interface {
	// Measure computes the static width and height this element requires offline.
	Measure(m TextMeasurer, th theme.Theme) (width, height float64)
	
	// Draw emits rendering commands into Painter within the allocated x, y, width, and height box.
	Draw(p Painter, x, y, width, height float64, th theme.Theme)
}

// GuideSet organizes non-data view components bounding the main plotting area.
type GuideSet struct {
	Top    []Element
	Bottom []Element
	Left   []Element
	Right  []Element
}
