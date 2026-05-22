package theme

import "testing"

// TestResolveAllNames checks that every Name constant returned by
// AllNames resolves cleanly and produces a Theme whose Name field
// matches the requested constant. This guards against a new preset
// being added to the constant block but forgotten in the Resolve
// switch.
func TestResolveAllNames(t *testing.T) {
	t.Parallel()

	for _, n := range AllNames() {
		th, err := Resolve(n)
		if err != nil {
			t.Errorf("Resolve(%q) returned error: %v", n, err)
			continue
		}
		// Default is an alias for dashboard, so its resolved Name field
		// is "dashboard" — accept that one mismatch.
		if n == Default {
			if th.Name != string(Dashboard) {
				t.Errorf("Resolve(Default) Name = %q, want %q", th.Name, Dashboard)
			}

			continue
		}
		// Seaborn (the base alias) resolves to seaborn_darkgrid.
		if n == Seaborn {
			if th.Name != string(SeabornDarkgrid) {
				t.Errorf("Resolve(Seaborn) Name = %q, want %q", th.Name, SeabornDarkgrid)
			}

			continue
		}

		if th.Name != string(n) {
			t.Errorf("Resolve(%q) Name = %q, want %q", n, th.Name, n)
		}
	}
}

// TestResolveEmpty checks that the empty name resolves to the default
// (dashboard) preset.
func TestResolveEmpty(t *testing.T) {
	t.Parallel()

	th, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve(\"\") error: %v", err)
	}

	if th.Name != string(Dashboard) {
		t.Errorf("Resolve(\"\") Name = %q, want %q", th.Name, Dashboard)
	}
}

// TestResolveUnknown checks that an unknown name returns an error
// rather than silently falling back.
func TestResolveUnknown(t *testing.T) {
	t.Parallel()

	if _, err := Resolve("not_a_real_theme"); err == nil {
		t.Error("Resolve(unknown) returned nil error")
	}
}

// TestPaletteCoverage asserts that themes derived from a matplotlib
// .mplstyle that sets axes.prop_cycle ship a non-empty Palette. Without
// this guard a missing palette silently degrades multi-series plots to
// the generic Tab10 cycle and breaks the visual identity of the preset.
func TestPaletteCoverage(t *testing.T) {
	t.Parallel()

	mustHavePalette := []Name{
		Ggplot, Classic, Grayscale, Bmh, Fivethirtyeight,
		DarkBackground, SolarizeLight2, TableauColorblind10,
		SeabornDarkgrid, SeabornWhitegrid, SeabornDark, SeabornWhite, SeabornTicks,
		SeabornDeep, SeabornMuted, SeabornBright, SeabornColorblind, SeabornPastel, SeabornDarkPalette,
		SeabornPaper, SeabornNotebook, SeabornTalk, SeabornPoster,
		PaulTol, Few, UCBerkeley,
	}
	for _, n := range mustHavePalette {
		th, err := Resolve(n)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", n, err)
		}

		if len(th.Palette) == 0 {
			t.Errorf("theme %q has empty Palette", n)
		}
	}
}

// TestHexHelper checks the hex parser the preset files rely on.
func TestHexHelper(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in      string
		r, g, b uint8
	}{
		{"E24A33", 0xE2, 0x4A, 0x33},
		{"#348ABD", 0x34, 0x8A, 0xBD},
		{"000000", 0, 0, 0},
		{"FFFFFF", 0xFF, 0xFF, 0xFF},
	}
	for _, c := range cases {
		got := hex(c.in)
		r, g, b, a := got.RGBA()
		// RGBA() returns 16-bit values; expand uint8 expectation.
		wantR := uint32(c.r)<<8 | uint32(c.r)
		wantG := uint32(c.g)<<8 | uint32(c.g)

		wantB := uint32(c.b)<<8 | uint32(c.b)
		if r != wantR || g != wantG || b != wantB || a != 0xFFFF {
			t.Errorf("hex(%q) = (%d,%d,%d,%d), want (%d,%d,%d,65535)", c.in, r, g, b, a, wantR, wantG, wantB)
		}
	}
}
