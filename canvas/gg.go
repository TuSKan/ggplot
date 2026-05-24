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
)

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

// GGCanvas wraps a gogpu/gg.Context to implement the [Canvas] interface.
// This is the primary CPU-based rendering backend.
type GGCanvas struct {
	ctx      *gg.Context
	fontSize float64
	tabNums  bool // tabular figures enabled
}

// NewGGCanvas creates a Canvas backed by a gogpu/gg rasterizer.
func NewGGCanvas(width, height int) *GGCanvas {
	initFonts()

	c := &GGCanvas{ctx: gg.NewContext(width, height)}
	c.SetFontSize(12) // load default font

	return c
}

// NewGGCanvasCPU creates a Canvas that uses pure-CPU analytic rasterization,
// bypassing the GPU accelerator. This produces deterministic output even
// when multiple canvases are created in a single process (useful for tests).
func NewGGCanvasCPU(width, height int) *GGCanvas {
	initFonts()

	c := &GGCanvas{ctx: gg.NewContext(width, height)}
	c.ctx.SetRasterizerMode(gg.RasterizerAnalytic)
	c.SetFontSize(12)

	return c
}

// FromGGContext wraps an existing gg.Context.
func FromGGContext(ctx *gg.Context) *GGCanvas {
	initFonts()

	c := &GGCanvas{ctx: ctx}
	c.SetFontSize(12)

	return c
}

// Context returns the underlying gg.Context for direct access when needed
// (e.g., font loading, image export).
func (c *GGCanvas) Context() *gg.Context { return c.ctx }

// Save pushes the current graphics state onto the stack.
func (c *GGCanvas) Save() { c.ctx.Push() }

// Restore pops the graphics state from the stack.
func (c *GGCanvas) Restore() { c.ctx.Pop() }

// Translate shifts the origin by (dx, dy).
func (c *GGCanvas) Translate(dx, dy float64) { c.ctx.Translate(dx, dy) }

// Rotate applies a rotation of angle radians.
func (c *GGCanvas) Rotate(angle float64) { c.ctx.Rotate(angle) }

// ScaleXY applies a non-uniform scale.
func (c *GGCanvas) ScaleXY(sx, sy float64) { c.ctx.Scale(sx, sy) }

// MoveTo starts a new sub-path at (x, y).
func (c *GGCanvas) MoveTo(x, y float64) { c.ctx.MoveTo(x, y) }

// LineTo adds a line segment from the current point to (x, y).
func (c *GGCanvas) LineTo(x, y float64) { c.ctx.LineTo(x, y) }

// QuadraticTo adds a quadratic Bézier curve via control point (cx, cy) to (x, y).
func (c *GGCanvas) QuadraticTo(cx, cy, x, y float64) { c.ctx.QuadraticTo(cx, cy, x, y) }

// CubicTo adds a cubic Bézier curve via (cx1, cy1) and (cx2, cy2) to (x, y).
func (c *GGCanvas) CubicTo(cx1, cy1, cx2, cy2, x, y float64) { c.ctx.CubicTo(cx1, cy1, cx2, cy2, x, y) }

// ClosePath closes the current sub-path.
func (c *GGCanvas) ClosePath() { c.ctx.ClosePath() }

// ClearPath discards the current path without drawing.
func (c *GGCanvas) ClearPath() { c.ctx.ClearPath() }

// SetColor sets the current drawing color for both fill and stroke.
func (c *GGCanvas) SetColor(col color.Color) {
	if col == nil {
		col = color.Black
	}

	c.ctx.SetColor(col)
}

// SetRGBA sets the current drawing color from normalized components.
func (c *GGCanvas) SetRGBA(r, g, b, a float64) {
	c.ctx.SetRGBA(r, g, b, a)
}

// SetLineWidth sets the stroke width in pixels.
func (c *GGCanvas) SetLineWidth(w float64) { c.ctx.SetLineWidth(w) }

// SetLineDash sets the dash pattern. Pass nil or empty for solid lines.
func (c *GGCanvas) SetLineDash(pattern ...float64) { c.ctx.SetDash(pattern...) }

// Fill fills the current path with the current color, then clears the path.
func (c *GGCanvas) Fill() { _ = c.ctx.Fill() }

// Stroke draws the current path outline, then clears the path.
func (c *GGCanvas) Stroke() { _ = c.ctx.Stroke() }

// FillPreserve fills without clearing the path (allows subsequent stroke).
func (c *GGCanvas) FillPreserve() { _ = c.ctx.FillPreserve() }

// Clip clips rendering to the current path.
func (c *GGCanvas) Clip() { c.ctx.Clip() }

// DrawCircle adds a circle path centered at (cx, cy) with radius r.
func (c *GGCanvas) DrawCircle(cx, cy, r float64) { c.ctx.DrawCircle(cx, cy, r) }

// DrawRectangle adds a rectangle path at (x, y) with dimensions (w, h).
func (c *GGCanvas) DrawRectangle(x, y, w, h float64) { c.ctx.DrawRectangle(x, y, w, h) }

// DrawLine adds a line path from (x1, y1) to (x2, y2).
func (c *GGCanvas) DrawLine(x1, y1, x2, y2 float64) { c.ctx.DrawLine(x1, y1, x2, y2) }

// DrawShape adds a path for the specified shape centered at (cx, cy) with size/radius r.
// Uses gg.Context.DrawRegularPolygon for polygon shapes for optimal rendering.
func (c *GGCanvas) DrawShape(shape string, cx, cy, r float64) {
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
func (c *GGCanvas) SetFontSize(size float64) {
	if size <= 0 {
		size = 12
	}

	c.fontSize = size

	// Build face options (tabular nums if active).
	var opts []text.FaceOption
	if c.tabNums {
		opts = append(opts, text.WithFeatures(text.TabularNums))
	}

	// Try system font resolver first (sans-serif family).
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
				c.ctx.SetFont(src.Face(size, opts...))

				return
			}
		}
	}

	// Fallback to embedded Go Regular.
	if embeddedSource != nil {
		c.ctx.SetFont(embeddedSource.Face(size, opts...))
	}
}

// SetTabularNums enables or disables tabular (monospaced) digit widths.
func (c *GGCanvas) SetTabularNums(enabled bool) {
	if c.tabNums == enabled {
		return
	}

	c.tabNums = enabled
	c.SetFontSize(c.fontSize) // re-resolve font with new features
}

// DrawStringAnchored draws text at (x, y) with anchor (ax, ay).
func (c *GGCanvas) DrawStringAnchored(s string, x, y, ax, ay float64) {
	c.ctx.DrawStringAnchored(s, x, y, ax, ay)
}

// MeasureString returns the width and height of the rendered text.
func (c *GGCanvas) MeasureString(s string) (float64, float64) {
	w, h := c.ctx.MeasureString(s)
	return w, h
}

// Clear fills the entire canvas with the given color.
func (c *GGCanvas) Clear(col color.Color) {
	// Use explicit RGBA to avoid gg's color model conversion issues.
	r, g, b, a := col.RGBA()
	c.ctx.SetRGBA(
		float64(r)/65535.0,
		float64(g)/65535.0,
		float64(b)/65535.0,
		float64(a)/65535.0,
	)
	c.ctx.Clear()
	// Also draw an opaque rectangle as a safety net (some gg versions
	// don't fully clear with alpha-blended colors).
	c.ctx.DrawRectangle(0, 0, float64(c.ctx.Width()), float64(c.ctx.Height()))
	_ = c.ctx.Fill()
}

// Width returns the canvas width in pixels.
func (c *GGCanvas) Width() int { return c.ctx.Width() }

// Height returns the canvas height in pixels.
func (c *GGCanvas) Height() int { return c.ctx.Height() }

// SavePNG writes the canvas to a PNG file.
func (c *GGCanvas) SavePNG(path string) error {
	if err := c.ctx.SavePNG(path); err != nil {
		return fmt.Errorf("canvas: SavePNG: %w", err)
	}

	return nil
}

// EncodePNG writes the canvas as PNG to the given writer.
func (c *GGCanvas) EncodePNG(w io.Writer) error {
	if err := c.ctx.EncodePNG(w); err != nil {
		return fmt.Errorf("canvas: EncodePNG: %w", err)
	}

	return nil
}

// DrawImage composites img onto the canvas at pixel position (x, y).
func (c *GGCanvas) DrawImage(img image.Image, x, y float64) {
	buf := gg.ImageBufFromImage(img)
	c.ctx.DrawImage(buf, x, y)
}

// Image returns the underlying image.
func (c *GGCanvas) Image() image.Image {
	return c.ctx.Image()
}

// Close releases the underlying gg.Context's GPU resources.
// Always call Close (or use defer) when the canvas is no longer needed
// to avoid GPU resource leaks (wgpu BindGroup/Buffer finalizer warnings).
func (c *GGCanvas) Close() error {
	if c.ctx == nil {
		return nil
	}

	err := c.ctx.Close()
	c.ctx = nil

	return errors.Join(err)
}

// Compile-time check.
var _ Canvas = (*GGCanvas)(nil)
