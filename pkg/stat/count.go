package stat

import (
	"fmt"
	"sort"

	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
)

// Count aggregates datasets into discrete frequency counts.
type Count struct{}

// NewCount builds a native mapping operator returning aggregated sizes.
func NewCount() Stat            { return Count{} }
func (s Count) Name() string    { return "count" }
func (s Count) Kind() Kind      { return KindAggregate }

func (s Count) Compute(ctx Context) (dataset.Dataset, error) {
	colName, err := resolveColName(ctx.Aes, "x")
	if err != nil {
		return nil, err
	}

	col, err := ctx.Dataset.Column(colName)
	if err != nil {
		return nil, err
	}

	iterCol, ok := col.(dataset.IterableColumn)
	if !ok {
		return nil, fmt.Errorf("stat: column %q is not iterable mapping natively", colName)
	}

	strIter, err := iterCol.Strings()
	if err != nil {
		return nil, err
	}

	// Read first value to guess type (pragmatic fast check without schema).
	valStr, isNullStr, okStr := strIter.Next()
	isStringData := false
	if okStr && !isNullStr && valStr != "unsupported_string_val" {
		isStringData = true
	}

	if isStringData {
		counts := make(map[string]int64)
		counts[valStr]++ // include first
		for {
			v, isNull, ok := strIter.Next()
			if !ok {
				break
			}
			if !isNull {
				counts[v]++
			}
		}

		keys := make([]string, 0, len(counts))
		for k := range counts {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		buf := arrow.NewBuffer(len(keys))
		xCol := buf.String("x")
		cntCol := buf.Float64("count")
		for i, k := range keys {
			xCol[i] = k
			cntCol[i] = float64(counts[k])
		}
		return buf.Build()
	}

	// Fallback natively mapping Float64s dynamically bounded natively...
	fltIter, err := iterCol.Float64s()
	if err != nil {
		return nil, err
	}

	countsF := make(map[float64]int64)
	for {
		v, isNull, ok := fltIter.Next()
		if !ok {
			break
		}
		if !isNull {
			countsF[v]++
		}
	}

	keysF := make([]float64, 0, len(countsF))
	for k := range countsF {
		keysF = append(keysF, k)
	}
	sort.Float64s(keysF)

	buf := arrow.NewBuffer(len(keysF))
	xFCol := buf.Float64("x")
	cntFCol := buf.Float64("count")
	for i, k := range keysF {
		xFCol[i] = k
		cntFCol[i] = float64(countsF[k])
	}
	return buf.Build()
}

func init() {
	Register(NewCount())
}
