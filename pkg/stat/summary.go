package stat

import (
	"fmt"
	"sort"

	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
)

type SummaryFun string

const (
	FunMean   SummaryFun = "mean"
	FunMedian SummaryFun = "median"
	FunMin    SummaryFun = "min"
	FunMax    SummaryFun = "max"
)

type SummaryOptions struct {
	Fun SummaryFun
}

// Summary groups data locally aggregating subsets dynamically bounding values.
type Summary struct {
	Options SummaryOptions
}

func NewSummary(opts SummaryOptions) Stat {
	if opts.Fun == "" {
		opts.Fun = FunMean
	}
	return Summary{Options: opts}
}

func (s Summary) Name() string { return "summary" }
func (s Summary) Kind() Kind   { return KindSummary }

func (s Summary) Compute(ctx Context) (dataset.Dataset, error) {
	xName, errX := resolveColName(ctx.Aes, "x")
	yName, errY := resolveColName(ctx.Aes, "y")
	if errY != nil {
		return nil, errY
	}

	yCol, err := ctx.Dataset.Column(yName)
	if err != nil {
		return nil, err
	}
	yIterCol, ok := yCol.(dataset.IterableColumn)
	if !ok {
		return nil, fmt.Errorf("stat: column %q is not iterable mapping", yName)
	}

	yIter, err := yIterCol.Float64s()
	if err != nil {
		return nil, err
	}

	// We'll group by "x" if "x" is bound softly.
	var xIter dataset.Float64Iterator
	hasX := errX == nil

	if hasX {
		xCol, err := ctx.Dataset.Column(xName)
		if err != nil {
			return nil, err
		}
		xIterCol, ok := xCol.(dataset.IterableColumn)
		if !ok {
			return nil, fmt.Errorf("stat: column %q is not iterable mapping", xName)
		}
		xIter, err = xIterCol.Float64s()
		if err != nil {
			return nil, err
		}
	}

	groups := make(map[float64][]float64)

	for {
		vy, yNull, okY := yIter.Next()
		if !okY {
			break
		}
		if yNull {
			if hasX {
				xIter.Next()
			}
			continue
		}

		groupKey := 0.0 // Global group natively
		if hasX {
			vx, xNull, okX := xIter.Next()
			if !okX || xNull {
				continue
			}
			groupKey = vx
		}

		groups[groupKey] = append(groups[groupKey], vy)
	}

	keys := make([]float64, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Float64s(keys)

	buf := arrow.NewBuffer(len(keys))
	var resX []float64
	if hasX {
		resX = buf.Float64("x")
	}
	resY := buf.Float64("y")

	for i, k := range keys {
		if hasX {
			resX[i] = k
		}
		vals := groups[k]

		switch s.Options.Fun {
		case FunMean:
			sum := 0.0
			for _, v := range vals {
				sum += v
			}
			resY[i] = sum / float64(len(vals))
		case FunMedian:
			sort.Float64s(vals)
			mid := len(vals) / 2
			if len(vals)%2 == 0 {
				resY[i] = (vals[mid-1] + vals[mid]) / 2.0
			} else {
				resY[i] = vals[mid]
			}
		case FunMin:
			minVal := vals[0]
			for _, v := range vals {
				if v < minVal {
					minVal = v
				}
			}
			resY[i] = minVal
		case FunMax:
			maxVal := vals[0]
			for _, v := range vals {
				if v > maxVal {
					maxVal = v
				}
			}
			resY[i] = maxVal
		default:
			return nil, &ErrUnsupportedMethod{Stat: "summary", Method: string(s.Options.Fun)}
		}
	}

	return buf.Build()
}

func init() {
	Register(NewSummary(SummaryOptions{}))
}
