package memory

import (
	"fmt"
	"math"

	"github.com/TuSKan/ggplot/dataset"
	dmath "github.com/TuSKan/ggplot/dataset/math"
)

// --- MathKernel: highway SIMD on raw slices ---

// applyUnaryFloat64 applies a scalar function element-wise on float64 column data.
func (e *Engine) applyUnaryFloat64(col dataset.AnyColumn, fn func(float64) float64) (dataset.AnyColumn, error) {
	c, ok := col.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("memory: MathKernel requires float64 column, got %T", col)
	}
	out := make([]float64, len(c.data))
	for i, v := range c.data {
		out[i] = fn(v)
	}
	return &float64Column{name: c.name, data: out}, nil
}

// applySliceTransform applies a highway (dst, src) transform on float64 column data.
func (e *Engine) applySliceTransform(col dataset.AnyColumn, fn func(dst, src []float64)) (dataset.AnyColumn, error) {
	c, ok := col.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("memory: MathKernel requires float64 column, got %T", col)
	}
	out := make([]float64, len(c.data))
	fn(out, c.data)
	return &float64Column{name: c.name, data: out}, nil
}

// applyBinaryFloat64 applies a binary function element-wise on two float64 columns.
func (e *Engine) applyBinaryFloat64(a, b dataset.AnyColumn, fn func(float64, float64) float64) (dataset.AnyColumn, error) {
	ca, ok := a.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("memory: MathKernel requires float64 column, got %T", a)
	}
	cb, ok := b.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("memory: MathKernel requires float64 column, got %T", b)
	}
	if len(ca.data) != len(cb.data) {
		return nil, fmt.Errorf("memory: column length mismatch: %d vs %d", len(ca.data), len(cb.data))
	}
	out := make([]float64, len(ca.data))
	for i := range ca.data {
		out[i] = fn(ca.data[i], cb.data[i])
	}
	return &float64Column{name: ca.name, data: out}, nil
}

// applyBinaryInt64 applies a binary function element-wise on two int64 columns.
func (e *Engine) applyBinaryInt64(a, b dataset.AnyColumn, fn func(int64, int64) int64) (dataset.AnyColumn, error) {
	ca, ok := a.(*int64Column)
	if !ok {
		return nil, fmt.Errorf("memory: bitwise op requires int64 column, got %T", a)
	}
	cb, ok := b.(*int64Column)
	if !ok {
		return nil, fmt.Errorf("memory: bitwise op requires int64 column, got %T", b)
	}
	if len(ca.data) != len(cb.data) {
		return nil, fmt.Errorf("memory: column length mismatch: %d vs %d", len(ca.data), len(cb.data))
	}
	out := make([]int64, len(ca.data))
	for i := range ca.data {
		out[i] = fn(ca.data[i], cb.data[i])
	}
	return &int64Column{name: ca.name, data: out}, nil
}

// --- Binary arithmetic ---

func (e *Engine) AddCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyBinaryFloat64(a, b, func(x, y float64) float64 { return x + y })
}

func (e *Engine) SubCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyBinaryFloat64(a, b, func(x, y float64) float64 { return x - y })
}

func (e *Engine) MulCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyBinaryFloat64(a, b, func(x, y float64) float64 { return x * y })
}

func (e *Engine) DivCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyBinaryFloat64(a, b, func(x, y float64) float64 { return x / y })
}

// --- Scalar arithmetic ---

func (e *Engine) AddScalar(col dataset.AnyColumn, val float64) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, func(v float64) float64 { return v + val })
}

func (e *Engine) MulScalar(col dataset.AnyColumn, val float64) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, func(v float64) float64 { return v * val })
}

// --- Unary numeric ---

func (e *Engine) Abs(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, math.Abs)
}

func (e *Engine) Neg(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, func(v float64) float64 { return -v })
}

func (e *Engine) Sign(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, func(v float64) float64 {
		if v > 0 {
			return 1
		}
		if v < 0 {
			return -1
		}
		return 0
	})
}

func (e *Engine) Sqrt(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, math.Sqrt)
}

func (e *Engine) Pow(col dataset.AnyColumn, exp float64) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, func(v float64) float64 { return math.Pow(v, exp) })
}

// --- Transcendental: logarithmic ---

func (e *Engine) Exp(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applySliceTransform(col, dmath.Exp[float64])
}

func (e *Engine) Ln(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applySliceTransform(col, dmath.Log[float64])
}

func (e *Engine) Log2(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, math.Log2)
}

func (e *Engine) Log10(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, math.Log10)
}

// --- Transcendental: trigonometric ---

func (e *Engine) Sin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applySliceTransform(col, dmath.Sin[float64])
}

func (e *Engine) Cos(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applySliceTransform(col, dmath.Cos[float64])
}

func (e *Engine) Tan(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, math.Tan)
}

func (e *Engine) Asin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, math.Asin)
}

func (e *Engine) Acos(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, math.Acos)
}

func (e *Engine) Atan(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, math.Atan)
}

func (e *Engine) Atan2(y, x dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyBinaryFloat64(y, x, math.Atan2)
}

// --- Transcendental: hyperbolic/special (highway) ---

func (e *Engine) Tanh(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applySliceTransform(col, dmath.Tanh[float64])
}

func (e *Engine) Sigmoid(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applySliceTransform(col, dmath.Sigmoid[float64])
}

func (e *Engine) Erf(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applySliceTransform(col, dmath.Erf[float64])
}

// --- Rounding ---

func (e *Engine) Round(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, math.Round)
}

func (e *Engine) Floor(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, math.Floor)
}

func (e *Engine) Ceil(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryFloat64(col, math.Ceil)
}

// --- Bitwise (int64 only) ---

func (e *Engine) BitAnd(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyBinaryInt64(a, b, func(x, y int64) int64 { return x & y })
}

func (e *Engine) BitOr(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyBinaryInt64(a, b, func(x, y int64) int64 { return x | y })
}

func (e *Engine) BitXor(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyBinaryInt64(a, b, func(x, y int64) int64 { return x ^ y })
}

func (e *Engine) BitNot(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	c, ok := col.(*int64Column)
	if !ok {
		return nil, fmt.Errorf("memory: BitNot requires int64 column, got %T", col)
	}
	out := make([]int64, len(c.data))
	for i, v := range c.data {
		out[i] = ^v
	}
	return &int64Column{name: c.name, data: out}, nil
}

func (e *Engine) BitShiftLeft(col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	c, ok := col.(*int64Column)
	if !ok {
		return nil, fmt.Errorf("memory: BitShiftLeft requires int64 column, got %T", col)
	}
	out := make([]int64, len(c.data))
	for i, v := range c.data {
		out[i] = v << uint(n)
	}
	return &int64Column{name: c.name, data: out}, nil
}

func (e *Engine) BitShiftRight(col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	c, ok := col.(*int64Column)
	if !ok {
		return nil, fmt.Errorf("memory: BitShiftRight requires int64 column, got %T", col)
	}
	out := make([]int64, len(c.data))
	for i, v := range c.data {
		out[i] = v >> uint(n)
	}
	return &int64Column{name: c.name, data: out}, nil
}
