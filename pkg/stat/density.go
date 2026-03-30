package stat

import (
	"fmt"
	"math"

	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
)

type DensityOptions struct {
	Bandwidth float64
	Points    int
}

type Density struct {
	Options DensityOptions
}

func NewDensity(opts DensityOptions) Stat {
	if opts.Points <= 0 {
		opts.Points = 512 // Default KDE resolution
	}
	return Density{Options: opts}
}

func (s Density) Name() string { return "density" }
func (s Density) Kind() Kind   { return KindSummary }

func (s Density) Compute(ctx Context) (dataset.Dataset, error) {
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
		return nil, fmt.Errorf("stat: column %q is not iterable natively", colName)
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

	n := len(vals)
	if n == 0 {
		buf := arrow.NewBuffer(0)
		buf.Float64("x")
		buf.Float64("density")
		return buf.Build()
	}

	bw := s.Options.Bandwidth
	if bw <= 0 {
		// Scott's rule of thumb (simplified version)
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		mean := sum / float64(n)
		variance := 0.0
		for _, v := range vals {
			variance += (v - mean) * (v - mean)
		}
		stdDev := math.Sqrt(variance / float64(n))
		bw = 1.06 * stdDev * math.Pow(float64(n), -1.0/5.0)
		if bw == 0 {
			bw = 1.0
		}
	}

	// Range expands slightly
	startX := minX - 3.0*bw
	endX := maxX + 3.0*bw
	step := (endX - startX) / float64(s.Options.Points-1)

	buf := arrow.NewBuffer(s.Options.Points)
	xCol := buf.Float64("x")
	yCol := buf.Float64("density")

	invBandwidth := 1.0 / bw
	normFactor := 1.0 / (float64(n) * bw * math.Sqrt(2.0*math.Pi))

	for i := 0; i < s.Options.Points; i++ {
		curX := startX + float64(i)*step
		xCol[i] = curX

		density := 0.0
		for _, v := range vals {
			u := (curX - v) * invBandwidth
			density += math.Exp(-0.5 * u * u)
		}
		yCol[i] = density * normFactor
	}

	return buf.Build()
}

func init() {
	Register(NewDensity(DensityOptions{}))
}
