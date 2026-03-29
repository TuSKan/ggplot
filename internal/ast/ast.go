package ast

import (
	"fmt"
	"strings"

	"github.com/TuSKan/ggplot/internal/dataset"
	"github.com/TuSKan/ggplot/internal/scale"
)

// AestheticMapping binds data columns to visual properties.
type AestheticMapping struct {
	X     string
	Y     string
	Color string
}

// Merge combines another mapping into this one, letting the other override blanks.
func (m AestheticMapping) Merge(other AestheticMapping) AestheticMapping {
	if m.X == "" {
		m.X = other.X
	}
	if m.Y == "" {
		m.Y = other.Y
	}
	if m.Color == "" {
		m.Color = other.Color
	}
	return m
}

// Geom definitions describe rendering behavior constraints.
type Geom interface {
	Name() string
	RequiredAesthetics() []string
}

// Stat definitions specify transformation rules.
type Stat interface {
	Name() string
}

// Layer specifies an immutable single graphical trace.
type Layer struct {
	Geom    Geom
	Stat    Stat
	Mapping AestheticMapping
}

// Plot defines the grammar tree top-level structure.
type Plot struct {
	Dataset any // Expected to be dataset.Dataset
	Mapping AestheticMapping
	Layers  []Layer
}

// RenderPlan is the concrete output post-validation.
type RenderPlan struct {
	Dataset any // Compiled primary dataset or ref
	Layers  []CompiledLayer
	Scales  map[string]any // Mapped generic scales trained across all datasets.
}

// CompiledLayer represents a fully validated layout structure.
type CompiledLayer struct {
	Geom    Geom
	Stat    Stat
	Mapping AestheticMapping
	// Additional layout offsets/domains would be here
}

// Compile validates the plot and computes the RenderPlan.
func (p *Plot) Compile() (*RenderPlan, error) {
	plan := &RenderPlan{
		Dataset: p.Dataset,
		Layers:  make([]CompiledLayer, len(p.Layers)),
		Scales:  make(map[string]any),
	}

	for i, l := range p.Layers {
		// 1. Merge global mappings with layer mappings (layer overrides global)
		// but since we want layer to override, we start with Layer, missing gets global.
		merged := l.Mapping.Merge(p.Mapping)

		// 2. Validate Geometry Aesthetics
		if l.Geom == nil {
			return nil, fmt.Errorf("layer %d: missing geom", i)
		}
		
		missing := []string{}
		for _, req := range l.Geom.RequiredAesthetics() {
			switch req {
			case "x":
				if merged.X == "" { missing = append(missing, "x") }
			case "y":
				if merged.Y == "" { missing = append(missing, "y") }
			}
		}

		if len(missing) > 0 {
			return nil, fmt.Errorf("layer %d (%s) missing required aesthetics: %s", i, l.Geom.Name(), strings.Join(missing, ", "))
		}

		// Train scales
		if p.Dataset != nil {
			ds, ok := p.Dataset.(dataset.Dataset)
			if ok {
				for _, colName := range []string{merged.X, merged.Y, merged.Color} {
					if colName == "" {
						continue
					}
					
					col, err := ds.Column(colName)
					if err != nil {
						continue
					}

					var s scale.Scale
					if existing, found := plan.Scales[colName]; found {
						s = existing.(scale.Scale)
						s.Train(col)
					} else {
						// Try Continuous first, fallback to Discrete
						cs := scale.NewContinuous()
						if err := cs.Train(col); err == nil {
							plan.Scales[colName] = cs
						} else {
							ds := scale.NewDiscrete()
							ds.Train(col)
							plan.Scales[colName] = ds
						}
					}
				}
			}
		}

		plan.Layers[i] = CompiledLayer{
			Geom:    l.Geom,
			Stat:    l.Stat,
			Mapping: merged,
		}
	}

	return plan, nil
}
