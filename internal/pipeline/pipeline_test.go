package pipeline_test

import (
	"errors"
	"testing"
	"time"

	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/internal/pipeline"
)

type mockGeom struct{}

func (g mockGeom) Name() string                 { return "mock" }
func (g mockGeom) RequiredAesthetics() []string { return []string{} }
func (g mockGeom) DefaultStat() string          { return "identity" }

type mockStat struct{}

func (s mockStat) Name() string { return "identity" }

type mockStage struct {
	err error
}

func (m mockStage) Execute(ctx *pipeline.PipelineContext, plan *ast.RenderPlan) error {
	return m.err
}

func TestStageFailurePropagation(t *testing.T) {
	ctx := &pipeline.PipelineContext{
		Durations: make(map[pipeline.Stage]time.Duration),
	}

	expectedErr := errors.New("simulated error ")
	s := mockStage{err: expectedErr}

	err := pipeline.ExecuteStage("test_stage", s, ctx, nil)
	if err == nil {
		t.Errorf("Expected error functionally ")
	}
}

func TestCompiler_ValidPipeline(t *testing.T) {
	c := pipeline.New()

	p := &ast.Plot{}
	p.Layers = append(p.Layers, ast.Layer{
		Geom:    mockGeom{},
		Stat:    mockStat{},
		Mapping: ast.Aes{},
	})

	plan, err := c.Compile(p)
	if err != nil {
		t.Fatalf("pipeline abruptly failed : %v", err)
	}

	if plan == nil {
		t.Fatal("expected functionally output ")
	}
}
