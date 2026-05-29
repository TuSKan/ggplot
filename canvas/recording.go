package canvas

import (
	"image"
	"image/color"
	"io"

	"github.com/gogpu/gg/recording"
	"github.com/gogpu/gg/text"

	"github.com/TuSKan/ggplot/fonts"
)

// RecordingCanvas wraps a recording.Recorder to implement the [Canvas] interface.
// All drawing operations are captured as a recording that can later be replayed
// into any backend (SVG, PDF, raster, etc.) via [Recording.Playback].
type RecordingCanvas struct {
	rec      *recording.Recorder
	face     text.Face // current font face, for anchor measurement
	fontSize float64
	tabNums  bool // tabular figures enabled
}

// NewRecordingCanvas creates a Canvas that records all drawing operations.
// After rendering, call [RecordingCanvas.FinishRecording] and replay into
// the desired backend (e.g., SVG, PDF).
func NewRecordingCanvas(width, height int) *RecordingCanvas {
	initFonts()

	c := &RecordingCanvas{rec: recording.NewRecorder(width, height)}
	c.SetFontSize(12)

	return c
}

// FinishRecording completes the recording and returns the captured operations.
func (c *RecordingCanvas) FinishRecording() *recording.Recording {
	return c.rec.FinishRecording()
}

// Save pushes the current graphics state onto the stack.
func (c *RecordingCanvas) Save() { c.rec.Save() }

// Restore pops the graphics state from the stack.
func (c *RecordingCanvas) Restore() { c.rec.Restore() }

// Translate shifts the origin by (dx, dy).
func (c *RecordingCanvas) Translate(dx, dy float64) { c.rec.Translate(dx, dy) }

// Rotate applies a rotation of angle radians.
func (c *RecordingCanvas) Rotate(angle float64) { c.rec.Rotate(angle) }

// ScaleXY applies a non-uniform scale.
func (c *RecordingCanvas) ScaleXY(sx, sy float64) { c.rec.Scale(sx, sy) }

// MoveTo starts a new sub-path at (x, y).
func (c *RecordingCanvas) MoveTo(x, y float64) { c.rec.MoveTo(x, y) }

// LineTo adds a line segment from the current point to (x, y).
func (c *RecordingCanvas) LineTo(x, y float64) { c.rec.LineTo(x, y) }

// QuadraticTo adds a quadratic Bézier curve via control point (cx, cy) to (x, y).
func (c *RecordingCanvas) QuadraticTo(cx, cy, x, y float64) { c.rec.QuadraticTo(cx, cy, x, y) }

// CubicTo adds a cubic Bézier curve via (cx1, cy1) and (cx2, cy2) to (x, y).
func (c *RecordingCanvas) CubicTo(cx1, cy1, cx2, cy2, x, y float64) {
	c.rec.CubicTo(cx1, cy1, cx2, cy2, x, y)
}

// ClosePath closes the current sub-path.
func (c *RecordingCanvas) ClosePath() { c.rec.ClosePath() }

// ClearPath discards the current path without drawing.
func (c *RecordingCanvas) ClearPath() { c.rec.ClearPath() }

// SetColor sets the current drawing color for both fill and stroke.
func (c *RecordingCanvas) SetColor(col color.Color) {
	if col == nil {
		col = color.Black
	}

	r, g, b, a := col.RGBA()
	c.rec.SetRGBA(
		float64(r)/65535.0,
		float64(g)/65535.0,
		float64(b)/65535.0,
		float64(a)/65535.0,
	)
}

// SetRGBA sets the current drawing color from normalized components.
func (c *RecordingCanvas) SetRGBA(r, g, b, a float64) { c.rec.SetRGBA(r, g, b, a) }

// SetLineWidth sets the stroke width in pixels.
func (c *RecordingCanvas) SetLineWidth(w float64) { c.rec.SetLineWidth(w) }

// SetLineDash sets the dash pattern. Pass nil or empty for solid lines.
func (c *RecordingCanvas) SetLineDash(pattern ...float64) { c.rec.SetDash(pattern...) }

// Fill fills the current path with the current color, then clears the path.
func (c *RecordingCanvas) Fill() { c.rec.Fill() }

// Stroke draws the current path outline, then clears the path.
func (c *RecordingCanvas) Stroke() { c.rec.Stroke() }

// FillPreserve fills without clearing the path (allows subsequent stroke).
func (c *RecordingCanvas) FillPreserve() { c.rec.FillPreserve() }

// Clip clips rendering to the current path.
func (c *RecordingCanvas) Clip() { c.rec.Clip() }

// DrawCircle adds a circle path centered at (cx, cy) with radius r.
func (c *RecordingCanvas) DrawCircle(cx, cy, r float64) { c.rec.DrawCircle(cx, cy, r) }

// DrawRectangle adds a rectangle path at (x, y) with dimensions (w, h).
func (c *RecordingCanvas) DrawRectangle(x, y, w, h float64) { c.rec.DrawRectangle(x, y, w, h) }

// DrawLine adds a line path from (x1, y1) to (x2, y2).
func (c *RecordingCanvas) DrawLine(x1, y1, x2, y2 float64) { c.rec.DrawLine(x1, y1, x2, y2) }

// DrawShape adds a path for the specified shape centered at (cx, cy) with size/radius r.
func (c *RecordingCanvas) DrawShape(shape string, cx, cy, r float64) {
	DrawShapePath(c, shape, cx, cy, r)
}

// --- Text ---

// SetFontSize resolves the best available font at the given size.
func (c *RecordingCanvas) SetFontSize(size float64) {
	if size <= 0 {
		size = 12
	}

	c.fontSize = size

	// Build face options (tabular nums if active).
	var opts []text.FaceOption
	if c.tabNums {
		opts = append(opts, text.WithFeatures(text.TabularNums))
	}

	// Try system font resolver first.
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
				face := src.Face(size, opts...)
				c.face = face
				c.rec.SetFont(face)

				return
			}
		}
	}

	// Fallback to embedded Go Regular.
	if embeddedSource != nil {
		face := embeddedSource.Face(size, opts...)
		c.face = face
		c.rec.SetFont(face)
	}
}

// SetTabularNums enables or disables tabular (monospaced) digit widths.
func (c *RecordingCanvas) SetTabularNums(enabled bool) {
	if c.tabNums == enabled {
		return
	}

	c.tabNums = enabled
	c.SetFontSize(c.fontSize) // re-resolve font with new features
}

// DrawStringAnchored draws text at (x, y) with anchor (ax, ay).
//
// The anchor is pre-applied here rather than deferred to the recorder: the
// recording playback drops the anchor (its Backend.DrawText takes no anchor and
// is handed a nil face), so anchored text would otherwise render left-aligned
// at the baseline in SVG/PDF. We replicate gg's exact anchor math against the
// current face and record a base-positioned DrawString, keeping vector text
// aligned with the raster output.
func (c *RecordingCanvas) DrawStringAnchored(s string, x, y, ax, ay float64) {
	if c.face == nil {
		c.rec.DrawString(s, x, y)

		return
	}

	w, _ := text.Measure(s, c.face)
	m := c.face.Metrics()
	h := m.Ascent + m.Descent
	// baseline-left position: x -= w*ax ; baseline = y + ascent - ay*h
	c.rec.DrawString(s, x-w*ax, y+m.Ascent-ay*h)
}

// MeasureString returns the width and height of the rendered text.
func (c *RecordingCanvas) MeasureString(s string) (float64, float64) {
	return c.rec.MeasureString(s)
}

// Clear fills the entire canvas with the given color.
func (c *RecordingCanvas) Clear(col color.Color) {
	r, g, b, a := col.RGBA()
	c.rec.SetRGBA(
		float64(r)/65535.0,
		float64(g)/65535.0,
		float64(b)/65535.0,
		float64(a)/65535.0,
	)
	c.rec.Clear()
	c.rec.DrawRectangle(0, 0, float64(c.rec.Width()), float64(c.rec.Height()))
	c.rec.Fill()
}

// Width returns the canvas width in pixels.
func (c *RecordingCanvas) Width() int { return c.rec.Width() }

// Height returns the canvas height in pixels.
func (c *RecordingCanvas) Height() int { return c.rec.Height() }

// DrawImage is a no-op for RecordingCanvas. SVG/PDF recording does not
// support raster compositing; parallel panel rendering is only used by
// raster backends (RasterCanvas).
func (c *RecordingCanvas) DrawImage(_ image.Image, _, _ float64) {}

// Close is a no-op for RecordingCanvas (no GPU resources to release).
func (c *RecordingCanvas) Close() error { return nil }

// Compile-time check.
var _ Canvas = (*RecordingCanvas)(nil)

// --- Export helpers ---

// ExportSVG replays a recording into the native SVG backend and writes to w.
func ExportSVG(r *recording.Recording, w io.Writer) (int64, error) {
	return exportSVG(r, w)
}

// ExportPDF replays a recording into the native PDF backend and writes to w.
func ExportPDF(r *recording.Recording, w io.Writer) (int64, error) {
	return exportPDF(r, w)
}
