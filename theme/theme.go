// Package theme provides visual styling configurations for plots.
// Themes control non-data visual elements: background colors, fonts,
// grid lines, tick marks, spacing, and the discrete color palette
// used when no explicit color scale is set.
//
// Elements follow ggplot2's inheritance hierarchy:
//
//	text                 → root text defaults
//	├── plot.title
//	├── plot.subtitle
//	├── axis.title       → axis.title.x, axis.title.y
//	├── axis.text        → axis.text.x, axis.text.y
//	├── legend.title
//	├── legend.text
//	└── strip.text
//
//	line                 → root line defaults
//	├── axis.line        → axis.line.x, axis.line.y
//	├── axis.ticks       → axis.ticks.x, axis.ticks.y
//	├── panel.grid.major
//	└── panel.grid.minor
//
//	rect                 → root rect defaults
//	├── plot.background
//	├── panel.background
//	├── panel.border
//	├── legend.background
//	├── legend.key
//	└── strip.background
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

func init() {
	MustRegister(Default, newDashboard)
}

// Name identifies a built-in theme.
type Name string

// Built-in theme names.
//
// Default resolves to the Dashboard card-style theme; users who want the
// matplotlib ggplot preset should use Theme("ggplot") explicitly.
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

	// Tableau is the tableau theme (Tableau10 palette, white panel).
	Tableau Name = "tableau"

	// Observable is the Observable modern theme (Observable10 palette).
	Observable Name = "observable"

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

	// ObservableDark is the dark variant of the Observable modern theme.
	ObservableDark Name = "observable_dark"
	// Dashboard is a clean card-style dashboard theme.
	Dashboard Name = "dashboard"
	// Quartz is an Apple-inspired minimal theme.
	Quartz Name = "quartz"
	// Air is a minimal, airy theme with hidden axes.
	Air Name = "air"
	// Ink is a slate-dark theme with bright accents.
	Ink Name = "ink"

	// Tufte is a Tufte-inspired minimal theme (serif, no grid).
	Tufte Name = "tufte"
	// Academic is a journal-ready theme with serif type.
	Academic Name = "academic"
	// Newsroom is a bold headline-first editorial theme.
	Newsroom Name = "newsroom"
	// Editorial is a warm serif editorial theme.
	Editorial Name = "editorial"
	// Monochrome is a strict black-and-white theme.
	Monochrome Name = "monochrome"

	// GitHubLight is GitHub's light color scheme.
	GitHubLight Name = "github_light"
	// GitHubDark is GitHub's dark color scheme.
	GitHubDark Name = "github_dark"
	// Nord is the Nord color palette theme.
	Nord Name = "nord"
	// Dracula is the Dracula color palette theme.
	Dracula Name = "dracula"
	// GruvboxLight is the Gruvbox light color scheme.
	GruvboxLight Name = "gruvbox_light"
	// GruvboxDark is the Gruvbox dark color scheme.
	GruvboxDark Name = "gruvbox_dark"

	// AstronomyDark is a deep-space dark theme with pastel accents.
	AstronomyDark Name = "astronomy_dark"
	// NASA is a NASA-blue institutional theme.
	NASA Name = "nasa"
	// Ocean is a blue-gradient ocean-inspired theme.
	Ocean Name = "ocean"
	// Earth is a warm earthy-tone theme.
	Earth Name = "earth"
	// Forest is a green nature-inspired theme.
	Forest Name = "forest"
	// Desert is a warm sand-toned theme.
	Desert Name = "desert"

	// HighContrast is a maximum-contrast accessibility theme.
	HighContrast Name = "high_contrast"
	// OkabeIto is the Okabe-Ito colorblind-safe palette theme.
	OkabeIto Name = "okabe_ito"
	// Viridis is a theme using the viridis perceptual colormap.
	Viridis Name = "viridis"
	// Cividis is a theme using the cividis colorblind-safe colormap.
	Cividis Name = "cividis"

	// Cyberpunk is a neon-on-dark cyberpunk theme.
	Cyberpunk Name = "cyberpunk"
	// Blueprint is a blueprint-paper theme with white-on-blue.
	Blueprint Name = "blueprint"
	// Terminal is a code-terminal theme with monospace type.
	Terminal Name = "terminal"
	// Retro is a vintage parchment-toned theme.
	Retro Name = "retro"
)

// Theme encapsulates the complete visual styling for a plot.
// Elements follow ggplot2's inheritance hierarchy, keyed by dotted path
// names (e.g. "text", "axis.title.x", "panel.background").
type Theme struct {
	// Name is the theme's display name.
	Name string

	// Palette is the discrete color cycle used when the plot has no
	// explicit color scale. May be nil; callers fall back to colormap.Tab10.
	Palette []color.Color

	// Spacing controls margins and inter-element spacing.
	Spacing Spacing

	// Geom holds default visual properties for geometry primitives.
	Geom GeomDefaults

	// Elements maps ggplot2-style dotted paths to Element values.
	// The resolver walks the inheritance chain to fill missing fields.
	Elements map[string]Element
}

// GeomDefaults holds theme-level visual defaults for geometry primitives
// (bars, histograms, areas, boxplots). These act as fallbacks when the user
// has not explicitly overridden a property with a geom option.
type GeomDefaults struct {
	// PatchEdgeColor is the stroke color drawn around filled geoms.
	// A nil value means "darken the fill color" (legacy behaviour).
	PatchEdgeColor color.Color
	// PatchEdgeWidth is the stroke line width for filled geoms (pixels).
	PatchEdgeWidth float64
	// PatchAlpha is the default fill opacity for filled geoms [0,1].
	// Zero is treated as "use geom's built-in default" (typically 0.85).
	PatchAlpha float64
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
	// TickLength is the length of axis tick marks in pixels.
	TickLength float64
	// GridLineCount is the number of major grid lines (0 = auto).
	GridLineCount int
	// GridDashPattern is the dash pattern for grid lines (nil = solid).
	GridDashPattern []float64
}

// --- Inheritance tree ---

// parentOf maps each element path to its parent in the inheritance chain.
// Root elements ("text", "line", "rect") have no parent.
var parentOf = map[string]string{
	"plot.title":        "text",
	"plot.subtitle":     "text",
	"axis.title":        "text",
	"axis.title.x":      "axis.title",
	"axis.title.y":      "axis.title",
	"axis.text":         "text",
	"axis.text.x":       "axis.text",
	"axis.text.y":       "axis.text",
	"legend.title":      "text",
	"legend.text":       "text",
	"strip.text":        "text",
	"annotation.text":   "text",
	"axis.line":         "line",
	"axis.line.x":       "axis.line",
	"axis.line.y":       "axis.line",
	"axis.ticks":        "line",
	"axis.ticks.x":      "axis.ticks",
	"axis.ticks.y":      "axis.ticks",
	"panel.grid.major":  "line",
	"panel.grid.minor":  "line",
	"plot.background":   "rect",
	"panel.background":  "rect",
	"panel.border":      "rect",
	"legend.background": "rect",
	"legend.key":        "rect",
	"strip.background":  "rect",
}

// --- Typed resolvers ---

// resolveText walks the inheritance chain for a text element path,
// merging child fields over parent defaults.
func (t Theme) resolveText(path string) ElementText {
	e, ok := t.Elements[path]
	if ok {
		if IsBlank(e) {
			return ElementText{}
		}

		if et, ok2 := e.(ElementText); ok2 {
			parent, hasParent := parentOf[path]
			if hasParent {
				return MergeText(et, t.resolveText(parent))
			}

			return et
		}
	}

	// No override at this path — inherit from parent.
	if parent, hasParent := parentOf[path]; hasParent {
		return t.resolveText(parent)
	}

	// Root "text" — return zero value (should be set by every preset).
	return ElementText{}
}

// resolveLine walks the inheritance chain for a line element path.
func (t Theme) resolveLine(path string) ElementLine {
	e, ok := t.Elements[path]
	if ok {
		if IsBlank(e) {
			return ElementLine{}
		}

		if el, ok2 := e.(ElementLine); ok2 {
			parent, hasParent := parentOf[path]
			if hasParent {
				return MergeLine(el, t.resolveLine(parent))
			}

			return el
		}
	}

	if parent, hasParent := parentOf[path]; hasParent {
		return t.resolveLine(parent)
	}

	return ElementLine{}
}

// resolveRect walks the inheritance chain for a rect element path.
func (t Theme) resolveRect(path string) ElementRect {
	e, ok := t.Elements[path]
	if ok {
		if IsBlank(e) {
			return ElementRect{}
		}

		if er, ok2 := e.(ElementRect); ok2 {
			parent, hasParent := parentOf[path]
			if hasParent {
				return MergeRect(er, t.resolveRect(parent))
			}

			return er
		}
	}

	if parent, hasParent := parentOf[path]; hasParent {
		return t.resolveRect(parent)
	}

	return ElementRect{}
}

// --- Public accessors ---
// These are the primary API for the rendering pipeline.

// PlotTitle returns the resolved plot title text element.
func (t Theme) PlotTitle() ElementText { return t.resolveText("plot.title") }

// PlotSubtitle returns the resolved plot subtitle text element.
func (t Theme) PlotSubtitle() ElementText { return t.resolveText("plot.subtitle") }

// AxisTitle returns the resolved axis title text element.
func (t Theme) AxisTitle() ElementText { return t.resolveText("axis.title") }

// AxisTitleX returns the resolved X-axis title text element.
func (t Theme) AxisTitleX() ElementText { return t.resolveText("axis.title.x") }

// AxisTitleY returns the resolved Y-axis title text element.
func (t Theme) AxisTitleY() ElementText { return t.resolveText("axis.title.y") }

// AxisTextElem returns the resolved axis tick-label text element.
func (t Theme) AxisTextElem() ElementText { return t.resolveText("axis.text") }

// LegendTitle returns the resolved legend title text element.
func (t Theme) LegendTitle() ElementText { return t.resolveText("legend.title") }

// LegendTextElem returns the resolved legend text element.
func (t Theme) LegendTextElem() ElementText { return t.resolveText("legend.text") }

// StripText returns the resolved facet strip text element.
func (t Theme) StripText() ElementText { return t.resolveText("strip.text") }

// AnnotationText returns the resolved annotation text element.
func (t Theme) AnnotationText() ElementText { return t.resolveText("annotation.text") }

// PlotBackground returns the resolved plot background rect element.
func (t Theme) PlotBackground() ElementRect { return t.resolveRect("plot.background") }

// PanelBackground returns the resolved panel background rect element.
func (t Theme) PanelBackground() ElementRect { return t.resolveRect("panel.background") }

// PanelBorder returns the resolved panel border rect element.
func (t Theme) PanelBorder() ElementRect { return t.resolveRect("panel.border") }

// PanelGridMajor returns the resolved major grid line element.
func (t Theme) PanelGridMajor() ElementLine { return t.resolveLine("panel.grid.major") }

// PanelGridMinor returns the resolved minor grid line element.
func (t Theme) PanelGridMinor() ElementLine { return t.resolveLine("panel.grid.minor") }

// AxisLine returns the resolved axis line element.
func (t Theme) AxisLine() ElementLine { return t.resolveLine("axis.line") }

// AxisTicks returns the resolved axis ticks line element.
func (t Theme) AxisTicks() ElementLine { return t.resolveLine("axis.ticks") }

// LegendBackground returns the resolved legend background rect element.
func (t Theme) LegendBackground() ElementRect { return t.resolveRect("legend.background") }

// LegendKey returns the resolved legend key rect element.
func (t Theme) LegendKey() ElementRect { return t.resolveRect("legend.key") }

// StripBackground returns the resolved facet strip background rect.
func (t Theme) StripBackground() ElementRect { return t.resolveRect("strip.background") }

// --- Registry ---

// Factory builds a Theme on demand.
type Factory func() Theme

var (
	registryMu sync.RWMutex
	registry   = make(map[Name]Factory)
)

// Register adds a named theme factory to the global registry.
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

// Resolve returns a Theme for the given name.
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

// --- Base theme builder ---

// baseTheme returns a neutral starting Theme that preset factories can
// patch. It carries the default inheritance root elements.
func baseTheme(name string) Theme {
	return Theme{
		Name: name,
		Spacing: Spacing{
			MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10,
			PanelSpacing: 10,
			TickLength:   5,
		},
		Geom: GeomDefaults{
			PatchEdgeColor: nil,
			PatchEdgeWidth: 0.5,
			PatchAlpha:     0,
		},
		Elements: map[string]Element{
			// Root text defaults.
			"text": ElementText{Family: "sans-serif", Size: 11, Color: color.Black},
			// Root line defaults.
			"line": ElementLine{Color: gray(60), Size: 0.5},
			// Root rect defaults.
			"rect": ElementRect{Fill: color.White, Color: color.RGBA{R: 200, G: 200, B: 200, A: 255}, Size: 1},

			// Plot-level.
			"plot.title":      ElementText{Size: 14, Bold: true},
			"plot.subtitle":   ElementText{Size: 11, Color: gray(80)},
			"plot.background": ElementRect{Fill: color.White},

			// Panel.
			"panel.background": ElementRect{Fill: color.White},
			"panel.border":     ElementRect{Color: color.RGBA{R: 200, G: 200, B: 200, A: 255}, Size: 1},
			"panel.grid.major": ElementLine{Color: color.RGBA{R: 200, G: 200, B: 200, A: 180}, Size: 0.5},
			"panel.grid.minor": ElementLine{Color: color.RGBA{R: 230, G: 230, B: 230, A: 120}, Size: 0.3},

			// Axis text (tick labels).
			"axis.title": ElementText{Size: 11, Color: gray(40)},
			"axis.text":  ElementText{Size: 10, Color: gray(60)},

			// Axis ticks.
			"axis.ticks": ElementLine{Color: gray(60), Size: 1},

			// Legend.
			"legend.text": ElementText{Size: 10, Color: gray(40)},

			// Annotation.
			"annotation.text": ElementText{Size: 10, Color: gray(60)},
		},
	}
}

// neutralPaletteTheme returns a neutral white-panel theme that only sets a
// custom color palette. Used for themes that differ only in their color cycle.
func neutralPaletteTheme(name string, colors ...color.Color) Theme {
	t := baseTheme(name)
	t.Elements["panel.border"] = ElementRect{Color: gray(180), Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: gray(220), Size: 0.5}
	t.Elements["panel.grid.minor"] = ElementLine{Color: gray(240), Size: 0.3}
	t.Elements["axis.ticks"] = ElementLine{Color: gray(60), Size: 1}
	t.Geom.PatchEdgeColor = gray(220)
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0
	t.Palette = colors

	return t
}

// --- Helpers ---

// gray builds an opaque gray of the given level (0–255).
func gray(v uint8) color.Color {
	return color.RGBA{R: v, G: v, B: v, A: 255}
}

// hex parses a 6-character hex color into an opaque RGBA.
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
