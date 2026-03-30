package expr_test

import (
	"testing"

	"github.com/TuSKan/ggplot/internal/expr"
	"github.com/TuSKan/ggplot/pkg/dataset"
)

type emptyDataset struct {
	len int
}

func (e emptyDataset) Columns() []string                     { return nil }
func (e emptyDataset) Column(string) (dataset.Column, error) { return nil, nil }
func (e emptyDataset) Len() int                              { return e.len }

func BenchmarkEvaluate(b *testing.B) {
	ds := emptyDataset{len: 10000}
	ev := expr.NewEvaluator(ds)
	// (10.0 + 5.0) * 2.0
	e := expr.Mul(expr.Add(expr.Lit(10.0), expr.Lit(5.0)), expr.Lit(2.0))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ev.EvalFloat64(e)
	}
}
