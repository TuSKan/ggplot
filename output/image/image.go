// Package image provides a [output.Surface] that renders a single frame into an
// in-memory image.Image. Blank-import it to register the "image" surface:
//
//	import _ "github.com/TuSKan/ggplot/output/image"
package image

import (
	"context"
	"fmt"
	stdimage "image"
	"image/draw"

	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/output"
)

//nolint:gochecknoinits // Blank-import surface registration (the idiomatic Go driver pattern, cf. image/png).
func init() { output.Register("image", newImageSurface) }

// imageSurface is a single-shot [output.Surface] that publishes the rendered
// frame as an in-memory image. It always uses the pure-CPU rasterizer so the
// result is deterministic and headless-safe.
type imageSurface struct {
	sw, sh   int
	cv       *canvas.RasterCanvas
	img      stdimage.Image
	consumed bool
}

func newImageSurface(_ context.Context, opt output.SurfaceOptions) (output.Surface, error) {
	scale := opt.Scale
	if scale <= 0 {
		scale = 1
	}

	return &imageSurface{
		sw: int(float64(opt.Width) * scale),
		sh: int(float64(opt.Height) * scale),
	}, nil
}

func (s *imageSurface) Acquire(_ context.Context) (canvas.Canvas, error) {
	if s.consumed {
		return nil, output.ErrSurfaceConsumed
	}

	s.consumed = true
	s.cv = canvas.NewRasterCanvasCPU(s.sw, s.sh)

	return s.cv, nil
}

// Commit snapshots the canvas pixels into an independently-owned image so the
// published result survives Close.
func (s *imageSurface) Commit(_ context.Context) error {
	if s.cv == nil {
		return output.ErrSurfaceConsumed
	}

	s.consumed = true

	src := s.cv.Image()
	b := src.Bounds()
	dst := stdimage.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	s.img = dst

	return nil
}

// Bounds is the scaled (device) drawing size.
func (s *imageSurface) Bounds() stdimage.Rectangle { return stdimage.Rect(0, 0, s.sw, s.sh) }

func (s *imageSurface) Close() error {
	if s.cv != nil {
		if err := s.cv.Close(); err != nil {
			return fmt.Errorf("image: close canvas: %w", err)
		}
	}

	return nil
}

// Image returns the rendered frame published by the last Commit, or nil if no
// frame has been committed. The façade [ggplot.Plot.Image] reads it.
func (s *imageSurface) Image() stdimage.Image { return s.img }
