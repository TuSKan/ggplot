//go:build js && wasm

package web

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"syscall/js"

	pcanvas "github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/output"
)

// svgSurface implements [output.LiveSurface] for browser SVG rendering.
// Each frame renders via [pcanvas.RecordingCanvas], exports to SVG with
// metadata (tooltips, links, ARIA), and injects the result as innerHTML.
type svgSurface struct {
	container js.Value
	events    chan output.Event
	bounds    image.Rectangle
	rec       *pcanvas.RecordingCanvas
	callbacks []jsCallback
	svgEl     js.Value    // current <svg> element in the DOM (for event cleanup)
	fps       *fpsTracker // toolbar FPS display
	ro        js.Value    // ResizeObserver — registered once, not per Commit
}

// newSVGSurface creates an SVG-mode surface inside container.
func newSVGSurface(container js.Value, w, h int) *svgSurface {
	events := make(chan output.Event, 64) //nolint:mnd // Buffer size matches output/window event queue depth.

	// ResizeObserver on the container — registered ONCE. SVG Commit replaces
	// innerHTML (destroying the <svg> element), so pointer events must be
	// re-registered per Commit, but the container itself persists.
	ro := observeResize(container, events)

	return &svgSurface{
		container: container,
		events:    events,
		bounds:    image.Rect(0, 0, w, h),
		fps:       newFPSTracker(),
		ro:        ro,
	}
}

// Acquire returns a RecordingCanvas for the next frame.
func (s *svgSurface) Acquire(_ context.Context) (pcanvas.Canvas, error) {
	s.fps.Begin()

	w, h := s.bounds.Dx(), s.bounds.Dy()
	s.rec = pcanvas.NewRecordingCanvas(w, h)

	return s.rec, nil
}

// Commit exports the recording to SVG with metadata and injects it into the
// container as innerHTML. Pointer event listeners are re-registered on the new
// <svg> element (the old one was destroyed by innerHTML replacement).
func (s *svgSurface) Commit(_ context.Context) error {
	if s.rec == nil {
		return nil
	}

	rec := s.rec.FinishRecording()
	meta := s.rec.MetadataMap()
	s.rec = nil

	var buf bytes.Buffer
	if _, err := pcanvas.ExportSVGWithMeta(rec, meta, &buf); err != nil {
		return fmt.Errorf("web: SVG export: %w", err)
	}

	// Remove old pointer event listeners before replacing innerHTML.
	releaseCallbacks(s.callbacks)
	s.callbacks = nil

	// Inject SVG into the container.
	s.container.Set("innerHTML", buf.String())

	// Find the <svg> element and register pointer event listeners on it.
	svgEl := s.container.Call("querySelector", "svg")
	if !svgEl.IsNull() && !svgEl.IsUndefined() {
		// Make SVG fill the container and handle pointer events.
		style := svgEl.Get("style")
		style.Set("width", "100%")
		style.Set("height", "100%")
		style.Set("touch-action", "none")
		style.Set("cursor", "grab")

		s.svgEl = svgEl
		// Only pointer events — ResizeObserver is on the container, registered once.
		s.callbacks = registerPointerEvents(svgEl, s.events)
	}

	s.fps.End()

	return nil
}

// Bounds returns the current logical drawing size.
func (s *svgSurface) Bounds() image.Rectangle { return s.bounds }

// Events returns the channel receiving DOM events.
func (s *svgSurface) Events() <-chan output.Event { return s.events }

// Close cleans up DOM event listeners, disconnects the ResizeObserver,
// and releases resources.
func (s *svgSurface) Close() error {
	releaseCallbacks(s.callbacks)
	s.callbacks = nil

	// Disconnect the ResizeObserver (removeEventListener doesn't work for RO).
	if !s.ro.IsUndefined() && !s.ro.IsNull() {
		s.ro.Call("disconnect")
	}

	// Clear the container.
	s.container.Set("innerHTML", "")

	return nil
}

// Compile-time interface check.
var _ output.LiveSurface = (*svgSurface)(nil)

// mountSVG creates an SVG surface and runs a Session event loop.
func mountSVG(ctx context.Context, src output.Source, fig output.Figure, container js.Value, o Options) error {
	surf := newSVGSurface(container, o.Width, o.Height)
	defer func() { _ = surf.Close() }()

	// Initial render.
	if err := output.Render(ctx, fig, surf); err != nil {
		return fmt.Errorf("web: initial SVG render: %w", err)
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
