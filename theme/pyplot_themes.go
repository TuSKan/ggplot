package theme

import "image/color"

func init() {
	MustRegister(PaulTol, newPaulTol)
	MustRegister(Few, newFew)
	MustRegister(UCBerkeley, newUCBerkeley)
	// Seasonal / contextual palette variants (pyplot-themes originals).
	MustRegister(Autumn1, newAutumn1)
	MustRegister(Autumn2, newAutumn2)
	MustRegister(Canyon, newCanyon)
	MustRegister(Chili, newChili)
	MustRegister(Tomato, newTomato)
}

// Themes contributed by raybuhr/pyplot-themes that don't overlap with
// matplotlib's stylelib.
//
// Source: github.com/raybuhr/pyplot-themes/blob/master/pyplot_themes/palettes.py

// newPaulTol uses Paul Tol's qualitative scheme (12 colors), white panel,
// light grid.
func newPaulTol() Theme {
	t := baseTheme("paul_tol")
	t.Background = color.White
	t.Panel.Background = color.White
	t.Panel.Border = gray(180)
	t.Panel.BorderWidth = 1
	t.Grid.MajorColor = gray(220)
	t.Grid.MajorWidth = 0.5
	t.Grid.DashPattern = nil
	t.Ticks.Color = gray(60)

	// paul_tol: white panel + gray (#EEEEEE) grid → light gray edge.
	t.Geom.PatchEdgeColor = gray(220)
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	t.Palette = []color.Color{
		hex("332288"), hex("6699CC"), hex("88CCEE"), hex("44AA99"),
		hex("117733"), hex("999933"), hex("DDCC77"), hex("661100"),
		hex("CC6677"), hex("AA4466"), hex("882255"), hex("AA4499"),
	}

	return t
}

// newFew uses the "Few medium" palette (Stephen Few's recommended
// categorical colors), white panel, no grid by default.
func newFew() Theme {
	t := baseTheme("few")
	t.Background = color.White
	t.Panel.Background = color.White
	t.Panel.Border = color.Black
	t.Panel.BorderWidth = 1
	t.Grid.MajorColor = color.Transparent
	t.Grid.MajorWidth = 0
	t.Grid.MinorColor = color.Transparent
	t.Grid.MinorWidth = 0
	t.Ticks.Color = color.Black

	t.Text.Title.Color = color.Black
	t.Text.Subtitle.Color = color.Black
	t.Text.AxisTitle.Color = color.Black
	t.Text.TickLabel.Color = color.Black
	t.Text.Legend.Color = color.Black

	// few: white panel, no grid, black border → black edges match the axis style.
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

// newUCBerkeley uses UC Berkeley's official palette, white panel, very
// light gray edges.
func newUCBerkeley() Theme {
	t := baseTheme("uc_berkeley")
	t.Background = color.White
	t.Panel.Background = color.White
	t.Panel.Border = hex("EEEEEE")
	t.Panel.BorderWidth = 1
	t.Grid.MajorColor = hex("EEEEEE")
	t.Grid.MajorWidth = 0.5
	t.Grid.DashPattern = nil
	t.Ticks.Color = gray(60)

	// uc_berkeley: white panel + very light (#EEEEEE) grid → same grid color as edge.
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

// newTableau uses matplotlib's classic Tableau10 palette (the upstream
// default cycle, identical to colormap.Tab10) on a white canvas.
func newTableau() Theme {
	t := baseTheme("tableau")
	t.Background = color.White
	t.Panel.Background = color.White
	t.Panel.Border = gray(180)
	t.Panel.BorderWidth = 1
	t.Grid.MajorColor = gray(220)
	t.Grid.MajorWidth = 0.5
	t.Grid.DashPattern = nil
	t.Ticks.Color = gray(60)

	// tableau: white panel + gray (#DDDDDD) grid → light gray edge.
	t.Geom.PatchEdgeColor = gray(220)
	t.Geom.PatchEdgeWidth = 0.5
	t.Geom.PatchAlpha = 1.0

	t.Palette = []color.Color{
		hex("1F77B4"), hex("FF7F0E"), hex("2CA02C"), hex("D62728"),
		hex("9467BD"), hex("8C564B"), hex("E377C2"), hex("7F7F7F"),
		hex("BCBD22"), hex("17BECF"),
	}

	return t
}

// ── Seasonal / contextual palette variants ──────────────────────────────────
// All five share the tableau chrome (white panel, light gray grid).
// Palettes verbatim from: raybuhr/pyplot-themes palettes.py
// https://duoparadigms.com/2013/10/11/10-color-palettes-perfect-autumnfall-season/

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
