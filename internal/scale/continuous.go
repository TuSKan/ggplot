package scale

import (
	"fmt"
	"math"

	"github.com/TuSKan/ggplot/internal/dataset"
)

// NA represents graphical missingness mapping
var NA float64 = math.NaN()

// Continuous defines a smooth bounding box.
type Continuous struct {
	Min     float64
	Max     float64
	hasData bool
}

func NewContinuous() *Continuous {
	return &Continuous{}
}

// Train expands the bounding limits based on the passed numerical column.
func (c *Continuous) Train(col dataset.Column) error {
	min, minErr := dataset.Min(col)
	max, maxErr := dataset.Max(col)
	if minErr != nil || maxErr != nil {
		return fmt.Errorf("scale continuous: could not calculate bounds")
	}

	if !c.hasData {
		c.Min = min
		c.Max = max
		c.hasData = true
		return nil
	}

	if min < c.Min {
		c.Min = min
	}
	if max > c.Max {
		c.Max = max
	}

	return nil
}

// Map interpolates a numerical value uniformly spanning exactly 0.0 to 1.0.
func (c *Continuous) Map(val float64) float64 {
	if !c.hasData || math.IsNaN(val) {
		return NA
	}
	if c.Max == c.Min {
		return 0.5 // Handle exact singular constants gracefully by centering
	}
	return (val - c.Min) / (c.Max - c.Min)
}
