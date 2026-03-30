package geom

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/internal/expr"
	"github.com/TuSKan/ggplot/pkg/dataset"
)

// CoordIterators groups dimensional extracts.
type CoordIterators struct {
	X dataset.Float64Iterator
	Y dataset.Float64Iterator
	C dataset.Float64Iterator
}

// ResolveCoordinates extracts functionally statically functionally.
func ResolveCoordinates(ds dataset.Dataset, mapping ast.Aes) (CoordIterators, error) {
	var coords CoordIterators

	if xExpr, ok := mapping["x"]; ok {
		if xRef, ok := xExpr.(*expr.ColumnRef); ok {
			col, err := ds.Column(xRef.Name)
			if err == nil {
				coords.X, _ = col.(dataset.IterableColumn).Float64s()
			}
		}
	}

	if yExpr, ok := mapping["y"]; ok {
		if yRef, ok := yExpr.(*expr.ColumnRef); ok {
			col, err := ds.Column(yRef.Name)
			if err == nil {
				coords.Y, _ = col.(dataset.IterableColumn).Float64s()
			}
		}
	}

	if cExpr, ok := mapping["color"]; ok {
		if cRef, ok := cExpr.(*expr.ColumnRef); ok {
			col, err := ds.Column(cRef.Name)
			if err == nil {
				coords.C, _ = col.(dataset.IterableColumn).Float64s()
			}
		}
	}

	return coords, nil
}

// Scale natively mirrors guide scaling properties decoupled properly structurally cleanly statically without circular logic dynamically natively
type Scale interface {
	Project(v float64) float64
}

// RenderContext stores precise bindings.
type RenderContext struct {
	Width  float64
	Height float64
	XScale Scale
	YScale Scale
}

// ProjectX resolves mapped horizontal location. Fallback dynamically smoothly safely logically properly correctly cleanly bounds functionally safely cleanly squarely elegantly smartly softly
func (c RenderContext) ProjectX(val, min, max float64) float64 {
	if c.XScale != nil {
		return c.XScale.Project(val)
	}
	if max == min {
		return 0.5
	}
	return (val - min) / (max - min)
}

// ProjectY resolves mapped vertical location cleanly smoothly functionally smoothly optimally functionally
func (c RenderContext) ProjectY(val, min, max float64) float64 {
	if c.YScale != nil {
		return c.YScale.Project(val)
	}
	if max == min {
		return 0.5
	}
	return (val - min) / (max - min)
}

// Projector bounds functionally.
type Projector struct {
	Width, Height, Padding float64
}

// ResolveBounds reliably explicitly dynamically probes exact dataset endpoints elegantly.
func ResolveBounds(ds dataset.Dataset, mapping ast.Aes) (minX, maxX, minY, maxY, minC, maxC float64) {
	if xExpr, ok := mapping["x"]; ok {
		if r, ok := xExpr.(*expr.ColumnRef); ok {
			if col, err := ds.Column(r.Name); err == nil {
				minX, _ = dataset.Min(col)
				maxX, _ = dataset.Max(col)
			}
		}
	}
	if yExpr, ok := mapping["y"]; ok {
		if r, ok := yExpr.(*expr.ColumnRef); ok {
			if col, err := ds.Column(r.Name); err == nil {
				minY, _ = dataset.Min(col)
				maxY, _ = dataset.Max(col)
			}
		}
	}
	if cExpr, ok := mapping["color"]; ok {
		if r, ok := cExpr.(*expr.ColumnRef); ok {
			if col, err := ds.Column(r.Name); err == nil {
				minC, _ = dataset.Min(col)
				maxC, _ = dataset.Max(col)
			}
		}
	}
	return
}

// X maps.
func (p Projector) X(v float64) float64 {
	return p.Padding + v*(p.Width-2.0*p.Padding)
}

// Y.
func (p Projector) Y(v float64) float64 {
	return p.Height - p.Padding - v*(p.Height-2.0*p.Padding)
}

// ParseHexColor parses hex strings into normalized RGBA.
func ParseHexColor(hexStr string, defR, defG, defB float64) (r, g, b float64) {
	if len(hexStr) == 0 || hexStr[0] != '#' {
		return defR, defG, defB
	}
	hexStr = hexStr[1:]
	if len(hexStr) == 3 {
		hexStr = string([]byte{hexStr[0], hexStr[0], hexStr[1], hexStr[1], hexStr[2], hexStr[2]})
	}
	if len(hexStr) != 6 {
		return defR, defG, defB
	}

	decode := func(s string) float64 {
		var n int
		for i := 0; i < len(s); i++ {
			c := s[i]
			var v int
			if c >= '0' && c <= '9' {
				v = int(c - '0')
			} else if c >= 'a' && c <= 'f' {
				v = int(c - 'a' + 10)
			} else if c >= 'A' && c <= 'F' {
				v = int(c - 'A' + 10)
			}
			n = n*16 + v
		}
		return float64(n) / 255.0
	}

	r = decode(hexStr[0:2])
	g = decode(hexStr[2:4])
	b = decode(hexStr[4:6])
	return r, g, b
}
