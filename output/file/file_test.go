package file_test

import (
	"bytes"
	"context"
	"errors"
	"image/color"
	"testing"

	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/output"
	_ "github.com/TuSKan/ggplot/output/file"
)

// rectFigure draws a filled rectangle so encoded output is non-trivial.
type rectFigure struct{}

func (rectFigure) Draw(_ context.Context, cv canvas.Canvas, w, h int) error {
	cv.Clear(color.White)
	cv.SetColor(color.RGBA{R: 200, G: 60, B: 60, A: 255})
	cv.DrawRectangle(1, 1, float64(w-2), float64(h-2))
	cv.Fill()

	return nil
}

func TestFileSurfaceEncodeFormats(t *testing.T) {
	t.Parallel()

	for _, format := range []string{"png", "svg", "pdf"} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			surf, err := output.NewSurface(context.Background(), "file",
				output.WithWriter(&buf),
				output.WithFormat(format),
				output.WithSize(60, 40),
			)
			if err != nil {
				t.Fatalf("NewSurface: %v", err)
			}

			defer func() { _ = surf.Close() }()

			if err := output.Render(context.Background(), rectFigure{}, surf); err != nil {
				t.Fatalf("Render: %v", err)
			}

			if buf.Len() == 0 {
				t.Errorf("%s: no bytes written", format)
			}

			bw, ok := surf.(interface{ BytesWritten() int64 })
			if !ok {
				t.Fatal("file surface does not report BytesWritten")
			}

			if bw.BytesWritten() == 0 {
				t.Errorf("%s: BytesWritten=0", format)
			}
		})
	}
}

func TestFileSurfaceUnsupportedFormat(t *testing.T) {
	t.Parallel()

	_, err := output.NewSurface(context.Background(), "file",
		output.WithWriter(&bytes.Buffer{}),
		output.WithFormat("gif"),
		output.WithSize(10, 10),
	)
	if !errors.Is(err, output.ErrUnsupportedFormat) {
		t.Fatalf("want ErrUnsupportedFormat, got %v", err)
	}
}

func TestFileSurfaceSingleShot(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	surf, err := output.NewSurface(context.Background(), "file",
		output.WithWriter(&buf),
		output.WithFormat("png"),
		output.WithSize(10, 10),
	)
	if err != nil {
		t.Fatalf("NewSurface: %v", err)
	}

	defer func() { _ = surf.Close() }()

	if _, err := surf.Acquire(context.Background()); err != nil {
		t.Fatalf("first Acquire: %v", err)
	}

	if _, err := surf.Acquire(context.Background()); !errors.Is(err, output.ErrSurfaceConsumed) {
		t.Fatalf("second Acquire: want ErrSurfaceConsumed, got %v", err)
	}
}
