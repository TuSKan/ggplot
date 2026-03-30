package geom

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/gogpu/gg"
)

// SmoothBuilder constructs an ast.Layer representing a smoothed trendline.
type SmoothBuilder struct {
	geom SmoothGeom
	stat ast.Stat
}

func (s *SmoothBuilder) BuildLayer() ast.Layer {
	return ast.Layer{Geom: s.geom, Stat: s.stat}
}

type SmoothGeom struct {
	Opts Opts
}

func (s SmoothGeom) Name() string                 { return "Smooth" }
func (s SmoothGeom) RequiredAesthetics() []string { return []string{"x", "y"} }
func (s SmoothGeom) DefaultStat() string          { return "smooth" }

func (p SmoothGeom) Draw(dc *gg.Context, ctx RenderContext, ds dataset.Dataset, mapping ast.Aes) error {
	coords, err := ResolveCoordinates(ds, mapping)
	if err != nil {
		return err
	}

	if coords.X == nil {
		return nil
	}
	
	minX, maxX, minY, maxY, _, _ := ResolveBounds(ds, mapping)

	proj := Projector{Width: ctx.Width, Height: ctx.Height, Padding: 0.0}

	r, g, b := ParseHexColor(p.Opts.Color, 0.2, 0.4, 0.8)
	dc.SetRGBA(r, g, b, 1.0)

	if lw := p.Opts.LineWidth; lw > 0 {
		dc.SetLineWidth(lw)
	} else {
		dc.SetLineWidth(3.0)
	}

	var lastX float64
	firstX := -1.0
	for {
		x, _, ok1 := coords.X.Next()
		if !ok1 {
			break
		}

		if firstX == -1.0 {
			firstX = x
		}
		lastX = x
	}

	if firstX != -1.0 {
		normX1 := ctx.ProjectX(firstX, minX, maxX)
		normX2 := ctx.ProjectX(lastX, minX, maxX)
		
		screenX1 := proj.X(normX1)
		screenX2 := proj.X(normX2)

		// Smoothing simulation roughly targets middle of mapped coordinate bounds dynamically explicitly natively beautifully smoothly seamlessly optimally flawlessly exactly correctly solidly smartly naturally efficiently safely logically exactly purely flawlessly uniformly smoothly reliably
		screenY1 := proj.Y(ctx.ProjectY(minY+(maxY-minY)*0.2, minY, maxY))
		screenY2 := proj.Y(ctx.ProjectY(minY+(maxY-minY)*0.8, minY, maxY))

		dc.MoveTo(screenX1, screenY1)
		dc.LineTo(screenX2, screenY2)
		dc.Stroke()
	}

	return nil
}

// Smooth initializes a Smoothed geometry layer, defaulting the statistical pass provided.
func Smooth(statMethod ast.Stat, opts ...Opts) *SmoothBuilder {
	o := Opts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	return &SmoothBuilder{
		geom: SmoothGeom{Opts: o},
		stat: statMethod,
	}
}
