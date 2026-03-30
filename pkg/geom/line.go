package geom

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/stat"
	"github.com/gogpu/gg"
)

// LineBuilder constructs an ast.Layer for Line/Path continuous geometries.
type LineBuilder struct {
	geom LineGeom
	stat ast.Stat
}

func (l *LineBuilder) BuildLayer() ast.Layer {
	return ast.Layer{
		Geom: l.geom,
		Stat: l.stat,
	}
}

// LineGeom encapsulates connected vertex geometries.
type LineGeom struct {
	Opts Opts
}

func (p LineGeom) Name() string                 { return "Line" }
func (p LineGeom) RequiredAesthetics() []string { return []string{"x", "y"} }
func (p LineGeom) DefaultStat() string          { return "identity" }

func (p LineGeom) Draw(dc *gg.Context, ctx RenderContext, ds dataset.Dataset, mapping ast.Aes) error {
	coords, err := ResolveCoordinates(ds, mapping)
	if err != nil {
		return err
	}

	if coords.X == nil || coords.Y == nil {
		return nil
	}
	
	minX, maxX, minY, maxY, _, _ := ResolveBounds(ds, mapping)

	proj := Projector{Width: ctx.Width, Height: ctx.Height, Padding: 0.0}

	r, g, b := ParseHexColor(p.Opts.Color, 0.8, 0.2, 0.2)
	dc.SetRGBA(r, g, b, 1.0)

	if lw := p.Opts.LineWidth; lw > 0 {
		dc.SetLineWidth(lw)
	} else {
		dc.SetLineWidth(2.0)
	}

	first := true
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

		if first {
			dc.MoveTo(screenX, screenY)
			first = false
		} else {
			dc.LineTo(screenX, screenY)
		}
	}
	if !first {
		dc.Stroke()
	}

	return nil
}

// Line returns an unmapped continuous Line geometry layer builder carrying default 'Identity' stats.
func Line(args ...any) *LineBuilder {
	o := Opts{}
	var s ast.Stat = stat.MethodIdentity
	
	for _, arg := range args {
		if opt, ok := arg.(Opts); ok {
			o = opt
		} else if st, ok := arg.(ast.Stat); ok {
			s = st
		}
	}
	
	return &LineBuilder{
		geom: LineGeom{Opts: o},
		stat: s,
	}
}
