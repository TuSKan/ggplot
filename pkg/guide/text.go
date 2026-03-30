package guide

import (
	"math"

	"github.com/TuSKan/ggplot/pkg/theme"
)

// Text renders static formatted string titles, sub-titles, and axis labels.
type Text struct {
	Text      string
	Font      theme.FontConfig
	Color     string // hex string like "#333333" mapped to color.Color by the theme/painter later, or kept as a constant. Wait, theme usually holds color strings.
	Rotation  float64 // Radians
	AlignX    string // "left", "center", "right"
	AlignY    string // "top", "middle", "bottom"
}




// Measure considers text orientation before returning bounds.
func (t Text) Measure(m TextMeasurer, th theme.Theme) (float64, float64) {
	if t.Text == "" {
		return 0, 0
	}
	
	w, h := m.MeasureText(t.Text, t.Font)
	
	// If rotated 90 degrees (pi/2) or 270 degrees, swap w and h
	// Let's do simple bounding box for typical pi/2 rotation:
	if math.Abs(math.Cos(t.Rotation)) < 0.001 {
		return h, w
	}
	
	// Approximation for arbitrary angle
	boundsW := math.Abs(w*math.Cos(t.Rotation)) + math.Abs(h*math.Sin(t.Rotation))
	boundsH := math.Abs(w*math.Sin(t.Rotation)) + math.Abs(h*math.Cos(t.Rotation))
	
	return boundsW, boundsH
}

func (t Text) Draw(p Painter, x, y, width, height float64, th theme.Theme) {
	if t.Text == "" {
		return
	}
	
	c := theme.ParseHexColor(t.Color)
	
	// Default to center if not specified
	alignX := t.AlignX
	if alignX == "" {
		alignX = "center"
	}
	alignY := t.AlignY
	if alignY == "" {
		alignY = "middle"
	}

	// Mid-points for standard placement bounds
	cx := x + width/2.0
	cy := y + height/2.0
	
	p.DrawText(t.Text, cx, cy, alignX, alignY, t.Font, c, t.Rotation)
}
