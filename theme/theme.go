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
	"errors"
	"fmt"
	"image/color"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
)

var (
	// ErrEmptyName is returned when registering a theme with an empty name.
	ErrEmptyName = errors.New("theme: cannot register empty name")
	// ErrNilFactory is returned when registering a nil factory.
	ErrNilFactory = errors.New("theme: cannot register nil factory")
	// ErrDuplicateName is returned when a theme name is already registered.
	ErrDuplicateName = errors.New("theme: name already registered")
	// ErrUnknownName is returned when resolving an unregistered theme name.
	ErrUnknownName = errors.New("theme: unknown name")
)

// Name identifies a built-in theme.
type Name string

// Built-in theme names.
//
// Default resolves to the matplotlib ggplot preset; users who want the old
// hand-tuned light theme should pick a specific named preset instead.
const (
	// Default is the default theme.
	Default Name = "default"

	// Minimal is the library-original minimal theme.
	Minimal Name = "minimal"

	// Dark is the dark theme.
	Dark Name = "dark"
	// BW is the black and white theme.
	BW Name = "bw"

	// Ggplot is the matplotlib ggplot preset.
	Ggplot Name = "ggplot"

	// Classic is the classic theme.
	Classic Name = "classic"

	// Grayscale is the grayscale theme.
	Grayscale Name = "grayscale"

	// Bmh is the bmh theme.
	Bmh Name = "bmh"

	// Fivethirtyeight is the fivethirtyeight theme.
	Fivethirtyeight Name = "fivethirtyeight"

	// DarkBackground is the dark background theme.
	DarkBackground Name = "dark_background"

	// SolarizeLight2 is the solarize light2 theme.
	SolarizeLight2 Name = "solarize_light2"

	// SolarizeDark is the solarize dark theme.
	SolarizeDark Name = "solarize_dark"

	// TableauColorblind10 is the tableau colorblind10 theme.
	TableauColorblind10 Name = "tableau_colorblind10"

	// Fast is the fast theme.
	Fast Name = "fast"

	// Seaborn family — chrome variants.
	Seaborn Name = "seaborn"

	// SeabornDarkgrid is the seaborn darkgrid theme.
	SeabornDarkgrid Name = "seaborn_darkgrid"

	// SeabornWhitegrid is the seaborn whitegrid theme.
	SeabornWhitegrid Name = "seaborn_whitegrid"

	// SeabornDark is the seaborn dark theme.
	SeabornDark Name = "seaborn_dark"

	// SeabornWhite is the seaborn white theme.
	SeabornWhite Name = "seaborn_white"

	// SeabornTicks is the seaborn ticks theme.
	SeabornTicks Name = "seaborn_ticks"

	// SeabornDeep is the Seaborn deep palette variant.
	SeabornDeep Name = "seaborn_deep"

	// SeabornMuted is the Seaborn muted palette variant.
	SeabornMuted Name = "seaborn_muted"

	// SeabornBright is the Seaborn bright palette variant.
	SeabornBright Name = "seaborn_bright"

	// SeabornColorblind is the Seaborn colorblind palette variant.
	SeabornColorblind Name = "seaborn_colorblind"

	// SeabornPastel is the Seaborn pastel palette variant.
	SeabornPastel Name = "seaborn_pastel"

	// SeabornDarkPalette is the Seaborn dark palette variant.
	SeabornDarkPalette Name = "seaborn_dark_palette"

	// SeabornPaper is the Seaborn paper font-size variant.
	SeabornPaper Name = "seaborn_paper"

	// SeabornNotebook is the Seaborn notebook font-size variant.
	SeabornNotebook Name = "seaborn_notebook"

	// SeabornTalk is the Seaborn talk font-size variant.
	SeabornTalk Name = "seaborn_talk"

	// SeabornPoster is the Seaborn poster font-size variant.
	SeabornPoster Name = "seaborn_poster"

	// PaulTol is the Paul Tol colorblind-safe palette from pyplot-themes.
	PaulTol Name = "paul_tol"

	// Few is the few theme.
	Few Name = "few"

	// FewLight is the few light theme.
	FewLight Name = "few_light"

	// FewDark is the few dark theme.
	FewDark Name = "few_dark"

	// UCBerkeley is the ucberkeley theme.
	UCBerkeley Name = "uc_berkeley"

	// Tableau is the tableau theme.

	// Colorblind-safe theme (Wong 2011, 8 colors).
	Colorblind Name = "colorblind"

	// Autumn1 is a seasonal palette on tableau chrome.
	Autumn1 Name = "autumn1"
	// Autumn2 is a seasonal palette on tableau chrome.
	Autumn2 Name = "autumn2"
	// Canyon is a seasonal palette on tableau chrome.
	Canyon Name = "canyon"
	// Chili is a seasonal palette on tableau chrome.
	Chili Name = "chili"
	// Tomato is a seasonal palette on tableau chrome.
	Tomato Name = "tomato"

	// SolarizeLight is the solarized light companion (pyplot-themes palette).
	SolarizeLight Name = "solarize_light"

	// Petroff10 (new in matplotlib 3.10).
	Petroff10 Name = "petroff10"
)

// Theme encapsulates the complete visual styling for a plot.
type Theme struct {
	// Name is the name of the theme.
	Name string
	// Background is the background color of the plot.
	Background color.Color
	// Panel is the style of the plot panel.
	Panel PanelStyle
	// Grid is the style of the plot grid.
	Grid GridStyle
	// Text is the style of the plot text.
	Text TextStyles
	// Ticks is the style of the plot ticks.
	Ticks TickStyle
	// Spacing is the spacing of the plot.
	Spacing Spacing
	// Palette is the discrete color cycle used when the plot has no
	// explicit color scale set. Mirrors matplotlib's axes.prop_cycle.
	// May be nil; callers fall back to colormap.Tab10.
	Palette []color.Color
	// Geom holds default visual properties for geometry primitives.
	// Individual geom options (WithColor, WithFill, etc.) always take precedence.
	Geom GeomDefaults
}

// GeomDefaults holds theme-level visual defaults for geometry primitives
// (bars, histograms, areas, boxplots). These act as fallbacks when the user
// has not explicitly overridden a property with a geom option.
type GeomDefaults struct {
	// PatchEdgeColor is the stroke color drawn around filled geoms.
	// Mirrors matplotlib's patch.edgecolor.
	// A nil value means "darken the fill color" (legacy behaviour).
	PatchEdgeColor color.Color
	// PatchEdgeWidth is the stroke line width for filled geoms (pixels).
	PatchEdgeWidth float64
	// PatchAlpha is the default fill opacity for filled geoms [0,1].
	// Zero is treated as "use geom's built-in default" (typically 0.85).
	PatchAlpha float64
}

// PanelStyle controls the data panel appearance.
type PanelStyle struct {
	// Background is the background color of the panel.
	Background color.Color
	// Border is the border color of the panel.
	Border color.Color
	// BorderWidth is the border width of the panel.
	BorderWidth float64
}

// GridStyle controls major and minor grid lines.
type GridStyle struct {
	// MajorColor is the color of the major grid lines.
	MajorColor color.Color
	// MajorWidth is the width of the major grid lines.
	MajorWidth float64
	// MinorColor is the color of the minor grid lines.
	MinorColor color.Color
	// MinorWidth is the width of the minor grid lines.
	MinorWidth float64
	// MajorLineCount is the number of major grid lines.
	MajorLineCount int       // 0 = auto
	DashPattern    []float64 // nil = solid, e.g. {4,4} = dashed, {2,3} = dotted
}

// TextStyles holds font configurations for different text roles.
type TextStyles struct {
	// Title is the font configuration for the title.
	Title FontConfig
	// Subtitle is the font configuration for the subtitle.
	Subtitle FontConfig
	// AxisTitle is the font configuration for the axis titles.
	AxisTitle FontConfig
	// TickLabel is the font configuration for the tick labels.
	TickLabel FontConfig
	// Legend is the font configuration for the legend.
	Legend     FontConfig
	Annotation FontConfig
}

// FontConfig encapsulates text rendering parameters.
type FontConfig struct {
	// Family is the font family.
	Family string
	// Size is the font size.
	Size float64
	// Color is the font color.
	Color color.Color
	// Bold is whether the font is bold.
	Bold bool
	// Italic is whether the font is italic.
	Italic bool
}

// TickStyle controls axis tick mark appearance.
type TickStyle struct {
	// Length is the length of the tick marks.
	Length float64
	// Width is the width of the tick marks.
	Width float64
	// Color is the color of the tick marks.
	Color color.Color
}

// Spacing controls margins and inter-element spacing.
type Spacing struct {
	// MarginTop is the top margin.
	MarginTop float64
	// MarginRight is the right margin.
	MarginRight float64
	// MarginBottom is the bottom margin.
	MarginBottom float64
	// MarginLeft is the left margin.
	MarginLeft float64
	// PanelSpacing is the spacing between panels.
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
		return ErrEmptyName
	}

	if f == nil {
		return fmt.Errorf("%w for %q", ErrNilFactory, name)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, dup := registry[name]; dup {
		return fmt.Errorf("%w: %q", ErrDuplicateName, name)
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
		return Theme{}, fmt.Errorf("%w: %q", ErrUnknownName, name)
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
		// GeomDefaults: nil PatchEdgeColor → legacy "darken fill" stroke.
		Geom: GeomDefaults{
			PatchEdgeColor: nil,
			PatchEdgeWidth: 0.5,
			PatchAlpha:     0,
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
func hex(s string) color.RGBA {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		panic(fmt.Sprintf("theme.hex: expected 6 hex chars, got %q", s))
	}

	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		panic(fmt.Sprintf("theme.hex: %v", err))
	}

	return color.RGBA{
		R: uint8(n >> 16), //nolint:gosec // G115: hex input is validated to 6 chars (24 bits); overflow impossible.
		G: uint8(n >> 8),  //nolint:gosec // G115: hex input is validated to 6 chars (24 bits); overflow impossible.
		B: uint8(n),       //nolint:gosec // G115: hex input is validated to 6 chars (24 bits); overflow impossible.
		A: 255,
	}
}

// hexA is hex with an explicit alpha (0–255).
func hexA(s string, a uint8) color.Color {
	c := hex(s)
	c.A = a

	return c
}
