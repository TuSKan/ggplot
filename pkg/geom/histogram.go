package geom

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/stat"
	"github.com/gogpu/gg"
)

// HistogramBuilder constructs an ast.Layer for distribution Histogram shapes.
type HistogramBuilder struct {
	geom HistogramGeom
	stat ast.Stat
}

func (h *HistogramBuilder) BuildLayer() ast.Layer {
	return ast.Layer{
		Geom: h.geom,
		Stat: h.stat,
	}
}

// HistogramGeom encapsulates binned distributions.
type HistogramGeom struct {
	Opts Opts
}

func (h HistogramGeom) Name() string                 { return "Histogram" }
func (h HistogramGeom) RequiredAesthetics() []string { return []string{"x"} }
func (h HistogramGeom) DefaultStat() string          { return "bin" }

func (h HistogramGeom) Draw(dc *gg.Context, ctx RenderContext, ds dataset.Dataset, mapping ast.Aes) error {
	colX, errX := ds.Column("x")
	colCount, errCount := ds.Column("count")
	if errX != nil || errCount != nil {
		return nil
	}

	iterXCol, ok1 := colX.(dataset.IterableColumn)
	iterCountCol, ok2 := colCount.(dataset.IterableColumn)
	if !ok1 || !ok2 {
		return nil
	}

	iterX, err1 := iterXCol.Float64s()
	iterCount, err2 := iterCountCol.Float64s()
	if err1 != nil || err2 != nil {
		return nil
	}
	
	minX, _ := dataset.Min(colX)
	maxX, _ := dataset.Max(colX)
	minY, _ := dataset.Min(colCount)
	maxY, _ := dataset.Max(colCount)

	proj := Projector{Width: ctx.Width, Height: ctx.Height, Padding: 0.0}

	op := 1.0
	if h.Opts.Opacity > 0 {
		op = h.Opts.Opacity
	}

	type binData struct {
		x     float64
		count float64
	}
	var binsList []binData
	var maxCount float64

	for {
		x, isNull1, ok1 := iterX.Next()
		c, isNull2, ok2 := iterCount.Next()
		if !ok1 || !ok2 {
			break
		}
		if !isNull1 && !isNull2 {
			binsList = append(binsList, binData{x, c})
			if c > maxCount {
				maxCount = c
			}
		}
	}

	bins := len(binsList)
	if bins == 0 || maxCount == 0 {
		return nil
	}

	binWidthPixels := ctx.Width / float64(bins)

	fr, fg, fb := ParseHexColor(h.Opts.Fill, 0.2, 0.4, 0.8)
	cr, cg, cb := ParseHexColor(h.Opts.Color, 0.0, 0.0, 0.0)

	for _, b := range binsList {
		if b.count == 0 {
			continue
		}

		normX := ctx.ProjectX(b.x, minX, maxX)
		normY := ctx.ProjectY(b.count, minY, maxY)

		xPos := proj.X(normX) - (binWidthPixels / 2.0)
		yPos := proj.Y(normY)

		baseY := proj.Y(ctx.ProjectY(0.0, minY, maxY))
		hVal := baseY - yPos

		dc.SetRGBA(fr, fg, fb, op)
		dc.DrawRectangle(xPos, yPos, binWidthPixels, hVal)
		dc.Fill()

		if h.Opts.Color != "" {
			dc.SetRGBA(cr, cg, cb, 1.0)
			dc.DrawRectangle(xPos, yPos, binWidthPixels, hVal)
			if lw := h.Opts.LineWidth; lw > 0 {
				dc.SetLineWidth(lw)
			} else {
				dc.SetLineWidth(1.0)
			}
			dc.Stroke()
		}
	}

	return nil
}

// Histogram returns an unmapped area distribution layer matching stats functionally.
func Histogram(args ...any) *HistogramBuilder {
	o := Opts{}
	var s ast.Stat = stat.MethodBin
	
	for _, arg := range args {
		if opt, ok := arg.(Opts); ok {
			o = opt
		} else if st, ok := arg.(ast.Stat); ok {
			s = st
		}
	}
	
	return &HistogramBuilder{
		geom: HistogramGeom{Opts: o},
		stat: s,
	}
}
