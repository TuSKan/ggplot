package scale_test

import (
	"strconv"
	"testing"

	"github.com/TuSKan/ggplot/pkg/scale"
	"github.com/TuSKan/ggplot/pkg/dataset"
)

type benchStringIter struct {
	items []string
	idx   int
}

func (s *benchStringIter) Next() (string, bool, bool) {
	if s.idx >= len(s.items) {
		return "", false, false
	}
	val := s.items[s.idx]
	s.idx++
	return val, false, true
}

type benchStringCol struct {
	items []string
}

func (s benchStringCol) Name() string { return "bCol" }
func (s benchStringCol) Len() int     { return len(s.items) }
func (s benchStringCol) Strings() (dataset.StringIterator, error) {
	return &benchStringIter{items: s.items}, nil
}

func BenchmarkDiscreteTrainLarge(b *testing.B) {
	var items []string
	for i := 0; i < 100000; i++ {
		items = append(items, "category_"+strconv.Itoa(i%1500))
	}
	col := benchStringCol{items: items}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := scale.NewDiscrete()
		_ = d.Train(col)
	}
}
