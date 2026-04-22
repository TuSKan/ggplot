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
// All data flows through the [dataset.Dataset] abstraction backed by Apache Arrow
// for zero-copy performance, with lazy evaluation for ETL operations.
package ggplot

import (
	"fmt"
	"image/color"
	"math"

	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/coord"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/facet"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/guide"
	"github.com/TuSKan/ggplot/internal/canvas"
	icolor "github.com/TuSKan/ggplot/internal/color"
	"github.com/TuSKan/ggplot/internal/grammar"
	"github.com/TuSKan/ggplot/scale"
	"github.com/TuSKan/ggplot/stat"
	"github.com/TuSKan/ggplot/theme"
)

// Plot is the immutable, declarative plot builder. Every method returns a new
// Plot with the modification applied, enabling a fluent chaining style.
//
// Plot is safe to share and reuse — modifying a derived plot does not
// affect the original.
type Plot struct {
	spec grammar.PlotSpec
}

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
	layers := make([]grammar.LayerSpec, len(p.spec.Layers))
	copy(layers, p.spec.Layers)

	scales := make(map[string]grammar.ScaleOverride, len(p.spec.ScaleOverrides))
	for k, v := range p.spec.ScaleOverrides {
		scales[k] = v
	}

	return &Plot{
		spec: grammar.PlotSpec{
			Dataset:        p.spec.Dataset,
			GlobalMapping:  p.spec.GlobalMapping.Merge(nil),
			Layers:         layers,
			ScaleOverrides: scales,
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

// Theme sets the theme by name.
func (p *Plot) Theme(name string) *Plot {
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

// CoordFlip swaps the x and y axes.
func (p *Plot) CoordFlip() *Plot {
	cloned := p.clone()
	cloned.spec.Coord = coord.Flip()
	return cloned
}

func ptrF64(v float64) *float64 { return &v }

// LegendPosition sets the legend placement.
// Valid values: "right" (default), "left", "top", "bottom", "none".
func (p *Plot) LegendPosition(pos string) *Plot {
	cloned := p.clone()
	cloned.spec.LegendPosition = pos
	return cloned
}

// ScaleX sets the x-axis scale type. Valid values: "log10", "sqrt", "reverse".
func (p *Plot) ScaleX(scaleType string) *Plot {
	cloned := p.clone()
	cloned.spec.ScaleOverrides["x"] = grammar.ScaleOverride{Type: scaleType}
	return cloned
}

// ScaleY sets the y-axis scale type. Valid values: "log10", "sqrt", "reverse".
func (p *Plot) ScaleY(scaleType string) *Plot {
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

// Save renders the plot to a PNG file at the given dimensions.
func (p *Plot) Save(filename string, width, height int) error {
	cv := canvas.NewGGCanvas(width, height)

	if err := p.renderTo(cv, width, height); err != nil {
		return err
	}

	return cv.SavePNG(filename)
}

// Render produces the rendered canvas for further processing.
// This is the low-level entry point; prefer [Save] or Show for common cases.
func (p *Plot) Render(width, height int) (*canvas.GGCanvas, error) {
	cv := canvas.NewGGCanvas(width, height)
	if err := p.renderTo(cv, width, height); err != nil {
		return nil, err
	}
	return cv, nil
}

// renderTo is the core rendering pipeline orchestrator.
//
// Pipeline: Stat Transform → Scale Training → Layout → Grid → Data → Axes → Labels.
func (p *Plot) renderTo(cv canvas.Canvas, width, height int) error {
	if len(p.spec.Layers) == 0 {
		return fmt.Errorf("ggplot: plot has no layers")
	}
	if p.spec.Dataset == nil {
		return fmt.Errorf("ggplot: plot has no dataset")
	}

	th := theme.Resolve(p.spec.ThemeName)
	cv.Clear(th.Background)

	// 1. Facet.
	panels, err := p.spec.Facet.Split(p.spec.Dataset)
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
			s := stat.Lookup(statName)

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
					if _, fErr := dataset.Min(col); fErr == nil {
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
				// Split data by group, assign palette colours.
				groups, subsets := dataset.GroupBy(ds, groupCol)
				if legendTitle == "" && colorCol != "" {
					legendTitle = colorCol
				}

				for gi, grpLabel := range groups {
					grpDS := subsets[gi]
					grpColor := icolor.Category10(gi)

					grpMerged := make(grammar.AesMap, len(merged))
					for k, v := range merged {
						grpMerged[k] = v
					}

					// Apply stat transform per-group.
					if s.Name() != "identity" {
						statMapping := make(map[string]string, len(grpMerged)+2)
						for k, v := range grpMerged {
							statMapping[k] = v
						}
						if layer.Geom.Params.Bins > 0 {
							statMapping["__bins"] = fmt.Sprintf("%d", layer.Geom.Params.Bins)
						}
						transformed, err := s.Compute(grpDS, statMapping)
						if err == nil && transformed != nil {
							grpDS = transformed
							grpMerged = updateMappingForStat(statName, grpMerged)
						}
					}

					resolved = append(resolved, resolvedLayer{
						geom:       layer.Geom,
						ds:         grpDS,
						mapping:    grpMerged,
						groupColor: grpColor,
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
								Color: grpColor,
							})
						}
					}
				}
			} else {
				// No grouping — single layer with optional fixed color.
				if s.Name() != "identity" {
					statMapping := make(map[string]string, len(merged)+2)
					for k, v := range merged {
						statMapping[k] = v
					}
					if layer.Geom.Params.Bins > 0 {
						statMapping["__bins"] = fmt.Sprintf("%d", layer.Geom.Params.Bins)
					}
					transformed, err := s.Compute(ds, statMapping)
					if err == nil && transformed != nil {
						ds = transformed
						merged = updateMappingForStat(statName, merged)
					}
				}
				resolved = append(resolved, resolvedLayer{geom: layer.Geom, ds: ds, mapping: merged, continuousColor: continuousColorCol})

				// Build continuous color bar legend spec.
				if continuousColorCol != "" && colorBarSpec == nil && pi == 0 {
					if col, err := ds.Column(continuousColorCol); err == nil {
						zMin, _ := dataset.Min(col)
						zMax, _ := dataset.Max(col)
						colorBarSpec = &guide.ColorBarSpec{
							Title: continuousColorCol,
							Min:   zMin,
							Max:   zMax,
						}
					}
				}

				// Collect legend entry from explicitly labeled layers.
				if layer.Geom.Params.Label != "" && layer.Geom.Params.Color != "" && pi == 0 {
					c := theme.ParseHexColor(layer.Geom.Params.Color)
					legendEntries = append(legendEntries, guide.LegendEntry{
						Label: layer.Geom.Params.Label,
						Color: c,
					})
				}
			}
		}

		// 2b. Train scale objects on stat-transformed data.
		// Detect discrete (string) X columns and build appropriate scale.
		var xScale scale.Scale
		yScale := scale.Resolve(p.spec.ScaleOverrides["y"].Type)
		xIsDiscrete := false

		// Probe the first resolved layer's X column to decide scale type.
		for _, rl := range resolved {
			if colName, ok := rl.mapping["x"]; ok {
				if col, err := rl.ds.Column(colName); err == nil {
					if _, fErr := dataset.Min(col); fErr != nil {
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
				if ic, ok2 := col.(dataset.IterableColumn); ok2 {
					si, err := ic.Strings()
					if err == nil {
						var positions []float64
						for {
							v, _, ok := si.Next()
							if !ok {
								break
							}
							positions = append(positions, ds.MapCategory(v))
						}
						// Build new dataset with numeric X.
						newDS, err := dataset.ReplaceColumn(rl.ds, xColName, positions)
						if err == nil {
							resolved[i].ds = newDS
						}
					}
				}
			}
			xScale = ds
		} else {
			xScale = scale.Resolve(p.spec.ScaleOverrides["x"].Type)
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
		// When flipped, the left axis shows the X scale ticks.
		leftAxisScale := yScale
		if p.spec.Coord.IsFlipped() {
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
					maxLblW, _ := cv.MeasureString(fmt.Sprintf("%.4g", colorBarSpec.Max))
					minLblW, _ := cv.MeasureString(fmt.Sprintf("%.4g", colorBarSpec.Min))
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
			setColorFromTheme(cv, th.Text.AxisTitle.Color)
			cv.SetFontSize(th.Text.TickLabel.Size)
			cv.DrawStringAnchored(panel.Label, dataX+cellW/2, dataY+stripH/2, 0.5, 0.5)
		}
		dataY += stripH
		cellH -= stripH

		// 2d. When the coord is flipped, swap scales and labels so that
		// the X axis displays what was Y and vice versa. The coord.Transform
		// handles the pixel mapping; we handle the scale/label swap.
		renderXScale := xScale
		renderYScale := yScale
		renderXLabel := p.spec.Labels.X
		renderYLabel := p.spec.Labels.Y
		if p.spec.Coord.IsFlipped() {
			renderXScale, renderYScale = yScale, xScale
			renderXLabel, renderYLabel = renderYLabel, renderXLabel
		}

		// 2d. Draw grid (in absolute coords).
		guide.DrawGrid(cv, renderXScale, renderYScale, dataX, dataY, cellW, cellH, th)

		// 2e. Panel background and border.
		setColorFromTheme(cv, th.Panel.Background)
		// (already drawn by grid background — skip duplicate fill)

		if th.Panel.BorderWidth > 0 {
			setColorFromTheme(cv, th.Panel.Border)
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
			drawLayer(cv, p.spec.Coord, rl.ds, rl.geom, rl.mapping, rl.groupColor, rl.continuousColor, cellW, cellH, xMin, xMax, yMin, yMax)
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
		setColorFromTheme(cv, th.Text.Title.Color)
		cv.SetFontSize(th.Text.Title.Size)
		cv.DrawStringAnchored(p.spec.Labels.Title, centerX, titleY+th.Text.Title.Size/2, 0.5, 0.5)
		titleY += th.Text.Title.Size + 8
	}
	if p.spec.Labels.Subtitle != "" {
		setColorFromTheme(cv, th.Text.Subtitle.Color)
		cv.SetFontSize(th.Text.Subtitle.Size)
		cv.DrawStringAnchored(p.spec.Labels.Subtitle, centerX, titleY+th.Text.Subtitle.Size/2, 0.5, 0.5)
	}
	if p.spec.Labels.Caption != "" {
		setColorFromTheme(cv, th.Text.TickLabel.Color)
		cv.SetFontSize(th.Text.TickLabel.Size)
		cv.DrawStringAnchored(p.spec.Labels.Caption, float64(width)-40, float64(height)-4, 1.0, 1.0)
	}

	return nil
}

// resolvedLayer is a layer after stat transformation has been applied.
type resolvedLayer struct {
	geom            geom.Layer
	ds              dataset.Dataset
	mapping         grammar.AesMap
	groupColor      color.Color // nil = use Params.Color; non-nil = assigned by colour aesthetic
	groupLabel      string      // label for this group (used by legend)
	continuousColor string      // non-empty = column name for continuous color mapping
}

// updateMappingForStat updates aesthetic mappings to point to the columns
// produced by a stat transform. Each stat produces specific output columns.
func updateMappingForStat(statName string, mapping grammar.AesMap) grammar.AesMap {
	result := make(grammar.AesMap, len(mapping))
	for k, v := range mapping {
		result[k] = v
	}
	switch statName {
	case "bin", "count":
		// Stat bin/count produces "x" and "count" columns.
		result["x"] = "x"
		result["y"] = "count"
	case "density":
		// Stat density produces "x" and "density" columns.
		result["x"] = "x"
		result["y"] = "density"
	case "smooth":
		// Stat smooth produces "x" and "y" columns.
		result["x"] = "x"
		result["y"] = "y"
	case "summary":
		// Stat summary produces "x" and "y" columns.
		result["x"] = "x"
		result["y"] = "y"
	case "boxplot":
		// Stat boxplot produces x, lower, q1, middle, q3, upper columns.
		// The drawBoxplot function reads these directly by name.
		result["x"] = "x"
		result["y"] = "middle"
	}
	return result
}
