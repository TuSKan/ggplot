// draw.go provides geometry rendering functions for the plotting pipeline.
// Each draw function maps data coordinates to pixel positions using a coordinate
// system and renders shapes via the canvas abstraction.
//
// The Drawer interface enables extensible geometry dispatch: third-party geoms
// can register their own draw functions via [RegisterDrawer].

package ggplot

import (
	"fmt"
	"image"
	"image/color"
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

	// Metadata channels (SVG tooltips, links, ARIA labels).
	// Pre-resolved string slices — nil when the channel is unmapped.
	titleVals     []string
	hrefVals      []string
	ariaLabelVals []string
}

// SetRowMetadata sets per-primitive SVG metadata (title, href, aria-label)
// for data row i. Call this before each canvas Fill/Stroke that corresponds
// to a data row. When no metadata channels are mapped, this is a no-op.
func (dc *DrawContext) SetRowMetadata(i int) {
	if dc.titleVals == nil && dc.hrefVals == nil && dc.ariaLabelVals == nil {
		return
	}

	meta := make(map[string]string, 3) //nolint:mnd // At most 3 metadata keys.

	if i < len(dc.titleVals) && dc.titleVals[i] != "" {
		meta["title"] = dc.titleVals[i]
	}

	if i < len(dc.hrefVals) && dc.hrefVals[i] != "" {
		meta["href"] = dc.hrefVals[i]
	}

	if i < len(dc.ariaLabelVals) && dc.ariaLabelVals[i] != "" {
		meta["aria_label"] = dc.ariaLabelVals[i]
	}

	if len(meta) > 0 {
		dc.Canvas.SetMetadata(meta)
	}
}

// HasMetadata reports whether any metadata channels are mapped.
func (dc *DrawContext) HasMetadata() bool {
	return dc.titleVals != nil || dc.hrefVals != nil || dc.ariaLabelVals != nil
}

// rowMetaFn returns a callback for inner draw functions that don't receive
// DrawContext. Returns nil when no metadata channels are mapped (zero cost).
func (dc *DrawContext) rowMetaFn() func(int) {
	if !dc.HasMetadata() {
		return nil
	}

	return dc.SetRowMetadata
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
	RegisterDrawer(geom.TypeCrossbar, DrawerFunc(drawCrossbarFn))
	RegisterDrawer(geom.TypeLinerange, DrawerFunc(drawLinerangeFn))
	RegisterDrawer(geom.TypePointrange, DrawerFunc(drawPointrangeFn))
	RegisterDrawer(geom.TypeCurve, DrawerFunc(drawCurveFn))
	RegisterDrawer(geom.TypeViolin, DrawerFunc(drawViolinFn))
	RegisterDrawer(geom.TypeDotplot, DrawerFunc(drawDotplotFn))
	RegisterDrawer(geom.TypeRaster, DrawerFunc(drawRasterFn))
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

	dc := DrawContext{
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
	}

	// Resolve metadata channels: title, href, aria_label.
	// These are passthrough aesthetics — not trained on scales, just string
	// columns carried through the pipeline for SVG metadata emission.
	if col, ok := rl.Mapping["title"]; ok && col != "" {
		if c, err := rl.Data.Column(col); err == nil {
			dc.titleVals = columnAsStrings(c)
		}
	}

	if col, ok := rl.Mapping["href"]; ok && col != "" {
		if c, err := rl.Data.Column(col); err == nil {
			dc.hrefVals = columnAsStrings(c)
		}
	}

	if col, ok := rl.Mapping["aria_label"]; ok && col != "" {
		if c, err := rl.Data.Column(col); err == nil {
			dc.ariaLabelVals = columnAsStrings(c)
		}
	}

	d.Draw(dc)
}

// --- Drawer adapters: bridge DrawContext to existing draw functions ---

func drawPointsFn(dc DrawContext) {
	drawPoints(dc)
}

func drawLineFn(dc DrawContext) {
	drawLine(dc)
}

func drawStepFn(dc DrawContext) {
	// Step line is a single polyline per group — set group-level metadata.
	dc.SetRowMetadata(0)

	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	drawStep(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params)
}

func drawBarsFn(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	drawBars(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params, dc.Theme, 0, dc.rowMetaFn())
}

func drawHistogramFn(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	// Half-pixel inset per side → 1px total gap between adjacent bins,
	// matching Observable Plot's continuous-bar default.
	drawBars(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params, dc.Theme, 0.5, dc.rowMetaFn())
}

func drawRectFn(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	drawBars(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params, dc.Theme, dc.Params.Inset, dc.rowMetaFn())
}

func drawAreaFn(dc DrawContext) {
	// Area is a single filled polygon per group — set group-level metadata.
	dc.SetRowMetadata(0)

	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	drawArea(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params)
}

func drawRugFn(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	drawRug(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params, dc.rowMetaFn())
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
	drawText(dc.Canvas, dc.Coord, dc.Data, xCol, yCol, dc.Mapping, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params, dc.rowMetaFn())
}

func drawBoxplotFn(dc DrawContext) {
	drawBoxplot(dc.Canvas, dc.Coord, dc.Data, dc.W, dc.H, dc.XMin, dc.XMax, dc.YMin, dc.YMax, dc.Params, dc.Theme, dc.rowMetaFn())
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

func drawPoints(dc DrawContext) { //nolint:gocognit // Rendering pipeline with decimation, batching, culling, and multi-aesthetic mapping — splitting reduces clarity.
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

	// Screen-space decimation: when there are far more data points than
	// pixels, many points map to the same pixel cell. We use a bitmap to
	// skip points that would overdraw an already-occupied cell. This turns
	// O(N) GPU path commands into O(grid_cells) — critical for 10K+ datasets.
	//
	// Adaptive bin size: at extreme densities, use coarser bins (2×2, 3×3,
	// or 4×4 pixel blocks) to cap rendered points at a manageable count.
	pw := int(dc.W + 1) //nolint:mnd // Pixel grid width.
	ph := int(dc.H + 1) //nolint:mnd // Pixel grid height.

	var pixelOccupied []bool

	binSize := 1 // pixels per grid cell

	screenCells := pw * ph
	if screenCells > 0 && n > 2*pw { //nolint:mnd // 2× panel width threshold.
		// Choose bin size based on data density relative to screen area.
		// Always use at least 2×2 bins when decimation is active — at 1px
		// point size this is visually imperceptible.
		binSize = 2 //nolint:mnd // Minimum bin size when decimation is active.

		if n > screenCells {
			binSize = 3 //nolint:mnd // >1× oversampling → 3×3 bins.
		}

		if n > 3*screenCells { //nolint:mnd // >3× oversampling → 4×4 bins.
			binSize = 4 //nolint:mnd
		}

		gw := (pw + binSize - 1) / binSize
		gh := (ph + binSize - 1) / binSize
		pixelOccupied = make([]bool, gw*gh)
	}

	gridW := (pw + binSize - 1) / binSize

	// Batched rendering: accumulate same-color shapes into a single path,
	// then flush with one Fill()/Stroke(). Screen-space decimation above
	// guarantees the batch size stays bounded by pixel count.

	var (
		batchR, batchG, batchB, batchA float64 // current batch color
		batchShape                     string  // current batch shape
		batchStroke                    bool    // current batch is stroked (not filled)
		batchN                         int     // points accumulated in current batch
	)

	flushBatch := func() {
		if batchN == 0 {
			return
		}

		dc.Canvas.SetRGBA(batchR, batchG, batchB, batchA)

		if batchStroke {
			dc.Canvas.SetLineWidth(1.5) //nolint:mnd // Standard stroke-shape line width.
			dc.Canvas.Stroke()
		} else {
			dc.Canvas.Fill()
		}

		batchN = 0
	}

	hasMeta := dc.HasMetadata()

	for i := range n {
		nx := normalize(xVals[i], dc.XMin, dc.XMax)
		ny := normalize(yVals[i], dc.YMin, dc.YMax)
		px, py := orientedTransform(dc.Coord, nx, ny, dc.W, dc.H, dc.Params.Orientation)

		ptRadius := r
		if i < len(sizeVals) && sizeMapper != nil {
			ptRadius = sizeMapper(sizeVals[i])
		}

		// Cull points fully outside the clip region — avoids GPU draw
		// commands for off-screen geometry during pan/zoom.
		if px+ptRadius < 0 || px-ptRadius > dc.W || py+ptRadius < 0 || py-ptRadius > dc.H {
			continue
		}

		// Screen-space decimation: skip points that map to an
		// already-occupied grid cell. Disabled when metadata is
		// present — every data row needs its own SVG primitive.
		if pixelOccupied != nil && !hasMeta {
			ix := int(px) / binSize
			iy := int(py) / binSize

			if ix >= 0 && ix < gridW && iy >= 0 && iy < (ph+binSize-1)/binSize {
				idx := iy*gridW + ix

				if pixelOccupied[idx] {
					continue
				}

				pixelOccupied[idx] = true
			}
		}

		ptAlpha := alpha
		if i < len(alphaVals) && alphaMapper != nil {
			ptAlpha = alphaMapper(alphaVals[i])
		}

		var ptR, ptG, ptB float64

		if i < len(zVals) && dc.ContScale != nil {
			gc := dc.ContScale.At(zVals[i])
			ptR, ptG, ptB = gc.R, gc.G, gc.B
		} else {
			ptR, ptG, ptB = cr, cg, cb
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

		isStroke := canvas.IsStrokeShape(shapeName)

		// When metadata is present, each point must be its own fill/stroke
		// so it gets individual SVG metadata. Flush before every point.
		if hasMeta {
			flushBatch()
			dc.SetRowMetadata(i)
		} else if batchN > 0 && (ptR != batchR || ptG != batchG || ptB != batchB ||
			ptAlpha != batchA || shapeName != batchShape || isStroke != batchStroke) {
			// Flush batch when color, alpha, shape, or stroke/fill changes.
			flushBatch()
		}

		batchR, batchG, batchB, batchA = ptR, ptG, ptB, ptAlpha
		batchShape = shapeName
		batchStroke = isStroke

		// For tiny points (≤1.5px radius), use rectangles instead of circles.
		// A 1px circle is 4 cubic beziers → expensive FanTessellator flattening.
		// A 1px rectangle is 4 LineTo → trivially tessellated. Visually identical.
		if ptRadius <= 1.5 && (shapeName == canvas.ShapeCircle || shapeName == "") { //nolint:mnd // 1.5px threshold.
			dc.Canvas.DrawRectangle(px-ptRadius, py-ptRadius, ptRadius*2, ptRadius*2) //nolint:mnd // Center the square.
		} else {
			dc.Canvas.DrawShape(shapeName, px, py, ptRadius)
		}

		batchN++
	}

	flushBatch()
}

func drawLine(dc DrawContext) { //nolint:gocognit // Rendering pipeline with min/max decimation — splitting reduces pipeline clarity.
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

	// Line is a single polyline per group — set group-level metadata.
	dc.SetRowMetadata(0)

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

		// Min/max line decimation: for each unique X pixel column, emit
		// only the min and max Y values. This preserves the visual
		// envelope (peaks and valleys) while reducing a 100K-point
		// polyline to ~2 × panel_width segments.
		panelW := int(dc.W + 1) //nolint:mnd // Panel pixel width.

		if len(pts) > 4*panelW { //nolint:mnd // Only decimate when worth it (4× oversampling threshold).
			// Pre-compute all pixel coordinates.
			type pxPt struct{ px, py float64 }

			pxPts := make([]pxPt, len(pts))
			for i := range pts {
				nx := normalize(pts[i].x, dc.XMin, dc.XMax)
				ny := normalize(pts[i].y, dc.YMin, dc.YMax)
				px, py := orientedTransform(dc.Coord, nx, ny, dc.W, dc.H, dc.Params.Orientation)
				pxPts[i] = pxPt{px, py}
			}

			dc.Canvas.MoveTo(pxPts[0].px, pxPts[0].py)

			i := 1

			for i < len(pxPts) {
				curIX := int(pxPts[i].px)

				// Collect all points in this X pixel column.
				minPY, maxPY := pxPts[i].py, pxPts[i].py
				minI, maxI := i, i
				j := i + 1

				for j < len(pxPts) && int(pxPts[j].px) == curIX {
					if pxPts[j].py < minPY {
						minPY = pxPts[j].py
						minI = j
					}

					if pxPts[j].py > maxPY {
						maxPY = pxPts[j].py
						maxI = j
					}

					j++
				}

				// Emit min then max (or max then min) in data order to
				// preserve the direction of the line.
				if minI <= maxI {
					dc.Canvas.LineTo(pxPts[minI].px, minPY)

					if maxI != minI {
						dc.Canvas.LineTo(pxPts[maxI].px, maxPY)
					}
				} else {
					dc.Canvas.LineTo(pxPts[maxI].px, maxPY)

					if minI != maxI {
						dc.Canvas.LineTo(pxPts[minI].px, minPY)
					}
				}

				i = j
			}

			dc.Canvas.Stroke()
		} else {
			// Small dataset — render all points directly.
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
}

func drawBars(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol string, w, h, xMin, xMax, yMin, yMax float64, p geom.Params, th theme.Theme, barInsetPx float64, metaFn func(int)) { //nolint:gocognit // Bar rendering pipeline with orientation, stacking, inset, and per-bar metadata — splitting reduces clarity.
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

	for i, pt := range pts {
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

		if metaFn != nil {
			metaFn(i)
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

func drawRug(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol string, w, h, xMin, xMax, yMin, yMax float64, p geom.Params, metaFn func(int)) {
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
	// Rendered as filled rectangles to avoid expensive StrokeExpander.
	// All ticks are batched into a single Fill() call.
	if xVals, err := ds.Float64(xCol); err == nil {
		cv.SetRGBA(cr, cg, cb, alpha)

		prevIX := -1

		for i, x := range xVals {
			nx := normalize(x, xMin, xMax)
			px1, py1 := orientedTransform(c, nx, 0, w, h, p.Orientation)

			// Screen-space decimation: skip ticks at the same pixel column.
			ix := int(px1)
			if ix == prevIX {
				continue
			}

			prevIX = ix

			_, py2 := orientedTransform(c, nx, rugFrac, w, h, p.Orientation)

			// Draw tick as a thin filled rectangle (width = lw pixels).
			tickH := py2 - py1
			if tickH < 0 {
				py1 += tickH
				tickH = -tickH
			}

			if metaFn != nil {
				metaFn(i)
				cv.DrawRectangle(px1-lw/2, py1, lw, tickH) //nolint:mnd // Center the rectangle on the tick position.
				cv.Fill()
			} else {
				cv.DrawRectangle(px1-lw/2, py1, lw, tickH) //nolint:mnd // Center the rectangle on the tick position.
			}
		}

		if metaFn == nil {
			cv.Fill()
		}
	}

	// Y rugs: tick marks along the left edge of the panel.
	if yVals, err := ds.Float64(yCol); err == nil {
		cv.SetRGBA(cr, cg, cb, alpha)

		prevIY := -1

		for i, y := range yVals {
			ny := normalize(y, yMin, yMax)
			px1, py1 := orientedTransform(c, 0, ny, w, h, p.Orientation)

			// Screen-space decimation: skip ticks at the same pixel row.
			iy := int(py1)
			if iy == prevIY {
				continue
			}

			prevIY = iy

			px2, _ := orientedTransform(c, rugFrac, ny, w, h, p.Orientation)

			// Draw tick as a thin filled rectangle (height = lw pixels).
			tickW := px2 - px1
			if tickW < 0 {
				px1 += tickW
				tickW = -tickW
			}

			if metaFn != nil {
				metaFn(i)
				cv.DrawRectangle(px1, py1-lw/2, tickW, lw) //nolint:mnd // Center the rectangle on the tick position.
				cv.Fill()
			} else {
				cv.DrawRectangle(px1, py1-lw/2, tickW, lw) //nolint:mnd // Center the rectangle on the tick position.
			}
		}

		if metaFn == nil {
			cv.Fill()
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

func drawText(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, xCol, yCol string, mapping AesMap, w, h, xMin, xMax, yMin, yMax float64, p geom.Params, metaFn func(int)) {
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

		if metaFn != nil {
			metaFn(i)
		}

		cv.SetRGBA(cr, cg, cb, alpha)
		cv.DrawStringAnchored(lbl, px, py-fontSize-4, 0.5, 1.0)
	}
}

func drawBoxplot(cv canvas.Canvas, c coord.Coord, ds dataset.Dataset, w, h, xMin, xMax, yMin, yMax float64, p geom.Params, _ theme.Theme, metaFn func(int)) {
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
		if metaFn != nil {
			metaFn(i)
		}

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

		dc.SetRowMetadata(i)

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

		dc.SetRowMetadata(i)

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

		dc.SetRowMetadata(i)

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

	// Polygon is a single closed shape per group — set group-level metadata.
	dc.SetRowMetadata(0)

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

	// Ribbon is a single filled band per group — set group-level metadata.
	dc.SetRowMetadata(0)

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

// ---------------------------------------------------------------------------
// Crossbar (box with median line between ymin/ymax)
// ---------------------------------------------------------------------------

func drawCrossbarFn(dc DrawContext) {
	xVals, err := dc.Data.Float64(dc.Mapping["x"])
	if err != nil {
		return
	}

	yVals, err := dc.Data.Float64(dc.Mapping["y"])
	if err != nil {
		return
	}

	yminVals, err := dc.Data.Float64(dc.Mapping["ymin"])
	if err != nil {
		return
	}

	ymaxVals, err := dc.Data.Float64(dc.Mapping["ymax"])
	if err != nil {
		return
	}

	n := min(len(xVals), len(yVals), len(yminVals), len(ymaxVals))

	fr, fg, fb := colormap.ParseRGB(dc.Params.Fill, 0.8, 0.8, 0.8)
	cr, cg, cb := colormap.ParseRGB(dc.Params.Color, 0.2, 0.2, 0.2)

	alpha := dc.Params.Alpha
	if alpha <= 0 {
		alpha = 0.8 //nolint:mnd // Default alpha for crossbar fill.
	}

	lw := dc.Params.LineWidth
	if lw <= 0 {
		lw = 1
	}

	barW := dc.Params.Width
	if barW <= 0 {
		barW = 0.5 //nolint:mnd // Default relative crossbar width.
	}

	// Pixel half-width of the crossbar.
	halfPx := barW * dc.W / (dc.XMax - dc.XMin) / 2 //nolint:mnd // Half-width in pixel space.

	for i := range n {
		nx := normalize(xVals[i], dc.XMin, dc.XMax)
		nyLo := normalize(yminVals[i], dc.YMin, dc.YMax)
		nyHi := normalize(ymaxVals[i], dc.YMin, dc.YMax)
		nyMid := normalize(yVals[i], dc.YMin, dc.YMax)

		px, pyLo := dc.Coord.Transform(nx, nyLo, dc.W, dc.H)
		_, pyHi := dc.Coord.Transform(nx, nyHi, dc.W, dc.H)
		_, pyMid := dc.Coord.Transform(nx, nyMid, dc.W, dc.H)

		dc.SetRowMetadata(i)

		// Filled rectangle from ymin to ymax.
		dc.Canvas.SetRGBA(fr, fg, fb, alpha)
		dc.Canvas.DrawRectangle(px-halfPx, pyHi, halfPx*2, pyLo-pyHi) //nolint:mnd // Full width = 2 * half.
		dc.Canvas.Fill()

		// Stroke the rectangle outline.
		dc.Canvas.SetRGBA(cr, cg, cb, alpha)
		dc.Canvas.SetLineWidth(lw)
		dc.Canvas.DrawRectangle(px-halfPx, pyHi, halfPx*2, pyLo-pyHi) //nolint:mnd // Full width = 2 * half.
		dc.Canvas.Stroke()

		// Median line.
		dc.Canvas.SetLineWidth(lw * 2) //nolint:mnd // Median line is thicker than border.
		dc.Canvas.MoveTo(px-halfPx, pyMid)
		dc.Canvas.LineTo(px+halfPx, pyMid)
		dc.Canvas.Stroke()
	}
}

// ---------------------------------------------------------------------------
// Linerange (vertical/horizontal line from ymin to ymax, no caps)
// ---------------------------------------------------------------------------

func drawLinerangeFn(dc DrawContext) {
	xVals, err := dc.Data.Float64(dc.Mapping["x"])
	if err != nil {
		return
	}

	yminVals, err := dc.Data.Float64(dc.Mapping["ymin"])
	if err != nil {
		return
	}

	ymaxVals, err := dc.Data.Float64(dc.Mapping["ymax"])
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

	dc.Canvas.SetLineWidth(lw)
	dc.Canvas.SetRGBA(cr, cg, cb, alpha)

	for i := range n {
		nx := normalize(xVals[i], dc.XMin, dc.XMax)
		nyLo := normalize(yminVals[i], dc.YMin, dc.YMax)
		nyHi := normalize(ymaxVals[i], dc.YMin, dc.YMax)

		px, pyLo := dc.Coord.Transform(nx, nyLo, dc.W, dc.H)
		_, pyHi := dc.Coord.Transform(nx, nyHi, dc.W, dc.H)

		dc.SetRowMetadata(i)

		dc.Canvas.MoveTo(px, pyLo)
		dc.Canvas.LineTo(px, pyHi)
		dc.Canvas.Stroke()
	}
}

// ---------------------------------------------------------------------------
// Pointrange (point at y + linerange from ymin to ymax)
// ---------------------------------------------------------------------------

func drawPointrangeFn(dc DrawContext) {
	xVals, err := dc.Data.Float64(dc.Mapping["x"])
	if err != nil {
		return
	}

	yVals, err := dc.Data.Float64(dc.Mapping["y"])
	if err != nil {
		return
	}

	yminVals, err := dc.Data.Float64(dc.Mapping["ymin"])
	if err != nil {
		return
	}

	ymaxVals, err := dc.Data.Float64(dc.Mapping["ymax"])
	if err != nil {
		return
	}

	n := min(len(xVals), len(yVals), len(yminVals), len(ymaxVals))

	cr, cg, cb := colormap.ParseRGB(dc.Params.Color, 0.2, 0.2, 0.2)

	alpha := dc.Params.Alpha
	if alpha <= 0 {
		alpha = 1.0
	}

	lw := dc.Params.LineWidth
	if lw <= 0 {
		lw = 1
	}

	sz := dc.Params.Size
	if sz <= 0 {
		sz = 3 //nolint:mnd // Default point size.
	}

	for i := range n {
		nx := normalize(xVals[i], dc.XMin, dc.XMax)
		ny := normalize(yVals[i], dc.YMin, dc.YMax)
		nyLo := normalize(yminVals[i], dc.YMin, dc.YMax)
		nyHi := normalize(ymaxVals[i], dc.YMin, dc.YMax)

		px, py := dc.Coord.Transform(nx, ny, dc.W, dc.H)
		_, pyLo := dc.Coord.Transform(nx, nyLo, dc.W, dc.H)
		_, pyHi := dc.Coord.Transform(nx, nyHi, dc.W, dc.H)

		dc.SetRowMetadata(i)

		// Vertical line from ymin to ymax.
		dc.Canvas.SetRGBA(cr, cg, cb, alpha)
		dc.Canvas.SetLineWidth(lw)
		dc.Canvas.MoveTo(px, pyLo)
		dc.Canvas.LineTo(px, pyHi)
		dc.Canvas.Stroke()

		// Point at (x, y).
		dc.Canvas.DrawCircle(px, py, sz)
		dc.Canvas.SetRGBA(cr, cg, cb, alpha)
		dc.Canvas.Fill()
	}
}

// ---------------------------------------------------------------------------
// Curve (quadratic bezier via Canvas.QuadraticTo)
// ---------------------------------------------------------------------------

func drawCurveFn(dc DrawContext) {
	xVals, err := dc.Data.Float64(dc.Mapping["x"])
	if err != nil {
		return
	}

	yVals, err := dc.Data.Float64(dc.Mapping["y"])
	if err != nil {
		return
	}

	xeVals, err := dc.Data.Float64(dc.Mapping["xend"])
	if err != nil {
		return
	}

	yeVals, err := dc.Data.Float64(dc.Mapping["yend"])
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

	curvature := dc.Params.Curvature

	dc.Canvas.SetLineWidth(lw)

	for i := range n {
		nx0 := normalize(xVals[i], dc.XMin, dc.XMax)
		ny0 := normalize(yVals[i], dc.YMin, dc.YMax)
		nx1 := normalize(xeVals[i], dc.XMin, dc.XMax)
		ny1 := normalize(yeVals[i], dc.YMin, dc.YMax)

		px0, py0 := dc.Coord.Transform(nx0, ny0, dc.W, dc.H)
		px1, py1 := dc.Coord.Transform(nx1, ny1, dc.W, dc.H)

		// Compute control point: offset perpendicular to the midpoint.
		mx := (px0 + px1) / 2 //nolint:mnd // Midpoint x.
		my := (py0 + py1) / 2 //nolint:mnd // Midpoint y.
		dx := px1 - px0
		dy := py1 - py0

		// Perpendicular direction scaled by curvature * segment length.
		segLen := math.Sqrt(dx*dx + dy*dy)
		cx := mx - dy*curvature
		cy := my + dx/segLen*segLen*curvature

		dc.SetRowMetadata(i)

		dc.Canvas.SetRGBA(cr, cg, cb, alpha)
		dc.Canvas.MoveTo(px0, py0)
		dc.Canvas.QuadraticTo(cx, cy, px1, py1)
		dc.Canvas.Stroke()
	}
}

// ---------------------------------------------------------------------------
// Violin (mirrored KDE polygon per group)
// ---------------------------------------------------------------------------

func drawViolinFn(dc DrawContext) {
	xVals, err := dc.Data.Float64(dc.Mapping["x"])
	if err != nil {
		return
	}

	yVals, err := dc.Data.Float64(dc.Mapping["y"])
	if err != nil {
		return
	}

	xminVals, err := dc.Data.Float64("xmin")
	if err != nil {
		return
	}

	xmaxVals, err := dc.Data.Float64("xmax")
	if err != nil {
		return
	}

	n := min(len(xVals), len(yVals), len(xminVals), len(xmaxVals))
	if n == 0 {
		return
	}

	fr, fg, fb := colormap.ParseRGB(dc.Params.Fill, 0.7, 0.7, 0.9)
	cr, cg, cb := colormap.ParseRGB(dc.Params.Color, 0.2, 0.2, 0.2)

	alpha := dc.Params.Alpha
	if alpha <= 0 {
		alpha = 0.6 //nolint:mnd // Default alpha for violin fill.
	}

	lw := dc.Params.LineWidth
	if lw <= 0 {
		lw = 1
	}

	// Width scaling: violin half-width is relative to available space.
	violinW := dc.Params.Width
	if violinW <= 0 {
		violinW = 0.8 //nolint:mnd // Default violin width fraction.
	}

	// Group rows by x position.
	type violinRow struct {
		x, y, xmin, xmax float64
	}

	groups := make(map[float64][]violinRow)

	for i := range n {
		r := violinRow{
			x:    xVals[i],
			y:    yVals[i],
			xmin: xminVals[i],
			xmax: xmaxVals[i],
		}
		groups[r.x] = append(groups[r.x], r)
	}

	for _, rows := range groups {
		if len(rows) < 2 { //nolint:mnd // Need at least 2 points for a polygon.
			continue
		}

		// Build polygon: right side (xmax) top-to-bottom, then left side (xmin) bottom-to-top.
		nR := len(rows)

		// Violin is a single polygon per group — set group-level metadata.
		dc.SetRowMetadata(0)

		// Right side: walk from first to last.
		dc.Canvas.SetRGBA(fr, fg, fb, alpha)

		nx0 := normalize(rows[0].xmax*violinW+(1-violinW)*rows[0].x, dc.XMin, dc.XMax)
		ny0 := normalize(rows[0].y, dc.YMin, dc.YMax)
		px0, py0 := dc.Coord.Transform(nx0, ny0, dc.W, dc.H)
		dc.Canvas.MoveTo(px0, py0)

		for j := 1; j < nR; j++ {
			nxj := normalize(rows[j].xmax*violinW+(1-violinW)*rows[j].x, dc.XMin, dc.XMax)
			nyj := normalize(rows[j].y, dc.YMin, dc.YMax)
			pxj, pyj := dc.Coord.Transform(nxj, nyj, dc.W, dc.H)
			dc.Canvas.LineTo(pxj, pyj)
		}

		// Left side: walk back from last to first.
		for j := nR - 1; j >= 0; j-- {
			nxj := normalize(rows[j].xmin*violinW+(1-violinW)*rows[j].x, dc.XMin, dc.XMax)
			nyj := normalize(rows[j].y, dc.YMin, dc.YMax)
			pxj, pyj := dc.Coord.Transform(nxj, nyj, dc.W, dc.H)
			dc.Canvas.LineTo(pxj, pyj)
		}

		dc.Canvas.ClosePath()
		dc.Canvas.Fill()

		// Stroke outline.
		dc.Canvas.SetRGBA(cr, cg, cb, alpha)
		dc.Canvas.SetLineWidth(lw)

		// Right side.
		nx0 = normalize(rows[0].xmax*violinW+(1-violinW)*rows[0].x, dc.XMin, dc.XMax)
		ny0 = normalize(rows[0].y, dc.YMin, dc.YMax)
		px0, py0 = dc.Coord.Transform(nx0, ny0, dc.W, dc.H)
		dc.Canvas.MoveTo(px0, py0)

		for j := 1; j < nR; j++ {
			nxj := normalize(rows[j].xmax*violinW+(1-violinW)*rows[j].x, dc.XMin, dc.XMax)
			nyj := normalize(rows[j].y, dc.YMin, dc.YMax)
			pxj, pyj := dc.Coord.Transform(nxj, nyj, dc.W, dc.H)
			dc.Canvas.LineTo(pxj, pyj)
		}

		// Left side.
		for j := nR - 1; j >= 0; j-- {
			nxj := normalize(rows[j].xmin*violinW+(1-violinW)*rows[j].x, dc.XMin, dc.XMax)
			nyj := normalize(rows[j].y, dc.YMin, dc.YMax)
			pxj, pyj := dc.Coord.Transform(nxj, nyj, dc.W, dc.H)
			dc.Canvas.LineTo(pxj, pyj)
		}

		dc.Canvas.ClosePath()
		dc.Canvas.Stroke()
	}
}

// ---------------------------------------------------------------------------
// Dotplot (stacked dots at binned positions)
// ---------------------------------------------------------------------------

func drawDotplotFn(dc DrawContext) {
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

	fr, fg, fb := colormap.ParseRGB(dc.Params.Fill, 0.3, 0.3, 0.3)
	cr, cg, cb := colormap.ParseRGB(dc.Params.Color, 0.2, 0.2, 0.2)

	alpha := dc.Params.Alpha
	if alpha <= 0 {
		alpha = 0.8 //nolint:mnd // Default alpha for dotplot.
	}

	lw := dc.Params.LineWidth
	if lw <= 0 {
		lw = 0.5 //nolint:mnd // Thin stroke around dots.
	}

	// Compute dot radius from Y domain so adjacent dots (y=0, y=1) touch.
	// One unit in data space → pixel distance via transform difference.
	_, py0 := dc.Coord.Transform(0, normalize(0, dc.YMin, dc.YMax), dc.W, dc.H)
	_, py1 := dc.Coord.Transform(0, normalize(1, dc.YMin, dc.YMax), dc.W, dc.H)
	unitPixels := math.Abs(py0 - py1)

	sz := unitPixels / 2 //nolint:mnd // Radius = half the distance between adjacent stack levels.
	if dc.Params.Size > 0 {
		sz = dc.Params.Size
	}

	for i := range n {
		nx := normalize(xVals[i], dc.XMin, dc.XMax)
		// Center each dot at y + 0.5 so the first dot sits above the baseline.
		ny := normalize(yVals[i]+0.5, dc.YMin, dc.YMax) //nolint:mnd // 0.5 offset centers dot in its stack slot.
		px, py := dc.Coord.Transform(nx, ny, dc.W, dc.H)

		dc.SetRowMetadata(i)

		// Fill.
		dc.Canvas.SetRGBA(fr, fg, fb, alpha)
		dc.Canvas.DrawCircle(px, py, sz)
		dc.Canvas.Fill()

		// Stroke.
		dc.Canvas.SetRGBA(cr, cg, cb, alpha)
		dc.Canvas.SetLineWidth(lw)
		dc.Canvas.DrawCircle(px, py, sz)
		dc.Canvas.Stroke()
	}
}

// drawRasterFn renders a dense pixel-aligned image grid. Each row in the
// dataset is a cell at (x, y) with a fill value mapped through the continuous
// color scale. The grid is composited as a single image.RGBA rather than
// drawing individual rectangles (much faster for dense grids like 500×500).
func drawRasterFn(dc DrawContext) {
	xCol, yCol := dc.Mapping["x"], dc.Mapping["y"]
	xVals, errX := dc.Data.Float64(xCol)
	yVals, errY := dc.Data.Float64(yCol)

	if errX != nil || errY != nil || len(xVals) == 0 {
		return
	}

	n := min(len(xVals), len(yVals))
	if n == 0 {
		return
	}

	// Read continuous color column.
	var zVals []float64
	if dc.ContColorCol != "" {
		zVals, _ = dc.Data.Float64(dc.ContColorCol)
	}

	if len(zVals) == 0 || dc.ContScale == nil {
		return // raster requires continuous fill values and a color scale
	}

	alpha := dc.Params.Alpha
	if alpha <= 0 {
		alpha = 1.0
	}

	alphaU8 := uint8(alpha * 255) //nolint:mnd // Alpha is clamped [0,1]; 255 is max uint8.

	// Detect grid: find unique sorted x and y values.
	xSet := make(map[float64]struct{}, n)
	ySet := make(map[float64]struct{}, n)

	for i := range n {
		xSet[xVals[i]] = struct{}{}
		ySet[yVals[i]] = struct{}{}
	}

	sortedX := make([]float64, 0, len(xSet))
	for v := range xSet {
		sortedX = append(sortedX, v)
	}

	sort.Float64s(sortedX)

	sortedY := make([]float64, 0, len(ySet))
	for v := range ySet {
		sortedY = append(sortedY, v)
	}

	sort.Float64s(sortedY)

	nx := len(sortedX)
	ny := len(sortedY)

	if nx == 0 || ny == 0 {
		return
	}

	// Build reverse lookup: value → index.
	xIdx := make(map[float64]int, nx)
	for i, v := range sortedX {
		xIdx[v] = i
	}

	yIdx := make(map[float64]int, ny)
	for i, v := range sortedY {
		yIdx[v] = i
	}

	// Build the raster image — (nx, ny), Y=0 is top (highest data y).
	img := image.NewRGBA(image.Rect(0, 0, nx, ny))

	// Fill with transparent initially.
	for i := range img.Pix {
		img.Pix[i] = 0
	}

	cm := dc.ContScale

	for i := range n {
		ix, okX := xIdx[xVals[i]]
		iy, okY := yIdx[yVals[i]]

		if !okX || !okY || i >= len(zVals) {
			continue
		}

		gc := cm.At(zVals[i])
		r := uint8(gc.R * 255) //nolint:mnd // Normalized [0,1] → [0,255].
		g := uint8(gc.G * 255) //nolint:mnd // Normalized [0,1] → [0,255].
		b := uint8(gc.B * 255) //nolint:mnd // Normalized [0,1] → [0,255].

		// Invert Y: highest data y → row 0 (top of image).
		imgY := ny - 1 - iy
		img.SetRGBA(ix, imgY, color.RGBA{R: r, G: g, B: b, A: alphaU8})
	}

	// Compute screen rectangle covering the data extent.
	nxMin := normalize(sortedX[0], dc.XMin, dc.XMax)
	nxMax := normalize(sortedX[nx-1], dc.XMin, dc.XMax)
	nyMin := normalize(sortedY[0], dc.YMin, dc.YMax)
	nyMax := normalize(sortedY[ny-1], dc.YMin, dc.YMax)

	// Add half-cell padding so pixels are centered on data points.
	if nx > 1 {
		halfCellX := (nxMax - nxMin) / float64(nx-1) / 2
		nxMin -= halfCellX
		nxMax += halfCellX
	} else {
		// Single column: expand to a visible width.
		nxMin = 0
		nxMax = 1
	}

	if ny > 1 {
		halfCellY := (nyMax - nyMin) / float64(ny-1) / 2
		nyMin -= halfCellY
		nyMax += halfCellY
	} else {
		nyMin = 0
		nyMax = 1
	}

	// Transform to pixel coordinates.
	px0, py0 := dc.Coord.Transform(nxMin, nyMax, dc.W, dc.H)
	px1, py1 := dc.Coord.Transform(nxMax, nyMin, dc.W, dc.H)

	destW := math.Abs(px1 - px0)
	destH := math.Abs(py1 - py0)
	destX := math.Min(px0, px1)
	destY := math.Min(py0, py1)

	if destW < 1 || destH < 1 {
		return
	}

	// Composite via native canvas transforms: the gg backend's DrawImage
	// respects the current transform matrix, so ScaleXY lets the GPU
	// texture sampler (or gg's internal rasterizer) handle upscaling.
	// This avoids a Go-side pixel loop and enables GPU acceleration
	// on the window surface.
	dc.Canvas.Save()
	dc.Canvas.Translate(destX, destY)
	dc.Canvas.ScaleXY(destW/float64(nx), destH/float64(ny))
	dc.Canvas.DrawImage(img, 0, 0)
	dc.Canvas.Restore()
}

// ---------------------------------------------------------------------------
// Annotations — fixed-coordinate visual elements drawn after data layers
// ---------------------------------------------------------------------------

// drawAnnotation dispatches rendering for a single [Annotation] in data space.
// Annotations bypass the data pipeline; their coordinates are normalized
// against the panel's trained scale bounds and transformed through the
// active coordinate system.
func drawAnnotation(
	cv canvas.Canvas,
	c coord.Coord,
	ann *Annotation,
	w, h, xMin, xMax, yMin, yMax float64,
	th theme.Theme,
) {
	switch ann.Type {
	case AnnotationText:
		drawAnnotationText(cv, c, ann, w, h, xMin, xMax, yMin, yMax, th)
	case AnnotationRect:
		drawAnnotationRect(cv, c, ann, w, h, xMin, xMax, yMin, yMax)
	case AnnotationSegment:
		drawAnnotationSegment(cv, c, ann, w, h, xMin, xMax, yMin, yMax)
	case AnnotationArrow:
		drawAnnotationArrow(cv, c, ann, w, h, xMin, xMax, yMin, yMax)
	case AnnotationLabel:
		drawAnnotationLabel(cv, c, ann, w, h, xMin, xMax, yMin, yMax, th)
	}
}

// drawAnnotationText renders a text annotation at fixed data coordinates.
func drawAnnotationText(
	cv canvas.Canvas,
	c coord.Coord,
	ann *Annotation,
	w, h, xMin, xMax, yMin, yMax float64,
	th theme.Theme,
) {
	nx := normalize(ann.X, xMin, xMax)
	ny := normalize(ann.Y, yMin, yMax)
	px, py := c.Transform(nx, ny, w, h)

	fontSize := ann.Params.FontSize
	if fontSize <= 0 {
		annText := th.AnnotationText()
		fontSize = annText.Size

		if fontSize <= 0 {
			fontSize = 10 //nolint:mnd // Default annotation font size.
		}
	}

	alpha := ann.Params.Alpha
	if alpha <= 0 {
		alpha = 1
	}

	cr, cg, cb := colormap.ParseRGB(ann.Params.Color, 0.1, 0.1, 0.1)

	cv.SetFontSize(fontSize)
	cv.SetRGBA(cr, cg, cb, alpha)
	cv.DrawStringAnchored(ann.Label, px, py-fontSize*0.3, 0.5, 0.5) //nolint:mnd // Centre anchor with slight upward offset.
}

// drawAnnotationRect renders a filled rectangle annotation spanning
// (xmin, ymin) to (xmax, ymax) in data coordinates.
func drawAnnotationRect(
	cv canvas.Canvas,
	c coord.Coord,
	ann *Annotation,
	w, h, xMin, xMax, yMin, yMax float64,
) {
	nx1 := normalize(ann.X, xMin, xMax)
	ny1 := normalize(ann.Y, yMin, yMax)
	nx2 := normalize(ann.XEnd, xMin, xMax)
	ny2 := normalize(ann.YEnd, yMin, yMax)

	px1, py1 := c.Transform(nx1, ny1, w, h)
	px2, py2 := c.Transform(nx2, ny2, w, h)

	// Ensure correct rectangle orientation (top-left, bottom-right).
	if px1 > px2 {
		px1, px2 = px2, px1
	}

	if py1 > py2 {
		py1, py2 = py2, py1
	}

	alpha := ann.Params.Alpha
	if alpha <= 0 {
		alpha = 0.3 //nolint:mnd // Default rect annotation alpha.
	}

	// Fill.
	if ann.Params.Fill != "" {
		fr, fg, fb := colormap.ParseRGB(ann.Params.Fill, 0.9, 0.85, 0.85)
		cv.SetRGBA(fr, fg, fb, alpha)
		cv.DrawRectangle(px1, py1, px2-px1, py2-py1)
		cv.Fill()
	}

	// Stroke.
	if ann.Params.Color != "" {
		lw := ann.Params.LineWidth
		if lw <= 0 {
			lw = 1
		}

		cr, cg, cb := colormap.ParseRGB(ann.Params.Color, 0.5, 0.5, 0.5)
		cv.SetRGBA(cr, cg, cb, alpha)
		cv.SetLineWidth(lw)
		cv.DrawRectangle(px1, py1, px2-px1, py2-py1)
		cv.Stroke()
	}

	// If neither fill nor color, draw a default semi-transparent rect.
	if ann.Params.Fill == "" && ann.Params.Color == "" {
		cv.SetRGBA(0.9, 0.85, 0.85, alpha) //nolint:mnd // Default rect color.
		cv.DrawRectangle(px1, py1, px2-px1, py2-py1)
		cv.Fill()
	}
}

// drawAnnotationSegment renders a line segment from (X, Y) to (XEnd, YEnd).
func drawAnnotationSegment(
	cv canvas.Canvas,
	c coord.Coord,
	ann *Annotation,
	w, h, xMin, xMax, yMin, yMax float64,
) {
	nx1 := normalize(ann.X, xMin, xMax)
	ny1 := normalize(ann.Y, yMin, yMax)
	nx2 := normalize(ann.XEnd, xMin, xMax)
	ny2 := normalize(ann.YEnd, yMin, yMax)

	px1, py1 := c.Transform(nx1, ny1, w, h)
	px2, py2 := c.Transform(nx2, ny2, w, h)

	lw := ann.Params.LineWidth
	if lw <= 0 {
		lw = 1.5 //nolint:mnd // Default annotation segment line width.
	}

	alpha := ann.Params.Alpha
	if alpha <= 0 {
		alpha = 0.8 //nolint:mnd // Default annotation segment alpha.
	}

	cr, cg, cb := colormap.ParseRGB(ann.Params.Color, 0.2, 0.2, 0.2)

	cv.SetRGBA(cr, cg, cb, alpha)
	cv.SetLineWidth(lw)
	cv.MoveTo(px1, py1)
	cv.LineTo(px2, py2)
	cv.Stroke()
}

// drawAnnotationArrow renders a segment with an arrowhead at (XEnd, YEnd).
func drawAnnotationArrow(
	cv canvas.Canvas,
	c coord.Coord,
	ann *Annotation,
	w, h, xMin, xMax, yMin, yMax float64,
) {
	nx1 := normalize(ann.X, xMin, xMax)
	ny1 := normalize(ann.Y, yMin, yMax)
	nx2 := normalize(ann.XEnd, xMin, xMax)
	ny2 := normalize(ann.YEnd, yMin, yMax)

	px1, py1 := c.Transform(nx1, ny1, w, h)
	px2, py2 := c.Transform(nx2, ny2, w, h)

	lw := ann.Params.LineWidth
	if lw <= 0 {
		lw = 1.5 //nolint:mnd // Default arrow line width.
	}

	alpha := ann.Params.Alpha
	if alpha <= 0 {
		alpha = 0.8 //nolint:mnd // Default arrow alpha.
	}

	cr, cg, cb := colormap.ParseRGB(ann.Params.Color, 0.2, 0.2, 0.2)

	// Draw shaft.
	cv.SetRGBA(cr, cg, cb, alpha)
	cv.SetLineWidth(lw)
	cv.MoveTo(px1, py1)
	cv.LineTo(px2, py2)
	cv.Stroke()

	// Draw arrowhead — a filled triangle at (px2, py2) pointing along the shaft.
	arrowSize := max(lw*4, 8) //nolint:mnd // Arrowhead size proportional to line width, minimum 8px.
	dx := px2 - px1
	dy := py2 - py1
	length := math.Sqrt(dx*dx + dy*dy)

	if length < 1 {
		return // degenerate: start == end
	}

	// Unit vector along shaft.
	ux := dx / length
	uy := dy / length

	// Perpendicular unit vector.
	perpX := -uy
	perpY := ux

	// Arrowhead vertices.
	halfW := arrowSize * 0.4 //nolint:mnd // Arrowhead half-width = 40% of size.
	tipX, tipY := px2, py2
	baseX := px2 - ux*arrowSize
	baseY := py2 - uy*arrowSize

	cv.SetRGBA(cr, cg, cb, alpha)
	cv.MoveTo(tipX, tipY)
	cv.LineTo(baseX+perpX*halfW, baseY+perpY*halfW)
	cv.LineTo(baseX-perpX*halfW, baseY-perpY*halfW)
	cv.ClosePath()
	cv.Fill()
}

// drawAnnotationLabel renders text with a filled background box at (X, Y).
func drawAnnotationLabel(
	cv canvas.Canvas,
	c coord.Coord,
	ann *Annotation,
	w, h, xMin, xMax, yMin, yMax float64,
	th theme.Theme,
) {
	nx := normalize(ann.X, xMin, xMax)
	ny := normalize(ann.Y, yMin, yMax)
	px, py := c.Transform(nx, ny, w, h)

	fontSize := ann.Params.FontSize
	if fontSize <= 0 {
		annText := th.AnnotationText()
		fontSize = annText.Size

		if fontSize <= 0 {
			fontSize = 10 //nolint:mnd // Default annotation font size.
		}
	}

	padding := ann.Params.Padding
	if padding <= 0 {
		padding = 4 //nolint:mnd // Default label padding.
	}

	alpha := ann.Params.Alpha
	if alpha <= 0 {
		alpha = 0.75 //nolint:mnd // Default label alpha — semi-transparent so data shows through.
	}

	// Measure text.
	cv.SetFontSize(fontSize)

	tw, th2 := cv.MeasureString(ann.Label)
	if th2 <= 0 {
		th2 = fontSize * 1.2 //nolint:mnd // Fallback text height.
	}

	// Background box dimensions (centred on anchor).
	boxW := tw + padding*2  //nolint:mnd // 2 sides.
	boxH := th2 + padding*2 //nolint:mnd // 2 sides.
	boxX := px - boxW/2     //nolint:mnd // Centre horizontally.
	boxY := py - boxH/2     //nolint:mnd // Centre vertically.

	// Draw background.
	fr, fg, fb := colormap.ParseRGB(ann.Params.Fill, 1, 1, 1) // default white
	cv.SetRGBA(fr, fg, fb, alpha)
	cv.DrawRectangle(boxX, boxY, boxW, boxH)
	cv.Fill()

	// Draw border.
	cr, cg, cb := colormap.ParseRGB(ann.Params.Color, 0.3, 0.3, 0.3) // default dark grey

	lw := ann.Params.LineWidth
	if lw <= 0 {
		lw = 0.5 //nolint:mnd // Thin border for label box.
	}

	cv.SetRGBA(cr, cg, cb, alpha)
	cv.SetLineWidth(lw)
	cv.DrawRectangle(boxX, boxY, boxW, boxH)
	cv.Stroke()

	// Draw text (centred in box).
	cv.SetRGBA(cr, cg, cb, 1) // text always fully opaque
	cv.DrawStringAnchored(ann.Label, px, py, 0.5, 0.5)
}
