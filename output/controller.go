package output

import "math"

// zoomStep is the per-scroll-notch zoom fraction for data-space zoom.
const dataZoomStep = 0.1

// DataSpaceController provides data-space pan (drag) and zoom (scroll wheel)
// that operates on scale bounds rather than a canvas-level affine transform.
//
// Axes remain at fixed screen positions; only the data visible within the
// panel changes. Tick labels update dynamically to reflect the current
// viewport. This is the interactive behavior expected in professional
// plotting tools (matplotlib, Plotly, ggplot2 shiny).
//
// Performance: each interaction mutates two float64 values per axis (O(1)),
// then triggers [ActionRedraw]. No rebuild, no data iteration.
//
// For faceted plots, zoom/pan applies only to the panel under the cursor.
func DataSpaceController() Controller { return &dataSpaceController{} }

type dataSpaceController struct {
	dragging     bool
	lastX, lastY float64

	// activePanel is the panel index where the current drag started.
	// -1 means no panel is active.
	activePanel int
}

var _ Controller = (*dataSpaceController)(nil)

func (c *dataSpaceController) OnEvent(ev Event, st *State) Action {
	switch ev.Kind {
	case EventPointerDown:
		c.lastX, c.lastY = ev.X, ev.Y

		c.activePanel = c.findPanel(ev.X, ev.Y, st)
		if c.activePanel >= 0 {
			c.dragging = true
		}

		return ActionIgnore

	case EventPointerUp:
		c.dragging = false
		c.activePanel = -1

		return ActionIgnore

	case EventPointerMove:
		if !c.dragging || c.activePanel < 0 {
			return ActionIgnore
		}

		c.pan(ev, st)

		return ActionRedraw

	case EventScroll:
		panel := c.findPanel(ev.X, ev.Y, st)
		if panel < 0 {
			return ActionIgnore
		}

		c.zoom(ev, st, panel)

		return ActionRedraw

	case EventDoubleClick:
		c.resetViewport(st)

		return ActionRedraw

	case EventResize:
		return ActionRedraw

	case EventClose:
		return ActionClose

	case EventKey:
		return ActionIgnore

	default:
		return ActionIgnore
	}
}

// findPanel returns the panel index under the cursor, or -1 if outside
// all panels. Uses PanelInfos from the Measurable interface.
func (c *dataSpaceController) findPanel(px, py float64, st *State) int {
	m, ok := st.Figure.(Measurable)
	if !ok {
		return 0 // non-measurable figure — assume single panel
	}

	panels := m.PanelInfos()
	for _, p := range panels {
		if p.ContainsPixel(px, py) {
			return p.Index
		}
	}

	return -1
}

// panelInfo returns the PanelInfo for the given index, or nil.
func (c *dataSpaceController) panelInfo(st *State, idx int) *PanelInfo {
	m, ok := st.Figure.(Measurable)
	if !ok {
		return nil
	}

	panels := m.PanelInfos()
	if idx < 0 || idx >= len(panels) {
		return nil
	}

	return &panels[idx]
}

// pan shifts the data-space viewport by the mouse drag delta.
func (c *dataSpaceController) pan(ev Event, st *State) {
	pi := c.panelInfo(st, c.activePanel)
	if pi == nil {
		return
	}

	bw := float64(pi.Bounds.Dx())

	bh := float64(pi.Bounds.Dy())
	if bw <= 0 || bh <= 0 {
		return
	}

	// Pixel delta → data-space delta.
	xRange := pi.XRange[1] - pi.XRange[0]
	yRange := pi.YRange[1] - pi.YRange[0]
	ddx := -(ev.X - c.lastX) / bw * xRange
	ddy := (ev.Y - c.lastY) / bh * yRange // screen Y is inverted

	c.lastX, c.lastY = ev.X, ev.Y

	// Shift the viewport.
	newXMin := pi.XRange[0] + ddx
	newXMax := pi.XRange[1] + ddx
	newYMin := pi.YRange[0] + ddy
	newYMax := pi.YRange[1] + ddy

	z, ok := st.Figure.(Zoomable)
	if !ok {
		return
	}

	z.SetPanelViewport(c.activePanel,
		[2]*float64{&newXMin, &newXMax},
		[2]*float64{&newYMin, &newYMax},
	)
}

// zoom adjusts the data-space viewport around the cursor position.
func (c *dataSpaceController) zoom(ev Event, st *State, panel int) {
	pi := c.panelInfo(st, panel)
	if pi == nil {
		return
	}

	bw := float64(pi.Bounds.Dx())

	bh := float64(pi.Bounds.Dy())
	if bw <= 0 || bh <= 0 {
		return
	}

	// Zoom factor: scroll down (DY < 0) zooms in, scroll up zooms out.
	f := 1 + dataZoomStep
	if ev.DY < 0 {
		f = 1 / (1 + dataZoomStep)
	}

	// Cursor position in data space.
	cursorDX, cursorDY := pi.PixelToData(ev.X, ev.Y)

	// Zoom around cursor: keep the cursor data point fixed.
	xRange := pi.XRange[1] - pi.XRange[0]
	yRange := pi.YRange[1] - pi.YRange[0]

	// Fractional cursor position within the current range.
	fx := (cursorDX - pi.XRange[0]) / xRange
	fy := (cursorDY - pi.YRange[0]) / yRange

	newXRange := xRange * f
	newYRange := yRange * f

	// Prevent extreme zoom (minimum 1e-12 range, maximum 1e12 range).
	if math.Abs(newXRange) < 1e-12 || math.Abs(newYRange) < 1e-12 { //nolint:mnd // Minimum zoom limit.
		return
	}

	if math.Abs(newXRange) > 1e12 || math.Abs(newYRange) > 1e12 { //nolint:mnd // Maximum zoom limit.
		return
	}

	newXMin := cursorDX - fx*newXRange
	newXMax := cursorDX + (1-fx)*newXRange
	newYMin := cursorDY - fy*newYRange
	newYMax := cursorDY + (1-fy)*newYRange

	z, ok := st.Figure.(Zoomable)
	if !ok {
		return
	}

	z.SetPanelViewport(panel,
		[2]*float64{&newXMin, &newXMax},
		[2]*float64{&newYMin, &newYMax},
	)
}

// resetViewport restores all panels to their original trained bounds.
func (c *dataSpaceController) resetViewport(st *State) {
	z, ok := st.Figure.(Zoomable)
	if !ok {
		return
	}

	z.ResetViewport()
}
