package scale

import (
	"fmt"

	"github.com/TuSKan/ggplot/pkg/dataset"
)

// Temporal bounds timestamps mapped functionally.
type Temporal struct {
	Min      int64
	Max      int64
	Reversed bool
	hasData  bool
}

func NewTemporal() *Temporal {
	return &Temporal{}
}

func (t *Temporal) Train(col dataset.Column) error {
	minF, minErr := dataset.Min(col)
	maxF, maxErr := dataset.Max(col)
	if minErr != nil || maxErr != nil {
		return fmt.Errorf("scale temporal: could not calculate bounds")
	}

	min, max := int64(minF), int64(maxF)

	if !t.hasData {
		t.Min = min
		t.Max = max
		t.hasData = true
		return nil
	}

	if min < t.Min {
		t.Min = min
	}
	if max > t.Max {
		t.Max = max
	}

	return nil
}

func (t *Temporal) Map(val int64) float64 {
	if !t.hasData {
		return NA
	}
	if t.Max == t.Min {
		if t.Reversed {
			return 0.5
		}
		return 0.5
	}

	v := float64(val-t.Min) / float64(t.Max-t.Min)
	if t.Reversed {
		v = 1.0 - v
	}
	return v
}

var _ TemporalMapper = (*Temporal)(nil)
