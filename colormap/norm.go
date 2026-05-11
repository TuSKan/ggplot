package colormap

import (
	"fmt"
	"math"
	"sort"

	"github.com/TuSKan/ggplot/dataset"
)

// Norm transforms an arbitrary scalar v into the unit interval [0,1] for
// input to a [Cmap]. Out-of-range values may be returned outside [0,1] —
// the consuming Cmap will clamp them or substitute Under/Over via
// [Cmap.WithExtremes].
type Norm interface {
	// Norm transforms v -> [0,1]. NaN is returned as NaN.
	Norm(v float64) float64

	// Inverse maps t in [0,1] back to data space.
	Inverse(t float64) float64

	// Bounds returns the current trained data range.
	Bounds() (vmin, vmax float64)

	// Train expands the bounds to cover values in col. Successive Train
	// calls accumulate (data range only grows).
	Train(col dataset.AnyColumn) error
}

// LinearNorm scales v linearly between Vmin and Vmax. Zero value is valid:
// it auto-trains on the first Train() call.
type LinearNorm struct {
	Vmin, Vmax float64
	trained    bool
}

// Norm maps v linearly to [0, 1] between Vmin and Vmax.
func (n *LinearNorm) Norm(v float64) float64 {
	if math.IsNaN(v) {
		return math.NaN()
	}

	if n.Vmax == n.Vmin {
		return 0.5
	}

	return (v - n.Vmin) / (n.Vmax - n.Vmin)
}

// Inverse maps t in [0, 1] back to the linear data range.
func (n *LinearNorm) Inverse(t float64) float64 {
	return n.Vmin + t*(n.Vmax-n.Vmin)
}

// Bounds returns the current trained data range.
func (n *LinearNorm) Bounds() (float64, float64) { return n.Vmin, n.Vmax }

// Train expands the range to cover values in col.
func (n *LinearNorm) Train(col dataset.AnyColumn) error {
	mn, mx, ok, err := minMaxColumn(col)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	n.expand(mn, mx)

	return nil
}

func (n *LinearNorm) expand(mn, mx float64) {
	if !n.trained {
		n.Vmin, n.Vmax = mn, mx
		n.trained = true

		return
	}

	if mn < n.Vmin {
		n.Vmin = mn
	}

	if mx > n.Vmax {
		n.Vmax = mx
	}
}

// LogNorm performs base-10 log scaling between Vmin and Vmax. Vmin must be
// strictly positive after training; values v <= 0 yield NaN.
type LogNorm struct {
	Vmin, Vmax float64
	trained    bool
}

// Norm maps v through log10 scaling to [0, 1].
func (n *LogNorm) Norm(v float64) float64 {
	if math.IsNaN(v) || v <= 0 || n.Vmin <= 0 || n.Vmax <= 0 {
		return math.NaN()
	}

	if n.Vmax == n.Vmin {
		return 0.5
	}

	return (math.Log10(v) - math.Log10(n.Vmin)) / (math.Log10(n.Vmax) - math.Log10(n.Vmin))
}

// Inverse maps t in [0, 1] back to the log-scaled data range.
func (n *LogNorm) Inverse(t float64) float64 {
	if n.Vmin <= 0 || n.Vmax <= 0 {
		return math.NaN()
	}

	logMin := math.Log10(n.Vmin)

	return math.Pow(10, logMin+t*(math.Log10(n.Vmax)-logMin))
}

// Bounds returns the current trained data range.
func (n *LogNorm) Bounds() (float64, float64) { return n.Vmin, n.Vmax }

// Train expands the range to cover positive values in col.
func (n *LogNorm) Train(col dataset.AnyColumn) error {
	mn, mx, ok, err := minMaxPositive(col)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("colormap: LogNorm requires strictly positive values in %q: %w", col.Name(), ErrParseColor)
	}

	if !n.trained {
		n.Vmin, n.Vmax = mn, mx
		n.trained = true

		return nil
	}

	if mn < n.Vmin {
		n.Vmin = mn
	}

	if mx > n.Vmax {
		n.Vmax = mx
	}

	return nil
}

// PowerNorm scales v with a power-law gamma between Vmin and Vmax. Equivalent
// to matplotlib's PowerNorm. Useful for emphasizing small or large values.
type PowerNorm struct {
	Gamma      float64
	Vmin, Vmax float64
	trained    bool
}

// Norm maps v through a power-law transform to [0, 1].
func (n *PowerNorm) Norm(v float64) float64 {
	if math.IsNaN(v) {
		return math.NaN()
	}

	if n.Vmax == n.Vmin {
		return 0.5
	}

	t := (v - n.Vmin) / (n.Vmax - n.Vmin)
	if t < 0 {
		return -math.Pow(-t, n.gamma())
	}

	return math.Pow(t, n.gamma())
}

// Inverse maps t in [0, 1] back to the power-scaled data range.
func (n *PowerNorm) Inverse(t float64) float64 {
	g := n.gamma()
	if g == 0 {
		return n.Vmin
	}

	var raw float64
	if t < 0 {
		raw = -math.Pow(-t, 1/g)
	} else {
		raw = math.Pow(t, 1/g)
	}

	return n.Vmin + raw*(n.Vmax-n.Vmin)
}

// Bounds returns the current trained data range.
func (n *PowerNorm) Bounds() (float64, float64) { return n.Vmin, n.Vmax }

// Train expands the range to cover values in col.
func (n *PowerNorm) Train(col dataset.AnyColumn) error {
	mn, mx, ok, err := minMaxColumn(col)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	if !n.trained {
		n.Vmin, n.Vmax = mn, mx
		n.trained = true

		return nil
	}

	if mn < n.Vmin {
		n.Vmin = mn
	}

	if mx > n.Vmax {
		n.Vmax = mx
	}

	return nil
}

func (n *PowerNorm) gamma() float64 {
	if n.Gamma <= 0 {
		return 1
	}

	return n.Gamma
}

// TwoSlopeNorm normalises v asymmetrically around Vcenter — values below the
// center fill [0, 0.5] and values above fill [0.5, 1]. Standard choice for
// diverging colormaps where zero (or another anchor) should sit on the
// neutral midpoint regardless of how lopsided the data is.
type TwoSlopeNorm struct {
	Vcenter    float64
	Vmin, Vmax float64
	trained    bool
}

// Norm maps v to [0, 1] with an asymmetric split at Vcenter.
func (n *TwoSlopeNorm) Norm(v float64) float64 {
	if math.IsNaN(v) {
		return math.NaN()
	}

	if v <= n.Vcenter {
		if n.Vcenter == n.Vmin {
			return 0
		}

		return 0.5 * (v - n.Vmin) / (n.Vcenter - n.Vmin)
	}

	if n.Vmax == n.Vcenter {
		return 1
	}

	return 0.5 + 0.5*(v-n.Vcenter)/(n.Vmax-n.Vcenter)
}

// Inverse maps t in [0, 1] back to the two-slope data range.
func (n *TwoSlopeNorm) Inverse(t float64) float64 {
	if t <= 0.5 {
		return n.Vmin + 2*t*(n.Vcenter-n.Vmin)
	}

	return n.Vcenter + 2*(t-0.5)*(n.Vmax-n.Vcenter)
}

// Bounds returns the current trained data range.
func (n *TwoSlopeNorm) Bounds() (float64, float64) { return n.Vmin, n.Vmax }

// Train expands the range to cover values in col.
func (n *TwoSlopeNorm) Train(col dataset.AnyColumn) error {
	mn, mx, ok, err := minMaxColumn(col)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	if !n.trained {
		n.Vmin, n.Vmax = mn, mx
		n.trained = true
	} else {
		if mn < n.Vmin {
			n.Vmin = mn
		}

		if mx > n.Vmax {
			n.Vmax = mx
		}
	}

	if n.Vcenter <= n.Vmin || n.Vcenter >= n.Vmax {
		return fmt.Errorf("colormap: TwoSlopeNorm Vcenter=%g must lie strictly within [Vmin=%g, Vmax=%g]", //nolint:err113 // error contains dynamic context values that vary per call site.
			n.Vcenter, n.Vmin, n.Vmax)
	}

	return nil
}

// BoundaryNorm bins v into one of len(Boundaries)-1 cells. The result is
// rounded to one of Ncolors discrete fractions so a Cmap.Resampled(Ncolors)
// produces a stepped colorbar with breakpoints at Boundaries.
type BoundaryNorm struct {
	Boundaries []float64 // ascending; len >= 2
	Ncolors    int       // typically len(Boundaries)-1
	Clip       bool      // clamp out-of-range to nearest boundary
}

// Norm maps v into discrete boundary cells as a fraction of [0, 1].
func (n *BoundaryNorm) Norm(v float64) float64 {
	if math.IsNaN(v) {
		return math.NaN()
	}

	b := n.Boundaries
	if len(b) < 2 {
		return math.NaN()
	}

	if v < b[0] {
		if n.Clip {
			v = b[0]
		} else {
			return -1
		}
	}

	if v > b[len(b)-1] {
		if n.Clip {
			v = b[len(b)-1]
		} else {
			return 2
		}
	}

	idx := max(sort.SearchFloat64s(b, v)-1, 0)

	if idx >= len(b)-1 {
		idx = len(b) - 2
	}

	nc := n.Ncolors
	if nc <= 0 {
		nc = len(b) - 1
	}
	// Map cell index → discrete t in [0,1].
	cells := len(b) - 1
	if cells == 1 {
		return 0.5
	}

	bin := int(float64(idx) * float64(nc) / float64(cells))
	if bin >= nc {
		bin = nc - 1
	}

	return float64(bin) / float64(nc-1)
}

// Inverse maps t in [0, 1] back to the midpoint of the matching boundary cell.
func (n *BoundaryNorm) Inverse(t float64) float64 {
	b := n.Boundaries
	if len(b) < 2 {
		return math.NaN()
	}

	if t <= 0 {
		return b[0]
	}

	if t >= 1 {
		return b[len(b)-1]
	}

	cells := len(b) - 1

	idx := int(t * float64(cells))
	if idx >= cells {
		idx = cells - 1
	}

	return 0.5 * (b[idx] + b[idx+1])
}

// Bounds returns the first and last boundary values.
func (n *BoundaryNorm) Bounds() (float64, float64) {
	if len(n.Boundaries) < 2 {
		return 0, 0
	}

	return n.Boundaries[0], n.Boundaries[len(n.Boundaries)-1]
}

// Train is a no-op: BoundaryNorm bounds are user-specified.
func (n *BoundaryNorm) Train(_ dataset.AnyColumn) error { return nil }

// AsinhNorm performs a smooth asinh (inverse-hyperbolic-sine) transform that
// behaves linearly near zero and logarithmically far from zero — useful for
// data spanning many decades that includes zeros or negatives.
type AsinhNorm struct {
	LinearWidth float64
	Vmin, Vmax  float64
	trained     bool
}

// Norm maps v through an inverse-hyperbolic-sine transform to [0, 1].
func (n *AsinhNorm) Norm(v float64) float64 {
	if math.IsNaN(v) {
		return math.NaN()
	}

	a := n.scaledAsinh(v)
	lo := n.scaledAsinh(n.Vmin)

	hi := n.scaledAsinh(n.Vmax)
	if hi == lo {
		return 0.5
	}

	return (a - lo) / (hi - lo)
}

// Inverse maps t in [0, 1] back to the asinh-scaled data range.
func (n *AsinhNorm) Inverse(t float64) float64 {
	lo := n.scaledAsinh(n.Vmin)
	hi := n.scaledAsinh(n.Vmax)
	a := lo + t*(hi-lo)
	w := n.linearWidth()

	return w * math.Sinh(a)
}

// Bounds returns the current trained data range.
func (n *AsinhNorm) Bounds() (float64, float64) { return n.Vmin, n.Vmax }

// Train expands the range to cover values in col.
func (n *AsinhNorm) Train(col dataset.AnyColumn) error {
	mn, mx, ok, err := minMaxColumn(col)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	if !n.trained {
		n.Vmin, n.Vmax = mn, mx
		n.trained = true

		return nil
	}

	if mn < n.Vmin {
		n.Vmin = mn
	}

	if mx > n.Vmax {
		n.Vmax = mx
	}

	return nil
}

func (n *AsinhNorm) linearWidth() float64 {
	if n.LinearWidth <= 0 {
		return 1
	}

	return n.LinearWidth
}

func (n *AsinhNorm) scaledAsinh(v float64) float64 {
	w := n.linearWidth()
	return math.Asinh(v / w)
}

// minMaxColumn extracts (min, max) from a numeric column. Returns ok=false
// if the column has zero length. Returns an error for non-numeric columns.
func minMaxColumn(col dataset.AnyColumn) (mn, mx float64, ok bool, err error) {
	switch c := col.(type) {
	case dataset.Column[float64]:
		vs := c.Values()
		if len(vs) == 0 {
			return 0, 0, false, nil
		}

		mn = math.Inf(1)
		mx = math.Inf(-1)

		for _, v := range vs {
			if math.IsNaN(v) {
				continue
			}

			if v < mn {
				mn = v
			}

			if v > mx {
				mx = v
			}
		}

		if math.IsInf(mn, 1) {
			return 0, 0, false, nil
		}

		return mn, mx, true, nil
	case dataset.Column[int64]:
		vs := c.Values()
		if len(vs) == 0 {
			return 0, 0, false, nil
		}

		mn = float64(vs[0])
		mx = mn

		for _, v := range vs[1:] {
			fv := float64(v)
			if fv < mn {
				mn = fv
			}

			if fv > mx {
				mx = fv
			}
		}

		return mn, mx, true, nil
	default:
		return 0, 0, false, fmt.Errorf("colormap: column %q has non-numeric type %s: %w", col.Name(), col.DType(), ErrParseColor)
	}
}

// minMaxPositive is like minMaxColumn but rejects values <= 0 (used by LogNorm).
func minMaxPositive(col dataset.AnyColumn) (mn, mx float64, ok bool, err error) {
	switch c := col.(type) {
	case dataset.Column[float64]:
		vs := c.Values()
		mn = math.Inf(1)
		mx = math.Inf(-1)

		for _, v := range vs {
			if math.IsNaN(v) || v <= 0 {
				continue
			}

			if v < mn {
				mn = v
			}

			if v > mx {
				mx = v
			}
		}

		if math.IsInf(mn, 1) {
			return 0, 0, false, nil
		}

		return mn, mx, true, nil
	case dataset.Column[int64]:
		vs := c.Values()
		mn = math.Inf(1)
		mx = math.Inf(-1)

		for _, v := range vs {
			if v <= 0 {
				continue
			}

			fv := float64(v)
			if fv < mn {
				mn = fv
			}

			if fv > mx {
				mx = fv
			}
		}

		if math.IsInf(mn, 1) {
			return 0, 0, false, nil
		}

		return mn, mx, true, nil
	default:
		return 0, 0, false, fmt.Errorf("colormap: column %q has non-numeric type %s: %w", col.Name(), col.DType(), ErrParseColor)
	}
}
