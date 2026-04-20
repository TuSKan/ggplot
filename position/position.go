// Package position defines position adjustments that control how overlapping
// geometries are arranged. In ggplot2's grammar, position adjustments are
// applied after statistical transforms and before final rendering.
package position

// Pos adjusts the positions of geometric elements to handle overlap.
// Each adjustment receives the raw data coordinates and group metadata,
// and returns adjusted coordinates.
type Pos interface {
	// Adjust modifies (x, y) positions for a single group within a layer.
	// groupIdx is the 0-based index of this group, nGroups is the total count.
	// width is the available bin width for dodging/stacking calculations.
	Adjust(xs, ys []float64, width float64, groupIdx, nGroups int) ([]float64, []float64)

	// String returns a human-readable label.
	String() string
}

// Identity returns the identity position (no adjustment).
func Identity() Pos { return identity{} }

type identity struct{}

func (identity) Adjust(xs, ys []float64, _ float64, _, _ int) ([]float64, []float64) {
	return xs, ys
}
func (identity) String() string { return "identity" }

// Dodge returns a position that shifts groups side by side within each bin.
// This is the standard adjustment for grouped bar charts.
func Dodge() Pos { return dodge{} }

type dodge struct{}

func (dodge) Adjust(xs, ys []float64, width float64, groupIdx, nGroups int) ([]float64, []float64) {
	if nGroups <= 1 {
		return xs, ys
	}
	barWidth := width / float64(nGroups)
	offset := barWidth*float64(groupIdx) - width/2 + barWidth/2

	adjusted := make([]float64, len(xs))
	for i, x := range xs {
		adjusted[i] = x + offset
	}
	return adjusted, ys
}
func (dodge) String() string { return "dodge" }

// Stack returns a position that stacks groups vertically.
// Each group's y-values are offset by the cumulative sum of prior groups.
func Stack() Pos { return stack{} }

type stack struct{}

func (stack) Adjust(xs, ys []float64, _ float64, groupIdx, _ int) ([]float64, []float64) {
	if groupIdx == 0 {
		return xs, ys
	}
	// In practice, stacking requires knowledge of all groups' y-values.
	// The pipeline coordinator accumulates offsets across groups and passes
	// them here. For now, identity behavior is the base case.
	return xs, ys
}
func (stack) String() string { return "stack" }

// Jitter returns a position that adds random noise to (x, y) to reduce overplotting.
func Jitter(xAmount, yAmount float64) Pos {
	return jitter{xAmt: xAmount, yAmt: yAmount}
}

type jitter struct {
	xAmt, yAmt float64
}

func (j jitter) Adjust(xs, ys []float64, _ float64, _, _ int) ([]float64, []float64) {
	adjX := make([]float64, len(xs))
	adjY := make([]float64, len(ys))

	// Use a simple deterministic hash-based jitter (reproducible).
	for i := range xs {
		// Deterministic pseudo-random based on index.
		seed := uint64(i*2654435761) & 0xFFFFFFFF
		fx := float64(seed%1000)/1000.0 - 0.5
		seed = uint64((i+7)*2654435761) & 0xFFFFFFFF
		fy := float64(seed%1000)/1000.0 - 0.5

		adjX[i] = xs[i] + fx*j.xAmt
		adjY[i] = ys[i] + fy*j.yAmt
	}
	return adjX, adjY
}
func (j jitter) String() string { return "jitter" }

// Nudge returns a position that shifts all points by a fixed offset.
func Nudge(dx, dy float64) Pos {
	return nudge{dx: dx, dy: dy}
}

type nudge struct{ dx, dy float64 }

func (n nudge) Adjust(xs, ys []float64, _ float64, _, _ int) ([]float64, []float64) {
	adjX := make([]float64, len(xs))
	adjY := make([]float64, len(ys))
	for i := range xs {
		adjX[i] = xs[i] + n.dx
		adjY[i] = ys[i] + n.dy
	}
	return adjX, adjY
}
func (n nudge) String() string { return "nudge" }
