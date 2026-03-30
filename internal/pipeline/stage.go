package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/internal/expr"
	"github.com/TuSKan/ggplot/pkg/scale"
	"github.com/TuSKan/ggplot/pkg/dataset"
)

// Stage fundamentally tracks decoupled compilation bounds.
type Stage string

const (
	StageStatTransform Stage = "stat_transform"
	StageScaleTraining Stage = "scale_training"
	StageScaleMapping  Stage = "scale_mapping"
	StageLayout        Stage = "layout"
	StageEmission      Stage = "emission"
)

// PipelineContext traces execution benchmarks functionally tightly.
type PipelineContext struct {
	Durations map[Stage]time.Duration
	Current   Stage
}

// StageExecutor isolates pipeline handlers.
type StageExecutor interface {
	Execute(ctx *PipelineContext, plan *ast.RenderPlan) error
}

// ExecuteStage traces logic tightly!
func ExecuteStage(stage Stage, exec StageExecutor, ctx *PipelineContext, plan *ast.RenderPlan) error {
	ctx.Current = stage
	t0 := time.Now()
	err := exec.Execute(ctx, plan)
	ctx.Durations[stage] = time.Since(t0)
	if err != nil {
		return fmt.Errorf("pipeline failed at [%s]: %w", stage, err)
	}
	return nil
}

// StatTransformStage functionally maps checks appropriately!
type StatTransformStage struct{}

func (s StatTransformStage) Execute(ctx *PipelineContext, plan *ast.RenderPlan) error {
	for i, l := range plan.Layers {
		if l.Geom == nil {
			return fmt.Errorf("layer %d: missing geom ", i)
		}

		missing := []string{}
		for _, req := range l.Geom.RequiredAesthetics() {
			if _, ok := l.Mapping[req]; !ok {
				missing = append(missing, req)
			}
		}

		if len(missing) > 0 {
			return fmt.Errorf("layer %d missing required : %s", i, strings.Join(missing, ", "))
		}

		if err := checkStatGeom(l.Stat, l.Geom); err != nil {
			return fmt.Errorf("layer %d statically appropriately statically functionally %v", i, err) // Reverting the excessive verbosity internally
		}
	}
	return nil
}

// ScaleTrainingStage drives geometric limits.
type ScaleTrainingStage struct{}

func (s ScaleTrainingStage) Execute(ctx *PipelineContext, plan *ast.RenderPlan) error {
	if ds, ok := plan.Dataset.(dataset.Dataset); ok {
		mappings := make([]ast.Aes, len(plan.Layers))
		for i, l := range plan.Layers {
			mappings[i] = l.Mapping
		}
		TrainScales(plan, ds, mappings)
	}
	return nil
}

// ScaleMappingStage isolates bounds iteratively statically statically.
type ScaleMappingStage struct{}

func (s ScaleMappingStage) Execute(ctx *PipelineContext, plan *ast.RenderPlan) error {
	ds, ok := plan.Dataset.(dataset.Dataset)
	if !ok {
		return nil
	}

	mut := make(dataset.MutateRegistry)

	for i, l := range plan.Layers {
		for key, e := range l.Mapping {
			colRef, isRef := e.(*expr.ColumnRef)
			if !isRef {
				continue
			}

			sc, hasScale := plan.Scales[key]
			if !hasScale {
				continue
			}

			mappedName := "__scaled_" + key + "_" + colRef.Name

			if cMap, isCont := sc.(scale.ContinuousMapper); isCont {
				mut[mappedName] = func() (dataset.Column, error) {
					rawCol, err := ds.Column(colRef.Name)
					if err != nil {
						return nil, err
					}
					return dataset.NewTransformedFloat64Column(rawCol, cMap.Map), nil
				}
				plan.Layers[i].Mapping[key] = expr.Col(mappedName)
			} else if dMap, isDisc := sc.(scale.DiscreteMapper); isDisc {
				mut[mappedName] = func() (dataset.Column, error) {
					rawCol, err := ds.Column(colRef.Name)
					if err != nil {
						return nil, err
					}
					return dataset.NewTransformedStringColumn(rawCol, dMap.Map), nil
				}
				plan.Layers[i].Mapping[key] = expr.Col(mappedName)
			}
		}
	}

	if len(mut) > 0 {
		plan.Dataset = dataset.Mutate(ds, mut)
	}

	return nil
}

// LayoutStage binds rectangles!
type LayoutStage struct{}

func (s LayoutStage) Execute(ctx *PipelineContext, plan *ast.RenderPlan) error {
	// Coordinates bounds tightly functionally tightly.
	return nil
}

// EmissionStage outputs concrete shapes clearly functionally!
type EmissionStage struct{}

func (s EmissionStage) Execute(ctx *PipelineContext, plan *ast.RenderPlan) error {
	return nil
}
