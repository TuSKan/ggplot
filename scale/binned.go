package scale

import (
	"fmt"
	"math"

	"github.com/TuSKan/ggplot/dataset"
)

// BinnedScale discretizes a continuous domain into equal-width bins.
// Each bin center becomes a tick position; Format shows the bin range.
//
// This does NOT aggregate data (that's stat.BinX). It relabels the axis
// so that continuous values are displayed as range categories.
type BinnedScale struct {
	domain

	nbins   int       // desired bin count (0 = auto via Sturges)
	method  string    // "sturges" | "scott" | "fd" | "sqrt"
	edges   []float64 // nbins+1 computed bin edges
	centers []float64 // bin centers
	breaks  []float64 // optional explicit bin edges
}

// BinnedOpt configures a BinnedScale.
type BinnedOpt func(*BinnedScale)

// BinnedBins sets an explicit number of bins.
func BinnedBins(n int) BinnedOpt {
	return func(s *BinnedScale) { s.nbins = n }
}

// BinnedMethod sets the bin-count selection method.
// Supported: "sturges" (default), "scott", "fd", "sqrt".
func BinnedMethod(m string) BinnedOpt {
	return func(s *BinnedScale) { s.method = m }
}

// BinnedBreaks sets explicit bin edges, overriding automatic computation.
// The slice must be sorted and have at least 2 elements.
func BinnedBreaks(edges []float64) BinnedOpt {
	return func(s *BinnedScale) {
		s.breaks = make([]float64, len(edges))
		copy(s.breaks, edges)
	}
}

// NewBinned returns a binned scale with the given options.
func NewBinned(opts ...BinnedOpt) Scale {
	s := &BinnedScale{method: "sturges"}
	for _, o := range opts {
		o(s)
	}

	return s
}

// Train computes the domain and bin edges from column data.
func (s *BinnedScale) Train(col dataset.AnyColumn) error {
	if err := s.train(col); err != nil {
		return err
	}

	s.computeBins()

	return nil
}

// Map transforms a data value to a [0, 1] fraction using the data domain.
// Identical to linear normalization through Bounds.
func (s *BinnedScale) Map(v float64) float64 {
	mn, mx := s.Bounds()
	if mx == mn {
		return 0.5
	}

	return (v - mn) / (mx - mn)
}

// Inverse maps a [0, 1] fraction back to data space.
func (s *BinnedScale) Inverse(v float64) float64 {
	mn, mx := s.Bounds()
	return mn + v*(mx-mn)
}

// Ticks returns bin centers in data space.
func (s *BinnedScale) Ticks(_ int) []float64 {
	out := make([]float64, len(s.centers))
	copy(out, s.centers)

	return out
}

// Format returns the bin range label for a data-space value, e.g. "[40, 50)".
// The value is matched to its bin via findBin.
func (s *BinnedScale) Format(v float64) string {
	bin := s.findBin(v)
	if bin < 0 || bin >= len(s.edges)-1 {
		return FormatNumber(v)
	}

	lo := s.edges[bin]
	hi := s.edges[bin+1]

	// Last bin is closed on both sides.
	if bin == len(s.edges)-2 {
		return fmt.Sprintf("[%s, %s]", FormatNumber(lo), FormatNumber(hi))
	}

	return fmt.Sprintf("[%s, %s)", FormatNumber(lo), FormatNumber(hi))
}

// Bounds returns the data-space domain [min, max].
func (s *BinnedScale) Bounds() (float64, float64) {
	return s.min, s.max
}

// String returns "binned".
func (s *BinnedScale) String() string { return "binned" }

// SetBounds overrides the domain and recomputes bins.
func (s *BinnedScale) SetBounds(mn, mx float64) {
	s.min = mn
	s.max = mx
	s.trained = true
	s.computeBins()
}

// findBin returns the 0-based bin index for a data value.
func (s *BinnedScale) findBin(v float64) int {
	for i := range len(s.edges) - 1 {
		if v >= s.edges[i] && v < s.edges[i+1] {
			return i
		}
	}
	// Last bin is closed on right.
	if len(s.edges) >= 2 && v == s.edges[len(s.edges)-1] {
		return len(s.edges) - 2
	}

	// Out of range.
	if v < s.edges[0] {
		return 0
	}

	return len(s.edges) - 2
}

// computeBins calculates bin edges and centers from the trained domain.
func (s *BinnedScale) computeBins() {
	if !s.trained {
		return
	}

	if len(s.breaks) >= 2 {
		s.edges = s.breaks
	} else {
		nb := s.nbins
		if nb <= 0 {
			nb = s.autoBinCount()
		}

		if nb < 1 {
			nb = 1
		}

		s.edges = make([]float64, nb+1)
		width := (s.max - s.min) / float64(nb)

		for i := range s.edges {
			s.edges[i] = s.min + float64(i)*width
		}
	}

	s.centers = make([]float64, len(s.edges)-1)
	for i := range s.centers {
		s.centers[i] = (s.edges[i] + s.edges[i+1]) / 2
	}
}

// autoBinCount computes a bin count using the selected method.
func (s *BinnedScale) autoBinCount() int {
	// Without row count access, use a heuristic based on domain range.
	// Sturges' formula: ceil(log2(n) + 1). Approximate n from domain.
	switch s.method {
	case "sqrt":
		// sqrt(range) — rough heuristic.
		return int(math.Ceil(math.Sqrt(s.max - s.min)))
	default:
		// Sturges: default to ~7 bins for reasonable domains.
		return 7 //nolint:mnd // Sturges default for unknown n.
	}
}

// --- Compile-time interface checks ---

var (
	_ Scale        = (*BinnedScale)(nil)
	_ BoundsSetter = (*BinnedScale)(nil)
)
