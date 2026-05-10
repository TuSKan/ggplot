package scale

import (
	"sort"

	"github.com/TuSKan/ggplot/dataset"
)

// Discrete returns a discrete (categorical) scale that maps string labels
// to evenly-spaced numeric positions. Categories are extracted during Train
// and sorted alphabetically for deterministic ordering.
func Discrete() *DiscreteScale {
	return &DiscreteScale{}
}

// DiscreteScale maps categorical string values to evenly-spaced [0, N-1]
// positions where N is the number of unique categories.
type DiscreteScale struct {
	categories []string       // sorted unique labels
	catIndex   map[string]int // label → index
	trained    bool
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

// Bounds returns the domain extent with 0.5 padding around categories.
func (s *DiscreteScale) Bounds() (float64, float64) {
	if len(s.categories) == 0 {
		return 0, 1
	}

	return -0.5, float64(len(s.categories)) - 0.5
}

func (s *DiscreteScale) String() string { return "discrete" }

// SetBounds overrides the domain bounds (used by pipeline padding logic).
func (s *DiscreteScale) SetBounds(_, _ float64) {
	// No-op: discrete scale bounds are derived from category count.
	// But we store them if forced.
}
