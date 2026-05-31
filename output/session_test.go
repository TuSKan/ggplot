package output_test

import (
	"context"
	"errors"
	"image"
	"sync"
	"testing"
	"time"

	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/output"
)

// nopFigure draws nothing; the session tests assert on frame/build counts, not
// pixels.
type nopFigure struct{}

func (nopFigure) Draw(_ context.Context, _ canvas.Canvas, _, _ int) error { return nil }

// countingSource counts how many times Build is called.
type countingSource struct {
	mu     sync.Mutex
	builds int
}

func (s *countingSource) Build(_ context.Context) (output.Figure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.builds++

	return nopFigure{}, nil
}

func (s *countingSource) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.builds
}

// fakeLive is a scripted LiveSurface: tests push events onto its channel.
type fakeLive struct {
	events chan output.Event
	bounds image.Rectangle

	mu      sync.Mutex
	commits int
	cv      *canvas.RasterCanvas
}

func newFakeLive(w, h int) *fakeLive {
	return &fakeLive{
		events: make(chan output.Event, 16),
		bounds: image.Rect(0, 0, w, h),
	}
}

func (f *fakeLive) Acquire(_ context.Context) (canvas.Canvas, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.cv != nil {
		_ = f.cv.Close()
	}

	f.cv = canvas.NewRasterCanvasCPU(f.bounds.Dx(), f.bounds.Dy())

	return f.cv, nil
}

func (f *fakeLive) Commit(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.commits++

	return nil
}

func (f *fakeLive) Bounds() image.Rectangle { return f.bounds }

func (f *fakeLive) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.cv != nil {
		return f.cv.Close() //nolint:wrapcheck // test helper.
	}

	return nil
}

func (f *fakeLive) Events() <-chan output.Event { return f.events }

func (f *fakeLive) commitCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.commits
}

// waitRun blocks until Run returns (or times out) and checks the error.
func waitRun(t *testing.T, done <-chan error, wantErr error) {
	t.Helper()

	select {
	case err := <-done:
		switch {
		case wantErr == nil && err != nil:
			t.Fatalf("Run returned error: %v", err)
		case wantErr != nil && !errors.Is(err, wantErr):
			t.Fatalf("Run error = %v, want %v", err, wantErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return in time")
	}
}

// runEvents starts the session, pushes events in order, and asserts Run returns
// nil. The caller should include a closing event (EventClose) as the last one.
func runEvents(t *testing.T, sess *output.Session, surf *fakeLive, events ...output.Event) {
	t.Helper()

	done := make(chan error, 1)
	go func() { done <- sess.Run(context.Background()) }()

	for _, ev := range events {
		surf.events <- ev
	}

	waitRun(t, done, nil)
}

func TestSessionInitialDrawAndClose(t *testing.T) {
	t.Parallel()

	src := &countingSource{}
	surf := newFakeLive(100, 80)
	sess := output.NewSession(src, surf)

	runEvents(t, sess, surf, output.Event{Kind: output.EventClose})

	if src.count() != 1 {
		t.Errorf("builds=%d, want 1", src.count())
	}

	if surf.commitCount() < 1 {
		t.Errorf("commits=%d, want >=1 (initial frame)", surf.commitCount())
	}
}

func TestSessionFastPathRedraw(t *testing.T) {
	t.Parallel()

	src := &countingSource{}
	surf := newFakeLive(100, 80)
	sess := output.NewSession(src, surf)

	// A pointer drag is a fast-path redraw — no rebuild.
	runEvents(t, sess, surf,
		output.Event{Kind: output.EventPointerDown, X: 10, Y: 10},
		output.Event{Kind: output.EventPointerMove, X: 25, Y: 18},
		output.Event{Kind: output.EventClose},
	)

	if src.count() != 1 {
		t.Errorf("builds=%d, want 1 (fast path must not rebuild)", src.count())
	}

	if surf.commitCount() < 2 {
		t.Errorf("commits=%d, want >=2 (initial + redraw)", surf.commitCount())
	}
}

func TestSessionSlowPathRebuild(t *testing.T) {
	t.Parallel()

	src := &countingSource{}
	surf := newFakeLive(100, 80)

	// A controller that rebuilds on a key press and closes on a close event.
	ctrl := output.ControllerFunc(func(ev output.Event, _ *output.State) output.Action {
		if ev.Kind == output.EventKey {
			return output.ActionRebuild
		}

		if ev.Kind == output.EventClose {
			return output.ActionClose
		}

		return output.ActionIgnore
	})

	sess := output.NewSession(src, surf, output.WithController(ctrl))

	runEvents(t, sess, surf,
		output.Event{Kind: output.EventKey, Key: "r"},
		output.Event{Kind: output.EventClose},
	)

	if src.count() != 2 {
		t.Errorf("builds=%d, want 2 (initial + rebuild)", src.count())
	}
}

func TestSessionExport(t *testing.T) {
	t.Parallel()

	src := &countingSource{}
	surf := newFakeLive(60, 40)
	export := newFakeLive(60, 40) // reuse fakeLive as an export target

	ctrl := output.ControllerFunc(func(ev output.Event, _ *output.State) output.Action {
		if ev.Kind == output.EventKey {
			return output.ActionExport
		}

		if ev.Kind == output.EventClose {
			return output.ActionClose
		}

		return output.ActionIgnore
	})

	sess := output.NewSession(src, surf, output.WithController(ctrl),
		output.WithExportSurface(func(_ context.Context) (output.Surface, error) {
			return export, nil
		}),
	)

	runEvents(t, sess, surf,
		output.Event{Kind: output.EventKey},
		output.Event{Kind: output.EventClose},
	)

	if export.commitCount() < 1 {
		t.Errorf("export commits=%d, want >=1", export.commitCount())
	}
}

func TestSessionContextCancel(t *testing.T) {
	t.Parallel()

	src := &countingSource{}
	surf := newFakeLive(50, 50)
	sess := output.NewSession(src, surf)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- sess.Run(ctx) }()

	cancel()

	waitRun(t, done, context.Canceled)
}

func TestSessionEventsChannelClose(t *testing.T) {
	t.Parallel()

	src := &countingSource{}
	surf := newFakeLive(50, 50)
	sess := output.NewSession(src, surf)

	done := make(chan error, 1)
	go func() { done <- sess.Run(context.Background()) }()

	close(surf.events)

	waitRun(t, done, nil)
}

func TestSessionAsyncRebuildCoalesces(t *testing.T) {
	t.Parallel()

	src := &countingSource{}
	surf := newFakeLive(80, 60)

	ctrl := output.ControllerFunc(func(ev output.Event, _ *output.State) output.Action {
		if ev.Kind == output.EventKey {
			return output.ActionRebuild
		}

		return output.ActionIgnore
	})

	sess := output.NewSession(src, surf,
		output.WithController(ctrl),
		output.WithRebuildDelay(20*time.Millisecond),
	)

	// Queue three rapid rebuild triggers, then close: with debouncing they
	// collapse to a single background build, flushed on channel close.
	for range 3 {
		surf.events <- output.Event{Kind: output.EventKey}
	}

	close(surf.events)

	done := make(chan error, 1)
	go func() { done <- sess.Run(context.Background()) }()

	waitRun(t, done, nil)

	if got := src.count(); got != 2 {
		t.Errorf("builds=%d, want 2 (initial + 1 coalesced rebuild)", got)
	}

	if surf.commitCount() < 2 {
		t.Errorf("commits=%d, want >=2 (initial + rebuilt frame)", surf.commitCount())
	}
}

func TestSessionSyncRebuildUnchanged(t *testing.T) {
	t.Parallel()

	src := &countingSource{}
	surf := newFakeLive(80, 60)

	ctrl := output.ControllerFunc(func(ev output.Event, _ *output.State) output.Action {
		if ev.Kind == output.EventKey {
			return output.ActionRebuild
		}

		return output.ActionIgnore
	})

	// No WithRebuildDelay: each rebuild is synchronous, so three keys build thrice.
	sess := output.NewSession(src, surf, output.WithController(ctrl))

	runEvents(t, sess, surf,
		output.Event{Kind: output.EventKey},
		output.Event{Kind: output.EventKey},
		output.Event{Kind: output.EventKey},
		output.Event{Kind: output.EventClose},
	)

	if got := src.count(); got != 4 {
		t.Errorf("builds=%d, want 4 (initial + 3 synchronous rebuilds)", got)
	}
}
