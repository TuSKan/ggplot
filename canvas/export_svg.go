// export_svg.go — SVG export backend.
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
	w, h  int
	buf   strings.Builder
	ctm   recording.Matrix   // tracked transform; used only to orient text
	stack []recording.Matrix // Save/Restore stack for ctm

	// Metadata side-channel: draw-op counter syncs with RecordingCanvas.
	metadata    map[int]map[string]string
	drawOpCount int
}

func newSVGBackend() *svgBackend {
	return &svgBackend{ctm: recording.Identity()}
}

func (b *svgBackend) Begin(width, height int) error {
	b.w, b.h = width, height
	b.ctm = recording.Identity()
	fmt.Fprintf(&b.buf, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"+
		"<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\""+
		" style=\"max-width:100%%;height:auto\">\n",
		width, height, width, height)

	return nil
}

func (b *svgBackend) End() error {
	b.buf.WriteString("</svg>\n")
	return nil
}

// The recorder bakes the current transform into every coordinate it emits (see
// recording.Recorder.TransformPoint), so the transform must NOT be re-applied to
// path geometry — doing so double-transforms it (e.g. shifts the panel's data
// layer off the axes). Paths/rects therefore use the baked world coordinates
// verbatim and never consult the matrix.
//
// Text is the exception: the recorder bakes only the anchor *position*, not the
// glyph *orientation*, so rotated text (axis titles, facet strips, slanted tick
// labels) would otherwise render upright. We track the CTM here purely to
// recover the rotation in [svgBackend.DrawText]; it is not applied to geometry.
func (b *svgBackend) Save() { b.stack = append(b.stack, b.ctm) }

func (b *svgBackend) Restore() {
	if n := len(b.stack); n > 0 {
		b.ctm = b.stack[n-1]
		b.stack = b.stack[:n-1]
	}
}

func (b *svgBackend) SetTransform(m recording.Matrix)          { b.ctm = m }
func (b *svgBackend) SetClip(_ *gg.Path, _ recording.FillRule) {}
func (b *svgBackend) ClearClip()                               {}

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

	fmt.Fprintf(&b.buf, "<path d=\"%s\" fill=\"%s\"%s%s stroke=\"none\"/>",
		d, fill, opacityAttr, ruleAttr)
	b.emitDrawOp()
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

	fmt.Fprintf(&b.buf, "<path d=\"%s\" fill=\"none\" stroke=\"%s\" stroke-width=\"%.2f\"%s/>",
		d, col, lw, extra)
	b.emitDrawOp()
}

func (b *svgBackend) FillRect(rect recording.Rect, brush recording.Brush) {
	fill, opacity := brushToSVG(brush)

	opacityAttr := ""
	if opacity < 1 {
		opacityAttr = fmt.Sprintf(` fill-opacity="%.3f"`, opacity)
	}

	fmt.Fprintf(&b.buf, "<rect x=\"%.2f\" y=\"%.2f\" width=\"%.2f\" height=\"%.2f\" fill=\"%s\"%s/>",
		rect.X(), rect.Y(), rect.Width(), rect.Height(), fill, opacityAttr)
	b.emitDrawOp()
}

func (b *svgBackend) DrawImage(_ image.Image, _, _ recording.Rect, _ recording.ImageOptions) {
	// Image embedding requires base64 inline data — deferred.
}

func (b *svgBackend) DrawText(s string, x, y float64, face text.Face, brush recording.Brush) {
	fill, _ := brushToSVG(brush)
	escaped := svgEscape(s)

	fontSize := 12.0
	if face != nil {
		fontSize = face.Size()
	}

	// Check if face has tabular-nums feature enabled.
	variantAttr := ""

	if face != nil {
		for _, f := range face.Features() {
			if f.Tag == [4]byte{'t', 'n', 'u', 'm'} && f.Value > 0 {
				variantAttr = ` font-variant-numeric="tabular-nums"`

				break
			}
		}
	}

	// The anchor position (x,y) is already baked into world space; recover only
	// the glyph orientation from the tracked CTM and rotate about that point.
	transformAttr := ""
	if deg := textRotationDeg(b.ctm); deg != 0 {
		transformAttr = fmt.Sprintf(` transform="rotate(%.4f %.2f %.2f)"`, deg, x, y)
	}

	fmt.Fprintf(&b.buf,
		"<text x=\"%.2f\" y=\"%.2f\" fill=\"%s\" font-family=\"sans-serif\" font-size=\"%.1f\"%s%s>%s</text>",
		x, y, fill, fontSize, variantAttr, transformAttr, escaped)
	b.emitDrawOp()
}

// textRotationDeg returns the rotation of m in degrees, using the same y-down
// convention as the recorder's gg.Rotate (which SVG's rotate() also uses), or 0
// if the rotation is negligible. Uniform scale does not affect the result.
func textRotationDeg(m recording.Matrix) float64 {
	const eps = 1e-4

	deg := math.Atan2(m.D, m.A) * 180 / math.Pi
	if math.Abs(deg) < eps {
		return 0
	}

	return deg
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
	return exportSVGWithMeta(r, nil, w)
}

func exportSVGWithMeta(r *recording.Recording, meta map[int]map[string]string, w io.Writer) (int64, error) {
	b := newSVGBackend()

	b.metadata = meta

	if err := r.Playback(b); err != nil {
		return 0, fmt.Errorf("canvas: SVG playback: %w", err)
	}

	return b.WriteTo(w)
}

// emitDrawOp wraps the last-emitted element with metadata (title, href,
// aria-label) if present for the current draw-op index, then writes a newline.
func (b *svgBackend) emitDrawOp() {
	if meta, ok := b.metadata[b.drawOpCount]; ok {
		// Find the start of the last element in the buffer.
		raw := b.buf.String()

		var wrappedElem string

		// Extract the last element (the one just written without newline).
		lastNewline := max(strings.LastIndex(raw, "\n"), 0)

		elem := raw[lastNewline:]
		if elem != "" && elem[0] == '\n' {
			elem = elem[1:]
		}

		// Add aria-label attribute to the element if present.
		if ariaLabel, ok := meta["aria_label"]; ok && ariaLabel != "" {
			// Insert aria-label before the closing /> or >
			if idx := strings.Index(elem, "/>"); idx >= 0 {
				elem = elem[:idx] + fmt.Sprintf(" aria-label=\"%s\"", svgEscape(ariaLabel)) + elem[idx:]
			} else if idx := strings.Index(elem, ">"); idx >= 0 {
				elem = elem[:idx] + fmt.Sprintf(" aria-label=\"%s\"", svgEscape(ariaLabel)) + elem[idx:]
			}
		}

		wrappedElem = elem

		// Wrap with <title> if present.
		if title, ok := meta["title"]; ok && title != "" {
			// For self-closing elements, convert to open/close form for <title> child.
			if strings.HasSuffix(wrappedElem, "/>") {
				// Self-closing: <path .../> → <path ...></path>
				tagEnd := strings.Index(wrappedElem, " ")
				if tagEnd < 0 {
					tagEnd = 1 // skip '<'
				}

				tagName := wrappedElem[1:tagEnd]
				wrappedElem = wrappedElem[:len(wrappedElem)-2] + ">" +
					"<title>" + svgEscape(title) + "</title>" +
					"</" + tagName + ">"
			} else {
				// Already has closing tag (e.g. <text>...</text>).
				// Insert <title> after the opening tag.
				if idx := strings.Index(wrappedElem, ">"); idx >= 0 {
					wrappedElem = wrappedElem[:idx+1] +
						"<title>" + svgEscape(title) + "</title>" +
						wrappedElem[idx+1:]
				}
			}
		}

		// Wrap with <a href> if present.
		if href, ok := meta["href"]; ok && href != "" {
			wrappedElem = fmt.Sprintf("<a href=\"%s\">", svgEscape(href)) + wrappedElem + "</a>"
		}

		// Replace the element in the buffer.
		b.buf.Reset()

		if lastNewline > 0 {
			b.buf.WriteString(raw[:lastNewline])
		}

		b.buf.WriteString("\n")
		b.buf.WriteString(wrappedElem)
	}

	b.buf.WriteString("\n")

	b.drawOpCount++
}
