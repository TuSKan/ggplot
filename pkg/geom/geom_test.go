package geom_test

import (
	"testing"

	"github.com/TuSKan/ggplot/pkg/geom"
)

func TestGeomBuildersCompile(t *testing.T) {
	// Simple tests mapping builders
	builders := []struct {
		name string
		b    interface{ Name() string }
	}{
		{"Point", geom.PointGeom{}},
		{"Line", geom.LineGeom{}},
		{"Bar", geom.BarGeom{}},
		{"Area", geom.AreaGeom{}},
		{"Polygon", geom.PolygonGeom{}},
		{"Histogram", geom.HistogramGeom{}},
	}

	for _, tc := range builders {
		t.Run(tc.name, func(t *testing.T) {
			if tc.b.Name() != tc.name {
				t.Errorf("expected %s, got %s", tc.name, tc.b.Name())
			}
		})
	}
}

func TestRequiredAesthetics(t *testing.T) {
	tests := []struct {
		geom geom.PointGeom
		req  []string
	}{
		{geom.PointGeom{}, []string{"x", "y"}},
	}
	for _, tc := range tests {
		got := tc.geom.RequiredAesthetics()
		for i, v := range got {
			if v != tc.req[i] {
				t.Errorf("missing aesthetic: got %s, want %s", v, tc.req[i])
			}
		}
	}
}
