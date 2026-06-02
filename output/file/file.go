// Package file provides a [output.Surface] that encodes a single frame to disk
// (or any io.Writer) as PNG, SVG, or PDF.
//
// Convenience functions [Save] and [Encode] build a [output.Source] and render
// it in one call. For lower-level control, blank-import this package and use
// [output.NewSurface]("file", ...) + [output.Render].
package file

import (
	"context"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/output"
)

//nolint:gochecknoinits // Blank-import surface registration (the idiomatic Go driver pattern, cf. image/png).
func init() { output.Register("file", newFileSurface) }

// fileSurface is a single-shot [output.Surface]: it accepts exactly one
// Acquire/Commit cycle and encodes that frame to its destination.
type fileSurface struct {
	opt      output.SurfaceOptions
	format   string // resolved encoder: "png", "svg", or "pdf"
	sw, sh   int    // scaled (device) pixel dimensions
	cv       canvas.Canvas
	consumed bool
	written  int64
}

func newFileSurface(_ context.Context, opt output.SurfaceOptions) (output.Surface, error) {
	format := strings.ToLower(opt.Format)
	if format == "" && opt.Path != "" {
		format = strings.TrimPrefix(strings.ToLower(filepath.Ext(opt.Path)), ".")
	}

	if format == "" {
		format = "png"
	}

	switch format {
	case "png", "svg", "pdf":
	default:
		return nil, fmt.Errorf("%w: %q", output.ErrUnsupportedFormat, format)
	}

	scale := opt.Scale
	if scale <= 0 {
		scale = 1
	}

	return &fileSurface{
		opt:    opt,
		format: format,
		sw:     int(float64(opt.Width) * scale),
		sh:     int(float64(opt.Height) * scale),
	}, nil
}

// Acquire returns a canvas for the single frame: a RecordingCanvas for vector
// formats (svg/pdf), a RasterCanvas for png.
func (s *fileSurface) Acquire(_ context.Context) (canvas.Canvas, error) {
	if s.consumed {
		return nil, output.ErrSurfaceConsumed
	}

	s.consumed = true

	switch s.format {
	case "svg", "pdf":
		s.cv = canvas.NewRecordingCanvas(s.sw, s.sh)
	default: // png
		if s.opt.CPU {
			s.cv = canvas.NewRasterCanvasCPU(s.sw, s.sh)
		} else {
			s.cv = canvas.NewRasterCanvas(s.sw, s.sh)
		}
	}

	return s.cv, nil
}

// Commit encodes the acquired frame and flushes it to the destination.
func (s *fileSurface) Commit(_ context.Context) error {
	if s.cv == nil {
		return output.ErrSurfaceConsumed
	}

	s.consumed = true

	dst, closeFn, err := s.destination()
	if err != nil {
		return err
	}

	cw := &countWriter{w: dst}

	encErr := s.encode(cw)

	s.written = cw.n

	closeErr := closeFn()

	if encErr != nil {
		return encErr
	}

	return closeErr
}

func (s *fileSurface) encode(w io.Writer) error {
	switch s.format {
	case "svg", "pdf":
		rec, ok := s.cv.(*canvas.RecordingCanvas)
		if !ok {
			return fmt.Errorf("file: %w: expected RecordingCanvas", output.ErrUnsupportedFormat)
		}

		var (
			n   int64
			err error
		)

		if s.format == "svg" {
			n, err = canvas.ExportSVG(rec.FinishRecording(), w)
		} else {
			n, err = canvas.ExportPDF(rec.FinishRecording(), w)
		}

		_ = n

		if err != nil {
			return fmt.Errorf("file: export %s: %w", s.format, err)
		}

		return nil
	default: // png
		raster, ok := s.cv.(*canvas.RasterCanvas)
		if !ok {
			return fmt.Errorf("file: %w: expected RasterCanvas", output.ErrUnsupportedFormat)
		}

		if err := raster.EncodePNG(w); err != nil {
			return fmt.Errorf("file: encode png: %w", err)
		}

		return nil
	}
}

// destination resolves where encoded bytes go: an explicit Writer if set,
// otherwise a newly created file at Path. closeFn closes the file and returns
// any error (no-op for a borrowed Writer).
func (s *fileSurface) destination() (io.Writer, func() error, error) {
	if s.opt.Writer != nil {
		return s.opt.Writer, func() error { return nil }, nil
	}

	if s.opt.Path == "" {
		return nil, nil, fmt.Errorf("file: %w: no path or writer", output.ErrUnsupportedFormat)
	}

	f, err := os.Create(s.opt.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("file: create %s: %w", s.opt.Path, err)
	}

	return f, func() error {
		if cerr := f.Close(); cerr != nil {
			return fmt.Errorf("file: close %s: %w", s.opt.Path, cerr)
		}

		return nil
	}, nil
}

// Bounds is the scaled (device) drawing size; Render draws at these dimensions.
func (s *fileSurface) Bounds() image.Rectangle { return image.Rect(0, 0, s.sw, s.sh) }

// Close releases the raster canvas, if any. RecordingCanvas needs no cleanup.
func (s *fileSurface) Close() error {
	if raster, ok := s.cv.(*canvas.RasterCanvas); ok {
		if err := raster.Close(); err != nil {
			return fmt.Errorf("file: close canvas: %w", err)
		}
	}

	return nil
}

// BytesWritten reports the number of bytes encoded by the last Commit.
func (s *fileSurface) BytesWritten() int64 { return s.written }

// ---------------------------------------------------------------------------
// Public convenience functions
// ---------------------------------------------------------------------------

// Save builds src and renders to a file at the given dimensions. The output
// format is inferred from the file extension (.png, .svg, .pdf). If height ≤ 0
// and the built figure implements [output.Sizer], it is inferred from width.
//
// Options: [output.WithScale] for HiDPI output, [output.WithCPU] to force the
// CPU rasterizer.
func Save(ctx context.Context, src output.Source, filename string, width, height int, opts ...output.SurfaceOpt) error {
	fig, h, err := buildAndResolve(ctx, src, width, height)
	if err != nil {
		return err
	}

	sopts := append([]output.SurfaceOpt{
		output.WithPath(filename),
		output.WithSize(width, h),
	}, opts...)

	surf, err := newFileSurface(ctx, output.BuildOptions(sopts...))
	if err != nil {
		return fmt.Errorf("file.Save: %w", err)
	}
	defer func() { _ = surf.Close() }()

	if err := output.Render(ctx, fig, surf); err != nil {
		return fmt.Errorf("file.Save: %w", err)
	}

	return nil
}

// Encode builds src and writes encoded bytes to dst in the given format
// ("png" (default), "svg", "pdf"). If height ≤ 0 and the built figure
// implements [output.Sizer], it is inferred from width. Returns the number of
// bytes written.
func Encode(ctx context.Context, src output.Source, dst io.Writer, format string, width, height int, opts ...output.SurfaceOpt) (int64, error) {
	fig, h, err := buildAndResolve(ctx, src, width, height)
	if err != nil {
		return 0, err
	}

	sopts := append([]output.SurfaceOpt{
		output.WithWriter(dst),
		output.WithFormat(format),
		output.WithSize(width, h),
	}, opts...)

	surf, err := newFileSurface(ctx, output.BuildOptions(sopts...))
	if err != nil {
		return 0, fmt.Errorf("file.Encode: %w", err)
	}
	defer func() { _ = surf.Close() }()

	if err := output.Render(ctx, fig, surf); err != nil {
		return 0, fmt.Errorf("file.Encode: %w", err)
	}

	fs := surf.(*fileSurface) //nolint:forcetypeassert,errcheck // we just created it above.

	return fs.BytesWritten(), nil
}

// buildAndResolve builds the source and resolves a non-positive height via
// [output.Sizer]. This is the shared preamble for [Save] and [Encode].
func buildAndResolve(ctx context.Context, src output.Source, width, height int) (output.Figure, int, error) {
	fig, err := src.Build(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("build: %w", err)
	}

	if height <= 0 {
		if sizer, ok := fig.(output.Sizer); ok {
			_, height = sizer.PreferredSize(width)
		}
	}

	if height <= 0 {
		height = width // fallback: square
	}

	return fig, height, nil
}

// countWriter counts bytes written through it.
type countWriter struct {
	w io.Writer
	n int64
}

func (cw *countWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += int64(n)

	if err != nil {
		return n, fmt.Errorf("file: write: %w", err)
	}

	return n, nil
}
