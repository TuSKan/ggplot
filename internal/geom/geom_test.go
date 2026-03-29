package geom

import (
	"testing"
)

func TestGeometries(t *testing.T) {
	var _ Geometry = nil
	_ = Point{}
	_ = Line{}
	_ = Bar{}
	_ = Polygon{}
}
