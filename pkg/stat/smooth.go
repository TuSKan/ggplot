package stat

import (
	"fmt"
	"math"

	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
)

type SmoothMethod string

const (
	MethodLinear SmoothMethod = "lm"
)

type SmoothOptions struct {
	Method SmoothMethod
	Points int
}

// Smooth generates regression lines interpolating continuous trends.
type Smooth struct {
	Options SmoothOptions
}

func NewSmooth(opts SmoothOptions) Stat {
	if opts.Points <= 0 {
		opts.Points = 80
	}
	if opts.Method == "" {
		opts.Method = MethodLinear
	}
	return Smooth{Options: opts}
}

func (s Smooth) Name() string { return "smooth" }
func (s Smooth) Kind() Kind   { return KindSummary }

func (s Smooth) Compute(ctx Context) (dataset.Dataset, error) {
	if s.Options.Method != MethodLinear {
		return nil, &ErrUnsupportedMethod{Stat: "smooth", Method: string(s.Options.Method)}
	}

	xName, err := resolveColName(ctx.Aes, "x")
	if err != nil {
		return nil, err
	}
	yName, err := resolveColName(ctx.Aes, "y")
	if err != nil {
		return nil, err
	}

	xCol, err := ctx.Dataset.Column(xName)
	if err != nil {
		return nil, err
	}
	yCol, err := ctx.Dataset.Column(yName)
	if err != nil {
		return nil, err
	}

	xIterCol, ok := xCol.(dataset.IterableColumn)
	if !ok {
		return nil, fmt.Errorf("stat: column %q is not iterable", xName)
	}
	yIterCol, ok := yCol.(dataset.IterableColumn)
	if !ok {
		return nil, fmt.Errorf("stat: column %q is not iterable", yName)
	}

	xIter, err := xIterCol.Float64s()
	if err != nil {
		return nil, err
	}
	yIter, err := yIterCol.Float64s()
	if err != nil {
		return nil, err
	}

	var xVals, yVals []float64
	sumX, sumY, sumXX, sumXY := 0.0, 0.0, 0.0, 0.0
	minX, maxX := math.MaxFloat64, -math.MaxFloat64

	n := 0.0

	for {
		vx, xNull, okX := xIter.Next()
		vy, yNull, okY := yIter.Next()
		if !okX || !okY {
			break
		}
		if xNull || yNull {
			continue
		}

		xVals = append(xVals, vx)
		yVals = append(yVals, vy)
		
		if vx < minX {
			minX = vx
		}
		if vx > maxX {
			maxX = vx
		}

		sumX += vx
		sumY += vy
		sumXX += vx * vx
		sumXY += vx * vy
		n += 1.0
	}

	if n < 2 {
		buf := arrow.NewBuffer(0)
		buf.Float64("x")
		buf.Float64("y")
		return buf.Build()
	}

	// Linear regression: y = beta * x + alpha
	denominator := (n * sumXX) - (sumX * sumX)
	var beta, alpha float64
	if denominator == 0 {
		beta = 0
		alpha = sumY / n
	} else {
		beta = ((n * sumXY) - (sumX * sumY)) / denominator
		alpha = (sumY - (beta * sumX)) / n
	}

	buf := arrow.NewBuffer(s.Options.Points)
	resX := buf.Float64("x")
	resY := buf.Float64("y")

	step := (maxX - minX) / float64(s.Options.Points-1)
	if s.Options.Points == 1 {
		step = 0
	}

	for i := 0; i < s.Options.Points; i++ {
		curX := minX + float64(i)*step
		resX[i] = curX
		resY[i] = beta*curX + alpha
	}

	return buf.Build()
}

func init() {
	Register(NewSmooth(SmoothOptions{}))
}
