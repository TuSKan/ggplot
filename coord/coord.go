// Package coord defines coordinate systems that control how data positions
// are mapped to the 2D plotting surface.
package coord

import "math"

// Coord transforms scaled data positions (in [0,1] normalized space)
// into plot-surface positions.
type Coord interface {
	// Transform maps normalized (x, y) ∈ [0,1]² to pixel coordinates
	// within the data area of dimensions (width, height).
	Transform(x, y, width, height float64) (px, py float64)

	// String returns a human-readable label.
	String() string
}

// Cartesian returns the default Cartesian coordinate system.
// x maps to horizontal (left→right), y maps to vertical (bottom→top).
func Cartesian() Coord { return cartesianCoord{} }

type cartesianCoord struct{}

func (cartesianCoord) Transform(x, y, w, h float64) (float64, float64) {
	return x * w, h - y*h
}
func (cartesianCoord) String() string { return "cartesian" }

// Polar returns a polar coordinate system where x maps to angle (theta,
// 0→2π) and y maps to radius (0→1). Center is at (w/2, h/2).
func Polar() Coord { return polarCoord{} }

type polarCoord struct{}

func (polarCoord) Transform(x, y, w, h float64) (float64, float64) {
	theta := x * 2 * math.Pi // x ∈ [0,1] → [0, 2π]
	maxR := math.Min(w, h) / 2
	r := y * maxR

	cx, cy := w/2, h/2
	px := cx + r*math.Cos(theta)
	py := cy - r*math.Sin(theta) // y-inverted for screen coords
	return px, py
}
func (polarCoord) String() string { return "polar" }
