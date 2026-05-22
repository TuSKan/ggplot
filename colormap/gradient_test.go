package colormap

import (
	"math"
	"testing"

	"github.com/gogpu/gg"
)

// nearEq is a tolerant approx check for CIELAB round-trip / gradient tests.
func nearEq(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

func TestGradient(t *testing.T) {
	t.Parallel()

	white := Color{R: 1, G: 1, B: 1, A: 1}
	blue := Color{R: 0, G: 0, B: 1, A: 1}

	cm := Gradient(white, blue)

	// t=0 should be white.
	c0 := cm.At(0)
	if !nearEq(c0.R, 1, 0.01) || !nearEq(c0.G, 1, 0.01) || !nearEq(c0.B, 1, 0.01) {
		t.Errorf("At(0): got %v, want ~white", c0)
	}

	// t=1 should be blue.
	c1 := cm.At(1)
	if !nearEq(c1.R, 0, 0.01) || !nearEq(c1.G, 0, 0.01) || !nearEq(c1.B, 1, 0.01) {
		t.Errorf("At(1): got %v, want ~blue", c1)
	}

	// t=0.5 should be a mid-color (not exactly RGB midpoint due to CIELAB).
	c5 := cm.At(0.5)
	if c5.R < 0.1 || c5.R > 0.95 {
		t.Errorf("At(0.5): R=%.3f out of expected CIELAB midpoint range", c5.R)
	}

	// Alpha should interpolate linearly.
	if !nearEq(c5.A, 1, 0.01) {
		t.Errorf("At(0.5): alpha=%.3f, want 1.0", c5.A)
	}
}

func TestGradient2(t *testing.T) {
	t.Parallel()

	red := Color{R: 1, G: 0, B: 0, A: 1}
	white := Color{R: 1, G: 1, B: 1, A: 1}
	blue := Color{R: 0, G: 0, B: 1, A: 1}

	cm := Gradient2(red, white, blue)

	// t=0 should be red.
	c0 := cm.At(0)
	if !nearEq(c0.R, 1, 0.02) || !nearEq(c0.G, 0, 0.02) {
		t.Errorf("At(0): got %v, want ~red", c0)
	}

	// t=0.5 should be white (midpoint).
	c5 := cm.At(0.5)
	if !nearEq(c5.R, 1, 0.02) || !nearEq(c5.G, 1, 0.02) || !nearEq(c5.B, 1, 0.02) {
		t.Errorf("At(0.5): got %v, want ~white", c5)
	}

	// t=1 should be blue.
	c1 := cm.At(1)
	if !nearEq(c1.R, 0, 0.02) || !nearEq(c1.B, 1, 0.02) {
		t.Errorf("At(1): got %v, want ~blue", c1)
	}
}

func TestGradientN(t *testing.T) {
	t.Parallel()

	red := Color{R: 1, G: 0, B: 0, A: 1}
	green := Color{R: 0, G: 0.8, B: 0, A: 1}
	blue := Color{R: 0, G: 0, B: 1, A: 1}

	cm := GradientN([]Color{red, green, blue})

	// t=0 → red, t=0.5 → green, t=1 → blue.
	c0 := cm.At(0)
	if !nearEq(c0.R, 1, 0.02) {
		t.Errorf("At(0): got %v, want ~red", c0)
	}

	c5 := cm.At(0.5)
	if !nearEq(c5.G, 0.8, 0.05) {
		t.Errorf("At(0.5): got G=%.3f, want ~0.8", c5.G)
	}

	c1 := cm.At(1)
	if !nearEq(c1.B, 1, 0.02) {
		t.Errorf("At(1): got %v, want ~blue", c1)
	}
}

func TestGradientNPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("GradientN(1 color) should panic")
		}
	}()

	GradientN([]Color{{R: 1, G: 0, B: 0, A: 1}})
}

func TestGradientReversed(t *testing.T) {
	t.Parallel()

	white := Color{R: 1, G: 1, B: 1, A: 1}
	black := Color{R: 0, G: 0, B: 0, A: 1}

	cm := Gradient(black, white)
	rev := cm.Reversed()

	// Original: At(0) = black, At(1) = white.
	// Reversed: At(0) = white, At(1) = black.
	c0 := rev.At(0)
	if !nearEq(c0.R, 1, 0.02) {
		t.Errorf("Reversed At(0): got R=%.3f, want ~1.0", c0.R)
	}

	c1 := rev.At(1)
	if !nearEq(c1.R, 0, 0.02) {
		t.Errorf("Reversed At(1): got R=%.3f, want ~0.0", c1.R)
	}
}

func TestGradientResampled(t *testing.T) {
	t.Parallel()

	white := Color{R: 1, G: 1, B: 1, A: 1}
	black := Color{R: 0, G: 0, B: 0, A: 1}

	cm := Gradient(black, white).Resampled(5)
	if cm.N() != 5 {
		t.Errorf("Resampled(5).N() = %d, want 5", cm.N())
	}
}

func TestGradientClamping(t *testing.T) {
	t.Parallel()

	white := Color{R: 1, G: 1, B: 1, A: 1}
	black := Color{R: 0, G: 0, B: 0, A: 1}

	cm := Gradient(black, white)

	// t < 0 → clamp to first stop.
	cNeg := cm.At(-1)
	if !nearEq(cNeg.R, 0, 0.02) {
		t.Errorf("At(-1): got R=%.3f, want ~0.0", cNeg.R)
	}

	// t > 1 → clamp to last stop.
	cOver := cm.At(2)
	if !nearEq(cOver.R, 1, 0.02) {
		t.Errorf("At(2): got R=%.3f, want ~1.0", cOver.R)
	}
}

func TestGradientWithExtremes(t *testing.T) {
	t.Parallel()

	white := Color{R: 1, G: 1, B: 1, A: 1}
	black := Color{R: 0, G: 0, B: 0, A: 1}
	red := Color{R: 1, G: 0, B: 0, A: 1}
	blue := Color{R: 0, G: 0, B: 1, A: 1}
	grey := Color{R: 0.5, G: 0.5, B: 0.5, A: 1}

	cm := Gradient(black, white).WithExtremes(&red, &blue, &grey)

	// Under → red.
	cUnder := cm.At(-1)
	if !nearEq(cUnder.R, 1, 0.01) || !nearEq(cUnder.G, 0, 0.01) {
		t.Errorf("At(-1) with under=red: got %v", cUnder)
	}

	// Over → blue.
	cOver := cm.At(2)
	if !nearEq(cOver.B, 1, 0.01) || !nearEq(cOver.R, 0, 0.01) {
		t.Errorf("At(2) with over=blue: got %v", cOver)
	}

	// NaN → grey.
	cBad := cm.At(math.NaN())
	if !nearEq(cBad.R, 0.5, 0.01) {
		t.Errorf("At(NaN) with bad=grey: got %v", cBad)
	}
}

func TestLabRoundtrip(t *testing.T) {
	t.Parallel()

	// Test that RGB → Lab → RGB is a round-trip for in-gamut colors.
	colors := []Color{
		{R: 1, G: 0, B: 0, A: 1},       // red
		{R: 0, G: 1, B: 0, A: 1},       // green
		{R: 0, G: 0, B: 1, A: 1},       // blue
		{R: 1, G: 1, B: 1, A: 1},       // white
		{R: 0, G: 0, B: 0, A: 1},       // black
		{R: 0.5, G: 0.3, B: 0.7, A: 1}, // mid
	}

	for _, c := range colors {
		l, a, b := rgbToLab(c.R, c.G, c.B)
		r, g, bl := labToRGB(l, a, b)

		if !nearEq(r, c.R, 0.005) || !nearEq(g, c.G, 0.005) || !nearEq(bl, c.B, 0.005) {
			t.Errorf("roundtrip (%.2f,%.2f,%.2f): got (%.4f,%.4f,%.4f)",
				c.R, c.G, c.B, r, g, bl)
		}
	}
}

func TestScaleNAColor(t *testing.T) {
	t.Parallel()

	s := NewContinuous(Viridis, nil)

	// Default: NaN returns cmap's bad color (transparent black).
	c := s.At(math.NaN())
	if c.A != 0 {
		t.Errorf("default NA: got alpha=%.2f, want 0", c.A)
	}

	// Set NA color to red.
	red := gg.RGBA{R: 1, G: 0, B: 0, A: 1}
	s.SetNAColor(&red)

	c = s.At(math.NaN())
	if !nearEq(c.R, 1, 0.01) || !nearEq(c.G, 0, 0.01) {
		t.Errorf("SetNAColor(red): At(NaN) = %v, want red", c)
	}

	// Nil reverts to default.
	s.SetNAColor(nil)

	c = s.At(math.NaN())
	if c.A != 0 {
		t.Errorf("reverted NA: got alpha=%.2f, want 0", c.A)
	}
}
