package stat

import (
	"context"
	"fmt"
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// StackY returns a Transform that cumulatively stacks y values.
// Produces a "ymin" column containing the base of each stacked segment:
// ymin = CumSum(y) - y, and replaces y with CumSum(y).
//
// Uses the engine's Windower.CumSum and MathKernel.SubCols — no manual
// float64 loops. Returns a lazy Dataset.
//
// Unlike geom.Stack() (a position adjustment that offsets bar positions
// across color groups), StackY is a data transform that accumulates
// y values within a single group's pipeline.
func StackY() Transform {
	return &stackYTransform{}
}

type stackYTransform struct{}

func (s *stackYTransform) Name() string           { return "stackY" }
func (s *stackYTransform) OutputSchema() []string { return []string{"ymin"} }

func (s *stackYTransform) OutputMapping() map[string]string { return nil }

func (s *stackYTransform) OutputHints() map[string]ChannelHint {
	return map[string]ChannelHint{"y": HintCumulative}
}

func (s *stackYTransform) Apply(ctx context.Context, in TransformInput) (TransformResult, error) {
	return applyStackAxis(ctx, in, "y", "ymin", "stackY")
}

// StackX returns a Transform that cumulatively stacks x values.
// Produces an "xmin" column containing the base of each stacked segment:
// xmin = CumSum(x) - x, and replaces x with CumSum(x).
//
// Uses the engine's Windower.CumSum and MathKernel.SubCols — no manual
// float64 loops. Returns a lazy Dataset.
//
// This is the horizontal counterpart of [StackY].
func StackX() Transform {
	return &stackXTransform{}
}

type stackXTransform struct{}

func (s *stackXTransform) Name() string           { return "stackX" }
func (s *stackXTransform) OutputSchema() []string { return []string{"xmin"} }

func (s *stackXTransform) OutputMapping() map[string]string { return nil }

func (s *stackXTransform) OutputHints() map[string]ChannelHint {
	return map[string]ChannelHint{"x": HintCumulative}
}

func (s *stackXTransform) Apply(ctx context.Context, in TransformInput) (TransformResult, error) {
	return applyStackAxis(ctx, in, "x", "xmin", "stackX")
}

// applyStackAxis implements the shared cumulative-stacking logic for both
// StackY (axis="y", minCol="ymin") and StackX (axis="x", minCol="xmin").
func applyStackAxis(_ context.Context, in TransformInput, axis, minCol, label string) (TransformResult, error) {
	valCol := in.Mapping[axis]

	if valCol == "" {
		return TransformResult{}, fmt.Errorf("%s: missing '%s' aesthetic: %w", label, axis, ErrMissingColumn)
	}

	eng := dataset.GetEngine(in.Data.Table())

	win, ok := eng.(dataset.Windower)
	if !ok {
		return TransformResult{}, fmt.Errorf("%s: engine %q: Windower: %w",
			label, eng.Name(), dataset.ErrUnsupportedEngine)
	}

	mk, ok := eng.(dataset.MathKernel)
	if !ok {
		return TransformResult{}, fmt.Errorf("%s: engine %q: MathKernel: %w",
			label, eng.Name(), dataset.ErrUnsupportedEngine)
	}

	col, err := in.Data.Column(valCol)
	if err != nil {
		return TransformResult{}, fmt.Errorf("%s: %w", label, err)
	}

	// CumSum gives us the stacked value (top of each segment).
	cumSumCol, err := win.CumSum(col)
	if err != nil {
		return TransformResult{}, fmt.Errorf("%s: cumsum: %w", label, err)
	}

	// min = cumsum - original (base of each segment).
	baseCol, err := mk.SubCols(cumSumCol, col)
	if err != nil {
		return TransformResult{}, fmt.Errorf("%s: %s: %w", label, minCol, err)
	}

	// Lazy: all Dataset verbs — materialized on Collect.
	outData := in.Data.
		WithColumn(baseCol).    // replace axis with min values
		Rename(valCol, minCol). // rename to minCol (lazy)
		WithColumn(cumSumCol)   // add cumsum back as axis

	outMapping := make(map[string]string, len(in.Mapping))
	maps.Copy(outMapping, in.Mapping)

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
