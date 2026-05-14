// Package geom provides geometry specifications for the Grammar of Graphics.
// Geometries define how data is visually represented — as points, lines, bars,
// areas, etc. Each geom is a pure declarative specification; it holds no
// rendering logic. The rendering pipeline reads the spec and dispatches
// drawing operations to the [canvas.Canvas] backend.
//
// Every geom constructor returns a [Layer] that can be added to a plot:
//
//	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
//	    Layer(geom.Point()).
//	    Layer(geom.Line())
package geom

import (
	"fmt"
	"strings"

	"github.com/TuSKan/ggplot/position"
	"github.com/TuSKan/ggplot/stat"
)

// Layer represents a declarative layer specification produced by a geom
// constructor. It carries the geometry type, optional stat/position overrides,
// per-layer aesthetic mappings, and visual parameters.
type Layer struct {
	Geom     Type              // geometry type (point, line, bar, etc.)
	StatName stat.Name         // stat name (stat.Identity, stat.Bin, stat.Count, etc.)
	Position position.Pos      // position adjustment
	Params   Params            // visual parameters specific to this geometry
	Mapping  map[string]string // per-layer aesthetic overrides (channel → column)

	// setFlags tracks which options were explicitly set by the user.
	// Used by Validate() to emit warnings for irrelevant options.
	setFlags OptFlag
	warnings []string // validation warnings from applyOpts
}

// Warnings returns any validation warnings generated during construction.
func (l *Layer) Warnings() []string { return l.warnings }

// Type identifies the kind of geometry.
type Type string

// TypePoint identifies a scatter point geometry.
const (
	TypePoint     Type = "point"
	TypeLine      Type = "line"
	TypeBar       Type = "bar"
	TypeHistogram Type = "histogram"
	TypeArea      Type = "area"
	TypePolygon   Type = "polygon"
	TypeSmooth    Type = "smooth"
	TypeText      Type = "text"
	TypeBoxPlot   Type = "boxplot"
	TypeErrorBar  Type = "errorbar"
	TypeDensity   Type = "density"
	TypeTile      Type = "tile"
	TypeRug       Type = "rug"
	TypeSegment   Type = "segment"
	TypeStep      Type = "step"
	TypeHLine     Type = "hline"
	TypeVLine     Type = "vline"
	TypeABLine    Type = "abline"
)

// Orientation controls which axis a directional geom extends along.
type Orientation string

// Vertical is the default orientation: bars grow upward.
const (
	Vertical   Orientation = "v" // default: bars grow upward, boxplots are vertical
	Horizontal Orientation = "h" // bars grow rightward, boxplots are horizontal
)

// Params holds visual parameters for geometries. Not all fields apply to
// every geometry type; unused fields are ignored during rendering but
// [Layer.Validate] will emit warnings for irrelevant options.
type Params struct {
	// Common
	Color     string  // hex color override (e.g., "#4C72B0")
	Fill      string  // hex fill color override
	Alpha     float64 // opacity [0, 1]
	LineWidth float64 // stroke width in pixels

	// Point-specific
	Size  float64 // point radius in pixels (default = 3)
	Shape string  // "circle", "square", "triangle", "diamond" (default = "circle")

	// Bar/Histogram-specific
	Width float64 // relative bar width [0, 1] (default = 0.8)
	Gap   float64 // gap between bars [0, 1] (0 = touching, 1 = invisible; default = 0.2)
	Bins  int     // number of bins (histogram, default = 30)

	// Text-specific
	FontSize   float64 // text font size in points
	FontFamily string  // font family name
	Angle      float64 // rotation angle in degrees

	// Smooth-specific
	Method string  // "lm", "loess"
	Span   float64 // loess span
	Points int     // number of interpolation points

	// Boxplot-specific
	Whisker string // whisker rule: "tukey" (default, 1.5×IQR), "range" (min-max)
	Notch   bool   // if true, compute notch confidence interval around median

	// Orientation
	Orientation Orientation // "v" (default) or "h" — controls axis extension direction

	// Legend
	Label string // legend label for this layer (used with manual colors)

	// Reference lines
	Intercept float64 // y-intercept (hline) or x-intercept (vline)
	Slope     float64 // slope for abline (y = slope*x + intercept)
}

// --- Option tracking for validation ---

// OptFlag tracks which parameters were explicitly set by the user.
// Exported so third-party packages can compose relevance masks for
// [RegisterGeomType].
type OptFlag uint32

// OptColor tracks whether WithColor was set.
const (
	OptColor       OptFlag = 1 << iota // common
	OptFill                            // common
	OptAlpha                           // common
	OptLineWidth                       // common
	OptSize                            // point, (also sets LineWidth)
	OptShape                           // point
	OptWidth                           // bar, histogram
	OptBins                            // histogram
	OptFontSize                        // text
	OptFontFamily                      // text
	OptAngle                           // text
	OptMethod                          // smooth
	OptSpan                            // smooth
	OptPoints                          // smooth, density
	OptWhisker                         // boxplot
	OptNotch                           // boxplot
	OptOrientation                     // bar, histogram, boxplot, area, density, rug
)

// paramRelevance maps geometry types to what parameters are meaningful for them.
var paramRelevance = map[Type]OptFlag{
	TypePoint:     OptColor | OptFill | OptAlpha | OptSize | OptShape,
	TypeLine:      OptColor | OptAlpha | OptLineWidth | OptSize, // Size → LineWidth
	TypeBar:       OptColor | OptFill | OptAlpha | OptWidth | OptLineWidth | OptOrientation,
	TypeHistogram: OptColor | OptFill | OptAlpha | OptWidth | OptBins | OptLineWidth | OptOrientation,
	TypeArea:      OptColor | OptFill | OptAlpha | OptLineWidth | OptOrientation,
	TypePolygon:   OptColor | OptFill | OptAlpha | OptLineWidth,
	TypeSmooth:    OptColor | OptAlpha | OptLineWidth | OptSize | OptMethod | OptSpan | OptPoints,
	TypeDensity:   OptColor | OptFill | OptAlpha | OptLineWidth | OptPoints | OptOrientation,
	TypeText:      OptColor | OptAlpha | OptFontSize | OptFontFamily | OptAngle,
	TypeStep:      OptColor | OptAlpha | OptLineWidth | OptSize,
	TypeRug:       OptColor | OptAlpha | OptLineWidth | OptOrientation,
	TypeBoxPlot:   OptColor | OptFill | OptAlpha | OptWidth | OptLineWidth | OptWhisker | OptNotch | OptOrientation,
	TypeErrorBar:  OptColor | OptAlpha | OptLineWidth | OptWidth,
	TypeSegment:   OptColor | OptAlpha | OptLineWidth,
	TypeTile:      OptColor | OptFill | OptAlpha,
	TypeHLine:     OptColor | OptAlpha | OptLineWidth,
	TypeVLine:     OptColor | OptAlpha | OptLineWidth,
	TypeABLine:    OptColor | OptAlpha | OptLineWidth,
}

// RegisterGeomType registers a custom geometry type with its relevant option
// flags. This makes geom.Type open: third-party packages can define new
// geometry types that participate in option validation via [Layer.Validate].
//
// Combined with [ggplot.RegisterDrawer], this enables fully custom geoms:
//
//	// In your package init():
//	const TypeViolin geom.Type = "violin"
//	geom.RegisterGeomType(TypeViolin, geom.OptColor|geom.OptFill|geom.OptAlpha|geom.OptWidth)
//	ggplot.RegisterDrawer(TypeViolin, myViolinDrawer)
func RegisterGeomType(t Type, relevantOpts OptFlag) {
	paramRelevance[t] = relevantOpts
}

// optName maps flags to human-readable names.
var optName = map[OptFlag]string{
	OptColor:       "WithColor",
	OptFill:        "WithFill",
	OptAlpha:       "WithAlpha",
	OptLineWidth:   "WithLineWidth",
	OptSize:        "WithSize",
	OptShape:       "WithShape",
	OptWidth:       "WithWidth",
	OptBins:        "WithBins",
	OptFontSize:    "WithFontSize",
	OptFontFamily:  "WithFontFamily",
	OptAngle:       "WithAngle",
	OptMethod:      "WithMethod",
	OptSpan:        "WithSpan",
	OptPoints:      "WithPoints",
	OptOrientation: "WithOrientation",
}

// Validate checks if the configured params are meaningful for this geometry
// type and returns a list of warning messages for irrelevant options.
//
// Example:
//
//	layer := geom.Point(geom.WithBins(30))  // bins are for histograms, not points
//	warnings := layer.Validate()
//	// warnings = ["geom_point: WithBins has no effect (relevant for: histogram)"]
func (l *Layer) Validate() []string {
	if l.setFlags == 0 {
		return nil
	}

	relevant, ok := paramRelevance[l.Geom]
	if !ok {
		return nil // unknown geom type, skip validation
	}

	irrelevant := l.setFlags &^ relevant
	if irrelevant == 0 {
		return nil
	}

	var warnings []string

	for flag := OptFlag(1); flag <= OptPoints; flag <<= 1 {
		if irrelevant&flag == 0 {
			continue
		}

		name := optName[flag]

		var relevantGeoms []string

		for geomType, mask := range paramRelevance {
			if mask&flag != 0 {
				relevantGeoms = append(relevantGeoms, string(geomType))
			}
		}

		warnings = append(warnings, fmt.Sprintf(
			"geom_%s: %s has no effect (relevant for: %s)",
			l.Geom, name, strings.Join(relevantGeoms, ", "),
		))
	}

	return warnings
}

// --- Geometry constructors ---

// Opt is a functional option for configuring geometry layers.
type Opt func(*Layer)

// WithStat sets the statistical transform for this layer.
func WithStat(name stat.Name) Opt { return func(l *Layer) { l.StatName = name } }

// WithPosition sets the position adjustment for this layer.
func WithPosition(p position.Pos) Opt { return func(l *Layer) { l.Position = p } }

// WithColor sets a fixed color override.
func WithColor(hex string) Opt {
	return func(l *Layer) { l.Params.Color = hex; l.setFlags |= OptColor }
}

// WithFill sets a fixed fill color override.
func WithFill(hex string) Opt {
	return func(l *Layer) { l.Params.Fill = hex; l.setFlags |= OptFill }
}

// WithAlpha sets the opacity.
func WithAlpha(a float64) Opt {
	return func(l *Layer) { l.Params.Alpha = a; l.setFlags |= OptAlpha }
}

// WithSize sets the point radius. Use [WithLineWidth] for stroke width.
func WithSize(s float64) Opt {
	return func(l *Layer) {
		l.Params.Size = s
		l.setFlags |= OptSize
	}
}

// WithLineWidth sets the stroke width.
func WithLineWidth(w float64) Opt {
	return func(l *Layer) { l.Params.LineWidth = w; l.setFlags |= OptLineWidth }
}

// WithShape sets the point shape.
func WithShape(shape string) Opt {
	return func(l *Layer) { l.Params.Shape = shape; l.setFlags |= OptShape }
}

// WithWidth sets the relative bar width [0, 1].
func WithWidth(w float64) Opt {
	return func(l *Layer) { l.Params.Width = w; l.setFlags |= OptWidth }
}

// WithGap sets the gap between bars as a fraction [0, 1].
// 0 means bars touch (no gap), 1 means 100% gap (bars invisible).
// The bar width is derived as 1 - gap. Default gap is 0.2.
func WithGap(g float64) Opt {
	return func(l *Layer) {
		if g < 0 {
			g = 0
		}

		if g > 1 {
			g = 1
		}

		l.Params.Gap = g
		l.Params.Width = 1 - g
		l.setFlags |= OptWidth
	}
}

// WithBins sets the number of bins for histograms.
func WithBins(n int) Opt {
	return func(l *Layer) { l.Params.Bins = n; l.setFlags |= OptBins }
}

// WithMethod sets the smoothing method.
func WithMethod(m string) Opt {
	return func(l *Layer) { l.Params.Method = m; l.setFlags |= OptMethod }
}

// WithSpan sets the loess smoothing span.
func WithSpan(s float64) Opt {
	return func(l *Layer) { l.Params.Span = s; l.setFlags |= OptSpan }
}

// WithPoints sets the interpolation point count.
func WithPoints(n int) Opt {
	return func(l *Layer) { l.Params.Points = n; l.setFlags |= OptPoints }
}

// WithFontSize sets the text font size.
func WithFontSize(size float64) Opt {
	return func(l *Layer) { l.Params.FontSize = size; l.setFlags |= OptFontSize }
}

// WithFontFamily sets the text font family.
func WithFontFamily(family string) Opt {
	return func(l *Layer) { l.Params.FontFamily = family; l.setFlags |= OptFontFamily }
}

// WithAngle sets the text rotation angle in degrees.
func WithAngle(deg float64) Opt {
	return func(l *Layer) { l.Params.Angle = deg; l.setFlags |= OptAngle }
}

// WithWhisker sets the boxplot whisker rule: "tukey" (1.5×IQR, default) or "range" (min-max).
func WithWhisker(rule string) Opt {
	return func(l *Layer) { l.Params.Whisker = rule; l.setFlags |= OptWhisker }
}

// WithNotch enables notched boxplots that show the 95% confidence interval around the median.
func WithNotch(enabled bool) Opt {
	return func(l *Layer) { l.Params.Notch = enabled; l.setFlags |= OptNotch }
}

// WithOrientation sets the axis extension direction for directional geoms.
// [Horizontal] makes bars grow rightward, boxplots lay sideways, etc.
func WithOrientation(o Orientation) Opt {
	return func(l *Layer) { l.Params.Orientation = o; l.setFlags |= OptOrientation }
}

// WithLabel sets a legend label for this layer. When used together with
// [WithColor], a legend entry is generated even without aes.Color grouping.
// This is the idiomatic way to add legends in wide-format multi-layer plots.
func WithLabel(name string) Opt {
	return func(l *Layer) { l.Params.Label = name }
}

// WithIntercept sets the intercept value for reference lines.
// For [HLine], this is the Y value. For [VLine], this is the X value.
func WithIntercept(v float64) Opt {
	return func(l *Layer) { l.Params.Intercept = v }
}

// applyOpts applies options and stores validation warnings on the layer.
func applyOpts(l *Layer, opts []Opt) {
	for _, o := range opts {
		o(l)
	}

	l.warnings = l.Validate()
}

// Point creates a scatter point geometry layer.
// Default stat: identity. Default position: identity.
//
// Relevant options: WithColor, WithFill, WithAlpha, WithSize, WithShape.
func Point(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypePoint,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{Size: 3, Alpha: 1.0, Shape: "circle"},
	}
	applyOpts(&l, opts)

	return l
}

// Line creates a connected line geometry layer.
//
// Relevant options: WithColor, WithAlpha, WithLineWidth, WithSize.
func Line(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeLine,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{LineWidth: 2, Alpha: 1.0},
	}
	applyOpts(&l, opts)

	return l
}

// Bar creates a bar chart geometry layer.
// Default stat: count. Default position: stack.
//
// Relevant options: WithColor, WithFill, WithAlpha, WithWidth, WithLineWidth.
func Bar(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeBar,
		StatName: stat.Count,
		Position: position.Stack(),
		Params:   Params{Width: 0.8, Alpha: 0.8},
	}
	applyOpts(&l, opts)

	return l
}

// Col creates a column geometry layer that uses raw Y values (stat: identity).
// This is equivalent to ggplot2's geom_col(). Use [Bar] when you want automatic
// counting/aggregation, and Col when you have pre-computed values.
//
// Default stat: identity. Default position: stack.
//
// Relevant options: WithColor, WithFill, WithAlpha, WithWidth, WithLineWidth.
func Col(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeBar,
		StatName: stat.Identity,
		Position: position.Stack(),
		Params:   Params{Width: 0.7},
	}
	applyOpts(&l, opts)

	return l
}

// Histogram creates a binned histogram geometry layer.
// Default stat: bin. Default position: stack.
//
// Relevant options: WithColor, WithFill, WithAlpha, WithWidth, WithBins, WithLineWidth.
func Histogram(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeHistogram,
		StatName: stat.Bin,
		Position: position.Stack(),
		Params:   Params{Bins: 30, Alpha: 1.0},
	}
	applyOpts(&l, opts)

	return l
}

// Area creates a filled area geometry layer.
//
// Relevant options: WithColor, WithFill, WithAlpha, WithLineWidth.
func Area(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeArea,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{Alpha: 0.6},
	}
	applyOpts(&l, opts)

	return l
}

// Polygon creates a closed polygon geometry layer.
//
// Relevant options: WithColor, WithFill, WithAlpha, WithLineWidth.
func Polygon(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypePolygon,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{Alpha: 0.6, LineWidth: 2},
	}
	applyOpts(&l, opts)

	return l
}

// Smooth creates a smoothed trendline geometry layer.
// Default stat: smooth. Default method: lm.
//
// Relevant options: WithColor, WithAlpha, WithLineWidth, WithSize, WithMethod, WithSpan, WithPoints.
func Smooth(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeSmooth,
		StatName: stat.Smooth,
		Position: position.Identity(),
		Params:   Params{LineWidth: 3, Alpha: 1.0, Method: "lm", Points: 80},
	}
	applyOpts(&l, opts)

	return l
}

// Density creates a kernel density estimation geometry layer.
//
// Relevant options: WithColor, WithFill, WithAlpha, WithLineWidth, WithPoints.
func Density(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeDensity,
		StatName: stat.Density,
		Position: position.Identity(),
		Params:   Params{Alpha: 0.6, Points: 512},
	}
	applyOpts(&l, opts)

	return l
}

// Text creates a text label geometry layer.
//
// Relevant options: WithColor, WithAlpha, WithFontSize, WithFontFamily, WithAngle.
func Text(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeText,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{FontSize: 10, Alpha: 1.0},
	}
	applyOpts(&l, opts)

	return l
}

// Step creates a step-function line geometry layer.
//
// Relevant options: WithColor, WithAlpha, WithLineWidth, WithSize.
func Step(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeStep,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{LineWidth: 2, Alpha: 1.0},
	}
	applyOpts(&l, opts)

	return l
}

// Boxplot creates a box-and-whisker geometry layer.
// It uses stat "boxplot" which computes the five-number summary
// (min, Q1, median, Q3, max) for each unique X group.
//
// Relevant options: WithColor, WithFill, WithAlpha, WithWidth, WithLineWidth.
func Boxplot(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeBoxPlot,
		StatName: stat.Boxplot,
		Position: position.Identity(),
		Params:   Params{Width: 0.5, Alpha: 0.8, LineWidth: 1.5},
	}
	applyOpts(&l, opts)

	return l
}

// Rug creates a rug (marginal tick) geometry layer.
//
// Relevant options: WithColor, WithAlpha, WithLineWidth.
func Rug(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeRug,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{LineWidth: 1, Alpha: 0.5},
	}
	applyOpts(&l, opts)

	return l
}

// HLine creates a horizontal reference line at the given Y intercept.
//
// Example:
//
//	geom.HLine(geom.WithIntercept(0), geom.WithColor("#CC0000"))
//
// Relevant options: WithIntercept, WithColor, WithAlpha, WithLineWidth, WithLabel.
func HLine(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeHLine,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{LineWidth: 1, Alpha: 0.8},
	}
	applyOpts(&l, opts)

	return l
}

// VLine creates a vertical reference line at the given X intercept.
//
// Example:
//
//	geom.VLine(geom.WithIntercept(5), geom.WithColor("#006600"))
//
// Relevant options: WithIntercept, WithColor, WithAlpha, WithLineWidth, WithLabel.
func VLine(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeVLine,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{LineWidth: 1, Alpha: 0.8},
	}
	applyOpts(&l, opts)

	return l
}

// WithSlope sets the slope for [ABLine]. The line equation is y = slope*x + intercept.
func WithSlope(s float64) Opt {
	return func(l *Layer) { l.Params.Slope = s }
}

// ABLine creates a reference line defined by y = slope*x + intercept.
// Use [WithIntercept] for the y-intercept and [WithSlope] for the slope.
//
// Example:
//
//	geom.ABLine(geom.WithIntercept(0), geom.WithSlope(1.5), geom.WithColor("#9B59B6"))
//
// Relevant options: WithIntercept, WithSlope, WithColor, WithAlpha, WithLineWidth, WithLabel.
func ABLine(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeABLine,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{LineWidth: 1, Alpha: 0.8, Slope: 1},
	}
	applyOpts(&l, opts)

	return l
}

// Tile creates a heatmap-cell geometry layer. Each row maps to a filled
// rectangle at (x, y) whose color comes from a continuous color scale.
//
// Required aesthetics: x, y, fill (or a continuous color column).
// Relevant options: WithColor, WithFill, WithAlpha.
func Tile(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeTile,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{Alpha: 1.0},
	}
	applyOpts(&l, opts)

	return l
}

// Segment creates a line-segment geometry layer. Each row draws a line
// from (x, y) to (xend, yend).
//
// Required aesthetics: x, y, xend, yend.
// Relevant options: WithColor, WithAlpha, WithLineWidth.
func Segment(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeSegment,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{LineWidth: 1, Alpha: 1.0},
	}
	applyOpts(&l, opts)

	return l
}

// ErrorBar creates an error bar geometry layer. Each row draws a vertical
// or horizontal line from (x, ymin) to (x, ymax) with small caps.
//
// Required aesthetics: x, ymin, ymax.
// Relevant options: WithColor, WithAlpha, WithLineWidth, WithWidth, WithOrientation.
func ErrorBar(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeErrorBar,
		StatName: stat.Identity,
		Position: position.Identity(),
		Params:   Params{LineWidth: 1, Alpha: 1.0, Width: 0.5},
	}
	applyOpts(&l, opts)

	return l
}
