// Package output is the back of the rendering pipe: it carries a [Figure]
// (a built plot) to a destination [Surface] — a file, an in-memory image, a
// desktop GPU window, or a browser canvas — through one frame model.
//
// The drawing seam lives in package canvas ([canvas.Canvas] and its
// RasterCanvas / RecordingCanvas backends). This package sits on top: a
// [Figure] paints itself onto a [canvas.Canvas], and a [Surface] decides where
// those pixels or vectors go. [Render] presents exactly one frame and is the
// entire static story (PNG/SVG/PDF/in-memory image); interactive surfaces add
// an event loop on top (see Session).
//
// Concrete surfaces live in subpackages (output/file, output/image,
// output/window, output/web) and register themselves via [Register] in their
// init(). Platform selection is therefore a blank import — no build tag ever
// appears in user code. This core package imports none of them.
package output

import (
	"context"
	"fmt"
	"image"

	"github.com/TuSKan/ggplot/canvas"
)

// Figure is something that can paint itself onto a [canvas.Canvas] at a given
// size. *ggplot.Built satisfies Figure with no change — its Draw method already
// has exactly this signature.
type Figure interface {
	Draw(ctx context.Context, dst canvas.Canvas, width, height int) error
}

// Source yields a fresh [Figure]. *ggplot.Plot is a Source. Live surfaces hold
// a Source so they can rebuild when an interaction crosses the trained data
// extent (scales retrain, stats recompute — the slow path).
type Source interface {
	Build(ctx context.Context) (Figure, error)
}

// Sizer is an optional [Figure] extension: given a width, it proposes a
// preferred size. Façades (Save/Encode/Image) use it to infer a height when the
// caller passes a non-positive one. *ggplot.Built implements Sizer by wrapping
// its existing auto-height rules.
type Sizer interface {
	PreferredSize(width int) (w, h int)
}

// Surface is a destination a [Figure] is drawn to — one interface for file,
// in-memory image, desktop GPU window, and browser canvas.
//
// Static surfaces (file, image) accept exactly one Acquire/Commit cycle and
// return [ErrSurfaceConsumed] on a second Acquire. Live surfaces accept
// repeated cycles. The single- vs. multi-shot contract is documented per
// implementation; the type is shared.
type Surface interface {
	// Acquire returns the drawing Canvas for the next frame, sized to Bounds.
	// The returned Canvas is valid only until the next Commit.
	Acquire(ctx context.Context) (canvas.Canvas, error)

	// Commit finalizes the frame acquired by Acquire:
	//   file   -> encode + flush bytes to disk
	//   image  -> publish the in-memory image
	//   window -> submit GPU work + present (zero-copy)
	//   web    -> submit + swapchain present
	Commit(ctx context.Context) error

	// Bounds is the current logical drawing size (device scale handled
	// internally). Fixed for file/image; tracks the window/canvas for live.
	Bounds() image.Rectangle

	Close() error
}

// LiveSurface is a [Surface] with an interactive lifecycle. Desktop windows and
// browser canvases implement it; file and image surfaces do not. Interaction is
// deliberately absent from the Surface contract — a PNG export never links an
// event loop.
type LiveSurface interface {
	Surface
	Events() <-chan Event
}

// Imager is implemented by surfaces that publish an in-memory image (the
// "image" surface). Image returns the frame published by the last Commit.
type Imager interface {
	Image() image.Image
}

// Render draws a [Figure] onto a [Surface] and presents exactly one frame. This
// is the entire static story — PNG, SVG, PDF, in-memory image — and is also the
// per-frame primitive the interactive Session loop calls for live surfaces.
func Render(ctx context.Context, fig Figure, surf Surface) error {
	cv, err := surf.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("output: acquire surface: %w", err)
	}

	b := surf.Bounds()
	if err := fig.Draw(ctx, cv, b.Dx(), b.Dy()); err != nil {
		return fmt.Errorf("output: draw figure: %w", err)
	}

	if err := surf.Commit(ctx); err != nil {
		return fmt.Errorf("output: commit surface: %w", err)
	}

	return nil
}
