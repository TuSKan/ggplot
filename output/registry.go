package output

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

// Sentinel errors returned by the output layer.
var (
	// ErrUnknownSurface is returned by [NewSurface] when no factory is
	// registered under the requested name — usually because the platform
	// subpackage providing it was not blank-imported.
	ErrUnknownSurface = errors.New("output: unknown surface (blank-import its package to register it)")

	// ErrSurfaceConsumed is returned by a single-shot [Surface] (file, image)
	// when Acquire is called a second time. Such surfaces present exactly one
	// frame.
	ErrSurfaceConsumed = errors.New("output: surface already consumed (single-shot surface allows one frame)")

	// ErrNotLive is returned by [NewLiveSurface] when the named surface is
	// registered but does not implement [LiveSurface] (e.g. "file").
	ErrNotLive = errors.New("output: surface is not interactive")

	// ErrUnsupportedFormat is returned by a surface factory when the requested
	// encoder format is not one it can produce.
	ErrUnsupportedFormat = errors.New("output: unsupported format")

	// ErrNoImage is returned when an image was requested from a surface that
	// does not implement [Imager].
	ErrNoImage = errors.New("output: surface does not produce an image")
)

// SurfaceOptions is the resolved configuration passed to a [SurfaceFactory].
// Fields not relevant to a given surface are ignored by its factory.
type SurfaceOptions struct {
	// Width and Height are the logical drawing size in pixels.
	Width, Height int

	// Path is the destination file path (file surface). The encoder is
	// inferred from its extension unless Format is set.
	Path string

	// Writer is an alternative destination for the file surface: when set,
	// encoded bytes are written here instead of to Path.
	Writer io.Writer

	// Format overrides the encoder ("png", "svg", "pdf"). Empty means infer
	// from Path.
	Format string

	// Scale is the device-pixel multiplier for HiDPI output (1 = none).
	Scale float64

	// CPU forces a pure-CPU rasterizer for raster output (deterministic,
	// headless-safe), bypassing the GPU accelerator.
	CPU bool
}

// SurfaceOpt configures [SurfaceOptions] functionally.
type SurfaceOpt func(*SurfaceOptions)

// WithSize sets the logical drawing size.
func WithSize(w, h int) SurfaceOpt {
	return func(o *SurfaceOptions) { o.Width, o.Height = w, h }
}

// WithPath sets the destination file path (file surface).
func WithPath(path string) SurfaceOpt {
	return func(o *SurfaceOptions) { o.Path = path }
}

// WithFormat overrides the encoder format ("png", "svg", "pdf").
func WithFormat(format string) SurfaceOpt {
	return func(o *SurfaceOptions) { o.Format = format }
}

// WithScale sets the device-pixel multiplier for HiDPI output.
func WithScale(scale float64) SurfaceOpt {
	return func(o *SurfaceOptions) { o.Scale = scale }
}

// WithWriter sets an io.Writer destination for the file surface (used in place
// of a path).
func WithWriter(w io.Writer) SurfaceOpt {
	return func(o *SurfaceOptions) { o.Writer = w }
}

// WithCPU forces a pure-CPU rasterizer for raster output.
func WithCPU(cpu bool) SurfaceOpt {
	return func(o *SurfaceOptions) { o.CPU = cpu }
}

// BuildOptions applies the given options and returns the resolved
// [SurfaceOptions]. Surface packages use this to build options from variadic
// [SurfaceOpt] slices.
func BuildOptions(opts ...SurfaceOpt) SurfaceOptions {
	return resolveOptions(opts...)
}

func resolveOptions(opts ...SurfaceOpt) SurfaceOptions {
	o := SurfaceOptions{Scale: 1}
	for _, opt := range opts {
		opt(&o)
	}

	return o
}

// SurfaceFactory constructs a [Surface] from resolved options. Platform
// subpackages register one via [Register] in their init().
type SurfaceFactory func(ctx context.Context, opt SurfaceOptions) (Surface, error)

var (
	registryMu      sync.RWMutex
	surfaceRegistry = map[string]SurfaceFactory{}
)

// Register associates a [SurfaceFactory] with a name. It is called from a
// platform subpackage's init(); registering the same name twice panics, as it
// indicates two packages claiming the same surface.
func Register(name string, f SurfaceFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, dup := surfaceRegistry[name]; dup {
		panic(fmt.Sprintf("output: surface %q already registered", name))
	}

	surfaceRegistry[name] = f
}

func lookupSurface(name string) (SurfaceFactory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	f, ok := surfaceRegistry[name]

	return f, ok
}

// NewSurface constructs a registered surface by name. It returns
// [ErrUnknownSurface] if the corresponding subpackage was not blank-imported.
func NewSurface(ctx context.Context, name string, opts ...SurfaceOpt) (Surface, error) {
	f, ok := lookupSurface(name)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownSurface, name)
	}

	return f(ctx, resolveOptions(opts...))
}

// NewLiveSurface is [NewSurface] plus a runtime assertion to [LiveSurface]. It
// returns [ErrNotLive] if the named surface is not interactive.
func NewLiveSurface(ctx context.Context, name string, opts ...SurfaceOpt) (LiveSurface, error) {
	surf, err := NewSurface(ctx, name, opts...)
	if err != nil {
		return nil, err
	}

	live, ok := surf.(LiveSurface)
	if !ok {
		_ = surf.Close()

		return nil, fmt.Errorf("%w: %q", ErrNotLive, name)
	}

	return live, nil
}
