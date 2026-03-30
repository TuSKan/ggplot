package layout

import (
	"testing"

	"github.com/TuSKan/ggplot/pkg/theme"
)

func BenchmarkLayoutCalculate_SinglePanel(b *testing.B) {
	cfg := Config{
		Width:     1024,
		Height:    768,
		Theme:     theme.Default(),
		HasTitle:  true,
		TitleText: "Benchmark Evaluation tightly ",
		HasXAxis:  true,
		XAxisParams: AxisParams{
			Title:      "Weight (lbs)",
			TickLabels: []string{"1000", "2000", "3000"},
		},
		HasYAxis: true,
		YAxisParams: AxisParams{
			Title:      "YA ",
			TickLabels: []string{"100", "200"},
		},
		FacetType: FacetNone,
	}

	measurer := DummyMeasurer{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Calculate(cfg, measurer)
	}
}
