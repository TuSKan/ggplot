package ggplot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Phase identifies the pipeline phase where an error occurred.
type Phase int

const (
	// PhaseBuild is the Plot.Build phase (data → stat → scale → position).
	PhaseBuild Phase = iota
	// PhaseDraw is the Built.Draw phase (rendering to canvas).
	PhaseDraw
	// PhaseRender is the Save/Encode/Image phase (format dispatch, I/O).
	PhaseRender
)

// String returns a human-readable phase name.
func (p Phase) String() string {
	switch p {
	case PhaseBuild:
		return "build"
	case PhaseDraw:
		return "draw"
	case PhaseRender:
		return "render"
	default:
		return "unknown"
	}
}

// Error is a structured error from the ggplot pipeline.
// It supports errors.Is (via Cause chain) and errors.As.
//
// Usage:
//
//	var ggErr *ggplot.Error
//	if errors.As(err, &ggErr) {
//	    fmt.Printf("phase=%s layer=%d stage=%s\n", ggErr.Phase, ggErr.Layer, ggErr.Stage)
//	}
type Error struct {
	Phase Phase  // pipeline phase (Build, Draw, Render)
	Layer int    // 0-based layer index, -1 if not layer-specific
	Stage string // sub-stage within the phase (e.g. "transform", "scale", "facet")
	Msg   string // short human-readable description
	Cause error  // underlying error (may be nil)
}

// Error returns a formatted error string.
//
// Examples:
//
//	"ggplot [build/facet]: facet split"
//	"ggplot [build/layer 2/transform]: transform pipeline failed for group \"A\""
//	"ggplot [render]: unsupported format \"gif\""
func (e *Error) Error() string {
	var b strings.Builder

	b.WriteString("ggplot [")
	b.WriteString(e.Phase.String())

	if e.Layer >= 0 {
		b.WriteString("/layer ")
		b.WriteString(strconv.Itoa(e.Layer))
	}

	if e.Stage != "" {
		b.WriteByte('/')
		b.WriteString(e.Stage)
	}

	b.WriteString("]: ")
	b.WriteString(e.Msg)

	if e.Cause != nil {
		b.WriteString(": ")
		b.WriteString(e.Cause.Error())
	}

	return b.String()
}

// Unwrap returns the underlying cause for errors.Is / errors.As traversal.
func (e *Error) Unwrap() error {
	return e.Cause
}

// Errorf creates a structured [*Error].
// Layer should be -1 if the error is not layer-specific.
func Errorf(phase Phase, layer int, stage string, cause error, msg string, args ...any) *Error {
	return &Error{
		Phase: phase,
		Layer: layer,
		Stage: stage,
		Msg:   fmt.Sprintf(msg, args...),
		Cause: cause,
	}
}

// Sentinel errors for the ggplot package.
var (
	// ErrUnsupportedFormat is returned for unsupported output formats.
	ErrUnsupportedFormat = errors.New("ggplot: unsupported output format")

	// ErrRenderFailed is returned when rendering fails.
	ErrRenderFailed = errors.New("ggplot: render failed")

	// ErrNoLayers is returned when there are no layers to render.
	ErrNoLayers = errors.New("ggplot: no layers to render")

	// ErrMissingAesthetic is returned when a required aesthetic is missing.
	ErrMissingAesthetic = errors.New("ggplot: missing required aesthetic")

	// ErrInvalidConfig is returned for invalid plot configuration.
	ErrInvalidConfig = errors.New("ggplot: invalid configuration")

	// ErrUnsupportedTransform is returned for unknown coord transform names.
	ErrUnsupportedTransform = errors.New("ggplot: unsupported coordinate transform")
)
