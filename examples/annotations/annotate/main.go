// Example annotate demonstrates the annotation API — layer-less visual
// elements at fixed data coordinates. Annotations bypass the data pipeline
// entirely, making them ideal for callouts, highlights, and labels.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// Generate sine wave data: 2 full periods.
	n := 300

	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range n {
		t := float64(i) / float64(n-1) * 4 * math.Pi //nolint:mnd // 2 full periods.
		xs[i] = t
		ys[i] = math.Sin(t)
	}

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// Key positions on the sine wave.
	peak1 := math.Pi / 2       //nolint:mnd // First peak  (y = +1).
	trough1 := 3 * math.Pi / 2 //nolint:mnd // First trough (y = −1).
	peak2 := 5 * math.Pi / 2   //nolint:mnd // Second peak  (y = +1).
	trough2 := 7 * math.Pi / 2 //nolint:mnd // Second trough (y = −1).
	period2 := 2 * math.Pi     // Start of second period.

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#1F77B4"), geom.WithLineWidth(2))).

		// --- RECT: shade first-period region (background) ---
		Annotate(ggplot.AnnotateRect(0, -1.5, period2, 1.5,
			geom.WithFill("#DBEAFE"),
			geom.WithAlpha(0.2),
		)).

		// --- RECT: shade second-period region (background) ---
		Annotate(ggplot.AnnotateRect(period2, -1.5, 4*math.Pi, 1.5,
			geom.WithFill("#FEF3C7"),
			geom.WithAlpha(0.15),
		)).

		// --- SEGMENT: horizontal zero baseline ---
		Annotate(ggplot.AnnotateSegment(0, 0, 4*math.Pi, 0,
			geom.WithColor("#BDC3C7"),
			geom.WithLineWidth(0.8),
			geom.WithAlpha(0.5),
		)).

		// --- SEGMENT: vertical period boundary ---
		Annotate(ggplot.AnnotateSegment(period2, -1.4, period2, 1.4,
			geom.WithColor("#95A5A6"),
			geom.WithLineWidth(1),
			geom.WithAlpha(0.6),
		)).

		// --- TEXT: peak labels (green) ---
		Annotate(ggplot.AnnotateText(peak1, 0.80, "peak",
			geom.WithColor("#27AE60"),
			geom.WithFontSize(11),
		)).
		Annotate(ggplot.AnnotateText(peak2, 0.80, "peak",
			geom.WithColor("#27AE60"),
			geom.WithFontSize(11),
		)).

		// --- TEXT: trough labels (red) ---
		Annotate(ggplot.AnnotateText(trough1, -0.80, "trough",
			geom.WithColor("#E74C3C"),
			geom.WithFontSize(11),
		)).
		Annotate(ggplot.AnnotateText(trough2, -0.80, "trough",
			geom.WithColor("#E74C3C"),
			geom.WithFontSize(11),
		)).

		// --- ARROW: callout from label to first peak ---
		Annotate(ggplot.AnnotateArrow(peak1+1.8, 0.55, peak1+0.15, 0.95,
			geom.WithColor("#2C3E50"),
			geom.WithLineWidth(1.5),
		)).

		// --- LABEL: value at first peak arrow origin (transparent bg) ---
		Annotate(ggplot.AnnotateLabel(peak1+1.8, 0.40, "y = 1.0",
			geom.WithFill("#FFFFFF"),
			geom.WithColor("#2C3E50"),
			geom.WithFontSize(10),
			geom.WithAlpha(0.6),
		)).

		// --- ARROW: callout to first trough ---
		Annotate(ggplot.AnnotateArrow(trough1+2.2, -0.50, trough1+0.15, -0.95,
			geom.WithColor("#C0392B"),
			geom.WithLineWidth(1.5),
		)).

		// --- LABEL: value at first trough (transparent bg) ---
		Annotate(ggplot.AnnotateLabel(trough1+2.2, -0.35, "y = -1.0",
			geom.WithFill("#FFFFFF"),
			geom.WithColor("#C0392B"),
			geom.WithFontSize(10),
			geom.WithAlpha(0.6),
		)).

		// --- LABEL: period boundary marker (transparent bg) ---
		Annotate(ggplot.AnnotateLabel(period2, 1.25, "T = 2\u03c0",
			geom.WithFill("#F8F9FA"),
			geom.WithColor("#7F8C8D"),
			geom.WithFontSize(9),
			geom.WithAlpha(0.6),
		)).

		// --- LABEL: period name labels at bottom (transparent bg) ---
		Annotate(ggplot.AnnotateLabel(math.Pi, -1.30, "Period 1",
			geom.WithFill("#DBEAFE"),
			geom.WithColor("#2980B9"),
			geom.WithFontSize(9),
			geom.WithAlpha(0.6),
		)).
		Annotate(ggplot.AnnotateLabel(3*math.Pi, -1.30, "Period 2",
			geom.WithFill("#FEF3C7"),
			geom.WithColor("#E67E22"),
			geom.WithFontSize(9),
			geom.WithAlpha(0.6),
		)).

		// Y-limits with ample headroom so no annotations are clipped.
		YLim(-1.5, 1.5).
		Labs(
			ggplot.Title("Annotation Showcase"),
			ggplot.Subtitle("Text · Rect · Segment · Arrow · Label"),
			ggplot.XLab("x (radians)"),
			ggplot.YLab("sin(x)"),
		).
		Theme("minimal")

	out := filepath.Join(dir, "annotate.png")
	if err := p.Save(context.Background(), out, 1000, 550); err != nil { //nolint:mnd // Wider canvas for rich example.
		log.Fatalln(err)
	}

	fmt.Println("saved", out)
}
