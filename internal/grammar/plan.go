package grammar

import (
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/geom"
)

// RenderPlan is the output of the compilation pipeline — a fully resolved,
// validated plot specification ready for rendering.
type RenderPlan struct {
	// Panels contains the list of facet panels to render, each with its
	// own dataset and compiled layers.
	Panels []PanelPlan

	// TrainedScales maps aesthetic channels to their trained scale state.
	TrainedScales map[string]TrainedScale

	// Labels holds resolved plot labels.
	Labels Labels

	// ThemeName is the resolved theme.
	ThemeName string
}

// PanelPlan describes a single panel (facet sub-plot) in the render plan.
type PanelPlan struct {
	Label   string
	Dataset dataset.Dataset
	Layers  []CompiledLayer
}

// CompiledLayer is a fully resolved layer ready for drawing.
type CompiledLayer struct {
	Geom    geom.Layer
	Mapping AesMap          // fully merged mapping (global + layer)
	Data    dataset.Dataset // post-stat dataset (may differ from panel dataset)
}

// TrainedScale holds the trained domain of a scale for an aesthetic channel.
type TrainedScale struct {
	Channel  string
	Type     string // "continuous", "discrete", "temporal"
	Min, Max float64
	Domain   []string // for discrete scales
}
