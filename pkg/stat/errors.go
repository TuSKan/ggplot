package stat

import "fmt"

// ErrMissingColumn indicates a required aesthetic or data mapping is unresolved.
type ErrMissingColumn struct {
	Name string
}

func (e *ErrMissingColumn) Error() string {
	return fmt.Sprintf("stat: missing required column %q", e.Name)
}

// ErrInvalidType indicates an operation was run on an incompatible type (e.g. smoothing over strings).
type ErrInvalidType struct {
	Column   string
	Expected string
	Got      string
}

func (e *ErrInvalidType) Error() string {
	return fmt.Sprintf("stat: column %q expects type %q, got %q", e.Column, e.Expected, e.Got)
}

// ErrUnsupportedMethod indicates a requested algorithmic method is unavailable.
type ErrUnsupportedMethod struct {
	Stat   string
	Method string
}

func (e *ErrUnsupportedMethod) Error() string {
	return fmt.Sprintf("stat: unsupported method %q for %s", e.Method, e.Stat)
}
