package geom

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/stat"
	"github.com/gogpu/gg"
)

// BarBuilder constructs an ast.Layer for Bar geometries.
type BarBuilder struct {
	geom BarGeom
	stat ast.Stat
}

func (b *BarBuilder) BuildLayer() ast.Layer {
	return ast.Layer{
		Geom: b.geom,
		Stat: b.stat,
	}
}

// BarGeom encapsulates column/bar primitives.
type BarGeom struct {
	Opts Opts
}

func (b BarGeom) Name() string                 { return "Bar" }
func (b BarGeom) RequiredAesthetics() []string { return []string{"x"} } // Y typically computed or mapped
func (b BarGeom) DefaultStat() string          { return "count" }

func (p BarGeom) Draw(dc *gg.Context, ctx RenderContext, ds dataset.Dataset, mapping ast.Aes) error {
	coords, err := ResolveCoordinates(ds, mapping)
	if err != nil {
		return err
	}

	if coords.X == nil {
		return nil
	}
	
	var iterCount dataset.Float64Iterator
	var minY, maxY float64
	if colCount, err := ds.Column("count"); err == nil {
		if iter, ok := colCount.(dataset.IterableColumn); ok {
			iterCount, _ = iter.Float64s()
		}
		minY, _ = dataset.Min(colCount)
		maxY, _ = dataset.Max(colCount)
	}

	minX, maxX, _, _, _, _ := ResolveBounds(ds, mapping)
	if minX == 0 && maxX == 0 {
		if colX, err := ds.Column("x"); err == nil {
			minX, _ = dataset.Min(colX)
			maxX, _ = dataset.Max(colX)
		}
	}

	proj := Projector{Width: ctx.Width, Height: ctx.Height, Padding: 0.0}

	op := 0.8
	if p.Opts.Opacity > 0 {
		op = p.Opts.Opacity
	}

	totalBars := ds.Len()
	if totalBars <= 0 {
		return nil
	}

	binPixels := (proj.Width - 2.0*proj.Padding) / float64(totalBars)

	var relWidth float64 = 0.8
	if p.Opts.Width > 0 && p.Opts.Width <= 1.0 {
		relWidth = p.Opts.Width
	}

	barW := binPixels * relWidth

	fr, fg, fb := ParseHexColor(p.Opts.Fill, 0.2, 0.4, 0.8)

	// In a statically-typed environment we fetch globally safely
	// We'll peek maxCount to normalize intelligently purely dynamically
	maxCount := 1.0
	var counts []float64
	var xs []float64
	
	for {
		x, _, ok1 := coords.X.Next()
		if !ok1 {
			break
		}
		y := 0.5
		if coords.Y != nil {
			yVal, _, _ := coords.Y.Next()
			y = yVal
		} else if iterCount != nil {
			yVal, _, _ := iterCount.Next()
			y = yVal
		}
		
		if y > maxCount {
			maxCount = y
		}
		xs = append(xs, x)
		counts = append(counts, y)
	}

	for i := 0; i < len(xs); i++ {
		x := xs[i]
		y := counts[i]

		normX := ctx.ProjectX(x, minX, maxX)
		
		normY := ctx.ProjectY(y, minY, maxY)

		screenX := proj.X(normX)
		
		screenY := proj.Y(normY)
		baseY := proj.Y(ctx.ProjectY(0.0, minY, maxY))
		barH := baseY - screenY

		dc.SetRGBA(fr, fg, fb, op)
		dc.DrawRectangle(screenX-barW/2.0, screenY, barW, barH)
		dc.Fill()

		cr, cg, cb := ParseHexColor(p.Opts.Color, 0.0, 0.0, 0.0) 
		if p.Opts.Color != "" {                                  
			dc.SetRGBA(cr, cg, cb, 1.0)
			if lw := p.Opts.LineWidth; lw > 0 {
				dc.SetLineWidth(lw)
			} else {
				dc.SetLineWidth(1.0)
			}
			dc.Stroke()
		}
	}

	return nil
}

// Bar returns an unmapped Bar geometry layer builder carrying default 'count' stats.
func Bar(opts ...Opts) *BarBuilder {
	o := Opts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	return &BarBuilder{
		geom: BarGeom{Opts: o},
		stat: stat.MethodCount, // Requires stats implementation
	}
}
