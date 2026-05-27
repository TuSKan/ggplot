// Example: Reference lines and text annotations.
package main

import (
	"context"
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

	referenceLines(dir)
	textLabels(dir)
}

func referenceLines(dir string) {
	// Generate sine wave data.
	n := 100
	x := make([]float64, n)

	y := make([]float64, n)
	for i := range x {
		t := float64(i) / float64(n-1) * 4 * math.Pi
		x[i] = t
		y[i] = math.Sin(t)
	}

	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", x),
		eng.NewFloat64Column("y", y),
	)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#1F77B4"), geom.WithLineWidth(2), geom.WithLabel("sin(x)"))).
		Layer(geom.HLine(
			geom.WithIntercept(0),
			geom.WithColor("#999999"),
			geom.WithLineWidth(1),
			geom.WithLabel("y = 0"),
		)).
		Layer(geom.HLine(
			geom.WithIntercept(0.5),
			geom.WithColor("#E74C3C"),
			geom.WithLineWidth(1),
			geom.WithAlpha(0.6),
			geom.WithLabel("y = 0.5"),
		)).
		Layer(geom.VLine(
			geom.WithIntercept(math.Pi),
			geom.WithColor("#2ECC71"),
			geom.WithLineWidth(1),
			geom.WithAlpha(0.6),
			geom.WithLabel("x = π"),
		)).
		Labs(
			ggplot.Title("Reference Lines"),
			ggplot.Subtitle("HLine and VLine annotations"),
			ggplot.XLab("x"),
			ggplot.YLab("sin(x)"),
		).
		Theme("minimal")

	out := filepath.Join(dir, "annotations.png")
	if err := p.Save(context.Background(), out, 900, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}

func textLabels(dir string) {
	// Peak/trough labels on a sine wave.
	peakX := []float64{math.Pi / 2, 3 * math.Pi / 2, 5 * math.Pi / 2, 7 * math.Pi / 2}
	peakY := []float64{1, -1, 1, -1}
	peakLabels := []string{"peak", "trough", "peak", "trough"}

	eng2 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng2,
		eng2.NewFloat64Column("x", peakX),
		eng2.NewFloat64Column("y", peakY),
		eng2.NewStringColumn("label", peakLabels),
	)

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Label("label")).
		Layer(geom.Point(geom.WithColor("#1F77B4"), geom.WithSize(5))).
		Layer(geom.Text(geom.WithColor("#E74C3C"), geom.WithFontSize(11))).
		YLim(-1.3, 1.3).
		Labs(
			ggplot.Title("Text Annotations"),
			ggplot.Subtitle("Labels at peaks and troughs"),
			ggplot.XLab("x"),
			ggplot.YLab("f(x)"),
		).
		Theme("minimal")

	out := filepath.Join(dir, "text_labels.png")
	if err := p.Save(context.Background(), out, 900, 500); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", out)
}
