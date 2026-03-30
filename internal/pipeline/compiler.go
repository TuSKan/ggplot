package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/TuSKan/ggplot/internal/ast"
)

// Compiler coordinates the translation from declarative AST to a RenderPlan.
type Compiler interface {
	Compile(p *ast.Plot) (*ast.RenderPlan, error)
}

// DefaultCompiler is the standard pipeline implementation mapping stages.
type DefaultCompiler struct {
	Stages []StageExecutor
}

// New creates a new DefaultCompiler resolving clearly.
func New() *DefaultCompiler {
	return &DefaultCompiler{
		Stages: []StageExecutor{
			StatTransformStage{},
			ScaleTrainingStage{},
			LayoutStage{},
			EmissionStage{},
		},
	}
}

func checkStatGeom(s ast.Stat, g ast.Geom) error {
	sn, gn := strings.ToLower(s.Name()), strings.ToLower(g.Name())

	if gn == "point" && sn != "identity" {
		return fmt.Errorf("incompatible stat %s for geometry %s", sn, gn)
	}
	if gn == "smooth" && (sn != "smooth" && sn != "loess" && sn != "lm") {
		return fmt.Errorf("incompatible stat %s for geometry %s", sn, gn)
	}
	return nil
}

// Compile runs the defined sequence of pipeline passes.
func (c *DefaultCompiler) Compile(p *ast.Plot) (*ast.RenderPlan, error) {
	plan := &ast.RenderPlan{
		Dataset: p.Dataset,
		Layers:  make([]ast.CompiledLayer, len(p.Layers)),
		Scales:  make(map[string]any),
		Theme:   p.Theme,
		Facet:   p.Facet,
	}

	// Preset Global Mappings
	for i, l := range p.Layers {
		plan.Layers[i] = ast.CompiledLayer{
			Geom:    l.Geom,
			Stat:    l.Stat,
			Mapping: l.Mapping.Merge(p.Mapping),
		}
	}

	ctx := &PipelineContext{
		Durations: make(map[Stage]time.Duration),
	}

	stagesOrder := []Stage{
		StageStatTransform,
		StageScaleTraining,
		StageLayout,
		StageEmission,
	}

	for i, exec := range c.Stages {
		if err := ExecuteStage(stagesOrder[i], exec, ctx, plan); err != nil {
			return nil, err
		}
	}

	return plan, nil
}
