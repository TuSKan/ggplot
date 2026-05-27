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

// ---------------------------------------------------------------------------
// coord.Trans — post-stat per-axis transforms (specification only)
// ---------------------------------------------------------------------------

// TransFunc names a bijective mathematical axis transformation.
//
// The coord package is a pure specification layer — it carries no math.
// The build pipeline resolves Name to the appropriate engine [MathKernel]
// operation for column-level transforms, and to the matching inverse
// function for tick-label formatting.
//
// Built-in names: "log10", "log2", "sqrt", "reverse", "identity".
type TransFunc struct {
	Name string
}

// IsIdentity reports whether this transform is a no-op.
func (t TransFunc) IsIdentity() bool { return t.Name == identityName }

const identityName = "identity"

// Built-in axis transforms.
//
// These are specification-only — they carry no math functions.
// The build pipeline resolves their Names to engine operations
// with domain clamping and to inverse functions for tick formatting.
var (
	// TransLog10 applies base-10 logarithm. Suitable for data spanning
	// multiple orders of magnitude. Values ≤ 0 are clamped by the
	// pipeline to a finite floor.
	TransLog10 = TransFunc{Name: "log10"}

	// TransLog2 applies base-2 logarithm. Useful for genomic fold-change,
	// doubling-time, and binary-scale data.
	TransLog2 = TransFunc{Name: "log2"}

	// TransSqrt applies square root. Useful for count data and variances.
	// Negative values are clamped to zero by the pipeline.
	TransSqrt = TransFunc{Name: "sqrt"}

	// TransReverse negates values, flipping axis direction.
	TransReverse = TransFunc{Name: "reverse"}

	// TransIdentity is a no-op transform. The build pipeline skips
	// identity transforms entirely.
	TransIdentity = TransFunc{Name: identityName}
)

// ---------------------------------------------------------------------------
// Transformer interface + transCoord implementation
// ---------------------------------------------------------------------------

// Transformer is an optional interface implemented by coordinate systems
// that apply per-axis mathematical transforms post-stat. The build pipeline
// checks for this after stat transforms and dispatches to the engine's
// MathKernel for column-level computation, before scale training.
type Transformer interface {
	// XTrans returns the X-axis TransFunc specification.
	XTrans() TransFunc
	// YTrans returns the Y-axis TransFunc specification.
	YTrans() TransFunc
}

// Trans returns a Cartesian coordinate system with per-axis mathematical
// transforms applied post-stat. Unlike scale transforms (e.g., scale.Log10)
// which transform data before stat computations, coord.Trans transforms
// the data columns via the engine after stats complete but before scales
// are trained. This means statistics run on raw data, but scales, ticks,
// and rendering all operate on transformed values.
//
// Use built-in transforms ([TransLog10], [TransSqrt], etc.).
func Trans(xtrans, ytrans TransFunc) Coord {
	if xtrans.Name == "" {
		xtrans = TransIdentity
	}

	if ytrans.Name == "" {
		ytrans = TransIdentity
	}

	return transCoord{xtrans: xtrans, ytrans: ytrans}
}

type transCoord struct {
	xtrans, ytrans TransFunc
}

// Transform implements [Coord]. Standard Cartesian mapping — the
// nonlinear transform was already applied to the data columns by the
// build pipeline, so normalised [0,1] → pixel is linear.
func (c transCoord) Transform(x, y, w, h float64) (float64, float64) {
	return x * w, h - y*h
}

func (c transCoord) String() string {
	return "trans(" + c.xtrans.Name + ", " + c.ytrans.Name + ")"
}

// XTrans implements [Transformer].
func (c transCoord) XTrans() TransFunc { return c.xtrans }

// YTrans implements [Transformer].
func (c transCoord) YTrans() TransFunc { return c.ytrans }

// ---------------------------------------------------------------------------
// Compile-time interface checks
// ---------------------------------------------------------------------------

var (
	_ Coord       = cartesianCoord{}
	_ Coord       = polarCoord{}
	_ Coord       = cartesianZoomCoord{}
	_ Coord       = fixedCoord{}
	_ Coord       = transCoord{}
	_ Zoomer      = cartesianZoomCoord{}
	_ Fixer       = fixedCoord{}
	_ Transformer = transCoord{}
)
