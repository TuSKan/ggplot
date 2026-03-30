package geom

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/stat"
	"github.com/gogpu/gg"
)

// PointBuilder constructs an ast.Layer for Point geometries.
type PointBuilder struct {
	geom PointGeom
	stat ast.Stat
}

func (b *PointBuilder) BuildLayer() ast.Layer {
	return ast.Layer{
		Geom: b.geom,
		Stat: b.stat,
	}
}

// PointGeom encapsulates geometric primitives.
type PointGeom struct {
	Opts Opts
}

func (p PointGeom) Name() string                 { return "Point" }
func (p PointGeom) RequiredAesthetics() []string { return []string{"x", "y"} }
func (p PointGeom) DefaultStat() string          { return "identity" }

func (p PointGeom) Draw(dc *gg.Context, ctx RenderContext, ds dataset.Dataset, mapping ast.Aes) error {
	coords, err := ResolveCoordinates(ds, mapping)
	if err != nil {
		return err
	}

	if coords.X == nil || coords.Y == nil {
		return nil
	}

	minX, maxX, minY, maxY, minC, maxC := ResolveBounds(ds, mapping)
	_ = minC
	_ = maxC

	proj := Projector{Width: ctx.Width, Height: ctx.Height, Padding: 0.0}

	r := 3.0
	if p.Opts.Radius > 0 {
		r = p.Opts.Radius
	}

	op := 1.0
	if p.Opts.Opacity > 0 {
		op = p.Opts.Opacity
	}

	colorR, colorG, colorB := ParseHexColor(p.Opts.Color, 0.2, 0.4, 0.8)

	for {
		x, _, ok1 := coords.X.Next()
		y, _, ok2 := coords.Y.Next()
		if !ok1 || !ok2 {
			break
		}

		normX := ctx.ProjectX(x, minX, maxX)
		normY := ctx.ProjectY(y, minY, maxY)

		screenX := proj.X(normX)
		screenY := proj.Y(normY)

		if coords.C != nil {
			c, _, _ := coords.C.Next()
			colorR, colorG, colorB = 1.0-c, c, 0.0
		}

		dc.SetRGBA(colorR, colorG, colorB, op)
		dc.DrawCircle(screenX, screenY, r)
		dc.Fill()
	}

	return nil
}

// Point returns an unmapped Point geometry layer builder carrying default 'Identity' stats.
func Point(opts ...Opts) *PointBuilder {
	o := Opts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	return &PointBuilder{
		geom: PointGeom{Opts: o},
		stat: stat.MethodIdentity,
	}
}
