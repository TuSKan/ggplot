package main

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/theme"
	"github.com/TuSKan/ggplot/pkg/output/file"
	"github.com/TuSKan/ggplot/pkg/output/ui"
	"github.com/TuSKan/ggplot/pkg/plot"
)

func main() {

	// 1. Dataset Generation encapsulated functionally fully.
	buf := arrow.NewBuffer(2000)

	// Claim zero-copy hardware memory slices.
	intensity := buf.Float64("Intensity")
	displacement := buf.Float64("Displacement")
	radius := buf.Float64("Radius")

	// populate
	generateButterflyCurveZeroCopy(2000, intensity, displacement, radius)

	ds, err := buf.Build()
	if err != nil {
		log.Fatalln(err)
	}

	darkTheme := theme.Default()
	darkTheme.Background = theme.ParseHexColor("#09090B")
	darkTheme.Typography.Title.Color = theme.ParseHexColor("#FFFFFF")
	darkTheme.Typography.Subtitle.Color = theme.ParseHexColor("#A1A1AA")
	darkTheme.Typography.AxisTitle.Color = theme.ParseHexColor("#E4E4E7")
	darkTheme.Typography.Legend.Color = theme.ParseHexColor("#E4E4E7")

	p := plot.New(ds).
		Aes(
			aes.X(aes.Col("Intensity")),
			aes.Y(aes.Col("Displacement")),
			aes.Color(aes.Col("Radius")),
		).
		Theme(darkTheme).
		Title("Butterfly Curve").
		Subtitle("Temple H. Fay's Parametric Equation").
		XAxis("Intensity").
		YAxis("Displacement").
		Legend("right", "Radius Scale").
		AddLayer(geom.Line(geom.Opts{Radius: 1.0, Opacity: 0.9, LineWidth: 1.5, Color: "#8B5CF6"}))

	out, err := p.Render(800, 800)
	if err != nil {
		log.Fatalln(err)
	}


	presenter := ui.NewGPUWindowPresenter()
	fmt.Println("Launching interactive GPU presenter overlay...")
	if err := presenter.Show(out); err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	file.NewFileExporter().Export(out, filepath.Join(filepath.Dir(filename), "butterfly.png"))
}

// generateButterflyCurveZeroCopy maps Temple H. Fay's parametric equation directly into zero-copy buffers directly.
func generateButterflyCurveZeroCopy(numPoints int, xData, yData, zData []float64) {
	maxT := 12.0 * math.Pi
	step := maxT / float64(numPoints-1)

	for i := 0; i < numPoints; i++ {
		t := float64(i) * step

		eCosT := math.Exp(math.Cos(t))
		cos4T := math.Cos(4.0 * t)
		sinT12 := math.Sin(t / 12.0)
		sinT12_5 := sinT12 * sinT12 * sinT12 * sinT12 * sinT12

		term := eCosT - 2.0*cos4T - sinT12_5

		xData[i] = math.Sin(t) * term
		yData[i] = math.Cos(t) * term
		zData[i] = math.Sqrt(xData[i]*xData[i] + yData[i]*yData[i])
	}
}
