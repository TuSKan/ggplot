package geom

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/pkg/stat"
)

// Opts defines geometry-specific parameters outside of mapped variable scales
// (like fixed radiuses or stroke opacities).
type Opts struct {
	Radius  float64
	Opacity float64
}

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

// SmoothBuilder constructs an ast.Layer representing a smoothed trendline.
type SmoothBuilder struct {
	geom SmoothGeom
	stat ast.Stat
}

func (s *SmoothBuilder) BuildLayer() ast.Layer {
	return ast.Layer{Geom: s.geom, Stat: s.stat}
}

type SmoothGeom struct{}

func (s SmoothGeom) Name() string                 { return "Smooth" }
func (s SmoothGeom) RequiredAesthetics() []string { return []string{"x", "y"} }

// Smooth initializes a Smoothed geometry layer, defaulting the statistical pass provided.
func Smooth(statMethod ast.Stat) *SmoothBuilder {
	return &SmoothBuilder{
		geom: SmoothGeom{},
		stat: statMethod,
	}
}
