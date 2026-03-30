package geom

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/pkg/scale"
	"github.com/TuSKan/ggplot/pkg/dataset"
)

// LayerContext binds data, scales, mask grouping, and mappings.
type LayerContext struct {
	Dataset  dataset.Dataset
	Bindings ast.Aes
	Scales   scale.Manager
	Mask     []int // Represents an active subset of rows internally
}

// Geometry compiles data and aesthetic mappings into generic drawing parameters functionally!
type Geometry interface {
	// Compile produces abstract draw commands.
	Compile(ctx *LayerContext) error
}

// Point geometry implementation.
type Point struct{}

func (p *Point) Compile(ctx *LayerContext) error { return nil }

// Line geometry implementation.
type Line struct{}

func (l *Line) Compile(ctx *LayerContext) error { return nil }

// Bar geometry implementation.
type Bar struct{}

func (b *Bar) Compile(ctx *LayerContext) error { return nil }

// Area geometry implementation.
type Area struct{}

func (a *Area) Compile(ctx *LayerContext) error { return nil }

// Polygon geometry implementation functionally.
type Polygon struct{}

func (p *Polygon) Compile(ctx *LayerContext) error { return nil }

// Histogram geometry.
type Histogram struct{}

func (h *Histogram) Compile(ctx *LayerContext) error { return nil }
