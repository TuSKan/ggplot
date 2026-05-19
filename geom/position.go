// Position adjustments control how overlapping geometries are arranged.
// In the grammar of graphics, position adjustments are applied after
// statistical transforms and before final rendering.

package geom

import "math/rand/v2"

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

// Stacker is optionally implemented by positions that produce stacked output.
// When implemented, AdjustStack returns (adjustedXs, yMin, yMax) where yMin is
// the bottom of each bar and yMax is the top. The pipeline uses yMin to set
// the bar's base coordinate.
type Stacker interface {
	AdjustStack(xs, ys []float64, width float64, groupIdx, nGroups int) (adjXs, yMin, yMax []float64)
}

// FillSetup is optionally implemented by position adjustments that need
// a pre-pass over all groups before per-group Adjust calls.
type FillSetup interface {
	Setup(allXs, allYs [][]float64)
}

// PosName identifies a position adjustment type for factory lookup.
type PosName string

// Position adjustment names.
const (
	PosIdentity PosName = "identity"
	PosDodge    PosName = "dodge"
	PosStack    PosName = "stack"
	PosFill     PosName = "fill"
	PosJitter   PosName = "jitter"
	PosNudge    PosName = "nudge"
)

// NewPos creates a fresh Pos instance by name. This is used by the build
// pipeline to create per-panel-layer position instances (avoiding shared
// state across panels). Returns IdentityPos for unknown names.
func NewPos(name PosName) Pos {
	switch name {
	case PosIdentity, "":
		return IdentityPos()
	case PosDodge:
		return Dodge()
	case PosStack:
		return Stack()
	case PosFill:
		return Fill()
	case PosJitter, PosNudge:
		return IdentityPos() // Jitter/Nudge require parameters; pipeline uses layer's instance directly
	default:
		return IdentityPos()
	}
}

// IdentityPos returns the identity position (no adjustment).
func IdentityPos() Pos { return identityPos{} }

type identityPos struct{}

func (identityPos) Adjust(xs, ys []float64, _ float64, _, _ int) ([]float64, []float64) {
	return xs, ys
}
func (identityPos) String() string { return string(PosIdentity) }

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
func (dodge) String() string { return string(PosDodge) }

// Stack returns a position that stacks groups vertically.
// Each group's y-values are offset by the cumulative sum of prior groups
// at the same x-value.
//
// Stack is stateful: it accumulates offsets across successive Adjust calls.
// The build pipeline must create a fresh Stack() instance per panel-layer
// to avoid cross-panel contamination.
func Stack() Pos { return &stack{offsets: make(map[float64]float64)} }

type stack struct {
	offsets map[float64]float64 // x-value -> cumulative Y offset
}

func (s *stack) Adjust(xs, ys []float64, w float64, gi, ng int) ([]float64, []float64) {
	_, _, yMax := s.AdjustStack(xs, ys, w, gi, ng)
	return xs, yMax
}

func (s *stack) AdjustStack(xs, ys []float64, _ float64, _, _ int) (adjXs, yMin, yMax []float64) {
	yMin = make([]float64, len(ys))
	yMax = make([]float64, len(ys))

	for i, x := range xs {
		base := s.offsets[x]
		yMin[i] = base
		yMax[i] = base + ys[i]
		s.offsets[x] = base + ys[i]
	}

	return xs, yMin, yMax
}

func (s *stack) String() string { return string(PosStack) }

// Fill returns a position that stacks groups and normalizes each x-bin
// to a total of 1.0 (100% stacked bar chart).
//
// Fill is a two-phase adjustment:
//  1. Setup phase: call [FillSetup.Setup] with all groups' (xs, ys) to compute totals.
//  2. Adjust phase: call Adjust for each group in order.
//
// If Setup is not called, Fill behaves like Stack (no normalization).
func Fill() Pos { return &fill{offsets: make(map[float64]float64)} }

type fill struct {
	totals  map[float64]float64 // x-value -> total Y across all groups
	offsets map[float64]float64 // x-value -> cumulative Y offset (normalized)
}

// Setup computes the total Y for each x-bin across all groups.
func (f *fill) Setup(allXs, allYs [][]float64) {
	f.totals = make(map[float64]float64)

	for gi := range allXs {
		for i, x := range allXs[gi] {
			if i < len(allYs[gi]) {
				f.totals[x] += allYs[gi][i]
			}
		}
	}
}

func (f *fill) Adjust(xs, ys []float64, w float64, gi, ng int) ([]float64, []float64) {
	_, _, yMax := f.AdjustStack(xs, ys, w, gi, ng)
	return xs, yMax
}

func (f *fill) AdjustStack(xs, ys []float64, _ float64, _, _ int) (adjXs, yMin, yMax []float64) {
	yMin = make([]float64, len(ys))
	yMax = make([]float64, len(ys))

	for i, x := range xs {
		total := f.totals[x]
		base := f.offsets[x]

		if total > 0 {
			normalized := ys[i] / total
			yMin[i] = base
			yMax[i] = base + normalized
			f.offsets[x] = base + normalized
		} else {
			yMin[i] = base
			yMax[i] = base
		}
	}

	return xs, yMin, yMax
}
func (f *fill) String() string { return string(PosFill) }

// Jitter returns a position that adds random noise to (x, y) to reduce overplotting.
// The jitter is reproducible: same data length produces same offsets.
func Jitter(xAmount, yAmount float64) Pos {
	return jitter{xAmt: xAmount, yAmt: yAmount}
}

type jitter struct {
	xAmt, yAmt float64
}

func (j jitter) Adjust(xs, ys []float64, _ float64, _, _ int) ([]float64, []float64) {
	adjX := make([]float64, len(xs))
	adjY := make([]float64, len(ys))

	// Reproducible PRNG seeded by data length for deterministic-per-dataset behavior.
	rng := rand.New(rand.NewPCG(42, uint64(len(xs)))) //nolint:gosec // G404: reproducible jitter uses math/rand intentionally; crypto not needed.
	for i := range xs {
		adjX[i] = xs[i] + (rng.Float64()-0.5)*j.xAmt
		adjY[i] = ys[i] + (rng.Float64()-0.5)*j.yAmt
	}

	return adjX, adjY
}
func (j jitter) String() string { return string(PosJitter) }

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
func (n nudge) String() string { return string(PosNudge) }
