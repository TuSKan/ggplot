package scale

import (
	"sort"

	"github.com/TuSKan/ggplot/dataset"
)

// DiscreteOpt configures a [DiscreteScale].
type DiscreteOpt func(*DiscreteScale)

// WithPaddingInner sets the inner padding between bars as a fraction of
// the band step [0, 1). 0 means bars touch; 0.2 (default) leaves a 20% gap.
func WithPaddingInner(f float64) DiscreteOpt {
	return func(s *DiscreteScale) { s.PaddingInner = f }
}

// WithPaddingOuter sets the outer padding before the first and after the last
// category as a fraction of the band step [0, ∞). 0.5 (default) gives half
// a step of breathing room on each side.
func WithPaddingOuter(f float64) DiscreteOpt {
	return func(s *DiscreteScale) { s.PaddingOuter = f }
}

// Discrete returns a discrete (categorical) scale that maps string labels
// to evenly-spaced numeric positions. Categories are extracted during Train
// and sorted alphabetically for deterministic ordering.
func Discrete(opts ...DiscreteOpt) *DiscreteScale {
	s := &DiscreteScale{
		PaddingInner: -1, // sentinel: use default
		PaddingOuter: -1, // sentinel: use default
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

// DiscreteScale maps categorical string values to evenly-spaced [0, N-1]
// positions where N is the number of unique categories.
type DiscreteScale struct {
	categories   []string       // sorted unique labels
	catIndex     map[string]int // label → index
	trained      bool
	PaddingInner float64 // gap between bars as fraction of step [0,1). Default 0.2
	PaddingOuter float64 // padding before first / after last category. Default 0.5
}

// Train extracts unique string values from the column. If the column is
// already numeric (e.g. after stat transforms), this is a no-op.
func (s *DiscreteScale) Train(col dataset.AnyColumn) error {
	sc, ok := col.(dataset.Column[string])
	if !ok {
		return nil // not a string column — skip
	}

	seen := make(map[string]struct{})

	if s.catIndex != nil {
		for _, c := range s.categories {
			seen[c] = struct{}{}
		}
	}

	for _, v := range sc.Values() {
		if v == "" {
			continue
		}

		seen[v] = struct{}{}
	}

	// Rebuild sorted categories.
	s.categories = make([]string, 0, len(seen))
	for c := range seen {
		s.categories = append(s.categories, c)
	}

	sort.Strings(s.categories)

	s.catIndex = make(map[string]int, len(s.categories))
	for i, c := range s.categories {
		s.catIndex[c] = i
	}

	s.trained = true

	return nil
}

// TrainValues trains the scale from a pre-built slice of category labels.
func (s *DiscreteScale) TrainValues(labels []string) {
	seen := make(map[string]struct{}, len(labels))
	for _, l := range labels {
		seen[l] = struct{}{}
	}

	s.categories = make([]string, 0, len(seen))
	for c := range seen {
		s.categories = append(s.categories, c)
	}

	sort.Strings(s.categories)

	s.catIndex = make(map[string]int, len(s.categories))
	for i, c := range s.categories {
		s.catIndex[c] = i
	}

	s.trained = true
}

// MapCategory maps a category label to its numeric position.
func (s *DiscreteScale) MapCategory(label string) float64 {
	if idx, ok := s.catIndex[label]; ok {
		return float64(idx)
	}

	return 0
}

// Categories returns the ordered category labels.
func (s *DiscreteScale) Categories() []string {
	return s.categories
}

// --- Scale interface ---

// Map transforms a category index to a [0, 1] fraction.
func (s *DiscreteScale) Map(v float64) float64 {
	n := float64(len(s.categories) - 1)
	if n <= 0 {
		return 0.5
	}

	return v / n
}

// Inverse maps a [0, 1] fraction back to a category index.
func (s *DiscreteScale) Inverse(v float64) float64 {
	n := float64(len(s.categories) - 1)
	if n <= 0 {
		return 0
	}

	return v * n
}

// Ticks returns a tick position for each category.
func (s *DiscreteScale) Ticks(_ int) []float64 {
	ticks := make([]float64, len(s.categories))
	for i := range ticks {
		ticks[i] = float64(i)
	}

	return ticks
}

// Format returns the category label for the nearest index.
func (s *DiscreteScale) Format(v float64) string {
	idx := int(v + 0.5) // round to nearest
	if idx >= 0 && idx < len(s.categories) {
		return s.categories[idx]
	}

	return FormatNumber(v)
}

// Bounds returns the domain extent with outer padding around categories.
// PaddingOuter controls the padding on each side as a fraction of the step;
// the default is 0.5 (half a step on each side).
func (s *DiscreteScale) Bounds() (float64, float64) {
	if len(s.categories) == 0 {
		return 0, 1
	}

	outer := s.PaddingOuter
	if outer < 0 {
		outer = 0.5 //nolint:mnd // Default outer padding: half a step.
	}

	return -outer, float64(len(s.categories)-1) + outer
}

func (s *DiscreteScale) String() string { return "discrete" }

// SetBounds overrides the domain bounds (used by pipeline padding logic).
func (s *DiscreteScale) SetBounds(_, _ float64) {
	// No-op: discrete scale bounds are derived from category count.
	// But we store them if forced.
}

// BandWidth returns the bar/band width as a fraction of the step, derived
// from PaddingInner. A PaddingInner of 0.2 yields BandWidth 0.8, meaning
// bars occupy 80% of each slot. Returns the default (0.8) when PaddingInner
// is unset.
func (s *DiscreteScale) BandWidth() float64 {
	if s.PaddingInner < 0 {
		return 0.8 //nolint:mnd // Default band width: 80% of step.
	}

	w := 1 - s.PaddingInner
	if w < 0.1 { //nolint:mnd // Floor at 10% to prevent invisible bars.
		w = 0.1
	}

	return w
}
