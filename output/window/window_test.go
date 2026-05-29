//go:build !js

package window_test

import (
	"testing"

	"github.com/TuSKan/ggplot/output"
	"github.com/TuSKan/ggplot/output/window"
)

func TestOptionsApply(t *testing.T) {
	t.Parallel()

	var o window.Options

	ctrl := output.DefaultController()
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

	o := window.Options{Controller: output.DefaultController()}
	window.WithController(nil)(&o)

	if o.Controller == nil {
		t.Error("WithController(nil) should not clear an existing controller")
	}
}
