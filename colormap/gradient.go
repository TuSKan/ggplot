package colormap

import (
	"fmt"
	"math"

	"github.com/gogpu/gg"
)

// Gradient returns a 2-stop continuous colormap that interpolates between
// low and high in CIELAB color space. CIELAB interpolation is perceptually
// uniform — equal steps in t produce equal perceived color differences.
func Gradient(low, high Color) Cmap {
	return &gradientCmap{
		name:   "gradient",
		cat:    Sequential,
		stops:  []Color{low, high},
		breaks: []float64{0, 1},
	}
}

// Gradient2 returns a 3-stop diverging colormap that interpolates through
// low → mid → high in CIELAB color space. The midpoint is at t=0.5.
func Gradient2(low, mid, high Color) Cmap {
	return &gradientCmap{
		name:   "gradient2",
		cat:    Diverging,
		stops:  []Color{low, mid, high},
		breaks: []float64{0, 0.5, 1},
	}
}

// GradientN returns an N-stop continuous colormap from evenly-spaced color
// stops, interpolated in CIELAB color space. Panics if len(colors) < 2.
func GradientN(colors []Color) Cmap {
	if len(colors) < 2 {
		panic("colormap.GradientN: need at least 2 colors")
	}

	stops := make([]Color, len(colors))
	copy(stops, colors)

	breaks := make([]float64, len(colors))
	n := float64(len(colors) - 1)

	for i := range colors {
		breaks[i] = float64(i) / n
	}

	return &gradientCmap{
		name:   fmt.Sprintf("gradientN_%d", len(colors)),
		cat:    Sequential,
		stops:  stops,
		breaks: breaks,
	}
}

// gradientCmap interpolates between color stops in CIELAB color space.
type gradientCmap struct {
	name   string
	cat    Category
	stops  []Color
	breaks []float64
	ext    extremes
}

func (c *gradientCmap) At(t float64) gg.RGBA {
	if r, ok := c.ext.resolve(t); ok {
		return r
	}

	if t <= 0 {
		return c.stops[0]
	}

	if t >= 1 {
		return c.stops[len(c.stops)-1]
	}

	// Find the segment [i, i+1] that contains t.
	for i := range len(c.breaks) - 1 {
		if t <= c.breaks[i+1] {
			// Normalize t within this segment.
			segLen := c.breaks[i+1] - c.breaks[i]
			frac := (t - c.breaks[i]) / segLen

			return lerpLab(c.stops[i], c.stops[i+1], frac)
		}
	}

	return c.stops[len(c.stops)-1]
}

func (c *gradientCmap) Name() string       { return c.name }
func (c *gradientCmap) N() int             { return 256 }
func (c *gradientCmap) Category() Category { return c.cat }

func (c *gradientCmap) Reversed() Cmap {
	rev := make([]Color, len(c.stops))
	for i, s := range c.stops {
		rev[len(c.stops)-1-i] = s
	}

	return &gradientCmap{
		name:   c.name + "_r",
		cat:    c.cat,
		stops:  rev,
		breaks: c.breaks,
		ext:    c.ext,
	}
}

func (c *gradientCmap) Resampled(n int) Cmap {
	if n < 1 {
		n = 1
	}

	colors := make([]gg.RGBA, n)
	if n == 1 {
		colors[0] = c.At(0.5)
	} else {
		for i := range n {
			colors[i] = c.At(float64(i) / float64(n-1))
		}
	}

	return &ListedCmap{
		name:   fmt.Sprintf("%s_n%d", c.name, n),
		cat:    c.cat,
		colors: colors,
		ext:    c.ext,
	}
}

func (c *gradientCmap) WithExtremes(under, over, bad *gg.RGBA) Cmap {
	clone := *c
	clone.stops = append([]Color(nil), c.stops...)
	clone.breaks = append([]float64(nil), c.breaks...)
	clone.ext = mergeExtremes(c.ext, under, over, bad)

	return &clone
}

// ---------------------------------------------------------------------------
// CIELAB color space conversion and interpolation
// ---------------------------------------------------------------------------

// lerpLab interpolates two sRGB colors in CIELAB space. This produces
// perceptually uniform gradients — equal steps in t appear equally different
// to the human eye.
func lerpLab(a, b Color, t float64) Color {
	aL, aA, aB := rgbToLab(a.R, a.G, a.B)
	bL, bA, bB := rgbToLab(b.R, b.G, b.B)

	L := aL + t*(bL-aL)
	A := aA + t*(bA-aA)
	B := aB + t*(bB-aB)
	alpha := a.A + t*(b.A-a.A)

	r, g, bl := labToRGB(L, A, B)

	return Color{R: r, G: g, B: bl, A: alpha}
}

// rgbToLab converts sRGB [0,1] to CIELAB via D65 XYZ.
func rgbToLab(r, g, b float64) (l, a, bOut float64) {
	// sRGB → linear RGB (inverse gamma)
	r = srgbToLinear(r)
	g = srgbToLinear(g)
	b = srgbToLinear(b)

	// Linear RGB → XYZ (D65 observer)
	x := 0.4124564*r + 0.3575761*g + 0.1804375*b
	y := 0.2126729*r + 0.7151522*g + 0.0721750*b
	z := 0.0193339*r + 0.1191920*g + 0.9503041*b

	// D65 reference white
	const xn, yn, zn = 0.95047, 1.0, 1.08883

	fx := labF(x / xn)
	fy := labF(y / yn)
	fz := labF(z / zn)

	l = 116*fy - 16
	a = 500 * (fx - fy)
	bOut = 200 * (fy - fz)

	return l, a, bOut
}

// labToRGB converts CIELAB to sRGB [0,1] via D65 XYZ.
func labToRGB(l, a, b float64) (r, g, bOut float64) {
	// CIELAB → XYZ
	const xn, yn, zn = 0.95047, 1.0, 1.08883

	fy := (l + 16) / 116
	fx := a/500 + fy
	fz := fy - b/200

	x := xn * labFInv(fx)
	y := yn * labFInv(fy)
	z := zn * labFInv(fz)

	// XYZ → linear RGB
	rl := 3.2404542*x - 1.5371385*y - 0.4985314*z
	gl := -0.9692660*x + 1.8760108*y + 0.0415560*z
	bl := 0.0556434*x - 0.2040259*y + 1.0572252*z

	// Linear RGB → sRGB (gamma)
	r = linearToSRGB(rl)
	g = linearToSRGB(gl)
	bOut = linearToSRGB(bl)

	// Clamp to [0,1] since out-of-gamut colors can escape.
	r = math.Max(0, math.Min(1, r))
	g = math.Max(0, math.Min(1, g))
	bOut = math.Max(0, math.Min(1, bOut))

	return r, g, bOut
}

// labF is the CIE nonlinear transformation f(t) used in XYZ → Lab.
func labF(t float64) float64 {
	const delta = 6.0 / 29.0 // ≈ 0.206897

	if t > delta*delta*delta {
		return math.Cbrt(t)
	}

	return t/(3*delta*delta) + 4.0/29.0
}

// labFInv is the inverse of labF, used in Lab → XYZ.
func labFInv(t float64) float64 {
	const delta = 6.0 / 29.0

	if t > delta {
		return t * t * t
	}

	return 3 * delta * delta * (t - 4.0/29.0)
}

// srgbToLinear converts a single sRGB channel to linear RGB.
func srgbToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}

	return math.Pow((c+0.055)/1.055, 2.4)
}

// linearToSRGB converts a single linear RGB channel to sRGB.
func linearToSRGB(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}

	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
}
