package plot

import (
	"image/color"
	"log"

	"github.com/TuSKan/ggplot/internal/fonts"
	"github.com/TuSKan/ggplot/pkg/theme"
	"github.com/gogpu/gg"
)

var globalResolver *fonts.Resolver

func init() {
	reg, err := fonts.NewRegistry()
	if err != nil {
		log.Printf("ggplot initialization warning: failed to initialize fonts registry: %v", err)
	}
	if reg != nil {
		globalResolver = fonts.NewResolver(reg, fonts.DefaultFallbackConfig())
	}
}

// ggMeasurer adapts gg.Context to guide.TextMeasurer natively.
type ggMeasurer struct {
	dc *gg.Context
}

func (m *ggMeasurer) MeasureText(text string, f theme.FontConfig) (float64, float64) {
	if globalResolver != nil {
		req := f.ToFaceRequest(96.0)
		if handle, err := globalResolver.LoadFace(req); err == nil && handle != nil {
			m.dc.SetFont(handle.TextFace())
		}
	}

	w, h := m.dc.MeasureString(text)
	if w == 0 && text != "" {
		w = float64(len(text)) * (f.Size * 0.6)
	}
	if h == 0 {
		h = f.Size
	}
	return w, h
}

// ggPainter adapts gg.Context to guide.Painter natively.
type ggPainter struct {
	dc *gg.Context
}

func (p *ggPainter) DrawText(text string, x, y float64, ax, ay string, f theme.FontConfig, c color.Color, rot float64) {
	if globalResolver != nil {
		req := f.ToFaceRequest(96.0)
		if handle, err := globalResolver.LoadFace(req); err == nil && handle != nil {
			p.dc.SetFont(handle.TextFace())
		}
	}

	p.dc.SetColor(c)

	alignX := 0.5
	alignY := 0.5

	switch ax {
	case "left":
		alignX = 0.0
	case "right":
		alignX = 1.0
	}
	switch ay {
	case "top":
		alignY = 1.0
	case "bottom":
		alignY = 0.0
	}

	p.dc.Push()
	p.dc.Translate(x, y)
	if rot != 0 {
		p.dc.Rotate(rot)
	}
	p.dc.DrawStringAnchored(text, 0, 0, alignX, alignY)
	p.dc.Pop()
}

func (p *ggPainter) DrawLine(x1, y1, x2, y2 float64, c color.Color, w float64, d []float64) {
	p.dc.SetColor(c)
	p.dc.SetLineWidth(w)
	if len(d) > 0 {
		p.dc.SetDash(d...)
	} else {
		p.dc.SetDash()
	}
	p.dc.DrawLine(x1, y1, x2, y2)
	p.dc.Stroke()
}

func (p *ggPainter) DrawRect(x, y, w, h float64, fill, stroke color.Color, sw float64) {
	if fill != color.Transparent {
		p.dc.SetColor(fill)
		p.dc.DrawRectangle(x, y, w, h)
		p.dc.Fill()
	}
	if sw > 0 {
		p.dc.SetColor(stroke)
		p.dc.SetLineWidth(sw)
		p.dc.DrawRectangle(x, y, w, h)
		p.dc.Stroke()
	}
}
