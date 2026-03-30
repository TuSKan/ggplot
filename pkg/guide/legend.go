package guide

import (
	"github.com/TuSKan/ggplot/pkg/theme"
)

// LegendSwatch represents a specific series binding.
type LegendSwatch struct {
	Label  string
	Fill   string
	Stroke string
}

// Legend provides categorical and continuous scaling reference panels.
type Legend struct {
	Title    string
	Swatches []LegendSwatch
	Columns  int
}

func (l Legend) Measure(m TextMeasurer, th theme.Theme) (float64, float64) {
	if len(l.Swatches) == 0 {
		return 0, 0
	}

	cols := l.Columns
	if cols <= 0 {
		cols = 1
	}

	var maxLabelW, maxLabelH float64
	for _, s := range l.Swatches {
		w, h := m.MeasureText(s.Label, th.Typography.Legend)
		if w > maxLabelW {
			maxLabelW = w
		}
		if h > maxLabelH {
			maxLabelH = h
		}
	}

	swatchSize := 15.0
	spacingX := 10.0
	spacingY := 5.0
	padOut := 10.0

	cellW := swatchSize + 5 + maxLabelW
	cellH := maxLabelH
	if swatchSize > cellH {
		cellH = swatchSize
	}

	rows := (len(l.Swatches) + cols - 1) / cols

	totalW := (float64(cols) * cellW) + (float64(cols-1) * spacingX) + (padOut * 2)
	totalH := (float64(rows) * cellH) + (float64(rows-1) * spacingY) + (padOut * 2)

	if l.Title != "" {
		w, h := m.MeasureText(l.Title, th.Typography.Legend)
		if w+padOut*2 > totalW {
			totalW = w + padOut*2
		}
		totalH += h + spacingY
	}

	return totalW, totalH
}

func (l Legend) Draw(p Painter, x, y, width, height float64, th theme.Theme) {
	if len(l.Swatches) == 0 {
		return
	}

	cols := l.Columns
	if cols <= 0 {
		cols = 1
	}

	padOut := 10.0
	swatchSize := 15.0
	spacingX := 10.0
	spacingY := 5.0

	titleColor := th.Typography.Legend.Color
	textColor := th.Typography.Legend.Color

	currY := y + padOut
	if l.Title != "" {
		p.DrawText(l.Title, x+padOut, currY, "left", "top", th.Typography.Legend, titleColor, 0.0)
		currY += th.Typography.Legend.Size*1.5 + spacingY
	}

	cellW := (width - (padOut * 2) - (float64(cols-1) * spacingX)) / float64(cols)
	cellH := swatchSize

	col := 0
	currX := x + padOut

	for _, s := range l.Swatches {
		fill := theme.ParseHexColor(s.Fill)
		stroke := theme.ParseHexColor(s.Stroke)

		p.DrawRect(currX, currY, swatchSize, swatchSize, fill, stroke, 1.0)
		p.DrawText(s.Label, currX+swatchSize+5, currY+(swatchSize/2.0), "left", "middle", th.Typography.Legend, textColor, 0.0)

		col++
		if col >= cols {
			col = 0
			currX = x + padOut
			currY += cellH + spacingY
		} else {
			currX += cellW + spacingX
		}
	}
}
