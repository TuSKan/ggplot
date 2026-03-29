package scale_test

import (
	"math"
	"testing"

	"github.com/TuSKan/ggplot/internal/dataset"
	"github.com/TuSKan/ggplot/internal/scale"
)

type mockNumericCol struct {
	min float64
	max float64
}

func (m mockNumericCol) Len() int                    { return 10 }
func (m mockNumericCol) Min() (float64, error)       { return m.min, nil }
func (m mockNumericCol) Max() (float64, error)       { return m.max, nil }

// Guarantee interface match
var _ dataset.Aggregator = mockNumericCol{}

func TestNumericContinuous(t *testing.T) {
	s := scale.NewContinuous()
	
	col1 := mockNumericCol{min: 50.0, max: 100.0}
	s.Train(col1)

	// Validate mapping bounds perfectly normalize
	if v := s.Map(50.0); v != 0.0 {
		t.Errorf("expected 0.0 at minimum bound, got %f", v)
	}
	if v := s.Map(100.0); v != 1.0 {
		t.Errorf("expected 1.0 at maximum bound, got %f", v)
	}
	if v := s.Map(75.0); v != 0.5 {
		t.Errorf("expected 0.5 at exact midpoint, got %f", v)
	}

	// Validate incremental training expands bounds
	col2 := mockNumericCol{min: 0.0, max: 80.0}
	s.Train(col2)
	if v := s.Map(0.0); v != 0.0 {
		t.Errorf("expected domain to expand downwards to 0.0, got %f", v)
	}
	if v := s.Map(100.0); v != 1.0 {
		t.Errorf("expected domain to retain original max 100.0, got %f", v)
	}

	// Validate missing / out of bound nulls map safely
	if !math.IsNaN(s.Map(math.NaN())) {
		t.Errorf("expected math.NaN mapping to propagate NA correctly")
	}
}

// categoryIterator Mocks sequential array iterations
type categoryIterator struct {
	items []string
	idx   int
}

func (c *categoryIterator) Next() (string, bool, bool) {
	if c.idx >= len(c.items) {
		return "", false, false
	}
	val := c.items[c.idx]
	c.idx++
	return val, false, true
}

type mockIterableCol struct {
	items []string
}

func (m mockIterableCol) Len() int { return len(m.items) }
func (m mockIterableCol) Strings() (dataset.StringIterator, error) {
	return &categoryIterator{items: m.items}, nil
}

func TestCategoricalDiscrete(t *testing.T) {
	d := scale.NewDiscrete()
	
	// Two overlapping datasets to test unique discrete unification
	colA := mockIterableCol{items: []string{"Apple", "Banana", "Apple"}}
	colB := mockIterableCol{items: []string{"Banana", "Cherry"}}
	
	d.Train(colA)
	d.Train(colB)
	
	if len(d.Domain) != 3 {
		t.Fatalf("expected unique domain set size of 3, got %d", len(d.Domain))
	}

	// Banana should be mapped centrally 
	// Apple (0), Banana (1), Cherry (2) expected based on occurrence order.
	if v := d.Map("Banana"); v != 0.5 {
		t.Errorf("expected Banana centrally mapped 0.5, got %f", v)
	}

	// Missing categories should map explicitly to NA
	if math.IsNaN(d.Map("Mango")) == false {
		t.Errorf("expected unseen key 'Mango' to yield NaN")
	}
}
