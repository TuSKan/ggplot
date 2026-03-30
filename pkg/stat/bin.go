package stat

import (
	"fmt"
	"math"

	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
)

type BinOptions struct {
	Bins int
}

type Bin struct {
	Options BinOptions
}

func NewBin(opts BinOptions) Stat {
	if opts.Bins <= 0 {
		opts.Bins = 30
	}
	return Bin{Options: opts}
}
func (s Bin) Name() string { return "bin" }
func (s Bin) Kind() Kind   { return KindAggregate }

func (s Bin) Compute(ctx Context) (dataset.Dataset, error) {
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

	fltIter, err := iterCol.Float64s()
	if err != nil {
		return nil, err
	}

	var vals []float64
	minX, maxX := math.MaxFloat64, -math.MaxFloat64
	for {
		v, isNull, ok := fltIter.Next()
		if !ok {
			break
		}
		if !isNull {
			if v < minX {
				minX = v
			}
			if v > maxX {
				maxX = v
			}
			vals = append(vals, v)
		}
	}

	if len(vals) == 0 {
		buf := arrow.NewBuffer(0)
		buf.Float64("x")
		buf.Float64("count")
		return buf.Build()
	}

	if minX == maxX {
		maxX = minX + 1.0 // span
	}

	span := maxX - minX
	width := span / float64(s.Options.Bins)

	counts := make([]int64, s.Options.Bins)
	for _, v := range vals {
		bin := int((v - minX) / width)
		if bin >= s.Options.Bins {
			bin = s.Options.Bins - 1
		} else if bin < 0 {
			bin = 0
		}
		counts[bin]++
	}

	buf := arrow.NewBuffer(s.Options.Bins)
	xCol := buf.Float64("x")
	cntCol := buf.Float64("count")

	for i := 0; i < s.Options.Bins; i++ {
		xCol[i] = minX + (float64(i)+0.5)*width
		cntCol[i] = float64(counts[i])
	}

	return buf.Build()
}

func init() {
	Register(NewBin(BinOptions{}))
}
