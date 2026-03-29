package layout

import (
	"testing"
)

func TestFacetGrid(t *testing.T) {
	bounds := Rect{Min: Point{X: 0, Y: 0}, Max: Point{X: 100, Y: 100}}
	panels := GeneratePanels(FacetGrid, bounds, 2, 2, 4, 10.0)

	if len(panels) != 4 {
		t.Fatalf("expected 4 panels natively sliced from grid, got %d", len(panels))
	}

	// 2 rows, 2 cols, 1 space (10) explicitly means:
	// totalWidth = 100 - 10 = 90
	// panelWidth = 45
	first := panels[0]
	if first.Width() != 45 {
		t.Errorf("expected cell width 45.0 natively divided, got %f", first.Width())
	}
	if first.Height() != 45 {
		t.Errorf("expected cell height 45.0, got %f", first.Height())
	}

	// second rect (row 0, col 1) x starts at 45 + 10 = 55
	second := panels[1]
	if second.Min.X != 55.0 {
		t.Errorf("expected second cell min.X spanning strictly at 55.0 properly capturing spacer, got %f", second.Min.X)
	}
}

func TestFacetWrap(t *testing.T) {
	bounds := Rect{Min: Point{X: 0, Y: 0}, Max: Point{X: 110, Y: 210}}

	// Request exactly 5 panels wrapped into 3 columns
	// Expected matrix:
	// P P P
	// P P -
	// Rows = ceil(5/3) = 2.
	// Spacing = 10 explicitly
	panels := GeneratePanels(FacetWrap, bounds, 0, 3, 5, 10.0)

	if len(panels) != 5 {
		t.Fatalf("expected precisely exactly 5 bounds cutting wrap short gracefully, got %d", len(panels))
	}

	// panelWidth: 3 cols -> 2 spaces = 20. (110 - 20) / 3 = 30 width
	// panelHeight: 2 rows -> 1 space = 10. (210 - 10) / 2 = 100 height

	p1 := panels[0]
	if p1.Width() != 30.0 || p1.Height() != 100.0 {
		t.Errorf("cell spatial width (%f) / height (%f) bounding mismatched expectation 30/100", p1.Width(), p1.Height())
	}

	p5 := panels[4]
	// p5 is row 1 (y = 110), col 1 (x = 40)
	if p5.Min.X != 40.0 {
		t.Errorf("expected 5th wrapper bounds pushed iteratively onto second column natively starting at X=40.0, got %f", p5.Min.X)
	}
	if p5.Min.Y != 110.0 {
		t.Errorf("expected 5th wrapper identically pushing down cleanly past space 110.0, got %f", p5.Min.Y)
	}
}
