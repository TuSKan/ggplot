package scale

import (
	"fmt"
	"math"
	"strconv"

	"github.com/TuSKan/ggplot/pkg/dataset"
)

// NA represents graphical missingness mapping
var NA float64 = math.NaN()

// Continuous defines a smooth bounding box.
type Continuous struct {
	Min      float64
	Max      float64
	Reversed bool
	hasData  bool
}

func NewContinuous() *Continuous {
	return &Continuous{}
}

// SetLimits forces the min and max limits explicitly disabling auto-training logic updates.
func (c *Continuous) SetLimits(min, max float64) {
	c.Min = min
	c.Max = max
	c.hasData = true
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
		if c.Reversed {
			return 0.5
		}
		return 0.5
	}

	v := (val - c.Min) / (c.Max - c.Min)
	if c.Reversed {
		v = 1.0 - v
	}
	return v
}

// niceNum returns a "nice" number approximately equal to x.
// If round is true, it rounds; otherwise it returns the ceiling.
func niceNum(x float64, round bool) float64 {
	exp := math.Floor(math.Log10(x))
	f := x / math.Pow(10, exp)
	var nf float64
	if round {
		if f < 1.5 {
			nf = 1
		} else if f < 3 {
			nf = 2
		} else if f < 7 {
			nf = 5
		} else {
			nf = 10
		}
	} else {
		if f <= 1 {
			nf = 1
		} else if f <= 2 {
			nf = 2
		} else if f <= 5 {
			nf = 5
		} else {
			nf = 10
		}
	}
	return nf * math.Pow(10, exp)
}

// Ticks implements guide.ContinuousScale.
func (c *Continuous) Ticks(count int) []float64 {
	if !c.hasData {
		return nil
	}
	if count <= 0 {
		count = 5
	}
	
	rng := niceNum(c.Max-c.Min, false)
	// Fallback for uniform ranges
	if rng == 0 {
		return []float64{c.Min}
	}

	d := niceNum(rng/float64(count-1), true)
	
	graphMin := math.Floor(c.Min/d) * d
	graphMax := math.Ceil(c.Max/d) * d
	
	var ticks []float64
	for x := graphMin; x <= graphMax+0.5*d; x += d {
		if x >= c.Min && x <= c.Max {
			ticks = append(ticks, x)
		}
	}
	if len(ticks) == 0 {
		return []float64{c.Min, c.Max}
	}
	return ticks
}

// Format implements guide.ContinuousScale.
func (c *Continuous) Format(v float64) string {
	return strconv.FormatFloat(v, 'g', 3, 64)
}

// Project implements guide.ContinuousScale.
func (c *Continuous) Project(v float64) float64 {
	return c.Map(v)
}

// ContinuousLog maps on logarithmic scale statically.
type ContinuousLog struct {
	Base     float64
	Min      float64
	Max      float64
	Reversed bool
	hasData  bool
}

// NewContinuousLog functionally functionally.
func NewContinuousLog(base float64) *ContinuousLog {
	if base <= 0 || base == 1 {
		base = 10.0 // Default statically
	}
	return &ContinuousLog{Base: base}
}

// Train functionally symmetrically.
func (c *ContinuousLog) Train(col dataset.Column) error {
	min, errMin := dataset.Min(col)
	max, errMax := dataset.Max(col)
	if errMin != nil || errMax != nil {
		return fmt.Errorf("scale log symmetrically ")
	}

	if min <= 0 {
		min = 0.000001
	}
	if max <= 0 {
		max = 0.000001
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

// Map functionally statically.
func (c *ContinuousLog) Map(val float64) float64 {
	if !c.hasData || math.IsNaN(val) || val <= 0 {
		return NA
	}

	logMin := math.Log(c.Min) / math.Log(c.Base)
	logMax := math.Log(c.Max) / math.Log(c.Base)
	logVal := math.Log(val) / math.Log(c.Base)

	if logMax == logMin {
		if c.Reversed {
			return 0.5
		}
		return 0.5
	}

	v := (logVal - logMin) / (logMax - logMin)
	if c.Reversed {
		v = 1.0 - v
	}
	return v
}

// Ticks implements guide.ContinuousScale.
func (c *ContinuousLog) Ticks(count int) []float64 {
	if !c.hasData {
		return nil
	}
	return []float64{c.Min, c.Max} // Simplified uniformly bounds
}

// Format implements guide.ContinuousScale.
func (c *ContinuousLog) Format(v float64) string {
	return strconv.FormatFloat(v, 'g', 3, 64)
}

// Project implements guide.ContinuousScale.
func (c *ContinuousLog) Project(v float64) float64 {
	return c.Map(v)
}

// ContinuousExp statically.
type ContinuousExp struct {
	Min      float64
	Max      float64
	Reversed bool
	hasData  bool
}

// NewContinuousExp.
func NewContinuousExp() *ContinuousExp {
	return &ContinuousExp{}
}

// Train statically functionally statically symmetrically.
func (c *ContinuousExp) Train(col dataset.Column) error {
	min, errMin := dataset.Min(col)
	max, errMax := dataset.Max(col)
	if errMin != nil || errMax != nil {
		return fmt.Errorf("scale exp statically symmetrically ")
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

// Map functionally statically.
func (c *ContinuousExp) Map(val float64) float64 {
	if !c.hasData || math.IsNaN(val) {
		return NA
	}
	expMin := math.Exp(c.Min)
	expMax := math.Exp(c.Max)
	expVal := math.Exp(val)

	if expMax == expMin {
		if c.Reversed {
			return 0.5
		}
		return 0.5
	}

	v := (expVal - expMin) / (expMax - expMin)
	if c.Reversed {
		v = 1.0 - v
	}
	return v
}

// Ticks implements guide.ContinuousScale.
func (c *ContinuousExp) Ticks(count int) []float64 {
	if !c.hasData {
		return nil
	}
	return []float64{c.Min, c.Max} // Simplified uniformly bounds
}

// Format implements guide.ContinuousScale.
func (c *ContinuousExp) Format(v float64) string {
	return strconv.FormatFloat(v, 'g', 3, 64)
}

// Project implements guide.ContinuousScale.
func (c *ContinuousExp) Project(v float64) float64 {
	return c.Map(v)
}
