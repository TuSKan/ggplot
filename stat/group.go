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

// aggFuncForReducer maps a reducer name to an engine-native AggFunc.
// Returns the AggFunc and true if the reducer is engine-supported.
func aggFuncForReducer(name string) (dataset.AggFunc, bool) {
	switch name {
	case "sum":
		return dataset.AggSum, true
	case "mean":
		return dataset.AggMean, true
	case "min":
		return dataset.AggMin, true
	case "max":
		return dataset.AggMax, true
	case "count":
		return dataset.AggCount, true
	case "median":
		return dataset.AggMedian, true
	case "variance":
		return dataset.AggVariance, true
	default:
		return 0, false
	}
}

func (g *groupTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
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

	// Try engine-native GroupBy + Summarize for known aggregations.
	// Stays lazy — materialized only when the pipeline caller Collects.
	if aggFn, ok := aggFuncForReducer(g.reducerName); ok {
		outData := in.Data.
			GroupBy(groupCol).
			Summarize(dataset.AggSpec{
				OutputName: valueCol,
				InputName:  valueCol,
				Fn:         aggFn,
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
