package scale

import (
	"math"
)

// Extended implements the Talbot-Lin-Hanrahan (2010) "An Extension of
// Wilkinson's Algorithm for Positioning Tick Labels on Axes" for optimal
// tick label placement.
//
// Reference: Talbot, J., Lin, S., Hanrahan, P. (2010)
// "An Extension of Wilkinson's Algorithm for Positioning Tick Labels on Axes",
// IEEE Transactions on Visualization and Computer Graphics, 16(6), 1036–1043.
// http://vis.stanford.edu/files/2010-TickLabels-InfoVis.pdf
//
// The algorithm optimizes four criteria simultaneously:
//   - Simplicity: preference for "nice" step sizes (1, 5, 2, 2.5, 4, 3)
//   - Coverage: how well the labels span the data range
//   - Density: how close the number of ticks is to the target
//   - Legibility: readability of the label format (simplified to 1.0 here)

// Q is the preference-ordered set of "nice" step multipliers.
// Order encodes simplicity: earlier values are preferred.
var Q = []float64{1, 5, 2, 2.5, 4, 3}

// weights for [simplicity, coverage, density, legibility].
var defaultWeights = [4]float64{0.2, 0.25, 0.5, 0.05}

// extendedWilkinson computes optimal tick positions for a data range
// [dmin, dmax] targeting approximately targetDensity ticks.
func extendedWilkinson(dmin, dmax float64, targetDensity int) []float64 {
	if targetDensity <= 0 {
		targetDensity = 5
	}

	if dmin == dmax {
		return []float64{dmin}
	}

	if dmin > dmax {
		dmin, dmax = dmax, dmin
	}

	w := defaultWeights
	bestScore := -2.0

	var bestLmin, bestLmax, bestLstep float64

	bestK := 0

	// j iterates "skip" values (1 = every Q, 2 = every 2nd, etc.)
	for j := 1; j < 20; j++ {
		for qi, q := range Q {
			sm := simplicityMax(qi, j)
			if dot4(sm, 1, 1, 1, w) < bestScore {
				goto nextJ
			}

			for k := 2; k < 40; k++ {
				dm := densityMax(k, targetDensity)
				if dot4(sm, 1, dm, 1, w) < bestScore {
					break
				}

				delta := (dmax - dmin) / float64(k+1) / float64(j) / q
				z := int(math.Ceil(math.Log10(delta)))

				for ; z < 20; z++ {
					lstep := q * float64(j) * math.Pow(10, float64(z))
					cm := coverageMax(dmin, dmax, lstep*(float64(k)-1))

					if dot4(sm, cm, dm, 1, w) < bestScore {
						break
					}

					minStart := int(math.Floor(dmax/lstep)) - (k - 1)
					maxStart := int(math.Ceil(dmin / lstep))

					for start := minStart; start <= maxStart; start++ {
						lmin := float64(start) * lstep
						lmax := lmin + float64(k-1)*lstep

						s := simplicity(qi, j, lmin, lmax, lstep)
						c := coverage(dmin, dmax, lmin, lmax)
						d := density(k, targetDensity, dmin, dmax, lmin, lmax)
						l := 1.0 // simplified legibility

						score := dot4(s, c, d, l, w)

						if score > bestScore {
							bestScore = score
							bestLmin = lmin
							bestLmax = lmax
							bestLstep = lstep
							bestK = k
						}
					}
				}
			}
		}

	nextJ:
	}

	// If we found no solution, fall back to simple ticks.
	if bestK == 0 || bestLstep <= 0 {
		return fallbackTicks(dmin, dmax, targetDensity)
	}

	// Generate the result.
	ticks := make([]float64, 0, bestK)
	for v := bestLmin; v <= bestLmax+bestLstep*0.5; v += bestLstep {
		ticks = append(ticks, roundSig(v, 12))
	}

	return ticks
}

// --- Scoring functions ---

// simplicity measures how "nice" the chosen step is.
// q index i in Q (0-based), skip j, whether the label range includes zero.
func simplicity(qi, j int, lmin, lmax, lstep float64) float64 {
	v := 0.0
	if containsZero(lmin, lmax, lstep) {
		v = 1.0
	}

	return 1.0 - float64(qi)/(float64(len(Q))-1.0) - float64(j) + v
}

// simplicityMax is the upper bound on simplicity for Q[qi] with skip j.
func simplicityMax(qi, j int) float64 {
	return 1.0 - float64(qi)/(float64(len(Q))-1.0) - float64(j) + 1.0
}

// coverage measures how well the labels span the data range.
func coverage(dmin, dmax, lmin, lmax float64) float64 {
	dataRange := dmax - dmin
	if dataRange <= 0 {
		return 1.0
	}

	halfCover := 0.5 * (dataRange - (lmax - lmin))

	return 1.0 - 0.5*(halfCover*halfCover)/(0.1*dataRange)/(0.1*dataRange)
}

// coverageMax is the upper bound on coverage.
func coverageMax(dmin, dmax, span float64) float64 {
	dataRange := dmax - dmin
	if span >= dataRange {
		return 1.0
	}

	halfCover := 0.5 * (dataRange - span)

	return 1.0 - 0.5*(halfCover*halfCover)/(0.1*dataRange)/(0.1*dataRange)
}

// density measures how close the tick density is to the target.
func density(k, m int, dmin, dmax, lmin, lmax float64) float64 {
	// r = actual density, rt = target density
	r := float64(k-1) / (lmax - lmin)

	rt := float64(m-1) / (dmax - dmin)
	if rt == 0 {
		return 1.0
	}

	return 2.0 - math.Max(r/rt, rt/r)
}

// densityMax is the upper bound on density for k ticks.
func densityMax(k, m int) float64 {
	if k >= m {
		return 2.0 - float64(k-1)/float64(m-1)
	}

	return 1.0
}

// --- Helpers ---

// containsZero returns true if zero is a tick mark in the sequence.
func containsZero(lmin, lmax, lstep float64) bool {
	if lstep <= 0 {
		return false
	}

	if lmin > 0 || lmax < 0 {
		return false
	}
	// Check if 0 is exactly on a step boundary.
	steps := -lmin / lstep

	return math.Abs(steps-math.Round(steps)) < 1e-10
}

// dot4 computes the weighted dot product of 4 components.
func dot4(a, b, c, d float64, w [4]float64) float64 { //nolint:unparam // d is the legibility score; always 1.0 in current Wilkinson simplification.
	return a*w[0] + b*w[1] + c*w[2] + d*w[3]
}

// roundSig rounds v to n significant digits.
func roundSig(v float64, n int) float64 {
	if v == 0 {
		return 0
	}

	d := math.Ceil(math.Log10(math.Abs(v)))
	power := float64(n) - d
	mag := math.Pow(10, power)

	return math.Round(v*mag) / mag
}

// fallbackTicks is the simple Heckbert nice-number algorithm used when
// the extended algorithm finds no solution.
func fallbackTicks(lo, hi float64, n int) []float64 {
	step := niceNum((hi-lo)/float64(n-1), true)
	lo = math.Floor(lo/step) * step
	hi = math.Ceil(hi/step) * step

	var ticks []float64
	for v := lo; v <= hi+step*0.5; v += step {
		ticks = append(ticks, roundTo(v, 10))
	}

	return ticks
}
