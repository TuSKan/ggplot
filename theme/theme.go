// Package theme provides visual styling configurations for plots.
// Themes control non-data visual elements: background colors, fonts,
// grid lines, tick marks, spacing, and the discrete color palette
// used when no explicit color scale is set.
//
// The preset catalog mirrors matplotlib's stylelib
// (https://matplotlib.org/stable/gallery/style_sheets/style_sheets_reference.html)
// and the curated set from raybuhr/pyplot-themes
// (https://github.com/raybuhr/pyplot-themes), with values lifted directly
// from the upstream .mplstyle files where they overlap.
package theme

import (
	"fmt"
	"image/color"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// Name identifies a built-in theme.
type Name string

// Built-in theme names.
//
// Default resolves to the matplotlib ggplot preset; users who want the old
// hand-tuned light theme should pick a specific named preset instead.
const (
	Default Name = "default"

	// Library originals (kept for callers that selected them by name).
	Minimal Name = "minimal"
	Dark    Name = "dark"
	BW      Name = "bw"

	// Matplotlib stylelib presets.
	Ggplot              Name = "ggplot"
	Classic             Name = "classic"
	Grayscale           Name = "grayscale"
	Bmh                 Name = "bmh"
	Fivethirtyeight     Name = "fivethirtyeight"
	DarkBackground      Name = "dark_background"
	SolarizeLight2      Name = "solarize_light2"
	TableauColorblind10 Name = "tableau_colorblind10"
	Fast                Name = "fast"

	// Seaborn family — chrome variants.
	Seaborn          Name = "seaborn"
	SeabornDarkgrid  Name = "seaborn_darkgrid"
	SeabornWhitegrid Name = "seaborn_whitegrid"
	SeabornDark      Name = "seaborn_dark"
	SeabornWhite     Name = "seaborn_white"
	SeabornTicks     Name = "seaborn_ticks"

	// Seaborn family — palette variants.
	SeabornDeep        Name = "seaborn_deep"
	SeabornMuted       Name = "seaborn_muted"
	SeabornBright      Name = "seaborn_bright"
	SeabornColorblind  Name = "seaborn_colorblind"
	SeabornPastel      Name = "seaborn_pastel"
	SeabornDarkPalette Name = "seaborn_dark_palette"

	// Seaborn family — font-size variants.
	SeabornPaper    Name = "seaborn_paper"
	SeabornNotebook Name = "seaborn_notebook"
	SeabornTalk     Name = "seaborn_talk"
	SeabornPoster   Name = "seaborn_poster"

	// Additions from raybuhr/pyplot-themes that don't overlap with matplotlib.
	PaulTol    Name = "paul_tol"
	Few        Name = "few"
	UCBerkeley Name = "uc_berkeley"
	Tableau    Name = "tableau"
)

// Theme encapsulates the complete visual styling for a plot.
type Theme struct {
	Name       string
	Background color.Color
	Panel      PanelStyle
	Grid       GridStyle
	Text       TextStyles
	Ticks      TickStyle
	Spacing    Spacing
	// Palette is the discrete color cycle used when the plot has no
	// explicit color scale set. Mirrors matplotlib's axes.prop_cycle.
	// May be nil; callers fall back to colormap.Tab10.
	Palette []color.Color
}

// PanelStyle controls the data panel appearance.
type PanelStyle struct {
	Background  color.Color
	Border      color.Color
	BorderWidth float64
}

// GridStyle controls major and minor grid lines.
type GridStyle struct {
	MajorColor     color.Color
	MajorWidth     float64
	MinorColor     color.Color
	MinorWidth     float64
	MajorLineCount int       // 0 = auto
	DashPattern    []float64 // nil = solid, e.g. {4,4} = dashed, {2,3} = dotted
}

// TextStyles holds font configurations for different text roles.
type TextStyles struct {
	Title      FontConfig
	Subtitle   FontConfig
	AxisTitle  FontConfig
	TickLabel  FontConfig
	Legend     FontConfig
	Annotation FontConfig
}

// FontConfig encapsulates text rendering parameters.
type FontConfig struct {
	Family string
	Size   float64
	Color  color.Color
	Bold   bool
	Italic bool
}

// TickStyle controls axis tick mark appearance.
type TickStyle struct {
	Length float64
	Width  float64
	Color  color.Color
}

// Spacing controls margins and inter-element spacing.
type Spacing struct {
	MarginTop    float64
	MarginRight  float64
	MarginBottom float64
	MarginLeft   float64
	PanelSpacing float64
}

// Factory builds a Theme on demand. Each preset is registered as a
// Factory so Resolve can return a fresh value per call (callers mutate
// the result via the Plot builder).
type Factory func() Theme

var (
	registryMu sync.RWMutex
	registry   = make(map[Name]Factory)
)

// Register adds a named theme factory to the global registry.
// Returns an error if name is already taken; use [MustRegister] for
// init-time registration where a duplicate is a programmer error.
//
// Intended both for built-in presets (registered via init() in their
// own files) and for user code that wants to add custom themes
// resolvable by name.
func Register(name Name, f Factory) error {
	if name == "" {
		return fmt.Errorf("theme: cannot register empty name")
	}
	if f == nil {
		return fmt.Errorf("theme: cannot register nil factory for %q", name)
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry[name]; dup {
		return fmt.Errorf("theme: name %q already registered", name)
	}
	registry[name] = f
	return nil
}

// MustRegister is [Register] but panics on error. Intended for init().
func MustRegister(name Name, f Factory) {
	if err := Register(name, f); err != nil {
		panic(err)
	}
}

// Resolve returns a Theme for the given name. Empty name resolves to
// the default (the matplotlib ggplot preset). Unknown names return an
// error.
func Resolve(name Name) (Theme, error) {
	if name == "" {
		name = Default
	}
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return Theme{}, fmt.Errorf("theme: unknown name %q", name)
	}
	return f(), nil
}

// AllNames returns every registered Name in alphabetical order.
func AllNames() []Name {
	registryMu.RLock()
	out := slices.Collect(maps.Keys(registry))
	registryMu.RUnlock()
	slices.Sort(out)
	return out
}

// baseTheme returns a neutral starting Theme that preset factories can
// patch. It carries no palette and no opinionated chrome — every
// preset is expected to override the fields it cares about.
func baseTheme(name string) Theme {
	return Theme{
		Name:       name,
		Background: color.White,
		Panel: PanelStyle{
			Background:  color.White,
			Border:      color.RGBA{R: 200, G: 200, B: 200, A: 255},
			BorderWidth: 1,
		},
		Grid: GridStyle{
			MajorColor:  color.RGBA{R: 200, G: 200, B: 200, A: 180},
			MajorWidth:  0.5,
			MinorColor:  color.RGBA{R: 230, G: 230, B: 230, A: 120},
			MinorWidth:  0.3,
			DashPattern: nil,
		},
		Text: TextStyles{
			Title:      FontConfig{Family: "sans-serif", Size: 14, Color: color.Black, Bold: true},
			Subtitle:   FontConfig{Family: "sans-serif", Size: 11, Color: gray(80)},
			AxisTitle:  FontConfig{Family: "sans-serif", Size: 11, Color: gray(40)},
			TickLabel:  FontConfig{Family: "sans-serif", Size: 10, Color: gray(60)},
			Legend:     FontConfig{Family: "sans-serif", Size: 10, Color: gray(40)},
			Annotation: FontConfig{Family: "sans-serif", Size: 10, Color: gray(60)},
		},
		Ticks: TickStyle{
			Length: 5, Width: 1, Color: gray(60),
		},
		Spacing: Spacing{
			MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10,
			PanelSpacing: 10,
		},
	}
}

// gray builds an opaque gray of the given level (0–255).
func gray(v uint8) color.Color {
	return color.RGBA{R: v, G: v, B: v, A: 255}
}

// hex parses a 6-character hex color (with optional leading '#') into an
// opaque RGBA. Panics on malformed input — preset files supply literals
// from upstream stylesheets, not user data.
func hex(s string) color.Color {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		panic(fmt.Sprintf("theme.hex: expected 6 hex chars, got %q", s))
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		panic(fmt.Sprintf("theme.hex: %v", err))
	}
	return color.RGBA{
		R: uint8(n >> 16),
		G: uint8(n >> 8),
		B: uint8(n),
		A: 255,
	}
}

// hexA is hex with an explicit alpha (0–255).
func hexA(s string, a uint8) color.Color {
	c := hex(s).(color.RGBA)
	c.A = a
	return c
}
