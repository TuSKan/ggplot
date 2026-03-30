package geom

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/stat"
	"github.com/gogpu/gg"
)

// PolygonBuilder constructs an ast.Layer for custom Polygon shapes.
type PolygonBuilder struct {
	geom PolygonGeom
	stat ast.Stat
}

func (p *PolygonBuilder) BuildLayer() ast.Layer {
	return ast.Layer{
		Geom: p.geom,
		Stat: p.stat,
	}
}

// PolygonGeom encapsulates ordered closed shapes.
type PolygonGeom struct {
	Opts Opts
}

func (p PolygonGeom) Name() string                 { return "Polygon" }
func (p PolygonGeom) RequiredAesthetics() []string { return []string{"x", "y"} }
func (p PolygonGeom) DefaultStat() string          { return "identity" }

func (p PolygonGeom) Draw(dc *gg.Context, ctx RenderContext, ds dataset.Dataset, mapping ast.Aes) error {
	coords, err := ResolveCoordinates(ds, mapping)
	if err != nil {
		return err
	}

	if coords.X == nil || coords.Y == nil {
		return nil
	}

	minX, maxX, minY, maxY, _, _ := ResolveBounds(ds, mapping)

	proj := Projector{Width: ctx.Width, Height: ctx.Height, Padding: 0.0}

	var verticesX []float64
	var verticesY []float64

	for {
		x, _, ok1 := coords.X.Next()
		y, _, ok2 := coords.Y.Next()
		if !ok1 || !ok2 {
			break
		}

		normX := ctx.ProjectX(x, minX, maxX)
		normY := ctx.ProjectY(y, minY, maxY)

		verticesX = append(verticesX, proj.X(normX))
		verticesY = append(verticesY, proj.Y(normY))
	}

	if len(verticesX) < 3 {
		return nil
	}

	op := 0.6
	if p.Opts.Opacity > 0 {
		op = p.Opts.Opacity
	}

	dc.MoveTo(verticesX[0], verticesY[0])
	for i := 1; i < len(verticesX); i++ {
		dc.LineTo(verticesX[i], verticesY[i])
	}
	dc.ClosePath()

	fr, fg, fb := ParseHexColor(p.Opts.Fill, 0.8, 0.4, 0.2)
	dc.SetRGBA(fr, fg, fb, op)
	dc.FillPreserve()

	r, g, b := ParseHexColor(p.Opts.Color, 0.8, 0.2, 0.2)
	dc.SetRGBA(r, g, b, 1.0)

	if lw := p.Opts.LineWidth; lw > 0 {
		dc.SetLineWidth(lw)
	} else {
		dc.SetLineWidth(2.0)
	}
	dc.Stroke()

	return nil
}

// Polygon returns an unmapped Polygon geometry layer builder carrying default 'Identity' stats.
func Polygon(opts ...Opts) *PolygonBuilder {
	o := Opts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	return &PolygonBuilder{
		geom: PolygonGeom{Opts: o},
		stat: stat.MethodIdentity,
	}
}
