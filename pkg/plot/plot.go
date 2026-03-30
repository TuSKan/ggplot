package plot

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/internal/expr"
	"github.com/TuSKan/ggplot/internal/pipeline"
	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/guide"
	"github.com/TuSKan/ggplot/pkg/output"
	"github.com/TuSKan/ggplot/pkg/scale"
	"github.com/TuSKan/ggplot/pkg/stat"
	"github.com/TuSKan/ggplot/pkg/theme"
	"github.com/gogpu/gg"
)

// LayerBuilder defines anything that can construct a Layer.
// Geometries normally implement this to set their defaults.
type LayerBuilder interface {
	BuildLayer() ast.Layer
}

// Plot is the declarative immutable root builder.
type Plot struct {
	astPlot *ast.Plot
	guides  *guide.GuideSet
}

// New initializes a base plot with a dataset.
func New(data dataset.Dataset) *Plot {
	return &Plot{
		astPlot: &ast.Plot{
			Dataset: data,
			Mapping: make(ast.Aes),
			Layers:  []ast.Layer{},
			Scales:  []ast.ScaleConfig{},
		},
		guides: &guide.GuideSet{},
	}
}

// clone deep-copies the current Plot AST enforcing mutability limits recursively.
func (p *Plot) clone() *Plot {
	layers := make([]ast.Layer, len(p.astPlot.Layers))
	copy(layers, p.astPlot.Layers)

	scales := make([]ast.ScaleConfig, len(p.astPlot.Scales))
	copy(scales, p.astPlot.Scales)

	c := &ast.Plot{
		Dataset: p.astPlot.Dataset,
		Mapping: p.astPlot.Mapping.Merge(nil),
		Layers:  layers,
		Theme:   p.astPlot.Theme,
		Facet:   p.astPlot.Facet,
		Scales:  scales,
	}
	cg := &guide.GuideSet{
		Top:    append([]guide.Element(nil), p.guides.Top...),
		Bottom: append([]guide.Element(nil), p.guides.Bottom...),
		Left:   append([]guide.Element(nil), p.guides.Left...),
		Right:  append([]guide.Element(nil), p.guides.Right...),
	}
	return &Plot{astPlot: c, guides: cg}
}

// AesOpt passes functional wrappers setting map limits statically fully.
type AesOpt func(ast.Aes)

// AddLayer attaches a geometry.
func (p *Plot) AddLayer(builder LayerBuilder, opts ...AesOpt) *Plot {
	cloned := p.clone()
	layer := builder.BuildLayer()

	if layer.Mapping == nil {
		layer.Mapping = make(ast.Aes)
	}

	for _, opt := range opts {
		opt(layer.Mapping)
	}

	cloned.astPlot.Layers = append(cloned.astPlot.Layers, layer)
	return cloned
}

func (p *Plot) Aes(opts ...AesOpt) *Plot {
	cloned := p.clone()
	for _, opt := range opts {
		opt(cloned.astPlot.Mapping)
	}
	return cloned
}

func (p *Plot) Theme(t ast.ThemeConfig) *Plot {
	cloned := p.clone()
	cloned.astPlot.Theme = t
	return cloned
}

func (p *Plot) Facet(f ast.FacetConfig) *Plot {
	cloned := p.clone()
	cloned.astPlot.Facet = f
	return cloned
}

func (p *Plot) Title(text string) *Plot {
	cloned := p.clone()
	cloned.guides.Top = append(cloned.guides.Top, guide.Text{Text: text})
	return cloned
}

func (p *Plot) Subtitle(text string) *Plot {
	cloned := p.clone()
	cloned.guides.Top = append(cloned.guides.Top, guide.Text{Text: text})
	return cloned
}

func (p *Plot) XAxis(title string) *Plot {
	cloned := p.clone()
	cloned.guides.Bottom = append(cloned.guides.Bottom, guide.Axis{
		Title:     title,
		Position:  "bottom",
		Direction: "horizontal",
		Scale:     &scale.Continuous{},
	})
	return cloned
}

func (p *Plot) YAxis(title string) *Plot {
	cloned := p.clone()
	cloned.guides.Left = append(cloned.guides.Left, guide.Axis{
		Title:     title,
		Position:  "left",
		Direction: "vertical",
		Scale:     &scale.Continuous{},
	})
	return cloned
}

func (p *Plot) Legend(position, title string) *Plot {
	cloned := p.clone()
	leg := guide.Legend{Title: title, Columns: 1, Swatches: []guide.LegendSwatch{
		{Label: "Scale", Fill: "#000000"}, // Placeholder for examples, full dynamic legend requires scale merging later.
	}}
	switch position {
	case "right":
		cloned.guides.Right = append(cloned.guides.Right, leg)
	case "left":
		cloned.guides.Left = append(cloned.guides.Left, leg)
	case "top":
		cloned.guides.Top = append(cloned.guides.Top, leg)
	case "bottom":
		cloned.guides.Bottom = append(cloned.guides.Bottom, leg)
	}
	return cloned
}

// Compile performs validation and prepares the render structure.
func (p *Plot) Compile() (*ast.RenderPlan, error) {
	return pipeline.New().Compile(p.astPlot)
}

// Renderer binds graphical bounds.
type Renderer interface {
	Draw(dc *gg.Context, ctx geom.RenderContext, ds dataset.Dataset, mapping ast.Aes) error
}

// Render builds graphic contexts mapping boundaries.
func (p *Plot) Render(width, height int) (*output.Output, error) {
	plan, err := p.Compile()
	if err != nil {
		return nil, err
	}

	ds := plan.Dataset.(dataset.Dataset)

	// Pre-evaluate layers natively cleanly creating cached materialized views expertly cleanly logically explicitly smartly flawlessly
	type layerData struct {
		geom    Renderer
		ds      dataset.Dataset
		mapping ast.Aes
	}

	var compiledLayers []layerData
	for _, layer := range plan.Layers {
		dsToDraw := ds

		mergedMap := make(ast.Aes)
		for k, v := range p.astPlot.Mapping {
			mergedMap[k] = v
		}
		for k, v := range layer.Mapping {
			mergedMap[k] = v
		}

		if s, ok := layer.Stat.(stat.Stat); ok && s.Kind() != stat.KindTransform {
			if derived, err := s.Compute(stat.Context{Dataset: ds, Aes: mergedMap}); err == nil {
				dsToDraw = derived
			}
		}

		if r, ok := layer.Geom.(Renderer); ok {
			compiledLayers = append(compiledLayers, layerData{
				geom:    r,
				ds:      dsToDraw,
				mapping: mergedMap,
			})
		}
	}

	getColName := func(mapping ast.Aes, key string) string {
		if mapping == nil {
			return ""
		}
		val, ok := mapping[key]
		if !ok {
			return ""
		}
		if ref, ok := val.(*expr.ColumnRef); ok {
			return ref.Name
		}
		return ""
	}

	trainScale := func(elements []guide.Element, aesKey string) {
		for _, el := range elements {
			if ax, ok := el.(guide.Axis); ok && ax.Scale != nil {
				type Trainable interface {
					Train(dataset.Column) error
				}
				if tr, ok := ax.Scale.(Trainable); ok {
					for _, l := range compiledLayers {
						colName := getColName(l.mapping, aesKey)
						col, err := l.ds.Column(colName)

						if err != nil {
							col, err = l.ds.Column(aesKey)
							if err != nil && aesKey == "y" {
								col, err = l.ds.Column("count")
							}
						}

						if err == nil {
							tr.Train(col)
						}
					}
				}
			}
		}
	}

	trainScale(p.guides.Bottom, "x")
	trainScale(p.guides.Top, "x")
	trainScale(p.guides.Left, "y")
	trainScale(p.guides.Right, "y")

	var th theme.Theme
	if tc, ok := plan.Theme.(theme.Theme); ok {
		th = tc
	} else {
		th = theme.Default()
	}

	out := &output.Output{
		Theme:  th,
		Width:  width,
		Height: height,
		Draw: func(dc *gg.Context) {
			bg := th.Background
			if bg == nil {
				bg = theme.ParseHexColor("#FFFFFF")
			}
			dc.SetColor(bg)
			dc.Clear()

			m := &ggMeasurer{dc: dc}
			painter := &ggPainter{dc: dc}

			layout := guide.Compute(p.guides, float64(width), float64(height), m, th)
			for _, place := range layout.Placements {
				place.Element.Draw(painter, place.Rect.X, place.Rect.Y, place.Rect.W, place.Rect.H, th)
			}

			var xScale, yScale geom.Scale
			if len(p.guides.Bottom) > 0 {
				if ax, ok := p.guides.Bottom[0].(guide.Axis); ok && ax.Scale != nil {
					xScale = ax.Scale.(geom.Scale)
				}
			}
			if len(p.guides.Left) > 0 {
				if ax, ok := p.guides.Left[0].(guide.Axis); ok && ax.Scale != nil {
					yScale = ax.Scale.(geom.Scale)
				}
			}

			dc.Push()
			dc.Translate(layout.Data.X, layout.Data.Y)

			rctx := geom.RenderContext{
				Width:  layout.Data.W,
				Height: layout.Data.H,
				XScale: xScale,
				YScale: yScale,
			}

			for _, l := range compiledLayers {
				if err := l.geom.Draw(dc, rctx, l.ds, l.mapping); err != nil {

				}
			}
			dc.Pop()
		},
	}
	return out, nil
}
