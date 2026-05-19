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

func (s *stackYTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	yCol := in.Mapping["y"]

	if yCol == "" {
		return TransformResult{}, fmt.Errorf("stackY: missing 'y' aesthetic: %w", ErrMissingColumn)
	}

	eng := dataset.GetEngine(in.Data.Table())

	win, ok := eng.(dataset.Windower)
	if !ok {
		return TransformResult{}, fmt.Errorf("stackY: engine %q: Windower: %w",
			eng.Name(), dataset.ErrUnsupportedEngine)
	}

	mk, ok := eng.(dataset.MathKernel)
	if !ok {
		return TransformResult{}, fmt.Errorf("stackY: engine %q: MathKernel: %w",
			eng.Name(), dataset.ErrUnsupportedEngine)
	}

	col, err := in.Data.Column(yCol)
	if err != nil {
		return TransformResult{}, fmt.Errorf("stackY: %w", err)
	}

	// CumSum gives us the stacked y (top of each bar).
	cumSumCol, err := win.CumSum(col)
	if err != nil {
		return TransformResult{}, fmt.Errorf("stackY: cumsum: %w", err)
	}

	// ymin = cumsum - original_y (base of each bar).
	// SubCols inherits the input column name ('y').
	// Strategy: replace y with ymin values, rename to 'ymin', then add cumsum as 'y'.
	yminCol, err := mk.SubCols(cumSumCol, col)
	if err != nil {
		return TransformResult{}, fmt.Errorf("stackY: ymin: %w", err)
	}

	// Lazy: all Dataset verbs — materialized on Collect.
	outData := in.Data.
		WithColumn(yminCol).  // replace 'y' with ymin values
		Rename(yCol, "ymin"). // rename to 'ymin' (lazy)
		WithColumn(cumSumCol) // add cumsum back as 'y'

	outMapping := make(map[string]string, len(in.Mapping))
	maps.Copy(outMapping, in.Mapping)

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
