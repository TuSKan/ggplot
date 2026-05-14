package theme

import "image/color"

func init() {
	MustRegister(PaulTol, newPaulTol)
	MustRegister(Few, newFew)
	MustRegister(UCBerkeley, newUCBerkeley)
	MustRegister(Tableau, newTableau)
	MustRegister(Autumn1, newAutumn1)
	MustRegister(Autumn2, newAutumn2)
	MustRegister(Canyon, newCanyon)
	MustRegister(Chili, newChili)
	MustRegister(Tomato, newTomato)
}

func newPaulTol() Theme {
	return neutralPaletteTheme("paul_tol",
		hex("332288"), hex("6699CC"), hex("88CCEE"), hex("44AA99"),
		hex("117733"), hex("999933"), hex("DDCC77"), hex("661100"),
		hex("CC6677"), hex("AA4466"), hex("882255"), hex("AA4499"),
	)
}

func newFew() Theme {
	t := baseTheme("few")
	t.Elements["text"] = ElementText{Family: "sans-serif", Size: 11, Color: color.Black}
	t.Elements["panel.border"] = ElementRect{Color: color.Black, Size: 1}
	t.Elements["panel.grid.major"] = ElementBlank{}
	t.Elements["panel.grid.minor"] = ElementBlank{}
	t.Elements["axis.ticks"] = ElementLine{Color: color.Black, Size: 1}
	t.Geom.PatchEdgeColor = color.Black
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0
	t.Palette = []color.Color{
		hex("4D4D4D"), hex("5DA5DA"), hex("FAA43A"),
		hex("60BD68"), hex("F17CB0"), hex("B2912F"),
		hex("B276B2"), hex("DECF3F"), hex("F15854"),
	}

	return t
}

func newUCBerkeley() Theme {
	t := baseTheme("uc_berkeley")
	t.Elements["panel.border"] = ElementRect{Color: hex("EEEEEE"), Size: 1}
	t.Elements["panel.grid.major"] = ElementLine{Color: hex("EEEEEE"), Size: 0.5}
	t.Elements["axis.ticks"] = ElementLine{Color: gray(60), Size: 1}
	t.Geom.PatchEdgeColor = hex("EEEEEE")
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0
	t.Palette = []color.Color{
		hex("003262"), hex("3B7EA1"), hex("FDB515"), hex("C4820E"),
		hex("D9661F"), hex("EE1F60"), hex("ED4E33"), hex("6C3302"),
		hex("DDD5C7"), hex("00B0DA"), hex("00A598"), hex("46535E"),
		hex("B9D3B6"), hex("CFDD45"), hex("859438"), hex("584F29"),
	}

	return t
}

// newTableau uses classic Tableau10 palette on a neutral white canvas.
func newTableau() Theme {
	return neutralPaletteTheme("tableau",
		hex("1F77B4"), hex("FF7F0E"), hex("2CA02C"), hex("D62728"),
		hex("9467BD"), hex("8C564B"), hex("E377C2"), hex("7F7F7F"),
		hex("BCBD22"), hex("17BECF"),
	)
}

// ── Seasonal / contextual palette variants ──

func seasonalTheme(name Name, palette []color.Color) Theme {
	t := newTableau()
	t.Name = string(name)
	t.Palette = palette

	return t
}

func newAutumn1() Theme {
	return seasonalTheme(Autumn1, []color.Color{
		hex("D1CEC5"), hex("997C67"), hex("755330"),
		hex("B0703C"), hex("DBA72E"), hex("E3CCA1"),
	})
}

func newAutumn2() Theme {
	return seasonalTheme(Autumn2, []color.Color{
		hex("6D7696"), hex("59484F"), hex("455C4F"),
		hex("CC5543"), hex("EDB579"), hex("DBE6AF"),
	})
}

func newCanyon() Theme {
	return seasonalTheme(Canyon, []color.Color{
		hex("6E352C"), hex("CF5230"), hex("F59A44"),
		hex("E3C598"), hex("8A6E64"), hex("6E612F"),
	})
}

func newChili() Theme {
	return seasonalTheme(Chili, []color.Color{
		hex("283811"), hex("66492F"), hex("B8997F"),
		hex("A68887"), hex("D94330"), hex("5C0811"),
	})
}

func newTomato() Theme {
	return seasonalTheme(Tomato, []color.Color{
		hex("D6CFC9"), hex("C2C290"), hex("4A572C"),
		hex("803018"), hex("E34819"), hex("E87F60"),
	})
}
