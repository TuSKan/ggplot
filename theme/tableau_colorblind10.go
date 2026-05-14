package theme

func init() { MustRegister(TableauColorblind10, newTableauColorblind10) }

// newTableauColorblind10 mirrors matplotlib's tableau-colorblind10.mplstyle.
func newTableauColorblind10() Theme {
	return neutralPaletteTheme("tableau_colorblind10",
		hex("006BA4"), hex("FF800E"), hex("ABABAB"), hex("595959"), hex("5F9ED1"),
		hex("C85200"), hex("898989"), hex("A2C8EC"), hex("FFBC79"), hex("CFCFCF"),
	)
}
