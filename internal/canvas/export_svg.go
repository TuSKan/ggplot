// Package canvas — SVG export backend.
//
// Implements recording.Backend against gg v0.43.2's current Path API (Iterate).
// The official gogpu/gg-svg v0.1.0 is incompatible with gg >= v0.40 due to
// path.Elements removal. This backend replaces it until upstream is fixed.
package canvas

import (
	"fmt"
	"image"
	"io"
	"math"
	"strings"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/recording"
	"github.com/gogpu/gg/text"
)

// svgBackend implements [recording.Backend] for SVG 1.1 output.
type svgBackend struct {
	w, h int
	buf  strings.Builder
}

func newSVGBackend() *svgBackend {
	return &svgBackend{}
}

func (b *svgBackend) Begin(width, height int) error {
	b.w, b.h = width, height
	b.buf.WriteString(fmt.Sprintf(
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"+
			"<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\">\n",
		width, height, width, height))
	return nil
}

func (b *svgBackend) End() error {
	b.buf.WriteString("</svg>\n")
	return nil
}

func (b *svgBackend) Save()    { b.buf.WriteString("<g>\n") }
func (b *svgBackend) Restore() { b.buf.WriteString("</g>\n") }

func (b *svgBackend) SetTransform(m recording.Matrix) {
	b.buf.WriteString(fmt.Sprintf(
		"<g transform=\"matrix(%.4f,%.4f,%.4f,%.4f,%.4f,%.4f)\">\n",
		m.A, m.D, m.B, m.E, m.C, m.F))
}

func (b *svgBackend) SetClip(path *gg.Path, rule recording.FillRule) {}
func (b *svgBackend) ClearClip()                                     {}

func (b *svgBackend) FillPath(path *gg.Path, brush recording.Brush, rule recording.FillRule) {
	d := pathToSVGData(path)
	if d == "" {
		return
	}
	fill, opacity := brushToSVG(brush)
	ruleAttr := ""
	if rule == recording.FillRuleEvenOdd {
		ruleAttr = ` fill-rule="evenodd"`
	}
	opacityAttr := ""
	if opacity < 1 {
		opacityAttr = fmt.Sprintf(` fill-opacity="%.3f"`, opacity)
	}
	fmt.Fprintf(&b.buf, "<path d=\"%s\" fill=\"%s\"%s%s stroke=\"none\"/>\n",
		d, fill, opacityAttr, ruleAttr)
}

func (b *svgBackend) StrokePath(path *gg.Path, brush recording.Brush, stroke recording.Stroke) {
	d := pathToSVGData(path)
	if d == "" {
		return
	}
	col, opacity := brushToSVG(brush)
	lw := stroke.Width
	if lw <= 0 {
		lw = 1
	}
	extra := ""
	if opacity < 1 {
		extra += fmt.Sprintf(` stroke-opacity="%.3f"`, opacity)
	}
	if len(stroke.DashPattern) > 0 {
		parts := make([]string, len(stroke.DashPattern))
		for i, v := range stroke.DashPattern {
			parts[i] = fmt.Sprintf("%.2f", v)
		}
		extra += fmt.Sprintf(` stroke-dasharray="%s"`, strings.Join(parts, ","))
	}
	fmt.Fprintf(&b.buf, "<path d=\"%s\" fill=\"none\" stroke=\"%s\" stroke-width=\"%.2f\"%s/>\n",
		d, col, lw, extra)
}

func (b *svgBackend) FillRect(rect recording.Rect, brush recording.Brush) {
	fill, opacity := brushToSVG(brush)
	opacityAttr := ""
	if opacity < 1 {
		opacityAttr = fmt.Sprintf(` fill-opacity="%.3f"`, opacity)
	}
	fmt.Fprintf(&b.buf, "<rect x=\"%.2f\" y=\"%.2f\" width=\"%.2f\" height=\"%.2f\" fill=\"%s\"%s/>\n",
		rect.X(), rect.Y(), rect.Width(), rect.Height(), fill, opacityAttr)
}

func (b *svgBackend) DrawImage(img image.Image, src, dst recording.Rect, opts recording.ImageOptions) {
	// Image embedding requires base64 inline data — deferred.
}

func (b *svgBackend) DrawText(s string, x, y float64, face text.Face, brush recording.Brush) {
	fill, _ := brushToSVG(brush)
	escaped := svgEscape(s)
	fmt.Fprintf(&b.buf,
		"<text x=\"%.2f\" y=\"%.2f\" fill=\"%s\" font-family=\"sans-serif\" font-size=\"12\">%s</text>\n",
		x, y, fill, escaped)
}

// WriteTo writes the SVG document to w.
func (b *svgBackend) WriteTo(w io.Writer) (int64, error) {
	n, err := io.WriteString(w, b.buf.String())
	return int64(n), err
}

// --- helpers ---

func pathToSVGData(path *gg.Path) string {
	if path == nil {
		return ""
	}
	var sb strings.Builder
	path.Iterate(func(verb gg.PathVerb, coords []float64) {
		switch verb {
		case gg.MoveTo:
			fmt.Fprintf(&sb, "M%.2f %.2f ", coords[0], coords[1])
		case gg.LineTo:
			fmt.Fprintf(&sb, "L%.2f %.2f ", coords[0], coords[1])
		case gg.QuadTo:
			fmt.Fprintf(&sb, "Q%.2f %.2f %.2f %.2f ", coords[0], coords[1], coords[2], coords[3])
		case gg.CubicTo:
			fmt.Fprintf(&sb, "C%.2f %.2f %.2f %.2f %.2f %.2f ",
				coords[0], coords[1], coords[2], coords[3], coords[4], coords[5])
		case gg.Close:
			sb.WriteString("Z ")
		}
	})
	return strings.TrimSpace(sb.String())
}

func brushToSVG(brush recording.Brush) (color string, opacity float64) {
	switch b := brush.(type) {
	case recording.SolidBrush:
		r := clampByte(b.Color.R)
		g := clampByte(b.Color.G)
		bl := clampByte(b.Color.B)
		return fmt.Sprintf("#%02X%02X%02X", r, g, bl), math.Max(0, math.Min(1, b.Color.A))
	default:
		return "#000000", 1
	}
}

func clampByte(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}

func svgEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func exportSVG(r *recording.Recording, w io.Writer) (int64, error) {
	b := newSVGBackend()
	if err := r.Playback(b); err != nil {
		return 0, err
	}
	return b.WriteTo(w)
}
