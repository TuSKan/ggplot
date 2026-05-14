package theme

import "image/color"

func init() { MustRegister(SolarizeLight, newSolarizeLight) }

// newSolarizeLight implements the pyplot-themes "Solarized light" scheme.
func newSolarizeLight() Theme {
	t := newSolarizeLight2()
	t.Name = "solarize_light"

	t.Palette = []color.Color{
		hex("FDF6E3"), hex("EEE8D5"), hex("93A1A1"), hex("839496"),
		hex("657B83"), hex("586E75"), hex("073642"), hex("002B36"),
	}

	return t
}
