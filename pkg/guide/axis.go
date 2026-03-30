package guide

import (
	"math"

	"github.com/TuSKan/ggplot/pkg/theme"
)

// Axis implements a standard X or Y axis drawing logic.
type Axis struct {
	Direction string // "horizontal" or "vertical"
	Position  string // "top", "bottom", "left", "right"
	Scale     ContinuousScale // The scale logic providing ticks
	Title     string
	// Visual options decoupled from theme for overrides
	TickCount int
}

func (a Axis) Measure(m TextMeasurer, th theme.Theme) (float64, float64) {
	if a.Scale == nil {
		return 0, 0
	}

	count := a.TickCount
	if count == 0 {
		count = 5 // Default heuristic
	}
	ticks := a.Scale.Ticks(count)

	font := th.Typography.TickLabel
	maxW, maxH := 0.0, 0.0

	// Measure all ticks
	for _, v := range ticks {
		label := a.Scale.Format(v)
		w, h := m.MeasureText(label, font)
		if w > maxW {
			maxW = w
		}
		if h > maxH {
			maxH = h
		}
	}

	titleFont := th.Typography.AxisTitle
	titleW, titleH := 0.0, 0.0
	if a.Title != "" {
		w, h := m.MeasureText(a.Title, titleFont)
		if a.Direction == "vertical" {
			titleW = h // rotated by 90d
			titleH = w
		} else {
			titleW = w
			titleH = h
		}
	}

	tickLen := th.Ticks.Length
	spacing := 5.0 // Margin between elements

	if a.Direction == "horizontal" {
		return maxW * float64(len(ticks)), titleH + maxH + tickLen + spacing
	}
	
	// vertical
	return titleW + maxW + tickLen + spacing, maxH * float64(len(ticks))
}

func (a Axis) Draw(p Painter, x, y, width, height float64, th theme.Theme) {
	if a.Scale == nil {
		return
	}

	count := a.TickCount
	if count == 0 {
		count = 5
	}
	ticks := a.Scale.Ticks(count)

	strokeColor := th.Ticks.Color
	textColor := th.Typography.TickLabel.Color

	// Draw Baseline
	if a.Direction == "horizontal" {
		baselineY := y
		if a.Position == "bottom" {
			baselineY = y
		} else if a.Position == "top" {
			baselineY = y + height
		}
		p.DrawLine(x, baselineY, x+width, baselineY, strokeColor, 1.0, nil)

		// Draw ticks
		for _, v := range ticks {
			frac := a.Scale.Project(v)
			px := x + (frac * width)
			
			tickY := baselineY + th.Ticks.Length
			if a.Position == "top" {
				tickY = baselineY - th.Ticks.Length
			}
			p.DrawLine(px, baselineY, px, tickY, strokeColor, 1.0, nil)
			
			// Labels
			label := a.Scale.Format(v)
			labelY := tickY + 12
			if a.Position == "top" {
				labelY = tickY - 2
			}
			p.DrawText(label, px, labelY, "center", "middle", th.Typography.TickLabel, textColor, 0.0)
		}

		if a.Title != "" {
			titleColor := th.Typography.AxisTitle.Color
			titleY := y + height - 5
			p.DrawText(a.Title, x+width/2.0, titleY, "center", "middle", th.Typography.AxisTitle, titleColor, 0.0)
		}

	} else {
		// Vertical
		baselineX := x + width
		if a.Position == "right" {
			baselineX = x
		}
		p.DrawLine(baselineX, y, baselineX, y+height, strokeColor, 1.0, nil)

		for _, v := range ticks {
			frac := a.Scale.Project(v)
			py := y + height - (frac * height) // Invert Y naturally for canvas coordinates

			tickX := baselineX - th.Ticks.Length
			if a.Position == "right" {
				tickX = baselineX + th.Ticks.Length
			}
			p.DrawLine(baselineX, py, tickX, py, strokeColor, 1.0, nil)
			
			// Labels
			label := a.Scale.Format(v)
			labelX := tickX - 5
			anchor := "right"
			if a.Position == "right" {
				labelX = tickX + 5
				anchor = "left"
			}
			p.DrawText(label, labelX, py, anchor, "middle", th.Typography.TickLabel, textColor, 0.0)
		}

		if a.Title != "" {
			titleColor := th.Typography.AxisTitle.Color
			titleX := x + 10 // Left edge
			if a.Position == "right" {
				titleX = x + width - 10
			}
			// Rotated by -pi/2
			p.DrawText(a.Title, titleX, y+height/2.0, "center", "middle", th.Typography.AxisTitle, titleColor, -math.Pi/2.0)
		}
	}
}
