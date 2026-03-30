package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/output/file"
	"github.com/TuSKan/ggplot/pkg/output/ui"
	"github.com/TuSKan/ggplot/pkg/plot"
	"github.com/TuSKan/ggplot/pkg/stat"
	"github.com/TuSKan/ggplot/pkg/theme"
)

func main() {
	// 1. Generate normal distribution utilizing Zero-Copy Buffer.
	buf := arrow.NewBuffer(1000)
	value := buf.Float64("value")

	for i := 0; i < 1000; i++ {
		// Box-Muller normal approximation functionally
		u1 := rand.Float64()
		u2 := rand.Float64()
		z := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
		value[i] = 50.0 + 15.0*z
	}

	ds, err := buf.Build()
	if err != nil {
		log.Fatalln(err)
	}

	darkTheme := theme.Default()
	darkTheme.Background = theme.ParseHexColor("#1E1E1E")
	darkTheme.Typography.Title.Color = theme.ParseHexColor("#FFFFFF")
	darkTheme.Typography.Subtitle.Color = theme.ParseHexColor("#AAAAAA")
	darkTheme.Typography.AxisTitle.Color = theme.ParseHexColor("#DDDDDD")
	darkTheme.Typography.TickLabel.Color = theme.ParseHexColor("#888888")
	darkTheme.Typography.Legend.Color = theme.ParseHexColor("#888888")

	p := plot.New(ds).
		Theme(darkTheme).
		Title("Normal Distribution").
		Subtitle("Generated via Zero-Copy Hardware Buffer").
		XAxis("Value").
		YAxis("Frequency").
		AddLayer(
			geom.Histogram(
				geom.Opts{Opacity: 0.85, Fill: "#00E5FF", Color: "#1E1E1E", LineWidth: 1.0},
				stat.NewBin(stat.BinOptions{Bins: 40}),
			),
			aes.X(aes.Col("value")),
		)

	out, err := p.Render(800, 600)
	if err != nil {
		log.Fatalln(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	err = file.NewFileExporter().Export(out, filepath.Join(filepath.Dir(filename), "histogram.png"))
	if err != nil {
		log.Fatalln(err)
	}

	presenter := ui.NewGPUWindowPresenter()
	fmt.Println("Launching interactive GPU presenter overlay...")
	if err := presenter.Show(out); err != nil {
		log.Fatalln(err)
	}
}
