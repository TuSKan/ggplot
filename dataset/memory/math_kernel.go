package memory

import (
	"fmt"
	"math"

	"github.com/TuSKan/ggplot/dataset"
	simd "github.com/TuSKan/ggplot/dataset/compute"
	dmath "github.com/TuSKan/ggplot/dataset/math"
)

// --- MathKernel: direct loops on raw slices ---
//
// All arithmetic and unary numeric ops use direct inline loops.
// This lets the Go SSA backend optimise freely (bounds-check elimination,
// loop unrolling) and avoids the heap-escaping Vec[T] overhead that
// go-highway's scalar fallback incurs when the function is not inlined.
//
// dmath slice-transforms (Exp, Ln, Sin, Cos, Tanh, Sigmoid, Erf)
// use sliceTransformFloat64 because dmath manages its own vectorisation.

// requireFloat64 extracts the raw slice from a float64 column.
func requireFloat64(col dataset.AnyColumn) (*float64Column, error) {
	c, ok := col.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("memory: MathKernel requires float64 column, got %T", col)
	}
	return c, nil
}

// requireFloat64Pair extracts two equal-length float64 columns.
func requireFloat64Pair(a, b dataset.AnyColumn) (*float64Column, *float64Column, error) {
	ca, err := requireFloat64(a)
	if err != nil {
		return nil, nil, err
	}
	cb, err := requireFloat64(b)
	if err != nil {
		return nil, nil, err
	}
	if len(ca.data) != len(cb.data) {
		return nil, nil, fmt.Errorf("memory: column length mismatch: %d vs %d", len(ca.data), len(cb.data))
	}
	return ca, cb, nil
}

// scalarUnaryFloat64 applies a scalar function element-wise on float64 column data.
// Used only for ops without an inlineable operator (Sign, trig, rounding, log2/log10).
func (e *Engine) scalarUnaryFloat64(col dataset.AnyColumn, fn func(float64) float64) (dataset.AnyColumn, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(c.data))
	for i, v := range c.data {
		out[i] = fn(v)
	}
	return &float64Column{name: c.name, data: out}, nil
}

// sliceTransformFloat64 applies a highway (dst, src) slice transform on float64 column data.
// Used by dmath SIMD transcendentals (Exp, Log, Sin, Cos, Tanh, Sigmoid, Erf).
func (e *Engine) sliceTransformFloat64(col dataset.AnyColumn, fn func(dst, src []float64)) (dataset.AnyColumn, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(c.data))
	fn(out, c.data)
	return &float64Column{name: c.name, data: out}, nil
}

// --- Binary arithmetic (compute.Slice* dual-path: SIMD when available, scalar otherwise) ---

func (e *Engine) AddCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	ca, cb, err := requireFloat64Pair(a, b)
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(ca.data))
	simd.SliceAdd(out, ca.data, cb.data)
	return &float64Column{name: ca.name, data: out}, nil
}

func (e *Engine) SubCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	ca, cb, err := requireFloat64Pair(a, b)
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(ca.data))
	simd.SliceSub(out, ca.data, cb.data)
	return &float64Column{name: ca.name, data: out}, nil
}

func (e *Engine) MulCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	ca, cb, err := requireFloat64Pair(a, b)
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(ca.data))
	simd.SliceMul(out, ca.data, cb.data)
	return &float64Column{name: ca.name, data: out}, nil
}

func (e *Engine) DivCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	ca, cb, err := requireFloat64Pair(a, b)
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(ca.data))
	simd.SliceDiv(out, ca.data, cb.data)
	return &float64Column{name: ca.name, data: out}, nil
}

// --- Scalar arithmetic (compute.Slice* dual-path) ---

func (e *Engine) AddScalar(col dataset.AnyColumn, val float64) (dataset.AnyColumn, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(c.data))
	simd.SliceAddScalar(out, c.data, val)
	return &float64Column{name: c.name, data: out}, nil
}

func (e *Engine) MulScalar(col dataset.AnyColumn, val float64) (dataset.AnyColumn, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(c.data))
	simd.SliceMulScalar(out, c.data, val)
	return &float64Column{name: c.name, data: out}, nil
}

// --- Unary numeric (compute.Slice* dual-path where available, scalar otherwise) ---

func (e *Engine) Abs(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(c.data))
	simd.SliceAbs(out, c.data)
	return &float64Column{name: c.name, data: out}, nil
}

func (e *Engine) Neg(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(c.data))
	simd.SliceNeg(out, c.data)
	return &float64Column{name: c.name, data: out}, nil
}

func (e *Engine) Sign(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, func(v float64) float64 {
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
	c, err := requireFloat64(col)
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(c.data))
	simd.SliceSqrt(out, c.data)
	return &float64Column{name: c.name, data: out}, nil
}

func (e *Engine) Pow(col dataset.AnyColumn, exp float64) (dataset.AnyColumn, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(c.data))
	simd.SlicePow(out, c.data, exp)
	return &float64Column{name: c.name, data: out}, nil
}

// --- Transcendental: logarithmic ---

func (e *Engine) Exp(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.sliceTransformFloat64(col, dmath.Exp[float64])
}

func (e *Engine) Ln(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.sliceTransformFloat64(col, dmath.Log[float64])
}

func (e *Engine) Log2(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Log2)
}

func (e *Engine) Log10(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Log10)
}

// --- Transcendental: trigonometric ---

func (e *Engine) Sin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.sliceTransformFloat64(col, dmath.Sin[float64])
}

func (e *Engine) Cos(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.sliceTransformFloat64(col, dmath.Cos[float64])
}

func (e *Engine) Tan(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Tan)
}

func (e *Engine) Asin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Asin)
}

func (e *Engine) Acos(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Acos)
}

func (e *Engine) Atan(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Atan)
}

func (e *Engine) Atan2(y, x dataset.AnyColumn) (dataset.AnyColumn, error) {
	cy, ok := y.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("memory: Atan2 requires float64 column, got %T", y)
	}
	cx, ok := x.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("memory: Atan2 requires float64 column, got %T", x)
	}
	if len(cy.data) != len(cx.data) {
		return nil, fmt.Errorf("memory: column length mismatch: %d vs %d", len(cy.data), len(cx.data))
	}
	out := make([]float64, len(cy.data))
	for i := range cy.data {
		out[i] = math.Atan2(cy.data[i], cx.data[i])
	}
	return &float64Column{name: cy.name, data: out}, nil
}

// --- Transcendental: hyperbolic/special (highway) ---

func (e *Engine) Tanh(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.sliceTransformFloat64(col, dmath.Tanh[float64])
}

func (e *Engine) Sigmoid(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.sliceTransformFloat64(col, dmath.Sigmoid[float64])
}

func (e *Engine) Erf(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.sliceTransformFloat64(col, dmath.Erf[float64])
}

// --- Rounding ---

func (e *Engine) Round(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Round)
}

func (e *Engine) Floor(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Floor)
}

func (e *Engine) Ceil(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Ceil)
}

// --- Bitwise (int64 only, direct inline loops) ---

// requireInt64Pair extracts two equal-length int64 columns.
func requireInt64Pair(a, b dataset.AnyColumn) (*int64Column, *int64Column, error) {
	ca, ok := a.(*int64Column)
	if !ok {
		return nil, nil, fmt.Errorf("memory: bitwise op requires int64 column, got %T", a)
	}
	cb, ok := b.(*int64Column)
	if !ok {
		return nil, nil, fmt.Errorf("memory: bitwise op requires int64 column, got %T", b)
	}
	if len(ca.data) != len(cb.data) {
		return nil, nil, fmt.Errorf("memory: column length mismatch: %d vs %d", len(ca.data), len(cb.data))
	}
	return ca, cb, nil
}

func (e *Engine) BitAnd(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	ca, cb, err := requireInt64Pair(a, b)
	if err != nil {
		return nil, err
	}
	out := make([]int64, len(ca.data))
	for i := range ca.data {
		out[i] = ca.data[i] & cb.data[i]
	}
	return &int64Column{name: ca.name, data: out}, nil
}

func (e *Engine) BitOr(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	ca, cb, err := requireInt64Pair(a, b)
	if err != nil {
		return nil, err
	}
	out := make([]int64, len(ca.data))
	for i := range ca.data {
		out[i] = ca.data[i] | cb.data[i]
	}
	return &int64Column{name: ca.name, data: out}, nil
}

func (e *Engine) BitXor(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	ca, cb, err := requireInt64Pair(a, b)
	if err != nil {
		return nil, err
	}
	out := make([]int64, len(ca.data))
	for i := range ca.data {
		out[i] = ca.data[i] ^ cb.data[i]
	}
	return &int64Column{name: ca.name, data: out}, nil
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
