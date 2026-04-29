package theme

import "image/color"

func init() {
	MustRegister(SeabornDarkgrid, newSeabornDarkgrid)
	MustRegister(Seaborn, newSeabornDarkgrid) // Seaborn is an alias for SeabornDarkgrid.
	MustRegister(SeabornWhitegrid, newSeabornWhitegrid)
	MustRegister(SeabornDark, newSeabornDark)
	MustRegister(SeabornWhite, newSeabornWhite)
	MustRegister(SeabornTicks, newSeabornTicks)
	MustRegister(SeabornDeep, newSeabornDeep)
	MustRegister(SeabornMuted, newSeabornMuted)
	MustRegister(SeabornBright, newSeabornBright)
	MustRegister(SeabornColorblind, newSeabornColorblind)
	MustRegister(SeabornPastel, newSeabornPastel)
	MustRegister(SeabornDarkPalette, newSeabornDarkPalette)
	MustRegister(SeabornPaper, newSeabornPaper)
	MustRegister(SeabornNotebook, newSeabornNotebook)
	MustRegister(SeabornTalk, newSeabornTalk)
	MustRegister(SeabornPoster, newSeabornPoster)
}

// Seaborn family. Sources:
//   matplotlib/lib/matplotlib/mpl-data/stylelib/seaborn-v0_8*.mplstyle
//
// Seaborn's grid variants share a common palette and typography; only
// the panel background, axes edge, and grid visibility differ. The
// palette variants share the chrome of seaborn-darkgrid (the seaborn
// default) and only swap the color cycle. The font-size variants share
// chrome and palette and only scale text sizes.

// seabornBase returns the chrome shared by all seaborn presets,
// patched per-variant by the factories below. Palette defaults to the
// seaborn "deep" cycle (the upstream default), which palette variants
// override.
func seabornBase(name string) Theme {
	t := baseTheme(name)

	t.Background = color.White

	// xtick.color / ytick.color: .15 (38/255)
	tickAndText := gray(38)

	t.Text.Title = FontConfig{Family: "sans-serif", Size: 12, Color: tickAndText, Bold: true}
	t.Text.Subtitle = FontConfig{Family: "sans-serif", Size: 11, Color: tickAndText}
	t.Text.AxisTitle = FontConfig{Family: "sans-serif", Size: 11, Color: tickAndText}
	t.Text.TickLabel = FontConfig{Family: "sans-serif", Size: 10, Color: tickAndText}
	t.Text.Legend = FontConfig{Family: "sans-serif", Size: 10, Color: tickAndText}
	t.Text.Annotation = FontConfig{Family: "sans-serif", Size: 10, Color: tickAndText}

	t.Ticks.Color = tickAndText
	t.Ticks.Length = 4
	t.Ticks.Width = 1

	t.Palette = seabornDeepPalette()
	return t
}

func seabornDeepPalette() []color.Color {
	return []color.Color{
		hex("4C72B0"), hex("55A868"), hex("C44E52"),
		hex("8172B2"), hex("CCB974"), hex("64B5CD"),
	}
}

// --- chrome variants ---

// newSeabornDarkgrid: gray panel (#EAEAF2), white grid, white edges.
// This is seaborn's default style and what `Seaborn` resolves to.
func newSeabornDarkgrid() Theme {
	t := seabornBase("seaborn_darkgrid")
	t.Panel.Background = hex("EAEAF2")
	t.Panel.Border = color.White
	t.Panel.BorderWidth = 0
	t.Grid.MajorColor = color.White
	t.Grid.MajorWidth = 1
	t.Grid.MinorColor = hexA("FFFFFF", 160)
	t.Grid.MinorWidth = 0.5
	t.Grid.DashPattern = nil
	return t
}

// newSeabornWhitegrid: white panel, light gray (.8) grid + edges.
func newSeabornWhitegrid() Theme {
	t := seabornBase("seaborn_whitegrid")
	t.Panel.Background = color.White
	t.Panel.Border = gray(204)
	t.Panel.BorderWidth = 1
	t.Grid.MajorColor = gray(204)
	t.Grid.MajorWidth = 0.5
	t.Grid.MinorColor = gray(230)
	t.Grid.MinorWidth = 0.3
	t.Grid.DashPattern = nil
	return t
}

// newSeabornDark: gray panel, no grid by default, dark edges.
func newSeabornDark() Theme {
	t := seabornBase("seaborn_dark")
	t.Panel.Background = hex("EAEAF2")
	t.Panel.Border = color.White
	t.Panel.BorderWidth = 0
	t.Grid.MajorColor = color.Transparent
	t.Grid.MajorWidth = 0
	t.Grid.MinorColor = color.Transparent
	t.Grid.MinorWidth = 0
	return t
}

// newSeabornWhite: white panel, no grid, dark (.15) spines.
func newSeabornWhite() Theme {
	t := seabornBase("seaborn_white")
	t.Panel.Background = color.White
	t.Panel.Border = gray(38)
	t.Panel.BorderWidth = 1
	t.Grid.MajorColor = color.Transparent
	t.Grid.MajorWidth = 0
	t.Grid.MinorColor = color.Transparent
	t.Grid.MinorWidth = 0
	return t
}

// newSeabornTicks: like white but ticks point outward and are visible.
func newSeabornTicks() Theme {
	t := newSeabornWhite()
	t.Name = "seaborn_ticks"
	t.Ticks.Length = 6
	t.Ticks.Width = 1
	return t
}

// --- palette variants (chrome = seaborn-darkgrid) ---

func newSeabornDeep() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_deep"
	t.Palette = seabornDeepPalette()
	return t
}

func newSeabornMuted() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_muted"
	t.Palette = []color.Color{
		hex("4878CF"), hex("6ACC65"), hex("D65F5F"),
		hex("B47CC7"), hex("C4AD66"), hex("77BEDB"),
	}
	return t
}

func newSeabornBright() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_bright"
	t.Palette = []color.Color{
		hex("003FFF"), hex("03ED3A"), hex("E8000B"),
		hex("8A2BE2"), hex("FFC400"), hex("00D7FF"),
	}
	return t
}

func newSeabornColorblind() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_colorblind"
	t.Palette = []color.Color{
		hex("0072B2"), hex("009E73"), hex("D55E00"),
		hex("CC79A7"), hex("F0E442"), hex("56B4E9"),
	}
	return t
}

func newSeabornPastel() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_pastel"
	t.Palette = []color.Color{
		hex("92C6FF"), hex("97F0AA"), hex("FF9F9A"),
		hex("D0BBFF"), hex("FFFEA3"), hex("B0E0E6"),
	}
	return t
}

func newSeabornDarkPalette() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_dark_palette"
	t.Palette = []color.Color{
		hex("001C7F"), hex("017517"), hex("8C0900"),
		hex("7600A1"), hex("B8860B"), hex("006374"),
	}
	return t
}

// --- font-size variants (chrome = seaborn-darkgrid, palette = deep) ---

func scaleSeabornFonts(t Theme, base, label float64) Theme {
	t.Text.Title.Size = base + 2
	t.Text.Subtitle.Size = base
	t.Text.AxisTitle.Size = label
	t.Text.TickLabel.Size = label - 1
	t.Text.Legend.Size = label - 1
	t.Text.Annotation.Size = label - 1
	return t
}

func newSeabornPaper() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_paper"
	return scaleSeabornFonts(t, 9, 8.4)
}

func newSeabornNotebook() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_notebook"
	return scaleSeabornFonts(t, 12, 11)
}

func newSeabornTalk() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_talk"
	return scaleSeabornFonts(t, 15, 13.5)
}

func newSeabornPoster() Theme {
	t := newSeabornDarkgrid()
	t.Name = "seaborn_poster"
	return scaleSeabornFonts(t, 18, 16.5)
}
