package pipeline_test

import (
	"testing"

	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/internal/pipeline"
)

func BenchmarkScaleMappingZeroCopy(b *testing.B) {
	ctx := &pipeline.PipelineContext{}
	plan := &ast.RenderPlan{}

	stage := pipeline.ScaleMappingStage{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Zero allocation mapping lazily
		_ = stage.Execute(ctx, plan)
	}
}

func BenchmarkPipelineExecution(b *testing.B) {
	c := pipeline.New()
	p := &ast.Plot{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compile(p)
	}
}
