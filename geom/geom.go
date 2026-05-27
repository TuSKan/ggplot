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

	"github.com/TuSKan/ggplot/stat"
)

// Layer represents a declarative layer specification produced by a geom
// constructor. It carries the geometry type, an ordered transform pipeline,
// position adjustment, per-layer aesthetic mappings, and visual parameters.
type Layer struct {
	Geom     Type              // geometry type (point, line, bar, etc.)
	Pipeline []stat.Transform  // ordered chain of transforms; nil = identity
	Position Pos               // position adjustment
	Params   Params            // visual parameters specific to this geometry
	Mapping  map[string]string // per-layer aesthetic overrides (channel → column)

	// setFlags tracks which options were explicitly set by the user.
	// Used by Validate() to emit warnings for irrelevant options.
	setFlags OptFlag
	warnings []string // validation warnings from applyOpts
	statCfg  statConfig
}

// Warnings returns any validation warnings generated during construction.
func (l *Layer) Warnings() []string { return l.warnings }

// Type identifies the kind of geometry.
type Type string

// TypePoint identifies a scatter point geometry.
const (
	TypePoint      Type = "point"
	TypeLine       Type = "line"
	TypeBar        Type = "bar"
	TypeHistogram  Type = "histogram"
	TypeArea       Type = "area"
	TypePolygon    Type = "polygon"
	TypeSmooth     Type = "smooth"
	TypeText       Type = "text"
	TypeBoxPlot    Type = "boxplot"
	TypeErrorBar   Type = "errorbar"
	TypeDensity    Type = "density"
	TypeTile       Type = "tile"
	TypeRug        Type = "rug"
	TypeSegment    Type = "segment"
	TypeStep       Type = "step"
	TypeHLine      Type = "hline"
	TypeVLine      Type = "vline"
	TypeABLine     Type = "abline"
	TypeRibbon     Type = "ribbon"     // filled band between ymin/ymax
	TypeDifference Type = "difference" // difference between two series
	TypeRect       Type = "rect"       // unified rectangle mark (replaces TypeBar/TypeHistogram for pipeline constructors)
	TypeCrossbar   Type = "crossbar"   // box with median line between ymin/ymax (no whiskers)
	TypeLinerange  Type = "linerange"  // vertical/horizontal line from ymin to ymax (no caps)
	TypePointrange Type = "pointrange" // point at y + linerange from ymin to ymax
	TypeCurve      Type = "curve"      // quadratic bezier curve between endpoints
	TypeViolin     Type = "violin"     // mirrored kernel density estimate per group
	TypeDotplot    Type = "dotplot"    // stacked dots at binned positions
	TypeRaster     Type = "raster"     // dense pixel-aligned image grid
	TypeLabel      Type = "label"      // text with background box (annotation)
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
//
// Note: stat-specific parameters (bins, method, bandwidth, whisker, notch)
// are owned by their respective [stat.Transform] options, not by Params.
type Params struct {
	// Common
	Color     string  // hex color override (e.g., "#4C72B0")
	Fill      string  // hex fill color override
	Alpha     float64 // opacity [0, 1]
	LineWidth float64 // stroke width in pixels

	// Point-specific
	Size  float64 // point radius in pixels (default = 3)
	Shape string  // "circle", "square", "triangle", "diamond" (default = "circle")

	// Bar/Histogram/Rect-specific
	Width float64 // relative bar width [0, 1] (default = 0.8)
	Gap   float64 // gap between bars [0, 1] (0 = touching, 1 = invisible; default = 0.2)
	Inset float64 // pixel inset per side between adjacent rects (default = 0; 0.5 for continuous bins)

	// Text-specific
	FontSize   float64 // text font size in points
	FontFamily string  // font family name
	Angle      float64 // rotation angle in degrees

	// Orientation
	Orientation Orientation // "v" (default) or "h" — controls axis extension direction

	// Legend
	Label string // legend label for this layer (used with manual colors)

	// Reference lines
	Intercept float64 // y-intercept (hline) or x-intercept (vline)
	Slope     float64 // slope for abline (y = slope*x + intercept)

	// Curve-specific
	Curvature float64 // bezier curvature multiplier (default = 0.5); 0 = straight line

	// Raster-specific
	Interpolate bool // true = bilinear interpolation; false = nearest-neighbor (default)

	// Label-specific
	Padding float64 // padding in pixels around label text for background box (default = 4)
}

// --- Option tracking for validation ---

// OptFlag tracks which parameters were explicitly set by the user.
// Exported so third-party packages can compose relevance masks for
// [RegisterGeomType].
type OptFlag uint32

// OptColor tracks whether WithColor was set.
const (
	OptColor         OptFlag = 1 << iota // common
	OptFill                              // common
	OptAlpha                             // common
	OptLineWidth                         // common
	OptSize                              // point, (also sets LineWidth)
	OptShape                             // point
	OptWidth                             // bar, histogram
	OptFontSize                          // text
	OptFontFamily                        // text
	OptAngle                             // text
	OptOrientation                       // bar, histogram, boxplot, area, density, rug
	OptBins                              // histogram
	OptMethod                            // smooth
	OptSmoothPoints                      // smooth
	OptBandwidth                         // density
	OptDensityPoints                     // density
	OptWhisker                           // boxplot
	OptNotch                             // boxplot
	OptCurvature                         // curve
	OptInterpolate                       // raster
	OptPadding                           // label
)

// paramRelevance maps geometry types to what parameters are meaningful for them.
var paramRelevance = map[Type]OptFlag{
	TypePoint:      OptColor | OptFill | OptAlpha | OptSize | OptShape,
	TypeLine:       OptColor | OptAlpha | OptLineWidth | OptSize, // Size → LineWidth
	TypeBar:        OptColor | OptFill | OptAlpha | OptWidth | OptLineWidth | OptOrientation,
	TypeHistogram:  OptColor | OptFill | OptAlpha | OptWidth | OptLineWidth | OptOrientation | OptBins,
	TypeArea:       OptColor | OptFill | OptAlpha | OptLineWidth | OptOrientation,
	TypePolygon:    OptColor | OptFill | OptAlpha | OptLineWidth,
	TypeSmooth:     OptColor | OptAlpha | OptLineWidth | OptSize | OptMethod | OptSmoothPoints,
	TypeDensity:    OptColor | OptFill | OptAlpha | OptLineWidth | OptOrientation | OptBandwidth | OptDensityPoints,
	TypeText:       OptColor | OptAlpha | OptFontSize | OptFontFamily | OptAngle,
	TypeStep:       OptColor | OptAlpha | OptLineWidth | OptSize,
	TypeRug:        OptColor | OptAlpha | OptLineWidth | OptOrientation,
	TypeBoxPlot:    OptColor | OptFill | OptAlpha | OptWidth | OptLineWidth | OptOrientation | OptWhisker | OptNotch,
	TypeErrorBar:   OptColor | OptAlpha | OptLineWidth | OptWidth,
	TypeSegment:    OptColor | OptAlpha | OptLineWidth,
	TypeTile:       OptColor | OptFill | OptAlpha,
	TypeHLine:      OptColor | OptAlpha | OptLineWidth,
	TypeVLine:      OptColor | OptAlpha | OptLineWidth,
	TypeABLine:     OptColor | OptAlpha | OptLineWidth,
	TypeRibbon:     OptColor | OptFill | OptAlpha | OptLineWidth,
	TypeDifference: OptColor | OptFill | OptAlpha | OptLineWidth,
	TypeRect:       OptColor | OptFill | OptAlpha | OptWidth | OptLineWidth | OptOrientation | OptBins,
	TypeCrossbar:   OptColor | OptFill | OptAlpha | OptWidth | OptLineWidth,
	TypeLinerange:  OptColor | OptAlpha | OptLineWidth,
	TypePointrange: OptColor | OptAlpha | OptLineWidth | OptSize,
	TypeCurve:      OptColor | OptAlpha | OptLineWidth | OptCurvature,
	TypeViolin:     OptColor | OptFill | OptAlpha | OptWidth | OptLineWidth | OptOrientation | OptBandwidth,
	TypeDotplot:    OptColor | OptFill | OptAlpha | OptSize | OptBins | OptOrientation,
	TypeRaster:     OptAlpha | OptInterpolate,
	TypeLabel:      OptColor | OptFill | OptAlpha | OptFontSize | OptFontFamily | OptPadding,
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
	OptColor:         "WithColor",
	OptFill:          "WithFill",
	OptAlpha:         "WithAlpha",
	OptLineWidth:     "WithLineWidth",
	OptSize:          "WithSize",
	OptShape:         "WithShape",
	OptWidth:         "WithWidth",
	OptFontSize:      "WithFontSize",
	OptFontFamily:    "WithFontFamily",
	OptAngle:         "WithAngle",
	OptOrientation:   "WithOrientation",
	OptBins:          "WithBins",
	OptMethod:        "WithMethod",
	OptSmoothPoints:  "WithSmoothPoints",
	OptBandwidth:     "WithBandwidth",
	OptDensityPoints: "WithDensityPoints",
	OptWhisker:       "WithWhisker",
	OptNotch:         "WithNotch",
	OptCurvature:     "WithCurvature",
	OptInterpolate:   "WithInterpolate",
}

// Validate checks if the configured params are meaningful for this geometry
// type and returns a list of warning messages for irrelevant options.
//
// Example:
//
//	layer := geom.Point(geom.WithWidth(0.5))  // width is for bars, not points
//	warnings := layer.Validate()
//	// warnings = ["geom_point: WithWidth has no effect (relevant for: bar, histogram)"]
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

	for flag := OptFlag(1); flag <= OptInterpolate; flag <<= 1 {
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

// Stat sets an explicit transform pipeline for this layer, overriding
// any defaults from sugar options (WithBins, WithMethod, etc.).
// Use this for advanced composition when the sugar options are insufficient.
//
//	geom.Histogram(geom.Stat(stat.BinX(stat.WithBins(50)), stat.Normalize()))
func Stat(transforms ...stat.Transform) Opt {
	return func(l *Layer) {
		l.Pipeline = transforms
		l.statCfg.explicit = true
	}
}

// WithPosition sets the position adjustment for this layer.
func WithPosition(p Pos) Opt { return func(l *Layer) { l.Position = p } }

// --- Stat-forwarding sugar options ---
// These configure the default pipeline for geometry types that have built-in
// statistical transforms. They are ignored if Stat() is used explicitly.

// WithBins sets the number of histogram bins. Relevant for [Histogram].
func WithBins(n int) Opt {
	return func(l *Layer) { l.statCfg.bins = n; l.statCfg.dirty = true; l.setFlags |= OptBins }
}

// WithMethod sets the smoothing method ("lm" or "loess"). Relevant for [Smooth].
func WithMethod(m string) Opt {
	return func(l *Layer) { l.statCfg.method = m; l.statCfg.dirty = true; l.setFlags |= OptMethod }
}

// WithSmoothPoints sets the output point count for smoothing. Relevant for [Smooth].
func WithSmoothPoints(n int) Opt {
	return func(l *Layer) { l.statCfg.smoothPoints = n; l.statCfg.dirty = true; l.setFlags |= OptSmoothPoints }
}

// WithBandwidth sets the KDE bandwidth. Relevant for [Density].
func WithBandwidth(bw float64) Opt {
	return func(l *Layer) { l.statCfg.bandwidth = bw; l.statCfg.dirty = true; l.setFlags |= OptBandwidth }
}

// WithDensityPoints sets the output point count for KDE. Relevant for [Density].
func WithDensityPoints(n int) Opt {
	return func(l *Layer) { l.statCfg.densityPoints = n; l.statCfg.dirty = true; l.setFlags |= OptDensityPoints }
}

// WithWhisker sets the whisker extent type (e.g. "tukey"). Relevant for [Boxplot].
func WithWhisker(w string) Opt {
	return func(l *Layer) { l.statCfg.whisker = w; l.statCfg.dirty = true; l.setFlags |= OptWhisker }
}

// WithNotch enables notched boxplot display. Relevant for [Boxplot].
func WithNotch(b bool) Opt {
	return func(l *Layer) { l.statCfg.notch = b; l.statCfg.dirty = true; l.setFlags |= OptNotch }
}

// statConfig holds stat parameter overrides set via sugar options.
// After all Opt functions run, finalizePipeline rebuilds the pipeline
// from these values (unless Stat() was used explicitly).
type statConfig struct {
	bins          int
	method        string
	smoothPoints  int
	bandwidth     float64
	densityPoints int
	whisker       string
	notch         bool
	dirty         bool // true if any sugar option was set
	explicit      bool // true if Stat() was called (overrides sugar)
}

// finalizePipeline rebuilds the layer's pipeline from statCfg if sugar
// options were set and Stat() was not called explicitly.
func finalizePipeline(l *Layer) {
	if l.statCfg.explicit || !l.statCfg.dirty {
		return
	}

	switch l.Geom { //nolint:exhaustive // Only histogram, smooth, density, and boxplot have stat pipelines.
	case TypeHistogram, TypeRect:
		var opts []stat.BinOption
		if l.statCfg.bins > 0 {
			opts = append(opts, stat.WithBins(l.statCfg.bins))
		}

		l.Pipeline = []stat.Transform{stat.BinX(opts...)}

	case TypeSmooth:
		var opts []stat.SmoothOption
		if l.statCfg.method != "" {
			opts = append(opts, stat.WithMethod(l.statCfg.method))
		}

		if l.statCfg.smoothPoints > 0 {
			opts = append(opts, stat.WithSmoothPoints(l.statCfg.smoothPoints))
		}

		l.Pipeline = []stat.Transform{stat.SmoothXY(opts...)}

	case TypeDensity:
		var opts []stat.DensityOption
		if l.statCfg.bandwidth > 0 {
			opts = append(opts, stat.WithBandwidth(l.statCfg.bandwidth))
		}

		if l.statCfg.densityPoints > 0 {
			opts = append(opts, stat.WithDensityPoints(l.statCfg.densityPoints))
		}

		l.Pipeline = []stat.Transform{stat.DensityX(opts...)}

	case TypeBoxPlot:
		var opts []stat.BoxplotOption
		if l.statCfg.whisker != "" {
			opts = append(opts, stat.WithWhisker(l.statCfg.whisker))
		}

		if l.statCfg.notch {
			opts = append(opts, stat.WithNotch(l.statCfg.notch))
		}

		l.Pipeline = []stat.Transform{stat.BoxplotY(opts...)}

	case TypeViolin:
		var opts []stat.ViolinOption
		if l.statCfg.bandwidth > 0 {
			opts = append(opts, stat.WithViolinBandwidth(l.statCfg.bandwidth))
		}

		l.Pipeline = []stat.Transform{stat.ViolinY(opts...)}

	case TypeDotplot:
		var opts []stat.DotBinOption
		if l.statCfg.bins > 0 {
			// Convert bin count to a bin width estimate.
			opts = append(opts, stat.WithDotBinWidth(float64(l.statCfg.bins)))
		}

		l.Pipeline = []stat.Transform{stat.DotBin(opts...)}

	default:
		// Other geom types have no stat pipeline to rebuild from sugar options.
	}
}

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

// applyOpts applies options, finalizes the pipeline from sugar config,
// and stores validation warnings on the layer.
func applyOpts(l *Layer, opts []Opt) {
	for _, o := range opts {
		o(l)
	}

	finalizePipeline(l)
	l.warnings = l.Validate()
}

// Point creates a scatter point geometry layer.
// Default stat: identity. Default position: identity.
//
// Relevant options: WithColor, WithFill, WithAlpha, WithSize, WithShape.
func Point(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypePoint,
		Position: IdentityPos(),
		Params:   Params{Size: 3, Alpha: 1.0, Shape: "circle"},
	}
	applyOpts(&l, opts)

	return l
}

// JitterPoint creates a jittered point geometry layer — equivalent to ggplot2's
// geom_jitter(). This is a [Point] with [Jitter] position.
//
// Default jitter width and height are both 0.4 data units.
// Default seed is 42 (deterministic). Change via [WithJitterWidth],
// [WithJitterHeight], and [WithJitterSeed].
//
// Relevant options: all [Point] options plus [WithJitterWidth],
// [WithJitterHeight], [WithJitterSeed].
func JitterPoint(opts ...Opt) Layer {
	l := Layer{
		Geom: TypePoint,
		// Defaults match ggplot2's geom_jitter: width=0.4, height=0.4, seed=42.
		Position: Jitter(0.4, 0.4),
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
		Position: IdentityPos(),
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
		Pipeline: []stat.Transform{stat.Count()},
		Position: Stack(),
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
		Position: Stack(),
		Params:   Params{Width: 0.7},
	}
	applyOpts(&l, opts)

	return l
}

// Histogram creates a binned histogram geometry layer.
// Default transform: BinX(WithBins(30)). Default position: stack.
//
// Relevant options: WithColor, WithFill, WithAlpha, WithWidth, WithLineWidth.
func Histogram(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeRect,
		Pipeline: []stat.Transform{stat.BinX(stat.WithBins(30))},
		Position: Stack(),
		Params:   Params{Alpha: 1.0, Inset: 0.5},
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
		Position: IdentityPos(),
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
		Position: IdentityPos(),
		Params:   Params{Alpha: 0.6, LineWidth: 2},
	}
	applyOpts(&l, opts)

	return l
}

// Smooth creates a smoothed trendline geometry layer.
// Default transform: SmoothXY with method "lm".
//
// Relevant options: WithColor, WithAlpha, WithLineWidth, WithSize.
func Smooth(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeSmooth,
		Pipeline: []stat.Transform{stat.SmoothXY(stat.WithMethod("lm"), stat.WithSmoothPoints(80))},
		Position: IdentityPos(),
		Params:   Params{LineWidth: 3, Alpha: 1.0},
	}
	applyOpts(&l, opts)

	return l
}

// Density creates a kernel density estimation geometry layer.
//
// Relevant options: WithColor, WithFill, WithAlpha, WithLineWidth.
func Density(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeDensity,
		Pipeline: []stat.Transform{stat.DensityX()},
		Position: IdentityPos(),
		Params:   Params{Alpha: 0.6},
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
		Position: IdentityPos(),
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
		Position: IdentityPos(),
		Params:   Params{LineWidth: 2, Alpha: 1.0},
	}
	applyOpts(&l, opts)

	return l
}

// Boxplot creates a box-and-whisker geometry layer.
// Default transform: BoxplotY with Tukey whiskers.
//
// Relevant options: WithColor, WithFill, WithAlpha, WithWidth, WithLineWidth.
func Boxplot(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeBoxPlot,
		Pipeline: []stat.Transform{stat.BoxplotY()},
		Position: IdentityPos(),
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
		Position: IdentityPos(),
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
		Position: IdentityPos(),
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
		Position: IdentityPos(),
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
		Position: IdentityPos(),
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
		Position: IdentityPos(),
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
		Position: IdentityPos(),
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
		Position: IdentityPos(),
		Params:   Params{LineWidth: 1, Alpha: 1.0, Width: 0.5},
	}
	applyOpts(&l, opts)

	return l
}

// Crossbar creates a box-with-median-line geometry layer. Each row draws
// a filled rectangle from (x - width/2, ymin) to (x + width/2, ymax) with
// a horizontal line at y (the median or central value).
//
// Required aesthetics: x, y, ymin, ymax.
// Relevant options: WithColor, WithFill, WithAlpha, WithWidth, WithLineWidth.
func Crossbar(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeCrossbar,
		Position: IdentityPos(),
		Params:   Params{Width: 0.5, Alpha: 0.8, LineWidth: 1},
	}
	applyOpts(&l, opts)

	return l
}

// Linerange creates a vertical (or horizontal) line from ymin to ymax
// without caps. This is a thinner variant of [ErrorBar].
//
// Required aesthetics: x, ymin, ymax.
// Relevant options: WithColor, WithAlpha, WithLineWidth.
func Linerange(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeLinerange,
		Position: IdentityPos(),
		Params:   Params{LineWidth: 1, Alpha: 1.0},
	}
	applyOpts(&l, opts)

	return l
}

// Pointrange creates a point-with-range geometry layer. Each row draws a
// vertical line from ymin to ymax plus a point at (x, y).
//
// Required aesthetics: x, y, ymin, ymax.
// Relevant options: WithColor, WithAlpha, WithLineWidth, WithSize.
func Pointrange(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypePointrange,
		Position: IdentityPos(),
		Params:   Params{Size: 3, LineWidth: 1, Alpha: 1.0},
	}
	applyOpts(&l, opts)

	return l
}

// WithCurvature sets the bezier curvature multiplier for [Curve].
// 0 produces a straight line, 0.5 (default) produces a moderate curve.
// Negative values curve in the opposite direction.
func WithCurvature(c float64) Opt {
	return func(l *Layer) { l.Params.Curvature = c; l.setFlags |= OptCurvature }
}

// Curve creates a quadratic bezier curve geometry layer. Each row draws
// a curved line from (x, y) to (xend, yend) with curvature controlled
// by [WithCurvature].
//
// Required aesthetics: x, y, xend, yend.
// Relevant options: WithColor, WithAlpha, WithLineWidth, WithCurvature.
func Curve(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeCurve,
		Position: IdentityPos(),
		Params:   Params{LineWidth: 1, Alpha: 1.0, Curvature: 0.5},
	}
	applyOpts(&l, opts)

	return l
}

// Violin creates a mirrored kernel density geometry layer. Each group's
// Y values are estimated via KDE, and the density is drawn symmetrically
// around the group's categorical X position.
//
// Default stat: [stat.ViolinY]. Default position: identity.
// Relevant options: WithColor, WithFill, WithAlpha, WithWidth, WithLineWidth, WithBandwidth.
func Violin(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeViolin,
		Pipeline: []stat.Transform{stat.ViolinY()},
		Position: IdentityPos(),
		Params:   Params{Width: 0.8, Alpha: 0.6, LineWidth: 1},
	}
	applyOpts(&l, opts)

	return l
}

// Dotplot creates a stacked-dot geometry layer. Each observation is
// represented as a dot, stacked vertically within bins.
//
// Default stat: [stat.DotBin]. Default position: identity.
// Relevant options: WithColor, WithFill, WithAlpha, WithSize, WithBins.
func Dotplot(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeDotplot,
		Pipeline: []stat.Transform{stat.DotBin()},
		Position: IdentityPos(),
		Params:   Params{Size: 3, Alpha: 0.8},
	}
	applyOpts(&l, opts)

	return l
}

// --- Observable Plot-style mark constructors ---
//
// These constructors take a [stat.Transform] as the first argument,
// enabling composable stat-mark pipelines:
//
//	geom.RectY(stat.BinX(stat.WithBins(40)))          // histogram
//	geom.RectY(stat.NormalizeY(), stat.BinX(...))     // proportions
//	geom.LineY(stat.BinX(stat.WithBins(40)))          // frequency polygon
//	geom.AreaY(stat.DensityX())                       // filled density
//	geom.PointY(stat.GroupX("mean"))                  // mean scatter
//
// Any transform can feed any mark — this is the Observable Plot pattern.
// For common patterns, prefer the sugar constructors ([Histogram], [Bar],
// [Smooth], [Density]) which configure defaults automatically.

// RectY creates a rectangle mark anchored at y. The transform defines
// what data flows into the rectangles. With no transform, it renders
// pre-computed x/y values (equivalent to [Col]).
//
//	geom.RectY(stat.BinX(stat.WithBins(40)))   // histogram
//	geom.RectY(stat.Count())                    // bar chart
//	geom.RectY(stat.NormalizeY(), stat.BinX())  // proportions
func RectY(pipeline []stat.Transform, opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeRect,
		Pipeline: pipeline,
		Position: Stack(),
		Params:   Params{Width: 0.8, Alpha: 0.85, Inset: 0.5},
	}
	applyOpts(&l, opts)

	return l
}

// LineY creates a connected-line mark with a transform pipeline.
// With BinX, this becomes a frequency polygon. With SmoothXY, a
// regression line.
//
//	geom.LineY(stat.BinX(stat.WithBins(40)))  // frequency polygon
//	geom.LineY(stat.SmoothXY())               // regression line
func LineY(pipeline []stat.Transform, opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeLine,
		Pipeline: pipeline,
		Position: IdentityPos(),
		Params:   Params{LineWidth: 2, Alpha: 1.0},
	}
	applyOpts(&l, opts)

	return l
}

// AreaY creates a filled-area mark with a transform pipeline.
// With DensityX, this becomes a filled KDE curve. With BinX, a
// filled histogram outline.
//
//	geom.AreaY(stat.DensityX())               // filled density
//	geom.AreaY(stat.BinX(stat.WithBins(40)))  // filled histogram
func AreaY(pipeline []stat.Transform, opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeArea,
		Pipeline: pipeline,
		Position: IdentityPos(),
		Params:   Params{Alpha: 0.6},
	}
	applyOpts(&l, opts)

	return l
}

// PointY creates a scatter point mark with a transform pipeline.
// With GroupX, this becomes a grouped aggregate scatter.
//
//	geom.PointY(stat.GroupX("mean"))           // mean per x
//	geom.PointY(stat.BinX(stat.WithBins(40))) // dot plot of bin centers
func PointY(pipeline []stat.Transform, opts ...Opt) Layer {
	l := Layer{
		Geom:     TypePoint,
		Pipeline: pipeline,
		Position: IdentityPos(),
		Params:   Params{Size: 3, Alpha: 1.0, Shape: "circle"},
	}
	applyOpts(&l, opts)

	return l
}

// RectX creates a rectangle mark anchored at x. This is the horizontal
// counterpart of [RectY]. With StackX, produces a horizontally stacked bar.
//
//	geom.RectX([]stat.Transform{stat.BinY(stat.WithBins(40))})  // horizontal histogram
//	geom.RectX([]stat.Transform{stat.Count()})                   // horizontal bar chart
func RectX(pipeline []stat.Transform, opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeRect,
		Pipeline: pipeline,
		Position: Stack(),
		Params:   Params{Width: 0.8, Alpha: 0.85, Inset: 0.5, Orientation: Horizontal},
	}
	applyOpts(&l, opts)

	return l
}

// LineX creates a connected-line mark with horizontal orientation.
// This is the horizontal counterpart of [LineY].
//
//	geom.LineX([]stat.Transform{stat.GroupY("mean")})  // horizontal mean line
func LineX(pipeline []stat.Transform, opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeLine,
		Pipeline: pipeline,
		Position: IdentityPos(),
		Params:   Params{LineWidth: 2, Alpha: 1.0, Orientation: Horizontal},
	}
	applyOpts(&l, opts)

	return l
}

// AreaX creates a filled-area mark with horizontal orientation.
// This is the horizontal counterpart of [AreaY].
//
//	geom.AreaX([]stat.Transform{stat.DensityY()})  // horizontal density fill
func AreaX(pipeline []stat.Transform, opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeArea,
		Pipeline: pipeline,
		Position: IdentityPos(),
		Params:   Params{Alpha: 0.6, Orientation: Horizontal},
	}
	applyOpts(&l, opts)

	return l
}

// RibbonY creates a filled band between ymin and ymax columns.
// Used for confidence intervals, prediction bands, error envelopes, etc.
//
//	geom.RibbonY([]stat.Transform{stat.SmoothXY()}, geom.WithAlpha(0.3))
func RibbonY(pipeline []stat.Transform, opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeRibbon,
		Pipeline: pipeline,
		Position: IdentityPos(),
		Params:   Params{Alpha: 0.3},
	}
	applyOpts(&l, opts)

	return l
}

// Difference creates a layer that fills the area between two series,
// coloring positive and negative differences. Requires ymin and ymax
// columns in the pipeline output.
//
//	geom.Difference([]stat.Transform{stat.DeltaXY()}, geom.WithAlpha(0.5))
func Difference(pipeline []stat.Transform, opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeDifference,
		Pipeline: pipeline,
		Position: IdentityPos(),
		Params:   Params{Alpha: 0.4},
	}
	applyOpts(&l, opts)

	return l
}

// WithInterpolate enables bilinear interpolation for [Raster]. When false
// (default), nearest-neighbor sampling is used — sharp pixel edges, good for
// discrete data. When true, bilinear interpolation produces smooth gradients,
// better for continuous fields.
func WithInterpolate(b bool) Opt {
	return func(l *Layer) { l.Params.Interpolate = b; l.setFlags |= OptInterpolate }
}

// WithPadding sets the padding in pixels around label text for its background
// box. Default is 4. Used by [TypeLabel] annotations.
func WithPadding(px float64) Opt {
	return func(l *Layer) { l.Params.Padding = px; l.setFlags |= OptPadding }
}

// WithJitterWidth sets the horizontal jitter amount in data units for
// [JitterPoint]. Default is 0.4. The actual displacement is uniform
// in [−width, +width], matching ggplot2's position_jitter semantics.
func WithJitterWidth(w float64) Opt {
	return func(l *Layer) {
		if j, ok := l.Position.(jitter); ok {
			l.Position = Jitter(w, j.yAmt, WithSeed(j.seed))
		}
	}
}

// WithJitterHeight sets the vertical jitter amount in data units for
// [JitterPoint]. Default is 0.4. The actual displacement is uniform
// in [−height, +height], matching ggplot2's position_jitter semantics.
func WithJitterHeight(h float64) Opt {
	return func(l *Layer) {
		if j, ok := l.Position.(jitter); ok {
			l.Position = Jitter(j.xAmt, h, WithSeed(j.seed))
		}
	}
}

// WithJitterSeed sets the PRNG seed for [JitterPoint]. Same seed + same
// data length = same displacement. Default is 42.
func WithJitterSeed(seed uint64) Opt {
	return func(l *Layer) {
		if j, ok := l.Position.(jitter); ok {
			l.Position = Jitter(j.xAmt, j.yAmt, WithSeed(seed))
		}
	}
}

// Raster creates a dense pixel-aligned image grid geometry layer. Each row
// in the dataset represents one cell on a regular grid at (x, y). The fill
// color comes from a continuous color scale mapped to a fill or z column.
//
// Unlike [Tile], which draws individual rectangles per row, Raster composites
// the grid into a single [image.RGBA] and renders it via [canvas.Canvas.DrawImage].
// This is orders of magnitude faster for dense grids (e.g. 500×500).
//
// Required aesthetics: x, y, fill (or continuous color column).
// Relevant options: [WithAlpha], [WithInterpolate].
func Raster(opts ...Opt) Layer {
	l := Layer{
		Geom:     TypeRaster,
		Position: IdentityPos(),
		Params:   Params{Alpha: 1.0},
	}
	applyOpts(&l, opts)

	return l
}
