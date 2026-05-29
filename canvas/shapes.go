package canvas

import "math"

// Shape name constants — single source of truth for all shape identifiers
// used throughout the library (scales, drawers, legends, canvas backends).
const (
	ShapeCircle       = "circle"
	ShapeSquare       = "square"
	ShapeTriangle     = "triangle"
	ShapeTriangleDown = "triangleDown"
	ShapeDiamond      = "diamond"
	ShapePlus         = "plus"
	ShapeCross        = "cross"
	ShapeStar         = "star"
	ShapePentagon     = "pentagon"
	ShapeHexagon      = "hexagon"
)

// Shapes returns the ordered list of all supported shape names.
// The order matches the default cycle used by [scale.ShapeScale].
func Shapes() []string {
	return []string{
		ShapeCircle,
		ShapeSquare,
		ShapeTriangle,
		ShapeDiamond,
		ShapeTriangleDown,
		ShapePlus,
		ShapeCross,
		ShapeStar,
		ShapePentagon,
		ShapeHexagon,
	}
}

// IsStrokeShape reports whether the shape should be rendered with stroke
// rather than fill (i.e., line-based shapes).
func IsStrokeShape(shape string) bool {
	return shape == ShapePlus || shape == ShapeCross
}

// DrawShapePath constructs a shape path using the Canvas path API.
// This is the fallback implementation used by [RecordingCanvas] and other
// backends that do not have native polygon primitives. [RasterCanvas] uses
// gg.Context.DrawRegularPolygon directly instead — see [RasterCanvas.DrawShape].
func DrawShapePath(c Canvas, shape string, cx, cy, r float64) {
	switch shape {
	case ShapeCircle:
		c.DrawCircle(cx, cy, r)
	case ShapeSquare:
		c.DrawRectangle(cx-r, cy-r, r*2, r*2) //nolint:mnd // symmetric bounding box
	case ShapeTriangle:
		drawRegularPolygonPath(c, 3, cx, cy, r, 0)
	case ShapeTriangleDown:
		drawRegularPolygonPath(c, 3, cx, cy, r, math.Pi)
	case ShapeDiamond:
		drawRegularPolygonPath(c, 4, cx, cy, r, math.Pi/4) //nolint:mnd // 45° rotation makes diamond
	case ShapePlus:
		c.DrawLine(cx, cy-r, cx, cy+r)
		c.DrawLine(cx-r, cy, cx+r, cy)
	case ShapeCross:
		c.DrawLine(cx-r, cy-r, cx+r, cy+r)
		c.DrawLine(cx-r, cy+r, cx+r, cy-r)
	case ShapeStar:
		drawStarPath(c, cx, cy, r)
	case ShapePentagon:
		drawRegularPolygonPath(c, 5, cx, cy, r, 0)
	case ShapeHexagon:
		drawRegularPolygonPath(c, 6, cx, cy, r, 0)
	default:
		// Unknown shape — fall back to circle.
		c.DrawCircle(cx, cy, r)
	}
}

// drawRegularPolygonPath draws a regular polygon using the Canvas path API.
// orientation follows gg convention: rotation=0 gives vertex-up for odd n,
// flat-top for even n.
func drawRegularPolygonPath(c Canvas, n int, cx, cy, r, rotation float64) {
	angle := 2.0 * math.Pi / float64(n)
	rotation -= math.Pi / 2

	if n%2 == 0 {
		rotation += angle / 2
	}

	for i := range n {
		a := rotation + angle*float64(i)
		px := cx + r*math.Cos(a)
		py := cy + r*math.Sin(a)

		if i == 0 {
			c.MoveTo(px, py)
		} else {
			c.LineTo(px, py)
		}
	}

	c.ClosePath()
}

// drawStarPath draws a five-pointed star with inner radius at 40% of outer.
func drawStarPath(c Canvas, cx, cy, r float64) {
	const (
		points     = 10
		innerRatio = 0.4 // inner radius as fraction of outer — visual convention for 5-pointed star
	)

	for i := range points {
		a := float64(i)*math.Pi/5.0 - math.Pi/2.0 //nolint:mnd // 10 vertices at π/5 spacing, rotated -π/2 for vertex-up
		rad := r

		if i%2 == 1 {
			rad = r * innerRatio
		}

		px := cx + rad*math.Cos(a)
		py := cy + rad*math.Sin(a)

		if i == 0 {
			c.MoveTo(px, py)
		} else {
			c.LineTo(px, py)
		}
	}

	c.ClosePath()
}
