//go:build !js

package window_test

import (
	"testing"
	"time"

	"github.com/TuSKan/ggplot/output"
	"github.com/TuSKan/ggplot/output/window"
)

func TestOptionsApply(t *testing.T) {
	t.Parallel()

	var o window.Options

	ctrl := output.DataSpaceController()
	for _, opt := range []window.Opt{
		window.WithTitle("My Plot"),
		window.WithSize(1024, 768),
		window.WithController(ctrl),
	} {
		opt(&o)
	}

	if o.Title != "My Plot" {
		t.Errorf("Title=%q, want %q", o.Title, "My Plot")
	}

	if o.Width != 1024 || o.Height != 768 {
		t.Errorf("size=%dx%d, want 1024x768", o.Width, o.Height)
	}

	if o.Controller == nil {
		t.Error("Controller not set")
	}
}

func TestWithControllerNilIgnored(t *testing.T) {
	t.Parallel()

	o := window.Options{Controller: output.DataSpaceController()}
	window.WithController(nil)(&o)

	if o.Controller == nil {
		t.Error("WithController(nil) should not clear an existing controller")
	}
}

func TestWithRebuildDelay(t *testing.T) {
	t.Parallel()

	var o window.Options
	window.WithRebuildDelay(50 * time.Millisecond)(&o)

	if o.RebuildDelay != 50*time.Millisecond {
		t.Errorf("RebuildDelay=%v, want 50ms", o.RebuildDelay)
	}
}

func TestWithRebuildError(t *testing.T) {
	t.Parallel()

	var called bool

	fn := func(_ error) { called = true }

	var o window.Options
	window.WithRebuildError(fn)(&o)

	if o.OnRebuildErr == nil {
		t.Fatal("OnRebuildErr not set")
	}

	o.OnRebuildErr(nil)

	if !called {
		t.Error("OnRebuildErr was not invoked")
	}
}
