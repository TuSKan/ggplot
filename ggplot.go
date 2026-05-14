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
//	    Layer(geom.Smooth(geom.WithMethod("lm"))).
//	    Labs(ggplot.Title("My Plot"), ggplot.XLab("X Axis")).
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
	"image/color"
	"io"
	"maps"
	"math"

	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gogpu/gg"

	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/canvas"
	"github.com/TuSKan/ggplot/colormap"
	"github.com/TuSKan/ggplot/coord"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/facet"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/position"
	"github.com/TuSKan/ggplot/scale"
	"github.com/TuSKan/ggplot/stat"
	"github.com/TuSKan/ggplot/theme"
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
func (p *Plot) FacetGrid(rowCol, colCol string) *Plot {
	cloned := p.clone()
	cloned.spec.Facet = facet.Grid(rowCol, colCol)

	return cloned
}

// Theme sets the visual theme.
func (p *Plot) Theme(name theme.Name) *Plot {
	cloned := p.clone()
	cloned.spec.ThemeName = name

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

// LegendPosition sets the legend placement.
func (p *Plot) LegendPosition(pos LegendPos) *Plot {
	cloned := p.clone()
	cloned.spec.LegendPosition = string(pos)

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

// Labs configures plot labels (title, subtitle, axis labels, caption).
func (p *Plot) Labs(opts ...LabOpt) *Plot {
	cloned := p.clone()
	for _, opt := range opts {
		opt(&cloned.spec.Labels)
	}

	return cloned
}

// LabOpt is a functional option for configuring plot labels.
type LabOpt func(*Labels)

// Title sets the plot title.
func Title(text string) LabOpt { return func(l *Labels) { l.Title = text } }

// Subtitle sets the plot subtitle.
func Subtitle(text string) LabOpt { return func(l *Labels) { l.Subtitle = text } }

// XLab sets the x-axis label.
func XLab(text string) LabOpt { return func(l *Labels) { l.X = text } }

// YLab sets the y-axis label.
func YLab(text string) LabOpt { return func(l *Labels) { l.Y = text } }

// Caption sets the plot caption.
func Caption(text string) LabOpt { return func(l *Labels) { l.Caption = text } }

// RenderOpt configures rendering output (scale, DPI, etc.).
type RenderOpt func(*renderConfig)

type renderConfig struct {
	scale float64
}

func defaultRenderConfig() renderConfig {
	return renderConfig{scale: 1.0}
}

// WithScale sets the DPI scale factor for rendering.
// scale=2.0 produces retina-resolution output (2× pixel density).
func WithScale(s float64) RenderOpt {
	return func(c *renderConfig) {
		if s > 0 {
			c.scale = s
		}
	}
}

// Save renders the plot to a file at the given dimensions.
// The output format is inferred from the file extension:
//
//	.png — raster PNG (default)
//	.svg — SVG 1.1 vector
//	.pdf — PDF 1.4 vector
//
// Options: [WithScale] for HiDPI output.
func (p *Plot) Save(ctx context.Context, filename string, width, height int, opts ...RenderOpt) error {
	built, err := p.Build(ctx)
	if err != nil {
		return fmt.Errorf("ggplot: %w", err)
	}

	return built.Save(ctx, filename, width, height, opts...)
}

// WriteTo renders the plot and writes the output to w in the given format.
// Supported formats: "png" (default), "svg", "pdf".
// Options: [WithScale] for HiDPI output.
// Returns the number of bytes written.
//
// Shorthand for [Plot.Build] followed by [Built.WriteTo].
func (p *Plot) WriteTo(ctx context.Context, w io.Writer, format string, width, height int, opts ...RenderOpt) (int64, error) {
	built, err := p.Build(ctx)
	if err != nil {
		return 0, fmt.Errorf("ggplot: %w", err)
	}

	return built.WriteTo(ctx, w, format, width, height, opts...)
}

// --- Built convenience methods ---

// DrawCanvas creates a new [canvas.GGCanvas] and draws the built plot onto it.
func (b *Built) DrawCanvas(ctx context.Context, width, height int) (*canvas.GGCanvas, error) {
	cv := canvas.NewGGCanvas(width, height)
	if err := b.Draw(ctx, cv, width, height); err != nil {
		return nil, err
	}

	return cv, nil
}

// Save renders the built plot to a file. Format is inferred from extension.
//
//	.png — raster PNG (default)
//	.svg — SVG 1.1 vector
//	.pdf — PDF 1.4 vector
//
// Options: [WithScale] for HiDPI output.
func (b *Built) Save(ctx context.Context, filename string, width, height int, opts ...RenderOpt) error {
	cfg := defaultRenderConfig()
	for _, o := range opts {
		o(&cfg)
	}

	sw, sh := int(float64(width)*cfg.scale), int(float64(height)*cfg.scale)

	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".svg", ".pdf":
		cv := canvas.NewRecordingCanvas(sw, sh)
		if err := b.Draw(ctx, cv, sw, sh); err != nil {
			return fmt.Errorf("ggplot: %w", err)
		}

		rec := cv.FinishRecording()

		f, err := os.Create(filename) //nolint:gosec // G304: user-provided plot output path.
		if err != nil {
			return fmt.Errorf("ggplot: %w", err)
		}
		defer func() { _ = f.Close() }()

		switch ext {
		case ".svg":
			_, err = canvas.ExportSVG(rec, f)
		case ".pdf":
			_, err = canvas.ExportPDF(rec, f)
		}

		if err != nil {
			return fmt.Errorf("ggplot: %w", err)
		}

		return nil

	default:
		cv := canvas.NewGGCanvas(sw, sh)
		if err := b.Draw(ctx, cv, sw, sh); err != nil {
			return fmt.Errorf("ggplot: %w", err)
		}

		if err := cv.SavePNG(filename); err != nil {
			return fmt.Errorf("ggplot: %w", err)
		}

		return nil
	}
}

// WriteTo writes the built plot to w in the given format.
// Supported formats: "png" (default), "svg", "pdf".
// Options: [WithScale] for HiDPI output.
// Returns the number of bytes written.
func (b *Built) WriteTo(ctx context.Context, w io.Writer, format string, width, height int, opts ...RenderOpt) (int64, error) {
	cfg := defaultRenderConfig()
	for _, o := range opts {
		o(&cfg)
	}

	sw, sh := int(float64(width)*cfg.scale), int(float64(height)*cfg.scale)

	switch format {
	case "svg", "pdf":
		cv := canvas.NewRecordingCanvas(sw, sh)
		if err := b.Draw(ctx, cv, sw, sh); err != nil {
			return 0, fmt.Errorf("ggplot: %w", err)
		}

		rec := cv.FinishRecording()

		switch format {
		case "svg":
			n, err := canvas.ExportSVG(rec, w)
			if err != nil {
				return n, fmt.Errorf("ggplot: %w", err)
			}

			return n, nil
		default: // pdf
			n, err := canvas.ExportPDF(rec, w)
			if err != nil {
				return n, fmt.Errorf("ggplot: %w", err)
			}

			return n, nil
		}

	case "png", "":
		cv := canvas.NewGGCanvas(sw, sh)
		if err := b.Draw(ctx, cv, sw, sh); err != nil {
			return 0, fmt.Errorf("ggplot: %w", err)
		}

		cw := &countWriter{w: w}
		if err := cv.EncodePNG(cw); err != nil {
			return cw.n, fmt.Errorf("ggplot: %w", err)
		}

		return cw.n, nil

	default:
		return 0, fmt.Errorf("ggplot: unsupported format %q (supported: png, svg, pdf): %w", format, ErrRenderFailed)
	}
}

// countWriter wraps an io.Writer and counts bytes written.
type countWriter struct {
	w io.Writer
	n int64
}

func (cw *countWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += int64(n)

	if err != nil {
		return n, fmt.Errorf("ggplot: %w", err)
	}

	return n, nil
}

// updateMappingForStat updates aesthetic mappings to point to the columns
// produced by a stat transform. Delegates to Stat.OutputMapping() so that
// third-party stats work without modifying this function.
func updateMappingForStat(s stat.Stat, mapping AesMap) AesMap {
	om := s.OutputMapping()
	if om == nil {
		return mapping // identity -- no rewriting
	}

	result := make(AesMap, len(mapping))
	maps.Copy(result, mapping)

	maps.Copy(result, om)

	return result
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
		return nil, nil, fmt.Errorf("column %q: %w", colName, err)
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
		return nil, nil, fmt.Errorf("unsupported column type %T for %q: %w", col, colName, ErrRenderFailed)
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
			return nil, nil, fmt.Errorf("select group %q: %w", label, serr)
		}

		subsets[i] = subset
	}

	return order, subsets, nil
}

// themePaletteCmap returns a discrete colormap derived from the theme's
// Palette, falling back to colormap.Tab10 when the theme carries no
// palette. Used as the default discrete color cycle when the plot has
// no explicit color scale set.
func themePaletteCmap(th theme.Theme) colormap.Cmap {
	if len(th.Palette) == 0 {
		return colormap.Tab10
	}

	colors := make([]gg.RGBA, len(th.Palette))
	for i, c := range th.Palette {
		r, g, b, a := c.RGBA()
		colors[i] = gg.RGBA{
			R: float64(r) / 65535.0,
			G: float64(g) / 65535.0,
			B: float64(b) / 65535.0,
			A: float64(a) / 65535.0,
		}
	}

	return colormap.NewListed("theme:"+th.Name, colormap.Qualitative, colors)
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
	panels    []BuiltPanel
	layout    Layout
	coord     coord.Coord
	theme     theme.Theme
	labels    Labels
	legendPos string
}

// BuiltLayer holds one resolved layer's data after stat transform and grouping.
// The Data dataset always contains the system column PANEL (int64). When the
// layer was produced by group splitting, the system column group (int64) is
// also present.
type BuiltLayer struct {
	Geom         geom.Layer
	Data         dataset.Dataset
	Mapping      AesMap
	ContColorCol string
	ContColScale *colormap.Scale
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
	XIsDiscrete   bool
}

// Layout holds the panel grid dimensions derived from faceting.
type Layout struct {
	Rows   int
	Cols   int
	Panels []PanelLayout
}

// PanelLayout holds per-panel geometry and trained scale state.
type PanelLayout struct {
	Row    int
	Col    int
	XScale scale.Scale
	YScale scale.Scale
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

// Build resolves the plot specification through the grammar pipeline and
// returns a [*Built] containing fully resolved layer data, trained scales,
// layout geometry, and theme. The result can be inspected via [Built.LayerData]
// or rendered via [Built.Draw].
//
// This is the Go equivalent of ggplot2's ggplot_build(plot).
func (p *Plot) Build(ctx context.Context) (*Built, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("ggplot: %w", err)
	}

	if len(p.spec.Layers) == 0 {
		return nil, fmt.Errorf("plot has no layers: %w", ErrRenderFailed)
	}

	// Materialise any lazy Dataset chain.
	collectedDS, collectErr := p.spec.Dataset.Collect(ctx)
	if collectErr != nil {
		return nil, fmt.Errorf("ggplot: collect dataset: %w", collectErr)
	}

	p.spec.Dataset = collectedDS

	if p.spec.Dataset.Table() == nil {
		return nil, fmt.Errorf("plot has no dataset: %w", ErrRenderFailed)
	}

	th, err := theme.Resolve(p.spec.ThemeName)
	if err != nil {
		return nil, fmt.Errorf("ggplot: %w", err)
	}

	// 1. Facet.
	facetPanels, err := p.spec.Facet.Split(ctx, p.spec.Dataset)
	if err != nil {
		return nil, fmt.Errorf("ggplot: facet split: %w", err)
	}

	rows, cols := p.spec.Facet.GridDims(len(facetPanels))
	if rows <= 0 {
		rows = 1
	}

	if cols <= 0 {
		cols = 1
	}

	// 2. Inject PANEL system column into each panel's dataset.
	eng := dataset.GetEngine(p.spec.Dataset.Table())

	for pi := range facetPanels {
		panelCol := dataset.ConstInt64Column(eng, ColPANEL, int64(pi), int(facetPanels[pi].Dataset.NumRows()))
		if panelCol != nil {
			augmented, cerr := facetPanels[pi].Dataset.WithColumn(panelCol).Collect(ctx)
			if cerr != nil {
				return nil, fmt.Errorf("ggplot: inject PANEL column: %w", cerr)
			}

			facetPanels[pi].Dataset = augmented
		}
	}

	// 3. Build each facet panel.
	builtPanels := make([]BuiltPanel, len(facetPanels))
	panelLayouts := make([]PanelLayout, len(facetPanels))

	for pi, panel := range facetPanels {
		bp, err := p.buildPanel(ctx, pi, panel.Dataset, panel.Label, th)
		if err != nil {
			return nil, err
		}

		builtPanels[pi] = bp
		panelLayouts[pi] = PanelLayout{
			Row:    pi / cols,
			Col:    pi % cols,
			XScale: bp.XScale,
			YScale: bp.YScale,
		}
	}

	return &Built{
		panels: builtPanels,
		layout: Layout{
			Rows:   rows,
			Cols:   cols,
			Panels: panelLayouts,
		},
		coord:     p.spec.Coord,
		theme:     th,
		labels:    p.spec.Labels,
		legendPos: p.spec.LegendPosition,
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

	for _, layer := range p.spec.Layers {
		layerStart := len(resolved) // track where this layer's BuiltLayers start
		merged := p.spec.GlobalMapping.Merge(layer.Mapping)
		statName := layer.Geom.StatName

		s, err := stat.Lookup(statName)
		if err != nil {
			return BuiltPanel{}, fmt.Errorf("ggplot: %w", err)
		}

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

		groupCol := merged["group"]
		if groupCol == "" {
			groupCol = colorCol // colour implies grouping (categorical only)
		}

		if groupCol != "" {
			colorScale := p.spec.ColorScales["color"]
			if colorScale == nil {
				colorScale = colormap.NewDiscrete(themePaletteCmap(th))
			}

			if col, err := ds.Column(groupCol); err == nil {
				_ = colorScale.Train(col)
			}

			groups, subsets, err := groupByColumn(ctx, ds, groupCol)
			if err != nil {
				return BuiltPanel{}, fmt.Errorf("ggplot: group split by %q: %w", groupCol, err)
			}

			if legendTitle == "" && colorCol != "" {
				legendTitle = colorCol
			}

			for gi, grpLabel := range groups {
				_ = gi
				grpDS := subsets[gi]
				grpRGBA := colorScale.At(grpLabel)

				grpMerged := make(AesMap, len(merged))
				maps.Copy(grpMerged, merged)

				if s.Name() != stat.Identity {
					statMapping := make(map[string]string, len(grpMerged))
					maps.Copy(statMapping, grpMerged)

					opts := stat.Options{
						Bins:    layer.Geom.Params.Bins,
						Method:  layer.Geom.Params.Method,
						Points:  layer.Geom.Params.Points,
						Whisker: layer.Geom.Params.Whisker,
						Notch:   layer.Geom.Params.Notch,
					}

					transformed, err := s.Compute(ctx, grpDS, statMapping, opts)
					if err != nil {
						return BuiltPanel{}, fmt.Errorf("ggplot: stat %q failed for group %q: %w",
							statName, grpLabel, err)
					}

					if transformed.Table() == nil {
						return BuiltPanel{}, fmt.Errorf("ggplot: stat %q produced nil table for group %q", //nolint:err113 // error contains dynamic context values that vary per call site.
							statName, grpLabel)
					}

					grpDS = transformed
					grpMerged = updateMappingForStat(s, grpMerged)
				}

				// Bake group color into geom params.
				grpGeom := layer.Geom
				hex := fmt.Sprintf("#%02X%02X%02X",
					uint8(grpRGBA.R*255+0.5),
					uint8(grpRGBA.G*255+0.5),
					uint8(grpRGBA.B*255+0.5))
				grpGeom.Params.Color = hex

				if grpGeom.Params.Fill == "" {
					grpGeom.Params.Fill = hex
				}

				// Inject group system column.
				grpCol := dataset.ConstInt64Column(eng, ColGroup, int64(gi), int(grpDS.NumRows()))
				if grpCol != nil {
					augmented, cerr := grpDS.WithColumn(grpCol).Collect(ctx)
					if cerr != nil {
						return BuiltPanel{}, fmt.Errorf("ggplot: inject group column: %w", cerr)
					}

					grpDS = augmented
				}

				resolved = append(resolved, BuiltLayer{
					Geom:    grpGeom,
					Data:    grpDS,
					Mapping: grpMerged,
				})

				if colorCol != "" && pi == 0 {
					alreadyHas := false

					for _, le := range legendEntries {
						if le.Label == grpLabel {
							alreadyHas = true
							break
						}
					}

					if !alreadyHas {
						legendEntries = append(legendEntries, LegendEntry{
							Label: grpLabel,
							Color: grpRGBA,
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
			if s.Name() != stat.Identity {
				statMapping := make(map[string]string, len(merged))
				maps.Copy(statMapping, merged)

				opts := stat.Options{
					Bins:    layer.Geom.Params.Bins,
					Method:  layer.Geom.Params.Method,
					Points:  layer.Geom.Params.Points,
					Whisker: layer.Geom.Params.Whisker,
					Notch:   layer.Geom.Params.Notch,
				}

				transformed, err := s.Compute(ctx, ds, statMapping, opts)
				if err != nil {
					return BuiltPanel{}, fmt.Errorf("ggplot: stat %q failed: %w", statName, err)
				}

				if transformed.Table() == nil {
					return BuiltPanel{}, fmt.Errorf("ggplot: stat %q produced nil table: %w", statName, ErrRenderFailed)
				}

				ds = transformed
				merged = updateMappingForStat(s, merged)
			}

			var contScale *colormap.Scale
			if continuousColorCol != "" {
				contScale = p.spec.ColorScales["color"]
				if contScale == nil {
					contScale = colormap.NewContinuous(colormap.Viridis, nil)
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
					Title: continuousColorCol,
					Cmap:  contScale.Cmap(),
					Norm:  contScale.Norm(),
				}
			}

			if layer.Geom.Params.Label != "" && layer.Geom.Params.Color != "" && pi == 0 {
				if c, err := colormap.Parse(layer.Geom.Params.Color); err == nil {
					legendEntries = append(legendEntries, LegendEntry{
						Label: layer.Geom.Params.Label,
						Color: c,
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

	// Train scales and apply position adjustments (after categorical mapping).
	xScale, yScale, xIsDiscrete, err := p.trainPanelScales(ctx, resolved, layerSpans)
	if err != nil {
		return BuiltPanel{}, err
	}

	return BuiltPanel{
		Label:         label,
		Layers:        resolved,
		XScale:        xScale,
		YScale:        yScale,
		LegendEntries: legendEntries,
		LegendTitle:   legendTitle,
		ColorBarSpec:  colorBarSpec,
		XIsDiscrete:   xIsDiscrete,
	}, nil
}

// layerSpan records the index range of a single geom layer within the
// resolved []BuiltLayer slice, along with its position adjustment and mapping.
// Used by trainPanelScales to apply position adjustments after categorical mapping.
type layerSpan struct {
	start, end int
	pos        position.Pos
	mapping    AesMap
}

// applyPositionAdjust applies the layer's position adjustment across groups.
// layers is the slice of BuiltLayers for a single geom layer (one per group).
// The function creates a fresh Pos instance to avoid shared state, reads X/Y
// from each layer's data, calls Adjust, and writes the adjusted values back.
func applyPositionAdjust(ctx context.Context, layers []BuiltLayer, layerPos position.Pos, mapping AesMap) error {
	if len(layers) == 0 || layerPos == nil {
		return nil
	}

	posName := position.Name(layerPos.String())
	if posName == position.NameIdentity || posName == "" {
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

	// Create a fresh position instance to avoid shared state.
	pos := position.New(posName)

	// If Fill, run the setup phase.
	if fs, ok := pos.(position.FillSetup); ok {
		fs.Setup(allXs, allYs)
	}

	// Apply position adjustment per group and write back.
	stacker, isStacker := pos.(position.Stacker)

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
			return fmt.Errorf("ggplot: position adjust group %d: %w", gi, err)
		}

		layers[gi].Data = collected

		// For dodge, narrow the bar width so groups fit side-by-side.
		if posName == position.NameDodge && nGroups > 1 {
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
		return nil, nil, false, fmt.Errorf("ggplot: %w", err)
	}

	if yOpts := p.spec.ScaleOverrides["y"].Opts; len(yOpts) > 0 {
		yScale = scale.Configure(yScale, yOpts...)
	}

	xIsDiscrete = false

	// Probe the first resolved layer's X column to decide scale type.
	for _, rl := range resolved {
		if colName, ok := rl.Mapping["x"]; ok {
			if col, err := rl.Data.Column(colName); err == nil {
				if col.DType() != dataset.DTypeFloat64 && col.DType() != dataset.DTypeInt64 {
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
		xScale, err = scale.Resolve(p.spec.ScaleOverrides["x"].Type)
		if err != nil {
			return nil, nil, false, fmt.Errorf("ggplot: %w", err)
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
		case geom.TypeBar, geom.TypeHistogram, geom.TypeArea, geom.TypeDensity:
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
				case geom.TypeBar, geom.TypeHistogram, geom.TypeBoxPlot:
					hasBars = true
				default:
				}
			}

			if hasBars {
				for _, rl := range resolved {
					n := rl.Data.NumRows()
					if n > 1 {
						halfBin := (xMax - xMin) / float64(n-1) / 2.0
						if halfBin > xPad {
							xPad = halfBin
						}
					} else if n == 1 {
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

	return xScale, yScale, xIsDiscrete, nil
}

// ---------------------------------------------------------------------------
// Draw pipeline
// ---------------------------------------------------------------------------

// Draw renders the built plot onto the given canvas at the specified dimensions.
//
// This is the Go equivalent of ggplot2's grid.draw(ggplot_gtable(built)).
func (b *Built) Draw(ctx context.Context, cv canvas.Canvas, width, height int) error { //nolint:gocognit,cyclop // Draw is a complex rendering pipeline — splitting further reduces clarity.
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("ggplot: %w", err)
	}

	th := b.theme
	cv.Clear(th.Background)

	rows := b.layout.Rows
	cols := b.layout.Cols

	// Cached layout for multi-panel consistency.
	var (
		cachedMTop, cachedMRight, cachedMBottom, cachedMLeft, cachedLegendW float64
		legendPos                                                           string
		hasLegend                                                           bool
	)

	for pi, bp := range b.panels {
		xScale := bp.XScale
		yScale := bp.YScale

		// Measure Y tick labels for left margin.
		leftAxisScale := yScale
		if allLayersHorizontal(builtLayersToLayerSpecs(bp.Layers)) {
			leftAxisScale = xScale
		}

		yTicks := leftAxisScale.Ticks(6)

		cv.SetFontSize(th.Text.TickLabel.Size)

		maxTickW := 0.0

		for _, v := range yTicks {
			tw, _ := cv.MeasureString(leftAxisScale.Format(v))
			if tw > maxTickW {
				maxTickW = tw
			}
		}

		var (
			mTop, mRight, mBottom, mLeft               float64
			panelW, panelH, cellW, cellH, dataX, dataY float64
			legendW                                    float64
		)

		if pi == 0 || len(b.panels) == 1 {
			mTop = 15.0
			mRight = 20.0
			mBottom = 15.0 + th.Text.TickLabel.Size + th.Ticks.Length + 14
			mLeft = 15.0 + maxTickW + th.Ticks.Length + 10

			if b.labels.Title != "" {
				mTop += th.Text.Title.Size + 8
			}

			if b.labels.Subtitle != "" {
				mTop += th.Text.Subtitle.Size + 4
			}

			if b.labels.X != "" {
				mBottom += th.Text.AxisTitle.Size + 8
			}

			if b.labels.Y != "" {
				titleGap := 5 + maxTickW + th.Text.AxisTitle.Size + 8
				if titleGap < 30+th.Text.AxisTitle.Size/2 {
					titleGap = 30 + th.Text.AxisTitle.Size/2
				}

				mLeft = 15.0 + titleGap + th.Ticks.Length
			}

			if b.labels.Caption != "" {
				mBottom += th.Text.TickLabel.Size + 4
			}

			legendPos = b.legendPos
			if legendPos == "" {
				legendPos = "right"
			}

			hasLegend = legendPos != "none" && (len(bp.LegendEntries) > 0 || bp.ColorBarSpec != nil)

			legendW = 0.0

			if hasLegend && (legendPos == "right" || legendPos == "left") {
				if len(bp.LegendEntries) > 0 {
					cv.SetFontSize(th.Text.Legend.Size)

					maxLabelW := 0.0

					for _, le := range bp.LegendEntries {
						tw, _ := cv.MeasureString(le.Label)
						if tw > maxLabelW {
							maxLabelW = tw
						}
					}

					legendW += maxLabelW + 12 + 8 + 15
				}

				if bp.ColorBarSpec != nil {
					cv.SetFontSize(th.Text.Legend.Size * 0.9)

					vmin, vmax := 0.0, 1.0
					if bp.ColorBarSpec.Norm != nil {
						vmin, vmax = bp.ColorBarSpec.Norm.Bounds()
					}

					maxLblW, _ := cv.MeasureString(fmt.Sprintf("%.4g", vmax))
					minLblW, _ := cv.MeasureString(fmt.Sprintf("%.4g", vmin))
					lblW := math.Max(maxLblW, minLblW)
					legendW += 14 + lblW + 12
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
		row := pi / cols
		col := pi % cols
		dataX = mLeft + float64(col)*cellW
		dataY = mTop + float64(row)*cellH

		// Facet strip label above each panel.
		stripH := 0.0
		if bp.Label != "" {
			stripH = th.Text.TickLabel.Size + 8
			sr, sg, sb, _ := th.Panel.Border.RGBA()
			cv.SetRGBA(float64(sr)/65535.0, float64(sg)/65535.0, float64(sb)/65535.0, 0.25)
			cv.DrawRectangle(dataX, dataY, cellW, stripH)
			cv.Fill()
			cv.SetColor(th.Text.AxisTitle.Color)
			cv.SetFontSize(th.Text.TickLabel.Size)
			cv.DrawStringAnchored(bp.Label, dataX+cellW/2, dataY+stripH/2, 0.5, 0.5)
		}

		dataY += stripH
		cellH -= stripH

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
		if th.Panel.BorderWidth > 0 {
			cv.SetColor(th.Panel.Border)
			cv.SetLineWidth(th.Panel.BorderWidth)
			cv.DrawRectangle(dataX, dataY, cellW, cellH)
			cv.Stroke()
		}

		// Draw data layers.
		cv.Save()
		cv.Translate(dataX, dataY)
		cv.DrawRectangle(0, 0, cellW, cellH)
		cv.Clip()

		xMin, xMax := xScale.Bounds()

		yMin, yMax := yScale.Bounds()
		for _, rl := range bp.Layers {
			drawLayer(cv, b.coord, rl.Data, rl.Geom, rl.Mapping,
				rl.ContColorCol, rl.ContColScale,
				cellW, cellH, xMin, xMax, yMin, yMax, th)
		}

		cv.Restore()

		// Draw axes.
		isMultiPanel := len(b.panels) > 1
		isBottomRow := row == rows-1
		isLeftCol := col == 0

		if !isMultiPanel || isBottomRow {
			drawXAxis(cv, renderXScale, renderXLabel, dataX, dataY+cellH, cellW, th)
		}

		if !isMultiPanel || isLeftCol {
			drawYAxis(cv, renderYScale, renderYLabel, dataX, dataY, cellH, th)
		}

		// Legend.
		if hasLegend {
			b.drawLegend(cv, bp, legendPos, dataX, dataY, cellW, cellH, mBottom, legendW, th)
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
		lx := dataX + cellW + 10

		ly := dataY + 5
		if len(bp.LegendEntries) > 0 {
			drawLegendVertical(cv, bp.LegendTitle, bp.LegendEntries, lx, ly, th)
			ly += float64(len(bp.LegendEntries)+1) * 20
		}

		if bp.ColorBarSpec != nil {
			barH := math.Min(cellH*0.25, 120)
			drawColorBar(cv, *bp.ColorBarSpec, lx, ly, barH, th)
		}
	case "left":
		lx := dataX - legendW - 5

		ly := dataY + 5
		if len(bp.LegendEntries) > 0 {
			drawLegendVertical(cv, bp.LegendTitle, bp.LegendEntries, lx, ly, th)
			ly += float64(len(bp.LegendEntries)+1) * 20
		}

		if bp.ColorBarSpec != nil {
			barH := math.Min(cellH*0.25, 120)
			drawColorBar(cv, *bp.ColorBarSpec, lx, ly, barH, th)
		}
	case "top":
		topY := dataY - 25
		if len(bp.LegendEntries) > 0 {
			drawLegendHorizontal(cv, bp.LegendTitle, bp.LegendEntries, dataX, topY, cellW, th)
		}

		if bp.ColorBarSpec != nil {
			barW := math.Min(cellW*0.3, 200)
			barX := dataX + cellW/2 - barW/2
			drawColorBarHorizontal(cv, *bp.ColorBarSpec, barX, topY, barW, th)
		}
	case "bottom":
		bottomY := dataY + cellH + mBottom - 25
		if len(bp.LegendEntries) > 0 {
			drawLegendHorizontal(cv, bp.LegendTitle, bp.LegendEntries, dataX, bottomY, cellW, th)
		}

		if bp.ColorBarSpec != nil {
			barW := math.Min(cellW*0.3, 200)
			barX := dataX + cellW/2 - barW/2
			drawColorBarHorizontal(cv, *bp.ColorBarSpec, barX, bottomY, barW, th)
		}
	}
}

// drawTitles renders title, subtitle, and caption.
func (b *Built) drawTitles(cv canvas.Canvas, width, height int, th theme.Theme) {
	centerX := float64(width) / 2
	titleY := 10.0

	if b.labels.Title != "" {
		cv.SetColor(th.Text.Title.Color)
		cv.SetFontSize(th.Text.Title.Size)
		cv.DrawStringAnchored(b.labels.Title, centerX, titleY+th.Text.Title.Size/2, 0.5, 0.5)
		titleY += th.Text.Title.Size + 8
	}

	if b.labels.Subtitle != "" {
		cv.SetColor(th.Text.Subtitle.Color)
		cv.SetFontSize(th.Text.Subtitle.Size)
		cv.DrawStringAnchored(b.labels.Subtitle, centerX, titleY+th.Text.Subtitle.Size/2, 0.5, 0.5)
	}

	if b.labels.Caption != "" {
		cv.SetColor(th.Text.TickLabel.Color)
		cv.SetFontSize(th.Text.TickLabel.Size)
		cv.DrawStringAnchored(b.labels.Caption, float64(width)-40, float64(height)-4, 1.0, 1.0)
	}
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
func drawXAxis(cv canvas.Canvas, sc scale.Scale, label string, x, y, w float64, th theme.Theme) {
	ticks := sc.Ticks(5)
	tickLen := th.Ticks.Length
	r, g, b, _ := rgbaOf(th.Ticks.Color)
	tr, tg, tb, _ := rgbaOf(th.Text.TickLabel.Color)

	// Baseline.
	cv.SetRGBA(r, g, b, 1)
	cv.SetLineWidth(th.Ticks.Width)
	cv.DrawLine(x, y, x+w, y)
	cv.Stroke()

	xMin, xMax := sc.Bounds()
	for _, v := range ticks {
		frac := (v - xMin) / (xMax - xMin)
		if frac < 0 || frac > 1 {
			continue
		}

		px := x + frac*w

		// Tick mark.
		cv.SetRGBA(r, g, b, 1)
		cv.DrawLine(px, y, px, y+tickLen)
		cv.Stroke()

		// Label.
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.Text.TickLabel.Size)
		cv.DrawStringAnchored(sc.Format(v), px, y+tickLen+12, 0.5, 0.5)
	}

	// Axis title.
	if label != "" {
		lr, lg, lb, _ := rgbaOf(th.Text.AxisTitle.Color)
		cv.SetRGBA(lr, lg, lb, 1)
		cv.SetFontSize(th.Text.AxisTitle.Size)
		cv.DrawStringAnchored(label, x+w/2, y+tickLen+28, 0.5, 0.5)
	}
}

// drawYAxis renders a vertical axis at the left of the data area.
func drawYAxis(cv canvas.Canvas, sc scale.Scale, label string, x, y, h float64, th theme.Theme) {
	ticks := sc.Ticks(5)
	tickLen := th.Ticks.Length
	r, g, b, _ := rgbaOf(th.Ticks.Color)
	tr, tg, tb, _ := rgbaOf(th.Text.TickLabel.Color)

	// Baseline.
	cv.SetRGBA(r, g, b, 1)
	cv.SetLineWidth(th.Ticks.Width)
	cv.DrawLine(x, y, x, y+h)
	cv.Stroke()

	yMin, yMax := sc.Bounds()
	minSpacing := th.Text.TickLabel.Size + 4 // minimum px between labels
	lastPy := -1000.0                        // track last drawn label position
	maxLabelW := 0.0                         // track widest drawn label

	// Measure + draw tick labels.
	cv.SetFontSize(th.Text.TickLabel.Size)

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

			tw, _ := cv.MeasureString(lbl)
			if tw > maxLabelW {
				maxLabelW = tw
			}

			cv.SetRGBA(tr, tg, tb, 1)
			cv.SetFontSize(th.Text.TickLabel.Size)
			cv.DrawStringAnchored(lbl, x-tickLen-5, py, 1.0, 0.5)
			lastPy = py
		}
	}

	// Axis title (rotated), positioned to the left of the widest tick label.
	if label != "" {
		lr, lg, lb, _ := rgbaOf(th.Text.AxisTitle.Color)
		cv.SetRGBA(lr, lg, lb, 1)
		cv.SetFontSize(th.Text.AxisTitle.Size)
		cv.Save()
		// The rotated title's horizontal extent equals its font size,
		// centred on the translate point. To avoid overlapping tick labels
		// (which extend left from x-tickLen-5 by maxLabelW) we offset by:
		//   5 (label gap) + maxLabelW + fontSize/2 + 8 (padding)
		titleOffset := 5 + maxLabelW + th.Text.AxisTitle.Size/2 + 8
		if titleOffset < 30 {
			titleOffset = 30 // minimum offset for short labels
		}

		cv.Translate(x-tickLen-titleOffset, y+h/2)
		cv.Rotate(-math.Pi / 2)
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
	cv.SetColor(th.Panel.Background)
	cv.DrawRectangle(x, y, w, h)
	cv.Fill()

	mr, mg, mb, ma := rgbaOf(th.Grid.MajorColor)
	cv.SetRGBA(mr, mg, mb, ma)
	cv.SetLineWidth(th.Grid.MajorWidth)

	// Apply dash pattern from theme.
	if len(th.Grid.DashPattern) > 0 {
		cv.SetLineDash(th.Grid.DashPattern...)
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
	if th.Grid.MinorColor == nil {
		return
	}

	mr, mg, mb, ma := rgbaOf(th.Grid.MinorColor)
	cv.SetRGBA(mr, mg, mb, ma)
	cv.SetLineWidth(th.Grid.MinorWidth)

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

// LegendEntry describes one item in the legend.
type LegendEntry struct {
	Label string
	Color gg.RGBA
}

// drawLegendVertical renders a categorical legend to the right of the data area.
func drawLegendVertical(cv canvas.Canvas, title string, entries []LegendEntry, x, y float64, th theme.Theme) {
	if len(entries) == 0 {
		return
	}

	swatchSize := 12.0
	spacing := 20.0
	curY := y

	if title != "" {
		r, g, b, _ := rgbaOf(th.Text.Legend.Color)
		cv.SetRGBA(r, g, b, 1)
		cv.SetFontSize(th.Text.Legend.Size)
		cv.DrawStringAnchored(title, x+swatchSize+5, curY, 0, 0.5)
		curY += spacing
	}

	for _, e := range entries {
		cv.SetRGBA(e.Color.R, e.Color.G, e.Color.B, e.Color.A)
		cv.DrawRectangle(x, curY-swatchSize/2, swatchSize, swatchSize)
		cv.Fill()

		r, g, b, _ := rgbaOf(th.Text.Legend.Color)
		cv.SetRGBA(r, g, b, 1)
		cv.SetFontSize(th.Text.Legend.Size)
		cv.DrawStringAnchored(e.Label, x+swatchSize+5, curY, 0, 0.5)

		curY += spacing
	}
}

// ColorBarSpec describes a continuous color bar legend.
//
// Cmap and Norm replace the previous opaque ColorFunc field: the bar walks
// Cmap.At directly across the [0,1] range, and Norm provides the data-space
// labels at the endpoints (and any future intermediate ticks).
type ColorBarSpec struct {
	Title string
	Cmap  colormap.Cmap
	Norm  colormap.Norm
}

// drawColorBar renders a continuous color bar legend at the given position.
// The bar is drawn vertically (top = max, bottom = min) as in ggplot2.
func drawColorBar(cv canvas.Canvas, spec ColorBarSpec, x, y, barH float64, th theme.Theme) {
	barW := 12.0

	// Title above the bar.
	tr, tg, tb, _ := rgbaOf(th.Text.Legend.Color)
	if spec.Title != "" {
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.Text.Legend.Size)
		cv.DrawStringAnchored(spec.Title, x+barW/2, y, 0.5, 1.0)
		y += th.Text.Legend.Size + 6
	}

	cm := spec.Cmap
	if cm == nil {
		cm = colormap.Viridis
	}

	// Draw gradient bar as thin horizontal strips (top = max, bottom = min).
	nStrips := max(int(barH), 2)

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
	cv.SetFontSize(th.Text.Legend.Size * 0.9)

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
	tr, tg, tb, _ := rgbaOf(th.Text.Legend.Color)

	if title != "" {
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.Text.Legend.Size)
		cv.DrawStringAnchored(title, curX, y, 0, 0.5)
		tw, _ := cv.MeasureString(title)
		curX += tw + gap*2
	}

	for _, e := range entries {
		if curX > x+maxW {
			break
		}

		cv.SetRGBA(e.Color.R, e.Color.G, e.Color.B, e.Color.A)
		cv.DrawRectangle(curX, y-swatchSize/2, swatchSize, swatchSize)
		cv.Fill()

		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.Text.Legend.Size * 0.9)
		cv.DrawStringAnchored(e.Label, curX+swatchSize+3, y, 0, 0.5)
		tw, _ := cv.MeasureString(e.Label)
		curX += swatchSize + 3 + tw + gap
	}
}

// drawColorBarHorizontal renders a horizontal continuous color bar legend.
func drawColorBarHorizontal(cv canvas.Canvas, spec ColorBarSpec, x, y, barW float64, th theme.Theme) {
	barH := 10.0

	tr, tg, tb, _ := rgbaOf(th.Text.Legend.Color)

	// Title to the left.
	startX := x

	if spec.Title != "" {
		cv.SetRGBA(tr, tg, tb, 1)
		cv.SetFontSize(th.Text.Legend.Size)
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
	cv.SetFontSize(th.Text.Legend.Size * 0.85)

	lo, hi := "low", "high"

	if spec.Norm != nil {
		vmin, vmax := spec.Norm.Bounds()
		lo = guideFormatNum(vmin)
		hi = guideFormatNum(vmax)
	}

	cv.DrawStringAnchored(lo, startX, y+barH+10, 0.5, 0.5)
	cv.DrawStringAnchored(hi, startX+availW, y+barH+10, 0.5, 0.5)
}

// --- Helpers ---

func rgbaOf(c color.Color) (float64, float64, float64, float64) {
	if c == nil {
		return 0, 0, 0, 1
	}

	r, g, b, a := c.RGBA()

	return float64(r) / 65535.0, float64(g) / 65535.0, float64(b) / 65535.0, float64(a) / 65535.0
}
