// Package canvas — PDF export backend.
//
// Implements recording.Backend against gg v0.43.2's current Path API (Iterate).
// The official gogpu/gg-pdf v0.1.0 is incompatible with gg >= v0.40 due to
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

// pdfBackend implements [recording.Backend] for minimal PDF 1.4 output.
// Generates a single-page document from recorded drawing operations.
type pdfBackend struct {
	w, h    int
	streams []string // content stream fragments
}

func newPDFBackend() *pdfBackend {
	return &pdfBackend{}
}

func (b *pdfBackend) Begin(width, height int) error {
	b.w, b.h = width, height
	return nil
}

func (b *pdfBackend) End() error { return nil }
func (b *pdfBackend) Save()      { b.streams = append(b.streams, "q\n") }
func (b *pdfBackend) Restore()   { b.streams = append(b.streams, "Q\n") }

func (b *pdfBackend) SetTransform(m recording.Matrix) {
	b.streams = append(b.streams, fmt.Sprintf("%.4f %.4f %.4f %.4f %.4f %.4f cm\n",
		m.A, m.D, m.B, m.E, m.C, m.F))
}

func (b *pdfBackend) SetClip(path *gg.Path, rule recording.FillRule) {
	d := pathToPDFOps(path, float64(b.h))
	if d == "" {
		return
	}
	b.streams = append(b.streams, d)
	if rule == recording.FillRuleEvenOdd {
		b.streams = append(b.streams, "W* n\n")
	} else {
		b.streams = append(b.streams, "W n\n")
	}
}

func (b *pdfBackend) ClearClip() {}

func (b *pdfBackend) FillPath(path *gg.Path, brush recording.Brush, rule recording.FillRule) {
	d := pathToPDFOps(path, float64(b.h))
	if d == "" {
		return
	}
	b.streams = append(b.streams, pdfSetFillColor(brush))
	b.streams = append(b.streams, d)
	if rule == recording.FillRuleEvenOdd {
		b.streams = append(b.streams, "f*\n")
	} else {
		b.streams = append(b.streams, "f\n")
	}
}

func (b *pdfBackend) StrokePath(path *gg.Path, brush recording.Brush, stroke recording.Stroke) {
	d := pathToPDFOps(path, float64(b.h))
	if d == "" {
		return
	}
	lw := stroke.Width
	if lw <= 0 {
		lw = 1
	}
	b.streams = append(b.streams, fmt.Sprintf("%.2f w\n", lw))
	b.streams = append(b.streams, pdfSetStrokeColor(brush))
	b.streams = append(b.streams, d)
	b.streams = append(b.streams, "S\n")
}

func (b *pdfBackend) FillRect(rect recording.Rect, brush recording.Brush) {
	h := float64(b.h)
	y := h - rect.Y() - rect.Height()
	b.streams = append(b.streams, pdfSetFillColor(brush))
	b.streams = append(b.streams, fmt.Sprintf("%.2f %.2f %.2f %.2f re f\n",
		rect.X(), y, rect.Width(), rect.Height()))
}

func (b *pdfBackend) DrawImage(img image.Image, src, dst recording.Rect, opts recording.ImageOptions) {
	// Inline image / XObject embedding deferred.
}

func (b *pdfBackend) DrawText(s string, x, y float64, face text.Face, brush recording.Brush) {
	h := float64(b.h)
	py := h - y // PDF Y-up
	escaped := pdfEscapeStr(s)
	b.streams = append(b.streams, pdfSetFillColor(brush))
	b.streams = append(b.streams, "BT\n")
	b.streams = append(b.streams, "/F1 12 Tf\n")
	b.streams = append(b.streams, fmt.Sprintf("%.2f %.2f Td\n", x, py))
	b.streams = append(b.streams, fmt.Sprintf("(%s) Tj\n", escaped))
	b.streams = append(b.streams, "ET\n")
}

// WriteTo writes the complete PDF document to w.
func (b *pdfBackend) WriteTo(w io.Writer) (int64, error) {
	cw := &pdfWriter{w: w}
	b.writePDF(cw)
	return cw.n, cw.err
}

func (b *pdfBackend) writePDF(w *pdfWriter) {
	content := strings.Join(b.streams, "")
	contentLen := len(content)

	var off [6]int

	w.s("%PDF-1.4\n")

	off[1] = int(w.n)
	w.s("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	off[2] = int(w.n)
	w.s("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	off[3] = int(w.n)
	w.s(fmt.Sprintf(
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Contents 5 0 R /Resources << /Font << /F1 4 0 R >> >> >>\nendobj\n",
		b.w, b.h))

	off[4] = int(w.n)
	w.s("4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	off[5] = int(w.n)
	w.s(fmt.Sprintf("5 0 obj\n<< /Length %d >>\nstream\n", contentLen))
	w.s(content)
	w.s("endstream\nendobj\n")

	xrefOff := int(w.n)
	w.s("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		w.s(fmt.Sprintf("%010d 00000 n \n", off[i]))
	}
	w.s("trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n")
	w.s(fmt.Sprintf("%d\n%%%%EOF\n", xrefOff))
}

type pdfWriter struct {
	w   io.Writer
	n   int64
	err error
}

func (w *pdfWriter) s(str string) {
	if w.err != nil {
		return
	}
	n, err := io.WriteString(w.w, str)
	w.n += int64(n)
	w.err = err
}

// --- PDF helpers ---

func pathToPDFOps(path *gg.Path, pageH float64) string {
	if path == nil {
		return ""
	}
	var sb strings.Builder
	path.Iterate(func(verb gg.PathVerb, coords []float64) {
		switch verb {
		case gg.MoveTo:
			fmt.Fprintf(&sb, "%.2f %.2f m\n", coords[0], pageH-coords[1])
		case gg.LineTo:
			fmt.Fprintf(&sb, "%.2f %.2f l\n", coords[0], pageH-coords[1])
		case gg.QuadTo:
			// Elevate to cubic: control point is repeated.
			fmt.Fprintf(&sb, "%.2f %.2f %.2f %.2f %.2f %.2f c\n",
				coords[0], pageH-coords[1],
				coords[0], pageH-coords[1],
				coords[2], pageH-coords[3])
		case gg.CubicTo:
			fmt.Fprintf(&sb, "%.2f %.2f %.2f %.2f %.2f %.2f c\n",
				coords[0], pageH-coords[1],
				coords[2], pageH-coords[3],
				coords[4], pageH-coords[5])
		case gg.Close:
			sb.WriteString("h\n")
		}
	})
	return sb.String()
}

func pdfSetFillColor(brush recording.Brush) string {
	switch b := brush.(type) {
	case recording.SolidBrush:
		return fmt.Sprintf("%.3f %.3f %.3f rg\n",
			math.Max(0, math.Min(1, b.Color.R)),
			math.Max(0, math.Min(1, b.Color.G)),
			math.Max(0, math.Min(1, b.Color.B)))
	default:
		return "0 0 0 rg\n"
	}
}

func pdfSetStrokeColor(brush recording.Brush) string {
	switch b := brush.(type) {
	case recording.SolidBrush:
		return fmt.Sprintf("%.3f %.3f %.3f RG\n",
			math.Max(0, math.Min(1, b.Color.R)),
			math.Max(0, math.Min(1, b.Color.G)),
			math.Max(0, math.Min(1, b.Color.B)))
	default:
		return "0 0 0 RG\n"
	}
}

func pdfEscapeStr(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

func exportPDF(r *recording.Recording, w io.Writer) (int64, error) {
	b := newPDFBackend()
	if err := r.Playback(b); err != nil {
		return 0, err
	}
	return b.WriteTo(w)
}
