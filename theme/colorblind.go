package theme

func init() { MustRegister(Colorblind, newColorblind) }

// newColorblind implements the Wong (2011) 8-color colorblind-safe palette.
func newColorblind() Theme {
	return neutralPaletteTheme("colorblind",
		hex("000000"), hex("E69F00"), hex("56B4E9"), hex("009E73"),
		hex("F0E442"), hex("0072B2"), hex("D55E00"), hex("CC79A7"),
	)
}
