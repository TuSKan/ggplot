package guide

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/TuSKan/ggplot/pkg/theme"
)

// Mocks

type mockMeasurer struct{}

func (m mockMeasurer) MeasureText(text string, f theme.FontConfig) (float64, float64) {
	// Simple mock mapping char count * 10
	return float64(len(text)) * 10.0, 15.0
}

type mockScale struct {
	ticks []float64
}

func (s mockScale) Ticks(count int) []float64 { return s.ticks }
func (s mockScale) Format(v float64) string   { return fmt.Sprintf("%.1f", v) }
func (s mockScale) Project(v float64) float64 { return v }

type mockPainter struct {
	drawTextCalls int
	drawLineCalls int
}

func (p *mockPainter) DrawText(text string, x, y float64, ax, ay string, f theme.FontConfig, c color.Color, rot float64) {
	p.drawTextCalls++
}
func (p *mockPainter) DrawLine(x1, y1, x2, y2 float64, c color.Color, w float64, d []float64) {
	p.drawLineCalls++
}
func (p *mockPainter) DrawRect(x, y, w, h float64, fill, stroke color.Color, sw float64) {}

func TestLayoutCompute(t *testing.T) {
	th := theme.Default()
	m := mockMeasurer{}

	gs := GuideSet{
		Top: []Element{
			Text{Text: "Main Title"},
			Text{Text: "Subtitle"},
		},
		Bottom: []Element{
			Axis{Direction: "horizontal", Position: "bottom", Scale: &mockScale{ticks: []float64{0, 1}}, Title: "X Axis"},
		},
		Left: []Element{
			&Axis{Direction: "vertical", Position: "left", Scale: &mockScale{ticks: []float64{0, 1}}, Title: "Y Axis"},
		},
		Right: []Element{
			&Legend{Title: "Cats", Swatches: []LegendSwatch{{Label: "A"}, {Label: "B"}}},
		},
	}

	layout := Compute(&gs, 800, 600, m, th)

	if layout.Outer.W != 800 || layout.Outer.H != 600 {
		t.Errorf("Outer bounds mutated")
	}

	if layout.Data.W >= 800 || layout.Data.H >= 600 {
		t.Errorf("Data space not reduced by guides")
	}

	if len(layout.Placements) != 5 {
		t.Errorf("Expected 5 mapped elements, got %d", len(layout.Placements))
	}
}

func TestAxisDraw(t *testing.T) {
	th := theme.Default()
	p := &mockPainter{}

	ax := Axis{
		Direction: "horizontal",
		Position:  "bottom",
		Scale:     &mockScale{ticks: []float64{0.0, 0.5, 1.0}},
		Title:     "X",
	}

	ax.Draw(p, 0, 500, 800, 50, th)

	if p.drawLineCalls != 4 { // 1 baseline + 3 ticks
		t.Errorf("Expected 4 lines, got %d", p.drawLineCalls)
	}
	if p.drawTextCalls != 4 { // 3 ticks + 1 title
		t.Errorf("Expected 4 text draws, got %d", p.drawTextCalls)
	}
}

func TestLegendMeasure(t *testing.T) {
	th := theme.Default()
	m := &mockMeasurer{}

	leg := Legend{
		Title: "Colors",
		Swatches: []LegendSwatch{
			{Label: "Red", Fill: "#FF0000"},
			{Label: "Blue", Fill: "#0000FF"},
		},
		Columns: 2,
	}

	w, h := leg.Measure(m, th)
	if w <= 0 || h <= 0 {
		t.Errorf("Legend measure failed: %v, %v", w, h)
	}
}


