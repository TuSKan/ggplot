package theme

import "image/color"

func init() {
	MustRegister(SolarizeLight, newSolarizeLight)
}

// newSolarizeLight implements the pyplot-themes "Solarized light" scheme.
// It uses the same warm-toned Solarized colors as solarize_light2 for the
// chrome, but swaps the palette to the 8 Solarized base tones from light
// to dark — giving plots a cohesive, monochromatic Solarized feel.
//
// Source: raybuhr/pyplot-themes — palettes.Solarized.light
// (which is Solarized.dark reversed: FDF6E3 → EEE8D5 → ... → 002B36)
func newSolarizeLight() Theme {
	// Reuse the solarize_light2 chrome (same warm-beige panel) …
	t := newSolarizeLight2()
	t.Name = "solarize_light"

	// … but swap to the Solarized base tones (light → dark) as the palette.
	// These are the Solarized base colors in reverse order to pyplot-themes.
	t.Palette = []color.Color{
		hex("FDF6E3"), // base3  — lightest
		hex("EEE8D5"), // base2
		hex("93A1A1"), // base1
		hex("839496"), // base0
		hex("657B83"), // base00
		hex("586E75"), // base01
		hex("073642"), // base02
		hex("002B36"), // base03 — darkest
	}
	return t
}
