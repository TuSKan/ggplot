// Example: Output layer.
//
// Demonstrates the unified output layer added in OUTPUT-SPEC.md Phases 1–5:
//   - file.Save        — file export (PNG / SVG / PDF) via the "file" surface
//   - oimg.Render      — render to an in-memory image.Image
//   - file.Encode      — stream encoded bytes to any io.Writer
//   - output.NewSurface + output.Render — the low-level Surface API
//
// The built-in "file" and "image" surfaces are auto-registered by importing the
// ggplot package, so no blank import is needed here. A live desktop window
// (output/window.Show) needs a display and is shown as a comment at the end.
package main

import (
	"context"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output"
	"github.com/TuSKan/ggplot/output/file"
	oimg "github.com/TuSKan/ggplot/output/image"
)

func main() {
	ctx := context.Background()
	eng := memory.NewEngine(ctx)

	_, srcFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(srcFile)

	// Simple scatter + smooth.
	n := 40
	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range n {
		xs[i] = float64(i)
		ys[i] = 2*float64(i) + 8*math.Sin(float64(i)/3)
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	if err != nil {
		log.Fatalln(err)
	}

	plot := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(5), geom.WithColor("coral"))).
		Layer(geom.Smooth(geom.WithColor("steelblue"))).
		Labels(ggplot.Title("Output Layer Demo"), ggplot.XLabel("x"), ggplot.YLabel("y"))

	// 1. Plot.Save — one call per format; the encoder is inferred from the
	//    extension. Each goes through output.Render + the "file" surface.
	for _, ext := range []string{"png", "svg", "pdf"} {
		out := filepath.Join(dir, "plot."+ext)
		if err := file.Save(ctx, plot, out, 800, 500); err != nil {
			log.Fatalln(err)
		}

		log.Println("Save     ->", filepath.Base(out))
	}

	// 2. oimg.Render — render into an in-memory image.Image, then hand it to the
	//    standard library (here, encode it ourselves) to show interop.
	img, err := oimg.Render(ctx, plot, 400, 250)
	if err != nil {
		log.Fatalln(err)
	}

	imgFile, err := os.Create(filepath.Join(dir, "image.png"))
	if err != nil {
		log.Fatalln(err)
	}

	if err := png.Encode(imgFile, img); err != nil {
		log.Fatalln(err)
	}

	_ = imgFile.Close()

	log.Printf("Image    -> image.png (%dx%d in-memory image)", img.Bounds().Dx(), img.Bounds().Dy())

	// 3. file.Encode — stream encoded bytes to any io.Writer (a file here, but
	//    it could be an HTTP response, gzip writer, blob store, ...).
	encFile, err := os.Create(filepath.Join(dir, "encoded.svg"))
	if err != nil {
		log.Fatalln(err)
	}

	written, err := file.Encode(ctx, plot, encFile, "svg", 600, 380)
	_ = encFile.Close()

	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Encode   -> encoded.svg (%d bytes via io.Writer)", written)

	// 4. The low-level Surface API: build once, then render onto a surface.
	fig, err := plot.Build(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	surf, err := output.NewSurface(ctx, "file",
		output.WithPath(filepath.Join(dir, "surface.png")),
		output.WithSize(640, 400),
	)
	if err != nil {
		log.Fatalln(err)
	}

	if err := output.Render(ctx, fig, surf); err != nil {
		log.Fatalln(err)
	}

	_ = surf.Close()

	log.Println("Surface  -> surface.png (output.NewSurface + output.Render)")

	// 5. output.Render — render the built figure onto any Surface (here, an
	//    in-memory image surface read back via output.Imager).
	imgSurf, err := output.NewSurface(ctx, "image", output.WithSize(320, 200))
	if err != nil {
		log.Fatalln(err)
	}

	if err := output.Render(ctx, fig, imgSurf); err != nil {
		log.Fatalln(err)
	}

	if im, ok := imgSurf.(output.Imager); ok {
		log.Printf("Render   -> image surface (%dx%d)", im.Image().Bounds().Dx(), im.Image().Bounds().Dy())
	}

	_ = imgSurf.Close()

	// 6. Interactive desktop window (not run here — needs a display/GPU):
	//
	//	import "github.com/TuSKan/ggplot/output/window"
	//
	//	// from main(), on the main goroutine:
	//	err := window.Show(ctx, plot,
	//	    window.WithTitle("ggplot"),
	//	    window.WithSize(900, 600))
	//
	// window.Show opens a GPU window, presents the plot zero-copy, supports
	// pan (drag) and wheel-zoom, and blocks until the window is closed. See the
	// runnable examples/window for the full program.

	log.Println("all output written to", dir)
}
