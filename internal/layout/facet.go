package layout

// FacetType distinguishes panel compilation flows.
type FacetType string

const (
	FacetNone FacetType = "none"
	FacetGrid FacetType = "grid"
	FacetWrap FacetType = "wrap"
)

// GeneratePanels explicitly divides a master constraint Rect deterministically into distinct mapped windows.
// For FacetNone passing 1 for cols/rows produces a master bound frame.
func GeneratePanels(fType FacetType, boundary Rect, rows, cols, count int, spacing float64) []Rect {
	if fType == FacetNone {
		return []Rect{boundary}
	}

	if fType == FacetGrid {
		return splitGrid(boundary, rows, cols, spacing)
	}

	if fType == FacetWrap {
		return splitWrap(boundary, cols, count, spacing)
	}

	return []Rect{}
}

func splitGrid(boundary Rect, rows, cols int, spacing float64) []Rect {
	if rows <= 0 || cols <= 0 {
		return []Rect{}
	}

	totalSpacingX := float64(cols-1) * spacing
	totalSpacingY := float64(rows-1) * spacing

	panelW := (boundary.Width() - totalSpacingX) / float64(cols)
	panelH := (boundary.Height() - totalSpacingY) / float64(rows)

	panels := make([]Rect, 0, rows*cols)

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			y1 := boundary.Min.Y + float64(r)*(panelH+spacing)
			x1 := boundary.Min.X + float64(c)*(panelW+spacing)

			panels = append(panels, Rect{
				Min: Point{X: x1, Y: y1},
				Max: Point{X: x1 + panelW, Y: y1 + panelH},
			})
		}
	}
	return panels
}

func splitWrap(boundary Rect, cols, count int, spacing float64) []Rect {
	if count <= 0 || cols <= 0 {
		return []Rect{}
	}

	rows := (count + cols - 1) / cols // Ceiling integer division

	totalSpacingX := float64(cols-1) * spacing
	totalSpacingY := float64(rows-1) * spacing

	panelW := (boundary.Width() - totalSpacingX) / float64(cols)
	panelH := (boundary.Height() - totalSpacingY) / float64(rows)

	panels := make([]Rect, 0, count)

	idx := 0
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if idx >= count {
				return panels
			}
			y1 := boundary.Min.Y + float64(r)*(panelH+spacing)
			x1 := boundary.Min.X + float64(c)*(panelW+spacing)

			panels = append(panels, Rect{
				Min: Point{X: x1, Y: y1},
				Max: Point{X: x1 + panelW, Y: y1 + panelH},
			})
			idx++
		}
	}
	return panels
}
