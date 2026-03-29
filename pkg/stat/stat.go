package stat

import "github.com/TuSKan/ggplot/internal/ast"

// Definition structures a basic statistical trait.
type Definition struct {
	methodName string
}

func (d Definition) Name() string { return d.methodName }

// Commonly used stats mapped to public API.
var (
	MethodIdentity = Definition{methodName: "identity"}
	MethodLoess    = Definition{methodName: "loess"}
	MethodCount    = Definition{methodName: "count"}
)

// Ensure Definition matches interface.
var _ ast.Stat = Definition{}
