//go:build js && wasm

package web

import (
	"context"
	"fmt"
	"syscall/js"
	"time"

	"github.com/TuSKan/ggplot/output"
)

const (
	defaultWidth  = 800
	defaultHeight = 600
)

// Options configures the browser surface opened by [Mount].
type Options struct {
	Width        int
	Height       int
	Controller   output.Controller
	RebuildDelay time.Duration // 0 = synchronous rebuild (default)
	OnRebuildErr func(error)   // called when an async rebuild fails (nil = silent)
	SVG          bool          // use SVG rendering instead of raster
	GPU          bool          // use WebGPU-accelerated rendering via gogpu + ggcanvas
}

// Opt functionally configures [Options].
type Opt func(*Options)

// WithSize sets the initial canvas size in logical pixels.
// When zero, the container element's clientWidth/clientHeight is used.
func WithSize(w, h int) Opt { return func(o *Options) { o.Width, o.Height = w, h } }

// WithController overrides the default data-space pan/zoom controller.
func WithController(c output.Controller) Opt {
	return func(o *Options) {
		if c != nil {
			o.Controller = c
		}
	}
}

// WithRebuildDelay enables async, debounced rebuilds. When set, rapid rebuild
// requests (e.g. repeated zoom-past-extent) are coalesced: the rebuild fires
// only after delay elapses with no new request. The last good figure keeps
// drawing until the background build completes. Zero (default) means synchronous.
func WithRebuildDelay(d time.Duration) Opt {
	return func(o *Options) { o.RebuildDelay = d }
}

// WithRebuildError sets a handler invoked when an async rebuild fails. The last
// good figure is kept on screen; the error is non-fatal.
func WithRebuildError(fn func(error)) Opt {
	return func(o *Options) { o.OnRebuildErr = fn }
}

// WithSVG enables SVG rendering mode. The figure is rendered to SVG via
// [canvas.RecordingCanvas] and injected as innerHTML, preserving metadata
// channels (tooltips, links, ARIA labels) as native DOM attributes. Interaction
// events are registered on the <svg> element.
//
// SVG mode re-renders the entire SVG tree on each rebuild, which is acceptable
// for typical plot sizes. It does not use a requestAnimationFrame loop for
// static views.
func WithSVG() Opt { return func(o *Options) { o.SVG = true } }

// Mount builds a ggplot figure and presents it in a browser container element.
// It creates a <canvas> (raster) or injects SVG (vector) inside the container,
// handles pan/zoom interaction via DOM events, and blocks until ctx is cancelled
// or an unrecoverable error occurs.
//
// containerID is the DOM id of the target element (typically a <div>).
// The element must exist when Mount is called.
//
// Mount integrates with [output.Session] for the full controller pipeline:
// fast-path redraw (re-render under viewport transform), slow-path rebuild
// (Source.Build when data extent changes), and debounced async rebuilds.
func Mount(ctx context.Context, src output.Source, containerID string, opts ...Opt) error {
	o := Options{
		Controller: output.DataSpaceController(),
	}
	for _, opt := range opts {
		opt(&o)
	}

	// Resolve the container element.
	doc := js.Global().Get("document")
	container := doc.Call("getElementById", containerID)
	if container.IsNull() || container.IsUndefined() {
		return fmt.Errorf("web: container element %q not found", containerID)
	}

	// Infer size from container if not specified.
	if o.Width <= 0 {
		o.Width = container.Get("clientWidth").Int()
	}
	if o.Height <= 0 {
		o.Height = container.Get("clientHeight").Int()
	}
	if o.Width <= 0 {
		o.Width = defaultWidth
	}
	if o.Height <= 0 {
		o.Height = defaultHeight
	}

	// Build the initial figure.
	fig, err := src.Build(ctx)
	if err != nil {
		return fmt.Errorf("web: initial build: %w", err)
	}

	// Infer height from Sizer if the figure supports it.
	if o.Height <= 0 {
		if sizer, ok := fig.(output.Sizer); ok {
			_, o.Height = sizer.PreferredSize(o.Width)
		}
	}

	if o.GPU {
		return mountGPU(ctx, src, fig, container, o)
	}

	if o.SVG {
		return mountSVG(ctx, src, fig, container, o)
	}

	return mountRaster(ctx, src, fig, container, o)
}

// mountRaster creates a <canvas> element, sets up the raster LiveSurface,
// and runs a Session event loop driven by requestAnimationFrame.
func mountRaster(ctx context.Context, src output.Source, fig output.Figure, container js.Value, o Options) error {
	surf, err := newRasterSurface(container, o.Width, o.Height)
	if err != nil {
		return fmt.Errorf("web: create raster surface: %w", err)
	}
	defer func() { _ = surf.Close() }()

	// Initial render.
	if err := output.Render(ctx, fig, surf); err != nil {
		return fmt.Errorf("web: initial render: %w", err)
	}

	// Create and run a Session for interactive event handling.
	sessOpts := []output.SessionOpt{
		output.WithController(o.Controller),
	}
	if o.RebuildDelay > 0 {
		sessOpts = append(sessOpts, output.WithRebuildDelay(o.RebuildDelay))
	}
	if o.OnRebuildErr != nil {
		sessOpts = append(sessOpts, output.WithRebuildError(o.OnRebuildErr))
	}

	sess := output.NewSession(src, surf, sessOpts...)
	return sess.Run(ctx)
}
