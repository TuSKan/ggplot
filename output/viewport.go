package output

import "image"

// PanelInfo describes the pixel geometry and data-space bounds of one panel.
// Returned by [Measurable] after a Draw call so that controllers can convert
// pixel positions to data coordinates and perform per-panel hit-testing.
type PanelInfo struct {
	// Index is the zero-based panel index in the figure's panel slice.
	Index int

	// Bounds is the data-area rectangle in figure pixel coordinates.
	// This is the region where data is drawn — axes/titles are outside.
	Bounds image.Rectangle

	// XRange is the current (possibly zoomed) X data-space bounds [min, max].
	XRange [2]float64
	// YRange is the current (possibly zoomed) Y data-space bounds [min, max].
	YRange [2]float64

	// OrigXRange is the full trained X data-space bounds (for reset).
	OrigXRange [2]float64
	// OrigYRange is the full trained Y data-space bounds (for reset).
	OrigYRange [2]float64
}

// PixelToData converts a pixel position (px, py) within this panel's Bounds
// to data-space coordinates. Returns (NaN, NaN) if the pixel is outside the
// panel bounds.
func (p PanelInfo) PixelToData(px, py float64) (dx, dy float64) {
	b := p.Bounds
	// Fractional position within the panel [0,1].
	fx := (px - float64(b.Min.X)) / float64(b.Dx())
	fy := (py - float64(b.Min.Y)) / float64(b.Dy())

	// X: left→right maps to XRange[0]→XRange[1].
	dx = p.XRange[0] + fx*(p.XRange[1]-p.XRange[0])
	// Y: top→bottom maps to YRange[1]→YRange[0] (screen Y is inverted).
	dy = p.YRange[1] - fy*(p.YRange[1]-p.YRange[0])

	return dx, dy
}

// ContainsPixel reports whether the pixel position (px, py) falls within
// this panel's data-area bounds.
func (p PanelInfo) ContainsPixel(px, py float64) bool {
	return px >= float64(p.Bounds.Min.X) && px < float64(p.Bounds.Max.X) &&
		py >= float64(p.Bounds.Min.Y) && py < float64(p.Bounds.Max.Y)
}

// Measurable is an optional [Figure] extension that exposes per-panel layout
// geometry after a Draw call. Controllers use this to convert pixel mouse
// positions to data coordinates for data-space pan/zoom.
//
// The returned PanelInfos are valid only for the width×height that was last
// passed to Draw. A resize triggers a new Draw, which updates the geometry.
type Measurable interface {
	// PanelInfos returns the geometry of all panels from the last Draw call.
	// Returns nil if Draw has not been called yet.
	PanelInfos() []PanelInfo
}

// Zoomable is an optional [Figure] extension that supports fast viewport
// changes without rebuilding the figure from source. The controller mutates
// scale bounds directly on the built figure, then triggers [ActionRedraw].
//
// This is O(1) — no data iteration, no stat recomputation, no memory
// allocation. Only the scale endpoints change; ticks regenerate on the
// next Draw call.
type Zoomable interface {
	// SetPanelViewport overrides the X and Y scale bounds for the panel at
	// panelIndex. Pass nil for either limit endpoint to keep the current
	// value. After calling this, the next Draw will render with the new
	// viewport (clipped data, updated ticks).
	SetPanelViewport(panelIndex int, xlim, ylim [2]*float64)

	// ResetViewport restores all panels to their original trained bounds.
	ResetViewport()
}
