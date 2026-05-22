package theme

import "github.com/TuSKan/ggplot/colormap"

// ColorDefaults maps a theme to its recommended colormaps by usage category.
// The scale pipeline consults these when the user hasn't explicitly set a
// colormap, so that dark themes get bright-ended ramps, accessibility
// themes get colorblind-safe maps, and editorial themes get restrained
// neutrals.
type ColorDefaults struct {
	// Discrete is the qualitative palette for categorical data.
	Discrete colormap.Cmap
	// Sequential is the continuous ramp for ordered data.
	Sequential colormap.Cmap
	// Diverging emphasises departure from a meaningful midpoint.
	Diverging colormap.Cmap
	// Cyclic wraps so that 0 and 1 are the same color (phase, angle).
	Cyclic colormap.Cmap
}

// defaultColorDefaults is the fallback used when a theme has no entry in the
// themeColorDefaults table.
var defaultColorDefaults = ColorDefaults{
	Discrete:   colormap.Tab10,
	Sequential: colormap.Viridis,
	Diverging:  colormap.RdBu,
	Cyclic:     colormap.Twilight,
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
	Default:             {colormap.Tab10, colormap.Blues, colormap.RdBu, colormap.Twilight},
	Ggplot:              {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	Classic:             {colormap.Tab10, colormap.Blues, colormap.RdBu, colormap.Twilight},
	Minimal:             {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	BW:                  {colormap.Tab10, colormap.Greys, colormap.RdGy, colormap.Twilight},
	Bmh:                 {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	Fast:                {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	Observable:          {colormap.Observable10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	Dashboard:           {colormap.Tab10, colormap.Blues, colormap.RdBu, colormap.Twilight},
	Quartz:              {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	Air:                 {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	Tableau:             {colormap.Tab10, colormap.Blues, colormap.RdBu, colormap.Twilight},
	TableauColorblind10: {colormap.Tab10, colormap.Blues, colormap.RdBu, colormap.Twilight},
	NASA:                {colormap.Tab10, colormap.Blues, colormap.RdBu, colormap.Twilight},
	GitHubLight:         {colormap.Tab10, colormap.Blues, colormap.RdBu, colormap.Twilight},
	GruvboxLight:        {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	Newsroom:            {colormap.Tab10, colormap.Blues, colormap.RdBu, colormap.Twilight},
	Ocean:               {colormap.Tab10, colormap.Blues, colormap.RdBu, colormap.Twilight},
	Earth:               {colormap.Tab10, colormap.YlOrBr, colormap.BrBG, colormap.Twilight},
	Forest:              {colormap.Tab10, colormap.Greens, colormap.BrBG, colormap.Twilight},
	Desert:              {colormap.Tab10, colormap.Oranges, colormap.BrBG, colormap.Twilight},
	Retro:               {colormap.Tab10, colormap.YlOrBr, colormap.RdBu, colormap.Twilight},

	// ── Seaborn family (light) ─────────────────────────────────────────
	Seaborn:           {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornWhite:      {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornWhitegrid:  {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornDarkgrid:   {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornTicks:      {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornPaper:      {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornNotebook:   {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornTalk:       {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornPoster:     {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornDeep:       {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornMuted:      {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornPastel:     {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornBright:     {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SeabornColorblind: {colormap.OkabeIto, colormap.Cividis, colormap.BrBG, colormap.Twilight},

	// ── Dark backgrounds ───────────────────────────────────────────────
	Dark:           {colormap.Observable10, colormap.Plasma, colormap.Coolwarm, colormap.Twilight},
	DarkBackground: {colormap.Observable10, colormap.Inferno, colormap.Coolwarm, colormap.Twilight},
	ObservableDark: {colormap.Observable10, colormap.Plasma, colormap.RdBu, colormap.Twilight},
	Ink:            {colormap.Observable10, colormap.Plasma, colormap.Coolwarm, colormap.Twilight},
	Nord:           {colormap.Observable10, colormap.Magma, colormap.Coolwarm, colormap.Twilight},
	Dracula:        {colormap.Observable10, colormap.Magma, colormap.Coolwarm, colormap.Twilight},
	GruvboxDark:    {colormap.Observable10, colormap.Inferno, colormap.RdBu, colormap.Twilight},
	GitHubDark:     {colormap.Observable10, colormap.Plasma, colormap.RdBu, colormap.Twilight},
	SolarizeDark:   {colormap.Observable10, colormap.Plasma, colormap.Coolwarm, colormap.Twilight},
	AstronomyDark:  {colormap.Observable10, colormap.Magma, colormap.Spectral, colormap.Twilight},
	Cyberpunk:      {colormap.Observable10, colormap.Inferno, colormap.Spectral, colormap.Twilight},
	Blueprint:      {colormap.Observable10, colormap.Plasma, colormap.Coolwarm, colormap.Twilight},
	Terminal:       {colormap.Observable10, colormap.Plasma, colormap.Coolwarm, colormap.Twilight},

	// ── Seaborn dark ───────────────────────────────────────────────────
	SeabornDark:        {colormap.Observable10, colormap.Plasma, colormap.Coolwarm, colormap.Twilight},
	SeabornDarkPalette: {colormap.Observable10, colormap.Plasma, colormap.Coolwarm, colormap.Twilight},

	// ── Solarized ──────────────────────────────────────────────────────
	SolarizeLight:   {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	SolarizeLight2:  {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	Fivethirtyeight: {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	Grayscale:       {colormap.OkabeIto, colormap.Greys, colormap.RdGy, colormap.Twilight},

	// ── Editorial / publication ─────────────────────────────────────────
	Tufte:      {colormap.OkabeIto, colormap.Greys, colormap.RdGy, colormap.Twilight},
	Academic:   {colormap.OkabeIto, colormap.Cividis, colormap.RdBu, colormap.Twilight},
	Editorial:  {colormap.OkabeIto, colormap.Greys, colormap.RdGy, colormap.Twilight},
	Monochrome: {colormap.OkabeIto, colormap.Greys, colormap.RdGy, colormap.Twilight},

	// ── Accessibility / perceptual ──────────────────────────────────────
	HighContrast: {colormap.OkabeIto, colormap.Cividis, colormap.RdBu, colormap.Twilight},
	OkabeIto:     {colormap.OkabeIto, colormap.Cividis, colormap.BrBG, colormap.Twilight},
	Colorblind:   {colormap.OkabeIto, colormap.Cividis, colormap.BrBG, colormap.Twilight},
	Viridis:      {colormap.OkabeIto, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	Cividis:      {colormap.OkabeIto, colormap.Cividis, colormap.BrBG, colormap.Twilight},

	// ── Palette-only themes ─────────────────────────────────────────────
	PaulTol:    {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	Few:        {colormap.Tab10, colormap.Blues, colormap.RdBu, colormap.Twilight},
	FewLight:   {colormap.Tab10, colormap.Blues, colormap.RdBu, colormap.Twilight},
	FewDark:    {colormap.Observable10, colormap.Plasma, colormap.Coolwarm, colormap.Twilight},
	UCBerkeley: {colormap.Tab10, colormap.Blues, colormap.RdBu, colormap.Twilight},
	Petroff10:  {colormap.Tab10, colormap.Viridis, colormap.RdBu, colormap.Twilight},
	Autumn1:    {colormap.Tab10, colormap.YlOrBr, colormap.RdBu, colormap.Twilight},
	Autumn2:    {colormap.Tab10, colormap.YlOrBr, colormap.RdBu, colormap.Twilight},
	Canyon:     {colormap.Tab10, colormap.Oranges, colormap.RdBu, colormap.Twilight},
	Chili:      {colormap.Tab10, colormap.Reds, colormap.RdBu, colormap.Twilight},
	Tomato:     {colormap.Tab10, colormap.Reds, colormap.RdBu, colormap.Twilight},
}

// DefaultColorDefaults returns the ColorDefaults for the given theme.
// If the theme has no explicit entry, the package-level defaults are returned
// (Tab10 discrete, Viridis sequential, RdBu diverging, Twilight cyclic).
func DefaultColorDefaults(name Name) ColorDefaults {
	if cd, ok := themeColorDefaults[name]; ok {
		return cd
	}

	return defaultColorDefaults
}

// DefaultCmapFor returns the recommended Cmap for the given theme and
// colormap category. This is the primary integration point for the scale
// pipeline: when a user hasn't set an explicit colormap, the scale can call
//
//	cmap := theme.DefaultCmapFor(th.Name, colormap.Sequential)
//
// and get a sensible default that matches the theme's visual language.
func DefaultCmapFor(name Name, cat colormap.Category) colormap.Cmap {
	cd := DefaultColorDefaults(name)

	switch cat {
	case colormap.Qualitative:
		return cd.Discrete
	case colormap.Sequential, colormap.PerceptuallyUniform:
		return cd.Sequential
	case colormap.Diverging:
		return cd.Diverging
	case colormap.Cyclic:
		return cd.Cyclic
	case colormap.Miscellaneous:
		return cd.Sequential
	default:
		return cd.Sequential
	}
}
