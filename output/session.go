package output

import (
	"context"
	"errors"
	"fmt"
	"image"
	"time"
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

	// Figure is the current figure being displayed (read-only for controllers
	// that need the trained extent, e.g. to decide redraw vs. rebuild).
	Figure Figure
}

// Controller decides, per event, what the [Session] does next. The default
// controller is [DataSpaceController] (data-space pan/zoom); swap it via
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
// uses [DataSpaceController] for data-space pan/zoom.
func NewSession(src Source, surf LiveSurface, opts ...SessionOpt) *Session {
	s := &Session{
		src:        src,
		surf:       surf,
		controller: DataSpaceController(),
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
	if err := Render(ctx, s.cur, s.surf); err != nil {
		return fmt.Errorf("session: render: %w", err)
	}

	return nil
}

func (s *Session) rebuild(ctx context.Context) error {
	fig, err := s.src.Build(ctx)
	if err != nil {
		return fmt.Errorf("session: rebuild: %w", err)
	}

	s.cur = fig

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
