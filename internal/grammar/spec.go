package grammar

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
	Opts   []scale.ScaleOpt  // functional options (WithBreaks, WithLabels, etc.)
}

// Labels holds all text annotations for a plot.
type Labels struct {
	Title    string
	Subtitle string
	X        string
	Y        string
	Caption  string
}
