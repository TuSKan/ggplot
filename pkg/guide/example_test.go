package guide_test

import (
	"fmt"
	"image/color"

	"github.com/TuSKan/ggplot/pkg/theme"
	"github.com/TuSKan/ggplot/pkg/guide"
)

// ExampleContinuousScale implements a trivial mock scale for documentation.
type ExampleContinuousScale struct{}

func (s ExampleContinuousScale) Ticks(count int) []float64 { return []float64{0, 0.5, 1.0} }
func (s ExampleContinuousScale) Format(v float64) string   { return fmt.Sprintf("%.1f", v) }
func (s ExampleContinuousScale) Project(v float64) float64 { return v }

// ExamplePainter provides a stdout-mock for visual testing documentation.
type ExamplePainter struct{}

func (p ExamplePainter) DrawText(text string, x, y float64, ax, ay string, f theme.FontConfig, c color.Color, rot float64) {
	fmt.Printf("DrawText: '%s' @(%.0f,%.0f)\n", text, x, y)
}
func (p ExamplePainter) DrawLine(x1, y1, x2, y2 float64, c color.Color, w float64, d []float64) {}
func (p ExamplePainter) DrawRect(x, y, w, h float64, fill, stroke color.Color, sw float64)      {}

// ExampleMeasurer mocks fonts for basic layout bounds resolution.
type ExampleMeasurer struct{}

func (m ExampleMeasurer) MeasureText(text string, f theme.FontConfig) (float64, float64) {
	return float64(len(text)) * 10, 15
}

func ExampleGuideSet() {
	th := theme.Default()
	m := ExampleMeasurer{}
	p := ExamplePainter{}

	gs := guide.GuideSet{
		Top: []guide.Element{
			guide.Text{Text: "Production Output"},
		},
		Bottom: []guide.Element{
			guide.Axis{Direction: "horizontal", Position: "bottom", Scale: &ExampleContinuousScale{}},
		},
	}

	layout := guide.Compute(&gs, 800, 600, m, th)

	for _, place := range layout.Placements {
		b := place.Rect
		place.Element.Draw(p, b.X, b.Y, b.W, b.H, th)
	}

	// Output:
	// DrawText: 'Production Output' @(400,8)
	// DrawText: '0.0' @(0,592)
	// DrawText: '0.5' @(400,592)
	// DrawText: '1.0' @(800,592)
}
