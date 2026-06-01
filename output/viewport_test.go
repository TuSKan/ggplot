package output

import (
	"context"
	"image"
	"math"
	"testing"

	"github.com/TuSKan/ggplot/canvas"
)

func TestPanelInfo_PixelToData(t *testing.T) {
	t.Parallel()

	pi := PanelInfo{
		Bounds: image.Rect(100, 50, 500, 350),
		XRange: [2]float64{0, 10},
		YRange: [2]float64{0, 100},
	}

	tests := []struct {
		name   string
		px, py float64
		wantDX float64
		wantDY float64
	}{
		{
			name: "top-left corner",
			px:   100, py: 50,
			wantDX: 0, wantDY: 100, // top = max Y (inverted)
		},
		{
			name: "bottom-right corner",
			px:   500, py: 350,
			wantDX: 10, wantDY: 0, // bottom = min Y
		},
		{
			name: "center",
			px:   300, py: 200,
			wantDX: 5, wantDY: 50,
		},
		{
			name: "quarter point",
			px:   200, py: 125,
			wantDX: 2.5, wantDY: 75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dx, dy := pi.PixelToData(tt.px, tt.py)
			if math.Abs(dx-tt.wantDX) > 1e-10 {
				t.Errorf("dx = %v, want %v", dx, tt.wantDX)
			}

			if math.Abs(dy-tt.wantDY) > 1e-10 {
				t.Errorf("dy = %v, want %v", dy, tt.wantDY)
			}
		})
	}
}

func TestPanelInfo_ContainsPixel(t *testing.T) {
	t.Parallel()

	pi := PanelInfo{
		Bounds: image.Rect(100, 50, 500, 350),
	}

	tests := []struct {
		name string
		px   float64
		py   float64
		want bool
	}{
		{"inside", 300, 200, true},
		{"top-left edge", 100, 50, true},
		{"bottom-right edge", 500, 350, false}, // exclusive upper bound
		{"outside left", 99, 200, false},
		{"outside top", 300, 49, false},
		{"outside right", 501, 200, false},
		{"outside bottom", 300, 351, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := pi.ContainsPixel(tt.px, tt.py); got != tt.want {
				t.Errorf("ContainsPixel(%v, %v) = %v, want %v", tt.px, tt.py, got, tt.want)
			}
		})
	}
}

func TestDataSpaceController_Zoom(t *testing.T) {
	t.Parallel()

	ctrl := DataSpaceController()

	// Create a mock figure that implements Measurable + Zoomable.
	fig := &mockZoomFigure{
		panels: []PanelInfo{
			{
				Index:      0,
				Bounds:     image.Rect(50, 50, 450, 350),
				XRange:     [2]float64{0, 100},
				YRange:     [2]float64{0, 200},
				OrigXRange: [2]float64{0, 100},
				OrigYRange: [2]float64{0, 200},
			},
		},
	}

	st := &State{
		Bounds: image.Rect(0, 0, 500, 400),
		Scale:  1,
		Figure: fig,
	}

	// Scroll down at center → should zoom in.
	action := ctrl.OnEvent(Event{
		Kind: EventScroll,
		X:    250, Y: 200, // center of panel
		DY: -1, // scroll down = zoom in
	}, st)

	if action != ActionRedraw {
		t.Fatalf("expected ActionRedraw, got %v", action)
	}

	// Verify zoom occurred — range should be smaller.
	xRange := fig.panels[0].XRange[1] - fig.panels[0].XRange[0]
	if xRange >= 100 {
		t.Errorf("X range should shrink after zoom in, got %v", xRange)
	}
}

func TestDataSpaceController_Pan(t *testing.T) {
	t.Parallel()

	ctrl := DataSpaceController()

	fig := &mockZoomFigure{
		panels: []PanelInfo{
			{
				Index:      0,
				Bounds:     image.Rect(50, 50, 450, 350),
				XRange:     [2]float64{0, 100},
				YRange:     [2]float64{0, 200},
				OrigXRange: [2]float64{0, 100},
				OrigYRange: [2]float64{0, 200},
			},
		},
	}

	st := &State{
		Bounds: image.Rect(0, 0, 500, 400),
		Scale:  1,
		Figure: fig,
	}

	// Mouse down in panel.
	ctrl.OnEvent(Event{Kind: EventPointerDown, X: 200, Y: 200}, st)

	// Drag right by 40 pixels — should shift X range left.
	action := ctrl.OnEvent(Event{Kind: EventPointerMove, X: 240, Y: 200}, st)
	if action != ActionRedraw {
		t.Fatalf("expected ActionRedraw on drag, got %v", action)
	}

	// XRange should shift left (negative dx because dragging right moves data left).
	if fig.panels[0].XRange[0] >= 0 {
		t.Errorf("X range min should decrease after dragging right, got %v", fig.panels[0].XRange[0])
	}

	// Mouse up.
	ctrl.OnEvent(Event{Kind: EventPointerUp, X: 240, Y: 200}, st)
}

func TestDataSpaceController_DoubleClick_Reset(t *testing.T) {
	t.Parallel()

	ctrl := DataSpaceController()

	fig := &mockZoomFigure{
		panels: []PanelInfo{
			{
				Index:      0,
				Bounds:     image.Rect(50, 50, 450, 350),
				XRange:     [2]float64{20, 80}, // zoomed in
				YRange:     [2]float64{50, 150},
				OrigXRange: [2]float64{0, 100},
				OrigYRange: [2]float64{0, 200},
			},
		},
	}

	st := &State{
		Bounds: image.Rect(0, 0, 500, 400),
		Scale:  1,
		Figure: fig,
	}

	action := ctrl.OnEvent(Event{Kind: EventDoubleClick, X: 250, Y: 200}, st)
	if action != ActionRedraw {
		t.Fatalf("expected ActionRedraw on double-click, got %v", action)
	}

	if !fig.resetCalled {
		t.Error("ResetViewport should have been called")
	}
}

func TestDataSpaceController_ClickOutsidePanel(t *testing.T) {
	t.Parallel()

	ctrl := DataSpaceController()

	fig := &mockZoomFigure{
		panels: []PanelInfo{
			{
				Index:  0,
				Bounds: image.Rect(50, 50, 450, 350),
			},
		},
	}

	st := &State{
		Bounds: image.Rect(0, 0, 500, 400),
		Scale:  1,
		Figure: fig,
	}

	// Click outside the panel — should not start drag.
	ctrl.OnEvent(Event{Kind: EventPointerDown, X: 10, Y: 10}, st)

	action := ctrl.OnEvent(Event{Kind: EventPointerMove, X: 50, Y: 50}, st)
	if action != ActionIgnore {
		t.Errorf("drag outside panel should be ignored, got %v", action)
	}
}

func TestDataSpaceController_ScrollOutsidePanel(t *testing.T) {
	t.Parallel()

	ctrl := DataSpaceController()

	fig := &mockZoomFigure{
		panels: []PanelInfo{
			{
				Index:  0,
				Bounds: image.Rect(50, 50, 450, 350),
			},
		},
	}

	st := &State{
		Bounds: image.Rect(0, 0, 500, 400),
		Scale:  1,
		Figure: fig,
	}

	// Scroll outside the panel — should be ignored.
	action := ctrl.OnEvent(Event{Kind: EventScroll, X: 10, Y: 10, DY: -1}, st)
	if action != ActionIgnore {
		t.Errorf("scroll outside panel should be ignored, got %v", action)
	}
}

// --- Mock figure for testing ---

type mockZoomFigure struct {
	panels      []PanelInfo
	resetCalled bool
}

func (m *mockZoomFigure) Draw(_ context.Context, _ canvas.Canvas, _, _ int) error { return nil }

func (m *mockZoomFigure) PanelInfos() []PanelInfo { return m.panels }

func (m *mockZoomFigure) SetPanelViewport(idx int, xlim, ylim [2]*float64) {
	if idx < 0 || idx >= len(m.panels) {
		return
	}

	if xlim[0] != nil {
		m.panels[idx].XRange[0] = *xlim[0]
	}

	if xlim[1] != nil {
		m.panels[idx].XRange[1] = *xlim[1]
	}

	if ylim[0] != nil {
		m.panels[idx].YRange[0] = *ylim[0]
	}

	if ylim[1] != nil {
		m.panels[idx].YRange[1] = *ylim[1]
	}
}

func (m *mockZoomFigure) ResetViewport() {
	m.resetCalled = true

	for i := range m.panels {
		m.panels[i].XRange = m.panels[i].OrigXRange
		m.panels[i].YRange = m.panels[i].OrigYRange
	}
}
