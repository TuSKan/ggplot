package stat

import (
	"context"
	"fmt"
	"maps"

	"github.com/TuSKan/ggplot/dataset"
)

// GroupX returns a Transform that groups data by the x channel and
// applies a named reducer to the y channel within each group.
// Produces sorted unique x values and the reduced y values.
//
// For engine-supported aggregations (sum, mean, min, max, count,
// median, variance), delegates to the engine's Aggregator interface
// via Dataset.GroupBy().Summarize(). For custom reducers, falls back
// to scalar computation.
//
// GroupX("mean") is equivalent to the existing SummaryXY transform.
func GroupX(reducer string) Transform {
	return &groupTransform{axis: "x", reducerName: reducer}
}

// GroupY returns a Transform that groups by y and reduces x.
func GroupY(reducer string) Transform {
	return &groupTransform{axis: "y", reducerName: reducer}
}

type groupTransform struct {
	axis        string // "x" groups by x and reduces y; "y" groups by y and reduces x
	reducerName string
}

func (g *groupTransform) Name() string { return "group" + g.axis }

func (g *groupTransform) OutputSchema() []string { return []string{"x", "y"} }

func (g *groupTransform) OutputMapping() map[string]string { return nil }

func (g *groupTransform) OutputHints() map[string]ChannelHint { return nil }

// aggFuncForReducer maps a reducer name to an engine-native AggFunc and an
// optional percentile parameter. The float64 is only meaningful when the
// returned AggFunc is AggPercentile.
func aggFuncForReducer(name string) (dataset.AggFunc, float64, bool) {
	switch name {
	case "sum":
		return dataset.AggSum, 0, true
	case "mean":
		return dataset.AggMean, 0, true
	case "min":
		return dataset.AggMin, 0, true
	case "max":
		return dataset.AggMax, 0, true
	case "count":
		return dataset.AggCount, 0, true
	case "median":
		return dataset.AggMedian, 0, true
	case "variance":
		return dataset.AggVariance, 0, true
	case "deviation", "stddev":
		return dataset.AggStdDev, 0, true
	case "first":
		return dataset.AggFirst, 0, true
	case "last":
		return dataset.AggLast, 0, true
	case "mode":
		return dataset.AggMode, 0, true
	case "p10":
		return dataset.AggPercentile, 0.10, true //nolint:mnd // p10 = 10th percentile.
	case "p25":
		return dataset.AggPercentile, 0.25, true //nolint:mnd // p25 = 25th percentile.
	case "p50":
		return dataset.AggPercentile, 0.50, true //nolint:mnd // p50 = 50th percentile (median).
	case "p75":
		return dataset.AggPercentile, 0.75, true //nolint:mnd // p75 = 75th percentile.
	case "p90":
		return dataset.AggPercentile, 0.90, true //nolint:mnd // p90 = 90th percentile.
	default:
		return 0, 0, false
	}
}

func (g *groupTransform) Apply(ctx context.Context, in TransformInput) (TransformResult, error) {
	// Determine group-by and value axes.
	groupAxis := g.axis
	valueAxis := "y"

	if g.axis == "y" {
		valueAxis = "x"
	}

	groupCol := in.Mapping[groupAxis]
	valueCol := in.Mapping[valueAxis]

	if groupCol == "" || valueCol == "" {
		return TransformResult{}, fmt.Errorf("group%s: missing %q or %q aesthetic: %w",
			g.axis, groupAxis, valueAxis, ErrMissingColumn)
	}

	// Handle proportion: count per group / total count.
	if g.reducerName == "proportion" || g.reducerName == "proportion-facet" {
		return g.applyProportion(ctx, in, groupCol, valueCol)
	}

	// Try engine-native GroupBy + Summarize for known aggregations.
	// Stays lazy — materialized only when the pipeline caller Collects.
	if aggFn, p, ok := aggFuncForReducer(g.reducerName); ok {
		outData := in.Data.
			GroupBy(groupCol).
			Summarize(dataset.AggSpec{
				OutputName: valueCol,
				InputName:  valueCol,
				Fn:         aggFn,
				P:          p,
			}).
			Arrange(groupCol)

		outMapping := make(map[string]string, len(in.Mapping)+len(g.OutputMapping()))
		maps.Copy(outMapping, in.Mapping)
		maps.Copy(outMapping, g.OutputMapping())

		return TransformResult{Data: outData, Mapping: outMapping}, nil
	}

	return TransformResult{}, fmt.Errorf("group%s: reducer %q not supported — extend engine Aggregator: %w",
		g.axis, g.reducerName, ErrUnsupportedType)
}

// applyProportion computes count-per-group / total-count.
// Fully lazy: GroupBy+Summarize+Arrange → Mutate(proportionMutator).
// No materialization in stat/ — the Mutate runs at Collect time inside the engine.
func (g *groupTransform) applyProportion(_ context.Context, in TransformInput, groupCol, valueCol string) (TransformResult, error) {
	total := float64(in.Data.NumRows())
	if total == 0 {
		total = 1 // avoid divide by zero
	}

	// Lazy: count per group → sort → mutate count→proportion.
	outData := in.Data.
		GroupBy(groupCol).
		Summarize(dataset.AggSpec{
			OutputName: valueCol,
			InputName:  valueCol,
			Fn:         dataset.AggCount,
		}).
		Arrange(groupCol).
		Mutate(valueCol, &proportionMutator{col: valueCol, invTotal: 1.0 / total})

	outMapping := make(map[string]string, len(in.Mapping))
	maps.Copy(outMapping, in.Mapping)

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}

// proportionMutator implements dataset.MutateFunc.
// At Collect time it reads the count column, casts int64→float64,
// and multiplies by 1/total — all through engine interfaces.
type proportionMutator struct {
	col      string  // column to transform
	invTotal float64 // 1 / total count
}

func (m *proportionMutator) Apply(tbl dataset.Table) (dataset.AnyColumn, error) {
	eng := dataset.GetEngine(tbl)

	col, err := tbl.Column(m.col)
	if err != nil {
		return nil, fmt.Errorf("proportion: %w", err)
	}

	// AggCount produces int64 — cast to float64 for MulScalar.
	caster, ok := eng.(dataset.Caster)
	if !ok {
		return nil, fmt.Errorf("proportion: engine %q: Caster: %w", eng.Name(), dataset.ErrUnsupportedEngine)
	}

	fcol, err := caster.Cast(col, dataset.DTypeFloat64)
	if err != nil {
		return nil, fmt.Errorf("proportion: cast: %w", err)
	}

	mk, ok := eng.(dataset.MathKernel)
	if !ok {
		return nil, fmt.Errorf("proportion: engine %q: MathKernel: %w", eng.Name(), dataset.ErrUnsupportedEngine)
	}

	result, mulErr := mk.MulScalar(fcol, m.invTotal)
	if mulErr != nil {
		return nil, fmt.Errorf("proportion: %w", mulErr)
	}

	return result, nil
}

// --- Dual-axis grouping ---

// Group returns a Transform that groups data and applies different reducers
// to both axes simultaneously. The group-by column is auto-detected from
// the "color" or "group" aesthetic in the mapping; use [WithGroupBy] to
// override.
//
// Example:
//
//	stat.Group("mean", "sum")                        // mean(x), sum(y), group by color
//	stat.Group("p50", "max", stat.WithGroupBy("id")) // p50(x), max(y), group by id
func Group(xReducer, yReducer string, opts ...GroupOption) Transform {
	cfg := groupDualConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	return &dualGroupTransform{
		xReducer: xReducer,
		yReducer: yReducer,
		groupBy:  cfg.groupBy,
	}
}

// GroupOption configures the [Group] transform.
type GroupOption func(*groupDualConfig)

type groupDualConfig struct {
	groupBy string // explicit group-by column; "" = auto-detect
}

// WithGroupBy sets an explicit group-by column for [Group].
// When empty, the group-by column is auto-detected from the "color"
// or "group" aesthetic.
func WithGroupBy(col string) GroupOption {
	return func(c *groupDualConfig) { c.groupBy = col }
}

type dualGroupTransform struct {
	xReducer string
	yReducer string
	groupBy  string
}

func (d *dualGroupTransform) Name() string           { return "group" }
func (d *dualGroupTransform) OutputSchema() []string { return []string{"x", "y"} }

func (d *dualGroupTransform) OutputMapping() map[string]string { return nil }

func (d *dualGroupTransform) OutputHints() map[string]ChannelHint { return nil }

func (d *dualGroupTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	// Resolve group-by column: explicit → color → group.
	groupCol := d.groupBy
	if groupCol == "" {
		if c, ok := in.Mapping["color"]; ok && c != "" {
			groupCol = c
		} else if g, ok := in.Mapping["group"]; ok && g != "" {
			groupCol = g
		}
	}

	if groupCol == "" {
		return TransformResult{}, fmt.Errorf("group: no group-by column — set color/group aesthetic or use WithGroupBy: %w", ErrMissingColumn)
	}

	xCol := in.Mapping["x"]
	yCol := in.Mapping["y"]

	if xCol == "" || yCol == "" {
		return TransformResult{}, fmt.Errorf("group: missing 'x' or 'y' aesthetic: %w", ErrMissingColumn)
	}

	xFn, xP, xOk := aggFuncForReducer(d.xReducer)
	if !xOk {
		return TransformResult{}, fmt.Errorf("group: x reducer %q not supported: %w", d.xReducer, ErrUnsupportedType)
	}

	yFn, yP, yOk := aggFuncForReducer(d.yReducer)
	if !yOk {
		return TransformResult{}, fmt.Errorf("group: y reducer %q not supported: %w", d.yReducer, ErrUnsupportedType)
	}

	outData := in.Data.
		GroupBy(groupCol).
		Summarize(
			dataset.AggSpec{OutputName: xCol, InputName: xCol, Fn: xFn, P: xP},
			dataset.AggSpec{OutputName: yCol, InputName: yCol, Fn: yFn, P: yP},
		).
		Arrange(groupCol)

	outMapping := make(map[string]string, len(in.Mapping))
	maps.Copy(outMapping, in.Mapping)

	return TransformResult{Data: outData, Mapping: outMapping}, nil
}
