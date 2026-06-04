package ggplot

import (
	"maps"

	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/colormap"
	"github.com/TuSKan/ggplot/coord"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/facet"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/scale"
	"github.com/TuSKan/ggplot/theme"
)

// System column names reserved by the build pipeline.
// These columns are injected into every layer's dataset during Build.
const (
	ColPANEL = "PANEL" // int64 -- facet panel index (0-based)
	ColGroup = "group" // int64 -- group index within a panel (0-based)
)

// PlotSpec is the fully declarative specification of a plot, produced by the
// user-facing builder API and consumed by the compilation pipeline.
type PlotSpec struct {
	// Dataset is the primary data source.
	Dataset dataset.Dataset

	// GlobalMapping maps aesthetic channels to column names at the plot level.
	// Per-layer mappings override these.
	GlobalMapping AesMap

	// Layers describes each visual layer (geom + stat + position + local aes).
	Layers []LayerSpec

	// ScaleOverrides holds user-specified scale configurations, keyed by
	// aesthetic channel ("x", "y", "color", etc.).
	ScaleOverrides map[string]ScaleOverride

	// ColorScales holds user-specified color/fill scales, keyed by
	// aesthetic channel ("color" or "fill"). nil entries fall back to
	// the auto-detected default ([colormap.Viridis] for continuous data,
	// [colormap.Tab10] for discrete data).
	ColorScales map[string]*colormap.Scale

	// Coord defines the coordinate system (default: Cartesian).
	Coord coord.Coord

	// Facet defines the faceting strategy (default: None).
	Facet facet.Facet

	// Theme holds the theme name or configuration.
	ThemeName theme.Name

	// Labels holds plot title, subtitle, axis labels, caption.
	Labels Labels

	// XLim and YLim hold optional user-specified axis limits.
	// nil means auto-detect from data.
	XLim [2]*float64
	YLim [2]*float64

	// LegendPosition controls where the legend is drawn.
	LegendPosition string

	// AxisGuideX controls X-axis label layout (dodging, overlap handling).
	AxisGuideX AxisGuide

	// ColorBarWidth sets the width of the continuous color bar in pixels.
	// Zero means default (12px).
	ColorBarWidth float64
	// ColorBarNBin sets the number of discrete gradient steps in the color bar.
	// Zero means default (256).
	ColorBarNBin int
	// LegendNCols sets the number of columns for the categorical legend.
	// Zero means single column (vertical) or single row (horizontal).
	LegendNCols int

	// Mapped scale overrides configured via Plot builder methods.
	SizeScale     scale.Scale
	AlphaScale    scale.Scale
	ShapeScale    scale.Scale
	LinetypeScale scale.Scale

	// Annotations holds fixed-coordinate visual elements that bypass the
	// data/stat/position pipeline. They are drawn after all data layers.
	Annotations []Annotation

	// SecondAxis holds a secondary Y-axis specification. When non-nil,
	// a right-side Y-axis is drawn with ticks derived from the primary Y
	// through the transform function pair.
	SecondAxis *scale.SecAxisSpec

	// ThemeOverrides holds per-plot theme element overrides.
	// Applied after the base theme is resolved, before rendering.
	ThemeOverrides []theme.Override

	// PlotMargin overrides the theme's default plot margins with typed units.
	// When nil, the theme's Spacing margins are used.
	PlotMargin *theme.PlotMargin

	// Alignment overrides block-level alignment for titles, labels, and
	// legend within the theme. When nil, theme defaults are used.
	Alignment *theme.BlockAlignment
}

// AesMap maps aesthetic channel names to column names.
type AesMap map[string]string

// Merge returns a new AesMap with entries from other as base, overridden
// by entries from the receiver (a takes priority).
func (a AesMap) Merge(other AesMap) AesMap {
	result := make(AesMap, len(a)+len(other))
	maps.Copy(result, other)
	maps.Copy(result, a)

	return result
}

// ToAesMap converts a slice of [aes.Mapping] into an [AesMap].
func ToAesMap(mappings []aes.Mapping) AesMap {
	m := make(AesMap, len(mappings))
	for _, am := range mappings {
		m[am.Channel] = am.Column
	}

	return m
}

// LayerSpec describes a single visual layer in the plot.
type LayerSpec struct {
	Geom    geom.Layer
	Mapping AesMap // per-layer aesthetic overrides
}

// ScaleOverride captures a user-requested scale for a specific aesthetic channel.
type ScaleOverride struct {
	Type   scale.Type        // e.g., scale.Log10, scale.Sqrt, scale.Reverse
	Params map[string]string // type-specific parameters
	Opts   []scale.Opt       // functional options (WithBreaks, WithLabels, etc.)
}

// Labels holds all text annotations for a plot.
type Labels struct {
	Title    string
	Subtitle string
	X        string
	Y        string
	Caption  string

	// Legend title overrides — when non-empty, override the default
	// column-name title for each legend type.
	Color string // categorical / continuous color legend
	Size  string // graduated-size legend
	Alpha string // continuous-alpha legend
}

// AxisGuide controls axis label layout — dodging, overlap handling,
// and rotation for dense categorical axes.
//
// NDodge sets the number of stagger rows for axis tick labels.
// When NDodge is 0, the renderer auto-detects overlapping labels and
// staggers them across 2 rows. Set NDodge to 1 to disable dodging.
// Values ≥ 2 force that many stagger rows.
type AxisGuide struct {
	NDodge int
}
