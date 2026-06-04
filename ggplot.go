// Package ggplot is a production-grade, pure-Go Grammar of Graphics plotting library.
//
// Inspired by R's ggplot2, it provides a declarative, composable API for
// building statistical visualizations from data, aesthetics, geometries,
// scales, coordinate systems, facets, and themes.
//
// # Quick Start
//
//	p := ggplot.New(ds,
//	    aes.X("x"),
//	    aes.Y("y"),
//	    aes.Color("group"),
//	).
//	    Layer(geom.Point(geom.WithSize(4), geom.WithAlpha(0.7))).
//	    Layer(geom.Smooth()).
//	    Labels(ggplot.Title("My Plot"), ggplot.XLabel("X Axis")).
//	    Theme("minimal").
//	    Save("output.png", 1200, 800)
//
// # Architecture
//
// The library follows a strict pipeline:
//
//	PlotSpec -> Validate -> Stat Transform -> Scale Training -> Layout -> Render
//
// All data flows through the [dataset.Dataset] abstraction. Multiple engine
// backends are supported: memory (Go slices), Apache Arrow (columnar arrays),
// and BigQuery (SQL pushdown). Arrow IPC and Parquet ingest provide zero-copy
// reads; constructing from Go slices requires one copy.
package ggplot

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"maps"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/gogpu/gg"
	"golang.org/x/sync/errgroup"

	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/colormap"
	"github.com/TuSKan/ggplot/coord"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/facet"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output"
	"github.com/TuSKan/ggplot/scale"
	"github.com/TuSKan/ggplot/stat"
	"github.com/TuSKan/ggplot/theme"

	// Register the built-in output surfaces so Save/Encode/Image work without
	// the caller needing a blank import.
	_ "github.com/TuSKan/ggplot/output/file"
	_ "github.com/TuSKan/ggplot/output/image"
)

// Plot is the immutable, declarative plot builder. Every method returns a new
// Plot with the modification applied, enabling a fluent chaining style.
//
// Plot is safe to share and reuse - modifying a derived plot does not
// affect the original.
type Plot struct {
	spec PlotSpec
}

// LegendPos controls legend placement.
type LegendPos string

// LegendRight places the legend to the right of the plot.
const (
	LegendRight  LegendPos = "right"
	LegendLeft   LegendPos = "left"
	LegendTop    LegendPos = "top"
	LegendBottom LegendPos = "bottom"
	LegendNone   LegendPos = "none"
)

// New initializes a plot with a dataset and optional global aesthetic mappings.
func New(ds dataset.Dataset, globalAes ...aes.Mapping) *Plot {
	return &Plot{
		spec: PlotSpec{
			Dataset:        ds,
			GlobalMapping:  ToAesMap(globalAes),
			Layers:         nil,
			ScaleOverrides: make(map[string]ScaleOverride),
			Coord:          coord.Cartesian(),
			Facet:          facet.None(),
		},
	}
}

// clone creates a deep copy of the plot spec for immutability.
func (p *Plot) clone() *Plot {
	// Deep-clone layers: each LayerSpec.Mapping must be independent.
	layers := make([]LayerSpec, len(p.spec.Layers))
	for i, l := range p.spec.Layers {
		m := make(AesMap, len(l.Mapping))
		maps.Copy(m, l.Mapping)

		layers[i] = LayerSpec{Geom: l.Geom, Mapping: m}
	}

	// Deep-clone scale overrides (ScaleOverride.Params is a map, Opts is a slice).
	scales := make(map[string]ScaleOverride, len(p.spec.ScaleOverrides))
	for k, v := range p.spec.ScaleOverrides {
		params := make(map[string]string, len(v.Params))
		maps.Copy(params, v.Params)

		opts := make([]scale.Opt, len(v.Opts))
		copy(opts, v.Opts)
		scales[k] = ScaleOverride{Type: v.Type, Params: params, Opts: opts}
	}

	// ColorScales hold pointer values; clone the map but share the pointed-to
	// Scale (Scale itself is treated as user-supplied - replacing the entry
	// for an aesthetic always installs a fresh value).
	var colorScales map[string]*colormap.Scale
	if len(p.spec.ColorScales) > 0 {
		colorScales = make(map[string]*colormap.Scale, len(p.spec.ColorScales))
		maps.Copy(colorScales, p.spec.ColorScales)
	}

	return &Plot{
		spec: PlotSpec{
			Dataset:        p.spec.Dataset,
			GlobalMapping:  p.spec.GlobalMapping.Merge(nil),
			Layers:         layers,
			ScaleOverrides: scales,
			ColorScales:    colorScales,
			Coord:          p.spec.Coord,
			Facet:          p.spec.Facet,
			ThemeName:      p.spec.ThemeName,
			Labels:         p.spec.Labels,
			XLim:           p.spec.XLim,
			YLim:           p.spec.YLim,
			LegendPosition: p.spec.LegendPosition,
			AxisGuideX:     p.spec.AxisGuideX,
			ColorBarWidth:  p.spec.ColorBarWidth,
			ColorBarNBin:   p.spec.ColorBarNBin,
			LegendNCols:    p.spec.LegendNCols,
			SizeScale:      p.spec.SizeScale,
			AlphaScale:     p.spec.AlphaScale,
			ShapeScale:     p.spec.ShapeScale,
			LinetypeScale:  p.spec.LinetypeScale,
			Annotations:    slices.Clone(p.spec.Annotations),
			SecondAxis:     p.spec.SecondAxis,
			ThemeOverrides: slices.Clone(p.spec.ThemeOverrides),
			PlotMargin:     p.spec.PlotMargin,
			Alignment:      p.spec.Alignment,
		},
	}
}

// ScaleColor configures the color aesthetic to use the given colormap.
// The cmap is composed with a default LinearNorm for continuous data, or
// used as a Listed palette for discrete data. Pass nil to clear an existing
// override and fall back to defaults.
func (p *Plot) ScaleColor(c colormap.Cmap) *Plot {
	cloned := p.clone()
	if cloned.spec.ColorScales == nil {
		cloned.spec.ColorScales = make(map[string]*colormap.Scale)
	}

	if c == nil {
		delete(cloned.spec.ColorScales, "color")
		return cloned
	}

	if _, ok := c.(*colormap.ListedCmap); ok {
		cloned.spec.ColorScales["color"] = colormap.NewDiscrete(c)
	} else {
		cloned.spec.ColorScales["color"] = colormap.NewContinuous(c, nil)
	}

	return cloned
}

// ScaleFill is the fill-aesthetic counterpart of [Plot.ScaleColor].
func (p *Plot) ScaleFill(c colormap.Cmap) *Plot {
	cloned := p.clone()
	if cloned.spec.ColorScales == nil {
		cloned.spec.ColorScales = make(map[string]*colormap.Scale)
	}

	if c == nil {
		delete(cloned.spec.ColorScales, "fill")
		return cloned
	}

	if _, ok := c.(*colormap.ListedCmap); ok {
		cloned.spec.ColorScales["fill"] = colormap.NewDiscrete(c)
	} else {
		cloned.spec.ColorScales["fill"] = colormap.NewContinuous(c, nil)
	}

	return cloned
}

// ScaleColorManual maps category labels to specific colors. Categories not
// in m fall back to the default Tab10 palette in the order they are
// encountered during dataset training.
func (p *Plot) ScaleColorManual(m map[string]colormap.Color) *Plot {
	cloned := p.clone()
	if cloned.spec.ColorScales == nil {
		cloned.spec.ColorScales = make(map[string]*colormap.Scale)
	}

	cloned.spec.ColorScales["color"] = colormap.NewManual(m)

	return cloned
}

// ScaleColorContinuous installs an explicit continuous color scale composed
// of the given Cmap and Norm. Use this to control LogNorm / TwoSlopeNorm /
// PowerNorm / data-range limits beyond the simple [Plot.ScaleColor] form.
func (p *Plot) ScaleColorContinuous(c colormap.Cmap, n colormap.Norm) *Plot {
	cloned := p.clone()
	if cloned.spec.ColorScales == nil {
		cloned.spec.ColorScales = make(map[string]*colormap.Scale)
	}

	cloned.spec.ColorScales["color"] = colormap.NewContinuous(c, n)

	return cloned
}

// Layer adds a geometry layer to the plot with optional per-layer aesthetic overrides.
func (p *Plot) Layer(l geom.Layer, localAes ...aes.Mapping) *Plot {
	cloned := p.clone()

	// Merge geom-level mapping with explicit per-layer overrides.
	mapping := make(AesMap)
	maps.Copy(mapping, l.Mapping)

	for _, am := range localAes {
		mapping[am.Channel] = am.Column
	}

	cloned.spec.Layers = append(cloned.spec.Layers, LayerSpec{
		Geom:    l,
		Mapping: mapping,
	})

	return cloned
}

// Aes adds or overrides global aesthetic mappings.
func (p *Plot) Aes(mappings ...aes.Mapping) *Plot {
	cloned := p.clone()
	for _, m := range mappings {
		cloned.spec.GlobalMapping[m.Channel] = m.Column
	}

	return cloned
}

// Coord sets the coordinate system.
func (p *Plot) Coord(c coord.Coord) *Plot {
	cloned := p.clone()
	cloned.spec.Coord = c

	return cloned
}

// FacetWrap applies wrap faceting by a column.
func (p *Plot) FacetWrap(col string, opts ...facet.WrapOpt) *Plot {
	cloned := p.clone()
	cloned.spec.Facet = facet.Wrap(col, opts...)

	return cloned
}

// FacetGrid applies grid faceting by row and column variables.
func (p *Plot) FacetGrid(rowCol, colCol string, opts ...facet.GridOpt) *Plot {
	cloned := p.clone()
	cloned.spec.Facet = facet.Grid(rowCol, colCol, opts...)

	return cloned
}

// Theme sets the visual theme.
func (p *Plot) Theme(name theme.Name) *Plot {
	cloned := p.clone()
	cloned.spec.ThemeName = name

	return cloned
}

// ThemeOverride applies per-plot theme element overrides.
// These are applied after the base theme is resolved, allowing fine-grained
// control over individual elements without creating a custom theme.
//
// Example — bold X-axis title:
//
//	p.ThemeOverride(theme.AxisTitleXOverride(theme.ElementText{Bold: true}))
func (p *Plot) ThemeOverride(overrides ...theme.Override) *Plot {
	cloned := p.clone()
	cloned.spec.ThemeOverrides = append(cloned.spec.ThemeOverrides, overrides...)

	return cloned
}

// PlotMargin sets explicit plot margins with unit support.
// Units can be theme.Pt, theme.Cm, theme.Inches, or theme.Lines.
//
// Example — 1cm top margin, 0.5in right margin:
//
//	p.PlotMargin(theme.PlotMargin{
//	    Top:   theme.Cm(1),
//	    Right: theme.Inches(0.5),
//	})
func (p *Plot) PlotMargin(m theme.PlotMargin) *Plot {
	cloned := p.clone()
	cloned.spec.PlotMargin = &m

	return cloned
}

// XLim sets explicit x-axis limits. Pass math.NaN() for either end to auto-detect.
func (p *Plot) XLim(lo, hi float64) *Plot {
	cloned := p.clone()
	cloned.spec.XLim = [2]*float64{new(lo), new(hi)}

	return cloned
}

// YLim sets explicit y-axis limits. Pass math.NaN() for either end to auto-detect.
func (p *Plot) YLim(lo, hi float64) *Plot {
	cloned := p.clone()
	cloned.spec.YLim = [2]*float64{new(lo), new(hi)}

	return cloned
}

// CoordFlip swaps the x and y axes. This is sugar for setting
// [geom.Horizontal] orientation on all layers and swapping the axis labels.
func (p *Plot) CoordFlip() *Plot {
	cloned := p.clone()
	for i := range cloned.spec.Layers {
		cloned.spec.Layers[i].Geom.Params.Orientation = geom.Horizontal
	}

	cloned.spec.Labels.X, cloned.spec.Labels.Y = cloned.spec.Labels.Y, cloned.spec.Labels.X

	return cloned
}

// CoordCartesian sets a Cartesian viewport zoom. Unlike [Plot.XLim]/[Plot.YLim]
// which set scale bounds early (potentially affecting stat computations),
// CoordCartesian overrides bounds after scale training — all data participates
// in stat computations, only the visible window changes.
//
// Pass math.NaN() for any endpoint to auto-detect from training.
func (p *Plot) CoordCartesian(xmin, xmax, ymin, ymax float64) *Plot {
	cloned := p.clone()

	xlim := [2]*float64{ptrFloat(xmin), ptrFloat(xmax)}
	ylim := [2]*float64{ptrFloat(ymin), ptrFloat(ymax)}

	cloned.spec.Coord = coord.CartesianZoom(xlim, ylim)

	return cloned
}

// CoordFixed sets a Cartesian coordinate system with a fixed aspect ratio.
// ratio is defined as (pixels per data-unit-y) / (pixels per data-unit-x).
// ratio = 1 gives equal scaling — one unit of x occupies the same pixel
// length as one unit of y.
func (p *Plot) CoordFixed(ratio float64) *Plot {
	cloned := p.clone()
	cloned.spec.Coord = coord.Fixed(ratio)

	return cloned
}

// CoordTrans sets a Cartesian coordinate system with per-axis mathematical
// transforms applied post-stat. Unlike scale transforms (e.g., scale.Log10)
// which transform data before stat computations, CoordTrans transforms
// the display without affecting statistics.
func (p *Plot) CoordTrans(xtrans, ytrans coord.TransFunc) *Plot {
	cloned := p.clone()
	cloned.spec.Coord = coord.Trans(xtrans, ytrans)

	return cloned
}

// ptrFloat returns a pointer to v, or nil if v is NaN.
func ptrFloat(v float64) *float64 {
	if math.IsNaN(v) {
		return nil
	}

	return &v
}

// LegendPosition sets the legend placement.
func (p *Plot) LegendPosition(pos LegendPos) *Plot {
	cloned := p.clone()
	cloned.spec.LegendPosition = string(pos)

	return cloned
}

// Align sets block-level alignment for titles, labels, and legend.
// Non-empty fields override the theme's default alignment.
//
// Example — left-aligned titles, right-aligned caption:
//
//	p.Align(theme.BlockAlignment{
//	    Title:   theme.AlignLeft,
//	    Caption: theme.AlignRight,
//	})
func (p *Plot) Align(a theme.BlockAlignment) *Plot {
	cloned := p.clone()
	cloned.spec.Alignment = &a

	return cloned
}

// ColorBarWidth sets the width of the continuous color bar in pixels.
// Zero resets to default (12px).
func (p *Plot) ColorBarWidth(w float64) *Plot {
	cloned := p.clone()
	cloned.spec.ColorBarWidth = w

	return cloned
}

// ColorBarNBin sets the number of discrete gradient steps in the color bar.
// Zero resets to default (256).
func (p *Plot) ColorBarNBin(n int) *Plot {
	cloned := p.clone()
	cloned.spec.ColorBarNBin = n

	return cloned
}

// LegendCols sets the number of columns for the categorical legend.
// Zero resets to single column (vertical) or single row (horizontal).
func (p *Plot) LegendCols(n int) *Plot {
	cloned := p.clone()
	cloned.spec.LegendNCols = n

	return cloned
}

// AxisLabelRows controls the X-axis label layout. N sets the number of
// stagger rows for tick labels. When n is 0 (default), overlapping labels
// are automatically detected and staggered across 2 rows. Set n to 1 to
// disable staggering. Values ≥ 2 force that many rows.
func (p *Plot) AxisLabelRows(n int) *Plot {
	cloned := p.clone()
	cloned.spec.AxisGuideX.NDodge = n

	return cloned
}

// ScaleX sets the x-axis scale type with optional configuration.
// Options: [scale.WithBreaks], [scale.WithLabels], [scale.WithFormatter],
// [scale.WithExpand], [scale.WithMinorBreaks], [scale.WithClipBounds].
func (p *Plot) ScaleX(scaleType scale.Type, opts ...scale.Opt) *Plot {
	cloned := p.clone()
	cloned.spec.ScaleOverrides["x"] = ScaleOverride{Type: scaleType, Opts: opts}

	return cloned
}

// ScaleY sets the y-axis scale type with optional configuration.
// Options: [scale.WithBreaks], [scale.WithLabels], [scale.WithFormatter],
// [scale.WithExpand], [scale.WithMinorBreaks], [scale.WithClipBounds].
func (p *Plot) ScaleY(scaleType scale.Type, opts ...scale.Opt) *Plot {
	cloned := p.clone()
	cloned.spec.ScaleOverrides["y"] = ScaleOverride{Type: scaleType, Opts: opts}

	return cloned
}

// ScaleSize configures a size scale mapping to the specified point-radius range.
func (p *Plot) ScaleSize(rangeMin, rangeMax float64) *Plot {
	cloned := p.clone()
	cloned.spec.SizeScale = scale.NewSize(rangeMin, rangeMax)

	return cloned
}

// ScaleSizeArea configures a size scale mapping values proportionally to point area.
func (p *Plot) ScaleSizeArea() *Plot {
	cloned := p.clone()
	cloned.spec.SizeScale = scale.NewSizeArea()

	return cloned
}

// ScaleAlpha configures an opacity scale mapping to the specified range.
func (p *Plot) ScaleAlpha(rangeMin, rangeMax float64) *Plot {
	cloned := p.clone()
	cloned.spec.AlphaScale = scale.NewAlpha(rangeMin, rangeMax)

	return cloned
}

// ScaleShape configures a categorical shape scale with automatic shape assignment.
func (p *Plot) ScaleShape() *Plot {
	cloned := p.clone()
	cloned.spec.ShapeScale = scale.NewShape()

	return cloned
}

// ScaleShapeManual configures a categorical shape scale with a manual mapping.
func (p *Plot) ScaleShapeManual(m map[string]string) *Plot {
	cloned := p.clone()
	cloned.spec.ShapeScale = scale.NewShapeManual(m)

	return cloned
}

// ScaleLinetype configures a categorical linetype scale with automatic dash pattern assignment.
func (p *Plot) ScaleLinetype() *Plot {
	cloned := p.clone()
	cloned.spec.LinetypeScale = scale.NewLinetype()

	return cloned
}

// ScaleLinetypeManual configures a categorical linetype scale with a manual mapping.
func (p *Plot) ScaleLinetypeManual(m map[string]string) *Plot {
	cloned := p.clone()
	cloned.spec.LinetypeScale = scale.NewLinetypeManual(m)

	return cloned
}

// ScaleSizeIdentity configures an identity scale for size, passing values through.
func (p *Plot) ScaleSizeIdentity() *Plot {
	cloned := p.clone()
	cloned.spec.SizeScale = scale.NewIdentity()

	return cloned
}

// ScaleAlphaIdentity configures an identity scale for alpha, passing values through.
func (p *Plot) ScaleAlphaIdentity() *Plot {
	cloned := p.clone()
	cloned.spec.AlphaScale = scale.NewIdentity()

	return cloned
}

// Labels configures plot labels (title, subtitle, axis labels, caption, and legend titles).
func (p *Plot) Labels(opts ...LabelOpt) *Plot {
	cloned := p.clone()
	for _, opt := range opts {
		opt(&cloned.spec.Labels)
	}

	return cloned
}

// LabelOpt is a functional option for configuring plot labels.
type LabelOpt func(*Labels)

// Title sets the plot title.
func Title(text string) LabelOpt { return func(l *Labels) { l.Title = text } }

// Subtitle sets the plot subtitle.
func Subtitle(text string) LabelOpt { return func(l *Labels) { l.Subtitle = text } }

// XLabel sets the x-axis label.
func XLabel(text string) LabelOpt { return func(l *Labels) { l.X = text } }

// YLabel sets the y-axis label.
func YLabel(text string) LabelOpt { return func(l *Labels) { l.Y = text } }

// Caption sets the plot caption.
func Caption(text string) LabelOpt { return func(l *Labels) { l.Caption = text } }

// ColorLabel sets the title for the categorical/continuous color legend.
// When set, this overrides the default column-name title.
func ColorLabel(text string) LabelOpt { return func(l *Labels) { l.Color = text } }

// SizeLabel sets the title for the graduated-size legend.
// When set, this overrides the default column-name title.
func SizeLabel(text string) LabelOpt { return func(l *Labels) { l.Size = text } }

// AlphaLabel sets the title for the continuous-alpha legend.
// When set, this overrides the default column-name title.
func AlphaLabel(text string) LabelOpt { return func(l *Labels) { l.Alpha = text } }

// SecondAxis adds a secondary Y-axis derived from the primary Y-axis via a
// transform pair. The secondary axis is rendered on the right side of the
// plot with its own tick labels and optional axis title.
//
// Example — show Fahrenheit alongside Celsius:
//
//	p.SecondAxis(scale.SecAxis(
//	    func(c float64) float64 { return c*9/5 + 32 },
//	    func(f float64) float64 { return (f - 32) * 5 / 9 },
//	    "Temperature (°F)",
//	))
func (p *Plot) SecondAxis(spec scale.SecAxisSpec) *Plot {
	cloned := p.clone()
	cloned.spec.SecondAxis = &spec

	return cloned
}

// DrawCanvas creates a new [canvas.RasterCanvas] and draws the built plot onto it.
// The caller owns the returned canvas and must call [canvas.RasterCanvas.Close]
// when finished to release GPU resources.
func (b *Built) DrawCanvas(ctx context.Context, width, height int) (*canvas.RasterCanvas, error) {
	cv := canvas.NewRasterCanvas(width, height)
	if err := b.Draw(ctx, cv, width, height); err != nil {
		_ = cv.Close()

		return nil, err
	}

	return cv, nil
}

// goldenRatio is the golden ratio used for default aspect-ratio inference.
const goldenRatio = 1.618

// autoHeight infers a plot height from the given width and the y-scale type
// of the first panel:
//
//   - discrete y: 18px per category + 100px padding, clamped to [240, width]
//   - continuous y (default): width / φ (golden ratio ≈ 1.618)
func (b *Built) autoHeight(width int) int {
	if len(b.panels) > 0 {
		if ds, ok := b.panels[0].YScale.(*scale.DiscreteScale); ok {
			n := len(ds.Categories())
			// 18px per category + axis/title padding
			return min(max(18*n+100, 240), width)
		}
	}

	return int(float64(width) / goldenRatio)
}

// groupByColumn splits a Dataset into subsets by the distinct values in the
// given column. Returns ordered unique labels, corresponding filtered datasets,
// and any error encountered.
//
// Performance: uses strconv (not fmt.Sprintf) for numeric->string conversion
// and SelectRows (not BoolMask) for O(group_size) extraction without
// allocating O(n) bool masks per group.
func groupByColumn(_ context.Context, ds dataset.Dataset, colName string) ([]string, []dataset.Dataset, error) {
	col, err := ds.Column(colName)
	if err != nil {
		return nil, nil, Errorf(PhaseBuild, -1, "group", err, "column %q", colName)
	}

	// Extract string labels from the column.
	// Uses strconv instead of fmt.Sprintf (~5x faster per value).
	var vals []string

	switch tc := col.(type) {
	case dataset.Column[string]:
		vals = tc.Values()
	case dataset.Column[float64]:
		raw := tc.Values()

		vals = make([]string, len(raw))
		for i, v := range raw {
			vals[i] = strconv.FormatFloat(v, 'g', -1, 64)
		}
	case dataset.Column[int64]:
		raw := tc.Values()

		vals = make([]string, len(raw))
		for i, v := range raw {
			vals[i] = strconv.FormatInt(v, 10)
		}
	default:
		return nil, nil, Errorf(PhaseBuild, -1, "group", ErrRenderFailed, "unsupported column type %T for %q", col, colName)
	}

	// Build index groups: map[label] -> []rowIndex.
	groupIndices := make(map[string][]int)

	var order []string

	for i, v := range vals {
		if _, exists := groupIndices[v]; !exists {
			order = append(order, v)
		}

		groupIndices[v] = append(groupIndices[v], i)
	}

	// Extract subsets using direct index selection (no bool-mask allocation).
	subsets := make([]dataset.Dataset, len(order))
	for i, label := range order {
		subset, serr := ds.SelectRows(groupIndices[label])
		if serr != nil {
			return nil, nil, Errorf(PhaseBuild, -1, "group", serr, "select group %q", label)
		}

		subsets[i] = subset
	}

	return order, subsets, nil
}

// allLayersHorizontal returns true if every layer in the list has
// Orientation == geom.Horizontal. Returns false for empty lists.
func allLayersHorizontal(layers []LayerSpec) bool {
	if len(layers) == 0 {
		return false
	}

	for _, l := range layers {
		if l.Geom.Params.Orientation != geom.Horizontal {
			return false
		}
	}

	return true
}

// ---------------------------------------------------------------------------
// Build pipeline
// ---------------------------------------------------------------------------

// Built is the result of [Plot.Build]. It holds fully resolved layer data,
// trained scales, layout geometry, and theme — everything needed to draw
// without re-running the grammar pipeline.
//
// This is the Go equivalent of ggplot2's ggplot_build(plot) → built.
type Built struct {
	panels      []BuiltPanel
	layout      Layout
	coord       coord.Coord
	theme       theme.Theme
	labels      Labels
	legendPos   string
	ndodgeX     int // X-axis label dodge rows (0 = auto, 1 = no dodge, ≥ 2 = forced)
	annotations []Annotation
	secAxis     *scale.SecAxisSpec // secondary Y-axis specification (nil = none)

	// vp holds the interactive viewport state: cached panel geometry
	// (populated during Draw) and original trained bounds (captured on
	// first zoom). Protected by an internal mutex for concurrent
	// draw/event goroutine access.
	vp viewportState
}

// BuiltLayer holds one resolved layer's data after stat transform and grouping.
// The Data dataset always contains the system column PANEL (int64). When the
// layer was produced by group splitting, the system column group (int64) is
// also present.
type BuiltLayer struct {
	Geom          geom.Layer
	Data          dataset.Dataset
	Mapping       AesMap
	ContColorCol  string
	ContColScale  *colormap.Scale
	SizeCol       string
	SizeScale     scale.Scale
	AlphaCol      string
	AlphaScale    scale.Scale
	ShapeCol      string
	ShapeScale    scale.Scale
	LinetypeCol   string
	LinetypeScale scale.Scale
}

// BuiltPanel holds one facet panel with its resolved layers and trained scales.
type BuiltPanel struct {
	Label         string
	Layers        []BuiltLayer
	XScale        scale.Scale
	YScale        scale.Scale
	LegendEntries []LegendEntry
	LegendTitle   string
	ColorBarSpec  *ColorBarSpec
	SizeLegend    *SizeLegendSpec  // graduated-size legend (nil = not mapped)
	AlphaLegend   *AlphaLegendSpec // continuous-alpha legend (nil = not mapped)
	XIsDiscrete   bool
}

// Layout holds the panel grid dimensions derived from faceting.
type Layout struct {
	Rows   int
	Cols   int
	Panels []PanelLayout
	FreeX  bool // per-panel independent X scales
	FreeY  bool // per-panel independent Y scales
}

// PanelLayout holds per-panel geometry and trained scale state.
type PanelLayout struct {
	Row    int
	Col    int
	RowVal string // raw row facet value (Grid only, "" for Wrap/None)
	ColVal string // raw column facet value (Grid only, "" for Wrap/None)
	XScale scale.Scale
	YScale scale.Scale
}

// panelGridRow returns the grid row for a panel. For Grid facets (RowVal set),
// it looks up the row index map. For Wrap/None it falls back to pi / cols.
func panelGridRow(fp facet.Panel, idx map[string]int, pi, cols int) int {
	if fp.RowVal != "" {
		if r, ok := idx[fp.RowVal]; ok {
			return r
		}
	}

	return pi / cols
}

// panelGridCol returns the grid column for a panel. For Grid facets (ColVal set),
// it looks up the column index map. For Wrap/None it falls back to pi % cols.
func panelGridCol(fp facet.Panel, idx map[string]int, pi, cols int) int {
	if fp.ColVal != "" {
		if c, ok := idx[fp.ColVal]; ok {
			return c
		}
	}

	return pi % cols
}

// unifyPanelScales applies shared scale bounds across all panels based on
// the facet's FreeScales() configuration. Panels with shared scales get
// their bounds unified to the union of all panels.
func unifyPanelScales(f facet.Facet, panels []BuiltPanel, layouts []PanelLayout) {
	if len(panels) <= 1 {
		return
	}

	freeX, freeY := f.FreeScales()
	if !freeX {
		unifyScaleBounds(panels, layouts, true)
	}

	if !freeY {
		unifyScaleBounds(panels, layouts, false)
	}
}

// unifyScaleBounds computes the union [min, max] across all panels for either
// X (isX=true) or Y (isX=false) and overrides each panel's scale bounds.
func unifyScaleBounds(panels []BuiltPanel, layouts []PanelLayout, isX bool) {
	unionMin, unionMax := math.Inf(1), math.Inf(-1)

	for i := range panels {
		var sc scale.Scale
		if isX {
			sc = panels[i].XScale
		} else {
			sc = panels[i].YScale
		}

		mn, mx := sc.Bounds()
		if mn < unionMin {
			unionMin = mn
		}

		if mx > unionMax {
			unionMax = mx
		}
	}

	if math.IsInf(unionMin, 1) {
		return // no valid bounds
	}

	for i := range panels {
		var sc scale.Scale
		if isX {
			sc = panels[i].XScale
		} else {
			sc = panels[i].YScale
		}

		if bs, ok := sc.(scale.BoundsSetter); ok {
			bs.SetBounds(unionMin, unionMax)
		}

		// Update PanelLayout references too.
		if i < len(layouts) {
			if isX {
				layouts[i].XScale = sc
			} else {
				layouts[i].YScale = sc
			}
		}
	}
}

// LayerData returns the resolved dataset for the given layer in the given panel.
// This is the data each geom actually sees after stat/position transforms —
// the primary introspection API for debugging and platform-independent testing.
func (b *Built) LayerData(panel, layer int) dataset.Dataset {
	if panel < 0 || panel >= len(b.panels) {
		return dataset.Dataset{}
	}

	p := b.panels[panel]
	if layer < 0 || layer >= len(p.Layers) {
		return dataset.Dataset{}
	}

	return p.Layers[layer].Data
}

// NumPanels returns the number of facet panels.
func (b *Built) NumPanels() int { return len(b.panels) }

// NumLayers returns the number of resolved layers in the given panel.
func (b *Built) NumLayers(panel int) int {
	if panel < 0 || panel >= len(b.panels) {
		return 0
	}

	return len(b.panels[panel].Layers)
}

// Theme returns the resolved theme.
func (b *Built) Theme() theme.Theme { return b.theme }

// Labels returns the resolved plot labels.
func (b *Built) Labels() Labels { return b.labels }

// PanelLayout returns the layout geometry.
func (b *Built) PanelLayout() Layout { return b.layout }

// PreferredSize implements [output.Sizer]: given a width, it proposes a height
// using the same auto-height rules as [Plot.Save]. The width is returned
// unchanged.
func (b *Built) PreferredSize(width int) (int, int) { return width, b.autoHeight(width) }

// Compile-time guarantees that the output layer's interfaces are satisfied:
// a built plot is a drawable Figure (and a Sizer), and a Plot is a Source.
var (
	_ output.Figure = (*Built)(nil)
	_ output.Sizer  = (*Built)(nil)
	_ output.Source = (*Plot)(nil)
)

// Build resolves the plot specification through the grammar pipeline and
// returns an [output.Figure] (concretely a [*Built]) containing fully resolved
// layer data, trained scales, layout geometry, and theme. Render it via
// [output.Render] or [Built.Draw]; introspect it by asserting to [*Built],
// which exposes [Built.LayerData], [Built.NumPanels], [Built.Explain], and more.
//
// This is the Go equivalent of ggplot2's ggplot_build(plot).
func (p *Plot) Build(ctx context.Context) (output.Figure, error) {
	b, err := p.build(ctx)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// build runs the grammar pipeline and returns the concrete [*Built]. Internal
// callers (Save, Encode, Image) use this to reach Built's full method set; the
// exported [Plot.Build] widens the return type to [output.Figure].
func (p *Plot) build(ctx context.Context) (*Built, error) {
	if err := ctx.Err(); err != nil {
		return nil, Errorf(PhaseBuild, -1, "context", err, "context cancelled")
	}

	if len(p.spec.Layers) == 0 {
		return nil, Errorf(PhaseBuild, -1, "validate", ErrNoLayers, "plot has no layers")
	}

	// Materialise any lazy Dataset chain.
	collectedDS, collectErr := p.spec.Dataset.Collect(ctx)
	if collectErr != nil {
		return nil, Errorf(PhaseBuild, -1, "collect", collectErr, "collect dataset")
	}

	p.spec.Dataset = collectedDS

	if p.spec.Dataset.Table() == nil {
		return nil, Errorf(PhaseBuild, -1, "validate", ErrRenderFailed, "plot has no dataset")
	}

	th, err := theme.Resolve(p.spec.ThemeName)
	if err != nil {
		return nil, Errorf(PhaseBuild, -1, "theme", err, "resolve theme")
	}

	// Apply per-plot theme overrides.
	if len(p.spec.ThemeOverrides) > 0 {
		th = theme.WithOverrides(th, p.spec.ThemeOverrides...)
	}

	// Apply per-plot margin override.
	if p.spec.PlotMargin != nil {
		th.PlotMargin = p.spec.PlotMargin
	}

	// Apply per-plot alignment overrides.
	applyBlockAlignment(&th, p.spec.Alignment)

	// 1. Facet.
	facetPanels, err := p.spec.Facet.Split(ctx, p.spec.Dataset)
	if err != nil {
		return nil, Errorf(PhaseBuild, -1, "facet", err, "facet split")
	}

	rows, cols := p.spec.Facet.GridDims(len(facetPanels))
	if rows <= 0 {
		rows = 1
	}

	if cols <= 0 {
		cols = 1
	}

	// 2. Build row/col index maps for grid placement.
	// For Grid facets, panels carry RowVal/ColVal that determines grid position.
	// Margins use "All" and appear in extra row/column at the end.
	rowIndex := make(map[string]int) // RowVal → grid row
	colIndex := make(map[string]int) // ColVal → grid col

	for _, fp := range facetPanels {
		if fp.RowVal != "" {
			if _, ok := rowIndex[fp.RowVal]; !ok && fp.RowVal != facet.MarginLabel {
				rowIndex[fp.RowVal] = len(rowIndex)
			}
		}

		if fp.ColVal != "" {
			if _, ok := colIndex[fp.ColVal]; !ok && fp.ColVal != facet.MarginLabel {
				colIndex[fp.ColVal] = len(colIndex)
			}
		}
	}

	// Place "All" margin at the end if present.
	for _, fp := range facetPanels {
		if fp.RowVal == facet.MarginLabel {
			if _, ok := rowIndex[facet.MarginLabel]; !ok {
				rowIndex[facet.MarginLabel] = len(rowIndex)
			}
		}

		if fp.ColVal == facet.MarginLabel {
			if _, ok := colIndex[facet.MarginLabel]; !ok {
				colIndex[facet.MarginLabel] = len(colIndex)
			}
		}
	}

	// 3. Inject PANEL system column into each panel's dataset.
	eng := dataset.GetEngine(p.spec.Dataset.Table())

	for pi := range facetPanels {
		panelCol := dataset.ConstInt64Column(eng, ColPANEL, int64(pi), facetPanels[pi].NumRows)
		if panelCol != nil {
			facetPanels[pi].Dataset = facetPanels[pi].Dataset.WithColumn(panelCol)
		}

		// Single Collect materializes the lazy filter + PANEL column.
		collected, cerr := facetPanels[pi].Dataset.Collect(ctx)
		if cerr != nil {
			return nil, Errorf(PhaseBuild, -1, "facet", cerr, "collect panel dataset")
		}

		facetPanels[pi].Dataset = collected
	}

	// 3. Build each facet panel (parallel when >1 panel).
	builtPanels := make([]BuiltPanel, len(facetPanels))
	panelLayouts := make([]PanelLayout, len(facetPanels))

	if len(facetPanels) == 1 {
		// Single-panel fast path — no errgroup overhead.
		bp, err := p.buildPanel(ctx, 0, facetPanels[0].Dataset, facetPanels[0].Label, th)
		if err != nil {
			return nil, err
		}

		builtPanels[0] = bp
		panelLayouts[0] = PanelLayout{
			Row:    panelGridRow(facetPanels[0], rowIndex, 0, cols),
			Col:    panelGridCol(facetPanels[0], colIndex, 0, cols),
			RowVal: facetPanels[0].RowVal,
			ColVal: facetPanels[0].ColVal,
			XScale: bp.XScale,
			YScale: bp.YScale,
		}
	} else {
		g, gctx := errgroup.WithContext(ctx)

		for pi, panel := range facetPanels {
			g.Go(func() error {
				bp, err := p.buildPanel(gctx, pi, panel.Dataset, panel.Label, th)
				if err != nil {
					return err
				}

				builtPanels[pi] = bp
				panelLayouts[pi] = PanelLayout{
					Row:    panelGridRow(panel, rowIndex, pi, cols),
					Col:    panelGridCol(panel, colIndex, pi, cols),
					RowVal: panel.RowVal,
					ColVal: panel.ColVal,
					XScale: bp.XScale,
					YScale: bp.YScale,
				}

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, Errorf(PhaseBuild, -1, "panel", err, "parallel panel build")
		}
	}

	// Unify scales when not free — shared bounds across all panels.
	unifyPanelScales(p.spec.Facet, builtPanels, panelLayouts)

	freeX, freeY := p.spec.Facet.FreeScales()

	return &Built{
		panels: builtPanels,
		layout: Layout{
			Rows:   rows,
			Cols:   cols,
			Panels: panelLayouts,
			FreeX:  freeX,
			FreeY:  freeY,
		},
		coord:       p.spec.Coord,
		theme:       th,
		labels:      p.spec.Labels,
		legendPos:   p.spec.LegendPosition,
		ndodgeX:     p.spec.AxisGuideX.NDodge,
		annotations: p.spec.Annotations,
		secAxis:     p.spec.SecondAxis,
	}, nil
}

// buildPanel resolves layers and trains scales for a single facet panel.
func (p *Plot) buildPanel(ctx context.Context, pi int, panelDS dataset.Dataset, label string, th theme.Theme) (BuiltPanel, error) { //nolint:gocognit,cyclop // panel building is a complex pipeline — further splitting reduces clarity.
	eng := dataset.GetEngine(panelDS.Table())
	resolved := make([]BuiltLayer, 0, len(p.spec.Layers)*4)

	// layerSpans tracks the index range of each geom layer within resolved.
	// Used to apply position adjustment per layer after categorical mapping.
	var layerSpans []layerSpan

	var legendEntries []LegendEntry

	legendTitle := ""

	var colorBarSpec *ColorBarSpec

	for li, layer := range p.spec.Layers {
		layerStart := len(resolved) // track where this layer's BuiltLayers start
		merged := layer.Mapping.Merge(p.spec.GlobalMapping)
		pipeline := layer.Geom.Pipeline

		ds := panelDS

		// Check for colour/group aesthetic mapping.
		colorCol := merged["color"]
		if colorCol == "" {
			colorCol = merged["colour"]
		}

		// Detect whether color column is numeric (continuous) or categorical.
		continuousColorCol := ""

		if colorCol != "" {
			if col, err := ds.Column(colorCol); err == nil {
				if col.DType() == dataset.DTypeFloat64 || col.DType() == dataset.DTypeInt64 {
					continuousColorCol = colorCol
					colorCol = "" // prevent grouping
				}
			}
		}

		linetypeCol := merged["linetype"]
		groupCol := merged["group"]

		if groupCol == "" {
			if colorCol != "" {
				groupCol = colorCol
			} else {
				groupCol = linetypeCol
			}
		}

		if groupCol != "" {
			colorScale := p.spec.ColorScales["color"]
			if colorScale == nil {
				colorScale = colormap.NewDiscrete(theme.DefaultCmapFor(theme.Name(th.Name), theme.AesColor, colormap.Qualitative))
			}

			if col, err := ds.Column(groupCol); err == nil {
				_ = colorScale.Train(col)
			}

			groups, subsets, err := groupByColumn(ctx, ds, groupCol)
			if err != nil {
				return BuiltPanel{}, Errorf(PhaseBuild, li, "group", err, "group split by %q", groupCol)
			}

			if legendTitle == "" {
				if colorCol != "" {
					legendTitle = colorCol
				} else if linetypeCol != "" {
					legendTitle = linetypeCol
				}
			}

			for gi, grpLabel := range groups {
				_ = gi
				grpDS := subsets[gi]
				grpRGBA := colorScale.At(grpLabel)

				grpMerged := make(AesMap, len(merged))
				maps.Copy(grpMerged, merged)

				if len(pipeline) > 0 {
					pipelineData, pipelineMapping, err := stat.RunPipeline(ctx, pipeline, grpDS, grpMerged)
					if err != nil {
						return BuiltPanel{}, Errorf(PhaseBuild, li, "transform", err, "pipeline failed for group %q",
							grpLabel)
					}

					if pipelineData.Table() == nil {
						// Lazy pipeline — materialize at Draw time.
						pipelineData, err = pipelineData.Collect(ctx)
						if err != nil {
							return BuiltPanel{}, Errorf(PhaseBuild, li, "transform", err, "pipeline collect failed for group %q",
								grpLabel)
						}
					}

					grpDS = pipelineData
					grpMerged = pipelineMapping
				}

				// Bake group color into geom params only if color was mapped.
				grpGeom := layer.Geom

				if colorCol != "" {
					hex := fmt.Sprintf("#%02X%02X%02X",
						uint8(grpRGBA.R*255+0.5),
						uint8(grpRGBA.G*255+0.5),
						uint8(grpRGBA.B*255+0.5))
					grpGeom.Params.Color = hex

					if grpGeom.Params.Fill == "" {
						grpGeom.Params.Fill = hex
					}
				}

				// Inject group system column.
				grpCol := dataset.ConstInt64Column(eng, ColGroup, int64(gi), int(grpDS.NumRows()))
				if grpCol != nil {
					augmented, cerr := grpDS.WithColumn(grpCol).Collect(ctx)
					if cerr != nil {
						return BuiltPanel{}, Errorf(PhaseBuild, li, "group", cerr, "inject group column")
					}

					grpDS = augmented
				}

				resolved = append(resolved, BuiltLayer{
					Geom:    grpGeom,
					Data:    grpDS,
					Mapping: grpMerged,
				})

				if (colorCol != "" || linetypeCol != "") && pi == 0 {
					alreadyHas := false

					for _, le := range legendEntries {
						if le.Label == grpLabel {
							alreadyHas = true
							break
						}
					}

					if !alreadyHas {
						entryColor := grpRGBA

						if colorCol == "" {
							// Use default geom color or black
							dcol := layer.Geom.Params.Color
							if dcol == "" {
								dcol = "#000000"
							}

							if parsed, err := colormap.Parse(dcol); err == nil {
								entryColor = parsed
							}
						}

						legendEntries = append(legendEntries, LegendEntry{
							Label: grpLabel,
							Color: entryColor,
							Glyph: glyphForGeom(layer.Geom.Geom),
						})
					}
				}
			}

			layerSpans = append(layerSpans, layerSpan{
				start:   layerStart,
				end:     len(resolved),
				pos:     layer.Geom.Position,
				mapping: merged,
			})
		} else {
			if len(pipeline) > 0 {
				pipelineData, pipelineMapping, err := stat.RunPipeline(ctx, pipeline, ds, merged)
				if err != nil {
					return BuiltPanel{}, Errorf(PhaseBuild, li, "transform", err, "pipeline failed")
				}

				if pipelineData.Table() == nil {
					// Lazy pipeline — materialize at Draw time.
					pipelineData, err = pipelineData.Collect(ctx)
					if err != nil {
						return BuiltPanel{}, Errorf(PhaseBuild, li, "transform", err, "pipeline collect failed")
					}
				}

				ds = pipelineData
				merged = pipelineMapping
			}

			var contScale *colormap.Scale
			if continuousColorCol != "" {
				contScale = p.spec.ColorScales["color"]
				if contScale == nil {
					contScale = colormap.NewContinuous(theme.DefaultCmapFor(theme.Name(th.Name), theme.AesColor, colormap.Sequential), nil)
				}

				if col, err := ds.Column(continuousColorCol); err == nil {
					_ = contScale.Train(col)
				}
			}

			resolved = append(resolved, BuiltLayer{
				Geom:         layer.Geom,
				Data:         ds,
				Mapping:      merged,
				ContColorCol: continuousColorCol,
				ContColScale: contScale,
			})

			if contScale != nil && colorBarSpec == nil && pi == 0 {
				colorBarSpec = &ColorBarSpec{
					Title:    continuousColorCol,
					Cmap:     contScale.Cmap(),
					Norm:     contScale.Norm(),
					BarWidth: p.spec.ColorBarWidth,
					NBin:     p.spec.ColorBarNBin,
				}
			}

			if layer.Geom.Params.Label != "" && layer.Geom.Params.Color != "" && pi == 0 {
				if c, err := colormap.Parse(layer.Geom.Params.Color); err == nil {
					legendEntries = append(legendEntries, LegendEntry{
						Label: layer.Geom.Params.Label,
						Color: c,
						Glyph: glyphForGeom(layer.Geom.Geom),
					})
				}
			}

			layerSpans = append(layerSpans, layerSpan{
				start:   layerStart,
				end:     len(resolved),
				pos:     layer.Geom.Position,
				mapping: merged,
			})
		}
	}

	// Apply coord post-stat transforms at the engine level (if coord
	// implements Transformer). This transforms x/y data columns via the
	// forward function before scale training — statistics have already
	// run on raw data, but scales will now train on transformed values.
	if tr, ok := p.spec.Coord.(coord.Transformer); ok {
		for i := range resolved {
			rl := &resolved[i]
			rl.Data = applyCoordTransform(ctx, rl.Data, rl.Mapping, tr)
		}
	}

	// Train scales and apply position adjustments (after categorical mapping).
	xScale, yScale, xIsDiscrete, err := p.trainPanelScales(ctx, resolved, layerSpans)
	if err != nil {
		return BuiltPanel{}, err
	}

	// Apply hint-aware axis formatters from pipeline transforms.
	// Only applies if the user hasn't explicitly set a formatter via scale overrides.
	xScale, yScale = applyHintFormatters(p, resolved, xScale, yScale, xIsDiscrete)

	// Apply coord.Trans tick formatters so axis labels show original
	// (inverse-transformed) values. The data is in transformed space but
	// tick labels should display original data units.
	if tr, ok := p.spec.Coord.(coord.Transformer); ok {
		if xFmt := coordTickFormatter(tr.XTrans()); xFmt != nil {
			xScale = scale.Configure(xScale, scale.WithFormatter(xFmt))
		}

		if yFmt := coordTickFormatter(tr.YTrans()); yFmt != nil {
			yScale = scale.Configure(yScale, scale.WithFormatter(yFmt))
		}
	}

	// Initialize non-position scales.
	sizeScale := p.spec.SizeScale
	if sizeScale == nil {
		sizeScale = scale.NewSizeDefault()
	}

	alphaScale := p.spec.AlphaScale
	if alphaScale == nil {
		alphaScale = scale.NewAlphaDefault()
	}

	shapeScale := p.spec.ShapeScale
	if shapeScale == nil {
		shapeScale = scale.NewShape()
	}

	linetypeScale := p.spec.LinetypeScale
	if linetypeScale == nil {
		linetypeScale = scale.NewLinetype()
	}

	// Train non-position scales and associate them with resolved layers.
	for i := range resolved {
		rl := &resolved[i]
		if sizeCol, ok := rl.Mapping["size"]; ok && sizeCol != "" {
			if col, err := rl.Data.Column(sizeCol); err == nil {
				_ = sizeScale.Train(col)
			}

			rl.SizeCol = sizeCol
			rl.SizeScale = sizeScale
		}

		if alphaCol, ok := rl.Mapping["alpha"]; ok && alphaCol != "" {
			if col, err := rl.Data.Column(alphaCol); err == nil {
				_ = alphaScale.Train(col)
			}

			rl.AlphaCol = alphaCol
			rl.AlphaScale = alphaScale
		}

		if shapeCol, ok := rl.Mapping["shape"]; ok && shapeCol != "" {
			if col, err := rl.Data.Column(shapeCol); err == nil {
				_ = shapeScale.Train(col)
			}

			rl.ShapeCol = shapeCol
			rl.ShapeScale = shapeScale
		}

		if linetypeCol, ok := rl.Mapping["linetype"]; ok && linetypeCol != "" {
			if col, err := rl.Data.Column(linetypeCol); err == nil {
				_ = linetypeScale.Train(col)
			}

			rl.LinetypeCol = linetypeCol
			rl.LinetypeScale = linetypeScale
		}
	}

	// Generate categorical legend entries for mapped shapes if not already present.
	for _, rl := range resolved {
		if rl.ShapeCol != "" && rl.ShapeScale != nil && pi == 0 {
			if legendTitle == "" {
				legendTitle = rl.ShapeCol
			}

			if shScale, ok := rl.ShapeScale.(*scale.ShapeScale); ok {
				for _, cat := range shScale.Categories() {
					alreadyHas := slices.ContainsFunc(legendEntries, func(le LegendEntry) bool {
						return le.Label == cat
					})

					if !alreadyHas {
						dcol := rl.Geom.Params.Color
						if dcol == "" {
							dcol = "#000000"
						}

						var entryColor gg.RGBA
						if parsed, err := colormap.Parse(dcol); err == nil {
							entryColor = parsed
						}

						legendEntries = append(legendEntries, LegendEntry{
							Label: cat,
							Color: entryColor,
							Glyph: GlyphPoint,
							Shape: shScale.ShapeName(cat),
						})
					}
				}
			}
		}
	}

	// Post-process legend entries to assign shape and linetype patterns.
	for i, le := range legendEntries {
		for _, rl := range resolved {
			if rl.ShapeCol != "" && rl.ShapeScale != nil {
				if shScale, ok := rl.ShapeScale.(*scale.ShapeScale); ok {
					if slices.Contains(shScale.Categories(), le.Label) {
						legendEntries[i].Shape = shScale.ShapeName(le.Label)
					}
				}
			}

			if rl.LinetypeCol != "" && rl.LinetypeScale != nil {
				if ltScale, ok := rl.LinetypeScale.(*scale.LinetypeScale); ok {
					if slices.Contains(ltScale.Categories(), le.Label) {
						legendEntries[i].Linetype = ltScale.DashPattern(le.Label)
					}
				}
			}
		}
	}
	// Build continuous size legend if any layer has a continuous size mapping.
	var sizeLegend *SizeLegendSpec

	for _, rl := range resolved {
		if rl.SizeCol != "" && rl.SizeScale != nil {
			if ss, ok := rl.SizeScale.(*scale.SizeScale); ok {
				ticks := ss.Ticks(5) //nolint:mnd // 5 representative sizes.
				entries := make([]SizeLegendEntry, len(ticks))

				for i, v := range ticks {
					entries[i] = SizeLegendEntry{
						Label:  ss.Format(v),
						Radius: ss.MapValue(v),
					}
				}

				title := rl.SizeCol
				if rl.Mapping["size"] != "" {
					title = rl.Mapping["size"]
				}

				if p.spec.Labels.Size != "" {
					title = p.spec.Labels.Size
				}

				sizeLegend = &SizeLegendSpec{
					Title:   title,
					Entries: entries,
				}

				break
			}
		}
	}

	// Build continuous alpha legend if any layer has a continuous alpha mapping.
	var alphaLegend *AlphaLegendSpec

	for _, rl := range resolved {
		if rl.AlphaCol != "" && rl.AlphaScale != nil {
			if as, ok := rl.AlphaScale.(*scale.AlphaScale); ok {
				minA, maxA := as.Range()
				minV, maxV := as.Bounds()

				title := rl.AlphaCol
				if rl.Mapping["alpha"] != "" {
					title = rl.Mapping["alpha"]
				}

				if p.spec.Labels.Alpha != "" {
					title = p.spec.Labels.Alpha
				}

				// Use the layer's geom base color, fall back to
				// first legend entry, then default steelblue.
				baseColor := gg.RGBA{R: 0.27, G: 0.51, B: 0.71, A: 1} //nolint:mnd // Default steelblue base color.

				if rl.Geom.Params.Color != "" {
					cr, cg, cb := colormap.ParseRGB(rl.Geom.Params.Color, baseColor.R, baseColor.G, baseColor.B)
					baseColor = gg.RGBA{R: cr, G: cg, B: cb, A: 1}
				} else if len(legendEntries) > 0 {
					baseColor = legendEntries[0].Color
				}

				alphaLegend = &AlphaLegendSpec{
					Title:     title,
					MinAlpha:  minA,
					MaxAlpha:  maxA,
					MinValue:  minV,
					MaxValue:  maxV,
					BaseColor: baseColor,
				}

				break
			}
		}
	}

	// Apply Labels() legend title overrides for categorical/continuous color legend.
	if p.spec.Labels.Color != "" {
		legendTitle = p.spec.Labels.Color
	}

	return BuiltPanel{
		Label:         label,
		Layers:        resolved,
		XScale:        xScale,
		YScale:        yScale,
		LegendEntries: legendEntries,
		LegendTitle:   legendTitle,
		ColorBarSpec:  colorBarSpec,
		SizeLegend:    sizeLegend,
		AlphaLegend:   alphaLegend,
		XIsDiscrete:   xIsDiscrete,
	}, nil
}

// layerSpan records the index range of a single geom layer within the
// resolved []BuiltLayer slice, along with its position adjustment and mapping.
// Used by trainPanelScales to apply position adjustments after categorical mapping.
type layerSpan struct {
	start, end int
	pos        geom.Pos
	mapping    AesMap
}

// panelRenderInfo captures per-panel layout data for parallel data-layer
// rendering in [Built.Draw]. Chrome (grid, axes, legend) is drawn
// sequentially; only the data layers are rendered in parallel.
type panelRenderInfo struct {
	panelIdx       int
	bp             BuiltPanel
	xScale, yScale scale.Scale
	dataX, dataY   float64
	cellW, cellH   float64
}

// applyPositionAdjust applies the layer's position adjustment across groups.
// layers is the slice of BuiltLayers for a single geom layer (one per group).
// The function creates a fresh Pos instance to avoid shared state, reads X/Y
// from each layer's data, calls Adjust, and writes the adjusted values back.
func applyPositionAdjust(ctx context.Context, layers []BuiltLayer, layerPos geom.Pos, mapping AesMap) error {
	if len(layers) == 0 || layerPos == nil {
		return nil
	}

	posName := geom.PosName(layerPos.String())
	if posName == geom.PosIdentity || posName == "" {
		return nil // identity is a no-op
	}

	xCol := mapping["x"]
	yCol := mapping["y"]

	if xCol == "" || yCol == "" {
		return nil
	}

	nGroups := len(layers)

	// Read all groups' X/Y values.
	allXs := make([][]float64, nGroups)
	allYs := make([][]float64, nGroups)

	canAdjust := true

	for gi := range layers {
		xs, errX := layers[gi].Data.Float64(xCol)
		ys, errY := layers[gi].Data.Float64(yCol)

		if errX != nil || errY != nil {
			canAdjust = false

			break
		}

		allXs[gi] = xs
		allYs[gi] = ys
	}

	if !canAdjust {
		return nil // non-numeric data — position adjustment not applicable
	}

	// Compute bin width from minimum X spacing across all groups.
	binWidth := computeBinWidth(allXs)

	// Create a fresh position instance for stateful positions (Stack, Fill)
	// to avoid shared state across panels. For stateless positions (Dodge,
	// Jitter, Nudge), reuse the layer's instance so that user-configured
	// parameters (jitter width/height/seed, nudge offsets) are preserved.
	var pos geom.Pos

	switch posName {
	case geom.PosStack, geom.PosFill:
		pos = geom.NewPos(posName) // fresh instance avoids cross-panel contamination
	case geom.PosIdentity:
		return nil // identity is a no-op (already handled above, but satisfies exhaustive)
	case geom.PosDodge, geom.PosJitter, geom.PosNudge:
		pos = layerPos // reuse configured instance (stateless: Dodge, Jitter, Nudge)
	}

	// If Fill, run the setup phase.
	if fs, ok := pos.(geom.FillSetup); ok {
		fs.Setup(allXs, allYs)
	}

	// Apply position adjustment per group and write back.
	stacker, isStacker := pos.(geom.Stacker)

	for gi := range layers {
		var adjXs, adjYs []float64

		var adjYMin []float64

		if isStacker {
			adjXs, adjYMin, adjYs = stacker.AdjustStack(allXs[gi], allYs[gi], binWidth, gi, nGroups)
		} else {
			adjXs, adjYs = pos.Adjust(allXs[gi], allYs[gi], binWidth, gi, nGroups)
		}

		// Write adjusted values back via ReplaceColumn (columns already exist).
		adjusted := dataset.ReplaceColumn(layers[gi].Data, xCol, adjXs)
		adjusted = dataset.ReplaceColumn(adjusted, yCol, adjYs)

		// For stacking positions, add a ymin column so the bar drawer
		// knows where the bottom of each bar segment is.
		// Use WithColumn (not ReplaceColumn) since "ymin" is a new column.
		if adjYMin != nil {
			eng := dataset.GetEngine(layers[gi].Data.Table())
			if factory, ok := eng.(dataset.ColumnFactory); ok {
				yminCol := factory.NewFloat64Column("ymin", adjYMin)
				adjusted = adjusted.WithColumn(yminCol)
			}
		}

		collected, err := adjusted.Collect(ctx)
		if err != nil {
			return Errorf(PhaseBuild, -1, "position", err, "position adjust group %d", gi)
		}

		layers[gi].Data = collected

		// For dodge, narrow the bar width so groups fit side-by-side.
		if posName == geom.PosDodge && nGroups > 1 {
			w := layers[gi].Geom.Params.Width
			if w <= 0 || w > 1 {
				w = 0.8 // default bar relative width
			}

			layers[gi].Geom.Params.Width = w / float64(nGroups)
		}
	}

	return nil
}

// computeBinWidth returns the minimum spacing between adjacent X values
// across all groups. Used as the width parameter for dodge/stack.
func computeBinWidth(allXs [][]float64) float64 {
	// Collect all unique X values.
	seen := make(map[float64]struct{})

	for _, xs := range allXs {
		for _, x := range xs {
			seen[x] = struct{}{}
		}
	}

	if len(seen) <= 1 {
		return 1.0
	}

	// Sort unique values.
	sorted := make([]float64, 0, len(seen))
	for x := range seen {
		sorted = append(sorted, x)
	}

	sort.Float64s(sorted)

	minSpacing := sorted[1] - sorted[0]

	for i := 2; i < len(sorted); i++ {
		sp := sorted[i] - sorted[i-1]
		if sp > 0 && sp < minSpacing {
			minSpacing = sp
		}
	}

	if minSpacing <= 0 {
		return 1.0
	}

	return minSpacing
}

// trainPanelScales detects, trains, and adjusts scales for a panel's resolved layers.
// After categorical X mapping, position adjustments are applied per layer span
// so that dodge/stack/fill see numeric X values.
func (p *Plot) trainPanelScales(ctx context.Context, resolved []BuiltLayer, spans []layerSpan) (xScale, yScale scale.Scale, xIsDiscrete bool, err error) { //nolint:gocognit,cyclop // scale training is inherently complex — splitting further reduces clarity.
	yScale, err = scale.Resolve(p.spec.ScaleOverrides["y"].Type)
	if err != nil {
		return nil, nil, false, Errorf(PhaseBuild, -1, "scale", err, "resolve y scale")
	}

	if yOpts := p.spec.ScaleOverrides["y"].Opts; len(yOpts) > 0 {
		yScale = scale.Configure(yScale, yOpts...)
	}

	xIsDiscrete = false

	// Probe the first resolved layer's X column to decide scale type.
	var xDType dataset.DType

	for _, rl := range resolved {
		if colName, ok := rl.Mapping["x"]; ok {
			if col, err := rl.Data.Column(colName); err == nil {
				xDType = col.DType()

				switch xDType { //nolint:exhaustive // only numeric and temporal types are continuous.
				case dataset.DTypeFloat64, dataset.DTypeInt64,
					dataset.DTypeTimestamp, dataset.DTypeDate, dataset.DTypeTime:
					// continuous
				default:
					xIsDiscrete = true
				}
			}

			break
		}
	}

	if xIsDiscrete {
		ds := scale.Discrete()

		for i, rl := range resolved {
			xColName, ok := rl.Mapping["x"]
			if !ok {
				continue
			}

			col, err := rl.Data.Column(xColName)
			if err != nil {
				continue
			}

			if err := ds.Train(col); err != nil {
				continue
			}

			if sc, ok2 := col.(dataset.Column[string]); ok2 {
				vals := sc.Values()

				positions := make([]float64, len(vals))
				for j, v := range vals {
					positions[j] = ds.MapCategory(v)
				}

				lazyDS := dataset.ReplaceColumn(rl.Data, xColName, positions)
				if cds, cerr := lazyDS.Collect(ctx); cerr == nil {
					resolved[i].Data = cds
				}
			}
		}

		xScale = ds
	} else {
		// Auto-detect DateTime scale for temporal X columns.
		if _, hasXOverride := p.spec.ScaleOverrides["x"]; !hasXOverride {
			switch xDType { //nolint:exhaustive // only temporal types need DateTime scale; default handles the rest.
			case dataset.DTypeTimestamp, dataset.DTypeDate, dataset.DTypeTime:
				xScale = scale.NewDateTime()
			default:
				xScale, err = scale.Resolve(p.spec.ScaleOverrides["x"].Type)
				if err != nil {
					return nil, nil, false, Errorf(PhaseBuild, -1, "scale", err, "resolve x scale")
				}
			}
		} else {
			xScale, err = scale.Resolve(p.spec.ScaleOverrides["x"].Type)
			if err != nil {
				return nil, nil, false, Errorf(PhaseBuild, -1, "scale", err, "resolve x scale")
			}
		}

		if xOpts := p.spec.ScaleOverrides["x"].Opts; len(xOpts) > 0 {
			xScale = scale.Configure(xScale, xOpts...)
		}
	}

	// Apply position adjustments now that X values are numeric.
	// This must happen after categorical mapping (above) and before Y training (below).
	for _, sp := range spans {
		if err := applyPositionAdjust(ctx, resolved[sp.start:sp.end], sp.pos, sp.mapping); err != nil {
			return nil, nil, false, err
		}
	}

	// Train scales on (now numeric) data.
	for _, rl := range resolved {
		if !xIsDiscrete {
			if colName, ok := rl.Mapping["x"]; ok {
				if col, err := rl.Data.Column(colName); err == nil {
					_ = xScale.Train(col)
				}
			}
		}

		if colName, ok := rl.Mapping["y"]; ok {
			if col, err := rl.Data.Column(colName); err == nil {
				_ = yScale.Train(col)
			}
		}

		// Train X scale on range/endpoint aesthetics so segments, tiles,
		// and error bars with horizontal orientation include their full extent.
		if !xIsDiscrete {
			for _, extra := range []string{"xmin", "xmax", "xend"} {
				if colName, ok := rl.Mapping[extra]; ok {
					if col, err := rl.Data.Column(colName); err == nil {
						_ = xScale.Train(col)
					}
				}
			}
		}

		// Train Y scale on range/endpoint aesthetics so error bars,
		// segments, and ribbons include their full extent.
		for _, extra := range []string{"ymin", "ymax", "yend"} {
			if colName, ok := rl.Mapping[extra]; ok {
				if col, err := rl.Data.Column(colName); err == nil {
					_ = yScale.Train(col)
				}
			}
		}

		if rl.Geom.Geom == geom.TypeBoxPlot {
			for _, extra := range []string{"lower", "q1", "middle", "q3", "upper"} {
				if col, err := rl.Data.Column(extra); err == nil {
					_ = yScale.Train(col)
				}
			}
		}
	}

	// Ensure Y starts at 0 for bar/histogram/area/density/boxplot.
	for _, rl := range resolved {
		switch rl.Geom.Geom { //nolint:exhaustive // intentional subset; default case handles the rest.
		case geom.TypeBar, geom.TypeHistogram, geom.TypeRect, geom.TypeArea, geom.TypeDensity,
			geom.TypeDotplot:
			yMin, yMax := yScale.Bounds()
			if yMin > 0 {
				if bs, ok := yScale.(scale.BoundsSetter); ok {
					bs.SetBounds(0, yMax)
				}
			}
		default:
		}
	}

	// Add padding.
	xMin, xMax := xScale.Bounds()
	yMin, yMax := yScale.Bounds()

	if !xIsDiscrete {
		xHasExpand := false
		if exp, ok := xScale.(scale.Expander); ok {
			xHasExpand = exp.HasExpand()
		}

		yHasExpand := false
		if exp, ok := yScale.(scale.Expander); ok {
			yHasExpand = exp.HasExpand()
		}

		xPad := 0.0
		yPad := 0.0

		if !xHasExpand {
			xPad = (xMax - xMin) * 0.05
			if xPad == 0 {
				xPad = 0.5
			}
		}

		if !yHasExpand {
			yPad = (yMax - yMin) * 0.05
			if yPad == 0 {
				yPad = 0.5
			}
		}

		if !xHasExpand {
			hasBars := false

			for _, rl := range resolved {
				switch rl.Geom.Geom { //nolint:exhaustive // intentional subset; default case handles the rest.
				case geom.TypeBar, geom.TypeHistogram, geom.TypeRect, geom.TypeBoxPlot,
					geom.TypeCrossbar, geom.TypeViolin, geom.TypeDotplot:
					hasBars = true
				default:
				}
			}

			if hasBars {
				for _, rl := range resolved {
					// Count distinct X positions for width-based padding.
					// After stat transforms (e.g. ViolinY), a layer with
					// 3 groups may have hundreds of rows. Using raw row
					// count would give near-zero padding; distinct X
					// positions give the correct group spacing.
					xCol := rl.Mapping["x"]
					if xCol == "" {
						continue
					}

					xv, xerr := rl.Data.Float64(xCol)
					if xerr != nil || len(xv) == 0 {
						continue
					}

					seen := make(map[float64]struct{}, len(xv))
					for _, v := range xv {
						seen[v] = struct{}{}
					}

					nDistinct := len(seen)
					if nDistinct > 1 {
						halfBin := (xMax - xMin) / float64(nDistinct-1) / 2.0
						if halfBin > xPad {
							xPad = halfBin
						}
					} else if nDistinct == 1 {
						xPad = 1.0
					}
				}
			}
		}

		if xPad > 0 {
			if bs, ok := xScale.(scale.BoundsSetter); ok {
				bs.SetBounds(xMin-xPad, xMax+xPad)
			}
		}

		if yPad > 0 {
			if yMin == 0 {
				if bs, ok := yScale.(scale.BoundsSetter); ok {
					bs.SetBounds(0, yMax+yPad)
				}
			} else {
				if bs, ok := yScale.(scale.BoundsSetter); ok {
					bs.SetBounds(yMin-yPad, yMax+yPad)
				}
			}
		}
	} else {
		yPad := (yMax - yMin) * 0.05
		if yPad == 0 {
			yPad = 0.5
		}

		if yMin == 0 {
			if bs, ok := yScale.(scale.BoundsSetter); ok {
				bs.SetBounds(0, yMax+yPad)
			}
		} else {
			if bs, ok := yScale.(scale.BoundsSetter); ok {
				bs.SetBounds(yMin-yPad, yMax+yPad)
			}
		}
	}

	// Apply user-specified axis limits.
	if p.spec.XLim[0] != nil || p.spec.XLim[1] != nil {
		curXMin, curXMax := xScale.Bounds()
		if p.spec.XLim[0] != nil && !math.IsNaN(*p.spec.XLim[0]) {
			curXMin = *p.spec.XLim[0]
		}

		if p.spec.XLim[1] != nil && !math.IsNaN(*p.spec.XLim[1]) {
			curXMax = *p.spec.XLim[1]
		}

		if bs, ok := xScale.(scale.BoundsSetter); ok {
			bs.SetBounds(curXMin, curXMax)
		}
	}

	if p.spec.YLim[0] != nil || p.spec.YLim[1] != nil {
		curYMin, curYMax := yScale.Bounds()
		if p.spec.YLim[0] != nil && !math.IsNaN(*p.spec.YLim[0]) {
			curYMin = *p.spec.YLim[0]
		}

		if p.spec.YLim[1] != nil && !math.IsNaN(*p.spec.YLim[1]) {
			curYMax = *p.spec.YLim[1]
		}

		if bs, ok := yScale.(scale.BoundsSetter); ok {
			bs.SetBounds(curYMin, curYMax)
		}
	}

	// Apply coord viewport zoom (if the coord implements Zoomer).
	// This runs after XLim/YLim so zoom bounds take final precedence.
	if z, ok := p.spec.Coord.(coord.Zoomer); ok {
		xlim, ylim := z.ZoomBounds()

		if xlim[0] != nil || xlim[1] != nil {
			curXMin, curXMax := xScale.Bounds()

			if xlim[0] != nil {
				curXMin = *xlim[0]
			}

			if xlim[1] != nil {
				curXMax = *xlim[1]
			}

			if bs, ok := xScale.(scale.BoundsSetter); ok {
				bs.SetBounds(curXMin, curXMax)
			}
		}

		if ylim[0] != nil || ylim[1] != nil {
			curYMin, curYMax := yScale.Bounds()

			if ylim[0] != nil {
				curYMin = *ylim[0]
			}

			if ylim[1] != nil {
				curYMax = *ylim[1]
			}

			if bs, ok := yScale.(scale.BoundsSetter); ok {
				bs.SetBounds(curYMin, curYMax)
			}
		}
	}

	return xScale, yScale, xIsDiscrete, nil
}

// collectPipelineHints merges OutputHints from all transforms in a pipeline.
// Later transforms' hints take precedence (last-writer-wins).
func collectPipelineHints(pipeline []stat.Transform) map[string]stat.ChannelHint {
	merged := make(map[string]stat.ChannelHint)

	for _, t := range pipeline {
		maps.Copy(merged, t.OutputHints())
	}

	return merged
}

// hintFormatter returns a tick-label formatting function for the given hint,
// or nil if no special formatting applies.
func hintFormatter(hint stat.ChannelHint) func(float64) string {
	switch hint {
	case stat.HintCount:
		return func(v float64) string { return fmt.Sprintf("%.0f", v) }
	case stat.HintProportion:
		return func(v float64) string { return fmt.Sprintf("%.0f%%", v*100) } //nolint:mnd // 100 converts proportion to percentage.
	case stat.HintProbability:
		return func(v float64) string { return fmt.Sprintf("%.2f", v) }
	case stat.HintCumulative:
		return func(v float64) string { return fmt.Sprintf("%.1f", v) }
	case stat.HintDeviation:
		return func(v float64) string {
			if v >= 0 {
				return fmt.Sprintf("+%.2f", v)
			}

			return fmt.Sprintf("%.2f", v)
		}
	case stat.HintNone, stat.HintInterval:
		return nil
	}

	return nil
}

// applyCoordTransform dispatches position columns to named [dataset.MathKernel]
// operations (Log10, Log2, Sqrt, Neg) post-stat, pre-scale-training.
func applyCoordTransform(ctx context.Context, ds dataset.Dataset, mapping AesMap, tr coord.Transformer) dataset.Dataset {
	eng := dataset.GetEngine(ds)
	mk, ok := eng.(dataset.MathKernel)

	if !ok {
		return ds
	}

	result := ds

	xt := tr.XTrans()
	if !xt.IsIdentity() {
		for _, key := range [...]string{"x", "xmin", "xmax", "xend"} {
			result = coordTransformColumn(result, mapping, key, xt.Name, mk)
		}
	}

	yt := tr.YTrans()
	if !yt.IsIdentity() {
		for _, key := range [...]string{"y", "ymin", "ymax", "yend"} {
			result = coordTransformColumn(result, mapping, key, yt.Name, mk)
		}
	}

	if collected, err := result.Collect(ctx); err == nil {
		return collected
	}

	return result
}

// coordTransformColumn dispatches a named transform to [dataset.MathKernel].
func coordTransformColumn(ds dataset.Dataset, mapping AesMap, aesKey, name string, mk dataset.MathKernel) dataset.Dataset {
	colName := mapping[aesKey]
	if colName == "" {
		return ds
	}

	col, err := ds.Column(colName)
	if err != nil {
		return ds
	}

	transformed, err := coordApplyKernel(mk, col, name)
	if err != nil {
		return ds
	}

	return ds.WithColumn(transformed)
}

// coordApplyKernel dispatches to native engine operations.
func coordApplyKernel(mk dataset.MathKernel, col dataset.AnyColumn, name string) (dataset.AnyColumn, error) {
	var (
		result dataset.AnyColumn
		err    error
	)

	switch name {
	case "log10":
		result, err = mk.Log10(col)
	case "log2":
		result, err = mk.Log2(col)
	case "sqrt":
		result, err = mk.Sqrt(col)
	case "reverse":
		result, err = mk.Neg(col)
	default:
		return nil, fmt.Errorf("coord transform %q: %w", name, ErrUnsupportedTransform)
	}

	if err != nil {
		return nil, fmt.Errorf("coord transform %q: %w", name, err)
	}

	return result, nil
}

// coordTickFormatter returns a scalar tick-label formatter that inverts
// the transform back to data space. Returns nil for identity.
func coordTickFormatter(t coord.TransFunc) func(float64) string {
	switch t.Name {
	case "log10":
		return func(v float64) string { return coordFormatTick(math.Pow(10, v)) } //nolint:mnd // 10^v.
	case "log2":
		return func(v float64) string { return coordFormatTick(math.Exp2(v)) }
	case "sqrt":
		return func(v float64) string { return coordFormatTick(v * v) }
	case "reverse":
		return func(v float64) string { return coordFormatTick(-v) }
	default:
		return nil
	}
}

// coordFormatTick formats a tick value, returning "" for non-finite values.
func coordFormatTick(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return ""
	}

	return scale.FormatNumber(v)
}

// applyHintFormatters checks all pipeline layers for OutputHints and wraps the
// X/Y scales with hint-based formatters. User-supplied formatter overrides are
// respected — hints only apply when no explicit formatter is set.
func applyHintFormatters(p *Plot, resolved []BuiltLayer, xScale, yScale scale.Scale, xIsDiscrete bool) (scale.Scale, scale.Scale) {
	// Collect hints from all layers (last-writer-wins per channel).
	allHints := make(map[string]stat.ChannelHint)

	for _, rl := range resolved {
		maps.Copy(allHints, collectPipelineHints(rl.Geom.Pipeline))
	}

	// Apply X hint (if present and no user override).
	if xHint, ok := allHints["x"]; ok && !xIsDiscrete {
		if fn := hintFormatter(xHint); fn != nil {
			// Only apply if user hasn't set a custom formatter via scale overrides.
			if _, hasXOverride := p.spec.ScaleOverrides["x"]; !hasXOverride {
				xScale = scale.Configure(xScale, scale.WithFormatter(fn))
			}
		}
	}

	// Apply Y hint (if present and no user override).
	if yHint, ok := allHints["y"]; ok {
		if fn := hintFormatter(yHint); fn != nil {
			if _, hasYOverride := p.spec.ScaleOverrides["y"]; !hasYOverride {
				yScale = scale.Configure(yScale, scale.WithFormatter(fn))
			}
		}
	}

	return xScale, yScale
}

// --- Introspection ---

// PipelineFor returns the ordered transform names for the given panel and layer.
// Panel and layer indices are zero-based.
func (b *Built) PipelineFor(panel, layer int) []string {
	if panel < 0 || panel >= len(b.panels) {
		return nil
	}

	bp := b.panels[panel]
	if layer < 0 || layer >= len(bp.Layers) {
		return nil
	}

	pipeline := bp.Layers[layer].Geom.Pipeline
	names := make([]string, len(pipeline))

	for i, t := range pipeline {
		names[i] = t.Name()
	}

	return names
}

// Explain returns a human-readable summary of the built plot's structure,
// including panels, layers, transform pipelines, and output hints.
func (b *Built) Explain() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Plot: %d panel(s), coord=%T\n", len(b.panels), b.coord)

	for pi, bp := range b.panels {
		fmt.Fprintf(&sb, "\nPanel %d", pi)

		if bp.Label != "" {
			fmt.Fprintf(&sb, " (%s)", bp.Label)
		}

		sb.WriteString(":\n")

		for li, rl := range bp.Layers {
			fmt.Fprintf(&sb, "  Layer %d: geom=%s", li, rl.Geom.Geom)

			if len(rl.Geom.Pipeline) > 0 {
				sb.WriteString(", pipeline=[")

				for ti, t := range rl.Geom.Pipeline {
					if ti > 0 {
						sb.WriteString(" → ")
					}

					sb.WriteString(t.Name())

					if hints := t.OutputHints(); len(hints) > 0 {
						sb.WriteString("{")

						first := true

						for ch, h := range hints {
							if !first {
								sb.WriteString(", ")
							}

							fmt.Fprintf(&sb, "%s:%s", ch, h)

							first = false
						}

						sb.WriteString("}")
					}
				}

				sb.WriteString("]")
			}

			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// ---------------------------------------------------------------------------
// Draw pipeline
// ---------------------------------------------------------------------------

// Draw renders the built plot onto the given canvas at the specified dimensions.
//
// This is the Go equivalent of ggplot2's grid.draw(ggplot_gtable(built)).
func (b *Built) Draw(ctx context.Context, cv canvas.Canvas, width, height int) error { //nolint:gocognit,cyclop // Draw is a complex rendering pipeline — splitting further reduces clarity.
	if err := ctx.Err(); err != nil {
		return Errorf(PhaseDraw, -1, "context", err, "context cancelled")
	}

	th := b.theme
	cv.Clear(th.PlotBackground().Fill)

	rows := b.layout.Rows
	cols := b.layout.Cols

	// Cached layout for multi-panel consistency.
	var (
		cachedMTop, cachedMRight, cachedMBottom, cachedMLeft, cachedLegendW float64
		legendPos                                                           string
		hasLegend                                                           bool
	)

	// panelRenderInfos collects per-panel data for parallel data-layer rendering.
	var panelRenderInfos []panelRenderInfo

	for pi, bp := range b.panels {
		xScale := bp.XScale
		yScale := bp.YScale

		// Measure Y tick labels for left margin.
		leftAxisScale := yScale
		if allLayersHorizontal(builtLayersToLayerSpecs(bp.Layers)) {
			leftAxisScale = xScale
		}

		yTicks := leftAxisScale.Ticks(6)

		cv.SetFontSize(th.AxisTextElem().Size)
		cv.SetTabularNums(true)

		maxTickW := 0.0

		for _, v := range yTicks {
			tw, _ := cv.MeasureString(leftAxisScale.Format(v))
			if tw > maxTickW {
				maxTickW = tw
			}
		}

		cv.SetTabularNums(false)

		var (
			mTop, mRight, mBottom, mLeft               float64
			panelW, panelH, cellW, cellH, dataX, dataY float64
			legendW                                    float64
		)

		if pi == 0 || len(b.panels) == 1 {
			baseT, baseR, baseB, baseL := th.ResolvedPlotMargin(th.AxisTextElem().Size * 1.2) //nolint:mnd // 1.2× line-height approximation for UnitLines.
			mTop = baseT
			mRight = baseR
			mBottom = baseB + th.AxisTextElem().Size + th.Spacing.TickLength + 14

			// When axis labels are rotated, their projected height can be
			// much larger than the raw font size. Measure the worst-case
			// rotated label extent for accurate bottom margin.
			xAngle := th.AxisTextElem().Angle
			if xAngle != 0 {
				// Measure the widest X-axis tick label.
				bottomAxisScale := xScale
				if allLayersHorizontal(builtLayersToLayerSpecs(bp.Layers)) {
					bottomAxisScale = yScale
				}

				cv.SetFontSize(th.AxisTextElem().Size)

				maxXLabelW := 0.0

				for _, v := range bottomAxisScale.Ticks(5) {
					tw, _ := cv.MeasureString(bottomAxisScale.Format(v))
					if tw > maxXLabelW {
						maxXLabelW = tw
					}
				}

				rad := xAngle * math.Pi / 180 //nolint:mnd // degrees-to-radians conversion.
				rotH := math.Abs(maxXLabelW*math.Sin(rad)) + math.Abs(th.AxisTextElem().Size*math.Cos(rad))
				mBottom = baseB + rotH + th.Spacing.TickLength + 14
			}

			mLeft = baseL + maxTickW + th.Spacing.TickLength + 10

			// When axis labels are dodged across multiple rows, the
			// bottom margin must grow by (nRows−1) × rowHeight.
			dodgeRowH := th.AxisTextElem().Size + 4 //nolint:mnd // Must match rowHeight in drawXAxis.
			if b.ndodgeX >= 2 {
				mBottom += float64(b.ndodgeX-1) * dodgeRowH
			} else if b.ndodgeX == 0 {
				// Auto-dodge: measure whether labels will overlap.
				// If so, predict 2-row stagger and reserve space.
				bottomAxisScale := xScale
				if allLayersHorizontal(builtLayersToLayerSpecs(bp.Layers)) {
					bottomAxisScale = yScale
				}

				cv.SetFontSize(th.AxisTextElem().Size)

				ticks := bottomAxisScale.Ticks(5)

				const labelGap = 4

				overlaps := false

				if len(ticks) >= 2 {
					// Estimate pixel positions assuming uniform spacing.
					plotW := float64(width) - mLeft - mRight

					for j := 1; j < len(ticks); j++ {
						prevW, _ := cv.MeasureString(bottomAxisScale.Format(ticks[j-1]))
						currW, _ := cv.MeasureString(bottomAxisScale.Format(ticks[j]))
						spacing := plotW / float64(len(ticks)-1)

						if spacing < (prevW+currW)/2+labelGap {
							overlaps = true

							break
						}
					}
				}

				if overlaps {
					mBottom += dodgeRowH // auto-dodge adds 1 extra row (2 total)
				}
			}

			if b.labels.Title != "" {
				mTop += th.PlotTitle().Size + 8
			}

			if b.labels.Subtitle != "" {
				mTop += th.PlotSubtitle().Size + 8 //nolint:mnd // Match title gap for consistent subtitle spacing.
			}

			if b.labels.X != "" {
				mBottom += th.AxisTitle().Size + 8
			}

			if b.labels.Y != "" {
				titleGap := 5 + maxTickW + th.AxisTitle().Size + 14
				if titleGap < 30+th.AxisTitle().Size/2 {
					titleGap = 30 + th.AxisTitle().Size/2
				}

				mLeft = baseL + titleGap + th.Spacing.TickLength
			}

			if b.labels.Caption != "" {
				mBottom += th.AxisTextElem().Size + 4
			}

			// Secondary Y-axis right margin.
			if b.secAxis != nil {
				// Measure secondary ticks to compute right margin.
				secScale := &scale.DerivedScale{Primary: leftAxisScale, Spec: *b.secAxis}
				secTicks := secScale.Ticks(5) //nolint:mnd // Match Y-axis tick count.
				maxSecTickW := 0.0

				cv.SetFontSize(th.AxisTextElem().Size)
				cv.SetTabularNums(true)

				for _, sv := range secTicks {
					tw, _ := cv.MeasureString(secScale.Format(sv))
					if tw > maxSecTickW {
						maxSecTickW = tw
					}
				}

				cv.SetTabularNums(false)

				secAxisW := th.Spacing.TickLength + 5 + maxSecTickW //nolint:mnd // tick + gap + labels.
				if b.secAxis.Name != "" {
					secAxisW += th.AxisTitle().Size + 8 //nolint:mnd // Title padding.
				}

				mRight += secAxisW
			}

			legendPos = b.legendPos
			if legendPos == "" {
				legendPos = "right"
			}

			hasLegend = legendPos != "none" && (len(bp.LegendEntries) > 0 || bp.ColorBarSpec != nil || bp.SizeLegend != nil || bp.AlphaLegend != nil)

			legendW = 0.0

			if hasLegend && (legendPos == "right" || legendPos == "left") {
				// Resolve title font size once — used to measure all legend titles.
				titleSize := th.LegendTitle().Size
				if titleSize == 0 {
					titleSize = th.LegendTextElem().Size
				}

				ls := th.Spacing.Legend.Resolved()

				if len(bp.LegendEntries) > 0 {
					cv.SetFontSize(th.LegendTextElem().Size)

					maxLabelW := 0.0

					for _, le := range bp.LegendEntries {
						tw, _ := cv.MeasureString(le.Label)
						if tw > maxLabelW {
							maxLabelW = tw
						}
					}

					legendW += maxLabelW + ls.SwatchSize + ls.SwatchGap + ls.Padding*2 + ls.SwatchGap

					// Ensure width fits the categorical legend title.
					if bp.LegendTitle != "" {
						cv.SetFontSize(titleSize)
						ttw, _ := cv.MeasureString(bp.LegendTitle)

						titleNeed := ttw + ls.Padding*2 + ls.SwatchGap*2
						if titleNeed > legendW {
							legendW = titleNeed
						}
					}
				}

				if bp.ColorBarSpec != nil {
					cv.SetFontSize(th.LegendTextElem().Size * 0.9)

					vmin, vmax := 0.0, 1.0
					if bp.ColorBarSpec.Norm != nil {
						vmin, vmax = bp.ColorBarSpec.Norm.Bounds()
					}

					maxLblW, _ := cv.MeasureString(fmt.Sprintf("%.4g", vmax))
					minLblW, _ := cv.MeasureString(fmt.Sprintf("%.4g", vmin))
					lblW := math.Max(maxLblW, minLblW)
					legendW += ls.BarWidth + lblW + ls.Padding*2
				}

				if bp.SizeLegend != nil {
					// Size legend width: max diameter + gap + label width.
					cv.SetFontSize(th.LegendTextElem().Size)

					maxR := 0.0
					maxSzLblW := 0.0

					for _, e := range bp.SizeLegend.Entries {
						if e.Radius > maxR {
							maxR = e.Radius
						}

						tw, _ := cv.MeasureString(e.Label)
						if tw > maxSzLblW {
							maxSzLblW = tw
						}
					}

					sizeLegW := maxR*2 + ls.SwatchGap + maxSzLblW + ls.Padding*2
					if sizeLegW > legendW {
						legendW = sizeLegW
					}

					// Ensure width fits the size legend title.
					if bp.SizeLegend.Title != "" {
						cv.SetFontSize(titleSize)
						ttw, _ := cv.MeasureString(bp.SizeLegend.Title)

						titleNeed := ttw + ls.Padding*2 + ls.SwatchGap*2
						if titleNeed > legendW {
							legendW = titleNeed
						}
					}
				}

				if bp.AlphaLegend != nil {
					// Alpha legend width: bar + gap + label.
					cv.SetFontSize(th.LegendTextElem().Size * 0.9) //nolint:mnd // Match alpha label size.

					alphaLblW, _ := cv.MeasureString("1.00")

					alphaLegW := ls.BarWidth + 4 + alphaLblW + ls.Padding*2 //nolint:mnd // bar + gap + label + padding.
					if alphaLegW > legendW {
						legendW = alphaLegW
					}

					// Ensure width fits the alpha legend title.
					if bp.AlphaLegend.Title != "" {
						cv.SetFontSize(titleSize)
						ttw, _ := cv.MeasureString(bp.AlphaLegend.Title)

						titleNeed := ttw + ls.Padding*2 + ls.SwatchGap*2
						if titleNeed > legendW {
							legendW = titleNeed
						}
					}
				}
			}

			switch legendPos {
			case "right":
				if hasLegend {
					mRight += legendW
				}
			case "left":
				if hasLegend {
					mLeft += legendW
				}
			case "top":
				if hasLegend {
					mTop += 30
				}
			case "bottom":
				if hasLegend {
					mBottom += 30
				}
			}

			cachedMTop = mTop
			cachedMRight = mRight
			cachedMBottom = mBottom
			cachedMLeft = mLeft
			cachedLegendW = legendW
		} else {
			mTop = cachedMTop
			mRight = cachedMRight
			mBottom = cachedMBottom
			mLeft = cachedMLeft
			legendW = cachedLegendW
		}

		panelW = float64(width) - mLeft - mRight
		panelH = float64(height) - mTop - mBottom

		if panelW < 10 {
			panelW = 10
		}

		if panelH < 10 {
			panelH = 10
		}

		cellW = panelW / float64(cols)
		cellH = panelH / float64(rows)

		pl := b.layout.Panels[pi]

		dataX = mLeft + float64(pl.Col)*cellW
		dataY = mTop + float64(pl.Row)*cellH

		// Facet strip label above each panel.
		stripH := 0.0
		stripLabel := bp.Label

		// For Grid facets with RowVal/ColVal, show column header on first
		// row and row header on first column for cleaner grid layouts.

		if pl.RowVal != "" && pl.ColVal != "" {
			if pl.Row == 0 {
				stripLabel = pl.ColVal
			} else {
				stripLabel = ""
			}
		}

		if stripLabel != "" {
			stripH = th.AxisTextElem().Size + 8 //nolint:mnd // Strip padding.

			if border := th.PanelBorder(); border.Color != nil {
				sr, sg, sb, _ := border.Color.RGBA()
				cv.SetRGBA(float64(sr)/65535.0, float64(sg)/65535.0, float64(sb)/65535.0, 0.25) //nolint:mnd // Quarter-opacity strip background.
			} else {
				cv.SetRGBA(0.6, 0.6, 0.6, 0.15) //nolint:mnd // Fallback light-grey strip background.
			}

			cv.DrawRectangle(dataX, dataY, cellW, stripH)
			cv.Fill()
			cv.SetColor(th.AxisTitle().Color)
			cv.SetFontSize(th.AxisTextElem().Size)
			cv.DrawStringAnchored(stripLabel, dataX+cellW/2, dataY+stripH/2, 0.5, 0.5) //nolint:mnd // Centered anchor.
		}

		dataY += stripH
		cellH -= stripH

		// Right-side row strip label for Grid facets (last column only).
		rowStripW := 0.0

		if pl.RowVal != "" && pl.ColVal != "" {
			if pl.Col == cols-1 {
				rowStripW = th.AxisTextElem().Size + 8 //nolint:mnd // Strip padding.

				if border := th.PanelBorder(); border.Color != nil {
					sr, sg, sb, _ := border.Color.RGBA()
					cv.SetRGBA(float64(sr)/65535.0, float64(sg)/65535.0, float64(sb)/65535.0, 0.25) //nolint:mnd // Quarter-opacity strip background.
				} else {
					cv.SetRGBA(0.6, 0.6, 0.6, 0.15) //nolint:mnd // Fallback light-grey strip background.
				}

				cv.DrawRectangle(dataX+cellW-rowStripW, dataY, rowStripW, cellH)
				cv.Fill()
				cv.SetColor(th.AxisTitle().Color)
				cv.SetFontSize(th.AxisTextElem().Size)

				// Draw rotated row label using Save/Translate/Rotate/Restore.
				cx := dataX + cellW - rowStripW/2 //nolint:mnd // Centre X of strip.
				cy := dataY + cellH/2             //nolint:mnd // Centre Y of strip.

				cv.Save()
				cv.Translate(cx, cy)
				cv.Rotate(-math.Pi / 2)                          //nolint:mnd // 90° counter-clockwise.
				cv.DrawStringAnchored(pl.RowVal, 0, 0, 0.5, 0.5) //nolint:mnd // Centred anchor.
				cv.Restore()
			}
		}

		cellW -= rowStripW

		// Apply fixed aspect ratio if coord implements Fixer.
		if f, ok := b.coord.(coord.Fixer); ok {
			ratio := f.AspectRatio()
			xMin, xMax := xScale.Bounds()
			yMin, yMax := yScale.Bounds()
			dataRangeX := xMax - xMin
			dataRangeY := yMax - yMin

			if dataRangeX > 0 && dataRangeY > 0 && ratio > 0 {
				// Current pixels-per-unit for each axis.
				ppuX := cellW / dataRangeX
				ppuY := cellH / dataRangeY

				// Desired: ppuY / ppuX = ratio → ppuY = ratio * ppuX.
				// Adjust the axis with excess pixel space.
				desiredPPUY := ratio * ppuX
				if desiredPPUY <= cellH {
					// Shrink height to match ratio; centre vertically.
					newH := desiredPPUY * dataRangeY
					dataY += (cellH - newH) / 2 //nolint:mnd // Centre offset.
					cellH = newH
				} else {
					// Shrink width to match ratio; centre horizontally.
					desiredPPUX := ppuY / ratio
					newW := desiredPPUX * dataRangeX
					dataX += (cellW - newW) / 2 //nolint:mnd // Centre offset.
					cellW = newW
				}
			}
		}

		// When all layers are horizontal, swap scales and labels.
		renderXScale := xScale
		renderYScale := yScale
		renderXLabel := b.labels.X

		renderYLabel := b.labels.Y

		if allLayersHorizontal(builtLayersToLayerSpecs(bp.Layers)) {
			renderXScale, renderYScale = yScale, xScale
			renderXLabel, renderYLabel = renderYLabel, renderXLabel
		}

		// Draw grid.
		drawGrid(cv, renderXScale, renderYScale, dataX, dataY, cellW, cellH, th)

		// Panel border.
		if th.PanelBorder().Size > 0 {
			cv.SetColor(th.PanelBorder().Color)
			cv.SetLineWidth(th.PanelBorder().Size)
			cv.DrawRectangle(dataX, dataY, cellW, cellH)
			cv.Stroke()
		}

		// Store panel data rendering info for parallel execution.
		panelRenderInfos = append(panelRenderInfos, panelRenderInfo{
			panelIdx: pi,
			bp:       bp,
			xScale:   xScale,
			yScale:   yScale,
			dataX:    dataX,
			dataY:    dataY,
			cellW:    cellW,
			cellH:    cellH,
		})

		// Draw axes.
		isMultiPanel := len(b.panels) > 1
		isBottomRow := pl.Row == rows-1
		isLeftCol := pl.Col == 0

		// With free X scales, every row needs its own X-axis; otherwise only the bottom row.
		drawXOnThisPanel := !isMultiPanel || isBottomRow || b.layout.FreeX
		if drawXOnThisPanel {
			drawXAxis(cv, renderXScale, renderXLabel, dataX, dataY+cellH, cellW, th, b.ndodgeX)
		}

		// With free Y scales, every column needs its own Y-axis; otherwise only the left column.
		drawYOnThisPanel := !isMultiPanel || isLeftCol || b.layout.FreeY
		if drawYOnThisPanel {
			drawYAxis(cv, renderYScale, renderYLabel, dataX, dataY, cellH, th)
		}

		// Secondary Y-axis (right side).
		isRightCol := pl.Col == cols-1
		if b.secAxis != nil && (!isMultiPanel || isRightCol) {
			secScale := &scale.DerivedScale{Primary: renderYScale, Spec: *b.secAxis}
			drawYAxisRight(cv, secScale, b.secAxis.Name, dataX+cellW, dataY, cellH, th)
		}

		// Legend.
		if hasLegend {
			b.drawLegend(cv, bp, legendPos, dataX, dataY, cellW, cellH, mBottom, legendW, th)
		}
	}

	// Cache panel geometry for the Measurable interface. Controllers use this
	// to convert pixel positions to data coordinates for interactive zoom.
	geoms := make([]panelGeometry, len(panelRenderInfos))
	for i, pri := range panelRenderInfos {
		geoms[i] = panelGeometry{
			dataX:  pri.dataX,
			dataY:  pri.dataY,
			panelW: pri.cellW,
			panelH: pri.cellH,
		}
	}

	b.storePanelGeometry(width, height, geoms)

	// Render data layers — parallel when multiple panels, sequential otherwise.
	if len(panelRenderInfos) == 1 {
		// Single panel: draw directly onto main canvas, no sub-canvas overhead.
		pri := panelRenderInfos[0]

		cv.Save()
		cv.Translate(pri.dataX, pri.dataY)
		cv.ClipRect(0, 0, pri.cellW, pri.cellH)

		xMin, xMax := pri.xScale.Bounds()
		yMin, yMax := pri.yScale.Bounds()

		// Draw background annotations (rect) behind data layers.
		for i := range b.annotations {
			if b.annotations[i].Type == AnnotationRect {
				drawAnnotation(cv, b.coord, &b.annotations[i],
					pri.cellW, pri.cellH, xMin, xMax, yMin, yMax, th)
			}
		}

		for _, rl := range pri.bp.Layers {
			drawLayer(cv, b.coord, rl,
				pri.cellW, pri.cellH, xMin, xMax, yMin, yMax, th)
		}

		// Draw foreground annotations (text, label, segment, arrow) on top.
		for i := range b.annotations {
			if b.annotations[i].Type != AnnotationRect {
				drawAnnotation(cv, b.coord, &b.annotations[i],
					pri.cellW, pri.cellH, xMin, xMax, yMin, yMax, th)
			}
		}

		cv.Restore()
	} else if len(panelRenderInfos) > 1 {
		// Multi-panel: render each panel's data layers to a sub-canvas
		// in parallel, then composite onto the main canvas.
		type subResult struct {
			idx  int
			img  image.Image
			x, y float64
		}

		results := make([]subResult, len(panelRenderInfos))

		g, gctx := errgroup.WithContext(ctx)

		for i, pri := range panelRenderInfos {
			g.Go(func() error {
				if err := gctx.Err(); err != nil {
					return Errorf(PhaseDraw, -1, "context", err, "panel %d cancelled", i)
				}

				cw := int(pri.cellW + 0.5) //nolint:mnd // Round to nearest pixel.
				ch := int(pri.cellH + 0.5) //nolint:mnd // Round to nearest pixel.

				if cw < 1 {
					cw = 1
				}

				if ch < 1 {
					ch = 1
				}

				sub := canvas.NewRasterCanvasCPU(cw, ch)

				xMin, xMax := pri.xScale.Bounds()
				yMin, yMax := pri.yScale.Bounds()

				// Draw background annotations (rect) behind data layers.
				for j := range b.annotations {
					if b.annotations[j].Type == AnnotationRect {
						drawAnnotation(sub, b.coord, &b.annotations[j],
							pri.cellW, pri.cellH, xMin, xMax, yMin, yMax, th)
					}
				}

				for _, rl := range pri.bp.Layers {
					drawLayer(sub, b.coord, rl,
						pri.cellW, pri.cellH, xMin, xMax, yMin, yMax, th)
				}

				// Draw foreground annotations (text, label, segment, arrow) on top.
				for j := range b.annotations {
					if b.annotations[j].Type != AnnotationRect {
						drawAnnotation(sub, b.coord, &b.annotations[j],
							pri.cellW, pri.cellH, xMin, xMax, yMin, yMax, th)
					}
				}

				results[i] = subResult{
					idx: i,
					img: sub.Image(),
					x:   pri.dataX,
					y:   pri.dataY,
				}

				_ = sub.Close()

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return Errorf(PhaseDraw, -1, "panel", err, "parallel panel rendering")
		}

		// Composite sub-images onto main canvas.
		for _, r := range results {
			cv.DrawImage(r.img, r.x, r.y)
		}
	}

	// Title, subtitle, caption — drawn ONCE outside the panel loop.
	b.drawTitles(cv, width, height, th)

	return nil
}

// drawLegend renders the legend for a single panel.
func (b *Built) drawLegend(cv canvas.Canvas, bp BuiltPanel, legendPos string, dataX, dataY, cellW, cellH, mBottom, legendW float64, th theme.Theme) {
	switch legendPos {
	case "right":
		lx := dataX + cellW + 10 //nolint:mnd // Gap from panel edge.
		ly := legendStartY(th.Spacing.LegendAlign, dataY, cellH, estimateLegendH(bp, cellH, th))

		drawLegendStack(cv, bp, lx, ly, cellH, th)
	case "left":
		lx := dataX - legendW - 5 //nolint:mnd // Gap from panel edge.
		ly := legendStartY(th.Spacing.LegendAlign, dataY, cellH, estimateLegendH(bp, cellH, th))

		drawLegendStack(cv, bp, lx, ly, cellH, th)
	case "top":
		topY := dataY - 25 //nolint:mnd // Above panel.
		if len(bp.LegendEntries) > 0 {
			drawLegendHorizontal(cv, bp.LegendTitle, bp.LegendEntries, dataX, topY, cellW, th)
		}

		if bp.ColorBarSpec != nil {
			barW := math.Min(cellW*0.3, 200) //nolint:mnd // Max 200px bar width.
			barX := dataX + cellW/2 - barW/2 //nolint:mnd // Centre bar.
			drawColorBarHorizontal(cv, *bp.ColorBarSpec, barX, topY, barW, th)
		}

		if bp.SizeLegend != nil {
			drawSizeLegendHorizontal(cv, *bp.SizeLegend, dataX, topY-20, cellW, th) //nolint:mnd // Above categorical legend.
		}

		if bp.AlphaLegend != nil {
			barW := math.Min(cellW*0.2, 150)                                        //nolint:mnd // Max 150px alpha bar width.
			barX := dataX + cellW/2 - barW/2                                        //nolint:mnd // Centre bar.
			drawAlphaLegendHorizontal(cv, *bp.AlphaLegend, barX, topY-20, barW, th) //nolint:mnd // Above.
		}
	case "bottom":
		bottomY := dataY + cellH + mBottom - 25 //nolint:mnd // Below panel.
		if len(bp.LegendEntries) > 0 {
			drawLegendHorizontal(cv, bp.LegendTitle, bp.LegendEntries, dataX, bottomY, cellW, th)
		}

		if bp.ColorBarSpec != nil {
			barW := math.Min(cellW*0.3, 200) //nolint:mnd // Max 200px bar width.
			barX := dataX + cellW/2 - barW/2 //nolint:mnd // Centre bar.
			drawColorBarHorizontal(cv, *bp.ColorBarSpec, barX, bottomY, barW, th)
		}

		if bp.SizeLegend != nil {
			drawSizeLegendHorizontal(cv, *bp.SizeLegend, dataX, bottomY+20, cellW, th) //nolint:mnd // Below categorical legend.
		}

		if bp.AlphaLegend != nil {
			barW := math.Min(cellW*0.2, 150)                                           //nolint:mnd // Max 150px alpha bar width.
			barX := dataX + cellW/2 - barW/2                                           //nolint:mnd // Centre bar.
			drawAlphaLegendHorizontal(cv, *bp.AlphaLegend, barX, bottomY+20, barW, th) //nolint:mnd // Below.
		}
	}
}

// drawTitles renders title, subtitle, and caption.
func (b *Built) drawTitles(cv canvas.Canvas, width, height int, th theme.Theme) {
	w := float64(width)
	h := float64(height)
	titleY := 10.0

	// Resolve title alignment (default = center).
	titleX, titleAnchorX := alignXAnchor(th.Spacing.TitleAlign, w)

	if b.labels.Title != "" {
		cv.SetColor(th.PlotTitle().Color)
		cv.SetFontSize(th.PlotTitle().Size)
		cv.DrawStringAnchored(b.labels.Title, titleX, titleY+th.PlotTitle().Size/2, titleAnchorX, 0.5)
		titleY += th.PlotTitle().Size + 8 //nolint:mnd // Title-to-subtitle gap.
	}

	if b.labels.Subtitle != "" {
		cv.SetColor(th.PlotSubtitle().Color)
		cv.SetFontSize(th.PlotSubtitle().Size)
		cv.DrawStringAnchored(b.labels.Subtitle, titleX, titleY+th.PlotSubtitle().Size/2, titleAnchorX, 0.5)
	}

	if b.labels.Caption != "" {
		// Resolve caption alignment (default = right, matching ggplot2).
		captionAlign := th.Spacing.CaptionAlign
		if captionAlign == "" {
			captionAlign = theme.AlignRight
		}

		captionX, captionAnchorX := alignXAnchor(captionAlign, w)

		cv.SetColor(th.AxisTextElem().Color)
		cv.SetFontSize(th.AxisTextElem().Size)
		cv.DrawStringAnchored(b.labels.Caption, captionX, h-4, captionAnchorX, 1.0) //nolint:mnd // 4px bottom padding.
	}
}

// applyBlockAlignment applies per-plot alignment overrides to the theme.
// Non-empty fields in a override the corresponding Spacing alignment.
func applyBlockAlignment(th *theme.Theme, a *theme.BlockAlignment) {
	if a == nil {
		return
	}

	if a.Title != "" {
		th.Spacing.TitleAlign = a.Title
	}

	if a.Caption != "" {
		th.Spacing.CaptionAlign = a.Caption
	}

	if a.XLabel != "" {
		th.Spacing.XLabelAlign = a.XLabel
	}

	if a.YLabel != "" {
		th.Spacing.YLabelAlign = a.YLabel
	}

	if a.Legend != "" {
		th.Spacing.LegendAlign = a.Legend
	}
}

// alignXAnchor returns the x coordinate and anchor for a given horizontal
// alignment within a region of width w. The default (empty) is center.
func alignXAnchor(align theme.Align, w float64) (x, anchorX float64) {
	const pad = 10.0 // edge padding for left/right alignment

	switch align { //nolint:exhaustive // Only horizontal values are meaningful here.
	case theme.AlignLeft:
		return pad, 0
	case theme.AlignRight:
		return w - pad, 1.0
	default: // center or empty
		return w / 2, 0.5
	}
}

// alignXAnchorInRegion returns the x coordinate and anchor for a given
// horizontal alignment within a sub-region [regionX, regionX+regionW].
func alignXAnchorInRegion(align theme.Align, regionX, regionW float64) (x, anchorX float64) {
	switch align { //nolint:exhaustive // Only horizontal values are meaningful here.
	case theme.AlignLeft:
		return regionX, 0
	case theme.AlignRight:
		return regionX + regionW, 1.0
	default: // center or empty
		return regionX + regionW/2, 0.5
	}
}

// alignYAnchorRotated returns the translate-Y and text anchorX for a
// 90° counter-clockwise rotated label aligned within a vertical region
// [regionY, regionY+regionH]. After CCW rotation the text's X-axis
// maps to screen Y: higher anchorX pushes the text toward smaller Y (top).
func alignYAnchorRotated(align theme.Align, regionY, regionH float64) (cy, anchorX float64) {
	switch align { //nolint:exhaustive // Only vertical values are meaningful here.
	case theme.AlignTop:
		return regionY, 1.0
	case theme.AlignBottom:
		return regionY + regionH, 0
	default: // center or empty
		return regionY + regionH/2, 0.5
	}
}

// legendStartY returns the Y coordinate to start the legend stack based on
// vertical alignment within the cell. Default (empty or "top") = top-aligned.
func legendStartY(align theme.Align, dataY, cellH, legendH float64) float64 {
	const pad = 5.0 // edge padding

	switch align { //nolint:exhaustive // Only vertical values are meaningful here.
	case theme.AlignCenter:
		return dataY + (cellH-legendH)/2
	case theme.AlignBottom:
		return dataY + cellH - legendH - pad
	default: // top or empty (matches ggplot2 default)
		return dataY + pad
	}
}

// estimateLegendH computes the approximate total height of the legend stack
// for vertical alignment. This is a layout estimate — it does not need to
// be pixel-perfect, just close enough for center/bottom placement.
func estimateLegendH(bp BuiltPanel, cellH float64, th theme.Theme) float64 {
	ls := th.Spacing.Legend.Resolved()
	h := 0.0

	if len(bp.LegendEntries) > 0 {
		n := float64(len(bp.LegendEntries))
		h += ls.TitleSpacing + ls.SwatchSize/2 + (n-1)*ls.EntrySpacing + ls.SwatchSize/2 + ls.GroupGap
	}

	if bp.ColorBarSpec != nil {
		barH := math.Min(cellH*0.25, 120) //nolint:mnd // Match drawLegendStack bar height.
		h += ls.TitleSpacing + barH + ls.GroupGap
	}

	if bp.SizeLegend != nil {
		// Approximate: title + 5 entries × entrySpacing.
		nEntries := float64(len(bp.SizeLegend.Entries))
		if nEntries == 0 {
			nEntries = 5 //nolint:mnd // Fallback estimate.
		}

		h += ls.TitleSpacing + nEntries*ls.EntrySpacing + ls.GroupGap
	}

	if bp.AlphaLegend != nil {
		barH := math.Min(cellH*0.15, 80) //nolint:mnd // Match drawLegendStack bar height.
		h += ls.TitleSpacing + barH
	}

	return h
}

// builtLayersToLayerSpecs converts BuiltLayer slice to LayerSpec slice
// for compatibility with allLayersHorizontal.
func builtLayersToLayerSpecs(layers []BuiltLayer) []LayerSpec {
	specs := make([]LayerSpec, len(layers))
	for i, l := range layers {
		specs[i] = LayerSpec{Geom: l.Geom, Mapping: l.Mapping}
	}

	return specs
}

// ---------------------------------------------------------------------------
// Guide rendering (axes, grid, legend, color bar)
// ---------------------------------------------------------------------------
func drawXAxis(cv canvas.Canvas, sc scale.Scale, label string, x, y, w float64, th theme.Theme, ndodge int) {
	ticks := sc.Ticks(5)
	tickLen := th.Spacing.TickLength
	r, g, b, _ := rgbaOf(th.AxisTicks().Color)
	tr, tg, tb, _ := rgbaOf(th.AxisTextElem().Color)

	// Baseline.
	cv.SetRGBA(r, g, b, 1)
	cv.SetLineWidth(th.AxisTicks().Size)
	cv.DrawLine(x, y, x+w, y)
	cv.Stroke()

	xMin, xMax := sc.Bounds()
	angle := th.AxisTextElem().Angle
	fontSize := th.AxisTextElem().Size
	baseLabelY := y + tickLen + 12 //nolint:mnd // Tick-to-label gap in pixels.

	// Pre-measure all label pixel positions and widths for overlap detection.
	type tickLabel struct {
		v       float64
		frac    float64
		px      float64 // center x in pixels
		lbl     string
		lblW    float64 // measured text width
		visible bool
	}

	cv.SetFontSize(fontSize)
	cv.SetTabularNums(true)

	labels := make([]tickLabel, 0, len(ticks))

	for _, v := range ticks {
		frac := (v - xMin) / (xMax - xMin)
		if frac < 0 || frac > 1 {
			continue
		}

		px := x + frac*w
		lbl := sc.Format(v)
		tw, _ := cv.MeasureString(lbl)

		labels = append(labels, tickLabel{
			v:    v,
			frac: frac,
			px:   px,
			lbl:  lbl,
			lblW: tw,
		})
	}

	cv.SetTabularNums(false)

	// Auto-dodge: when ndodge == 0, detect overlapping labels and enable
	// 2-row staggering if any pair of adjacent labels would collide.
	effectiveNDodge := ndodge
	if effectiveNDodge == 0 && angle == 0 {
		// Check for overlaps in a single-row layout.
		const labelGap = 4 // minimum px between adjacent labels

		for i := 1; i < len(labels); i++ {
			prevRight := labels[i-1].px + labels[i-1].lblW/2
			currLeft := labels[i].px - labels[i].lblW/2

			if currLeft-prevRight < labelGap {
				effectiveNDodge = 2 //nolint:mnd // Auto-dodge to 2 rows on overlap.

				break
			}
		}
	}

	// Mark all labels as visible, then skip overlapping ones within each
	// dodge row. This ensures labels don't collide even after staggering.
	for i := range labels {
		labels[i].visible = true
	}

	if angle == 0 {
		const labelGap = 4

		nRows := max(effectiveNDodge, 1)

		// Track the last visible label per dodge row.
		lastVisible := make([]int, nRows)
		for i := range lastVisible {
			lastVisible[i] = -1
		}

		for i := range labels {
			row := 0
			if nRows > 1 {
				row = i % nRows
			}

			prev := lastVisible[row]
			if prev >= 0 {
				prevRight := labels[prev].px + labels[prev].lblW/2
				currLeft := labels[i].px - labels[i].lblW/2

				if currLeft-prevRight < labelGap {
					labels[i].visible = false

					continue
				}
			}

			lastVisible[row] = i
		}
	}

	// Draw tick marks and labels.
	rowHeight := fontSize + 4 //nolint:mnd // Per-row spacing for dodged labels.

	for i, tl := range labels {
		// Tick mark — always drawn.
		cv.SetRGBA(r, g, b, 1)
		cv.DrawLine(tl.px, y, tl.px, y+tickLen)
		cv.Stroke()

		if !tl.visible {
			continue
		}

		// Determine Y offset for this label's dodge row.
		labelY := baseLabelY

		if effectiveNDodge >= 2 {
			row := i % effectiveNDodge
			labelY += float64(row) * rowHeight
		}

		// Label.
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(fontSize)
		cv.SetTabularNums(true)

		if angle != 0 {
			cv.Save()
			cv.Translate(tl.px, labelY)
			cv.Rotate(angle * math.Pi / 180) //nolint:mnd // degrees-to-radians conversion.
			// Right-aligned anchor for rotated text so labels don't overlap the axis.
			cv.DrawStringAnchored(tl.lbl, 0, 0, 1.0, 0.5)
			cv.Restore()
		} else {
			cv.DrawStringAnchored(tl.lbl, tl.px, labelY, 0.5, 0.5)
		}

		cv.SetTabularNums(false)
	}

	// Axis title.
	if label != "" {
		lr, lg, lb, _ := rgbaOf(th.AxisTitle().Color)
		cv.SetRGBA(lr, lg, lb, 1)
		cv.SetFontSize(th.AxisTitle().Size)

		// Push title below the last dodge row.
		titleGapY := tickLen + 28 //nolint:mnd // Tick-to-title gap in pixels.
		if effectiveNDodge >= 2 {
			titleGapY += float64(effectiveNDodge-1) * rowHeight
		}

		// Resolve horizontal alignment (default = center).
		labelX, anchorX := alignXAnchorInRegion(th.Spacing.XLabelAlign, x, w)
		cv.DrawStringAnchored(label, labelX, y+titleGapY, anchorX, 0.5)
	}
}

// drawYAxis renders a vertical axis at the left of the data area.
func drawYAxis(cv canvas.Canvas, sc scale.Scale, label string, x, y, h float64, th theme.Theme) {
	ticks := sc.Ticks(5)
	tickLen := th.Spacing.TickLength
	r, g, b, _ := rgbaOf(th.AxisTicks().Color)
	tr, tg, tb, _ := rgbaOf(th.AxisTextElem().Color)

	// Baseline.
	cv.SetRGBA(r, g, b, 1)
	cv.SetLineWidth(th.AxisTicks().Size)
	cv.DrawLine(x, y, x, y+h)
	cv.Stroke()

	yMin, yMax := sc.Bounds()
	minSpacing := th.AxisTextElem().Size + 4 // minimum px between labels
	lastPy := -1000.0                        // track last drawn label position
	maxLabelW := 0.0                         // track widest drawn label

	// Measure + draw tick labels.
	cv.SetFontSize(th.AxisTextElem().Size)

	for _, v := range ticks {
		frac := (v - yMin) / (yMax - yMin)
		if frac < 0 || frac > 1 {
			continue
		}

		py := y + h - frac*h // invert y

		// Tick mark (always drawn).
		cv.SetRGBA(r, g, b, 1)
		cv.DrawLine(x, py, x-tickLen, py)
		cv.Stroke()

		// Label -- skip if too close to previous label.
		if math.Abs(py-lastPy) >= minSpacing {
			lbl := sc.Format(v)

			cv.SetTabularNums(true)

			tw, _ := cv.MeasureString(lbl)
			if tw > maxLabelW {
				maxLabelW = tw
			}

			cv.SetRGBA(tr, tg, tb, 1)
			cv.SetFontSize(th.AxisTextElem().Size)
			cv.DrawStringAnchored(lbl, x-tickLen-5, py, 1.0, 0.5)
			cv.SetTabularNums(false)

			lastPy = py
		}
	}

	// Axis title (rotated), positioned to the left of the widest tick label.
	if label != "" {
		lr, lg, lb, _ := rgbaOf(th.AxisTitle().Color)
		cv.SetRGBA(lr, lg, lb, 1)
		cv.SetFontSize(th.AxisTitle().Size)
		cv.Save()
		// The rotated title's horizontal extent equals its font size,
		// centred on the translate point. To avoid overlapping tick labels
		// (which extend left from x-tickLen-5 by maxLabelW) we offset by:
		//   5 (label gap) + maxLabelW + fontSize/2 + 8 (padding)
		titleOffset := 5 + maxLabelW + th.AxisTitle().Size/2 + 8
		if titleOffset < 30 {
			titleOffset = 30 // minimum offset for short labels
		}

		// Resolve vertical alignment for Y label (default = center).
		labelCY, anchorAX := alignYAnchorRotated(th.Spacing.YLabelAlign, y, h)
		cv.Translate(x-tickLen-titleOffset, labelCY)
		cv.Rotate(-math.Pi / 2)
		cv.DrawStringAnchored(label, 0, 0, anchorAX, 0.5)
		cv.Restore()
	}
}

// drawYAxisRight renders a vertical axis at the right of the data area.
// Used for secondary Y-axes. Mirrors drawYAxis with ticks/labels on the right.
func drawYAxisRight(cv canvas.Canvas, sc scale.Scale, label string, x, y, h float64, th theme.Theme) {
	ticks := sc.Ticks(5)
	tickLen := th.Spacing.TickLength
	r, g, b, _ := rgbaOf(th.AxisTicks().Color)
	tr, tg, tb, _ := rgbaOf(th.AxisTextElem().Color)

	// Baseline.
	cv.SetRGBA(r, g, b, 1)
	cv.SetLineWidth(th.AxisTicks().Size)
	cv.DrawLine(x, y, x, y+h)
	cv.Stroke()

	yMin, yMax := sc.Bounds()
	minSpacing := th.AxisTextElem().Size + 4 // minimum px between labels
	lastPy := -1000.0                        // track last drawn label position
	maxLabelW := 0.0                         // track widest drawn label

	// Measure + draw tick labels.
	cv.SetFontSize(th.AxisTextElem().Size)

	for _, v := range ticks {
		frac := (v - yMin) / (yMax - yMin)
		if frac < 0 || frac > 1 {
			continue
		}

		py := y + h - frac*h // invert y

		// Tick mark (always drawn) — extends right.
		cv.SetRGBA(r, g, b, 1)
		cv.DrawLine(x, py, x+tickLen, py)
		cv.Stroke()

		// Label — anchored on the left, skip if too close to previous label.
		if math.Abs(py-lastPy) >= minSpacing {
			lbl := sc.Format(v)

			cv.SetTabularNums(true)

			tw, _ := cv.MeasureString(lbl)
			if tw > maxLabelW {
				maxLabelW = tw
			}

			cv.SetRGBA(tr, tg, tb, 1)
			cv.SetFontSize(th.AxisTextElem().Size)
			cv.DrawStringAnchored(lbl, x+tickLen+5, py, 0.0, 0.5) //nolint:mnd // Label gap.
			cv.SetTabularNums(false)

			lastPy = py
		}
	}

	// Axis title (rotated), positioned to the right of the widest tick label.
	if label != "" {
		lr, lg, lb, _ := rgbaOf(th.AxisTitle().Color)
		cv.SetRGBA(lr, lg, lb, 1)
		cv.SetFontSize(th.AxisTitle().Size)
		cv.Save()

		titleOffset := 5 + maxLabelW + th.AxisTitle().Size/2 + 8 //nolint:mnd // Same spacing as left axis.
		if titleOffset < 30 {                                    //nolint:mnd // Minimum offset.
			titleOffset = 30 //nolint:mnd // Minimum offset for short labels.
		}

		cv.Translate(x+tickLen+titleOffset, y+h/2)
		cv.Rotate(math.Pi / 2)
		cv.DrawStringAnchored(label, 0, 0, 0.5, 0.5)
		cv.Restore()
	}
}

// drawGrid renders major grid lines in the data area, first filling the panel
// background with the theme's Panel.Background color.
// Grid lines use the theme's DashPattern (nil = solid, e.g. {4,4} = dashed).
// When a scale implements [scale.MinorTicker], minor grid lines are drawn
// between the major ones using the theme's MinorColor / MinorWidth.
func drawGrid(cv canvas.Canvas, xScale, yScale scale.Scale, x, y, w, h float64, th theme.Theme) {
	// Fill panel background first so it appears behind all grid lines and data.
	cv.SetColor(th.PanelBackground().Fill)
	cv.DrawRectangle(x, y, w, h)
	cv.Fill()

	mr, mg, mb, ma := rgbaOf(th.PanelGridMajor().Color)
	cv.SetRGBA(mr, mg, mb, ma)
	cv.SetLineWidth(th.PanelGridMajor().Size)

	// Apply dash pattern from theme.
	if len(th.Spacing.GridDashPattern) > 0 {
		cv.SetLineDash(th.Spacing.GridDashPattern...)
	}

	// Vertical grid lines (from x ticks).
	xMin, xMax := xScale.Bounds()
	for _, v := range xScale.Ticks(5) {
		frac := (v - xMin) / (xMax - xMin)
		if frac < 0 || frac > 1 {
			continue
		}

		px := x + frac*w
		cv.DrawLine(px, y, px, y+h)
		cv.Stroke()
	}

	// Horizontal grid lines (from y ticks).
	yMin, yMax := yScale.Bounds()
	for _, v := range yScale.Ticks(5) {
		frac := (v - yMin) / (yMax - yMin)
		if frac < 0 || frac > 1 {
			continue
		}

		py := y + h - frac*h
		cv.DrawLine(x, py, x+w, py)
		cv.Stroke()
	}

	// Reset to solid lines before drawing minor grid.
	cv.SetLineDash()

	// --- Minor grid lines ---
	drawMinorLines(cv, xScale, yScale, x, y, w, h, th)
}

// drawMinorLines renders minor grid lines for scales that implement
// [scale.MinorTicker]. Minor lines use the theme's MinorColor/MinorWidth.
func drawMinorLines(cv canvas.Canvas, xScale, yScale scale.Scale, x, y, w, h float64, th theme.Theme) {
	if th.PanelGridMinor().Color == nil {
		return
	}

	mr, mg, mb, ma := rgbaOf(th.PanelGridMinor().Color)
	cv.SetRGBA(mr, mg, mb, ma)
	cv.SetLineWidth(th.PanelGridMinor().Size)

	xMin, xMax := xScale.Bounds()
	yMin, yMax := yScale.Bounds()

	// Vertical minor grid lines.
	if mt, ok := xScale.(scale.MinorTicker); ok {
		for _, v := range mt.MinorTicks() {
			frac := (v - xMin) / (xMax - xMin)
			if frac < 0 || frac > 1 {
				continue
			}

			px := x + frac*w
			cv.DrawLine(px, y, px, y+h)
			cv.Stroke()
		}
	}

	// Horizontal minor grid lines.
	if mt, ok := yScale.(scale.MinorTicker); ok {
		for _, v := range mt.MinorTicks() {
			frac := (v - yMin) / (yMax - yMin)
			if frac < 0 || frac > 1 {
				continue
			}

			py := y + h - frac*h
			cv.DrawLine(x, py, x+w, py)
			cv.Stroke()
		}
	}
}

// --- Legend ---

// LegendGlyph controls the key shape drawn beside each legend entry.
type LegendGlyph int

const (
	// GlyphRect draws a filled square swatch (bars, histogram, tile, area).
	GlyphRect LegendGlyph = iota
	// GlyphPoint draws a filled circle (point, rug).
	GlyphPoint
	// GlyphLine draws a horizontal line stroke (line, smooth, step, segment).
	GlyphLine
)

// glyphForGeom returns the appropriate legend glyph for a geometry type.
func glyphForGeom(t geom.Type) LegendGlyph {
	switch t { //nolint:exhaustive // Only groupable geom types produce legend entries.
	case geom.TypePoint, geom.TypeRug:
		return GlyphPoint
	case geom.TypeLine, geom.TypeSmooth, geom.TypeStep, geom.TypeSegment:
		return GlyphLine
	default:
		return GlyphRect
	}
}

// LegendEntry describes one item in the legend.
type LegendEntry struct {
	Label    string
	Color    gg.RGBA
	Glyph    LegendGlyph
	Shape    string    // e.g. "square", "triangle"
	Linetype []float64 // dash pattern
}

// drawGlyph draws a legend key glyph at (x, y) with the given size.
// The color must be set by the caller before calling drawGlyph.
func drawGlyph(cv canvas.Canvas, e LegendEntry, x, y, size float64) {
	switch e.Glyph {
	case GlyphRect:
		cv.DrawRectangle(x, y-size/2, size, size)
		cv.Fill()
	case GlyphPoint:
		if e.Shape != "" {
			if canvas.IsStrokeShape(e.Shape) {
				cv.SetLineWidth(1.5)
				cv.DrawShape(e.Shape, x+size/2, y, size/2)
				cv.Stroke()
			} else {
				cv.DrawShape(e.Shape, x+size/2, y, size/2)
				cv.Fill()
			}
		} else {
			radius := size / 2
			cv.DrawCircle(x+radius, y, radius)
			cv.Fill()
		}
	case GlyphLine:
		lw := 2.5 //nolint:mnd // Standard legend line stroke width.
		cv.SetLineWidth(lw)

		if len(e.Linetype) > 0 {
			cv.SetLineDash(e.Linetype...)
		}

		cv.DrawLine(x, y, x+size, y)
		cv.Stroke()

		if len(e.Linetype) > 0 {
			cv.SetLineDash()
		}
	}
}

// drawLegendStack draws all vertical legend groups (categorical, colour bar,
// size, alpha) stacked from top to bottom, using theme-controlled spacing.
func drawLegendStack(cv canvas.Canvas, bp BuiltPanel, lx, ly, cellH float64, th theme.Theme) {
	ls := th.Spacing.Legend.Resolved()

	if len(bp.LegendEntries) > 0 {
		drawLegendVertical(cv, bp.LegendTitle, bp.LegendEntries, lx, ly, th)
		// Advance past: title + n entries (each centered) + half of last swatch.
		n := float64(len(bp.LegendEntries))
		ly += ls.TitleSpacing + ls.SwatchSize/2 + (n-1)*ls.EntrySpacing + ls.SwatchSize/2 + ls.GroupGap
	}

	if bp.ColorBarSpec != nil {
		barH := math.Min(cellH*0.25, 120) //nolint:mnd // Max 120px bar height.
		drawColorBar(cv, *bp.ColorBarSpec, lx, ly, barH, th)
		ly += barH + ls.GroupGap
	}

	if bp.SizeLegend != nil {
		bottomY := drawSizeLegend(cv, *bp.SizeLegend, lx, ly, th)
		ly = bottomY + ls.GroupGap
	}

	if bp.AlphaLegend != nil {
		barH := math.Min(cellH*0.15, 80) //nolint:mnd // Max 80px alpha bar height.
		drawAlphaLegend(cv, *bp.AlphaLegend, lx, ly, barH, th)
	}
}

// drawLegendVertical renders a categorical legend to the right of the data area.
func drawLegendVertical(cv canvas.Canvas, title string, entries []LegendEntry, x, y float64, th theme.Theme) {
	if len(entries) == 0 {
		return
	}

	ls := th.Spacing.Legend.Resolved()
	swatchSize := ls.SwatchSize
	spacing := ls.EntrySpacing
	padding := ls.Padding

	// Measure total legend height for background rect.
	totalH := padding
	if title != "" {
		totalH += ls.TitleSpacing
	}

	totalH += float64(len(entries)) * spacing
	totalH += padding

	// Measure max label width for background rect width.
	cv.SetFontSize(th.LegendTextElem().Size)

	maxLabelW := 0.0

	for _, e := range entries {
		tw, _ := cv.MeasureString(e.Label)
		if tw > maxLabelW {
			maxLabelW = tw
		}
	}

	totalW := padding + swatchSize + ls.SwatchGap + maxLabelW + padding

	// Draw legend background.
	bg := th.LegendBackground()
	if !theme.IsBlank(bg) && bg.Fill != nil {
		r, g, b, a := bg.Fill.RGBA()
		if a > 0 {
			cv.SetRGBA(float64(r)/65535.0, float64(g)/65535.0, float64(b)/65535.0, float64(a)/65535.0)
			cv.DrawRectangle(x-padding, y-spacing/2-padding, totalW, totalH)
			cv.Fill()
		}

		// Background border.
		if bg.Color != nil && bg.Size > 0 {
			br, bg2, bb, ba := bg.Color.RGBA()
			if ba > 0 {
				cv.SetRGBA(float64(br)/65535.0, float64(bg2)/65535.0, float64(bb)/65535.0, float64(ba)/65535.0)
				cv.SetLineWidth(bg.Size)
				cv.DrawRectangle(x-padding, y-spacing/2-padding, totalW, totalH)
				cv.Stroke()
			}
		}
	}

	curY := y

	if title != "" {
		titleElem := th.LegendTitle()
		tr, tg, tb, _ := rgbaOf(titleElem.Color)
		cv.SetRGBA(tr, tg, tb, 1)

		titleSize := titleElem.Size
		if titleSize == 0 {
			titleSize = th.LegendTextElem().Size
		}

		cv.SetFontSize(titleSize)

		// Left-align the title within the legend column.
		cv.DrawStringAnchored(title, x, curY, 0, 0.5)
		// Advance by TitleSpacing (visual gap) + half swatch so the swatch
		// centre lands at curY and its top edge is TitleSpacing from the title.
		curY += ls.TitleSpacing + swatchSize/2
	}

	keyRect := th.LegendKey()

	for _, e := range entries {
		// Draw legend key background behind swatch.
		if !theme.IsBlank(keyRect) && keyRect.Fill != nil {
			kr, kg, kb, ka := keyRect.Fill.RGBA()
			if ka > 0 {
				cv.SetRGBA(float64(kr)/65535.0, float64(kg)/65535.0, float64(kb)/65535.0, float64(ka)/65535.0)
				cv.DrawRectangle(x-1, curY-swatchSize/2-1, swatchSize+2, swatchSize+2) //nolint:mnd // 1px padding around key.
				cv.Fill()
			}
		}

		cv.SetRGBA(e.Color.R, e.Color.G, e.Color.B, e.Color.A)
		drawGlyph(cv, e, x, curY, swatchSize)

		r, g, b, _ := rgbaOf(th.LegendTextElem().Color)
		cv.SetRGBA(r, g, b, 1)
		cv.SetFontSize(th.LegendTextElem().Size)
		cv.DrawStringAnchored(e.Label, x+swatchSize+ls.SwatchGap, curY, 0, 0.5)

		curY += spacing
	}
}

// ColorBarSpec describes a continuous color bar legend.
//
// Cmap and Norm replace the previous opaque ColorFunc field: the bar walks
// Cmap.At directly across the [0,1] range, and Norm provides the data-space
// labels at the endpoints (and any future intermediate ticks).
type ColorBarSpec struct {
	Title    string
	Cmap     colormap.Cmap
	Norm     colormap.Norm
	BarWidth float64 // 0 = default (12px)
	NBin     int     // 0 = default (pixel-matched strips)
}

// SizeLegendSpec describes a graduated-size legend showing representative
// point sizes from a continuous size mapping.
type SizeLegendSpec struct {
	Title   string
	Entries []SizeLegendEntry
}

// SizeLegendEntry is one representative value in a graduated-size legend.
type SizeLegendEntry struct {
	Label  string
	Radius float64 // pixel radius for this representative value
}

// AlphaLegendSpec describes a continuous-alpha legend rendered as a
// gradient strip from minimum to maximum opacity.
type AlphaLegendSpec struct {
	Title     string
	MinAlpha  float64 // minimum opacity [0,1]
	MaxAlpha  float64 // maximum opacity [0,1]
	MinValue  float64 // data-space value at min opacity
	MaxValue  float64 // data-space value at max opacity
	BaseColor gg.RGBA // the colour whose opacity varies
}

// drawColorBar renders a continuous color bar legend at the given position.
// The bar is drawn vertically (top = max, bottom = min) as in ggplot2.
func drawColorBar(cv canvas.Canvas, spec ColorBarSpec, x, y, barH float64, th theme.Theme) {
	barW := spec.BarWidth
	if barW <= 0 {
		barW = th.Spacing.Legend.Resolved().BarWidth
	}

	// Title above the bar.
	tr, tg, tb, _ := rgbaOf(th.LegendTextElem().Color)
	if spec.Title != "" {
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.LegendTextElem().Size)
		cv.DrawStringAnchored(spec.Title, x, y, 0, 0.5)
		y += th.Spacing.Legend.Resolved().TitleSpacing
	}

	cm := spec.Cmap
	if cm == nil {
		cm = colormap.Viridis
	}

	// Draw gradient bar as thin horizontal strips (top = max, bottom = min).
	nStrips := spec.NBin
	if nStrips <= 0 {
		nStrips = max(int(barH), 2) // pixel-matched strips
	}

	stripH := barH / float64(nStrips)
	for i := range nStrips {
		// t=1 at top (max), t=0 at bottom (min).
		t := 1.0 - float64(i)/float64(nStrips-1)
		c := cm.At(t)
		cv.SetRGBA(c.R, c.G, c.B, c.A)
		cv.DrawRectangle(x, y+float64(i)*stripH, barW, stripH+0.5)
		cv.Fill()
	}

	// Outline.
	cv.SetRGBA(tr, tg, tb, 0.5)
	cv.SetLineWidth(0.5)
	cv.DrawRectangle(x, y, barW, barH)
	cv.Stroke()

	// Max label (top) and Min label (bottom). Use the Norm's data-space
	// bounds when available; otherwise fall back to "high" / "low".
	cv.SetRGBA(tr, tg, tb, 1)
	cv.SetFontSize(th.LegendTextElem().Size * 0.9)

	labelX := x + barW + 4
	hi, lo := "high", "low"

	if spec.Norm != nil {
		vmin, vmax := spec.Norm.Bounds()
		hi = guideFormatNum(vmax)
		lo = guideFormatNum(vmin)
	}

	cv.DrawStringAnchored(hi, labelX, y+4, 0, 0.5)
	cv.DrawStringAnchored(lo, labelX, y+barH-4, 0, 0.5)
}

func guideFormatNum(v float64) string {
	if v == float64(int(v)) {
		return fmt.Sprintf("%.1f", v)
	}

	return fmt.Sprintf("%.2g", v)
}

// drawLegendHorizontal renders a categorical legend as a horizontal row.
func drawLegendHorizontal(cv canvas.Canvas, title string, entries []LegendEntry, x, y, maxW float64, th theme.Theme) {
	if len(entries) == 0 {
		return
	}

	swatchSize := 10.0
	gap := 8.0
	curX := x

	if title != "" {
		titleElem := th.LegendTitle()
		tr, tg, tb, _ := rgbaOf(titleElem.Color)
		cv.SetRGBA(tr, tg, tb, 1)

		titleSize := titleElem.Size
		if titleSize == 0 {
			titleSize = th.LegendTextElem().Size
		}

		cv.SetFontSize(titleSize)
		cv.DrawStringAnchored(title, curX, y, 0, 0.5)
		tw, _ := cv.MeasureString(title)
		curX += tw + gap*2 //nolint:mnd // Double gap after title.
	}

	tr, tg, tb, _ := rgbaOf(th.LegendTextElem().Color)
	keyRect := th.LegendKey()

	for _, e := range entries {
		if curX > x+maxW {
			break
		}

		// Draw legend key background behind swatch.
		if !theme.IsBlank(keyRect) && keyRect.Fill != nil {
			kr, kg, kb, ka := keyRect.Fill.RGBA()
			if ka > 0 {
				cv.SetRGBA(float64(kr)/65535.0, float64(kg)/65535.0, float64(kb)/65535.0, float64(ka)/65535.0)
				cv.DrawRectangle(curX-1, y-swatchSize/2-1, swatchSize+2, swatchSize+2) //nolint:mnd // 1px padding around key.
				cv.Fill()
			}
		}

		cv.SetRGBA(e.Color.R, e.Color.G, e.Color.B, e.Color.A)
		drawGlyph(cv, e, curX, y, swatchSize)

		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.LegendTextElem().Size * 0.9)               //nolint:mnd // Slightly smaller for horizontal legend.
		cv.DrawStringAnchored(e.Label, curX+swatchSize+3, y, 0, 0.5) //nolint:mnd // swatch + gap offset.
		tw, _ := cv.MeasureString(e.Label)
		curX += swatchSize + 3 + tw + gap //nolint:mnd // swatch + gap + label + inter-entry gap.
	}
}

// drawColorBarHorizontal renders a horizontal continuous color bar legend.
func drawColorBarHorizontal(cv canvas.Canvas, spec ColorBarSpec, x, y, barW float64, th theme.Theme) {
	barH := 10.0

	tr, tg, tb, _ := rgbaOf(th.LegendTextElem().Color)

	// Title to the left.
	startX := x

	if spec.Title != "" {
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.LegendTextElem().Size)
		cv.DrawStringAnchored(spec.Title, startX, y+barH/2, 0, 0.5)
		tw, _ := cv.MeasureString(spec.Title)
		startX += tw + 8
	}

	cm := spec.Cmap
	if cm == nil {
		cm = colormap.Viridis
	}

	availW := barW - (startX - x)
	if availW < 20 {
		availW = 20
	}

	// Draw gradient bar as thin vertical strips.
	nStrips := max(int(availW), 2)

	stripW := availW / float64(nStrips)
	for i := range nStrips {
		t := float64(i) / float64(nStrips-1)
		c := cm.At(t)
		cv.SetRGBA(c.R, c.G, c.B, c.A)
		cv.DrawRectangle(startX+float64(i)*stripW, y, stripW+0.5, barH)
		cv.Fill()
	}

	// Outline.
	cv.SetRGBA(tr, tg, tb, 0.4)
	cv.SetLineWidth(0.5)
	cv.DrawRectangle(startX, y, availW, barH)
	cv.Stroke()

	// Min / Max labels.
	cv.SetRGBA(tr, tg, tb, 1)
	cv.SetFontSize(th.LegendTextElem().Size * 0.85)

	lo, hi := "low", "high"

	if spec.Norm != nil {
		vmin, vmax := spec.Norm.Bounds()
		lo = guideFormatNum(vmin)
		hi = guideFormatNum(vmax)
	}

	cv.DrawStringAnchored(lo, startX, y+barH+10, 0.5, 0.5)        //nolint:mnd // label offset below bar.
	cv.DrawStringAnchored(hi, startX+availW, y+barH+10, 0.5, 0.5) //nolint:mnd // label offset below bar.
}

// drawSizeLegend renders a graduated-size legend vertically.
// Each entry is drawn as a circle of the corresponding radius, from smallest
// to largest, with data-space labels to the right.
func drawSizeLegend(cv canvas.Canvas, spec SizeLegendSpec, x, y float64, th theme.Theme) float64 {
	if len(spec.Entries) == 0 {
		return y
	}

	tr, tg, tb, _ := rgbaOf(th.LegendTextElem().Color)

	curY := y

	// Pre-compute max radius and label width for layout.
	maxRadius := 0.0
	maxLabelW := 0.0

	cv.SetFontSize(th.LegendTextElem().Size)

	for _, e := range spec.Entries {
		if e.Radius > maxRadius {
			maxRadius = e.Radius
		}

		tw, _ := cv.MeasureString(e.Label)
		if tw > maxLabelW {
			maxLabelW = tw
		}
	}

	if spec.Title != "" {
		titleElem := th.LegendTitle()
		titleR, titleG, titleB, _ := rgbaOf(titleElem.Color)
		cv.SetRGBA(titleR, titleG, titleB, 1)

		titleSize := titleElem.Size
		if titleSize == 0 {
			titleSize = th.LegendTextElem().Size
		}

		cv.SetFontSize(titleSize)

		// Left-align the title within the legend column.
		cv.DrawStringAnchored(spec.Title, x, curY, 0, 0.5)
		// Advance by TitleSpacing + first entry radius so the circle top
		// is TitleSpacing from the title.
		firstR := spec.Entries[0].Radius
		if firstR < 1 {
			firstR = 1
		}

		curY += th.Spacing.Legend.Resolved().TitleSpacing + firstR
	}

	for i, e := range spec.Entries {
		// Centre circle at (x + maxRadius, curY).
		cx := x + maxRadius
		r := e.Radius

		if r < 1 {
			r = 1
		}

		// Draw circle outline (filled with theme text colour at low alpha).
		cv.SetRGBA(tr, tg, tb, 0.3) //nolint:mnd // 30% fill for size legend circles.
		cv.DrawCircle(cx, curY, r)
		cv.Fill()

		// Draw circle border.
		cv.SetRGBA(tr, tg, tb, 0.7) //nolint:mnd // 70% border for size legend circles.
		cv.SetLineWidth(0.8)        //nolint:mnd // Thin stroke for legend circle border.
		cv.DrawCircle(cx, curY, r)
		cv.Stroke()

		// Label to the right of the largest possible circle.
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.LegendTextElem().Size)

		labelX := x + maxRadius*2 + th.Spacing.Legend.Resolved().SwatchGap
		cv.DrawStringAnchored(e.Label, labelX, curY, 0, 0.5)

		// Advance to next entry (skip after the last one — GroupGap handles it).
		if i < len(spec.Entries)-1 {
			spacing := max(maxRadius*2+4, 16) //nolint:mnd // Minimum 16px spacing between entries.
			curY += spacing
		}
	}
	// Return the bottom edge of the last circle so GroupGap works consistently.
	lastR := spec.Entries[len(spec.Entries)-1].Radius
	if lastR < 1 {
		lastR = 1
	}

	return curY + lastR
}

// drawSizeLegendHorizontal renders a graduated-size legend horizontally.
func drawSizeLegendHorizontal(cv canvas.Canvas, spec SizeLegendSpec, x, y, maxW float64, th theme.Theme) {
	if len(spec.Entries) == 0 {
		return
	}

	tr, tg, tb, _ := rgbaOf(th.LegendTextElem().Color)
	curX := x

	if spec.Title != "" {
		titleElem := th.LegendTitle()
		titleR, titleG, titleB, _ := rgbaOf(titleElem.Color)
		cv.SetRGBA(titleR, titleG, titleB, 1)

		titleSize := titleElem.Size
		if titleSize == 0 {
			titleSize = th.LegendTextElem().Size
		}

		cv.SetFontSize(titleSize)
		cv.DrawStringAnchored(spec.Title, curX, y, 0, 0.5)
		tw, _ := cv.MeasureString(spec.Title)
		curX += tw + 12 //nolint:mnd // Gap after title.
	}

	for _, e := range spec.Entries {
		if curX > x+maxW {
			break
		}

		r := e.Radius
		if r < 1 {
			r = 1
		}

		cx := curX + r

		cv.SetRGBA(tr, tg, tb, 0.3) //nolint:mnd // 30% fill.
		cv.DrawCircle(cx, y, r)
		cv.Fill()

		cv.SetRGBA(tr, tg, tb, 0.7) //nolint:mnd // 70% border.
		cv.SetLineWidth(0.8)        //nolint:mnd // Thin stroke.
		cv.DrawCircle(cx, y, r)
		cv.Stroke()

		// Label below circle.
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.LegendTextElem().Size * 0.85)     //nolint:mnd // Slightly smaller for compact layout.
		cv.DrawStringAnchored(e.Label, cx, y+r+8, 0.5, 0.5) //nolint:mnd // Below circle.
		tw, _ := cv.MeasureString(e.Label)
		curX += r*2 + max(tw, 4) + 8 //nolint:mnd // diameter + label + gap.
	}
}

// drawAlphaLegend renders a continuous-alpha gradient strip vertically.
// The strip fades from MinAlpha (bottom) to MaxAlpha (top) using the
// base colour, with endpoint labels.
func drawAlphaLegend(cv canvas.Canvas, spec AlphaLegendSpec, x, y, barH float64, th theme.Theme) {
	barW := th.Spacing.Legend.Resolved().BarWidth

	tr, tg, tb, _ := rgbaOf(th.LegendTextElem().Color)
	curY := y

	if spec.Title != "" {
		titleElem := th.LegendTitle()
		titleR, titleG, titleB, _ := rgbaOf(titleElem.Color)
		cv.SetRGBA(titleR, titleG, titleB, 1)

		titleSize := titleElem.Size
		if titleSize == 0 {
			titleSize = th.LegendTextElem().Size
		}

		cv.SetFontSize(titleSize)
		cv.DrawStringAnchored(spec.Title, x, curY, 0, 0.5)
		curY += th.Spacing.Legend.Resolved().TitleSpacing
	}

	// Draw gradient strip (top = MaxAlpha, bottom = MinAlpha).
	nStrips := max(int(barH), 2)
	stripH := barH / float64(nStrips)

	for i := range nStrips {
		// t=1 at top (max), t=0 at bottom (min).
		t := 1.0 - float64(i)/float64(nStrips-1)
		alpha := spec.MinAlpha + t*(spec.MaxAlpha-spec.MinAlpha)
		cv.SetRGBA(spec.BaseColor.R, spec.BaseColor.G, spec.BaseColor.B, alpha)
		cv.DrawRectangle(x, curY+float64(i)*stripH, barW, stripH+0.5) //nolint:mnd // 0.5px overlap to avoid gaps.
		cv.Fill()
	}

	// Outline.
	cv.SetRGBA(tr, tg, tb, 0.5) //nolint:mnd // Semi-transparent outline.
	cv.SetLineWidth(0.5)        //nolint:mnd // Thin outline.
	cv.DrawRectangle(x, curY, barW, barH)
	cv.Stroke()

	// Labels: data-space values at top (max) and bottom (min).
	cv.SetRGBA(tr, tg, tb, 1)
	cv.SetFontSize(th.LegendTextElem().Size * 0.9) //nolint:mnd // Slightly smaller for legend labels.

	labelX := x + barW + 4                                                                //nolint:mnd // Gap between bar and labels.
	cv.DrawStringAnchored(scale.FormatNumber(spec.MaxValue), labelX, curY+4, 0, 0.5)      //nolint:mnd // Top label offset.
	cv.DrawStringAnchored(scale.FormatNumber(spec.MinValue), labelX, curY+barH-4, 0, 0.5) //nolint:mnd // Bottom label offset.
}

// drawAlphaLegendHorizontal renders a continuous-alpha gradient strip horizontally.
func drawAlphaLegendHorizontal(cv canvas.Canvas, spec AlphaLegendSpec, x, y, barW float64, th theme.Theme) {
	barH := 10.0 //nolint:mnd // Default horizontal bar height.

	tr, tg, tb, _ := rgbaOf(th.LegendTextElem().Color)
	startX := x

	if spec.Title != "" {
		titleElem := th.LegendTitle()
		titleR, titleG, titleB, _ := rgbaOf(titleElem.Color)
		cv.SetRGBA(titleR, titleG, titleB, 1)

		titleSize := titleElem.Size
		if titleSize == 0 {
			titleSize = th.LegendTextElem().Size
		}

		cv.SetFontSize(titleSize)
		cv.DrawStringAnchored(spec.Title, startX, y+barH/2, 0, 0.5) //nolint:mnd // Left-aligned.
		tw, _ := cv.MeasureString(spec.Title)
		startX += tw + 8 //nolint:mnd // Title-to-bar gap.
	}

	availW := barW - (startX - x)
	if availW < 20 { //nolint:mnd // Minimum bar width.
		availW = 20
	}

	nStrips := max(int(availW), 2)
	stripW := availW / float64(nStrips)

	for i := range nStrips {
		t := float64(i) / float64(nStrips-1)
		alpha := spec.MinAlpha + t*(spec.MaxAlpha-spec.MinAlpha)
		cv.SetRGBA(spec.BaseColor.R, spec.BaseColor.G, spec.BaseColor.B, alpha)
		cv.DrawRectangle(startX+float64(i)*stripW, y, stripW+0.5, barH) //nolint:mnd // 0.5px overlap.
		cv.Fill()
	}

	// Outline.
	cv.SetRGBA(tr, tg, tb, 0.4) //nolint:mnd // Semi-transparent outline.
	cv.SetLineWidth(0.5)        //nolint:mnd // Thin outline.
	cv.DrawRectangle(startX, y, availW, barH)
	cv.Stroke()

	// Labels: data-space values at left (min) and right (max).
	cv.SetRGBA(tr, tg, tb, 1)
	cv.SetFontSize(th.LegendTextElem().Size * 0.85)                                              //nolint:mnd // Slightly smaller.
	cv.DrawStringAnchored(scale.FormatNumber(spec.MinValue), startX, y+barH+10, 0.5, 0.5)        //nolint:mnd // Below left.
	cv.DrawStringAnchored(scale.FormatNumber(spec.MaxValue), startX+availW, y+barH+10, 0.5, 0.5) //nolint:mnd // Below right.
}

// --- Helpers ---

func rgbaOf(c color.Color) (float64, float64, float64, float64) {
	if c == nil {
		return 0, 0, 0, 1
	}

	r, g, b, a := c.RGBA()

	return float64(r) / 65535.0, float64(g) / 65535.0, float64(b) / 65535.0, float64(a) / 65535.0
}
