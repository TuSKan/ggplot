package theme

import "github.com/TuSKan/ggplot/colormap"

// ColorDefaults maps a theme to its recommended colormaps by aesthetic and
// data category. The scale pipeline consults these when the user hasn't
// explicitly set a colormap, so that dark themes get bright-ended ramps,
// accessibility themes get colorblind-safe maps, and editorial themes get
// restrained neutrals.
//
// Color* fields are used for stroke/outline aesthetics (aes.Color).
// Fill* fields are used for fill aesthetics (aes.Fill).
// Zero-value Fill* fields fall back to the corresponding Color* field.
type ColorDefaults struct {
	// ColorDiscrete is the qualitative palette for categorical color data.
	ColorDiscrete colormap.Cmap
	// ColorSequential is the continuous ramp for ordered color data.
	ColorSequential colormap.Cmap
	// FillDiscrete is the qualitative palette for categorical fill data.
	// Falls back to ColorDiscrete when nil.
	FillDiscrete colormap.Cmap
	// FillSequential is the continuous ramp for ordered fill data.
	// Falls back to ColorSequential when nil.
	FillSequential colormap.Cmap
	// Diverging emphasises departure from a meaningful midpoint.
	Diverging colormap.Cmap
	// Cyclic wraps so that 0 and 1 are the same color (phase, angle).
	Cyclic colormap.Cmap
}

// defaultColorDefaults is the fallback used when a theme has no entry in the
// themeColorDefaults table.
var defaultColorDefaults = ColorDefaults{
	ColorDiscrete:   colormap.Tab10,
	ColorSequential: colormap.Viridis,
	FillDiscrete:    colormap.Tab10,
	FillSequential:  colormap.Blues,
	Diverging:       colormap.RdBu,
	Cyclic:          colormap.Twilight,
}

// themeColorDefaults maps theme names to sensible default colormaps.
// Selection logic:
//   - Dark backgrounds → Magma/Inferno/Plasma (bright high-end pops)
//   - Light backgrounds → Viridis/Blues (dark low-end visible on white)
//   - Accessibility → Cividis (deuteranopia + protanopia safe)
//   - Editorial/print → Greys (B&W safe)
//   - Cyclic → always Twilight (only perceptually-uniform cyclic map)
//
//nolint:gochecknoglobals // Immutable theme-level configuration table.
var themeColorDefaults = map[Name]ColorDefaults{
	// ── Light backgrounds ──────────────────────────────────────────────
	Default:             cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Ggplot:              cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Classic:             cd(colormap.Tab10, colormap.Blues, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Minimal:             cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	BW:                  cd(colormap.Tab10, colormap.Greys, colormap.Tab10, colormap.Greys, colormap.RdGy),
	Bmh:                 cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Fast:                cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Observable:          cd(colormap.Observable10, colormap.Viridis, colormap.Observable10, colormap.Blues, colormap.RdBu),
	Dashboard:           cd(colormap.Tab10, colormap.Blues, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Quartz:              cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Air:                 cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Tableau:             cd(colormap.Tab10, colormap.Blues, colormap.Tab10, colormap.Blues, colormap.RdBu),
	TableauColorblind10: cd(colormap.Tab10, colormap.Blues, colormap.Tab10, colormap.Blues, colormap.RdBu),
	NASA:                cd(colormap.Tab10, colormap.Blues, colormap.Tab10, colormap.Blues, colormap.RdBu),
	GitHubLight:         cd(colormap.Tab10, colormap.Blues, colormap.Tab10, colormap.Blues, colormap.RdBu),
	GruvboxLight:        cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Newsroom:            cd(colormap.Tab10, colormap.Blues, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Ocean:               cd(colormap.Tab10, colormap.Blues, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Earth:               cd(colormap.Tab10, colormap.YlOrBr, colormap.Tab10, colormap.YlOrBr, colormap.BrBG),
	Forest:              cd(colormap.Tab10, colormap.Greens, colormap.Tab10, colormap.Greens, colormap.BrBG),
	Desert:              cd(colormap.Tab10, colormap.Oranges, colormap.Tab10, colormap.Oranges, colormap.BrBG),
	Retro:               cd(colormap.Tab10, colormap.YlOrBr, colormap.Tab10, colormap.YlOrBr, colormap.RdBu),

	// ── Seaborn family (light) ─────────────────────────────────────────
	SeabornWhite:      cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Seaborn:           cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SeabornWhitegrid:  cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SeabornDarkgrid:   cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SeabornTicks:      cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SeabornPaper:      cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SeabornNotebook:   cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SeabornTalk:       cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SeabornPoster:     cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SeabornDeep:       cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SeabornMuted:      cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SeabornPastel:     cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SeabornBright:     cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SeabornColorblind: cd(colormap.OkabeIto, colormap.Cividis, colormap.OkabeIto, colormap.Cividis, colormap.BrBG),

	// ── Dark backgrounds ───────────────────────────────────────────────
	Dark:           cd(colormap.Observable10, colormap.Plasma, colormap.Observable10, colormap.Inferno, colormap.Coolwarm),
	DarkBackground: cd(colormap.Observable10, colormap.Inferno, colormap.Observable10, colormap.Magma, colormap.Coolwarm),
	ObservableDark: cd(colormap.Observable10, colormap.Plasma, colormap.Observable10, colormap.Inferno, colormap.RdBu),
	Ink:            cd(colormap.Observable10, colormap.Plasma, colormap.Observable10, colormap.Inferno, colormap.Coolwarm),
	Nord:           cd(colormap.Observable10, colormap.Magma, colormap.Observable10, colormap.Inferno, colormap.Coolwarm),
	Dracula:        cd(colormap.Observable10, colormap.Magma, colormap.Observable10, colormap.Inferno, colormap.Coolwarm),
	GruvboxDark:    cd(colormap.Observable10, colormap.Inferno, colormap.Observable10, colormap.Magma, colormap.RdBu),
	GitHubDark:     cd(colormap.Observable10, colormap.Plasma, colormap.Observable10, colormap.Inferno, colormap.RdBu),
	SolarizeDark:   cd(colormap.Observable10, colormap.Plasma, colormap.Observable10, colormap.Inferno, colormap.Coolwarm),
	AstronomyDark:  cd(colormap.Observable10, colormap.Magma, colormap.Observable10, colormap.Magma, colormap.Spectral),
	Cyberpunk:      cd(colormap.Observable10, colormap.Inferno, colormap.Observable10, colormap.Magma, colormap.Spectral),
	Blueprint:      cd(colormap.Observable10, colormap.Plasma, colormap.Observable10, colormap.Inferno, colormap.Coolwarm),
	Terminal:       cd(colormap.Observable10, colormap.Plasma, colormap.Observable10, colormap.Inferno, colormap.Coolwarm),

	// ── Seaborn dark ───────────────────────────────────────────────────
	SeabornDark:        cd(colormap.Observable10, colormap.Plasma, colormap.Observable10, colormap.Inferno, colormap.Coolwarm),
	SeabornDarkPalette: cd(colormap.Observable10, colormap.Plasma, colormap.Observable10, colormap.Inferno, colormap.Coolwarm),

	// ── Solarized ──────────────────────────────────────────────────────
	SolarizeLight:   cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	SolarizeLight2:  cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Fivethirtyeight: cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Grayscale:       cd(colormap.OkabeIto, colormap.Greys, colormap.OkabeIto, colormap.Greys, colormap.RdGy),

	// ── Editorial / publication ─────────────────────────────────────────
	Tufte:      cd(colormap.OkabeIto, colormap.Greys, colormap.OkabeIto, colormap.Greys, colormap.RdGy),
	Academic:   cd(colormap.OkabeIto, colormap.Cividis, colormap.OkabeIto, colormap.Cividis, colormap.RdBu),
	Editorial:  cd(colormap.OkabeIto, colormap.Greys, colormap.OkabeIto, colormap.Greys, colormap.RdGy),
	Monochrome: cd(colormap.OkabeIto, colormap.Greys, colormap.OkabeIto, colormap.Greys, colormap.RdGy),

	// ── Accessibility / perceptual ──────────────────────────────────────
	HighContrast: cd(colormap.OkabeIto, colormap.Cividis, colormap.OkabeIto, colormap.Cividis, colormap.RdBu),
	OkabeIto:     cd(colormap.OkabeIto, colormap.Cividis, colormap.OkabeIto, colormap.Cividis, colormap.BrBG),
	Colorblind:   cd(colormap.OkabeIto, colormap.Cividis, colormap.OkabeIto, colormap.Cividis, colormap.BrBG),
	Viridis:      cd(colormap.OkabeIto, colormap.Viridis, colormap.OkabeIto, colormap.Viridis, colormap.RdBu),
	Cividis:      cd(colormap.OkabeIto, colormap.Cividis, colormap.OkabeIto, colormap.Cividis, colormap.BrBG),

	// ── Palette-only themes ─────────────────────────────────────────────
	PaulTol:    cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Few:        cd(colormap.Tab10, colormap.Blues, colormap.Tab10, colormap.Blues, colormap.RdBu),
	FewLight:   cd(colormap.Tab10, colormap.Blues, colormap.Tab10, colormap.Blues, colormap.RdBu),
	FewDark:    cd(colormap.Observable10, colormap.Plasma, colormap.Observable10, colormap.Inferno, colormap.Coolwarm),
	UCBerkeley: cd(colormap.Tab10, colormap.Blues, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Petroff10:  cd(colormap.Tab10, colormap.Viridis, colormap.Tab10, colormap.Blues, colormap.RdBu),
	Autumn1:    cd(colormap.Tab10, colormap.YlOrBr, colormap.Tab10, colormap.YlOrBr, colormap.RdBu),
	Autumn2:    cd(colormap.Tab10, colormap.YlOrBr, colormap.Tab10, colormap.YlOrBr, colormap.RdBu),
	Canyon:     cd(colormap.Tab10, colormap.Oranges, colormap.Tab10, colormap.Oranges, colormap.RdBu),
	Chili:      cd(colormap.Tab10, colormap.Reds, colormap.Tab10, colormap.Reds, colormap.RdBu),
	Tomato:     cd(colormap.Tab10, colormap.Reds, colormap.Tab10, colormap.Reds, colormap.RdBu),
}

// cd is a shorthand constructor for ColorDefaults. Cyclic is always Twilight.
func cd(colorDisc, colorSeq, fillDisc, fillSeq, div colormap.Cmap) ColorDefaults {
	return ColorDefaults{
		ColorDiscrete:   colorDisc,
		ColorSequential: colorSeq,
		FillDiscrete:    fillDisc,
		FillSequential:  fillSeq,
		Diverging:       div,
		Cyclic:          colormap.Twilight,
	}
}

// DefaultColorDefaults returns the ColorDefaults for the given theme.
// If the theme has no explicit entry, the package-level defaults are returned
// (Tab10 discrete, Viridis sequential, Blues fill-sequential, RdBu diverging,
// Twilight cyclic).
func DefaultColorDefaults(name Name) ColorDefaults {
	if cd, ok := themeColorDefaults[name]; ok {
		return cd
	}

	return defaultColorDefaults
}

// Aesthetic identifies the color or fill aesthetic channel.
type Aesthetic string

const (
	// AesColor is the stroke/outline color aesthetic.
	AesColor Aesthetic = "color"
	// AesFill is the fill color aesthetic.
	AesFill Aesthetic = "fill"
)

// DefaultCmapFor returns the recommended Cmap for the given theme, aesthetic,
// and colormap category. This is the primary integration point for the scale
// pipeline: when a user hasn't set an explicit colormap, the scale can call
//
//	cmap := theme.DefaultCmapFor(th.Name, theme.AesColor, colormap.Qualitative)
//
// and get a sensible default that matches the theme's visual language.
func DefaultCmapFor(name Name, aes Aesthetic, cat colormap.Category) colormap.Cmap {
	d := DefaultColorDefaults(name)

	switch cat {
	case colormap.Qualitative:
		if aes == AesFill && d.FillDiscrete != nil {
			return d.FillDiscrete
		}

		return d.ColorDiscrete
	case colormap.Sequential, colormap.PerceptuallyUniform:
		if aes == AesFill && d.FillSequential != nil {
			return d.FillSequential
		}

		return d.ColorSequential
	case colormap.Diverging:
		return d.Diverging
	case colormap.Cyclic:
		return d.Cyclic
	case colormap.Miscellaneous:
		if aes == AesFill && d.FillSequential != nil {
			return d.FillSequential
		}

		return d.ColorSequential
	default:
		return d.ColorSequential
	}
}
