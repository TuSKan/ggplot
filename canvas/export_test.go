package canvas_test

import (
	"bytes"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/TuSKan/ggplot/canvas"
)

// The recorder bakes the active transform into every coordinate, so the export
// backends must not re-apply it. These tests guard against the double-transform
// regression that shifted the panel's data layer off the axes.

func TestExportSVGBakesTransformOnce(t *testing.T) {
	t.Parallel()

	cv := canvas.NewRecordingCanvas(100, 100)
	cv.Save()
	cv.Translate(10, 20)
	cv.MoveTo(0, 0)
	cv.LineTo(5, 0)
	cv.Stroke()
	cv.Restore()

	var buf bytes.Buffer
	if _, err := canvas.ExportSVG(cv.FinishRecording(), &buf); err != nil {
		t.Fatalf("ExportSVG: %v", err)
	}

	svg := buf.String()

	if strings.Contains(svg, "transform=") {
		t.Errorf("SVG must not re-apply transforms (coordinates are pre-baked):\n%s", svg)
	}

	// Local (0,0)-(5,0) under Translate(10,20) bakes to world (10,20)-(15,20).
	if !strings.Contains(svg, "M10.00 20.00") {
		t.Errorf("expected line baked to world (10,20); SVG:\n%s", svg)
	}
}

func TestExportPDFNoTransformOps(t *testing.T) {
	t.Parallel()

	cv := canvas.NewRecordingCanvas(100, 100)
	cv.Save()
	cv.Translate(10, 20)
	cv.MoveTo(0, 0)
	cv.LineTo(5, 0)
	cv.Stroke()
	cv.Restore()

	var buf bytes.Buffer
	if _, err := canvas.ExportPDF(cv.FinishRecording(), &buf); err != nil {
		t.Fatalf("ExportPDF: %v", err)
	}

	pdf := buf.String()

	if strings.Contains(pdf, " cm\n") {
		t.Error("PDF must not emit a cm transform; coordinates are pre-baked")
	}

	// World (10,20) with PDF Y-up on a height-100 page is (10, 80).
	if !strings.Contains(pdf, "10.00 80.00 m") {
		t.Errorf("expected line baked to world; PDF stream:\n%s", pdf)
	}
}

func TestExportSVGAnchorsText(t *testing.T) {
	t.Parallel()

	cv := canvas.NewRecordingCanvas(200, 100)
	cv.SetFontSize(12)
	cv.DrawStringAnchored("Hello", 100, 50, 0.5, 0.5) // horizontally centered on x=100

	var buf bytes.Buffer
	if _, err := canvas.ExportSVG(cv.FinishRecording(), &buf); err != nil {
		t.Fatalf("ExportSVG: %v", err)
	}

	m := regexp.MustCompile(`<text x="([0-9.]+)"`).FindStringSubmatch(buf.String())
	if m == nil {
		t.Fatalf("no <text> element emitted:\n%s", buf.String())
	}

	x, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		t.Fatalf("parse text x: %v", err)
	}

	// Center anchor (ax=0.5) must shift the emitted x left of the anchor point.
	if x >= 100 {
		t.Errorf("centered text x=%.2f, want < 100 (anchor must be pre-applied)", x)
	}
}

// Rotated text (axis titles, facet strips, slanted ticks) is drawn under a
// Save/Translate/Rotate. The recorder bakes only the anchor position, so the
// backends must recover the glyph orientation from the tracked CTM. These tests
// guard against the regression where rotated vector text rendered upright.

func TestExportSVGRotatesText(t *testing.T) {
	t.Parallel()

	cv := canvas.NewRecordingCanvas(200, 200)
	cv.SetFontSize(12)
	cv.Save()
	cv.Translate(20, 100)
	cv.Rotate(-math.Pi / 2) // y-axis title: reads bottom-to-top
	cv.DrawStringAnchored("y axis", 0, 0, 0.5, 0.5)
	cv.Restore()

	var buf bytes.Buffer
	if _, err := canvas.ExportSVG(cv.FinishRecording(), &buf); err != nil {
		t.Fatalf("ExportSVG: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, `transform="rotate(-90`) {
		t.Errorf("rotated text must emit a rotate(-90 ...) transform; SVG:\n%s", svg)
	}
}

func TestExportSVGDoesNotRotateUprightText(t *testing.T) {
	t.Parallel()

	cv := canvas.NewRecordingCanvas(200, 200)
	cv.SetFontSize(12)
	cv.Save()
	cv.Translate(20, 100) // translation only — no rotation
	cv.DrawStringAnchored("title", 0, 0, 0.5, 0.5)
	cv.Restore()

	var buf bytes.Buffer
	if _, err := canvas.ExportSVG(cv.FinishRecording(), &buf); err != nil {
		t.Fatalf("ExportSVG: %v", err)
	}

	if strings.Contains(buf.String(), "transform=") {
		t.Errorf("upright text must not emit a transform; SVG:\n%s", buf.String())
	}
}

func TestExportPDFRotatesText(t *testing.T) {
	t.Parallel()

	cv := canvas.NewRecordingCanvas(200, 200)
	cv.SetFontSize(12)
	cv.Save()
	cv.Translate(20, 100)
	cv.Rotate(-math.Pi / 2)
	cv.DrawStringAnchored("y axis", 0, 0, 0.5, 0.5)
	cv.Restore()

	var buf bytes.Buffer
	if _, err := canvas.ExportPDF(cv.FinishRecording(), &buf); err != nil {
		t.Fatalf("ExportPDF: %v", err)
	}

	pdf := buf.String()
	if !strings.Contains(pdf, " Tm\n") {
		t.Errorf("rotated text must use a Tm text matrix; PDF stream:\n%s", pdf)
	}

	if strings.Contains(pdf, " Td\n") {
		t.Errorf("rotated text must not fall back to Td; PDF stream:\n%s", pdf)
	}
}

func TestExportSVGResponsiveStyle(t *testing.T) {
	t.Parallel()

	cv := canvas.NewRecordingCanvas(400, 300)

	var buf bytes.Buffer
	if _, err := canvas.ExportSVG(cv.FinishRecording(), &buf); err != nil {
		t.Fatalf("ExportSVG: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, `style="max-width:100%;height:auto"`) {
		t.Errorf("SVG root must have responsive style; SVG:\n%s", svg)
	}
}

func TestExportSVGWithMetaTitle(t *testing.T) {
	t.Parallel()

	cv := canvas.NewRecordingCanvas(100, 100)
	cv.SetMetadata(map[string]string{"title": "My Tooltip"})
	cv.DrawRectangle(10, 10, 30, 30)
	cv.Fill()

	rec := cv.FinishRecording()
	meta := cv.MetadataMap()

	var buf bytes.Buffer
	if _, err := canvas.ExportSVGWithMeta(rec, meta, &buf); err != nil {
		t.Fatalf("ExportSVGWithMeta: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "<title>My Tooltip</title>") {
		t.Errorf("SVG must contain <title> tooltip; SVG:\n%s", svg)
	}
}

func TestExportSVGWithMetaHref(t *testing.T) {
	t.Parallel()

	cv := canvas.NewRecordingCanvas(100, 100)
	cv.SetMetadata(map[string]string{"href": "https://example.com"})
	cv.DrawRectangle(10, 10, 30, 30)
	cv.Fill()

	rec := cv.FinishRecording()
	meta := cv.MetadataMap()

	var buf bytes.Buffer
	if _, err := canvas.ExportSVGWithMeta(rec, meta, &buf); err != nil {
		t.Fatalf("ExportSVGWithMeta: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, `<a href="https://example.com">`) {
		t.Errorf("SVG must contain <a href> wrapper; SVG:\n%s", svg)
	}

	if !strings.Contains(svg, `</a>`) {
		t.Errorf("SVG must close <a> wrapper; SVG:\n%s", svg)
	}
}

func TestExportSVGWithMetaAriaLabel(t *testing.T) {
	t.Parallel()

	cv := canvas.NewRecordingCanvas(100, 100)
	cv.SetMetadata(map[string]string{"aria_label": "Data point A"})
	cv.DrawRectangle(10, 10, 30, 30)
	cv.Fill()

	rec := cv.FinishRecording()
	meta := cv.MetadataMap()

	var buf bytes.Buffer
	if _, err := canvas.ExportSVGWithMeta(rec, meta, &buf); err != nil {
		t.Fatalf("ExportSVGWithMeta: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, `aria-label="Data point A"`) {
		t.Errorf("SVG must contain aria-label attribute; SVG:\n%s", svg)
	}
}

func TestExportSVGWithMetaCombined(t *testing.T) {
	t.Parallel()

	cv := canvas.NewRecordingCanvas(100, 100)
	cv.SetMetadata(map[string]string{
		"title":      "Point 1",
		"href":       "https://example.com/1",
		"aria_label": "First data point",
	})
	cv.DrawCircle(50, 50, 5)
	cv.Fill()

	rec := cv.FinishRecording()
	meta := cv.MetadataMap()

	var buf bytes.Buffer
	if _, err := canvas.ExportSVGWithMeta(rec, meta, &buf); err != nil {
		t.Fatalf("ExportSVGWithMeta: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, `<title>Point 1</title>`) {
		t.Errorf("missing title; SVG:\n%s", svg)
	}

	if !strings.Contains(svg, `<a href="https://example.com/1">`) {
		t.Errorf("missing href; SVG:\n%s", svg)
	}

	if !strings.Contains(svg, `aria-label="First data point"`) {
		t.Errorf("missing aria-label; SVG:\n%s", svg)
	}
}

func TestExportSVGNoMetadata(t *testing.T) {
	t.Parallel()

	// No SetMetadata calls — should produce clean SVG without metadata wrappers.
	cv := canvas.NewRecordingCanvas(100, 100)
	cv.DrawRectangle(10, 10, 30, 30)
	cv.Fill()

	var buf bytes.Buffer
	if _, err := canvas.ExportSVG(cv.FinishRecording(), &buf); err != nil {
		t.Fatalf("ExportSVG: %v", err)
	}

	svg := buf.String()

	if strings.Contains(svg, "<title>") {
		t.Errorf("SVG without metadata should not contain <title>; SVG:\n%s", svg)
	}

	if strings.Contains(svg, "<a ") {
		t.Errorf("SVG without metadata should not contain <a>; SVG:\n%s", svg)
	}

	if strings.Contains(svg, "aria-label") {
		t.Errorf("SVG without metadata should not contain aria-label; SVG:\n%s", svg)
	}
}
