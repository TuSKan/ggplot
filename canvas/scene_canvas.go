package canvas

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/scene"
	"github.com/gogpu/gg/text"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/TuSKan/ggplot/fonts"
)

// SceneCanvas implements [Canvas] by recording drawing commands into a
// [scene.Scene], the GPU-accelerated scene graph used by gogpu/gg.
//
// Instead of rasterizing to a pixel buffer (like [RasterCanvas]), all paths,
// fills, strokes, and text are recorded as vector scene commands that the GPU
// renders via SDF shapes, MSDF text, and hardware path processing.
//
// This gives:
//   - GPU-accelerated rendering (no CPU rasterization)
//   - Resolution independence (zoom without re-rasterization)
//   - Coordinate alignment with gogpu/ui widgets (for hit-testing, tooltips)
//
// SceneCanvas is NOT thread-safe. All drawing must occur on a single goroutine
// (typically the UI/draw goroutine).
type SceneCanvas struct {
	sc   *scene.Scene
	w, h int

	// --- Current drawing state ---
	path      *scene.Path // accumulates path commands; nil when empty
	color     gg.RGBA     // current fill/stroke color (straight alpha, [0,1])
	lineWidth float32     // current stroke width in logical pixels
	dashPat   []float64   // current dash pattern (nil = solid)

	// Transform — composed affine applied to all drawing operations.
	// scene.Scene also maintains its own transform stack (PushTransform/
	// PopTransform). We use the scene's stack so that fills, strokes, clips,
	// and text all see the same combined transform.
	//
	// NOTE: we track transformDepth so that Restore can undo transforms
	// pushed by the matching Save.

	// --- State stack (Save/Restore) ---
	stateStack []drawState

	// clipDepth tracks how many PushClip calls have been made since the last
	// Save. Restore pops clips back to the matching Save's depth.
	clipDepth int

	// transformDepth tracks how many PushTransform calls have been made since
	// the last Save. Restore pops transforms back to the matching depth.
	transformDepth int

	// --- Font state ---
	fontSize float64
	tabNums  bool
	face     text.Face        // cached resolved face at current fontSize
	source   *text.FontSource // cached font source for current settings
}

// drawState is a snapshot of the mutable canvas state saved by [Save].
type drawState struct {
	color          gg.RGBA
	lineWidth      float32
	dashPat        []float64
	clipDepth      int
	transformDepth int
	fontSize       float64
	tabNums        bool
	face           text.Face
	source         *text.FontSource
}

// NewSceneCanvas creates a Canvas backed by the given [scene.Scene].
// Width and height define the logical canvas dimensions in pixels.
//
// The scene must already be created (typically from gogpu/ui's rendering
// pipeline). Drawing commands are appended to the scene's current encoding.
func NewSceneCanvas(sc *scene.Scene, width, height int) *SceneCanvas {
	initFonts() // ensure fonts are ready (shared with RasterCanvas)

	c := &SceneCanvas{
		sc:        sc,
		w:         width,
		h:         height,
		color:     gg.RGBA{R: 0, G: 0, B: 0, A: 1}, // default: opaque black
		lineWidth: 1,
	}
	c.resolveFont(12) //nolint:mnd // default font size

	return c
}

// Scene returns the underlying scene.Scene.
func (c *SceneCanvas) Scene() *scene.Scene { return c.sc }

// Compile-time check.
var _ Canvas = (*SceneCanvas)(nil)

// ---------- State management ----------

// Save pushes the current graphics state onto the internal stack.
func (c *SceneCanvas) Save() {
	c.stateStack = append(c.stateStack, drawState{
		color:          c.color,
		lineWidth:      c.lineWidth,
		dashPat:        c.dashPat,
		clipDepth:      c.clipDepth,
		transformDepth: c.transformDepth,
		fontSize:       c.fontSize,
		tabNums:        c.tabNums,
		face:           c.face,
		source:         c.source,
	})
}

// Restore pops the graphics state from the stack, undoing any transforms and
// clips added since the matching [Save].
func (c *SceneCanvas) Restore() {
	if len(c.stateStack) == 0 {
		return
	}

	st := c.stateStack[len(c.stateStack)-1]
	c.stateStack = c.stateStack[:len(c.stateStack)-1]

	// Pop clips added since the matching Save.
	for c.clipDepth > st.clipDepth {
		c.sc.PopClip()
		c.clipDepth--
	}

	// Pop transforms added since the matching Save.
	for c.transformDepth > st.transformDepth {
		c.sc.PopTransform()
		c.transformDepth--
	}

	c.color = st.color
	c.lineWidth = st.lineWidth
	c.dashPat = st.dashPat
	c.fontSize = st.fontSize
	c.tabNums = st.tabNums
	c.face = st.face
	c.source = st.source
}

// ---------- Transforms ----------

// Translate shifts the origin by (dx, dy).
func (c *SceneCanvas) Translate(dx, dy float64) {
	c.sc.PushTransform(scene.TranslateAffine(float32(dx), float32(dy)))
	c.transformDepth++
}

// Rotate applies a rotation of angle radians around the current origin.
func (c *SceneCanvas) Rotate(angle float64) {
	c.sc.PushTransform(scene.RotateAffine(float32(angle)))
	c.transformDepth++
}

// ScaleXY applies a non-uniform scale.
func (c *SceneCanvas) ScaleXY(sx, sy float64) {
	c.sc.PushTransform(scene.ScaleAffine(float32(sx), float32(sy)))
	c.transformDepth++
}

// ---------- Path construction ----------

func (c *SceneCanvas) ensurePath() *scene.Path {
	if c.path == nil {
		c.path = scene.NewPath()
	}

	return c.path
}

// MoveTo starts a new sub-path at (x, y).
func (c *SceneCanvas) MoveTo(x, y float64) {
	c.ensurePath().MoveTo(float32(x), float32(y))
}

// LineTo adds a line segment from the current point to (x, y).
func (c *SceneCanvas) LineTo(x, y float64) {
	c.ensurePath().LineTo(float32(x), float32(y))
}

// QuadraticTo adds a quadratic Bézier curve to (x, y) via control (cx, cy).
func (c *SceneCanvas) QuadraticTo(cx, cy, x, y float64) {
	c.ensurePath().QuadTo(float32(cx), float32(cy), float32(x), float32(y))
}

// CubicTo adds a cubic Bézier curve to (x, y) via control points.
func (c *SceneCanvas) CubicTo(cx1, cy1, cx2, cy2, x, y float64) {
	c.ensurePath().CubicTo(
		float32(cx1), float32(cy1),
		float32(cx2), float32(cy2),
		float32(x), float32(y),
	)
}

// ClosePath closes the current sub-path.
func (c *SceneCanvas) ClosePath() {
	if c.path != nil {
		c.path.Close()
	}
}

// ClearPath discards the current path without drawing.
func (c *SceneCanvas) ClearPath() {
	c.path = nil
}

// ---------- Drawing ----------

// SetColor sets the current drawing color for both fill and stroke.
func (c *SceneCanvas) SetColor(col color.Color) {
	if col == nil {
		col = color.Black
	}

	c.color = gg.FromColor(col)
}

// SetRGBA sets the current drawing color from normalized components.
func (c *SceneCanvas) SetRGBA(r, g, b, a float64) {
	c.color = gg.RGBA{R: r, G: g, B: b, A: a}
}

// SetLineWidth sets the stroke width in pixels.
func (c *SceneCanvas) SetLineWidth(w float64) {
	c.lineWidth = float32(w)
}

// SetLineDash sets the dash pattern. Pass nil or empty for solid lines.
//
// When a dash pattern is active, Stroke() decomposes the current path into
// dashed sub-paths before recording them as scene strokes. This emulates
// the Cairo/gg dash behavior at the scene-recording level since the GPU
// scene graph does not natively support dash patterns.
func (c *SceneCanvas) SetLineDash(pattern ...float64) {
	if len(pattern) == 0 {
		c.dashPat = nil

		return
	}

	c.dashPat = make([]float64, len(pattern))
	copy(c.dashPat, pattern)
}

// brush returns a solid brush from the current color.
func (c *SceneCanvas) brush() scene.Brush {
	return scene.SolidBrush(c.color)
}

// strokeStyle returns the current stroke style.
func (c *SceneCanvas) strokeStyle() *scene.StrokeStyle {
	return &scene.StrokeStyle{
		Width:      c.lineWidth,
		MiterLimit: 10, //nolint:mnd // Standard SVG/Canvas miter limit default.
		Cap:        scene.LineCapButt,
		Join:       scene.LineJoinMiter,
	}
}

// Fill fills the current path with the current color, then clears the path.
func (c *SceneCanvas) Fill() {
	if c.path == nil || c.path.IsEmpty() {
		return
	}

	c.sc.Fill(
		scene.FillNonZero,
		scene.IdentityAffine(),
		c.brush(),
		scene.NewPathShape(c.path),
	)
	c.path = nil
}

// Stroke draws the current path outline, then clears the path.
func (c *SceneCanvas) Stroke() {
	if c.path == nil || c.path.IsEmpty() {
		return
	}

	if len(c.dashPat) > 0 {
		c.strokeDashed(c.path)
	} else {
		c.sc.Stroke(
			c.strokeStyle(),
			scene.IdentityAffine(),
			c.brush(),
			scene.NewPathShape(c.path),
		)
	}

	c.path = nil
}

// FillPreserve fills without clearing the path (allows subsequent stroke).
func (c *SceneCanvas) FillPreserve() {
	if c.path == nil || c.path.IsEmpty() {
		return
	}

	c.sc.Fill(
		scene.FillNonZero,
		scene.IdentityAffine(),
		c.brush(),
		scene.NewPathShape(c.path),
	)
	// path NOT cleared — preserved for subsequent Stroke
}

// Clip clips rendering to the current path. Call [Save]/[Restore] to undo.
func (c *SceneCanvas) Clip() {
	if c.path == nil || c.path.IsEmpty() {
		return
	}

	c.sc.PushClip(scene.NewPathShape(c.path))
	c.clipDepth++
	c.path = nil
}

// ---------- Convenience shapes ----------

// DrawCircle adds a circle path centered at (cx, cy) with radius r.
func (c *SceneCanvas) DrawCircle(cx, cy, r float64) {
	c.ensurePath().Circle(float32(cx), float32(cy), float32(r))
}

// DrawRectangle adds a rectangle path at (x, y) with dimensions (w, h).
func (c *SceneCanvas) DrawRectangle(x, y, w, h float64) {
	c.ensurePath().Rectangle(float32(x), float32(y), float32(w), float32(h))
}

// DrawLine adds a line path from (x1, y1) to (x2, y2).
func (c *SceneCanvas) DrawLine(x1, y1, x2, y2 float64) {
	p := c.ensurePath()
	p.MoveTo(float32(x1), float32(y1))
	p.LineTo(float32(x2), float32(y2))
}

// DrawShape adds a path for the specified shape centered at (cx, cy) with
// size/radius r. Supported shapes: "circle", "square", "triangle", "diamond",
// "triangleDown", "plus", "cross", "star", "pentagon", "hexagon".
func (c *SceneCanvas) DrawShape(shape string, cx, cy, r float64) {
	switch shape {
	case ShapeCircle:
		c.DrawCircle(cx, cy, r)
	case ShapeSquare:
		c.drawRegularPolygon(4, cx, cy, r, 0) //nolint:mnd // 4-sided polygon
	case ShapeTriangle:
		c.drawRegularPolygon(3, cx, cy, r, 0) //nolint:mnd // 3-sided polygon
	case ShapeTriangleDown:
		c.drawRegularPolygon(3, cx, cy, r, math.Pi) //nolint:mnd // 3-sided polygon rotated π
	case ShapeDiamond:
		c.drawRegularPolygon(4, cx, cy, r, math.Pi/4) //nolint:mnd // 4-sided polygon rotated 45°
	case ShapePentagon:
		c.drawRegularPolygon(5, cx, cy, r, 0) //nolint:mnd // 5-sided polygon
	case ShapeHexagon:
		c.drawRegularPolygon(6, cx, cy, r, 0) //nolint:mnd // 6-sided polygon
	case ShapePlus:
		c.DrawLine(cx, cy-r, cx, cy+r)
		c.DrawLine(cx-r, cy, cx+r, cy)
	case ShapeCross:
		c.DrawLine(cx-r, cy-r, cx+r, cy+r)
		c.DrawLine(cx-r, cy+r, cx+r, cy-r)
	case ShapeStar:
		DrawShapePath(c, shape, cx, cy, r)
	default:
		c.DrawCircle(cx, cy, r)
	}
}

// drawRegularPolygon adds a regular polygon path centered at (cx, cy).
func (c *SceneCanvas) drawRegularPolygon(sides int, cx, cy, r, startAngle float64) {
	p := c.ensurePath()

	for i := range sides {
		angle := startAngle + float64(i)*2*math.Pi/float64(sides) - math.Pi/2
		x := cx + r*math.Cos(angle)
		y := cy + r*math.Sin(angle)

		if i == 0 {
			p.MoveTo(float32(x), float32(y))
		} else {
			p.LineTo(float32(x), float32(y))
		}
	}

	p.Close()
}

// ---------- Text ----------

// sceneFontState holds the lazily-initialized font state for SceneCanvas text.
// This is separate from RasterCanvas's fontState because SceneCanvas uses
// scene.DrawText which shapes+records glyph runs, while RasterCanvas uses
// gg.Context.DrawStringAnchored which renders to pixels.
var (
	sceneFontOnce     sync.Once
	sceneFontResolver *fonts.Resolver
	sceneEmbedded     *text.FontSource
)

func initSceneFonts() {
	sceneFontOnce.Do(func() {
		registry, _ := fonts.NewRegistry()
		sceneFontResolver = fonts.NewResolver(registry, fonts.DefaultFallbackConfig())

		src, err := text.NewFontSource(goregular.TTF)
		if err == nil {
			sceneEmbedded = src
		}

		text.SetShaper(text.NewGoTextShaper())
	})
}

// resolveFont resolves the best available font at the given size and caches
// both the FontSource and Face for subsequent text operations.
func (c *SceneCanvas) resolveFont(size float64) {
	if size <= 0 {
		size = 12
	}

	c.fontSize = size

	var opts []text.FaceOption
	if c.tabNums {
		opts = append(opts, text.WithFeatures(text.TabularNums))
	}

	initSceneFonts()

	// Try system font resolver (sans-serif family).
	if sceneFontResolver != nil {
		handle, err := sceneFontResolver.LoadFace(fonts.FaceRequest{
			Family:        "sans-serif",
			Weight:        fonts.WeightNormal,
			AllowFallback: true,
			Size:          size,
			DPI:           72,
		})
		if err == nil && handle != nil {
			if src := handle.FontSource(); src != nil {
				c.source = src
				c.face = src.Face(size, opts...)

				return
			}
		}
	}

	// Fallback to embedded Go Regular.
	if sceneEmbedded != nil {
		c.source = sceneEmbedded
		c.face = sceneEmbedded.Face(size, opts...)
	}
}

// SetFontSize resolves the best available font at the given size.
func (c *SceneCanvas) SetFontSize(size float64) {
	c.resolveFont(size)
}

// SetTabularNums enables or disables tabular (monospaced) digit widths.
func (c *SceneCanvas) SetTabularNums(enabled bool) {
	if c.tabNums == enabled {
		return
	}

	c.tabNums = enabled
	c.resolveFont(c.fontSize) // re-resolve with new feature set
}

// DrawStringAnchored draws text at (x, y) with anchor (ax, ay) ∈ [0,1].
// (0, 0) = top-left, (0.5, 0.5) = center, (1, 1) = bottom-right.
//
// Text is recorded as a TagText glyph run in the scene. The GPU renderer
// handles text resolution (atlas, outline, SDF) at render time.
func (c *SceneCanvas) DrawStringAnchored(s string, x, y, ax, ay float64) {
	if s == "" || c.face == nil {
		return
	}

	// Measure text advance to compute anchored position.
	tw, _ := text.Measure(s, c.face)

	metrics := c.face.Metrics()
	th := metrics.Ascent + metrics.Descent

	// Anchored position: (0, 0.5) = left-aligned, vertically centered.
	drawX := x - tw*ax
	drawY := y + metrics.Ascent - th*ay

	_ = c.sc.DrawText(s, c.face, float32(drawX), float32(drawY), c.brush())
}

// MeasureString returns the width and height of the rendered text.
func (c *SceneCanvas) MeasureString(s string) (float64, float64) {
	if c.face == nil {
		return 0, 0
	}

	w, h := text.Measure(s, c.face)

	return w, h
}

// ---------- Output ----------

// Clear fills the entire canvas with the given color.
func (c *SceneCanvas) Clear(col color.Color) {
	brush := scene.SolidBrush(gg.FromColor(col))
	shape := scene.NewRectShape(0, 0, float32(c.w), float32(c.h))
	c.sc.Fill(scene.FillNonZero, scene.IdentityAffine(), brush, shape)
}

// Width returns the canvas width in logical pixels.
func (c *SceneCanvas) Width() int { return c.w }

// Height returns the canvas height in logical pixels.
func (c *SceneCanvas) Height() int { return c.h }

// DrawImage composites the given image onto the canvas at position (x, y).
func (c *SceneCanvas) DrawImage(img image.Image, x, y float64) {
	if img == nil {
		return
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= 0 || h <= 0 {
		return
	}

	rgba := imageToSceneRGBA(img)
	scImg := scene.NewImage(w, h)
	scImg.Data = rgba.Pix
	c.sc.DrawImage(scImg, scene.TranslateAffine(float32(x), float32(y)))
}

// Close is a no-op for SceneCanvas. The scene.Scene is owned by the caller
// (typically gogpu/ui's rendering pipeline) and is not closed here.
func (c *SceneCanvas) Close() error {
	c.path = nil
	c.stateStack = nil

	return nil
}

// ---------- Dash support ----------

// strokeDashed decomposes the path into dashed sub-paths and strokes each one.
// This emulates Cairo/gg dash behavior at the scene-recording level.
func (c *SceneCanvas) strokeDashed(path *scene.Path) {
	if path == nil || path.IsEmpty() || len(c.dashPat) == 0 {
		return
	}

	style := c.strokeStyle()
	brush := c.brush()

	// Flatten the path to line segments for dash decomposition.
	segments := flattenPath(path)
	if len(segments) == 0 {
		return
	}

	patLen := dashPatternLength(c.dashPat)
	if patLen <= 0 {
		// Degenerate pattern — draw solid.
		c.sc.Stroke(style, scene.IdentityAffine(), brush, scene.NewPathShape(path))

		return
	}

	// Walk the flattened line segments, alternating dash (draw) / gap (skip).
	var (
		dashPath  *scene.Path
		patIdx    int
		remaining = c.dashPat[0]
		drawing   = true // first element is dash (drawn)
	)

	for _, seg := range segments {
		dx := seg.x2 - seg.x1
		dy := seg.y2 - seg.y1
		segLen := float64(math.Sqrt(float64(dx*dx + dy*dy)))

		if segLen == 0 {
			continue
		}

		consumed := float64(0)

		for consumed < segLen {
			step := math.Min(remaining, segLen-consumed)
			frac := float32(step / segLen)

			startX := seg.x1 + float32(consumed/segLen)*dx
			startY := seg.y1 + float32(consumed/segLen)*dy
			endX := startX + frac*dx
			endY := startY + frac*dy

			if drawing {
				if dashPath == nil {
					dashPath = scene.NewPath()
				}

				dashPath.MoveTo(startX, startY)
				dashPath.LineTo(endX, endY)
			}

			consumed += step
			remaining -= step

			if remaining <= 0 {
				// Flush current dash segment.
				if drawing && dashPath != nil && !dashPath.IsEmpty() {
					c.sc.Stroke(style, scene.IdentityAffine(), brush, scene.NewPathShape(dashPath))
					dashPath = nil
				}

				drawing = !drawing
				patIdx = (patIdx + 1) % len(c.dashPat)
				remaining = c.dashPat[patIdx]
			}
		}
	}

	// Flush trailing dash.
	if drawing && dashPath != nil && !dashPath.IsEmpty() {
		c.sc.Stroke(style, scene.IdentityAffine(), brush, scene.NewPathShape(dashPath))
	}
}

// lineSegment is a flattened line segment for dash decomposition.
type lineSegment struct {
	x1, y1, x2, y2 float32
}

// flattenPath converts a scene.Path to a list of line segments by evaluating
// all curves. Used for dash decomposition.
func flattenPath(path *scene.Path) []lineSegment {
	if path == nil || path.IsEmpty() {
		return nil
	}

	var segments []lineSegment

	var curX, curY float32

	for elem := range path.Elements() {
		switch elem.Verb {
		case scene.MoveTo:
			curX, curY = elem.Points[0].X, elem.Points[0].Y
		case scene.LineTo:
			segments = append(segments, lineSegment{curX, curY, elem.Points[0].X, elem.Points[0].Y})
			curX, curY = elem.Points[0].X, elem.Points[0].Y
		case scene.QuadTo:
			// Flatten quadratic to line segments.
			flatQuad(&segments, curX, curY,
				elem.Points[0].X, elem.Points[0].Y,
				elem.Points[1].X, elem.Points[1].Y)
			curX, curY = elem.Points[1].X, elem.Points[1].Y
		case scene.CubicTo:
			// Flatten cubic to line segments.
			flatCubic(&segments, curX, curY,
				elem.Points[0].X, elem.Points[0].Y,
				elem.Points[1].X, elem.Points[1].Y,
				elem.Points[2].X, elem.Points[2].Y)
			curX, curY = elem.Points[2].X, elem.Points[2].Y
		case scene.Close:
			// Close handled implicitly if needed.
		}
	}

	return segments
}

const flattenSteps = 8 //nolint:mnd // Subdivision steps for curve flattening in dash decomposition.

// flatQuad subdivides a quadratic Bézier into line segments.
func flatQuad(segs *[]lineSegment, x0, y0, cx, cy, x1, y1 float32) {
	prevX, prevY := x0, y0

	for i := 1; i <= flattenSteps; i++ {
		t := float32(i) / flattenSteps
		mt := 1 - t

		x := mt*mt*x0 + 2*mt*t*cx + t*t*x1
		y := mt*mt*y0 + 2*mt*t*cy + t*t*y1
		*segs = append(*segs, lineSegment{prevX, prevY, x, y})
		prevX, prevY = x, y
	}
}

// flatCubic subdivides a cubic Bézier into line segments.
func flatCubic(segs *[]lineSegment, x0, y0, c1x, c1y, c2x, c2y, x1, y1 float32) {
	prevX, prevY := x0, y0

	for i := 1; i <= flattenSteps; i++ {
		t := float32(i) / flattenSteps
		mt := 1 - t

		x := mt*mt*mt*x0 + 3*mt*mt*t*c1x + 3*mt*t*t*c2x + t*t*t*x1 //nolint:mnd // Cubic Bézier formula.
		y := mt*mt*mt*y0 + 3*mt*mt*t*c1y + 3*mt*t*t*c2y + t*t*t*y1 //nolint:mnd // Cubic Bézier formula.
		*segs = append(*segs, lineSegment{prevX, prevY, x, y})
		prevX, prevY = x, y
	}
}

// dashPatternLength returns the total length of one full dash+gap cycle.
func dashPatternLength(pat []float64) float64 {
	total := 0.0
	for _, v := range pat {
		total += v
	}

	return total
}

// imageToSceneRGBA converts an image.Image to *image.RGBA.
func imageToSceneRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	return rgba
}
