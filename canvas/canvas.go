// Package canvas defines the rendering backend abstraction.
// All drawing operations go through the [Canvas] interface, which decouples
// geometry rendering from the specific graphics library (gogpu/gg, SVG, etc.).
package canvas

import (
	"image"
	"image/color"
)

// Canvas is the core rendering abstraction. Implementations provide the
// actual drawing primitives targeting specific backends (CPU rasterizer,
// GPU pipeline, SVG writer, etc.).
//
// The API is modeled after gogpu/gg's Context for natural integration,
// but abstracted so alternative backends can be swapped without changing
// any rendering logic.
type Canvas interface {
	// --- State management ---

	// Save pushes the current graphics state (color, transform, clip) onto a stack.
	Save()
	// Restore pops the graphics state stack.
	Restore()

	// --- Coordinate transforms ---

	// Translate shifts the origin by (dx, dy).
	Translate(dx, dy float64)
	// Rotate applies a rotation of angle radians around the current origin.
	Rotate(angle float64)
	// ScaleXY applies a non-uniform scale.
	ScaleXY(sx, sy float64)

	// --- Path construction ---

	// MoveTo starts a new sub-path at (x, y).
	MoveTo(x, y float64)
	// LineTo adds a line segment from the current point to (x, y).
	LineTo(x, y float64)
	// QuadraticTo adds a quadratic Bézier curve to (x, y) via control point (cx, cy).
	QuadraticTo(cx, cy, x, y float64)
	// CubicTo adds a cubic Bézier curve to (x, y) via control points (cx1, cy1) and (cx2, cy2).
	CubicTo(cx1, cy1, cx2, cy2, x, y float64)
	// ClosePath closes the current sub-path with a line to the start point.
	ClosePath()
	// ClearPath discards the current path without drawing.
	ClearPath()

	// --- Drawing ---

	// SetColor sets the current drawing color for both fill and stroke.
	SetColor(c color.Color)
	// SetRGBA sets the current drawing color from normalized components.
	SetRGBA(r, g, b, a float64)
	// SetLineWidth sets the stroke width in pixels.
	SetLineWidth(w float64)
	// SetLineDash sets the dash pattern. Pass nil or empty for solid lines.
	SetLineDash(pattern ...float64)
	// Fill fills the current path with the current color, then clears the path.
	Fill()
	// Stroke draws the current path outline, then clears the path.
	Stroke()
	// FillPreserve fills without clearing the path (allows subsequent stroke).
	FillPreserve()
	// Clip clips rendering to the current path. Call after constructing a path
	// (e.g. DrawRectangle). Use Save/Restore to undo the clip.
	Clip()

	// --- Convenience primitives ---

	// DrawCircle adds a circle path centered at (cx, cy) with radius r.
	DrawCircle(cx, cy, r float64)
	// DrawRectangle adds a rectangle path at (x, y) with dimensions (w, h).
	DrawRectangle(x, y, w, h float64)
	// DrawLine adds a line path from (x1, y1) to (x2, y2).
	DrawLine(x1, y1, x2, y2 float64)

	// --- Text ---

	// SetFontSize sets the font size in points.
	SetFontSize(size float64)
	// SetTabularNums enables or disables tabular (monospaced) digit widths
	// for subsequent text rendering. When enabled, digits occupy uniform
	// horizontal space for aligned numeric columns (e.g., axis tick labels).
	// Backends that don't support OpenType features may ignore this.
	SetTabularNums(enabled bool)
	// DrawStringAnchored draws text at (x, y) with anchor (ax, ay) ∈ [0,1].
	// (0,0) = top-left, (0.5,0.5) = center, (1,1) = bottom-right.
	DrawStringAnchored(text string, x, y, ax, ay float64)
	// MeasureString returns the width and height of the rendered text.
	MeasureString(text string) (w, h float64)

	// --- Output ---

	// Clear fills the entire canvas with the given color.
	Clear(c color.Color)
	// Width returns the canvas width in pixels.
	Width() int
	// Height returns the canvas height in pixels.
	Height() int
	// DrawImage composites the given image onto the canvas at position (x, y).
	// Used for compositing panel sub-canvases during parallel rendering.
	DrawImage(img image.Image, x, y float64)

	// Close releases backend resources (GPU buffers, etc.).
	// Implementations that hold no resources may return nil.
	Close() error
}

// TextAnchor defines text alignment as normalized coordinates.
type TextAnchor struct {
	X, Y float64 // [0,1] where (0,0) = top-left, (1,1) = bottom-right
}

// Common text anchors.
var (
	AnchorTopLeft      = TextAnchor{0, 0}
	AnchorTopCenter    = TextAnchor{0.5, 0}
	AnchorTopRight     = TextAnchor{1, 0}
	AnchorMiddleLeft   = TextAnchor{0, 0.5}
	AnchorCenter       = TextAnchor{0.5, 0.5}
	AnchorMiddleRight  = TextAnchor{1, 0.5}
	AnchorBottomLeft   = TextAnchor{0, 1}
	AnchorBottomCenter = TextAnchor{0.5, 1}
	AnchorBottomRight  = TextAnchor{1, 1}
)

// FontWeight describes font weight for text rendering.
type FontWeight int

// FontWeight constants for common weights.
const (
	WeightNormal FontWeight = 400
	WeightBold   FontWeight = 700
)
