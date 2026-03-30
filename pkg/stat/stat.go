package stat

import (
	"fmt"
	"sync"

	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/internal/expr"
	"github.com/TuSKan/ggplot/pkg/dataset"
)

// Kind describes the expected output layout of the Stat.
type Kind string

const (
	KindTransform Kind = "transform" // 1:1 mapping
	KindAggregate Kind = "aggregate" // n:m grouping
	KindSummary   Kind = "summary"   // n:1 reduction
)

// Context encapsulates data and declarative mappings provided by the engine.
type Context struct {
	Dataset dataset.Dataset
	Aes     ast.Aes
	Groups  []string
}

// Stat represents a foundational data transformer within the grammar of graphics.
type Stat interface {
	Name() string
	Kind() Kind
	Compute(Context) (dataset.Dataset, error)
}

var (
	registry = make(map[string]Stat)
	mu       sync.RWMutex
)

// Register installs a Stat into the global engine registry.
func Register(s Stat) {
	mu.Lock()
	defer mu.Unlock()
	registry[s.Name()] = s
}

// Lookup queries the global engine registry.
func Lookup(name string) (Stat, bool) {
	mu.RLock()
	defer mu.RUnlock()
	s, ok := registry[name]
	return s, ok
}

// resolveColName finds the bound column name in the Aes map natively.
// Only handles direct ColumnRef bindings lazily evaluated natively.
func resolveColName(mapping ast.Aes, aesKey string) (string, error) {
	if mapping == nil {
		return "", &ErrMissingColumn{Name: aesKey}
	}
	e, ok := mapping[aesKey]
	if !ok {
		return "", &ErrMissingColumn{Name: aesKey}
	}
	if ref, isRef := e.(*expr.ColumnRef); isRef {
		return ref.Name, nil
	}
	return "", fmt.Errorf("stat: lazy resolution for complex expressions is unsupported for %q", aesKey)
}

// Identity implements the identity stat, making no transformations.
type Identity struct{}

func NewIdentity() Stat { return Identity{} }
func (s Identity) Name() string { return "identity" }
func (s Identity) Kind() Kind { return KindTransform }
func (s Identity) Compute(ctx Context) (dataset.Dataset, error) {
	return ctx.Dataset, nil
}

// Global default stat implementations to satisfy backward compatibility
// across geometries relying on ast.Stat abstractions trivially.
var (
	MethodIdentity = NewIdentity()
	MethodCount    = NewCount()
	MethodBin      = NewBin(BinOptions{})
	MethodLoess    = NewSmooth(SmoothOptions{Method: "loess"})
)

func init() {
	Register(MethodIdentity)
}

