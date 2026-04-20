// Package output provides rendering output abstractions for exporting
// and displaying plots.
package output

// Output encapsulates a fully compiled, rendered plot alongside its
// dimensions. It serves as a unified result that can be saved to file
// or displayed in an interactive window.
type Output struct {
	Width  int
	Height int
	// The underlying canvas is accessible via the concrete type
	// returned by ggplot.Render().
}

// Exporter defines a backend that saves rendered output to a file.
type Exporter interface {
	Export(filename string) error
}

// Presenter defines a backend that displays rendered output in a window.
type Presenter interface {
	Show() error
}
