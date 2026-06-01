package ggplot

import (
	"testing"

	"github.com/TuSKan/ggplot/output"
	"github.com/TuSKan/ggplot/scale"
)

func TestBuilt_Measurable_NoDrawYet(t *testing.T) {
	t.Parallel()

	b := &Built{
		panels: []BuiltPanel{
			{XScale: newTrainedLinear(0, 10), YScale: newTrainedLinear(0, 100)},
		},
	}

	infos := b.PanelInfos()
	if infos != nil {
		t.Errorf("PanelInfos before Draw should return nil, got %v", infos)
	}
}

func TestBuilt_Zoomable_SetPanelViewport(t *testing.T) {
	t.Parallel()

	b := &Built{
		panels: []BuiltPanel{
			{XScale: newTrainedLinear(0, 10), YScale: newTrainedLinear(0, 100)},
		},
	}

	xmin, xmax := 2.0, 8.0
	ymin, ymax := 20.0, 80.0
	b.SetPanelViewport(0, [2]*float64{&xmin, &xmax}, [2]*float64{&ymin, &ymax})

	gotXMin, gotXMax := b.panels[0].XScale.Bounds()
	if gotXMin != 2 || gotXMax != 8 {
		t.Errorf("X bounds = [%v, %v], want [2, 8]", gotXMin, gotXMax)
	}

	gotYMin, gotYMax := b.panels[0].YScale.Bounds()
	if gotYMin != 20 || gotYMax != 80 {
		t.Errorf("Y bounds = [%v, %v], want [20, 80]", gotYMin, gotYMax)
	}
}

func TestBuilt_Zoomable_ResetViewport(t *testing.T) {
	t.Parallel()

	b := &Built{
		panels: []BuiltPanel{
			{XScale: newTrainedLinear(0, 10), YScale: newTrainedLinear(0, 100)},
		},
	}

	// Zoom in.
	xmin, xmax := 2.0, 8.0
	ymin, ymax := 20.0, 80.0
	b.SetPanelViewport(0, [2]*float64{&xmin, &xmax}, [2]*float64{&ymin, &ymax})

	// Reset.
	b.ResetViewport()

	gotXMin, gotXMax := b.panels[0].XScale.Bounds()
	if gotXMin != 0 || gotXMax != 10 {
		t.Errorf("after reset X bounds = [%v, %v], want [0, 10]", gotXMin, gotXMax)
	}

	gotYMin, gotYMax := b.panels[0].YScale.Bounds()
	if gotYMin != 0 || gotYMax != 100 {
		t.Errorf("after reset Y bounds = [%v, %v], want [0, 100]", gotYMin, gotYMax)
	}
}

func TestBuilt_Zoomable_PartialLimits(t *testing.T) {
	t.Parallel()

	b := &Built{
		panels: []BuiltPanel{
			{XScale: newTrainedLinear(0, 10), YScale: newTrainedLinear(0, 100)},
		},
	}

	// Only set X min, leave X max and both Y untouched.
	xmin := 3.0
	b.SetPanelViewport(0, [2]*float64{&xmin, nil}, [2]*float64{nil, nil})

	gotXMin, gotXMax := b.panels[0].XScale.Bounds()
	if gotXMin != 3 || gotXMax != 10 {
		t.Errorf("partial X bounds = [%v, %v], want [3, 10]", gotXMin, gotXMax)
	}

	gotYMin, gotYMax := b.panels[0].YScale.Bounds()
	if gotYMin != 0 || gotYMax != 100 {
		t.Errorf("Y bounds should be unchanged = [%v, %v], want [0, 100]", gotYMin, gotYMax)
	}
}

func TestBuilt_Zoomable_OutOfBoundsPanel(_ *testing.T) { //nolint:paralleltest // No assertions, just panic safety check.
	b := &Built{
		panels: []BuiltPanel{
			{XScale: newTrainedLinear(0, 10), YScale: newTrainedLinear(0, 100)},
		},
	}

	// Should not panic.
	xmin := 5.0
	b.SetPanelViewport(-1, [2]*float64{&xmin, nil}, [2]*float64{nil, nil})
	b.SetPanelViewport(99, [2]*float64{&xmin, nil}, [2]*float64{nil, nil}) //nolint:mnd // Deliberately out-of-range index.
}

func TestBuilt_PanelInfos_AfterStoreGeometry(t *testing.T) {
	t.Parallel()

	b := &Built{
		panels: []BuiltPanel{
			{XScale: newTrainedLinear(0, 10), YScale: newTrainedLinear(0, 100)},
			{XScale: newTrainedLinear(5, 15), YScale: newTrainedLinear(50, 150)},
		},
	}

	// Simulate what Draw does — store geometry.
	b.storePanelGeometry(800, 600, []panelGeometry{
		{dataX: 60, dataY: 30, panelW: 350, panelH: 250},
		{dataX: 420, dataY: 30, panelW: 350, panelH: 250},
	})

	infos := b.PanelInfos()
	if len(infos) != 2 {
		t.Fatalf("expected 2 panels, got %d", len(infos))
	}

	// Check first panel.
	if infos[0].Bounds.Min.X != 60 || infos[0].Bounds.Min.Y != 30 {
		t.Errorf("panel 0 origin = %v, want (60, 30)", infos[0].Bounds.Min)
	}

	if infos[0].XRange != [2]float64{0, 10} {
		t.Errorf("panel 0 XRange = %v, want [0, 10]", infos[0].XRange)
	}

	// Check second panel.
	if infos[1].XRange != [2]float64{5, 15} {
		t.Errorf("panel 1 XRange = %v, want [5, 15]", infos[1].XRange)
	}
}

func TestBuilt_InterfaceCompliance(_ *testing.T) { //nolint:paralleltest // Compile-time check only.
	var _ output.Measurable = (*Built)(nil)

	var _ output.Zoomable = (*Built)(nil)
}

// newTrainedLinear creates a LinearScale with pre-set bounds.
func newTrainedLinear(mn, mx float64) *scale.LinearScale {
	s := &scale.LinearScale{}
	s.SetBounds(mn, mx)

	return s
}
