package guide

import (
	"github.com/TuSKan/ggplot/pkg/theme"
)

// Bounds tracks a calculated placement frame.
type Bounds struct {
	X, Y, W, H float64
}

// Placement associates an element with its calculated bounds.
type Placement struct {
	Element Element
	Rect    Bounds
}

// GuideLayout processes a GuideSet returning layout bounds mappings.
type GuideLayout struct {
	Outer      Bounds
	Data       Bounds
	Placements []Placement
}

// Compute processes bands functionally resolving the inner Data frame mapping constraints.
func Compute(gs *GuideSet, width, height float64, m TextMeasurer, th theme.Theme) *GuideLayout {
	gl := &GuideLayout{
		Outer: Bounds{X: 0, Y: 0, W: width, H: height},
	}

	// Measure all thicknesses
	topH := 0.0
	bottomH := 0.0
	leftW := 0.0
	rightW := 0.0

	var topSizes []float64
	var bottomSizes []float64
	var leftSizes []float64
	var rightSizes []float64

	for _, el := range gs.Top {
		_, h := el.Measure(m, th)
		topSizes = append(topSizes, h)
		topH += h
	}

	for _, el := range gs.Bottom {
		_, h := el.Measure(m, th)
		bottomSizes = append(bottomSizes, h)
		bottomH += h
	}

	for _, el := range gs.Left {
		w, _ := el.Measure(m, th)
		leftSizes = append(leftSizes, w)
		leftW += w
	}

	for _, el := range gs.Right {
		w, _ := el.Measure(m, th)
		rightSizes = append(rightSizes, w)
		rightW += w
	}

	// Resolve the central Data Area first!
	gl.Data = Bounds{
		X: leftW,
		Y: topH,
		W: width - leftW - rightW,
		H: height - topH - bottomH,
	}

	// Distribute geometries

	// Top Elements (drawn from very top 0 downwards)
	currY := 0.0
	for i, el := range gs.Top {
		gl.Placements = append(gl.Placements, Placement{
			Element: el,
			Rect: Bounds{
				X: gl.Data.X,
				Y: currY,
				W: gl.Data.W,
				H: topSizes[i],
			},
		})
		currY += topSizes[i]
	}

	// Bottom Elements (drawn directly below data downwards)
	currY = gl.Data.Y + gl.Data.H
	for i, el := range gs.Bottom {
		gl.Placements = append(gl.Placements, Placement{
			Element: el,
			Rect: Bounds{
				X: gl.Data.X,
				Y: currY,
				W: gl.Data.W,
				H: bottomSizes[i],
			},
		})
		currY += bottomSizes[i]
	}

	// Left Elements (drawn from Data.Y downwards, but Left to Right)
	currX := 0.0
	for i, el := range gs.Left {
		gl.Placements = append(gl.Placements, Placement{
			Element: el,
			Rect: Bounds{
				X: currX,
				Y: gl.Data.Y,
				W: leftSizes[i],
				H: gl.Data.H,
			},
		})
		currX += leftSizes[i]
	}

	// Right Elements (drawn directly right of data Left to Right)
	currX = gl.Data.X + gl.Data.W
	for i, el := range gs.Right {
		gl.Placements = append(gl.Placements, Placement{
			Element: el,
			Rect: Bounds{
				X: currX,
				Y: gl.Data.Y,
				W: rightSizes[i],
				H: gl.Data.H,
			},
		})
		currX += rightSizes[i]
	}

	return gl
}
