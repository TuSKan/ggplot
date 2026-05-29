package image_test

import (
	"context"
	"image/color"
	"testing"

	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/output"
	_ "github.com/TuSKan/ggplot/output/image"
)

// rectFigure draws a filled rectangle onto the canvas.
type rectFigure struct{}

func (rectFigure) Draw(_ context.Context, cv canvas.Canvas, w, h int) error {
	cv.Clear(color.White)
	cv.SetColor(color.RGBA{R: 40, G: 120, B: 200, A: 255})
	cv.DrawRectangle(1, 1, float64(w-2), float64(h-2))
	cv.Fill()

	return nil
}

func TestImageSurfaceProducesImage(t *testing.T) {
	t.Parallel()

	surf, err := output.NewSurface(context.Background(), "image", output.WithSize(50, 25))
	if err != nil {
		t.Fatalf("NewSurface: %v", err)
	}

	defer func() { _ = surf.Close() }()

	if err := output.Render(context.Background(), rectFigure{}, surf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	im, ok := surf.(output.Imager)
	if !ok {
		t.Fatal("image surface is not an Imager")
	}

	got := im.Image()
	if got == nil {
		t.Fatal("nil image")
	}

	if b := got.Bounds(); b.Dx() != 50 || b.Dy() != 25 {
		t.Errorf("bounds=%v, want 50x25", b)
	}
}

func TestImageSurfaceNotLive(t *testing.T) {
	t.Parallel()

	if _, err := output.NewLiveSurface(context.Background(), "image", output.WithSize(10, 10)); err == nil {
		t.Fatal("image surface should not be a LiveSurface")
	}
}
