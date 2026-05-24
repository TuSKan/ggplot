package scale

import (
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// LinetypeScale maps discrete (categorical) data values to dash patterns.
type LinetypeScale struct {
	DiscreteScale

	manual map[string]string
}

// NewLinetype returns a new LinetypeScale.
func NewLinetype() *LinetypeScale {
	return &LinetypeScale{}
}

// NewLinetypeManual returns a manual LinetypeScale with a custom mapping.
func NewLinetypeManual(m map[string]string) *LinetypeScale {
	return &LinetypeScale{
		manual: maps.Clone(m),
	}
}

// Default linetypes in cycle order
var defaultLinetypes = []string{"solid", "dashed", "dotted", "dotdash", "longdash", "twodash"}

// Dash patterns matching typical drawing libraries
var linetypePatterns = map[string][]float64{
	"solid":    nil,
	"dashed":   {6, 6},
	"dotted":   {2, 2},
	"dotdash":  {2, 4, 6, 4},
	"longdash": {12, 6},
	"twodash":  {2, 2, 6, 2},
}

// Train trains the discrete categories.
func (s *LinetypeScale) Train(col dataset.AnyColumn) error {
	return s.DiscreteScale.Train(col)
}

// LinetypeName resolves a category label to a linetype name (e.g. "dashed").
func (s *LinetypeScale) LinetypeName(label string) string {
	if s.manual != nil {
		if lt, ok := s.manual[label]; ok {
			return lt
		}
	}

	if len(s.categories) == 0 {
		// Fallback for untrained or single values: if the label itself is a known linetype, return it.
		if _, ok := linetypePatterns[label]; ok {
			return label
		}

		return "solid"
	}

	return defaultLinetypes[max(0, int(s.MapCategory(label)))%len(defaultLinetypes)]
}

// DashPattern returns the []float64 dash array for the given category.
func (s *LinetypeScale) DashPattern(label string) []float64 {
	name := s.LinetypeName(label)
	return linetypePatterns[name]
}

// String returns "linetype".
func (s *LinetypeScale) String() string {
	return "linetype"
}

// Verify interface compliance
var _ Scale = (*LinetypeScale)(nil)
