package scale

import (
	"maps"
	"slices"

	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/dataset"
)

// ShapeScale maps discrete (categorical) data values to shape names.
type ShapeScale struct {
	DiscreteScale

	manual map[string]string
}

// NewShape returns a new ShapeScale.
func NewShape() *ShapeScale {
	return &ShapeScale{}
}

// NewShapeManual returns a manual ShapeScale with custom mapping.
func NewShapeManual(m map[string]string) *ShapeScale {
	return &ShapeScale{
		manual: maps.Clone(m),
	}
}

// Train trains the discrete categories.
func (s *ShapeScale) Train(col dataset.AnyColumn) error {
	return s.DiscreteScale.Train(col)
}

// ShapeName maps a category label to a shape name (e.g. "circle", "square").
func (s *ShapeScale) ShapeName(label string) string {
	if s.manual != nil {
		if sh, ok := s.manual[label]; ok {
			return sh
		}
	}

	shapes := canvas.Shapes()

	if len(s.categories) == 0 {
		// Fallback for untrained or single values: if the label itself is a known shape, return it.
		if slices.Contains(shapes, label) {
			return label
		}

		return canvas.ShapeCircle
	}

	return shapes[max(0, int(s.MapCategory(label)))%len(shapes)]
}

// String returns "shape".
func (s *ShapeScale) String() string {
	return "shape"
}

// Verify interface compliance
var _ Scale = (*ShapeScale)(nil)
