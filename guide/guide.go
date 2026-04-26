// Package guide provides axis, legend, and title rendering for plots.
// Guides are the non-data visual elements that border the plotting area
// and help readers interpret the data.
package guide

import (
	"fmt"
	"image/color"
	"math"

	"github.com/TuSKan/ggplot/colormap"
	"github.com/TuSKan/ggplot/internal/canvas"
	"github.com/TuSKan/ggplot/scale"
	"github.com/TuSKan/ggplot/theme"
	"github.com/gogpu/gg"
)

// DrawXAxis renders a horizontal axis at the bottom of the data area.
func DrawXAxis(cv canvas.Canvas, sc scale.Scale, label string, x, y, w float64, th theme.Theme) {
	ticks := sc.Ticks(5)
	tickLen := th.Ticks.Length
	r, g, b, _ := rgbaOf(th.Ticks.Color)
	tr, tg, tb, _ := rgbaOf(th.Text.TickLabel.Color)

	// Baseline.
	cv.SetRGBA(r, g, b, 1)
	cv.SetLineWidth(th.Ticks.Width)
	cv.DrawLine(x, y, x+w, y)
	cv.Stroke()

	xMin, xMax := sc.Bounds()
	for _, v := range ticks {
		frac := (v - xMin) / (xMax - xMin)
		if frac < 0 || frac > 1 {
			continue
		}
		px := x + frac*w

		// Tick mark.
		cv.SetRGBA(r, g, b, 1)
		cv.DrawLine(px, y, px, y+tickLen)
		cv.Stroke()

		// Label.
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.Text.TickLabel.Size)
		cv.DrawStringAnchored(sc.Format(v), px, y+tickLen+12, 0.5, 0.5)
	}

	// Axis title.
	if label != "" {
		lr, lg, lb, _ := rgbaOf(th.Text.AxisTitle.Color)
		cv.SetRGBA(lr, lg, lb, 1)
		cv.SetFontSize(th.Text.AxisTitle.Size)
		cv.DrawStringAnchored(label, x+w/2, y+tickLen+28, 0.5, 0.5)
	}
}

// DrawYAxis renders a vertical axis at the left of the data area.
func DrawYAxis(cv canvas.Canvas, sc scale.Scale, label string, x, y, h float64, th theme.Theme) {
	ticks := sc.Ticks(5)
	tickLen := th.Ticks.Length
	r, g, b, _ := rgbaOf(th.Ticks.Color)
	tr, tg, tb, _ := rgbaOf(th.Text.TickLabel.Color)

	// Baseline.
	cv.SetRGBA(r, g, b, 1)
	cv.SetLineWidth(th.Ticks.Width)
	cv.DrawLine(x, y, x, y+h)
	cv.Stroke()

	yMin, yMax := sc.Bounds()
	minSpacing := th.Text.TickLabel.Size + 4 // minimum px between labels
	lastPy := -1000.0                        // track last drawn label position
	for _, v := range ticks {
		frac := (v - yMin) / (yMax - yMin)
		if frac < 0 || frac > 1 {
			continue
		}
		py := y + h - frac*h // invert y

		// Tick mark (always drawn).
		cv.SetRGBA(r, g, b, 1)
		cv.DrawLine(x, py, x-tickLen, py)
		cv.Stroke()

		// Label — skip if too close to previous label.
		if math.Abs(py-lastPy) >= minSpacing {
			cv.SetRGBA(tr, tg, tb, 1)
			cv.SetFontSize(th.Text.TickLabel.Size)
			cv.DrawStringAnchored(sc.Format(v), x-tickLen-5, py, 1.0, 0.5)
			lastPy = py
		}
	}

	// Axis title (rotated).
	if label != "" {
		lr, lg, lb, _ := rgbaOf(th.Text.AxisTitle.Color)
		cv.SetRGBA(lr, lg, lb, 1)
		cv.SetFontSize(th.Text.AxisTitle.Size)
		cv.Save()
		cv.Translate(x-tickLen-30, y+h/2)
		cv.Rotate(-math.Pi / 2)
		cv.DrawStringAnchored(label, 0, 0, 0.5, 0.5)
		cv.Restore()
	}
}

// DrawGrid renders major grid lines in the data area.
// Grid lines use the theme's DashPattern (nil = solid, e.g. {4,4} = dashed).
func DrawGrid(cv canvas.Canvas, xScale, yScale scale.Scale, x, y, w, h float64, th theme.Theme) {
	mr, mg, mb, ma := rgbaOf(th.Grid.MajorColor)
	cv.SetRGBA(mr, mg, mb, ma)
	cv.SetLineWidth(th.Grid.MajorWidth)

	// Apply dash pattern from theme.
	if len(th.Grid.DashPattern) > 0 {
		cv.SetLineDash(th.Grid.DashPattern...)
	}

	// Vertical grid lines (from x ticks).
	xMin, xMax := xScale.Bounds()
	for _, v := range xScale.Ticks(5) {
		frac := (v - xMin) / (xMax - xMin)
		if frac < 0 || frac > 1 {
			continue
		}
		px := x + frac*w
		cv.DrawLine(px, y, px, y+h)
		cv.Stroke()
	}

	// Horizontal grid lines (from y ticks).
	yMin, yMax := yScale.Bounds()
	for _, v := range yScale.Ticks(5) {
		frac := (v - yMin) / (yMax - yMin)
		if frac < 0 || frac > 1 {
			continue
		}
		py := y + h - frac*h
		cv.DrawLine(x, py, x+w, py)
		cv.Stroke()
	}

	// Reset to solid lines.
	cv.SetLineDash()
}

// --- Legend ---

// LegendEntry describes one item in the legend.
type LegendEntry struct {
	Label string
	Color gg.RGBA
}

// DrawLegend renders a categorical legend to the right of the data area.
func DrawLegend(cv canvas.Canvas, title string, entries []LegendEntry, x, y float64, th theme.Theme) {
	if len(entries) == 0 {
		return
	}

	swatchSize := 12.0
	spacing := 20.0
	curY := y

	if title != "" {
		r, g, b, _ := rgbaOf(th.Text.Legend.Color)
		cv.SetRGBA(r, g, b, 1)
		cv.SetFontSize(th.Text.Legend.Size)
		cv.DrawStringAnchored(title, x+swatchSize+5, curY, 0, 0.5)
		curY += spacing
	}

	for _, e := range entries {
		cv.SetRGBA(e.Color.R, e.Color.G, e.Color.B, e.Color.A)
		cv.DrawRectangle(x, curY-swatchSize/2, swatchSize, swatchSize)
		cv.Fill()

		r, g, b, _ := rgbaOf(th.Text.Legend.Color)
		cv.SetRGBA(r, g, b, 1)
		cv.SetFontSize(th.Text.Legend.Size)
		cv.DrawStringAnchored(e.Label, x+swatchSize+5, curY, 0, 0.5)

		curY += spacing
	}
}

// ColorBarSpec describes a continuous color bar legend.
//
// Cmap and Norm replace the previous opaque ColorFunc field: the bar walks
// Cmap.At directly across the [0,1] range, and Norm provides the data-space
// labels at the endpoints (and any future intermediate ticks).
type ColorBarSpec struct {
	Title string
	Cmap  colormap.Cmap
	Norm  colormap.Norm
}

// DrawColorBar renders a continuous color bar legend at the given position.
// The bar is drawn vertically (top = max, bottom = min) as in ggplot2.
func DrawColorBar(cv canvas.Canvas, spec ColorBarSpec, x, y, barH float64, th theme.Theme) {
	barW := 12.0

	// Title above the bar.
	tr, tg, tb, _ := rgbaOf(th.Text.Legend.Color)
	if spec.Title != "" {
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.Text.Legend.Size)
		cv.DrawStringAnchored(spec.Title, x+barW/2, y, 0.5, 1.0)
		y += th.Text.Legend.Size + 6
	}

	cm := spec.Cmap
	if cm == nil {
		cm = colormap.Viridis
	}

	// Draw gradient bar as thin horizontal strips (top = max, bottom = min).
	nStrips := int(barH)
	if nStrips < 2 {
		nStrips = 2
	}
	stripH := barH / float64(nStrips)
	for i := 0; i < nStrips; i++ {
		// t=1 at top (max), t=0 at bottom (min).
		t := 1.0 - float64(i)/float64(nStrips-1)
		c := cm.At(t)
		cv.SetRGBA(c.R, c.G, c.B, c.A)
		cv.DrawRectangle(x, y+float64(i)*stripH, barW, stripH+0.5)
		cv.Fill()
	}

	// Outline.
	cv.SetRGBA(tr, tg, tb, 0.5)
	cv.SetLineWidth(0.5)
	cv.DrawRectangle(x, y, barW, barH)
	cv.Stroke()

	// Max label (top) and Min label (bottom). Use the Norm's data-space
	// bounds when available; otherwise fall back to "high" / "low".
	cv.SetRGBA(tr, tg, tb, 1)
	cv.SetFontSize(th.Text.Legend.Size * 0.9)
	labelX := x + barW + 4
	hi, lo := "high", "low"
	if spec.Norm != nil {
		vmin, vmax := spec.Norm.Bounds()
		hi = formatNum(vmax)
		lo = formatNum(vmin)
	}
	cv.DrawStringAnchored(hi, labelX, y+4, 0, 0.5)
	cv.DrawStringAnchored(lo, labelX, y+barH-4, 0, 0.5)
}

func formatNum(v float64) string {
	if v == float64(int(v)) {
		return fmt.Sprintf("%.1f", v)
	}
	return fmt.Sprintf("%.2g", v)
}

// DrawLegendHorizontal renders a categorical legend as a horizontal row.
func DrawLegendHorizontal(cv canvas.Canvas, title string, entries []LegendEntry, x, y, maxW float64, th theme.Theme) {
	if len(entries) == 0 {
		return
	}

	swatchSize := 10.0
	gap := 8.0
	curX := x
	tr, tg, tb, _ := rgbaOf(th.Text.Legend.Color)

	if title != "" {
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.Text.Legend.Size)
		cv.DrawStringAnchored(title, curX, y, 0, 0.5)
		tw, _ := cv.MeasureString(title)
		curX += tw + gap*2
	}

	for _, e := range entries {
		if curX > x+maxW {
			break
		}
		cv.SetRGBA(e.Color.R, e.Color.G, e.Color.B, e.Color.A)
		cv.DrawRectangle(curX, y-swatchSize/2, swatchSize, swatchSize)
		cv.Fill()

		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.Text.Legend.Size * 0.9)
		cv.DrawStringAnchored(e.Label, curX+swatchSize+3, y, 0, 0.5)
		tw, _ := cv.MeasureString(e.Label)
		curX += swatchSize + 3 + tw + gap
	}
}

// DrawColorBarHorizontal renders a horizontal continuous color bar legend.
func DrawColorBarHorizontal(cv canvas.Canvas, spec ColorBarSpec, x, y, barW float64, th theme.Theme) {
	barH := 10.0

	tr, tg, tb, _ := rgbaOf(th.Text.Legend.Color)

	// Title to the left.
	startX := x
	if spec.Title != "" {
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.Text.Legend.Size)
		cv.DrawStringAnchored(spec.Title, startX, y+barH/2, 0, 0.5)
		tw, _ := cv.MeasureString(spec.Title)
		startX += tw + 8
	}

	cm := spec.Cmap
	if cm == nil {
		cm = colormap.Viridis
	}

	availW := barW - (startX - x)
	if availW < 20 {
		availW = 20
	}

	// Draw gradient bar as thin vertical strips.
	nStrips := int(availW)
	if nStrips < 2 {
		nStrips = 2
	}
	stripW := availW / float64(nStrips)
	for i := 0; i < nStrips; i++ {
		t := float64(i) / float64(nStrips-1)
		c := cm.At(t)
		cv.SetRGBA(c.R, c.G, c.B, c.A)
		cv.DrawRectangle(startX+float64(i)*stripW, y, stripW+0.5, barH)
		cv.Fill()
	}

	// Outline.
	cv.SetRGBA(tr, tg, tb, 0.4)
	cv.SetLineWidth(0.5)
	cv.DrawRectangle(startX, y, availW, barH)
	cv.Stroke()

	// Min / Max labels.
	cv.SetRGBA(tr, tg, tb, 1)
	cv.SetFontSize(th.Text.Legend.Size * 0.85)
	lo, hi := "low", "high"
	if spec.Norm != nil {
		vmin, vmax := spec.Norm.Bounds()
		lo = formatNum(vmin)
		hi = formatNum(vmax)
	}
	cv.DrawStringAnchored(lo, startX, y+barH+10, 0.5, 0.5)
	cv.DrawStringAnchored(hi, startX+availW, y+barH+10, 0.5, 0.5)
}

// --- Helpers ---

func rgbaOf(c color.Color) (float64, float64, float64, float64) {
	if c == nil {
		return 0, 0, 0, 1
	}
	r, g, b, a := c.RGBA()
	return float64(r) / 65535.0, float64(g) / 65535.0, float64(b) / 65535.0, float64(a) / 65535.0
}
