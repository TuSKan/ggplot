package layout

// Config defines layout calculation properties.
type Config struct {
	Width, Height float64
	Margin        Margin

	HasTitle  bool
	TitleText string
	TitleSize float64

	HasSubtitle  bool
	SubtitleText string
	SubtitleSize float64

	HasXAxis    bool
	XAxisParams AxisParams

	HasYAxis    bool
	YAxisParams AxisParams

	HasLegend    bool
	LegendWidth  float64 // Assumed pre-calculated or fixed width
	LegendHeight float64

	FacetType  FacetType
	FacetRows  int
	FacetCols  int
	FacetCount int
	Spacing    float64
}

// AxisParams specifies title and tick requirements logically.
type AxisParams struct {
	Title         string
	TitleSize     float64
	TickLabels    []string
	TickLabelSize float64
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
	DataRect Rect
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
	
	// Create working boundary cutting outer margins.
	boundary := plan.OuterRect
	boundary.Min.X += cfg.Margin.Left
	boundary.Min.Y += cfg.Margin.Top
	boundary.Max.X -= cfg.Margin.Right
	boundary.Max.Y -= cfg.Margin.Bottom

	// Slice Title (Top)
	if cfg.HasTitle && cfg.TitleText != "" {
		_, h := m.MeasureText(cfg.TitleText, cfg.TitleSize)
		plan.TitleRect = boundary.SliceTop(h)
	}

	// Slice Subtitle (Top - below title)
	if cfg.HasSubtitle && cfg.SubtitleText != "" {
		_, h := m.MeasureText(cfg.SubtitleText, cfg.SubtitleSize)
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
			_, h := m.MeasureText(cfg.XAxisParams.Title, cfg.XAxisParams.TitleSize)
			totalXTitleH += h
		}
		
		maxTickH := 0.0
		for _, label := range cfg.XAxisParams.TickLabels {
			_, h := m.MeasureText(label, cfg.XAxisParams.TickLabelSize)
			if h > maxTickH { maxTickH = h }
		}

		plan.XAxisRect = boundary.SliceBottom(totalXTitleH + maxTickH)
	}

	// Slice Y-Axis (Left)
	if cfg.HasYAxis {
		totalYTitleW := 0.0
		if cfg.YAxisParams.Title != "" {
			w, _ := m.MeasureText(cfg.YAxisParams.Title, cfg.YAxisParams.TitleSize)
			totalYTitleW += w
		}

		maxTickW := 0.0
		for _, label := range cfg.YAxisParams.TickLabels {
			w, _ := m.MeasureText(label, cfg.YAxisParams.TickLabelSize)
			if w > maxTickW { maxTickW = w }
		}

		plan.YAxisRect = boundary.SliceLeft(totalYTitleW + maxTickW)
	}

	// Remaining Boundary allocated to Facet generation
	panelRects := GeneratePanels(
		cfg.FacetType,
		boundary,
		cfg.FacetRows,
		cfg.FacetCols,
		cfg.FacetCount,
		cfg.Spacing,
	)

	for _, rect := range panelRects {
		plan.Panels = append(plan.Panels, PanelLayout{DataRect: rect})
	}

	return plan
}
