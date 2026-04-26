// draw.go contains geometry rendering functions for the ggplot rendering pipeline.
// Each draw function maps data coordinates to pixel positions using a coordinate
// system and renders shapes via the canvas abstraction.
package ggplot

import (
	"fmt"
	"math"
	"sort"

	"github.com/TuSKan/ggplot/colormap"
	"github.com/TuSKan/ggplot/coord"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/internal/canvas"
	"github.com/TuSKan/ggplot/internal/grammar"
	"github.com/gogpu/gg"
)

// drawLayer dispatches rendering to the appropriate geom-specific draw function.
//
// groupColor (when non-nil) overrides the layer's Params.Color/Fill — used for
// categorical color groups where a single color applies to the whole layer.
//
// contColorCol (when non-empty) names a numeric column whose values feed into
// contScale.At(v) to produce a per-datum color (continuous color mapping).
// contScale must be non-nil if contColorCol is set.
func drawLayer(
	cv canvas.Canvas,
	c coord.Coord,
	ds dataset.Dataset,
	g geom.Layer,
	mapping grammar.AesMap,
	groupColor *gg.RGBA,
	contColorCol string,
	contScale *colormap.Scale,
	w, h, xMin, xMax, yMin, yMax float64,
) {
	xCol := mapping["x"]
	yCol := mapping["y"]

	// If a group colour was assigned, override the layer's fixed color.
	params := g.Params
	if groupColor != nil {
		hex := fmt.Sprintf("#%02X%02X%02X",
			uint8(groupColor.R*255+0.5),
			uint8(groupColor.G*255+0.5),
			uint8(groupColor.B*255+0.5))
		params.Color = hex
		if params.Fill == "" {
			params.Fill = hex
		}
	}

	switch g.Geom {
	case geom.TypePoint:
		drawPoints(cv, c, ds, xCol, yCol, contColorCol, contScale, w, h, xMin, xMax, yMin, yMax, params)
	case geom.TypeLine, geom.TypeSmooth:
		drawLine(cv, c, ds, xCol, yCol, contColorCol, contScale, w, h, xMin, xMax, yMin, yMax, params)
	case geom.TypeStep:
		drawStep(cv, c, ds, xCol, yCol, w, h, xMin, xMax, yMin, yMax, params)
	case geom.TypeBar, geom.TypeHistogram:
		drawBars(cv, c, ds, xCol, yCol, w, h, xMin, xMax, yMin, yMax, params)
	case geom.TypeArea:
		drawArea(cv, c, ds, xCol, yCol, w, h, xMin, xMax, yMin, yMax, params)
	case geom.TypeDensity:
		drawArea(cv, c, ds, xCol, yCol, w, h, xMin, xMax, yMin, yMax, params)
	case geom.TypeRug:
		drawRug(cv, c, ds, xCol, yCol, w, h, xMin, xMax, yMin, yMax, params)
	case geom.TypeHLine:
		drawHLine(cv, c, w, h, xMin, xMax, yMin, yMax, params)
	case geom.TypeVLine:
		drawVLine(cv, c, w, h, xMin, xMax, yMin, yMax, params)
	case geom.TypeText:
		drawText(cv, c, ds, xCol, yCol, mapping, w, h, xMin, xMax, yMin, yMax, params)
	case geom.TypeBoxPlot:
		drawBoxplot(cv, c, ds, w, h, xMin, xMax, yMin, yMax, params)
	}
}

func drawPoints(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol, contColorCol string, contScale *colormap.Scale, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	xVals := getFloat64Values(ds, xCol)
	yVals := getFloat64Values(ds, yCol)
	if xVals == nil || yVals == nil {
		return
	}

	r := p.Size
	if r <= 0 {
		r = 3
	}
	alpha := p.Alpha
	if alpha <= 0 {
		alpha = 1
	}

	// Continuous color: read z values through the scale.
	zVals := getFloat64Values(ds, contColorCol)
	cr, cg, cb := resolveColor(p.Color, 0.3, 0.5, 0.8)

	n := len(xVals)
	if len(yVals) < n {
		n = len(yVals)
	}
	for i := 0; i < n; i++ {
		nx := normalize(xVals[i], xMin, xMax)
		ny := normalize(yVals[i], yMin, yMax)
		px, py := c.Transform(nx, ny, w, h)

		if i < len(zVals) && contScale != nil {
			gc := contScale.At(zVals[i])
			cv.SetRGBA(gc.R, gc.G, gc.B, alpha)
		} else {
			cv.SetRGBA(cr, cg, cb, alpha)
		}
		cv.DrawCircle(px, py, r)
		cv.Fill()
	}
}

func drawLine(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol, contColorCol string, contScale *colormap.Scale, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	xVals := getFloat64Values(ds, xCol)
	yVals := getFloat64Values(ds, yCol)
	if xVals == nil || yVals == nil {
		return
	}

	lw := p.LineWidth
	if lw <= 0 {
		lw = 2
	}
	alpha := p.Alpha
	if alpha <= 0 {
		alpha = 1.0
	}

	// Collect all points.
	type linePt struct{ x, y float64 }
	n := len(xVals)
	if len(yVals) < n {
		n = len(yVals)
	}
	pts := make([]linePt, n)
	for i := 0; i < n; i++ {
		pts[i] = linePt{xVals[i], yVals[i]}
	}
	if len(pts) < 2 {
		return
	}

	// Continuous color samples the scale at each segment midpoint.
	zVals := getFloat64Values(ds, contColorCol)

	cv.SetLineWidth(lw)

	if len(zVals) >= len(pts) && contScale != nil {
		// Per-segment gradient coloring.
		for i := 1; i < len(pts); i++ {
			nx0 := normalize(pts[i-1].x, xMin, xMax)
			ny0 := normalize(pts[i-1].y, yMin, yMax)
			nx1 := normalize(pts[i].x, xMin, xMax)
			ny1 := normalize(pts[i].y, yMin, yMax)
			px0, py0 := c.Transform(nx0, ny0, w, h)
			px1, py1 := c.Transform(nx1, ny1, w, h)

			zMid := (zVals[i-1] + zVals[i]) / 2
			gc := contScale.At(zMid)
			cv.SetRGBA(gc.R, gc.G, gc.B, alpha)
			cv.DrawLine(px0, py0, px1, py1)
			cv.Stroke()
		}
	} else {
		// Uniform color polyline.
		cr, cg, cb := resolveColor(p.Color, 0.8, 0.2, 0.2)
		cv.SetRGBA(cr, cg, cb, alpha)
		nx := normalize(pts[0].x, xMin, xMax)
		ny := normalize(pts[0].y, yMin, yMax)
		px, py := c.Transform(nx, ny, w, h)
		cv.MoveTo(px, py)
		for i := 1; i < len(pts); i++ {
			nx = normalize(pts[i].x, xMin, xMax)
			ny = normalize(pts[i].y, yMin, yMax)
			px, py = c.Transform(nx, ny, w, h)
			cv.LineTo(px, py)
		}
		cv.Stroke()
	}
}

func drawBars(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol string, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	// Collect all points first so we can compute spacing.
	type barPt struct{ x, y float64 }
	var pts []barPt

	xVals := getFloat64Values(ds, xCol)
	if xVals == nil {
		return
	}

	yVals := getFloat64Values(ds, yCol)
	if yVals == nil {
		yVals = getFloat64Values(ds, "count")
	}

	for i, x := range xVals {
		y := 1.0
		if yVals != nil && i < len(yVals) {
			y = yVals[i]
		}
		pts = append(pts, barPt{x, y})
	}

	if len(pts) == 0 {
		return
	}

	// Compute half-bar-width in pixel space.
	dataRange := xMax - xMin
	if dataRange <= 0 {
		dataRange = 1
	}

	// Sort by X so spacing calculation works correctly.
	sort.Slice(pts, func(i, j int) bool { return pts[i].x < pts[j].x })

	// Determine the minimum spacing between adjacent X values.
	var minSpacing float64
	if len(pts) > 1 {
		minSpacing = math.Abs(pts[1].x - pts[0].x)
		for i := 2; i < len(pts); i++ {
			sp := math.Abs(pts[i].x - pts[i-1].x)
			if sp > 0 && sp < minSpacing {
				minSpacing = sp
			}
		}
	} else {
		minSpacing = dataRange
	}
	if minSpacing <= 0 {
		minSpacing = 1
	}

	relW := p.Width
	if relW <= 0 || relW > 1 {
		relW = 0.8 // default = 20% gap
	}

	// Convert spacing to pixels, apply relative width.
	// When flipped, X data maps to vertical axis → use h instead of w.
	barAxisLen := w
	if c.IsFlipped() {
		barAxisLen = h
	}
	pixPerUnit := barAxisLen / dataRange
	halfBarPx := (minSpacing * pixPerUnit * relW) / 2
	if halfBarPx < 1 {
		halfBarPx = 1
	}

	alpha := p.Alpha
	if alpha <= 0 {
		alpha = 0.85
	}
	fr, fg, fb := resolveColor(p.Fill, 0.2, 0.4, 0.8)

	// Baseline Y in normalized space (Y=0).
	baseNy := normalize(0, yMin, yMax)

	for _, pt := range pts {
		nx := normalize(pt.x, xMin, xMax)
		ny := normalize(pt.y, yMin, yMax)

		// Get center and baseline pixel positions via coord.Transform.
		cx, cyVal := c.Transform(nx, ny, w, h)
		cxBase, cyBase := c.Transform(nx, baseNy, w, h)

		var rx, ry, rw, rh float64
		if c.IsFlipped() {
			// When flipped, X data maps to vertical axis.
			// Bar width offset is vertical, bar length is horizontal.
			rx = math.Min(cx, cxBase)
			ry = cyVal - halfBarPx
			rw = math.Abs(cx - cxBase)
			rh = halfBarPx * 2
		} else {
			// Normal: bar width offset is horizontal.
			rx = cx - halfBarPx
			ry = math.Min(cyVal, cyBase)
			rw = halfBarPx * 2
			rh = math.Abs(cyVal - cyBase)
		}

		cv.SetRGBA(fr, fg, fb, alpha)
		cv.DrawRectangle(rx, ry, rw, rh)
		cv.FillPreserve()

		// Outline: slightly darker than fill.
		cv.SetRGBA(fr*0.7, fg*0.7, fb*0.7, alpha)
		cv.SetLineWidth(0.5)
		cv.Stroke()
	}
}

func drawArea(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol string, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	xVals := getFloat64Values(ds, xCol)
	yVals := getFloat64Values(ds, yCol)
	if xVals == nil || yVals == nil {
		return
	}

	// Collect normalized data points and their corresponding baselines.
	type normPt struct{ nx, ny float64 }
	n := len(xVals)
	if len(yVals) < n {
		n = len(yVals)
	}
	npts := make([]normPt, n)
	for i := 0; i < n; i++ {
		npts[i] = normPt{
			nx: normalize(xVals[i], xMin, xMax),
			ny: normalize(yVals[i], yMin, yMax),
		}
	}

	if len(npts) < 2 {
		return
	}

	// Baseline: Y=0 in normalized space.
	baseNy := normalize(0, yMin, yMax)

	// Build the polygon path: baseline start → data points → baseline end.
	bx0, by0 := c.Transform(npts[0].nx, baseNy, w, h)
	cv.MoveTo(bx0, by0)
	for _, np := range npts {
		px, py := c.Transform(np.nx, np.ny, w, h)
		cv.LineTo(px, py)
	}
	// Close back to baseline at the last X position.
	bxN, byN := c.Transform(npts[len(npts)-1].nx, baseNy, w, h)
	cv.LineTo(bxN, byN)
	cv.ClosePath()

	alpha := p.Alpha
	if alpha <= 0 {
		alpha = 0.6
	}
	fr, fg, fb := resolveColor(p.Fill, 0.1, 0.7, 0.3)
	cv.SetRGBA(fr, fg, fb, alpha)
	cv.FillPreserve()

	cr, cg, cb := resolveColor(p.Color, 0.0, 0.0, 0.0)
	cv.SetRGBA(cr, cg, cb, 1.0)
	cv.SetLineWidth(1)
	cv.Stroke()
}

func drawStep(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol string, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	xVals := getFloat64Values(ds, xCol)
	yVals := getFloat64Values(ds, yCol)
	if xVals == nil || yVals == nil {
		return
	}

	lw := p.LineWidth
	if lw <= 0 {
		lw = 2
	}
	alpha := p.Alpha
	if alpha <= 0 {
		alpha = 1
	}
	cr, cg, cb := resolveColor(p.Color, 0.3, 0.5, 0.8)

	// Collect normalized data points.
	type normPt struct{ nx, ny float64 }
	n := len(xVals)
	if len(yVals) < n {
		n = len(yVals)
	}
	npts := make([]normPt, n)
	for i := 0; i < n; i++ {
		npts[i] = normPt{
			nx: normalize(xVals[i], xMin, xMax),
			ny: normalize(yVals[i], yMin, yMax),
		}
	}

	if len(npts) < 2 {
		return
	}

	cv.SetRGBA(cr, cg, cb, alpha)
	cv.SetLineWidth(lw)

	// Step function in normalized space: advance X to next point, keep old Y,
	// then advance Y. Transform each intermediate point through coord.
	px0, py0 := c.Transform(npts[0].nx, npts[0].ny, w, h)
	cv.MoveTo(px0, py0)
	for i := 1; i < len(npts); i++ {
		// Horizontal step: move to next X, keep previous Y.
		sx, sy := c.Transform(npts[i].nx, npts[i-1].ny, w, h)
		cv.LineTo(sx, sy)
		// Vertical step: now move Y to current value.
		vx, vy := c.Transform(npts[i].nx, npts[i].ny, w, h)
		cv.LineTo(vx, vy)
	}
	cv.Stroke()
}

func drawRug(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol string, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	lw := p.LineWidth
	if lw <= 0 {
		lw = 0.5
	}
	alpha := p.Alpha
	if alpha <= 0 {
		alpha = 0.5
	}
	cr, cg, cb := resolveColor(p.Color, 0.2, 0.2, 0.2)

	const rugFrac = 0.02 // 2% of panel extent

	// X rugs: tick marks along the bottom edge of the panel.
	if xVals := getFloat64Values(ds, xCol); xVals != nil {
		cv.SetRGBA(cr, cg, cb, alpha)
		cv.SetLineWidth(lw)
		for _, x := range xVals {
			nx := normalize(x, xMin, xMax)
			px1, py1 := c.Transform(nx, 0, w, h)
			px2, py2 := c.Transform(nx, rugFrac, w, h)
			cv.MoveTo(px1, py1)
			cv.LineTo(px2, py2)
			cv.Stroke()
		}
	}

	// Y rugs: tick marks along the left edge of the panel.
	if yVals := getFloat64Values(ds, yCol); yVals != nil {
		cv.SetRGBA(cr, cg, cb, alpha)
		cv.SetLineWidth(lw)
		for _, y := range yVals {
			ny := normalize(y, yMin, yMax)
			px1, py1 := c.Transform(0, ny, w, h)
			px2, py2 := c.Transform(rugFrac, ny, w, h)
			cv.MoveTo(px1, py1)
			cv.LineTo(px2, py2)
			cv.Stroke()
		}
	}
}

func drawHLine(cv canvas.Canvas, c coord.Coord, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	ny := normalize(p.Intercept, yMin, yMax)
	if ny < 0 || ny > 1 {
		return // outside visible range
	}

	lw := p.LineWidth
	if lw <= 0 {
		lw = 1
	}
	alpha := p.Alpha
	if alpha <= 0 {
		alpha = 0.8
	}
	cr, cg, cb := resolveColor(p.Color, 0.6, 0.1, 0.1)

	// Draw from left (X=0) to right (X=1) in normalized space.
	px1, py1 := c.Transform(0, ny, w, h)
	px2, py2 := c.Transform(1, ny, w, h)

	cv.SetRGBA(cr, cg, cb, alpha)
	cv.SetLineWidth(lw)
	cv.MoveTo(px1, py1)
	cv.LineTo(px2, py2)
	cv.Stroke()
}

func drawVLine(cv canvas.Canvas, c coord.Coord, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	nx := normalize(p.Intercept, xMin, xMax)
	if nx < 0 || nx > 1 {
		return // outside visible range
	}

	lw := p.LineWidth
	if lw <= 0 {
		lw = 1
	}
	alpha := p.Alpha
	if alpha <= 0 {
		alpha = 0.8
	}
	cr, cg, cb := resolveColor(p.Color, 0.1, 0.1, 0.6)

	// Draw from bottom (Y=0) to top (Y=1) in normalized space.
	px1, py1 := c.Transform(nx, 0, w, h)
	px2, py2 := c.Transform(nx, 1, w, h)

	cv.SetRGBA(cr, cg, cb, alpha)
	cv.SetLineWidth(lw)
	cv.MoveTo(px1, py1)
	cv.LineTo(px2, py2)
	cv.Stroke()
}

func drawText(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol string, mapping grammar.AesMap, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	xVals := getFloat64Values(ds, xCol)
	yVals := getFloat64Values(ds, yCol)
	if xVals == nil || yVals == nil {
		return
	}

	// Determine label column from the "label" aesthetic mapping.
	labelCol := mapping["label"]
	if labelCol == "" {
		labelCol = "label" // fallback for backwards compat
	}
	var labels []string
	if col, err := ds.Column(labelCol); err == nil {
		if sc, ok := col.(dataset.Column[string]); ok {
			labels = sc.Values()
		}
	}

	fontSize := p.FontSize
	if fontSize <= 0 {
		fontSize = 10
	}
	alpha := p.Alpha
	if alpha <= 0 {
		alpha = 1
	}
	cr, cg, cb := resolveColor(p.Color, 0.1, 0.1, 0.1)

	cv.SetFontSize(fontSize)
	cv.SetRGBA(cr, cg, cb, alpha)

	n := len(xVals)
	if len(yVals) < n {
		n = len(yVals)
	}
	for i := 0; i < n; i++ {
		nx := normalize(xVals[i], xMin, xMax)
		ny := normalize(yVals[i], yMin, yMax)
		px, py := c.Transform(nx, ny, w, h)

		var lbl string
		if i < len(labels) {
			lbl = labels[i]
		} else {
			lbl = fmt.Sprintf("%.4g", yVals[i])
		}

		cv.SetRGBA(cr, cg, cb, alpha)
		cv.DrawStringAnchored(lbl, px, py-fontSize-4, 0.5, 1.0)
	}
}

func drawBoxplot(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	// Read stat_boxplot output columns.
	xVals := getFloat64Values(ds, "x")
	lowerVals := getFloat64Values(ds, "lower")
	q1Vals := getFloat64Values(ds, "q1")
	midVals := getFloat64Values(ds, "middle")
	q3Vals := getFloat64Values(ds, "q3")
	upperVals := getFloat64Values(ds, "upper")
	if xVals == nil || lowerVals == nil || q1Vals == nil || midVals == nil || q3Vals == nil || upperVals == nil {
		return
	}

	boxW := p.Width
	if boxW <= 0 || boxW > 1 {
		boxW = 0.5
	}
	alpha := p.Alpha
	if alpha <= 0 {
		alpha = 0.8
	}
	lw := p.LineWidth
	if lw <= 0 {
		lw = 1.5
	}
	fr, fg, fb := resolveColor(p.Fill, 0.9, 0.9, 0.9)
	cr, cg, cb := resolveColor(p.Color, 0.2, 0.2, 0.2)

	// Compute half-box-width in pixel space.
	xRange := xMax - xMin
	if xRange <= 0 {
		xRange = 1
	}
	pixPerUnit := w / xRange

	groupSpacing := 1.0
	halfPixel := (groupSpacing * pixPerUnit * boxW) / 2

	maxHalf := math.Min(w*0.08, 40)
	if halfPixel > maxHalf {
		halfPixel = maxHalf
	}
	if halfPixel < 3 {
		halfPixel = 3
	}

	n := len(xVals)
	for i := 0; i < n; i++ {
		if i >= len(lowerVals) || i >= len(q1Vals) || i >= len(midVals) || i >= len(q3Vals) || i >= len(upperVals) {
			break
		}

		nx := normalize(xVals[i], xMin, xMax)
		nLo := normalize(lowerVals[i], yMin, yMax)
		nQ1 := normalize(q1Vals[i], yMin, yMax)
		nMd := normalize(midVals[i], yMin, yMax)
		nQ3 := normalize(q3Vals[i], yMin, yMax)
		nHi := normalize(upperVals[i], yMin, yMax)

		// Center X pixel and Y positions via coord.Transform.
		cx, _ := c.Transform(nx, 0.5, w, h)
		_, pyLo := c.Transform(nx, nLo, w, h)
		_, pyQ1 := c.Transform(nx, nQ1, w, h)
		_, pyMd := c.Transform(nx, nMd, w, h)
		_, pyQ3 := c.Transform(nx, nQ3, w, h)
		_, pyHi := c.Transform(nx, nHi, w, h)

		// Box rectangle in pixel space.
		rx := cx - halfPixel
		ry := math.Min(pyQ1, pyQ3)
		rw := halfPixel * 2
		rh := math.Abs(pyQ3 - pyQ1)

		// Fill box.
		cv.SetRGBA(fr, fg, fb, alpha)
		cv.DrawRectangle(rx, ry, rw, rh)
		cv.Fill()

		// Box outline.
		cv.SetRGBA(cr, cg, cb, 1.0)
		cv.SetLineWidth(lw)
		cv.DrawRectangle(rx, ry, rw, rh)
		cv.Stroke()

		// Median line.
		cv.SetLineWidth(lw * 1.5)
		cv.MoveTo(cx-halfPixel, pyMd)
		cv.LineTo(cx+halfPixel, pyMd)
		cv.Stroke()

		// Lower whisker: center line from lower to q1.
		cv.SetLineWidth(lw)
		cv.MoveTo(cx, pyLo)
		cv.LineTo(cx, pyQ1)
		cv.Stroke()

		// Upper whisker: center line from q3 to upper.
		cv.MoveTo(cx, pyQ3)
		cv.LineTo(cx, pyHi)
		cv.Stroke()

		// Whisker caps (horizontal bars at lower and upper).
		capHalf := halfPixel * 0.4
		cv.MoveTo(cx-capHalf, pyLo)
		cv.LineTo(cx+capHalf, pyLo)
		cv.Stroke()

		cv.MoveTo(cx-capHalf, pyHi)
		cv.LineTo(cx+capHalf, pyHi)
		cv.Stroke()
	}
}
