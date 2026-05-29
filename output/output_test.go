package output

import (
	"context"
	"errors"
	"image"
	"testing"

	"github.com/TuSKan/ggplot/canvas"
)

// fakeFigure records whether it was drawn.
type fakeFigure struct{ drawn bool }

func (f *fakeFigure) Draw(_ context.Context, _ canvas.Canvas, _, _ int) error {
	f.drawn = true

	return nil
}

// fakeSurface is a minimal in-memory Surface for exercising Render and the
// registry without a platform backend.
type fakeSurface struct {
	w, h              int
	acquired, commits int
	closed            bool
	cv                *canvas.RasterCanvas
}

func (s *fakeSurface) Acquire(_ context.Context) (canvas.Canvas, error) {
	s.acquired++
	s.cv = canvas.NewRasterCanvasCPU(s.w, s.h)

	return s.cv, nil
}

func (s *fakeSurface) Commit(_ context.Context) error {
	s.commits++

	return nil
}

func (s *fakeSurface) Bounds() image.Rectangle { return image.Rect(0, 0, s.w, s.h) }

func (s *fakeSurface) Close() error {
	s.closed = true

	if s.cv != nil {
		return s.cv.Close() //nolint:wrapcheck // test helper.
	}

	return nil
}

func TestRenderPresentsOneFrame(t *testing.T) {
	t.Parallel()

	surf := &fakeSurface{w: 16, h: 16}
	fig := &fakeFigure{}

	if err := Render(context.Background(), fig, surf); err != nil {
		t.Fatalf("Render: %v", err)
	}

	defer func() { _ = surf.Close() }()

	if !fig.drawn {
		t.Error("figure was not drawn")
	}

	if surf.acquired != 1 || surf.commits != 1 {
		t.Errorf("acquired=%d commits=%d, want 1/1", surf.acquired, surf.commits)
	}
}

func TestNewSurfaceUnknown(t *testing.T) {
	t.Parallel()

	if _, err := NewSurface(context.Background(), "does-not-exist"); !errors.Is(err, ErrUnknownSurface) {
		t.Fatalf("want ErrUnknownSurface, got %v", err)
	}
}

func TestRegisterAndNewSurface(t *testing.T) {
	t.Parallel()

	const name = "test-fake"

	Register(name, func(_ context.Context, opt SurfaceOptions) (Surface, error) {
		return &fakeSurface{w: opt.Width, h: opt.Height}, nil
	})

	surf, err := NewSurface(context.Background(), name, WithSize(20, 30))
	if err != nil {
		t.Fatalf("NewSurface: %v", err)
	}

	if b := surf.Bounds(); b.Dx() != 20 || b.Dy() != 30 {
		t.Errorf("bounds=%v, want 20x30", b)
	}
}

func TestNewLiveSurfaceNotLive(t *testing.T) {
	t.Parallel()

	const name = "test-static"

	Register(name, func(_ context.Context, _ SurfaceOptions) (Surface, error) {
		return &fakeSurface{}, nil
	})

	if _, err := NewLiveSurface(context.Background(), name); !errors.Is(err, ErrNotLive) {
		t.Fatalf("want ErrNotLive, got %v", err)
	}
}

func TestResolveOptionsDefaultScale(t *testing.T) {
	t.Parallel()

	if o := resolveOptions(); o.Scale != 1 {
		t.Errorf("default Scale=%v, want 1", o.Scale)
	}
}
