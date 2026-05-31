package output

import (
	"context"
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/TuSKan/ggplot/canvas"
)

// Action is the decision a [Controller] returns for an [Event]: what the
// [Session] should do next.
type Action uint8

const (
	// ActionIgnore does nothing.
	ActionIgnore Action = iota
	// ActionRedraw is the fast path: re-render the current Figure under the
	// (possibly updated) viewport transform in [State]. GPU-bound, no rebuild.
	ActionRedraw
	// ActionRebuild is the slow path: call Source.Build again because the data
	// extent changed (viewport crossed the trained bounds, or a brush changed a
	// stat input). Scales retrain and stats recompute.
	ActionRebuild
	// ActionExport renders the current frame to the session's export surface,
	// if one was configured via [WithExportSurface].
	ActionExport
	// ActionClose ends the session.
	ActionClose
)

// State is the mutable interaction state shared between the [Session] and its
// [Controller]. The controller reads events and mutates the viewport; the
// session reads the viewport to render.
type State struct {
	// Bounds is the surface's current logical drawing size, refreshed before
	// each event is dispatched.
	Bounds image.Rectangle

	// OffsetX, OffsetY, and Scale define the viewport affine transform applied
	// on the fast path: device = offset + scale * figure. Scale 0 is treated
	// as 1.
	OffsetX float64
	OffsetY float64
	Scale   float64

	// Figure is the current figure being displayed (read-only for controllers
	// that need the trained extent, e.g. to decide redraw vs. rebuild).
	Figure Figure
}

// Controller decides, per event, what the [Session] does next. The default
// controller provides pan (pointer drag) and wheel-zoom; swap it via
// [WithController] for linked brushing, custom gestures, etc.
type Controller interface {
	OnEvent(ev Event, st *State) Action
}

// ControllerFunc adapts a plain function to the [Controller] interface.
type ControllerFunc func(ev Event, st *State) Action

// OnEvent calls f.
func (f ControllerFunc) OnEvent(ev Event, st *State) Action { return f(ev, st) }

// Session drives a [Source] onto a [LiveSurface]: build once, draw, then
// re-render on events. It owns the fast-path / slow-path policy.
//
// The fast path ([ActionRedraw]) re-renders the current figure under a viewport
// affine transform — cheap and frequent. The slow path ([ActionRebuild]) calls
// Source.Build again when the data extent changes. By default the rebuild is
// synchronous; [WithRebuildDelay] makes it asynchronous and debounced — the last
// good figure keeps drawing while the next one is computed off the event loop.
type Session struct {
	src        Source
	surf       LiveSurface
	controller Controller
	exportFn   func(ctx context.Context) (Surface, error)

	state State
	cur   Figure

	// Async rebuild (active only when rebuildDelay > 0). These fields are
	// touched solely on the Run goroutine; the background build goroutine only
	// reads the immutable src and sends its result on the results channel.
	rebuildDelay time.Duration
	onRebuildErr func(error)
	results      chan rebuildResult
	timer        *time.Timer
	building     bool
	dirty        bool
	buildCancel  context.CancelFunc
}

// rebuildResult carries the outcome of a background [Source.Build].
type rebuildResult struct {
	fig Figure
	err error
}

// SessionOpt configures a [Session].
type SessionOpt func(*Session)

// WithController overrides the default pan/zoom controller.
func WithController(c Controller) SessionOpt {
	return func(s *Session) {
		if c != nil {
			s.controller = c
		}
	}
}

// WithExportSurface sets the factory used to create a destination surface when
// a controller returns [ActionExport].
func WithExportSurface(fn func(ctx context.Context) (Surface, error)) SessionOpt {
	return func(s *Session) { s.exportFn = fn }
}

// WithRebuildDelay enables asynchronous, debounced rebuilds. When d > 0, an
// [ActionRebuild] no longer blocks the event loop: rapid triggers within d are
// coalesced into a single background [Source.Build], the last good figure stays
// on screen until it completes, and the result is swapped in when ready. When
// d <= 0 (the default) rebuilds run synchronously on the event loop.
//
// Pending async rebuilds are flushed when the event channel closes, but are
// dropped on [ActionClose]/[EventClose] or context cancellation (the session is
// shutting down).
func WithRebuildDelay(d time.Duration) SessionOpt {
	return func(s *Session) { s.rebuildDelay = d }
}

// WithRebuildError sets a handler for errors from asynchronous rebuilds. Such
// errors are non-fatal: the last good figure is kept and the session continues.
// Without a handler they are dropped. (Synchronous rebuilds instead return the
// error from [Session.Run].)
func WithRebuildError(fn func(error)) SessionOpt {
	return func(s *Session) { s.onRebuildErr = fn }
}

// NewSession creates a session that drives src onto surf. With no options it
// uses the default pan/zoom controller.
func NewSession(src Source, surf LiveSurface, opts ...SessionOpt) *Session {
	s := &Session{
		src:        src,
		surf:       surf,
		controller: &defaultController{},
		state:      State{Scale: 1},
		results:    make(chan rebuildResult, 1),
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

// Run builds the initial figure, draws one frame, then processes events until
// the surface's event channel closes, a controller returns [ActionClose], an
// [EventClose] arrives, or ctx is cancelled.
func (s *Session) Run(ctx context.Context) error {
	fig, err := s.src.Build(ctx)
	if err != nil {
		return fmt.Errorf("session: initial build: %w", err)
	}

	s.cur = fig

	if err := s.render(ctx); err != nil {
		return err
	}

	defer s.stopAsync()

	events := s.surf.Events()

	for {
		// Settled: events drained and no rebuild pending or in flight. Closing
		// the event channel sets events to nil so we keep servicing the debounce
		// timer and any in-flight build (flushing the last scheduled rebuild)
		// before returning.
		if events == nil && s.timer == nil && !s.building {
			return nil
		}

		var timerC <-chan time.Time
		if s.timer != nil {
			timerC = s.timer.C
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("session: %w", ctx.Err())
		case ev, ok := <-events:
			if !ok {
				events = nil

				continue
			}

			done, err := s.dispatch(ctx, ev)
			if err != nil {
				return err
			}

			if done {
				return nil
			}
		case <-timerC:
			s.timer = nil
			s.startBuild(ctx)
		case res := <-s.results:
			if err := s.finishBuild(ctx, res); err != nil {
				return err
			}
		}
	}
}

// stopAsync cancels any in-flight background build and stops the debounce timer.
// Called on Run exit so a shutting-down session leaves no goroutine or timer.
func (s *Session) stopAsync() {
	if s.buildCancel != nil {
		s.buildCancel()
		s.buildCancel = nil
	}

	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}

// dispatch routes one event through the controller and performs the resulting
// action. It returns done=true when the session should stop.
func (s *Session) dispatch(ctx context.Context, ev Event) (bool, error) {
	s.state.Bounds = s.surf.Bounds()
	s.state.Figure = s.cur

	switch s.controller.OnEvent(ev, &s.state) {
	case ActionIgnore:
	case ActionRedraw:
		if err := s.render(ctx); err != nil {
			return false, err
		}
	case ActionRebuild:
		if s.rebuildDelay <= 0 {
			if err := s.rebuild(ctx); err != nil {
				return false, err
			}
		} else {
			s.scheduleRebuild()
		}
	case ActionExport:
		if err := s.export(ctx); err != nil {
			return false, err
		}
	case ActionClose:
		return true, nil
	}

	// A close event always ends the session, regardless of the controller.
	return ev.Kind == EventClose, nil
}

func (s *Session) render(ctx context.Context) error {
	v := &viewportFigure{
		fig:     s.cur,
		offsetX: s.state.OffsetX,
		offsetY: s.state.OffsetY,
		scale:   s.state.Scale,
	}

	if err := Render(ctx, v, s.surf); err != nil {
		return fmt.Errorf("session: render: %w", err)
	}

	return nil
}

func (s *Session) rebuild(ctx context.Context) error {
	fig, err := s.src.Build(ctx)
	if err != nil {
		return fmt.Errorf("session: rebuild: %w", err)
	}

	// The rebuilt figure reflects the new data extent, so the viewport
	// transform resets to identity.
	s.cur = fig
	s.state.OffsetX, s.state.OffsetY, s.state.Scale = 0, 0, 1

	return s.render(ctx)
}

// scheduleRebuild arms (or resets) the debounce timer for an async rebuild.
// Repeated calls within the delay window coalesce into a single build.
func (s *Session) scheduleRebuild() {
	if s.timer == nil {
		s.timer = time.NewTimer(s.rebuildDelay)

		return
	}

	if !s.timer.Stop() {
		select {
		case <-s.timer.C:
		default:
		}
	}

	s.timer.Reset(s.rebuildDelay)
}

// startBuild launches a background [Source.Build]. If one is already running the
// request is coalesced (replayed once the current build finishes) rather than
// starting a second concurrent build.
func (s *Session) startBuild(ctx context.Context) {
	if s.building {
		s.dirty = true

		return
	}

	s.building = true

	bctx, cancel := context.WithCancel(ctx)
	s.buildCancel = cancel

	go func() {
		fig, err := s.src.Build(bctx)
		s.results <- rebuildResult{fig: fig, err: err}
	}()
}

// finishBuild applies a completed background build. On success it swaps in the
// new figure, resets the viewport, and re-renders; on error it keeps the last
// good figure and notifies the optional handler. A request that arrived while
// the build was in flight re-arms the debounce timer.
func (s *Session) finishBuild(ctx context.Context, res rebuildResult) error {
	s.building = false

	if s.buildCancel != nil {
		s.buildCancel()
		s.buildCancel = nil
	}

	switch {
	case res.err != nil:
		if s.onRebuildErr != nil && !errors.Is(res.err, context.Canceled) {
			s.onRebuildErr(fmt.Errorf("session: rebuild: %w", res.err))
		}
	default:
		s.cur = res.fig
		s.state.OffsetX, s.state.OffsetY, s.state.Scale = 0, 0, 1
		s.state.Figure = res.fig

		if err := s.render(ctx); err != nil {
			return err
		}
	}

	if s.dirty {
		s.dirty = false
		s.scheduleRebuild()
	}

	return nil
}

func (s *Session) export(ctx context.Context) error {
	if s.exportFn == nil {
		return nil
	}

	surf, err := s.exportFn(ctx)
	if err != nil {
		return fmt.Errorf("session: export surface: %w", err)
	}

	defer func() { _ = surf.Close() }()

	if err := Render(ctx, s.cur, surf); err != nil {
		return fmt.Errorf("session: export render: %w", err)
	}

	return nil
}

// viewportFigure wraps a Figure with a viewport affine transform applied on the
// fast path. It implements [Figure].
type viewportFigure struct {
	fig              Figure
	offsetX, offsetY float64
	scale            float64
}

func (v *viewportFigure) Draw(ctx context.Context, dst canvas.Canvas, width, height int) error {
	return drawViewport(ctx, v.fig, dst, width, height, v.offsetX, v.offsetY, v.scale)
}

// drawViewport draws fig onto dst under the given viewport affine transform
// (device = offset + scale * figure). It is the shared fast-path primitive.
func drawViewport(ctx context.Context, fig Figure, dst canvas.Canvas, width, height int, offsetX, offsetY, scale float64) error {
	if scale == 0 {
		scale = 1
	}

	if offsetX != 0 || offsetY != 0 || scale != 1 {
		dst.Save()
		dst.Translate(offsetX, offsetY)
		dst.ScaleXY(scale, scale)

		defer dst.Restore()
	}

	if err := fig.Draw(ctx, dst, width, height); err != nil {
		return fmt.Errorf("output: viewport draw: %w", err)
	}

	return nil
}

// DrawViewport draws fig onto dst under the viewport affine transform held in
// st. It is the shared fast-path primitive used by [Session] and by live
// surface backends (e.g. output/window) that own their own frame loop.
func DrawViewport(ctx context.Context, fig Figure, dst canvas.Canvas, width, height int, st *State) error {
	return drawViewport(ctx, fig, dst, width, height, st.OffsetX, st.OffsetY, st.Scale)
}

// DefaultController returns a fresh pan + wheel-zoom controller — the one
// [NewSession] uses when no controller is supplied. Live backends reuse it so
// interaction behaves identically across the window and browser targets.
func DefaultController() Controller { return &defaultController{} }

// zoomStep is the per-scroll-notch zoom fraction for the default controller.
const zoomStep = 0.1

// defaultController provides pan (pointer drag) and wheel-zoom around the
// cursor. Hovering and key presses are ignored.
type defaultController struct {
	dragging     bool
	lastX, lastY float64
}

var _ Controller = (*defaultController)(nil)

func (c *defaultController) OnEvent(ev Event, st *State) Action {
	switch ev.Kind {
	case EventPointerDown:
		c.dragging = true
		c.lastX, c.lastY = ev.X, ev.Y

		return ActionIgnore
	case EventPointerUp:
		c.dragging = false

		return ActionIgnore
	case EventPointerMove:
		if !c.dragging {
			return ActionIgnore
		}

		st.OffsetX += ev.X - c.lastX
		st.OffsetY += ev.Y - c.lastY
		c.lastX, c.lastY = ev.X, ev.Y

		return ActionRedraw
	case EventScroll:
		c.zoom(ev, st)

		return ActionRedraw
	case EventResize:
		return ActionRedraw
	case EventKey:
		return ActionIgnore
	case EventClose:
		return ActionClose
	}

	return ActionIgnore
}

// zoom adjusts the viewport scale around the cursor so the world point under
// the cursor stays fixed.
func (c *defaultController) zoom(ev Event, st *State) {
	f := 1 + zoomStep
	if ev.DY < 0 {
		f = 1 / (1 + zoomStep)
	}

	if st.Scale == 0 {
		st.Scale = 1
	}

	st.OffsetX = ev.X*(1-f) + f*st.OffsetX
	st.OffsetY = ev.Y*(1-f) + f*st.OffsetY
	st.Scale *= f
}
