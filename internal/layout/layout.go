package layout

import "github.com/TuSKan/ggplot/pkg/theme"

// Config defines layout calculation properties.
type Config struct {
	Width, Height float64
	Theme         theme.Theme

	HasTitle  bool
	TitleText string

	HasSubtitle  bool
	SubtitleText string

	HasXAxis    bool
	XAxisParams AxisParams

	HasYAxis    bool
	YAxisParams AxisParams

	HasLegend    bool
	LegendWidth  float64
	LegendHeight float64

	FacetType  FacetType
	FacetRows  int
	FacetCols  int
	FacetCount int
}

type AxisParams struct {
	Title      string
	TickLabels []string
}

// LayoutPlan yields deterministic box coordinates.
type LayoutPlan struct {
	OuterRect    Rect
	TitleRect    Rect
	SubtitleRect Rect
	XAxisRect    Rect
	YAxisRect    Rect
	LegendRect   Rect

	Panels []PanelLayout
}

// PanelLayout encapsulates single facet geometry mapping space.
type PanelLayout struct {
	PanelRect Rect // Outer panel padding including strips.
	DataRect  Rect // dataset projection!
}

// Calculate applies Guillotine division deriving exact box coordinates.
func Calculate(cfg Config, m TextMeasurer) LayoutPlan {
	if m == nil {
		m = DummyMeasurer{} // Fallback
	}

	plan := LayoutPlan{
		OuterRect: Rect{
			Min: Point{X: 0, Y: 0},
			Max: Point{X: cfg.Width, Y: cfg.Height},
		},
	}

	// Create working boundary cutting outer margins functionally.
	boundary := plan.OuterRect
	boundary.Min.X += cfg.Theme.Spacing.MarginLeft
	boundary.Min.Y += cfg.Theme.Spacing.MarginTop
	boundary.Max.X -= cfg.Theme.Spacing.MarginRight
	boundary.Max.Y -= cfg.Theme.Spacing.MarginBottom

	// Slice Title (Top)
	if cfg.HasTitle && cfg.TitleText != "" {
		_, h := m.MeasureText(cfg.TitleText, cfg.Theme.Typography.Title)
		plan.TitleRect = boundary.SliceTop(h)
	}

	// Slice Subtitle (Top - below title)
	if cfg.HasSubtitle && cfg.SubtitleText != "" {
		_, h := m.MeasureText(cfg.SubtitleText, cfg.Theme.Typography.Subtitle)
		plan.SubtitleRect = boundary.SliceTop(h)
	}

	// Slice Legend (Right)
	if cfg.HasLegend {
		plan.LegendRect = boundary.SliceRight(cfg.LegendWidth)
	}

	// Slice X-Axis (Bottom)
	if cfg.HasXAxis {
		totalXTitleH := 0.0
		if cfg.XAxisParams.Title != "" {
			_, h := m.MeasureText(cfg.XAxisParams.Title, cfg.Theme.Typography.AxisTitle)
			totalXTitleH += h
		}

		maxTickH := 0.0
		for _, label := range cfg.XAxisParams.TickLabels {
			_, h := m.MeasureText(label, cfg.Theme.Typography.TickLabel)
			if h > maxTickH {
				maxTickH = h
			}
		}

		plan.XAxisRect = boundary.SliceBottom(totalXTitleH + maxTickH + cfg.Theme.Ticks.Length)
	}

	// Slice Y-Axis (Left)
	if cfg.HasYAxis {
		totalYTitleW := 0.0
		if cfg.YAxisParams.Title != "" {
			w, _ := m.MeasureText(cfg.YAxisParams.Title, cfg.Theme.Typography.AxisTitle)
			totalYTitleW += w
		}

		maxTickW := 0.0
		for _, label := range cfg.YAxisParams.TickLabels {
			w, _ := m.MeasureText(label, cfg.Theme.Typography.TickLabel)
			if w > maxTickW {
				maxTickW = w
			}
		}

		plan.YAxisRect = boundary.SliceLeft(totalYTitleW + maxTickW + cfg.Theme.Ticks.Length)
	}

	// Remaining Boundary allocated to Facet generation
	panelRects := GeneratePanels(
		cfg.FacetType,
		boundary,
		cfg.FacetRows,
		cfg.FacetCols,
		cfg.FacetCount,
		cfg.Theme.Spacing.PanelSpacing,
	)

	// Panel Rect padding deduction.
	padding := 5.0
	for _, rect := range panelRects {
		dataBound := rect
		dataBound.Min.X += padding
		dataBound.Min.Y += padding
		dataBound.Max.X -= padding
		dataBound.Max.Y -= padding
		plan.Panels = append(plan.Panels, PanelLayout{
			PanelRect: rect,
			DataRect:  dataBound,
		})
	}

	return plan
}
