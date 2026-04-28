package canvas

import (
	"image/color"
	"io"

	"github.com/TuSKan/ggplot/internal/fonts"
	"github.com/gogpu/gg/recording"
)

// RecordingCanvas wraps a recording.Recorder to implement the [Canvas] interface.
// All drawing operations are captured as a recording that can later be replayed
// into any backend (SVG, PDF, raster, etc.) via [Recording.Playback].
type RecordingCanvas struct {
	rec      *recording.Recorder
	fontSize float64
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

// --- State ---
func (c *RecordingCanvas) Save()    { c.rec.Save() }
func (c *RecordingCanvas) Restore() { c.rec.Restore() }

// --- Transforms ---
func (c *RecordingCanvas) Translate(dx, dy float64) { c.rec.Translate(dx, dy) }
func (c *RecordingCanvas) Rotate(angle float64)     { c.rec.Rotate(angle) }
func (c *RecordingCanvas) ScaleXY(sx, sy float64)   { c.rec.Scale(sx, sy) }

// --- Path ---
func (c *RecordingCanvas) MoveTo(x, y float64)              { c.rec.MoveTo(x, y) }
func (c *RecordingCanvas) LineTo(x, y float64)              { c.rec.LineTo(x, y) }
func (c *RecordingCanvas) QuadraticTo(cx, cy, x, y float64) { c.rec.QuadraticTo(cx, cy, x, y) }
func (c *RecordingCanvas) CubicTo(cx1, cy1, cx2, cy2, x, y float64) {
	c.rec.CubicTo(cx1, cy1, cx2, cy2, x, y)
}
func (c *RecordingCanvas) ClosePath() { c.rec.ClosePath() }
func (c *RecordingCanvas) ClearPath() { c.rec.ClearPath() }

// --- Drawing ---
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
func (c *RecordingCanvas) SetRGBA(r, g, b, a float64)     { c.rec.SetRGBA(r, g, b, a) }
func (c *RecordingCanvas) SetLineWidth(w float64)         { c.rec.SetLineWidth(w) }
func (c *RecordingCanvas) SetLineDash(pattern ...float64) { c.rec.SetDash(pattern...) }
func (c *RecordingCanvas) Fill()                          { c.rec.Fill() }
func (c *RecordingCanvas) Stroke()                        { c.rec.Stroke() }
func (c *RecordingCanvas) FillPreserve()                  { c.rec.FillPreserve() }
func (c *RecordingCanvas) Clip()                          { c.rec.Clip() }

// --- Primitives ---
func (c *RecordingCanvas) DrawCircle(cx, cy, r float64)     { c.rec.DrawCircle(cx, cy, r) }
func (c *RecordingCanvas) DrawRectangle(x, y, w, h float64) { c.rec.DrawRectangle(x, y, w, h) }
func (c *RecordingCanvas) DrawLine(x1, y1, x2, y2 float64)  { c.rec.DrawLine(x1, y1, x2, y2) }

// --- Text ---

// SetFontSize resolves the best available font at the given size.
func (c *RecordingCanvas) SetFontSize(size float64) {
	if size <= 0 {
		size = 12
	}
	c.fontSize = size

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
			if face := handle.TextFace(); face != nil {
				c.rec.SetFont(face)
				return
			}
		}
	}

	// Fallback to embedded Go Regular.
	if embeddedSource != nil {
		c.rec.SetFont(embeddedSource.Face(size))
	}
}

func (c *RecordingCanvas) DrawStringAnchored(s string, x, y, ax, ay float64) {
	c.rec.DrawStringAnchored(s, x, y, ax, ay)
}

func (c *RecordingCanvas) MeasureString(s string) (float64, float64) {
	return c.rec.MeasureString(s)
}

// --- Output ---
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

func (c *RecordingCanvas) Width() int  { return c.rec.Width() }
func (c *RecordingCanvas) Height() int { return c.rec.Height() }

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
