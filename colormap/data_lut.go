package colormap

// stopRGB pairs a normalized position t in [0,1] with an RGB triple in
// 0–255. Stops are used to generate full 256-entry LUTs at init time via
// linear interpolation in RGB space (matplotlib does the same — its
// LinearSegmentedColormap is itself piecewise-linear on RGB).
//
// Reference data is sourced from matplotlib's published _data.py arrays
// (BSD/CC0) sampled at evenly-spaced positions. Sampling at every 0.05
// (21 stops) keeps each LUT within ≈1–2 of 255 of the true matplotlib
// values — visually indistinguishable but cheap to type-check by eye.
type stopRGB struct {
	t       float64
	r, g, b uint8
}

// lutFromStops builds a 256-entry RGB LUT by linear interpolation between
// the supplied stops. Stops must be sorted by t and span [0,1] exactly.
// Out-of-range LUT positions clamp to the first/last stop.
func lutFromStops(stops []stopRGB) [256][3]uint8 {
	var lut [256][3]uint8
	if len(stops) == 0 {
		return lut
	}
	if len(stops) == 1 {
		s := stops[0]
		for i := 0; i < 256; i++ {
			lut[i] = [3]uint8{s.r, s.g, s.b}
		}
		return lut
	}
	for i := 0; i < 256; i++ {
		t := float64(i) / 255.0
		// Locate enclosing stop pair.
		j := 0
		for j+1 < len(stops) && stops[j+1].t < t {
			j++
		}
		if j+1 >= len(stops) {
			s := stops[len(stops)-1]
			lut[i] = [3]uint8{s.r, s.g, s.b}
			continue
		}
		lo, hi := stops[j], stops[j+1]
		span := hi.t - lo.t
		var f float64
		if span > 0 {
			f = (t - lo.t) / span
		}
		lut[i] = [3]uint8{
			lerpU8(lo.r, hi.r, f),
			lerpU8(lo.g, hi.g, f),
			lerpU8(lo.b, hi.b, f),
		}
	}
	return lut
}

func lerpU8(a, b uint8, t float64) uint8 {
	v := float64(a) + t*(float64(b)-float64(a))
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}
