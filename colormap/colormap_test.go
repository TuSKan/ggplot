package colormap

import (
	"context"
	"math"
	"testing"

	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/gogpu/gg"
)

const epsilon = 1e-9

func approxEq(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func colorsEqual(a, b gg.RGBA) bool {
	return approxEq(a.R, b.R) && approxEq(a.G, b.G) && approxEq(a.B, b.B) && approxEq(a.A, b.A)
}

// --- Cmap ---

func TestLinearSegmented_AtBounds(t *testing.T) {
	c := Viridis
	low := c.At(0)
	high := c.At(1)
	if low.R == high.R && low.G == high.G && low.B == high.B {
		t.Fatalf("Viridis: At(0) and At(1) returned identical colors")
	}
	// Clamp behavior.
	if !colorsEqual(c.At(-0.5), c.At(0)) {
		t.Errorf("At(-0.5) should clamp to At(0)")
	}
	if !colorsEqual(c.At(1.5), c.At(1)) {
		t.Errorf("At(1.5) should clamp to At(1)")
	}
}

func TestLinearSegmented_NaN_DefaultBad(t *testing.T) {
	got := Viridis.At(math.NaN())
	if got.A != 0 {
		t.Errorf("default bad should be transparent; got A=%v", got.A)
	}
}

func TestCmap_Reversed(t *testing.T) {
	c := Plasma
	r := c.Reversed()
	if !colorsEqual(c.At(0), r.At(1)) {
		t.Errorf("Reversed: At(0) of base != At(1) of reversed")
	}
	if !colorsEqual(c.At(1), r.At(0)) {
		t.Errorf("Reversed: At(1) of base != At(0) of reversed")
	}
	if r.Name() != "plasma_r" {
		t.Errorf("Reversed name = %q, want plasma_r", r.Name())
	}
}

func TestCmap_Resampled(t *testing.T) {
	r := Viridis.Resampled(4)
	if r.N() != 4 {
		t.Errorf("Resampled N() = %d, want 4", r.N())
	}
	// All four sample points should yield distinct colors.
	seen := make(map[gg.RGBA]bool)
	for i := 0; i < 4; i++ {
		c := r.At(float64(i) / 3.0)
		if seen[c] {
			t.Errorf("Resampled produced duplicate color at i=%d", i)
		}
		seen[c] = true
	}
}

func TestCmap_WithExtremes(t *testing.T) {
	red := gg.RGBA{R: 1, G: 0, B: 0, A: 1}
	blue := gg.RGBA{R: 0, G: 0, B: 1, A: 1}
	gray := gg.RGBA{R: 0.5, G: 0.5, B: 0.5, A: 1}
	c := Viridis.WithExtremes(&red, &blue, &gray)

	if got := c.At(-0.1); !colorsEqual(got, red) {
		t.Errorf("Under: got %+v, want red", got)
	}
	if got := c.At(1.1); !colorsEqual(got, blue) {
		t.Errorf("Over: got %+v, want blue", got)
	}
	if got := c.At(math.NaN()); !colorsEqual(got, gray) {
		t.Errorf("Bad: got %+v, want gray", got)
	}
	// Original is untouched.
	if got := Viridis.At(-0.1); colorsEqual(got, red) {
		t.Errorf("WithExtremes mutated the base cmap")
	}
}

func TestListedCmap_Cycle(t *testing.T) {
	tab := Tab10.(*ListedCmap)
	c0 := tab.Color(0)
	c10 := tab.Color(10)
	if !colorsEqual(c0, c10) {
		t.Errorf("Tab10 should cycle: Color(0) != Color(10)")
	}
}

// --- Norm ---

func TestLinearNorm_RoundTrip(t *testing.T) {
	n := &LinearNorm{Vmin: 0, Vmax: 100}
	for _, v := range []float64{0, 25, 50, 75, 100} {
		t0 := n.Norm(v)
		v2 := n.Inverse(t0)
		if !approxEq(v, v2) {
			t.Errorf("Inverse(Norm(%v)) = %v, want %v", v, v2, v)
		}
	}
}

func TestLogNorm_PositiveOnly(t *testing.T) {
	n := &LogNorm{Vmin: 1, Vmax: 1000}
	if !math.IsNaN(n.Norm(0)) {
		t.Errorf("LogNorm.Norm(0) should be NaN")
	}
	if !math.IsNaN(n.Norm(-1)) {
		t.Errorf("LogNorm.Norm(-1) should be NaN")
	}
	if got := n.Norm(10); !approxEq(got, 1.0/3.0) {
		t.Errorf("LogNorm.Norm(10) = %v, want 1/3", got)
	}
}

func TestPowerNorm_Gamma(t *testing.T) {
	n := &PowerNorm{Gamma: 2, Vmin: 0, Vmax: 1}
	if got := n.Norm(0.5); !approxEq(got, 0.25) {
		t.Errorf("PowerNorm gamma=2: Norm(0.5) = %v, want 0.25", got)
	}
}

func TestTwoSlopeNorm_Asymmetry(t *testing.T) {
	n := &TwoSlopeNorm{Vcenter: 0, Vmin: -1, Vmax: 9}
	if got := n.Norm(0); !approxEq(got, 0.5) {
		t.Errorf("TwoSlopeNorm at center: %v, want 0.5", got)
	}
	if got := n.Norm(-1); !approxEq(got, 0) {
		t.Errorf("TwoSlopeNorm at vmin: %v, want 0", got)
	}
	if got := n.Norm(9); !approxEq(got, 1) {
		t.Errorf("TwoSlopeNorm at vmax: %v, want 1", got)
	}
}

func TestBoundaryNorm_Quantize(t *testing.T) {
	n := &BoundaryNorm{Boundaries: []float64{0, 1, 2, 3}, Ncolors: 3, Clip: true}
	if got := n.Norm(0.5); !approxEq(got, 0) {
		t.Errorf("BoundaryNorm bin 0: %v, want 0", got)
	}
	if got := n.Norm(2.5); !approxEq(got, 1) {
		t.Errorf("BoundaryNorm bin 2: %v, want 1", got)
	}
}

// --- Scale ---

func TestScale_Continuous_Train(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	col := eng.NewFloat64Column("z", []float64{0, 5, 10})
	s := NewContinuous(Viridis, nil)
	if err := s.Train(col); err != nil {
		t.Fatalf("Train: %v", err)
	}
	mn, mx := s.Norm().Bounds()
	if mn != 0 || mx != 10 {
		t.Errorf("Bounds = (%v, %v), want (0, 10)", mn, mx)
	}
	cLow := s.At(0.0)
	cHigh := s.At(10.0)
	if colorsEqual(cLow, cHigh) {
		t.Errorf("continuous scale should map 0 and 10 to different colors")
	}
}

func TestScale_Discrete_Train(t *testing.T) {
	eng := memory.NewEngine(context.Background())
	col := eng.NewStringColumn("g", []string{"a", "b", "a", "c", "b"})
	s := NewDiscrete(Tab10)
	if err := s.Train(col); err != nil {
		t.Fatalf("Train: %v", err)
	}
	cats := s.Categories()
	if len(cats) != 3 || cats[0] != "a" || cats[1] != "b" || cats[2] != "c" {
		t.Errorf("Categories = %v, want [a b c]", cats)
	}
	if !colorsEqual(s.At("a"), s.At("a")) {
		t.Errorf("same label should always map to same color")
	}
	if colorsEqual(s.At("a"), s.At("b")) {
		t.Errorf("different labels should map to different colors")
	}
}

func TestScale_Manual(t *testing.T) {
	red := gg.RGBA{R: 1, A: 1}
	s := NewManual(map[string]Color{"x": red})
	if got := s.At("x"); !colorsEqual(got, red) {
		t.Errorf("manual override not respected: got %+v, want red", got)
	}
}

// --- Registry ---

func TestRegistry_Resolve(t *testing.T) {
	c, err := Resolve("viridis")
	if err != nil {
		t.Fatal(err)
	}
	if c.Name() != "viridis" {
		t.Errorf("Resolve(viridis).Name() = %q", c.Name())
	}

	r, err := Resolve("viridis_r")
	if err != nil {
		t.Fatal(err)
	}
	if r.Name() != "viridis_r" {
		t.Errorf("Resolve(viridis_r).Name() = %q", r.Name())
	}
	if !colorsEqual(c.At(0), r.At(1)) {
		t.Errorf("_r suffix should reverse the gradient")
	}
}

func TestRegistry_UnknownName(t *testing.T) {
	if _, err := Resolve("does_not_exist"); err == nil {
		t.Errorf("Resolve should error on unknown name")
	}
}

func TestRegistry_NamesByCategory(t *testing.T) {
	pu := NamesByCategory(PerceptuallyUniform)
	want := []string{"cividis", "inferno", "magma", "plasma", "viridis"}
	if len(pu) != len(want) {
		t.Fatalf("PerceptuallyUniform names = %v, want %v", pu, want)
	}
	for i, n := range want {
		if pu[i] != n {
			t.Errorf("PerceptuallyUniform[%d] = %q, want %q", i, pu[i], n)
		}
	}
}

// --- Parse ---

func TestParse_Hex(t *testing.T) {
	cases := map[string]gg.RGBA{
		"#ff0000": {R: 1, G: 0, B: 0, A: 1},
		"00ff00":  {R: 0, G: 1, B: 0, A: 1},
		"#00f":    {R: 0, G: 0, B: 1, A: 1},
	}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q): %v", in, err)
			continue
		}
		if !approxEq(got.R, want.R) || !approxEq(got.G, want.G) || !approxEq(got.B, want.B) {
			t.Errorf("Parse(%q) = %+v, want %+v", in, got, want)
		}
	}
}

func TestParse_Named(t *testing.T) {
	cases := []string{"red", "Red", "REBECCAPURPLE", "coral", "transparent"}
	for _, in := range cases {
		if _, err := Parse(in); err != nil {
			t.Errorf("Parse(%q): %v", in, err)
		}
	}
}

func TestParse_TabAlias(t *testing.T) {
	c, err := Parse("tab:blue")
	if err != nil {
		t.Fatal(err)
	}
	want := Tab10.(*ListedCmap).Color(0)
	if !approxEq(c.R, want.R) || !approxEq(c.G, want.G) || !approxEq(c.B, want.B) {
		t.Errorf("tab:blue = %+v, want tab10[0] = %+v", c, want)
	}
}

func TestParse_RGBFunc(t *testing.T) {
	c, err := Parse("rgb(255, 0, 0)")
	if err != nil {
		t.Fatal(err)
	}
	if !approxEq(c.R, 1) || !approxEq(c.G, 0) || !approxEq(c.B, 0) {
		t.Errorf("rgb(255,0,0) = %+v", c)
	}

	c2, err := Parse("rgba(0, 128, 0, 0.5)")
	if err != nil {
		t.Fatal(err)
	}
	if !approxEq(c2.A, 0.5) {
		t.Errorf("rgba alpha = %v, want 0.5", c2.A)
	}
}

func TestParse_HSLFunc(t *testing.T) {
	// hsl(120, 100%, 50%) is pure green.
	c, err := Parse("hsl(120, 100%, 50%)")
	if err != nil {
		t.Fatal(err)
	}
	if c.G < 0.9 || c.R > 0.1 || c.B > 0.1 {
		t.Errorf("hsl(120,100%%,50%%) should be ~green, got %+v", c)
	}
}

func TestParse_Invalid(t *testing.T) {
	for _, in := range []string{"", "not a color", "rgb(", "tab:nope"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) should error", in)
		}
	}
}

// --- LUT correctness (smoke check the perceptual stops are present) ---

func TestPerceptualLUT_StopAlignment(t *testing.T) {
	// At t=0, viridis should be the dark purple stop ≈ (68,1,84).
	c := Viridis.At(0)
	r := uint8(c.R*255 + 0.5)
	g := uint8(c.G*255 + 0.5)
	b := uint8(c.B*255 + 0.5)
	if r != 68 || g != 1 || b != 84 {
		t.Errorf("Viridis.At(0) = (%d,%d,%d), want (68,1,84)", r, g, b)
	}
	// At t=1, viridis should be the bright yellow stop ≈ (253,231,37).
	c = Viridis.At(1)
	r = uint8(c.R*255 + 0.5)
	g = uint8(c.G*255 + 0.5)
	b = uint8(c.B*255 + 0.5)
	if r != 253 || g != 231 || b != 37 {
		t.Errorf("Viridis.At(1) = (%d,%d,%d), want (253,231,37)", r, g, b)
	}
}
