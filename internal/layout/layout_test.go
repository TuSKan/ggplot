package layout

import (
	"reflect"
	"testing"

	"github.com/TuSKan/ggplot/pkg/theme"
)

func TestSinglePanelLayout(t *testing.T) {
	th := theme.Default()
	th.Spacing.MarginTop = 50
	th.Spacing.MarginRight = 50
	th.Spacing.MarginBottom = 50
	th.Spacing.MarginLeft = 50
	th.Typography.Title.Size = 20
	// height: 30

	cfg := Config{
		Width: 1000, Height: 1000,
		Theme:     th,
		HasTitle:  true,
		TitleText: "Scatter Plot",
		HasXAxis:  true,
		XAxisParams: AxisParams{
			Title:      "Horsepower",
			TickLabels: []string{"50", "100", "150", "200"},
		},
		HasYAxis: true,
		YAxisParams: AxisParams{
			Title:      "YA",
			TickLabels: []string{"100", "200"},
		},
		FacetType: FacetNone,
	}

	plan := Calculate(cfg, DummyMeasurer{})

	// Title Height = 30
	// Y-axis slice = TitleW (14) + maxTickW (15) + Ticks.Length(4) = 33
	// X-axis slice = TitleH (21) + maxTickH (15) + Ticks.Length(4) = 40
	// Padding inside PanelLayout = 5

	expectedDataRect := Rect{
		Min: Point{X: 86, Y: 85},
		Max: Point{X: 945, Y: 908},
	}

	if len(plan.Panels) != 1 {
		t.Fatalf("expected 1 single panel, got %d", len(plan.Panels))
	}

	derived := plan.Panels[0].DataRect
	if !reflect.DeepEqual(derived, expectedDataRect) {
		t.Errorf("Golden layout geometry mismatch.\nExpected: %+v\nGot: %+v", expectedDataRect, derived)
	}
}
