package scale_test

import (
	"fmt"
	"testing"

	"github.com/TuSKan/ggplot/internal/dataset"
	"github.com/TuSKan/ggplot/internal/scale"
)

type stringGenIter struct {
	max int
	i   int
}

func (s *stringGenIter) Next() (string, bool, bool) {
	if s.i >= s.max {
		return "", false, false
	}
	val := fmt.Sprintf("level-%d", s.i)
	s.i++
	return val, false, true
}

type mockGenCol struct {
	max int
}

func (m mockGenCol) Len() int { return m.max }
func (m mockGenCol) Strings() (dataset.StringIterator, error) {
	return &stringGenIter{max: m.max}, nil
}

func BenchmarkDiscreteScaleTrain(b *testing.B) {
	col := mockGenCol{max: 1_000_000} // Synthesize 1-million unique string combinations.

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := scale.NewDiscrete()
		_ = s.Train(col)
	}
}
