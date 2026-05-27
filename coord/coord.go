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

// Zoomer is an optional interface implemented by coordinate systems that
// provide viewport zoom limits. The build pipeline checks for this after
// scale training and overrides scale bounds accordingly.
type Zoomer interface {
	// ZoomBounds returns the zoom viewport limits.
	// nil entries mean "use the trained scale bounds".
	ZoomBounds() (xlim, ylim [2]*float64)
}

// CartesianZoom returns a Cartesian coordinate system with viewport zoom.
// Unlike [Plot.XLim]/[Plot.YLim] which set scale bounds early, CartesianZoom
// overrides bounds after scale training without filtering data — all
// data participates in stat computations, only the visible window changes.
//
// Pass nil for either limit endpoint to auto-detect from training.
func CartesianZoom(xlim, ylim [2]*float64) Coord {
	return cartesianZoomCoord{xlim: xlim, ylim: ylim}
}

type cartesianZoomCoord struct {
	xlim, ylim [2]*float64
}

func (cartesianZoomCoord) Transform(x, y, w, h float64) (float64, float64) {
	return x * w, h - y*h
}

func (cartesianZoomCoord) String() string { return "cartesian_zoom" }

// ZoomBounds implements [Zoomer].
func (c cartesianZoomCoord) ZoomBounds() (xlim, ylim [2]*float64) {
	return c.xlim, c.ylim
}

// Fixer is an optional interface implemented by coordinate systems that
// enforce a fixed aspect ratio. The render pipeline checks for this when
// computing panel dimensions.
type Fixer interface {
	// AspectRatio returns the desired y/x aspect ratio.
	// ratio = 1 means one unit of x occupies the same pixel length as one
	// unit of y. ratio = 2 means y is stretched to twice the x scale.
	AspectRatio() float64
}

// Fixed returns a Cartesian coordinate system with a fixed aspect ratio.
// ratio is defined as (pixels per data-unit-y) / (pixels per data-unit-x).
// ratio = 1 gives equal scaling — one unit of x occupies the same pixel
// length as one unit of y.
func Fixed(ratio float64) Coord {
	if ratio <= 0 {
		ratio = 1
	}

	return fixedCoord{ratio: ratio}
}

type fixedCoord struct {
	ratio float64
}

func (c fixedCoord) Transform(x, y, w, h float64) (float64, float64) {
	return x * w, h - y*h
}

func (fixedCoord) String() string { return "fixed" }

// AspectRatio implements [Fixer].
func (c fixedCoord) AspectRatio() float64 { return c.ratio }
