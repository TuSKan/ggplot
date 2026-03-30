package gpu

import (
	"image/color"

	"github.com/TuSKan/ggplot/internal/fonts"
	"github.com/TuSKan/ggplot/internal/render"
	"github.com/TuSKan/ggplot/pkg/theme"
	"github.com/gogpu/gg"
	_ "github.com/gogpu/gg/gpu" // Opt-in explicit hardware parallel acceleration!
)

// Backend mirrors the software pipeline but enforces internal graphic hardware capabilities appropriately.
type Backend struct {
	dc       *gg.Context
	resolver *fonts.Resolver
}

// New triggers explicit GPU layout acceleration!
func New(width, height int, res *fonts.Resolver) *Backend {
	return &Backend{
		dc:       gg.NewContext(width, height),
		resolver: res,
	}
}

// Verify at compile time that gpu.Backend implements render.Backend!
var _ render.Backend = (*Backend)(nil)

func (b *Backend) SetClipRect(r render.Rect) {
	b.dc.ClearPath()
	b.dc.DrawRectangle(r.Min.X, r.Min.Y, r.Max.X-r.Min.X, r.Max.Y-r.Min.Y)
	b.dc.Clip()
}

func (b *Backend) ClearClip() {
	b.dc.ResetClip()
}

func (b *Backend) DrawPoint(x, y, radius float64, s render.Style) {
	b.dc.DrawCircle(x, y, radius)
	if s.Fill != nil {
		b.dc.SetColor(s.Fill)
		b.dc.FillPreserve()
	}
	if s.StrokeWidth > 0 && s.Stroke != nil {
		b.dc.SetColor(s.Stroke)
		b.dc.SetLineWidth(s.StrokeWidth)
		b.dc.Stroke()
	} else {
		b.dc.ClearPath()
	}
}

func (b *Backend) DrawLine(x1, y1, x2, y2 float64, s render.Style) {
	if (s.Stroke == nil) || (s.StrokeWidth <= 0) {
		return
	}
	b.dc.SetColor(s.Stroke)
	b.dc.SetLineWidth(s.StrokeWidth)
	b.dc.DrawLine(x1, y1, x2, y2)
	b.dc.Stroke()
}

func (b *Backend) DrawPolygon(points []render.Point, s render.Style) {
	if len(points) < 3 {
		return
	}
	b.dc.MoveTo(points[0].X, points[0].Y)
	for i := 1; i < len(points); i++ {
		b.dc.LineTo(points[i].X, points[i].Y)
	}
	b.dc.ClosePath()

	if s.Fill != nil {
		b.dc.SetColor(s.Fill)
		b.dc.FillPreserve()
	}
	if s.StrokeWidth > 0 && s.Stroke != nil {
		b.dc.SetColor(s.Stroke)
		b.dc.SetLineWidth(s.StrokeWidth)
		b.dc.Stroke()
	} else {
		b.dc.ClearPath()
	}
}

func (b *Backend) DrawRect(r render.Rect, s render.Style) {
	b.dc.DrawRectangle(r.Min.X, r.Min.Y, r.Max.X-r.Min.X, r.Max.Y-r.Min.Y)
	if s.Fill != nil {
		b.dc.SetColor(s.Fill)
		b.dc.FillPreserve()
	}
	if s.StrokeWidth > 0 && s.Stroke != nil {
		b.dc.SetColor(s.Stroke)
		b.dc.SetLineWidth(s.StrokeWidth)
		b.dc.Stroke()
	} else {
		b.dc.ClearPath()
	}
}

func (b *Backend) DrawText(text string, x, y float64, ax, ay float64, s render.Style, f theme.FontConfig) {
	if s.Fill != nil {
		b.dc.SetColor(s.Fill)
	} else {
		b.dc.SetColor(color.Black)
	}

	if b.resolver != nil {
		handle, err := b.resolver.LoadFace(f.ToFaceRequest(96.0))
		if err == nil && handle != nil {
			b.dc.SetFont(handle.TextFace())
		}
	}

	b.dc.DrawStringAnchored(text, x, y, ax, ay)
}

func (b *Backend) Save(path string) error {
	return b.dc.SavePNG(path)
}
