package plot

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/internal/dataset"
)

// LayerBuilder defines anything that can construct a Layer.
// Geometries normally implement this to set their defaults.
type LayerBuilder interface {
	BuildLayer() ast.Layer
}

// Plot is the declarative immutable root builder.
type Plot struct {
	astPlot *ast.Plot
}

// New initializes a base plot with a dataset.
func New(data dataset.Dataset) *Plot {
	return &Plot{
		astPlot: &ast.Plot{
			Dataset: data,
			Layers:  []ast.Layer{},
		},
	}
}

// clone deep-copies the current Plot AST for immutability.
func (p *Plot) clone() *Plot {
	layers := make([]ast.Layer, len(p.astPlot.Layers))
	copy(layers, p.astPlot.Layers)
	c := &ast.Plot{
		Dataset: p.astPlot.Dataset,
		Mapping: p.astPlot.Mapping,
		Layers:  layers,
	}
	return &Plot{astPlot: c}
}

// AddLayer attaches a geometry layer and overrides mapping lazily.
func (p *Plot) AddLayer(builder LayerBuilder, opts ...func(*ast.AestheticMapping)) *Plot {
	cloned := p.clone()
	layer := builder.BuildLayer()
	
	for _, opt := range opts {
		opt(&layer.Mapping)
	}

	cloned.astPlot.Layers = append(cloned.astPlot.Layers, layer)
	return cloned
}

// Compile performs validation and prepares the render structure.
func (p *Plot) Compile() (*ast.RenderPlan, error) {
	return p.astPlot.Compile()
}
