// draw.go provides geometry rendering functions for the plotting pipeline.
// Each draw function maps data coordinates to pixel positions using a coordinate
// system and renders shapes via the canvas abstraction.
//
// The Drawer interface enables extensible geometry dispatch: third-party geoms
// can register their own draw functions via [RegisterDrawer].

package ggplot

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/colormap"
	"github.com/TuSKan/ggplot/coord"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/scale"
	"github.com/TuSKan/ggplot/theme"
)

// DrawContext holds the rendering parameters passed to a [Drawer].
// It encapsulates the canvas, coordinate system, data, aesthetic mappings,
// and panel bounds so that Drawer implementations are self-contained.
type DrawContext struct {
	Canvas        canvas.Canvas
	Coord         coord.Coord
	Data          dataset.Dataset
	Mapping       AesMap
	Params        geom.Params
	Theme         theme.Theme     // active theme for default styling
	ContColorCol  string          // continuous color column (empty if none)
	ContScale     *colormap.Scale // continuous color scale (nil if none)
	W, H          float64         // panel size in pixels
	XMin, XMax    float64         // data domain bounds
	YMin, YMax    float64         // data domain bounds
	SizeCol       string
	SizeScale     scale.Scale
	AlphaCol      string
	AlphaScale    scale.Scale
	ShapeCol      string
	ShapeScale    scale.Scale
	LinetypeCol   string
	LinetypeScale scale.Scale
}

// Drawer renders a geometry type onto the canvas. Implementations are
// registered via [RegisterDrawer] and looked up by [geom.Type] during rendering.
type Drawer interface {
	Draw(ctx DrawContext)
}

// DrawerFunc is an adapter to allow use of ordinary functions as [Drawer]s.
type DrawerFunc func(DrawContext)

// Draw calls f(ctx).
func (f DrawerFunc) Draw(ctx DrawContext) { f(ctx) }

// --- Registry ---

var drawers = map[geom.Type]Drawer{}

// RegisterDrawer registers a [Drawer] for a geometry type. Replaces any
// previously registered drawer for the same type. Third-party geom types
// should call this at init() time.
func RegisterDrawer(t geom.Type, d Drawer) { drawers[t] = d }

// LookupDrawer returns the registered [Drawer] for the given type, or nil.
func LookupDrawer(t geom.Type) Drawer { return drawers[t] }

func init() {
	RegisterDrawer(geom.TypePoint, DrawerFunc(drawPointsFn))
	RegisterDrawer(geom.TypeLine, DrawerFunc(drawLineFn))
	RegisterDrawer(geom.TypeSmooth, DrawerFunc(drawLineFn))
	RegisterDrawer(geom.TypeStep, DrawerFunc(drawStepFn))
	RegisterDrawer(geom.TypeBar, DrawerFunc(drawBarsFn))
	RegisterDrawer(geom.TypeHistogram, DrawerFunc(drawHistogramFn))
	RegisterDrawer(geom.TypeArea, DrawerFunc(drawAreaFn))
	RegisterDrawer(geom.TypeDensity, DrawerFunc(drawAreaFn))
	RegisterDrawer(geom.TypeRug, DrawerFunc(drawRugFn))
	RegisterDrawer(geom.TypeHLine, DrawerFunc(drawHLineFn))
	RegisterDrawer(geom.TypeVLine, DrawerFunc(drawVLineFn))
	RegisterDrawer(geom.TypeABLine, DrawerFunc(drawABLineFn))
	RegisterDrawer(geom.TypeText, DrawerFunc(drawTextFn))
	RegisterDrawer(geom.TypeBoxPlot, DrawerFunc(drawBoxplotFn))
	RegisterDrawer(geom.TypeTile, DrawerFunc(drawTileFn))
	RegisterDrawer(geom.TypeSegment, DrawerFunc(drawSegmentFn))
	RegisterDrawer(geom.TypeErrorBar, DrawerFunc(drawErrorBarFn))
	RegisterDrawer(geom.TypePolygon, DrawerFunc(drawPolygonFn))
	RegisterDrawer(geom.TypeRibbon, DrawerFunc(drawRibbonFn))
	RegisterDrawer(geom.TypeDifference, DrawerFunc(drawDifferenceFn))
	RegisterDrawer(geom.TypeRect, DrawerFunc(drawRectFn))
}

// drawLayer dispatches rendering to the registered Drawer for the layer's geom type.
//
// contColorCol (when non-empty) names a numeric column whose values feed into
// contScale.At(v) to produce a per-datum color (continuous color mapping).
// contScale must be non-nil if contColorCol is set.
//
// Group colors are baked into g.Params.Color during the build phase, so
// drawLayer does not need a separate groupColor parameter.
func drawLayer(
	cv canvas.Canvas,
	c coord.Coord,
	rl BuiltLayer,
	w, h, xMin, xMax, yMin, yMax float64,
	th theme.Theme,
) {
	d := LookupDrawer(rl.Geom.Geom)
	if d == nil {
		return // unknown geom type — silently skip (validated at construction)
	}

	d.Draw(DrawContext{
		Canvas:        cv,
		Coord:         c,
		Data:          rl.Data,
		Mapping:       rl.Mapping,
		Params:        rl.Geom.Params,
		Theme:         th,
		ContColorCol:  rl.ContColorCol,
		ContScale:     rl.ContColScale,
		W:             w,
		H:             h,
		XMin:          xMin,
		XMax:          xMax,
		YMin:          yMin,
		YMax:          yMax,
		SizeCol:       rl.SizeCol,
		SizeScale:     rl.SizeScale,
		AlphaCol:      rl.AlphaCol,
		AlphaScale:    rl.AlphaScale,
		ShapeCol:      rl.ShapeCol,
		ShapeScale:    rl.ShapeScale,
		LinetypeCol:   rl.LinetypeCol,
		LinetypeScale: rl.LinetypeScale,
	})
}

// --- Drawer adapters: bridge DrawContext to existing draw functions ---

func drawPointsFn(dc DrawContext) {
	drawPoints(dc)
}

func drawLineFn(dc DrawContext) {
	drawLine(dc)
}

func drawStepFn(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	drawStep(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params)
}

func drawBarsFn(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	drawBars(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params, dc.Theme, 0)
}

func drawHistogramFn(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	// Half-pixel inset per side → 1px total gap between adjacent bins,
	// matching Observable Plot's continuous-bar default.
	drawBars(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params, dc.Theme, 0.5)
}

func drawRectFn(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	drawBars(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params, dc.Theme, dc.Params.Inset)
}

func drawAreaFn(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	drawArea(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params)
}

func drawRugFn(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	drawRug(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params)
}

func drawHLineFn(dc DrawContext) {
	drawHLine(dc.Canvas, dc.Coord, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params)
}

func drawVLineFn(dc DrawContext) {
	drawVLine(dc.Canvas, dc.Coord, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params)
}

func drawABLineFn(dc DrawContext) {
	drawABLine(dc.Canvas, dc.Coord, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params)
}

func drawTextFn(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	drawText(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.Mapping, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params)
}

func drawBoxplotFn(dc DrawContext) {
	drawBoxplot(dc.Canvas, dc.Coord, dc.Data, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params, dc.Theme)
}

func columnAsStrings(col dataset.AnyColumn) []string {
	if col == nil {
		return nil
	}

	switch tc := col.(type) {
	case dataset.Column[string]:
		return tc.Values()
	case dataset.Column[float64]:
		raw := tc.Values()
		vals := make([]string, len(raw))

		for i, v := range raw {
			vals[i] = strconv.FormatFloat(v, 'g', -1, 64)
		}

		return vals
	case dataset.Column[int64]:
		raw := tc.Values()
		vals := make([]string, len(raw))

		for i, v := range raw {
			vals[i] = strconv.FormatInt(v, 10)
		}

		return vals
	default:
		return nil
	}
}

func drawPoints(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	xVals, errX := dc.Data.Float64(xCol)
	yVals, errY := dc.Data.Float64(yCol)

	if errX != nil || errY != nil {
		return
	}

	r := dc.Params.Size
	if r <= 0 {
		r = 3
	}

	alpha := dc.Params.Alpha
	if alpha <= 0 {
		alpha = 1
	}

	// Size mapping
	var (
		sizeVals   []float64
		sizeMapper func(float64) float64
	)

	if dc.SizeCol != "" && dc.SizeScale != nil {
		sizeVals, _ = dc.Data.Float64(dc.SizeCol)
		if s, ok := dc.SizeScale.(scale.ValueMapper); ok {
			sizeMapper = s.MapValue
		} else {
			sizeMapper = dc.SizeScale.Map
		}
	}

	// Alpha mapping
	var (
		alphaVals   []float64
		alphaMapper func(float64) float64
	)

	if dc.AlphaCol != "" && dc.AlphaScale != nil {
		alphaVals, _ = dc.Data.Float64(dc.AlphaCol)
		if s, ok := dc.AlphaScale.(scale.ValueMapper); ok {
			alphaMapper = s.MapValue
		} else {
			alphaMapper = dc.AlphaScale.Map
		}
	}

	// Shape mapping
	var shapeVals []string

	if dc.ShapeCol != "" {
		if col, err := dc.Data.Column(dc.ShapeCol); err == nil {
			shapeVals = columnAsStrings(col)
		}
	}

	// Continuous color mapping
	var zVals []float64
	if dc.ContColorCol != "" {
		zVals, _ = dc.Data.Float64(dc.ContColorCol)
	}

	cr, cg, cb := colormap.ParseRGB(dc.Params.Color, 0.3, 0.5, 0.8)
	n := min(len(yVals), len(xVals))

	for i := range n {
		nx := normalize(xVals[i], dc.XMin, dc.XMax)
		ny := normalize(yVals[i], dc.YMin, dc.YMax)
		px, py := orientedTransform(dc.Coord, nx, ny, dc.W, dc.H, dc.Params.Orientation)

		ptAlpha := alpha
		if i < len(alphaVals) && alphaMapper != nil {
			ptAlpha = alphaMapper(alphaVals[i])
		}

		if i < len(zVals) && dc.ContScale != nil {
			gc := dc.ContScale.At(zVals[i])
			dc.Canvas.SetRGBA(gc.R, gc.G, gc.B, ptAlpha)
		} else {
			dc.Canvas.SetRGBA(cr, cg, cb, ptAlpha)
		}

		ptRadius := r
		if i < len(sizeVals) && sizeMapper != nil {
			ptRadius = sizeMapper(sizeVals[i])
		}

		shapeName := dc.Params.Shape
		if shapeName == "" {
			shapeName = canvas.ShapeCircle
		}

		if i < len(shapeVals) && dc.ShapeScale != nil {
			if s, ok := dc.ShapeScale.(*scale.ShapeScale); ok {
				shapeName = s.ShapeName(shapeVals[i])
			} else if _, ok := dc.ShapeScale.(*scale.IdentityScale); ok {
				shapeName = shapeVals[i]
			}
		}

		if canvas.IsStrokeShape(shapeName) {
			dc.Canvas.SetLineWidth(1.5)
			dc.Canvas.DrawShape(shapeName, px, py, ptRadius)
			dc.Canvas.Stroke()
		} else {
			dc.Canvas.DrawShape(shapeName, px, py, ptRadius)
			dc.Canvas.Fill()
		}
	}
}

func drawLine(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	xVals, errX := dc.Data.Float64(xCol)
	yVals, errY := dc.Data.Float64(yCol)

	if errX != nil || errY != nil {
		return
	}

	lw := dc.Params.LineWidth
	if lw <= 0 {
		lw = 2
	}

	alpha := dc.Params.Alpha
	if alpha <= 0 {
		alpha = 1.0
	}

	// Collect all points.
	type linePt struct{ x, y float64 }

	n := min(len(yVals), len(xVals))
	pts := make([]linePt, n)

	for i := range n {
		pts[i] = linePt{xVals[i], yVals[i]}
	}

	if len(pts) < 2 {
		return
	}

	// Linetype mapping (constant for this line/group)
	var dashPattern []float64

	if dc.LinetypeCol != "" && dc.LinetypeScale != nil {
		if col, err := dc.Data.Column(dc.LinetypeCol); err == nil {
			vals := columnAsStrings(col)
			if len(vals) > 0 {
				if s, ok := dc.LinetypeScale.(*scale.LinetypeScale); ok {
					dashPattern = s.DashPattern(vals[0])
				}
			}
		}
	}

	if len(dashPattern) > 0 {
		dc.Canvas.SetLineDash(dashPattern...)
		defer dc.Canvas.SetLineDash()
	}

	// Alpha mapping
	var (
		alphaVals   []float64
		alphaMapper func(float64) float64
	)

	if dc.AlphaCol != "" && dc.AlphaScale != nil {
		alphaVals, _ = dc.Data.Float64(dc.AlphaCol)
		if s, ok := dc.AlphaScale.(scale.ValueMapper); ok {
			alphaMapper = s.MapValue
		} else {
			alphaMapper = dc.AlphaScale.Map
		}
	}

	// Continuous color mapping
	var zVals []float64
	if dc.ContColorCol != "" {
		zVals, _ = dc.Data.Float64(dc.ContColorCol)
	}

	dc.Canvas.SetLineWidth(lw)

	if (len(zVals) >= len(pts) && dc.ContScale != nil) || len(alphaVals) >= len(pts) {
		// Draw segment-by-segment because either color or alpha is varying.
		cr, cg, cb := colormap.ParseRGB(dc.Params.Color, 0.8, 0.2, 0.2)
		for i := 1; i < len(pts); i++ {
			nx0 := normalize(pts[i-1].x, dc.XMin, dc.XMax)
			ny0 := normalize(pts[i-1].y, dc.YMin, dc.YMax)
			nx1 := normalize(pts[i].x, dc.XMin, dc.XMax)
			ny1 := normalize(pts[i].y, dc.YMin, dc.YMax)
			px0, py0 := orientedTransform(dc.Coord, nx0, ny0, dc.W, dc.H, dc.Params.Orientation)
			px1, py1 := orientedTransform(dc.Coord, nx1, ny1, dc.W, dc.H, dc.Params.Orientation)

			ptAlpha := alpha
			if i-1 < len(alphaVals) && alphaMapper != nil {
				ptAlpha = alphaMapper(alphaVals[i-1])
			}

			if i-1 < len(zVals) && dc.ContScale != nil {
				zMid := (zVals[i-1] + zVals[i]) / 2
				gc := dc.ContScale.At(zMid)
				dc.Canvas.SetRGBA(gc.R, gc.G, gc.B, ptAlpha)
			} else {
				dc.Canvas.SetRGBA(cr, cg, cb, ptAlpha)
			}

			dc.Canvas.DrawLine(px0, py0, px1, py1)
			dc.Canvas.Stroke()
		}
	} else {
		// Uniform color and alpha polyline.
		cr, cg, cb := colormap.ParseRGB(dc.Params.Color, 0.8, 0.2, 0.2)
		dc.Canvas.SetRGBA(cr, cg, cb, alpha)

		nx := normalize(pts[0].x, dc.XMin, dc.XMax)
		ny := normalize(pts[0].y, dc.YMin, dc.YMax)
		px, py := orientedTransform(dc.Coord, nx, ny, dc.W, dc.H, dc.Params.Orientation)
		dc.Canvas.MoveTo(px, py)

		for i := 1; i < len(pts); i++ {
			nx = normalize(pts[i].x, dc.XMin, dc.XMax)
			ny = normalize(pts[i].y, dc.YMin, dc.YMax)
			px, py = orientedTransform(dc.Coord, nx, ny, dc.W, dc.H, dc.Params.Orientation)
			dc.Canvas.LineTo(px, py)
		}

		dc.Canvas.Stroke()
	}
}

func drawBars(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol string, w, h, xMin, xMax, yMin, yMax float64, p geom.Params, th theme.Theme, barInsetPx float64) {
	// Collect all points first so we can compute spacing.
	type barPt struct{ x, y, ymin float64 }

	var pts []barPt

	xVals, err := ds.Float64(xCol)
	if err != nil {
		return
	}

	yVals, err := ds.Float64(yCol)
	if err != nil {
		yVals, _ = ds.Float64("count")
	}

	// Read optional ymin column (injected by stack/fill position adjustments).
	yMinVals, _ := ds.Float64("ymin")

	for i, x := range xVals {
		y := 1.0
		if yVals != nil && i < len(yVals) {
			y = yVals[i]
		}

		ym := 0.0
		if yMinVals != nil && i < len(yMinVals) {
			ym = yMinVals[i]
		}

		pts = append(pts, barPt{x, y, ym})
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
	// When horizontal, X data maps to vertical axis → use h instead of w.
	barAxisLen := w
	if p.Orientation == geom.Horizontal {
		barAxisLen = h
	}

	pixPerUnit := barAxisLen / dataRange

	halfBarPx := (minSpacing * pixPerUnit * relW) / 2
	if halfBarPx < 1 {
		halfBarPx = 1
	}

	alpha := p.Alpha
	if alpha <= 0 {
		if th.Geom.PatchAlpha > 0 {
			alpha = th.Geom.PatchAlpha
		} else {
			alpha = 0.85
		}
	}

	fr, fg, fb := colormap.ParseRGB(p.Fill, 0.2, 0.4, 0.8)

	for _, pt := range pts {
		nx := normalize(pt.x, xMin, xMax)
		ny := normalize(pt.y, yMin, yMax)
		baseNy := normalize(pt.ymin, yMin, yMax)

		// Get center and baseline pixel positions via coord.Transform.
		// orientedTransform swaps nx/ny when horizontal.
		cx, cyVal := orientedTransform(c, nx, ny, w, h, p.Orientation)
		cxBase, cyBase := orientedTransform(c, nx, baseNy, w, h, p.Orientation)
		_ = cyBase

		var rx, ry, rw, rh float64
		if p.Orientation == geom.Horizontal {
			// Horizontal: bar length is horizontal (value extent),
			// bar thickness is vertical (category spacing).
			rx = math.Min(cx, cxBase)
			ry = cyVal - halfBarPx
			rw = math.Abs(cx - cxBase)
			rh = halfBarPx * 2
		} else {
			// Vertical (default): bar width offset is horizontal.
			rx = cx - halfBarPx
			ry = math.Min(cyVal, cyBase)
			rw = halfBarPx * 2
			rh = math.Abs(cyVal - cyBase)
		}

		// Apply inset: shrink each bar by barInsetPx on each side.
		if barInsetPx > 0 {
			if p.Orientation == geom.Horizontal {
				ry += barInsetPx
				rh -= 2 * barInsetPx
			} else {
				rx += barInsetPx
				rw -= 2 * barInsetPx
			}

			if rw < 0.5 { //nolint:mnd // Never collapse a bar to less than half a pixel.
				rw = 0.5
			}

			if rh < 0.5 { //nolint:mnd // Never collapse a bar to less than half a pixel.
				rh = 0.5
			}
		}

		cv.SetRGBA(fr, fg, fb, alpha)
		cv.DrawRectangle(rx, ry, rw, rh)
		cv.FillPreserve()

		// Outline: use theme PatchEdgeColor when set, otherwise darken fill.
		edgeW := th.Geom.PatchEdgeWidth
		if edgeW <= 0 {
			edgeW = 0.5
		}

		if th.Geom.PatchEdgeColor != nil {
			cv.SetColor(th.Geom.PatchEdgeColor)
		} else {
			cv.SetRGBA(fr*0.7, fg*0.7, fb*0.7, alpha)
		}

		cv.SetLineWidth(edgeW)
		cv.Stroke()
	}
}

func drawArea(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol string, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	xVals, errX := ds.Float64(xCol)

	yVals, errY := ds.Float64(yCol)
	if errX != nil || errY != nil {
		return
	}

	// Collect normalized data points and their corresponding baselines.
	type normPt struct{ nx, ny float64 }

	n := min(len(yVals), len(xVals))

	npts := make([]normPt, n)
	for i := range n {
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
	bx0, by0 := orientedTransform(c, npts[0].nx, baseNy, w, h, p.Orientation)
	cv.MoveTo(bx0, by0)

	for _, np := range npts {
		px, py := orientedTransform(c, np.nx, np.ny, w, h, p.Orientation)
		cv.LineTo(px, py)
	}
	// Close back to baseline at the last X position.
	bxN, byN := orientedTransform(c, npts[len(npts)-1].nx, baseNy, w, h, p.Orientation)
	cv.LineTo(bxN, byN)
	cv.ClosePath()

	alpha := p.Alpha
	if alpha <= 0 {
		alpha = 0.6
	}

	fr, fg, fb := colormap.ParseRGB(p.Fill, 0.1, 0.7, 0.3)
	cv.SetRGBA(fr, fg, fb, alpha)
	cv.FillPreserve()

	cr, cg, cb := colormap.ParseRGB(p.Color, 0.0, 0.0, 0.0)
	cv.SetRGBA(cr, cg, cb, 1.0)
	cv.SetLineWidth(1)
	cv.Stroke()
}

func drawStep(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol string, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	xVals, errX := ds.Float64(xCol)

	yVals, errY := ds.Float64(yCol)
	if errX != nil || errY != nil {
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

	cr, cg, cb := colormap.ParseRGB(p.Color, 0.3, 0.5, 0.8)

	// Collect normalized data points.
	type normPt struct{ nx, ny float64 }

	n := min(len(yVals), len(xVals))

	npts := make([]normPt, n)
	for i := range n {
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
	px0, py0 := orientedTransform(c, npts[0].nx, npts[0].ny, w, h, p.Orientation)
	cv.MoveTo(px0, py0)

	for i := 1; i < len(npts); i++ {
		// Horizontal step: move to next X, keep previous Y.
		sx, sy := orientedTransform(c, npts[i].nx, npts[i-1].ny, w, h, p.Orientation)
		cv.LineTo(sx, sy)
		// Vertical step: now move Y to current value.
		vx, vy := orientedTransform(c, npts[i].nx, npts[i].ny, w, h, p.Orientation)
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

	cr, cg, cb := colormap.ParseRGB(p.Color, 0.2, 0.2, 0.2)

	const rugFrac = 0.02 // 2% of panel extent

	// X rugs: tick marks along the bottom edge of the panel.
	if xVals, err := ds.Float64(xCol); err == nil {
		cv.SetRGBA(cr, cg, cb, alpha)
		cv.SetLineWidth(lw)

		for _, x := range xVals {
			nx := normalize(x, xMin, xMax)
			px1, py1 := orientedTransform(c, nx, 0, w, h, p.Orientation)
			px2, py2 := orientedTransform(c, nx, rugFrac, w, h, p.Orientation)

			cv.MoveTo(px1, py1)
			cv.LineTo(px2, py2)
			cv.Stroke()
		}
	}

	// Y rugs: tick marks along the left edge of the panel.
	if yVals, err := ds.Float64(yCol); err == nil {
		cv.SetRGBA(cr, cg, cb, alpha)
		cv.SetLineWidth(lw)

		for _, y := range yVals {
			ny := normalize(y, yMin, yMax)
			px1, py1 := orientedTransform(c, 0, ny, w, h, p.Orientation)
			px2, py2 := orientedTransform(c, rugFrac, ny, w, h, p.Orientation)

			cv.MoveTo(px1, py1)
			cv.LineTo(px2, py2)
			cv.Stroke()
		}
	}
}

func drawHLine(cv canvas.Canvas, c coord.Coord, w, h, _, _, yMin, yMax float64, p geom.Params) {
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

	cr, cg, cb := colormap.ParseRGB(p.Color, 0.6, 0.1, 0.1)

	// Draw from left (X=0) to right (X=1) in normalized space.
	px1, py1 := orientedTransform(c, 0, ny, w, h, p.Orientation)
	px2, py2 := orientedTransform(c, 1, ny, w, h, p.Orientation)

	cv.SetRGBA(cr, cg, cb, alpha)
	cv.SetLineWidth(lw)
	cv.MoveTo(px1, py1)
	cv.LineTo(px2, py2)
	cv.Stroke()
}

func drawVLine(cv canvas.Canvas, c coord.Coord, w, h, xMin, xMax, _, _ float64, p geom.Params) {
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

	cr, cg, cb := colormap.ParseRGB(p.Color, 0.1, 0.1, 0.6)

	// Draw from bottom (Y=0) to top (Y=1) in normalized space.
	px1, py1 := orientedTransform(c, nx, 0, w, h, p.Orientation)
	px2, py2 := orientedTransform(c, nx, 1, w, h, p.Orientation)

	cv.SetRGBA(cr, cg, cb, alpha)
	cv.SetLineWidth(lw)
	cv.MoveTo(px1, py1)
	cv.LineTo(px2, py2)
	cv.Stroke()
}

func drawABLine(cv canvas.Canvas, c coord.Coord, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	slope := p.Slope
	intercept := p.Intercept

	// Compute Y values at the left and right edges of the data domain.
	y0 := slope*xMin + intercept
	y1 := slope*xMax + intercept

	// Clip to the visible Y range using parametric line clipping.
	// Parametric: x(t) = xMin + t*(xMax-xMin), y(t) = y0 + t*(y1-y0), t ∈ [0,1]
	t0, t1 := 0.0, 1.0
	dy := y1 - y0

	// Clip against yMin.
	if dy != 0 {
		tY := (yMin - y0) / dy
		if dy < 0 {
			if tY < t1 {
				t1 = tY
			}
		} else {
			if tY > t0 {
				t0 = tY
			}
		}
	} else if y0 < yMin || y0 > yMax {
		return // horizontal line outside Y range
	}

	// Clip against yMax.
	if dy != 0 {
		tY := (yMax - y0) / dy
		if dy < 0 {
			if tY > t0 {
				t0 = tY
			}
		} else {
			if tY < t1 {
				t1 = tY
			}
		}
	}

	if t0 >= t1 {
		return // entirely outside visible range
	}

	// Compute clipped endpoints in data space.
	cx0 := xMin + t0*(xMax-xMin)
	cy0 := y0 + t0*dy
	cx1 := xMin + t1*(xMax-xMin)
	cy1 := y0 + t1*dy

	// Normalize to [0,1].
	nx0 := normalize(cx0, xMin, xMax)
	ny0 := normalize(cy0, yMin, yMax)
	nx1 := normalize(cx1, xMin, xMax)
	ny1 := normalize(cy1, yMin, yMax)

	lw := p.LineWidth
	if lw <= 0 {
		lw = 1
	}

	alpha := p.Alpha
	if alpha <= 0 {
		alpha = 0.8
	}

	cr, cg, cb := colormap.ParseRGB(p.Color, 0.5, 0.2, 0.7)

	px1, py1 := orientedTransform(c, nx0, ny0, w, h, p.Orientation)
	px2, py2 := orientedTransform(c, nx1, ny1, w, h, p.Orientation)

	cv.SetRGBA(cr, cg, cb, alpha)
	cv.SetLineWidth(lw)
	cv.MoveTo(px1, py1)
	cv.LineTo(px2, py2)
	cv.Stroke()
}

func drawText(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol string, mapping AesMap, w, h, xMin, xMax, yMin, yMax float64, p geom.Params) {
	xVals, errX := ds.Float64(xCol)

	yVals, errY := ds.Float64(yCol)
	if errX != nil || errY != nil {
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

	cr, cg, cb := colormap.ParseRGB(p.Color, 0.1, 0.1, 0.1)

	cv.SetFontSize(fontSize)
	cv.SetRGBA(cr, cg, cb, alpha)

	n := min(len(yVals), len(xVals))

	for i := range n {
		nx := normalize(xVals[i], xMin, xMax)
		ny := normalize(yVals[i], yMin, yMax)
		px, py := orientedTransform(c, nx, ny, w, h, p.Orientation)

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

func drawBoxplot(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, w, h, xMin, xMax, yMin, yMax float64, p geom.Params, _ theme.Theme) {
	// Read stat_boxplot output columns.
	xVals, errX := ds.Float64("x")
	lowerVals, errL := ds.Float64("lower")
	q1Vals, errQ1 := ds.Float64("q1")
	midVals, errM := ds.Float64("middle")
	q3Vals, errQ3 := ds.Float64("q3")

	upperVals, errU := ds.Float64("upper")
	if errX != nil || errL != nil || errQ1 != nil || errM != nil || errQ3 != nil || errU != nil {
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

	fr, fg, fb := colormap.ParseRGB(p.Fill, 0.9, 0.9, 0.9)
	cr, cg, cb := colormap.ParseRGB(p.Color, 0.2, 0.2, 0.2)

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
	for i := range n {
		if i >= len(lowerVals) || i >= len(q1Vals) || i >= len(midVals) || i >= len(q3Vals) || i >= len(upperVals) {
			break
		}

		nx := normalize(xVals[i], xMin, xMax)
		nLo := normalize(lowerVals[i], yMin, yMax)
		nQ1 := normalize(q1Vals[i], yMin, yMax)
		nMd := normalize(midVals[i], yMin, yMax)
		nQ3 := normalize(q3Vals[i], yMin, yMax)
		nHi := normalize(upperVals[i], yMin, yMax)

		// Use orientedTransform for all pixel positions.
		// For vertical: cx = category center (x-axis), py* = value positions (y-axis)
		// For horizontal: cx = category center (y-axis), px* = value positions (x-axis)
		if p.Orientation == geom.Horizontal {
			// Horizontal boxplot: category on Y axis, values on X axis.
			_, cy := orientedTransform(c, nx, 0.5, w, h, p.Orientation)
			pxLo, _ := orientedTransform(c, nx, nLo, w, h, p.Orientation)
			pxQ1, _ := orientedTransform(c, nx, nQ1, w, h, p.Orientation)
			pxMd, _ := orientedTransform(c, nx, nMd, w, h, p.Orientation)
			pxQ3, _ := orientedTransform(c, nx, nQ3, w, h, p.Orientation)
			pxHi, _ := orientedTransform(c, nx, nHi, w, h, p.Orientation)

			// Box rectangle (horizontal orientation).
			rx := math.Min(pxQ1, pxQ3)
			ry := cy - halfPixel
			rw := math.Abs(pxQ3 - pxQ1)
			rh := halfPixel * 2

			cv.SetRGBA(fr, fg, fb, alpha)
			cv.DrawRectangle(rx, ry, rw, rh)
			cv.Fill()

			cv.SetRGBA(cr, cg, cb, 1.0)
			cv.SetLineWidth(lw)
			cv.DrawRectangle(rx, ry, rw, rh)
			cv.Stroke()

			// Median line (vertical within box).
			cv.SetLineWidth(lw * 1.5)
			cv.MoveTo(pxMd, cy-halfPixel)
			cv.LineTo(pxMd, cy+halfPixel)
			cv.Stroke()

			// Lower whisker.
			cv.SetLineWidth(lw)
			cv.MoveTo(pxLo, cy)
			cv.LineTo(pxQ1, cy)
			cv.Stroke()

			// Upper whisker.
			cv.MoveTo(pxQ3, cy)
			cv.LineTo(pxHi, cy)
			cv.Stroke()

			// Whisker caps (vertical bars).
			capHalf := halfPixel * 0.4
			cv.MoveTo(pxLo, cy-capHalf)
			cv.LineTo(pxLo, cy+capHalf)
			cv.Stroke()

			cv.MoveTo(pxHi, cy-capHalf)
			cv.LineTo(pxHi, cy+capHalf)
			cv.Stroke()
		} else {
			// Vertical boxplot (default): category on X axis, values on Y axis.
			cx, _ := orientedTransform(c, nx, 0.5, w, h, p.Orientation)
			_, pyLo := orientedTransform(c, nx, nLo, w, h, p.Orientation)
			_, pyQ1 := orientedTransform(c, nx, nQ1, w, h, p.Orientation)
			_, pyMd := orientedTransform(c, nx, nMd, w, h, p.Orientation)
			_, pyQ3 := orientedTransform(c, nx, nQ3, w, h, p.Orientation)
			_, pyHi := orientedTransform(c, nx, nHi, w, h, p.Orientation)

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
}

// orientedTransform applies coord.Transform, swapping the normalized x/y
// when the geom orientation is [geom.Horizontal]. This centralizes the
// coordinate swap that was previously handled by the removed flippedCoord.
func orientedTransform(c coord.Coord, nx, ny, w, h float64, o geom.Orientation) (px, py float64) {
	if o == geom.Horizontal {
		nx, ny = ny, nx
	}

	return c.Transform(nx, ny, w, h)
}

// normalize maps a value to [0, 1] within [lo, hi].
func normalize(v, lo, hi float64) float64 {
	if hi == lo {
		return 0.5
	}

	return (v - lo) / (hi - lo)
}

// ---------------------------------------------------------------------------
// Tile (heatmap cells)
// ---------------------------------------------------------------------------

func drawTileFn(dc DrawContext) {
	xVals, err := dc.Data.Float64(dc.Mapping["x"])
	if err != nil {
		return
	}

	yVals, err := dc.Data.Float64(dc.Mapping["y"])
	if err != nil {
		return
	}

	n := min(len(xVals), len(yVals))
	if n == 0 {
		return
	}

	// Auto-detect cell size from minimum spacing.
	dx := autoSpacing(xVals)
	dy := autoSpacing(yVals)

	fr, fg, fb := colormap.ParseRGB(dc.Params.Fill, 0.2, 0.4, 0.8)

	alpha := dc.Params.Alpha
	if alpha <= 0 {
		alpha = 1.0
	}

	for i := range n {
		x, y := xVals[i], yVals[i]

		// Normalize the four corners.
		nxLo := normalize(x-dx/2, dc.XMin, dc.XMax)
		nxHi := normalize(x+dx/2, dc.XMin, dc.XMax)
		nyLo := normalize(y-dy/2, dc.YMin, dc.YMax)
		nyHi := normalize(y+dy/2, dc.YMin, dc.YMax)

		// Transform corners to pixel coords.
		px0, py0 := dc.Coord.Transform(nxLo, nyLo, dc.W, dc.H)
		px1, py1 := dc.Coord.Transform(nxHi, nyHi, dc.W, dc.H)

		rx := math.Min(px0, px1)
		ry := math.Min(py0, py1)
		rw := math.Abs(px1 - px0)
		rh := math.Abs(py1 - py0)

		// Per-datum color from continuous scale, or fallback to fill.
		if dc.ContScale != nil && dc.ContColorCol != "" {
			zVals, zErr := dc.Data.Float64(dc.ContColorCol)
			if zErr == nil && i < len(zVals) {
				c := dc.ContScale.At(zVals[i])
				dc.Canvas.SetRGBA(c.R, c.G, c.B, alpha)
			} else {
				dc.Canvas.SetRGBA(fr, fg, fb, alpha)
			}
		} else {
			dc.Canvas.SetRGBA(fr, fg, fb, alpha)
		}

		dc.Canvas.DrawRectangle(rx, ry, rw, rh)
		dc.Canvas.Fill()
	}
}

// autoSpacing returns the minimum absolute spacing between sorted values,
// or 1.0 if fewer than 2 values.
func autoSpacing(vals []float64) float64 {
	if len(vals) < 2 {
		return 1.0
	}

	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)

	sp := math.Abs(sorted[1] - sorted[0])
	for i := 2; i < len(sorted); i++ {
		d := math.Abs(sorted[i] - sorted[i-1])
		if d > 0 && d < sp {
			sp = d
		}
	}

	if sp <= 0 {
		sp = 1.0
	}

	return sp
}

// ---------------------------------------------------------------------------
// Segment (x,y → xend,yend line segments)
// ---------------------------------------------------------------------------

func drawSegmentFn(dc DrawContext) {
	xVals, err := dc.Data.Float64(dc.Mapping["x"])
	if err != nil {
		return
	}

	yVals, err := dc.Data.Float64(dc.Mapping["y"])
	if err != nil {
		return
	}

	xeCol := dc.Mapping["xend"]
	yeCol := dc.Mapping["yend"]

	xeVals, err := dc.Data.Float64(xeCol)
	if err != nil {
		return
	}

	yeVals, err := dc.Data.Float64(yeCol)
	if err != nil {
		return
	}

	n := min(len(xVals), len(yVals), len(xeVals), len(yeVals))

	cr, cg, cb := colormap.ParseRGB(dc.Params.Color, 0.2, 0.2, 0.2)

	alpha := dc.Params.Alpha
	if alpha <= 0 {
		alpha = 1.0
	}

	lw := dc.Params.LineWidth
	if lw <= 0 {
		lw = 1
	}

	dc.Canvas.SetLineWidth(lw)

	for i := range n {
		nx0 := normalize(xVals[i], dc.XMin, dc.XMax)
		ny0 := normalize(yVals[i], dc.YMin, dc.YMax)
		nx1 := normalize(xeVals[i], dc.XMin, dc.XMax)
		ny1 := normalize(yeVals[i], dc.YMin, dc.YMax)

		px0, py0 := dc.Coord.Transform(nx0, ny0, dc.W, dc.H)
		px1, py1 := dc.Coord.Transform(nx1, ny1, dc.W, dc.H)

		dc.Canvas.SetRGBA(cr, cg, cb, alpha)
		dc.Canvas.MoveTo(px0, py0)
		dc.Canvas.LineTo(px1, py1)
		dc.Canvas.Stroke()
	}
}

// ---------------------------------------------------------------------------
// ErrorBar (vertical or horizontal error bars with caps)
// ---------------------------------------------------------------------------

func drawErrorBarFn(dc DrawContext) {
	xVals, err := dc.Data.Float64(dc.Mapping["x"])
	if err != nil {
		return
	}

	yminCol := dc.Mapping["ymin"]
	ymaxCol := dc.Mapping["ymax"]

	yminVals, err := dc.Data.Float64(yminCol)
	if err != nil {
		return
	}

	ymaxVals, err := dc.Data.Float64(ymaxCol)
	if err != nil {
		return
	}

	n := min(len(xVals), len(yminVals), len(ymaxVals))

	cr, cg, cb := colormap.ParseRGB(dc.Params.Color, 0.2, 0.2, 0.2)

	alpha := dc.Params.Alpha
	if alpha <= 0 {
		alpha = 1.0
	}

	lw := dc.Params.LineWidth
	if lw <= 0 {
		lw = 1
	}

	// Cap half-width in pixels.
	capW := dc.Params.Width
	if capW <= 0 {
		capW = 0.5
	}

	capPx := capW * 4 //nolint:mnd // Convert relative width to a reasonable pixel cap size.

	dc.Canvas.SetLineWidth(lw)
	dc.Canvas.SetRGBA(cr, cg, cb, alpha)

	for i := range n {
		nx := normalize(xVals[i], dc.XMin, dc.XMax)
		nyLo := normalize(yminVals[i], dc.YMin, dc.YMax)
		nyHi := normalize(ymaxVals[i], dc.YMin, dc.YMax)

		px, pyLo := dc.Coord.Transform(nx, nyLo, dc.W, dc.H)
		_, pyHi := dc.Coord.Transform(nx, nyHi, dc.W, dc.H)

		// Vertical stem.
		dc.Canvas.MoveTo(px, pyLo)
		dc.Canvas.LineTo(px, pyHi)
		dc.Canvas.Stroke()

		// Bottom cap.
		dc.Canvas.MoveTo(px-capPx, pyLo)
		dc.Canvas.LineTo(px+capPx, pyLo)
		dc.Canvas.Stroke()

		// Top cap.
		dc.Canvas.MoveTo(px-capPx, pyHi)
		dc.Canvas.LineTo(px+capPx, pyHi)
		dc.Canvas.Stroke()
	}
}

// ---------------------------------------------------------------------------
// Polygon (closed paths through grouped x/y points)
// ---------------------------------------------------------------------------

func drawPolygonFn(dc DrawContext) {
	xVals, err := dc.Data.Float64(dc.Mapping["x"])
	if err != nil {
		return
	}

	yVals, err := dc.Data.Float64(dc.Mapping["y"])
	if err != nil {
		return
	}

	n := min(len(xVals), len(yVals))
	if n < 3 { //nolint:mnd // A polygon needs at least 3 vertices.
		return
	}

	fr, fg, fb := colormap.ParseRGB(dc.Params.Fill, 0.2, 0.4, 0.8)

	alpha := dc.Params.Alpha
	if alpha <= 0 {
		alpha = 0.6
	}

	lw := dc.Params.LineWidth
	if lw <= 0 {
		lw = 1
	}

	// Build closed path.
	for i := range n {
		nx := normalize(xVals[i], dc.XMin, dc.XMax)
		ny := normalize(yVals[i], dc.YMin, dc.YMax)
		px, py := dc.Coord.Transform(nx, ny, dc.W, dc.H)

		if i == 0 {
			dc.Canvas.MoveTo(px, py)
		} else {
			dc.Canvas.LineTo(px, py)
		}
	}

	dc.Canvas.ClosePath()

	// Fill.
	dc.Canvas.SetRGBA(fr, fg, fb, alpha)
	dc.Canvas.FillPreserve()

	// Stroke.
	cr, cg, cb := colormap.ParseRGB(dc.Params.Color, fr*0.7, fg*0.7, fb*0.7)
	dc.Canvas.SetRGBA(cr, cg, cb, alpha)
	dc.Canvas.SetLineWidth(lw)
	dc.Canvas.Stroke()
}

// drawRibbonFn renders a filled band between ymin and ymax columns.
func drawRibbonFn(dc DrawContext) {
	xCol := dc.Mapping["x"]

	xVals, errX := dc.Data.Float64(xCol)
	if errX != nil {
		return
	}

	yminVals, errMin := dc.Data.Float64("ymin")
	if errMin != nil {
		return
	}

	ymaxVals, errMax := dc.Data.Float64("ymax")
	if errMax != nil {
		return
	}

	n := min(len(xVals), min(len(yminVals), len(ymaxVals)))
	if n < 2 {
		return
	}

	alpha := dc.Params.Alpha
	if alpha <= 0 {
		alpha = 0.3
	}

	fr, fg, fb := colormap.ParseRGB(dc.Params.Fill, 0.2, 0.5, 0.8)

	// Forward pass: trace the upper edge (ymax).
	nx0 := normalize(xVals[0], dc.XMin, dc.XMax)
	ny0 := normalize(ymaxVals[0], dc.YMin, dc.YMax)
	px, py := orientedTransform(dc.Coord, nx0, ny0, dc.W, dc.H, dc.Params.Orientation)
	dc.Canvas.MoveTo(px, py)

	for i := 1; i < n; i++ {
		nx := normalize(xVals[i], dc.XMin, dc.XMax)
		ny := normalize(ymaxVals[i], dc.YMin, dc.YMax)
		px, py = orientedTransform(dc.Coord, nx, ny, dc.W, dc.H, dc.Params.Orientation)
		dc.Canvas.LineTo(px, py)
	}

	// Reverse pass: trace the lower edge (ymin) back.
	for i := n - 1; i >= 0; i-- {
		nx := normalize(xVals[i], dc.XMin, dc.XMax)
		ny := normalize(yminVals[i], dc.YMin, dc.YMax)
		px, py = orientedTransform(dc.Coord, nx, ny, dc.W, dc.H, dc.Params.Orientation)
		dc.Canvas.LineTo(px, py)
	}

	dc.Canvas.ClosePath()

	// Fill.
	dc.Canvas.SetRGBA(fr, fg, fb, alpha)
	dc.Canvas.FillPreserve()

	// Stroke.
	lw := dc.Params.LineWidth
	if lw <= 0 {
		lw = 0.5
	}

	cr, cg, cb := colormap.ParseRGB(dc.Params.Color, fr*0.7, fg*0.7, fb*0.7)
	dc.Canvas.SetRGBA(cr, cg, cb, alpha)
	dc.Canvas.SetLineWidth(lw)
	dc.Canvas.Stroke()
}

// drawDifferenceFn renders the area between two series with dual fill.
// Positive regions (ymax > ymin) use the fill color, negative regions
// use a complementary color. Falls back to ribbon rendering for now.
func drawDifferenceFn(dc DrawContext) {
	// Difference is structurally the same as ribbon — the dual-color
	// split requires clipping which canvas backends may not support.
	// We render it as a ribbon with the fill color for now, producing
	// correct shape. Dual-color clipping will be added when the canvas
	// abstraction supports clip paths.
	drawRibbonFn(dc)
}
