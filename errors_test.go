package ggplot

import (
	"errors"
	"fmt"
	"testing"
)

// Test-only sentinel errors for err113 compliance.
var (
	errTestColumn     = errors.New("column x not found")
	errTestUnderlying = errors.New("underlying problem")
)

func TestError_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "build/no layer/facet",
			err:  Errorf(PhaseBuild, -1, "facet", nil, "facet split"),
			want: "ggplot [build/facet]: facet split",
		},
		{
			name: "build/layer 2/transform with cause",
			err:  Errorf(PhaseBuild, 2, "transform", errTestColumn, "pipeline failed for group %q", "A"),
			want: `ggplot [build/layer 2/transform]: pipeline failed for group "A": column x not found`,
		},
		{
			name: "draw/no stage",
			err:  Errorf(PhaseDraw, -1, "", nil, "context cancelled"),
			want: "ggplot [draw]: context cancelled",
		},
		{
			name: "render/format",
			err:  Errorf(PhaseRender, -1, "", ErrUnsupportedFormat, "unsupported format %q", "gif"),
			want: `ggplot [render]: unsupported format "gif": ggplot: unsupported output format`,
		},
		{
			name: "layer 0",
			err:  Errorf(PhaseBuild, 0, "scale", nil, "scale training failed"),
			want: "ggplot [build/layer 0/scale]: scale training failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() =\n  %q\nwant\n  %q", got, tt.want)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	t.Parallel()

	cause := errTestUnderlying
	err := Errorf(PhaseBuild, 1, "transform", cause, "pipeline failed")

	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(err, cause) = false, want true")
	}
}

func TestError_Is_Sentinel(t *testing.T) {
	t.Parallel()

	err := Errorf(PhaseBuild, -1, "validate", ErrNoLayers, "no layers")

	if !errors.Is(err, ErrNoLayers) {
		t.Errorf("errors.Is(err, ErrNoLayers) = false, want true")
	}

	if errors.Is(err, ErrRenderFailed) {
		t.Errorf("errors.Is(err, ErrRenderFailed) = true, want false")
	}
}

func TestError_Is_WrappedSentinel(t *testing.T) {
	t.Parallel()

	// Two-level wrapping: Error → fmt.Errorf → sentinel
	inner := fmt.Errorf("column missing: %w", ErrMissingAesthetic)
	outer := Errorf(PhaseBuild, 3, "scale", inner, "training failed")

	if !errors.Is(outer, ErrMissingAesthetic) {
		t.Errorf("errors.Is through two-level wrap = false, want true")
	}
}

func TestError_As(t *testing.T) {
	t.Parallel()

	err := Errorf(PhaseBuild, 5, "position", nil, "adjust failed")

	// Wrap it in fmt.Errorf to test As through wrapping.
	wrapped := fmt.Errorf("plot failed: %w", err)

	var ggErr *Error

	if !errors.As(wrapped, &ggErr) {
		t.Fatalf("errors.As failed")
	}

	if ggErr.Phase != PhaseBuild {
		t.Errorf("Phase = %v, want PhaseBuild", ggErr.Phase)
	}

	if ggErr.Layer != 5 {
		t.Errorf("Layer = %d, want 5", ggErr.Layer)
	}

	if ggErr.Stage != "position" {
		t.Errorf("Stage = %q, want %q", ggErr.Stage, "position")
	}
}

func TestPhase_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		phase Phase
		want  string
	}{
		{PhaseBuild, "build"},
		{PhaseDraw, "draw"},
		{PhaseRender, "render"},
		{Phase(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := tt.phase.String(); got != tt.want {
				t.Errorf("Phase(%d).String() = %q, want %q", tt.phase, got, tt.want)
			}
		})
	}
}
