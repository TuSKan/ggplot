package ggplot

import (
	"image"
	"sync"

	"github.com/TuSKan/ggplot/output"
	"github.com/TuSKan/ggplot/scale"
)

// panelGeometry holds the pixel-space layout of one panel's data area,
// populated during Built.Draw and cached for the controller.
type panelGeometry struct {
	dataX, dataY   float64 // top-left corner of data area in figure coords
	panelW, panelH float64 // data area dimensions
}

// viewportState holds per-panel original trained bounds for Reset,
// and the mutex protecting concurrent access from the draw and event
// goroutines.
type viewportState struct {
	mu sync.Mutex

	// origBounds stores the original trained scale bounds per panel,
	// captured once during the first viewport operation.
	origBounds []panelOrigBounds

	// lastGeom caches per-panel pixel geometry from the most recent Draw.
	lastGeom []panelGeometry

	// lastWidth, lastHeight record the figure size from the last Draw.
	lastWidth, lastHeight int
}

// panelOrigBounds stores the original trained X/Y scale bounds for one panel.
type panelOrigBounds struct {
	xMin, xMax float64
	yMin, yMax float64
}

// --- Measurable interface ---

// PanelInfos returns the per-panel geometry and data-space bounds from the
// most recent Draw call. Returns nil if Draw has not been called.
//
// The controller uses this to convert pixel mouse positions to data
// coordinates and to perform per-panel hit-testing.
func (b *Built) PanelInfos() []output.PanelInfo {
	b.vp.mu.Lock()
	defer b.vp.mu.Unlock()

	if len(b.vp.lastGeom) == 0 {
		return nil
	}

	infos := make([]output.PanelInfo, len(b.panels))
	for i, bp := range b.panels {
		var g panelGeometry
		if i < len(b.vp.lastGeom) {
			g = b.vp.lastGeom[i]
		}

		xMin, xMax := bp.XScale.Bounds()
		yMin, yMax := bp.YScale.Bounds()

		info := output.PanelInfo{
			Index:  i,
			Bounds: image.Rect(int(g.dataX), int(g.dataY), int(g.dataX+g.panelW), int(g.dataY+g.panelH)),
			XRange: [2]float64{xMin, xMax},
			YRange: [2]float64{yMin, yMax},
		}

		// Fill original bounds if captured.
		if i < len(b.vp.origBounds) {
			ob := b.vp.origBounds[i]
			info.OrigXRange = [2]float64{ob.xMin, ob.xMax}
			info.OrigYRange = [2]float64{ob.yMin, ob.yMax}
		} else {
			// Not yet captured — current bounds are original.
			info.OrigXRange = info.XRange
			info.OrigYRange = info.YRange
		}

		infos[i] = info
	}

	return infos
}

// --- Zoomable interface ---

// SetPanelViewport overrides the X and Y scale bounds for the panel at
// panelIndex. Pass nil for either limit endpoint to keep the current value.
//
// This is O(1): it mutates only two float64 values per axis. The next Draw
// call renders with the new viewport — clipped data, updated tick labels,
// axes at fixed positions.
func (b *Built) SetPanelViewport(panelIndex int, xlim, ylim [2]*float64) {
	if panelIndex < 0 || panelIndex >= len(b.panels) {
		return
	}

	b.vp.mu.Lock()
	defer b.vp.mu.Unlock()

	// Capture original bounds on first viewport mutation.
	b.captureOrigBoundsLocked()

	bp := &b.panels[panelIndex]

	if xlim[0] != nil || xlim[1] != nil {
		curXMin, curXMax := bp.XScale.Bounds()
		if xlim[0] != nil {
			curXMin = *xlim[0]
		}

		if xlim[1] != nil {
			curXMax = *xlim[1]
		}

		if bs, ok := bp.XScale.(scale.BoundsSetter); ok {
			bs.SetBounds(curXMin, curXMax)
		}
	}

	if ylim[0] != nil || ylim[1] != nil {
		curYMin, curYMax := bp.YScale.Bounds()
		if ylim[0] != nil {
			curYMin = *ylim[0]
		}

		if ylim[1] != nil {
			curYMax = *ylim[1]
		}

		if bs, ok := bp.YScale.(scale.BoundsSetter); ok {
			bs.SetBounds(curYMin, curYMax)
		}
	}
}

// ResetViewport restores all panels to their original trained bounds.
func (b *Built) ResetViewport() {
	b.vp.mu.Lock()
	defer b.vp.mu.Unlock()

	if len(b.vp.origBounds) == 0 {
		return // never zoomed
	}

	for i, ob := range b.vp.origBounds {
		if i >= len(b.panels) {
			break
		}

		bp := &b.panels[i]

		if bs, ok := bp.XScale.(scale.BoundsSetter); ok {
			bs.SetBounds(ob.xMin, ob.xMax)
		}

		if bs, ok := bp.YScale.(scale.BoundsSetter); ok {
			bs.SetBounds(ob.yMin, ob.yMax)
		}
	}
}

// captureOrigBoundsLocked stores the current scale bounds of all panels as
// the "original" (pre-zoom) state. Called once on the first viewport mutation.
// Must be called with vp.mu held.
func (b *Built) captureOrigBoundsLocked() {
	if len(b.vp.origBounds) > 0 {
		return // already captured
	}

	b.vp.origBounds = make([]panelOrigBounds, len(b.panels))
	for i, bp := range b.panels {
		xMin, xMax := bp.XScale.Bounds()
		yMin, yMax := bp.YScale.Bounds()
		b.vp.origBounds[i] = panelOrigBounds{xMin: xMin, xMax: xMax, yMin: yMin, yMax: yMax}
	}
}

// storePanelGeometry caches the pixel-space geometry of all panels after
// a Draw call. Called from within Built.Draw with the computed layout values.
func (b *Built) storePanelGeometry(width, height int, geoms []panelGeometry) {
	b.vp.mu.Lock()
	defer b.vp.mu.Unlock()

	b.vp.lastGeom = geoms
	b.vp.lastWidth = width
	b.vp.lastHeight = height
}

// Compile-time interface checks.
var (
	_ output.Measurable = (*Built)(nil)
	_ output.Zoomable   = (*Built)(nil)
)
