package theme

func init() { MustRegister(Petroff10, newPetroff10) }

// newPetroff10 mirrors matplotlib's petroff10.mplstyle (matplotlib 3.10).
func newPetroff10() Theme {
	return neutralPaletteTheme("petroff10",
		hex("3f90da"), hex("ffa90e"), hex("bd1f01"), hex("94a4a2"), hex("832db6"),
		hex("a96b59"), hex("e76300"), hex("b9ac70"), hex("717581"), hex("92dadd"),
	)
}
