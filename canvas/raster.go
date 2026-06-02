package canvas

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"sync"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/TuSKan/ggplot/fonts"
)

// fontState holds the shared, lazily-initialized font resolver.
var (
	fontOnce     sync.Once
	fontResolver *fonts.Resolver
	// Embedded fallback used when no system font is found.
	embeddedSource *text.FontSource
	// faceCache avoids re-resolving system fonts + constructing Face objects
	// on every SetFontSize call. Key: faceCacheKey, Value: text.Face.
	faceCache sync.Map
)

// faceCacheKey identifies a cached font face.
type faceCacheKey struct {
	size    float64
	tabNums bool
}

func initFonts() {
	fontOnce.Do(func() {
		// 1. Build the system font registry and resolver.
		registry, _ := fonts.NewRegistry()
		fontResolver = fonts.NewResolver(registry, fonts.DefaultFallbackConfig())

		// 2. Parse embedded Go Regular as guaranteed fallback.
		src, err := text.NewFontSource(goregular.TTF)
		if err == nil {
			embeddedSource = src
		}

		// 3. Enable HarfBuzz-level shaping for OpenType feature support (tnum, liga, etc.).
		text.SetShaper(text.NewGoTextShaper())
	})
}

// RasterCanvas wraps a gogpu/gg.Context to implement the [Canvas] interface.
// This is the primary CPU-based rendering backend.
type RasterCanvas struct {
	ctx      *gg.Context
	fontSize float64
	tabNums  bool // tabular figures enabled
}

// NewRasterCanvas creates a Canvas backed by a gogpu/gg rasterizer.
func NewRasterCanvas(width, height int) *RasterCanvas {
	initFonts()

	c := &RasterCanvas{ctx: gg.NewContext(width, height)}
	c.SetFontSize(12) // load default font

	return c
}

// NewRasterCanvasCPU creates a Canvas that uses pure-CPU analytic rasterization,
// bypassing the GPU accelerator. This produces deterministic output even
// when multiple canvases are created in a single process (useful for tests).
func NewRasterCanvasCPU(width, height int) *RasterCanvas {
	initFonts()

	c := &RasterCanvas{ctx: gg.NewContext(width, height)}
	c.ctx.SetRasterizerMode(gg.RasterizerAnalytic)
	c.SetFontSize(12)

	return c
}

// RasterFromContext wraps an existing gg.Context.
func RasterFromContext(ctx *gg.Context) *RasterCanvas {
	initFonts()

	c := &RasterCanvas{ctx: ctx}
	c.SetFontSize(12)

	return c
}

// Context returns the underlying gg.Context for direct access when needed
// (e.g., font loading, image export).
func (c *RasterCanvas) Context() *gg.Context { return c.ctx }

// Save pushes the current graphics state onto the stack.
func (c *RasterCanvas) Save() { c.ctx.Push() }

// Restore pops the graphics state from the stack.
func (c *RasterCanvas) Restore() { c.ctx.Pop() }

// Translate shifts the origin by (dx, dy).
func (c *RasterCanvas) Translate(dx, dy float64) { c.ctx.Translate(dx, dy) }

// Rotate applies a rotation of angle radians.
func (c *RasterCanvas) Rotate(angle float64) { c.ctx.Rotate(angle) }

// ScaleXY applies a non-uniform scale.
func (c *RasterCanvas) ScaleXY(sx, sy float64) { c.ctx.Scale(sx, sy) }

// MoveTo starts a new sub-path at (x, y).
func (c *RasterCanvas) MoveTo(x, y float64) { c.ctx.MoveTo(x, y) }

// LineTo adds a line segment from the current point to (x, y).
func (c *RasterCanvas) LineTo(x, y float64) { c.ctx.LineTo(x, y) }

// QuadraticTo adds a quadratic Bézier curve via control point (cx, cy) to (x, y).
func (c *RasterCanvas) QuadraticTo(cx, cy, x, y float64) { c.ctx.QuadraticTo(cx, cy, x, y) }

// CubicTo adds a cubic Bézier curve via (cx1, cy1) and (cx2, cy2) to (x, y).
func (c *RasterCanvas) CubicTo(cx1, cy1, cx2, cy2, x, y float64) {
	c.ctx.CubicTo(cx1, cy1, cx2, cy2, x, y)
}

// ClosePath closes the current sub-path.
func (c *RasterCanvas) ClosePath() { c.ctx.ClosePath() }

// ClearPath discards the current path without drawing.
func (c *RasterCanvas) ClearPath() { c.ctx.ClearPath() }

// SetColor sets the current drawing color for both fill and stroke.
func (c *RasterCanvas) SetColor(col color.Color) {
	if col == nil {
		col = color.Black
	}

	c.ctx.SetColor(col)
}

// SetRGBA sets the current drawing color from normalized components.
func (c *RasterCanvas) SetRGBA(r, g, b, a float64) {
	c.ctx.SetRGBA(r, g, b, a)
}

// SetLineWidth sets the stroke width in pixels.
func (c *RasterCanvas) SetLineWidth(w float64) { c.ctx.SetLineWidth(w) }

// SetLineDash sets the dash pattern. Pass nil or empty for solid lines.
func (c *RasterCanvas) SetLineDash(pattern ...float64) { c.ctx.SetDash(pattern...) }

// Fill fills the current path with the current color, then clears the path.
func (c *RasterCanvas) Fill() { _ = c.ctx.Fill() }

// Stroke draws the current path outline, then clears the path.
func (c *RasterCanvas) Stroke() { _ = c.ctx.Stroke() }

// FillPreserve fills without clearing the path (allows subsequent stroke).
func (c *RasterCanvas) FillPreserve() { _ = c.ctx.FillPreserve() }

// Clip clips rendering to the current path.
func (c *RasterCanvas) Clip() { c.ctx.Clip() }

// ClipRect clips rendering to the axis-aligned rectangle (x, y, w, h).
// Uses gg.Context.ClipRect which pushes a rect onto the clip stack
// without rasterizing a full-resolution anti-aliased clip mask.
// Profile showed the generic Clip() path consuming ~75% of frame time
// via MaskClipper.rasterizeScanlineAA — this bypasses that entirely.
func (c *RasterCanvas) ClipRect(x, y, w, h float64) {
	c.ctx.ClipRect(x, y, w, h)
}

// DrawCircle adds a circle path centered at (cx, cy) with radius r.
func (c *RasterCanvas) DrawCircle(cx, cy, r float64) { c.ctx.DrawCircle(cx, cy, r) }

// DrawRectangle adds a rectangle path at (x, y) with dimensions (w, h).
func (c *RasterCanvas) DrawRectangle(x, y, w, h float64) { c.ctx.DrawRectangle(x, y, w, h) }

// DrawLine adds a line path from (x1, y1) to (x2, y2).
func (c *RasterCanvas) DrawLine(x1, y1, x2, y2 float64) { c.ctx.DrawLine(x1, y1, x2, y2) }

// DrawShape adds a path for the specified shape centered at (cx, cy) with size/radius r.
// Uses gg.Context.DrawRegularPolygon for polygon shapes for optimal rendering.
func (c *RasterCanvas) DrawShape(shape string, cx, cy, r float64) {
	switch shape {
	case ShapeCircle:
		c.ctx.DrawCircle(cx, cy, r)
	case ShapeSquare:
		c.ctx.DrawRegularPolygon(4, cx, cy, r, 0) //nolint:mnd // 4-sided polygon
	case ShapeTriangle:
		c.ctx.DrawRegularPolygon(3, cx, cy, r, 0) //nolint:mnd // 3-sided polygon
	case ShapeTriangleDown:
		c.ctx.DrawRegularPolygon(3, cx, cy, r, math.Pi) //nolint:mnd // 3-sided polygon rotated π
	case ShapeDiamond:
		c.ctx.DrawRegularPolygon(4, cx, cy, r, math.Pi/4) //nolint:mnd // 4-sided polygon rotated 45°
	case ShapePentagon:
		c.ctx.DrawRegularPolygon(5, cx, cy, r, 0) //nolint:mnd // 5-sided polygon
	case ShapeHexagon:
		c.ctx.DrawRegularPolygon(6, cx, cy, r, 0) //nolint:mnd // 6-sided polygon
	case ShapePlus:
		c.ctx.DrawLine(cx, cy-r, cx, cy+r)
		c.ctx.DrawLine(cx-r, cy, cx+r, cy)
	case ShapeCross:
		c.ctx.DrawLine(cx-r, cy-r, cx+r, cy+r)
		c.ctx.DrawLine(cx-r, cy+r, cx+r, cy-r)
	case ShapeStar:
		// Star is not a regular polygon — use path-based fallback.
		DrawShapePath(c, shape, cx, cy, r)
	default:
		c.ctx.DrawCircle(cx, cy, r)
	}
}

// --- Text ---

// SetFontSize resolves the best available font at the given size.
// Resolution order: system font via fonts resolver -> embedded Go Regular fallback.
func (c *RasterCanvas) SetFontSize(size float64) {
	if size <= 0 {
		size = 12
	}

	c.fontSize = size

	if f := resolveFace(size, c.tabNums); f != nil {
		c.ctx.SetFont(f)
	}
}

// resolveFace returns a cached text.Face for the given size and feature set,
// creating and caching one on the first call for each unique (size, tabNums).
func resolveFace(size float64, tabNums bool) text.Face {
	key := faceCacheKey{size: size, tabNums: tabNums}

	if cached, ok := faceCache.Load(key); ok {
		if f, ok := cached.(text.Face); ok {
			return f
		}
	}

	var opts []text.FaceOption
	if tabNums {
		opts = append(opts, text.WithFeatures(text.TabularNums))
	}

	var face text.Face

	if fontResolver != nil {
		handle, err := fontResolver.LoadFace(fonts.FaceRequest{
			Family:        "sans-serif",
			Weight:        fonts.WeightNormal,
			AllowFallback: true,
			Size:          size,
			DPI:           72,
		})
		if err == nil && handle != nil {
			if src := handle.FontSource(); src != nil {
				face = src.Face(size, opts...)
			}
		}
	}

	if face == nil && embeddedSource != nil {
		face = embeddedSource.Face(size, opts...)
	}

	if face != nil {
		faceCache.Store(key, face)
	}

	return face
}

// SetTabularNums enables or disables tabular (monospaced) digit widths.
func (c *RasterCanvas) SetTabularNums(enabled bool) {
	if c.tabNums == enabled {
		return
	}

	c.tabNums = enabled
	c.SetFontSize(c.fontSize) // re-resolve font with new features
}

// DrawStringAnchored draws text at (x, y) with anchor (ax, ay).
func (c *RasterCanvas) DrawStringAnchored(s string, x, y, ax, ay float64) {
	c.ctx.DrawStringAnchored(s, x, y, ax, ay)
}

// MeasureString returns the width and height of the rendered text.
func (c *RasterCanvas) MeasureString(s string) (float64, float64) {
	w, h := c.ctx.MeasureString(s)
	return w, h
}

// Clear fills the entire canvas with the given color.
// Uses DrawRectangle+Fill instead of ctx.Clear() because the GPU accelerator
// needs a draw command to update the surface — ctx.Clear() only writes to
// the CPU pixmap and leaves the GPU surface stale.
func (c *RasterCanvas) Clear(col color.Color) {
	r, g, b, a := col.RGBA()
	c.ctx.SetRGBA(
		float64(r)/65535.0,
		float64(g)/65535.0,
		float64(b)/65535.0,
		float64(a)/65535.0,
	)
	c.ctx.DrawRectangle(0, 0, float64(c.ctx.Width()), float64(c.ctx.Height()))
	_ = c.ctx.Fill()
}

// Width returns the canvas width in pixels.
func (c *RasterCanvas) Width() int { return c.ctx.Width() }

// Height returns the canvas height in pixels.
func (c *RasterCanvas) Height() int { return c.ctx.Height() }

// SavePNG writes the canvas to a PNG file.
func (c *RasterCanvas) SavePNG(path string) error {
	if err := c.ctx.SavePNG(path); err != nil {
		return fmt.Errorf("canvas: SavePNG: %w", err)
	}

	return nil
}

// EncodePNG writes the canvas as PNG to the given writer.
func (c *RasterCanvas) EncodePNG(w io.Writer) error {
	if err := c.ctx.EncodePNG(w); err != nil {
		return fmt.Errorf("canvas: EncodePNG: %w", err)
	}

	return nil
}

// DrawImage composites img onto the canvas at pixel position (x, y).
func (c *RasterCanvas) DrawImage(img image.Image, x, y float64) {
	buf := gg.ImageBufFromImage(img)
	c.ctx.DrawImage(buf, x, y)
}

// Image returns the underlying image.
func (c *RasterCanvas) Image() image.Image {
	return c.ctx.Image()
}

// Close releases the underlying gg.Context's GPU resources.
// Always call Close (or use defer) when the canvas is no longer needed
// to avoid GPU resource leaks (wgpu BindGroup/Buffer finalizer warnings).
func (c *RasterCanvas) Close() error {
	if c.ctx == nil {
		return nil
	}

	err := c.ctx.Close()
	c.ctx = nil

	return errors.Join(err)
}

// Compile-time check.
var _ Canvas = (*RasterCanvas)(nil)
