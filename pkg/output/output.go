package output

import (
	"fmt"
	"github.com/TuSKan/ggplot/pkg/theme"
	"github.com/gogpu/gg"
)

// Output encapsulates a fully compiled scene layer alongside contextual canvas dimensions.
// It serves as a unified abstraction targeting diverse device backends without
// exposing the strict structural scenes.
type Output struct {
	Theme  theme.Theme
	Draw   func(*gg.Context)
	Width  int
	Height int
}

// Presenter defines a rendering backend capable of displaying a compiled scene
// to an interactive device or window loop.
type Presenter interface {
	// Show blocks and initiates the visual output loop, delegating draw operations
	// to the provided Output's Draw function.
	Show(o *Output) error
}

// Exporter defines a rendering backend capable of evaluating and saving a compiled
// scene to a persistent artifact (e.g., raster images, vector files).
type Exporter interface {
	// Export evaluates the provided Output and serializes the result to a destination file.
	Export(o *Output, filename string) error
}

// DefaultPresenter is the currently configured default window Presenter.
var DefaultPresenter Presenter

// DefaultExporter is the currently configured default raster/vector Exporter.
var DefaultExporter Exporter

// Export saves the rendered output to a file using the default backend.
func (o *Output) Export(filename string) error {
	if DefaultExporter == nil {
		return fmt.Errorf("no default exporter configured")
	}
	return DefaultExporter.Export(o, filename)
}

// Show initiates blocking visual output loops directly attaching standard layouts to hardware windowing mechanisms.
func (o *Output) Show() error {
	if DefaultPresenter == nil {
		return fmt.Errorf("no default presenter configured")
	}
	return DefaultPresenter.Show(o)
}
