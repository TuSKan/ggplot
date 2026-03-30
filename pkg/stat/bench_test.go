package stat

import (
	"testing"

	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/internal/expr"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
)

func createBenchDataset(size int) Context {
	buf := arrow.NewBuffer(size)
	xCol := buf.Float64("x")
	yCol := buf.Float64("y")

	for i := 0; i < size; i++ {
		xCol[i] = float64(i % 100)
		yCol[i] = float64(i)
	}

	ds, _ := buf.Build()

	return Context{
		Dataset: ds,
		Aes: ast.Aes{
			"x": expr.Col("x"),
			"y": expr.Col("y"),
		},
	}
}

func BenchmarkCount(b *testing.B) {
	ctx := createBenchDataset(10000)
	s := NewCount()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = s.Compute(ctx)
	}
}

func BenchmarkBin(b *testing.B) {
	ctx := createBenchDataset(10000)
	s := NewBin(BinOptions{Bins: 50})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = s.Compute(ctx)
	}
}

func BenchmarkSmooth(b *testing.B) {
	ctx := createBenchDataset(10000)
	s := NewSmooth(SmoothOptions{Method: MethodLinear})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = s.Compute(ctx)
	}
}
