package layout

import "math"

// Point explicitly bounds 2D coordinates structurally.
type Point struct {
	X, Y float64
}

// Rect specifies a 2D bounding frame.
type Rect struct {
	Min Point // Top-Left globally
	Max Point // Bottom-Right globally
}

// Area constraints
func (r Rect) Width() float64  { return math.Max(0, r.Max.X - r.Min.X) }
func (r Rect) Height() float64 { return math.Max(0, r.Max.Y - r.Min.Y) }

// SliceTop cuts precisely `h` units horizontally along the top shrinking the main bounds.
func (r *Rect) SliceTop(h float64) Rect {
	cut := Rect{Min: r.Min, Max: Point{X: r.Max.X, Y: r.Min.Y + h}}
	r.Min.Y += h
	return cut
}

// SliceBottom cuts precisely `h` units horizontally strictly from the bottom shrinking upwards.
func (r *Rect) SliceBottom(h float64) Rect {
	cut := Rect{Min: Point{X: r.Min.X, Y: r.Max.Y - h}, Max: r.Max}
	r.Max.Y -= h
	return cut
}

// SliceLeft cuts precisely `w` units vertically from the left boundary sliding right.
func (r *Rect) SliceLeft(w float64) Rect {
	cut := Rect{Min: r.Min, Max: Point{X: r.Min.X + w, Y: r.Max.Y}}
	r.Min.X += w
	return cut
}

// SliceRight cuts precisely `w` units vertically strictly from the right boundary sliding left.
func (r *Rect) SliceRight(w float64) Rect {
	cut := Rect{Min: Point{X: r.Max.X - w, Y: r.Min.Y}, Max: r.Max}
	r.Max.X -= w
	return cut
}

// Margin allocates explicit spatial padding.
type Margin struct {
	Top, Right, Bottom, Left float64
}
