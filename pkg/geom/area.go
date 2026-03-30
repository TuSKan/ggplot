package geom

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/stat"
	"github.com/gogpu/gg"
)

// AreaBuilder constructs an ast.Layer for Area continuous geometries.
type AreaBuilder struct {
	geom AreaGeom
	stat ast.Stat
}

func (a *AreaBuilder) BuildLayer() ast.Layer {
	return ast.Layer{
		Geom: a.geom,
		Stat: a.stat,
	}
}

// AreaGeom encapsulates filled path geometries connected to 0.
type AreaGeom struct {
	Opts Opts
}

func (a AreaGeom) Name() string                 { return "Area" }
func (a AreaGeom) RequiredAesthetics() []string { return []string{"x", "y"} }
func (a AreaGeom) DefaultStat() string          { return "identity" }

func (a AreaGeom) Draw(dc *gg.Context, ctx RenderContext, ds dataset.Dataset, mapping ast.Aes) error {
	coords, err := ResolveCoordinates(ds, mapping)
	if err != nil {
		return err
	}

	if coords.X == nil || coords.Y == nil {
		return nil
	}
	
	minX, maxX, minY, maxY, _, _ := ResolveBounds(ds, mapping)

	proj := Projector{Width: ctx.Width, Height: ctx.Height, Padding: 0.0}

	var step []float64
	var value []float64
	for {
		x, _, ok1 := coords.X.Next()
		y, _, ok2 := coords.Y.Next()
		if !ok1 || !ok2 {
			break
		}
		normX := ctx.ProjectX(x, minX, maxX)
		normY := ctx.ProjectY(y, minY, maxY)
		step = append(step, normX)
		value = append(value, normY)
	}

	if len(step) < 2 {
		return nil
	}

	baseY := ctx.ProjectY(0.0, minY, maxY)

	dc.MoveTo(proj.X(step[0]), proj.Y(baseY))
	for i := 0; i < len(step); i++ {
		dc.LineTo(proj.X(step[i]), proj.Y(value[i]))
	}
	dc.LineTo(proj.X(step[len(step)-1]), proj.Y(baseY))
	dc.ClosePath()

	op := 0.6
	if a.Opts.Opacity > 0 {
		op = a.Opts.Opacity
	}

	fr, fg, fb := ParseHexColor(a.Opts.Fill, 0.1, 0.7, 0.3)
	dc.SetRGBA(fr, fg, fb, op)
	dc.FillPreserve()

	r, g, b := ParseHexColor(a.Opts.Color, 0.0, 0.0, 0.0)
	dc.SetRGBA(r, g, b, 1.0)
	dc.Stroke()

	return nil
}

// Area returns an unmapped Area geometry layer builder carrying default 'Identity' stats.
func Area(opts ...Opts) *AreaBuilder {
	o := Opts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	return &AreaBuilder{
		geom: AreaGeom{Opts: o},
		stat: stat.MethodIdentity,
	}
}
