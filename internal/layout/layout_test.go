package layout

import (
	"reflect"
	"testing"
)

func TestSinglePanelLayout(t *testing.T) {
	cfg := Config{
		Width: 1000, Height: 1000,
		Margin:    Margin{Top: 50, Right: 50, Bottom: 50, Left: 50},
		HasTitle:  true,
		TitleText: "Scatter Plot",
		TitleSize: 20, // Dummy measurer -> height: 30
		HasXAxis:  true,
		XAxisParams: AxisParams{
			Title:         "X Axis",
			TitleSize:     14, // height: 21
			TickLabels:    []string{"1", "2"},
			TickLabelSize: 10, // max h: 15
		}, 
		// Total X-Axis sliced bottom = 36
		HasYAxis:  true,
		YAxisParams: AxisParams{
			Title: "Y Axis",
			TitleSize: 14, // width: 8.5 * len("Y Axis") -> len 6 * 7
			TickLabels: []string{"100", "200"},
			TickLabelSize: 10, // width: 3 * 5 = 15
		},
		FacetType: FacetNone,
	}

	plan := Calculate(cfg, DummyMeasurer{})
	
	// Golden Expected Geometry Mapping
	expectedDataRect := Rect{
		Min: Point{X: 50 + (6*7) + 15, Y: 50 + 30}, // 50 margin + 42 Y title + 15 ticks = 107
		Max: Point{X: 950, Y: 950 - 36}, // 950 margin - 36 X title/ticks = 914
	}

	if len(plan.Panels) != 1 {
		t.Fatalf("expected 1 single panel, got %d", len(plan.Panels))
	}
	
	derived := plan.Panels[0].DataRect
	if !reflect.DeepEqual(derived, expectedDataRect) {
		t.Errorf("Golden layout geometry mismatch.\nExpected: %+v\nGot: %+v", expectedDataRect, derived)
	}
}
