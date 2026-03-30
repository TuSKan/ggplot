package render

import (
	"image/color"

	"github.com/TuSKan/ggplot/pkg/theme"
)

// Point dictates physical coordinate bounds iteratively bounding rendering passes.
type Point struct{ X, Y float64 }

// Rect sets spatial constraints for clipping/layout overlays.
type Rect struct{ Min, Max Point }

// Style provides core physical geometric definitions configuring pixels.
type Style struct {
	Fill        color.Color
	Stroke      color.Color
	StrokeWidth float64
}

// Backend isolates actual graphics context (whether CPU buffers or GPU primitives)
// restricting grammar components from executing concrete graphic actions.
type Backend interface {
	SetClipRect(r Rect)
	ClearClip()

	DrawPoint(x, y, radius float64, s Style)
	DrawLine(x1, y1, x2, y2 float64, s Style)
	DrawPolygon(points []Point, s Style)
	DrawRect(r Rect, s Style)

	// DrawText positions evaluated anchors allowing multi-line string constraints.
	DrawText(text string, x, y float64, alignH, alignV float64, s Style, f theme.FontConfig)

	Save(path string) error
}
