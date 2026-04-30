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
//	PlotSpec → Validate → Stat Transform → Scale Training → Layout → Render
//
// All data flows through the [dataset.Dataset] abstraction. Multiple engine
// backends are supported: memory (Go slices), Apache Arrow (columnar arrays),
// and BigQuery (SQL pushdown). Arrow IPC and Parquet ingest provide zero-copy
// reads; constructing from Go slices requires one copy.
package ggplot

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/colormap"
	"github.com/TuSKan/ggplot/coord"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/facet"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/guide"
	"github.com/TuSKan/ggplot/internal/canvas"
	"github.com/TuSKan/ggplot/internal/grammar"
	"github.com/TuSKan/ggplot/scale"
	"github.com/TuSKan/ggplot/stat"
	"github.com/TuSKan/ggplot/theme"
	"github.com/gogpu/gg"
)

// Plot is the immutable, declarative plot builder. Every method returns a new
// Plot with the modification applied, enabling a fluent chaining style.
//
// Plot is safe to share and reuse — modifying a derived plot does not
// affect the original.
type Plot struct {
	spec grammar.PlotSpec
}

// LegendPos controls legend placement.
type LegendPos string

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
		spec: grammar.PlotSpec{
			Dataset:        ds,
			GlobalMapping:  grammar.ToAesMap(globalAes),
			Layers:         nil,
			ScaleOverrides: make(map[string]grammar.ScaleOverride),
			Coord:          coord.Cartesian(),
			Facet:          facet.None(),
		},
	}
}

// clone creates a deep copy of the plot spec for immutability.
func (p *Plot) clone() *Plot {
	// Deep-clone layers: each LayerSpec.Mapping must be independent.
	layers := make([]grammar.LayerSpec, len(p.spec.Layers))
	for i, l := range p.spec.Layers {
		m := make(grammar.AesMap, len(l.Mapping))
		for k, v := range l.Mapping {
			m[k] = v
		}
		layers[i] = grammar.LayerSpec{Geom: l.Geom, Mapping: m}
	}

	// Deep-clone scale overrides (ScaleOverride.Params is a map).
	scales := make(map[string]grammar.ScaleOverride, len(p.spec.ScaleOverrides))
	for k, v := range p.spec.ScaleOverrides {
		params := make(map[string]string, len(v.Params))
		for pk, pv := range v.Params {
			params[pk] = pv
		}
		scales[k] = grammar.ScaleOverride{Type: v.Type, Params: params}
	}

	// ColorScales hold pointer values; clone the map but share the pointed-to
	// Scale (Scale itself is treated as user-supplied — replacing the entry
	// for an aesthetic always installs a fresh value).
	var colorScales map[string]*colormap.Scale
	if len(p.spec.ColorScales) > 0 {
		colorScales = make(map[string]*colormap.Scale, len(p.spec.ColorScales))
		for k, v := range p.spec.ColorScales {
			colorScales[k] = v
		}
	}

	return &Plot{
		spec: grammar.PlotSpec{
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
	mapping := make(grammar.AesMap)
	for k, v := range l.Mapping {
		mapping[k] = v
	}
	for _, am := range localAes {
		mapping[am.Channel] = am.Column
	}

	cloned.spec.Layers = append(cloned.spec.Layers, grammar.LayerSpec{
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
func (p *Plot) XLim(min, max float64) *Plot {
	cloned := p.clone()
	cloned.spec.XLim = [2]*float64{ptrF64(min), ptrF64(max)}
	return cloned
}

// YLim sets explicit y-axis limits. Pass math.NaN() for either end to auto-detect.
func (p *Plot) YLim(min, max float64) *Plot {
	cloned := p.clone()
	cloned.spec.YLim = [2]*float64{ptrF64(min), ptrF64(max)}
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

func ptrF64(v float64) *float64 { return &v }

// LegendPosition sets the legend placement.
func (p *Plot) LegendPosition(pos LegendPos) *Plot {
	cloned := p.clone()
	cloned.spec.LegendPosition = string(pos)
	return cloned
}

// ScaleX sets the x-axis scale type.
func (p *Plot) ScaleX(scaleType scale.Type) *Plot {
	cloned := p.clone()
	cloned.spec.ScaleOverrides["x"] = grammar.ScaleOverride{Type: scaleType}
	return cloned
}

// ScaleY sets the y-axis scale type.
func (p *Plot) ScaleY(scaleType scale.Type) *Plot {
	cloned := p.clone()
	cloned.spec.ScaleOverrides["y"] = grammar.ScaleOverride{Type: scaleType}
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
type LabOpt func(*grammar.Labels)

// Title sets the plot title.
func Title(text string) LabOpt { return func(l *grammar.Labels) { l.Title = text } }

// Subtitle sets the plot subtitle.
func Subtitle(text string) LabOpt { return func(l *grammar.Labels) { l.Subtitle = text } }

// XLab sets the x-axis label.
func XLab(text string) LabOpt { return func(l *grammar.Labels) { l.X = text } }

// YLab sets the y-axis label.
func YLab(text string) LabOpt { return func(l *grammar.Labels) { l.Y = text } }

// Caption sets the plot caption.
func Caption(text string) LabOpt { return func(l *grammar.Labels) { l.Caption = text } }

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
	cfg := defaultRenderConfig()
	for _, o := range opts {
		o(&cfg)
	}
	sw, sh := int(float64(width)*cfg.scale), int(float64(height)*cfg.scale)
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".svg", ".pdf":
		return p.saveVector(ctx, filename, ext, sw, sh)
	default:
		return p.saveRaster(ctx, filename, sw, sh)
	}
}

// saveRaster renders to PNG via GGCanvas.
func (p *Plot) saveRaster(ctx context.Context, filename string, width, height int) error {
	cv := canvas.NewGGCanvas(width, height)
	if err := p.renderTo(ctx, cv, width, height); err != nil {
		return err
	}
	return cv.SavePNG(filename)
}

// saveVector renders via RecordingCanvas and exports to SVG or PDF.
func (p *Plot) saveVector(ctx context.Context, filename, ext string, width, height int) error {
	cv := canvas.NewRecordingCanvas(width, height)
	if err := p.renderTo(ctx, cv, width, height); err != nil {
		return err
	}
	rec := cv.FinishRecording()

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	switch ext {
	case ".svg":
		_, err = canvas.ExportSVG(rec, f)
	case ".pdf":
		_, err = canvas.ExportPDF(rec, f)
	}
	return err
}

// Render produces the rendered canvas for further processing.
func (p *Plot) Render(ctx context.Context, width, height int) (*canvas.GGCanvas, error) {
	cv := canvas.NewGGCanvas(width, height)
	if err := p.renderTo(ctx, cv, width, height); err != nil {
		return nil, err
	}
	return cv, nil
}

// WriteTo renders the plot and writes the output to w in the given format.
// Supported formats: "png" (default), "svg", "pdf".
// Options: [WithScale] for HiDPI output.
// Returns the number of bytes written.
func (p *Plot) WriteTo(ctx context.Context, w io.Writer, format string, width, height int, opts ...RenderOpt) (int64, error) {
	cfg := defaultRenderConfig()
	for _, o := range opts {
		o(&cfg)
	}
	sw, sh := int(float64(width)*cfg.scale), int(float64(height)*cfg.scale)
	switch format {
	case "svg":
		return p.writeVector(ctx, w, format, sw, sh)
	case "pdf":
		return p.writeVector(ctx, w, format, sw, sh)
	case "png", "":
		cv := canvas.NewGGCanvas(sw, sh)
		if err := p.renderTo(ctx, cv, sw, sh); err != nil {
			return 0, err
		}
		cw := &countWriter{w: w}
		if err := cv.EncodePNG(cw); err != nil {
			return cw.n, err
		}
		return cw.n, nil
	default:
		return 0, fmt.Errorf("ggplot: unsupported format %q (supported: png, svg, pdf)", format)
	}
}

func (p *Plot) writeVector(ctx context.Context, w io.Writer, format string, width, height int) (int64, error) {
	cv := canvas.NewRecordingCanvas(width, height)
	if err := p.renderTo(ctx, cv, width, height); err != nil {
		return 0, err
	}
	rec := cv.FinishRecording()

	switch format {
	case "svg":
		return canvas.ExportSVG(rec, w)
	case "pdf":
		return canvas.ExportPDF(rec, w)
	default:
		return 0, fmt.Errorf("ggplot: unsupported vector format %q", format)
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
	return n, err
}

// renderTo is the core rendering pipeline orchestrator.
//
// Pipeline: Stat Transform → Scale Training → Layout → Grid → Data → Axes → Labels.
func (p *Plot) renderTo(ctx context.Context, cv canvas.Canvas, width, height int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(p.spec.Layers) == 0 {
		return fmt.Errorf("ggplot: plot has no layers")
	}
	// Materialise any lazy Dataset chain before rendering.
	collectedDS, collectErr := p.spec.Dataset.Collect(ctx)
	if collectErr != nil {
		return fmt.Errorf("ggplot: collect dataset: %w", collectErr)
	}
	p.spec.Dataset = collectedDS

	if p.spec.Dataset.Table() == nil {
		return fmt.Errorf("ggplot: plot has no dataset")
	}

	th, err := theme.Resolve(p.spec.ThemeName)
	if err != nil {
		return fmt.Errorf("ggplot: %w", err)
	}
	cv.Clear(th.Background)

	// 1. Facet.
	panels, err := p.spec.Facet.Split(ctx, p.spec.Dataset)
	if err != nil {
		return fmt.Errorf("ggplot: facet split: %w", err)
	}
	rows, cols := p.spec.Facet.GridDims(len(panels))
	if rows <= 0 {
		rows = 1
	}
	if cols <= 0 {
		cols = 1
	}

	// Cached layout for multi-panel consistency.
	var cachedMTop, cachedMRight, cachedMBottom, cachedMLeft, cachedLegendW float64
	var legendPos string
	var hasLegend bool

	// 2. For each facet panel.
	for pi, panel := range panels {
		// 2a. Stat transforms + colour/group splitting.
		resolved := make([]resolvedLayer, 0, len(p.spec.Layers)*4)
		var legendEntries []guide.LegendEntry
		legendTitle := ""
		var colorBarSpec *guide.ColorBarSpec // continuous color legend

		for _, layer := range p.spec.Layers {
			merged := p.spec.GlobalMapping.Merge(layer.Mapping)
			statName := layer.Geom.StatName
			s, err := stat.Lookup(statName)
			if err != nil {
				return fmt.Errorf("ggplot: %w", err)
			}

			ds := panel.Dataset

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
						// Numeric column → continuous color mapping, no grouping.
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
				// Resolve the discrete color scale: user override, else
				// the active theme's palette, else Tab10.
				colorScale := p.spec.ColorScales["color"]
				if colorScale == nil {
					colorScale = colormap.NewDiscrete(themePaletteCmap(th))
				}
				// Train on the group column so labels get a stable index.
				if col, err := ds.Column(groupCol); err == nil {
					_ = colorScale.Train(col)
				}

				// Split data by group, assign palette colours.
				groups, subsets, err := groupByColumn(ctx, ds, groupCol)
				if err != nil {
					return fmt.Errorf("ggplot: group split by %q: %w", groupCol, err)
				}
				if legendTitle == "" && colorCol != "" {
					legendTitle = colorCol
				}

				for gi, grpLabel := range groups {
					_ = gi // index unused now that color comes from the scale
					grpDS := subsets[gi]
					grpRGBA := colorScale.At(grpLabel)

					grpMerged := make(grammar.AesMap, len(merged))
					for k, v := range merged {
						grpMerged[k] = v
					}

					// Apply stat transform per-group.
					if s.Name() != stat.Identity {
						statMapping := make(map[string]string, len(grpMerged))
						for k, v := range grpMerged {
							statMapping[k] = v
						}
						opts := stat.Options{
							Bins:    layer.Geom.Params.Bins,
							Method:  layer.Geom.Params.Method,
							Points:  layer.Geom.Params.Points,
							Whisker: layer.Geom.Params.Whisker,
							Notch:   layer.Geom.Params.Notch,
						}
						transformed, err := s.Compute(ctx, grpDS, statMapping, opts)
						if err != nil {
							return fmt.Errorf("ggplot: stat %q failed for group %q: %w",
								statName, grpLabel, err)
						}
						if transformed.Table() == nil {
							return fmt.Errorf("ggplot: stat %q produced nil table for group %q",
								statName, grpLabel)
						}
						grpDS = transformed
						grpMerged = updateMappingForStat(s, grpMerged)
					}

					grpColorCopy := grpRGBA
					resolved = append(resolved, resolvedLayer{
						geom:       layer.Geom,
						ds:         grpDS,
						mapping:    grpMerged,
						groupColor: &grpColorCopy,
						groupLabel: grpLabel,
					})

					// Accumulate legend entries (deduplicated).
					if colorCol != "" && pi == 0 {
						alreadyHas := false
						for _, le := range legendEntries {
							if le.Label == grpLabel {
								alreadyHas = true
								break
							}
						}
						if !alreadyHas {
							legendEntries = append(legendEntries, guide.LegendEntry{
								Label: grpLabel,
								Color: grpRGBA,
							})
						}
					}
				}
			} else {
				// No grouping — single layer with optional fixed color or
				// continuous color mapping.
				if s.Name() != stat.Identity {
					statMapping := make(map[string]string, len(merged))
					for k, v := range merged {
						statMapping[k] = v
					}
					opts := stat.Options{
						Bins:    layer.Geom.Params.Bins,
						Method:  layer.Geom.Params.Method,
						Points:  layer.Geom.Params.Points,
						Whisker: layer.Geom.Params.Whisker,
						Notch:   layer.Geom.Params.Notch,
					}
					transformed, err := s.Compute(ctx, ds, statMapping, opts)
					if err != nil {
						return fmt.Errorf("ggplot: stat %q failed: %w", statName, err)
					}
					if transformed.Table() == nil {
						return fmt.Errorf("ggplot: stat %q produced nil table", statName)
					}
					ds = transformed
					merged = updateMappingForStat(s, merged)
				}

				// Resolve and train the continuous color scale (if any).
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

				resolved = append(resolved, resolvedLayer{
					geom:         layer.Geom,
					ds:           ds,
					mapping:      merged,
					contColCol:   continuousColorCol,
					contColScale: contScale,
				})

				// Build continuous color bar legend spec.
				if contScale != nil && colorBarSpec == nil && pi == 0 {
					colorBarSpec = &guide.ColorBarSpec{
						Title: continuousColorCol,
						Cmap:  contScale.Cmap(),
						Norm:  contScale.Norm(),
					}
				}

				// Collect legend entry from explicitly labeled layers.
				if layer.Geom.Params.Label != "" && layer.Geom.Params.Color != "" && pi == 0 {
					if c, err := colormap.Parse(layer.Geom.Params.Color); err == nil {
						legendEntries = append(legendEntries, guide.LegendEntry{
							Label: layer.Geom.Params.Label,
							Color: c,
						})
					}
				}
			}
		}

		// 2b. Train scale objects on stat-transformed data.
		// Detect discrete (string) X columns and build appropriate scale.
		var xScale scale.Scale
		yScale, err := scale.Resolve(p.spec.ScaleOverrides["y"].Type)
		if err != nil {
			return fmt.Errorf("ggplot: %w", err)
		}
		xIsDiscrete := false

		// Probe the first resolved layer's X column to decide scale type.
		for _, rl := range resolved {
			if colName, ok := rl.mapping["x"]; ok {
				if col, err := rl.ds.Column(colName); err == nil {
					if col.DType() != dataset.DTypeFloat64 && col.DType() != dataset.DTypeInt64 {
						// Column is not numeric — treat as discrete.
						xIsDiscrete = true
					}
				}
				break
			}
		}

		if xIsDiscrete {
			// Build discrete scale from string X column values.
			ds := scale.Discrete()
			for i, rl := range resolved {
				xColName, ok := rl.mapping["x"]
				if !ok {
					continue
				}
				col, err := rl.ds.Column(xColName)
				if err != nil {
					continue
				}
				if err := ds.Train(col); err != nil {
					continue
				}

				// Replace string column with float64 positions.
				if sc, ok2 := col.(dataset.Column[string]); ok2 {
					vals := sc.Values()
					positions := make([]float64, len(vals))
					for j, v := range vals {
						positions[j] = ds.MapCategory(v)
					}
					// Build new dataset with numeric X.
					lazyDS := dataset.ReplaceColumn(rl.ds, xColName, positions)
					if cds, cerr := lazyDS.Collect(ctx); cerr == nil {
						resolved[i].ds = cds
					}
				}
			}
			xScale = ds
		} else {
			xScale, err = scale.Resolve(p.spec.ScaleOverrides["x"].Type)
			if err != nil {
				return fmt.Errorf("ggplot: %w", err)
			}
		}

		// Train scales on (now numeric) data.
		for _, rl := range resolved {
			if !xIsDiscrete {
				if colName, ok := rl.mapping["x"]; ok {
					if col, err := rl.ds.Column(colName); err == nil {
						_ = xScale.Train(col)
					}
				}
			}
			if colName, ok := rl.mapping["y"]; ok {
				if col, err := rl.ds.Column(colName); err == nil {
					_ = yScale.Train(col)
				}
			}
			// For boxplot stat: also train Y on whisker/quartile columns.
			if rl.geom.Geom == geom.TypeBoxPlot {
				for _, extra := range []string{"lower", "q1", "middle", "q3", "upper"} {
					if col, err := rl.ds.Column(extra); err == nil {
						_ = yScale.Train(col)
					}
				}
			}
		}

		// Ensure Y starts at 0 for bar/histogram/area/density/boxplot.
		for _, rl := range resolved {
			switch rl.geom.Geom {
			case geom.TypeBar, geom.TypeHistogram, geom.TypeArea, geom.TypeDensity:
				yMin, yMax := yScale.Bounds()
				if yMin > 0 {
					if bs, ok := yScale.(scale.BoundsSetter); ok {
						bs.SetBounds(0, yMax)
					}
				}
			}
		}

		// Add padding (only for continuous scales).
		xMin, xMax := xScale.Bounds()
		yMin, yMax := yScale.Bounds()

		if !xIsDiscrete {
			xPad := (xMax - xMin) * 0.05
			yPad := (yMax - yMin) * 0.05
			if xPad == 0 {
				xPad = 0.5
			}
			if yPad == 0 {
				yPad = 0.5
			}

			// For bar/histogram/boxplot, add extra X padding so edge elements don't clip.
			hasBars := false
			for _, rl := range resolved {
				switch rl.geom.Geom {
				case geom.TypeBar, geom.TypeHistogram, geom.TypeBoxPlot:
					hasBars = true
				}
			}
			if hasBars {
				for _, rl := range resolved {
					n := rl.ds.NumRows()
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

			if bs, ok := xScale.(scale.BoundsSetter); ok {
				bs.SetBounds(xMin-xPad, xMax+xPad)
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
		} else {
			// Discrete X: use natural bounds from DiscreteScale (already padded with 0.5).
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

		// Apply user-specified axis limits (overrides auto-bounds).
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

		// 2c. Layout: measure Y tick labels for left margin.
		// When all layers are horizontal, the left axis shows the X scale ticks.
		leftAxisScale := yScale
		if allLayersHorizontal(p.spec.Layers) {
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

		// Margins — compute once for the first panel, reuse for all.
		var mTop, mRight, mBottom, mLeft float64
		var panelW, panelH, cellW, cellH, dataX, dataY float64
		var legendW float64

		if pi == 0 || len(panels) == 1 {
			mTop = 15.0
			mRight = 20.0
			mBottom = 15.0 + th.Text.TickLabel.Size + th.Ticks.Length + 14
			mLeft = 15.0 + maxTickW + th.Ticks.Length + 10

			if p.spec.Labels.Title != "" {
				mTop += th.Text.Title.Size + 8
			}
			if p.spec.Labels.Subtitle != "" {
				mTop += th.Text.Subtitle.Size + 4
			}
			if p.spec.Labels.X != "" {
				mBottom += th.Text.AxisTitle.Size + 8
			}
			if p.spec.Labels.Y != "" {
				mLeft += th.Text.AxisTitle.Size + 12
			}
			if p.spec.Labels.Caption != "" {
				mBottom += th.Text.TickLabel.Size + 4
			}

			// Reserve space for legend (respecting position).
			legendPos = p.spec.LegendPosition
			if legendPos == "" {
				legendPos = "right"
			}
			hasLegend = legendPos != "none" && (len(legendEntries) > 0 || colorBarSpec != nil)

			// Compute legend width for vertical (left/right) positions.
			legendW = 0.0
			if hasLegend && (legendPos == "right" || legendPos == "left") {
				if len(legendEntries) > 0 {
					cv.SetFontSize(th.Text.Legend.Size)
					maxLabelW := 0.0
					for _, le := range legendEntries {
						tw, _ := cv.MeasureString(le.Label)
						if tw > maxLabelW {
							maxLabelW = tw
						}
					}
					legendW += maxLabelW + 12 + 8 + 15
				}
				if colorBarSpec != nil {
					cv.SetFontSize(th.Text.Legend.Size * 0.9)
					vmin, vmax := 0.0, 1.0
					if colorBarSpec.Norm != nil {
						vmin, vmax = colorBarSpec.Norm.Bounds()
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

			// Cache for multi-panel reuse.
			cachedMTop = mTop
			cachedMRight = mRight
			cachedMBottom = mBottom
			cachedMLeft = mLeft
			cachedLegendW = legendW
		} else {
			// Reuse cached layout for consistent panel alignment.
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

		// For multi-panel facets.
		cellW = panelW / float64(cols)
		cellH = panelH / float64(rows)
		row := pi / cols
		col := pi % cols
		dataX = mLeft + float64(col)*cellW
		dataY = mTop + float64(row)*cellH

		// Facet strip label above each panel.
		stripH := 0.0
		if panel.Label != "" {
			stripH = th.Text.TickLabel.Size + 8
			// Strip background.
			sr, sg, sb, _ := th.Panel.Border.RGBA()
			cv.SetRGBA(float64(sr)/65535.0, float64(sg)/65535.0, float64(sb)/65535.0, 0.25)
			cv.DrawRectangle(dataX, dataY, cellW, stripH)
			cv.Fill()
			// Strip text.
			cv.SetColor(th.Text.AxisTitle.Color)
			cv.SetFontSize(th.Text.TickLabel.Size)
			cv.DrawStringAnchored(panel.Label, dataX+cellW/2, dataY+stripH/2, 0.5, 0.5)
		}
		dataY += stripH
		cellH -= stripH

		// 2d. When all layers are horizontal, swap scales and labels so that
		// the X axis displays what was Y and vice versa.
		renderXScale := xScale
		renderYScale := yScale
		renderXLabel := p.spec.Labels.X
		renderYLabel := p.spec.Labels.Y
		if allLayersHorizontal(p.spec.Layers) {
			renderXScale, renderYScale = yScale, xScale
			renderXLabel, renderYLabel = renderYLabel, renderXLabel
		}

		// 2d. Draw grid (fills panel background then draws grid lines).
		guide.DrawGrid(cv, renderXScale, renderYScale, dataX, dataY, cellW, cellH, th)

		// 2e. Panel border.
		if th.Panel.BorderWidth > 0 {
			cv.SetColor(th.Panel.Border)
			cv.SetLineWidth(th.Panel.BorderWidth)
			cv.DrawRectangle(dataX, dataY, cellW, cellH)
			cv.Stroke()
		}

		// 2f. Draw data layers (translate into panel, clip to panel bounds).
		cv.Save()
		cv.Translate(dataX, dataY)
		cv.DrawRectangle(0, 0, cellW, cellH)
		cv.Clip()
		xMin, xMax = xScale.Bounds()
		yMin, yMax = yScale.Bounds()
		for _, rl := range resolved {
			drawLayer(cv, p.spec.Coord, rl.ds, rl.geom, rl.mapping,
				rl.groupColor, rl.contColCol, rl.contColScale,
				cellW, cellH, xMin, xMax, yMin, yMax, th)
		}
		cv.Restore()

		// 2g. Draw axes and tick labels (in absolute coords, outside panel clip).
		// For multi-panel facets: X axis only on bottom row, Y axis only on left col.
		isMultiPanel := len(panels) > 1
		isBottomRow := row == rows-1
		isLeftCol := col == 0

		if !isMultiPanel || isBottomRow {
			guide.DrawXAxis(cv, renderXScale, renderXLabel, dataX, dataY+cellH, cellW, th)
		}
		if !isMultiPanel || isLeftCol {
			guide.DrawYAxis(cv, renderYScale, renderYLabel, dataX, dataY, cellH, th)
		}

		// 2h. Legend.
		if hasLegend {
			switch legendPos {
			case "right":
				lx := dataX + cellW + 10
				ly := dataY + 5
				if len(legendEntries) > 0 {
					guide.DrawLegend(cv, legendTitle, legendEntries, lx, ly, th)
					ly += float64(len(legendEntries)+1) * 20
				}
				if colorBarSpec != nil {
					barH := math.Min(cellH*0.25, 120)
					guide.DrawColorBar(cv, *colorBarSpec, lx, ly, barH, th)
				}
			case "left":
				lx := dataX - legendW - 5
				ly := dataY + 5
				if len(legendEntries) > 0 {
					guide.DrawLegend(cv, legendTitle, legendEntries, lx, ly, th)
					ly += float64(len(legendEntries)+1) * 20
				}
				if colorBarSpec != nil {
					barH := math.Min(cellH*0.25, 120)
					guide.DrawColorBar(cv, *colorBarSpec, lx, ly, barH, th)
				}
			case "top":
				topY := dataY - 25
				if len(legendEntries) > 0 {
					guide.DrawLegendHorizontal(cv, legendTitle, legendEntries, dataX, topY, cellW, th)
				}
				if colorBarSpec != nil {
					barW := math.Min(cellW*0.3, 200)
					barX := dataX + cellW/2 - barW/2
					guide.DrawColorBarHorizontal(cv, *colorBarSpec, barX, topY, barW, th)
				}
			case "bottom":
				bottomY := dataY + cellH + mBottom - 25
				if len(legendEntries) > 0 {
					guide.DrawLegendHorizontal(cv, legendTitle, legendEntries, dataX, bottomY, cellW, th)
				}
				if colorBarSpec != nil {
					barW := math.Min(cellW*0.3, 200)
					barX := dataX + cellW/2 - barW/2
					guide.DrawColorBarHorizontal(cv, *colorBarSpec, barX, bottomY, barW, th)
				}
			}
		}
	}

	// 3. Title, subtitle, caption — drawn ONCE outside the panel loop.
	centerX := float64(width) / 2
	titleY := 10.0
	if p.spec.Labels.Title != "" {
		cv.SetColor(th.Text.Title.Color)
		cv.SetFontSize(th.Text.Title.Size)
		cv.DrawStringAnchored(p.spec.Labels.Title, centerX, titleY+th.Text.Title.Size/2, 0.5, 0.5)
		titleY += th.Text.Title.Size + 8
	}
	if p.spec.Labels.Subtitle != "" {
		cv.SetColor(th.Text.Subtitle.Color)
		cv.SetFontSize(th.Text.Subtitle.Size)
		cv.DrawStringAnchored(p.spec.Labels.Subtitle, centerX, titleY+th.Text.Subtitle.Size/2, 0.5, 0.5)
	}
	if p.spec.Labels.Caption != "" {
		cv.SetColor(th.Text.TickLabel.Color)
		cv.SetFontSize(th.Text.TickLabel.Size)
		cv.DrawStringAnchored(p.spec.Labels.Caption, float64(width)-40, float64(height)-4, 1.0, 1.0)
	}

	return nil
}

// resolvedLayer is a layer after stat transformation has been applied.
type resolvedLayer struct {
	geom         geom.Layer
	ds           dataset.Dataset
	mapping      grammar.AesMap
	groupColor   *gg.RGBA        // nil = use Params.Color; non-nil = assigned by colour aesthetic
	groupLabel   string          // label for this group (used by legend)
	contColCol   string          // non-empty = column name for continuous color mapping
	contColScale *colormap.Scale // resolved continuous color scale (nil = no continuous color)
}

// updateMappingForStat updates aesthetic mappings to point to the columns
// produced by a stat transform. Delegates to Stat.OutputMapping() so that
// third-party stats work without modifying this function.
func updateMappingForStat(s stat.Stat, mapping grammar.AesMap) grammar.AesMap {
	om := s.OutputMapping()
	if om == nil {
		return mapping // identity — no rewriting
	}
	result := make(grammar.AesMap, len(mapping))
	for k, v := range mapping {
		result[k] = v
	}
	for aes, col := range om {
		result[aes] = col
	}
	return result
}

// groupByColumn splits a Dataset into subsets by the distinct values in the
// given column. Returns ordered unique labels, corresponding filtered datasets,
// and any error encountered.
//
// Performance: uses strconv (not fmt.Sprintf) for numeric→string conversion
// and SelectRows (not BoolMask) for O(group_size) extraction without
// allocating O(n) bool masks per group.
func groupByColumn(_ context.Context, ds dataset.Dataset, colName string) ([]string, []dataset.Dataset, error) {
	col, err := ds.Column(colName)
	if err != nil {
		return nil, nil, fmt.Errorf("column %q: %w", colName, err)
	}

	// Extract string labels from the column.
	// Uses strconv instead of fmt.Sprintf (~5× faster per value).
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
		return nil, nil, fmt.Errorf("unsupported column type %T for %q", col, colName)
	}

	// Build index groups: map[label] → []rowIndex.
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
func allLayersHorizontal(layers []grammar.LayerSpec) bool {
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
