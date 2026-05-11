package ggplot

import "errors"

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
)
