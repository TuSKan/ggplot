package geom

import (
	"testing"
)

func TestGeometries(t *testing.T) {
	// Simple test simulating compilation with LayerContext bounds.
	ctx := &LayerContext{
		Mask: []int{0, 2, 4}, // Mock active subset
	}

	geometries := []Geometry{
		&Point{},
		&Line{},
		&Bar{},
		&Area{},
		&Polygon{},
		&Histogram{},
	}

	for _, g := range geometries {
		if err := g.Compile(ctx); err != nil {
			t.Errorf("geom Compilation failed for %T: %v", g, err)
		}
	}
}
