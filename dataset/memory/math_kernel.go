package memory

import (
	"fmt"
	"math"

	"github.com/TuSKan/ggplot/dataset"
	simd "github.com/TuSKan/ggplot/dataset/compute"
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
		return nil, fmt.Errorf("MathKernel: got %T: %w", col, ErrRequiresFloat64)
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
		return nil, nil, fmt.Errorf("memory: column length mismatch: %d vs %d: %w", len(ca.data), len(cb.data), ErrLengthMismatch)
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

// MapFloat64 implements [dataset.MathKernel]. It applies a scalar function
// element-wise on a float64 column. This is the generic fallback for
// domain-specific transforms not covered by named SIMD-accelerated operations.
func (e *Engine) MapFloat64(col dataset.AnyColumn, fn func(float64) float64) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, fn)
}

// sliceTransformFloat64 applies a highway (dst, src) slice transform on float64 column data.
// Currently unused: go-highway v0.0.12 AVX2 codegen bug causes SIGILL on AMD EPYC.
// Will be re-enabled when the upstream fix lands.
//
//nolint:unused // retained for future dmath SIMD re-enablement
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

// AddCols returns element-wise addition of two float64 columns.
func (e *Engine) AddCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	ca, cb, err := requireFloat64Pair(a, b)
	if err != nil {
		return nil, err
	}

	out := make([]float64, len(ca.data))
	simd.SliceAdd(out, ca.data, cb.data)

	return &float64Column{name: ca.name, data: out}, nil
}

// SubCols returns element-wise subtraction of two float64 columns.
func (e *Engine) SubCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	ca, cb, err := requireFloat64Pair(a, b)
	if err != nil {
		return nil, err
	}

	out := make([]float64, len(ca.data))
	simd.SliceSub(out, ca.data, cb.data)

	return &float64Column{name: ca.name, data: out}, nil
}

// MulCols returns element-wise multiplication of two float64 columns.
func (e *Engine) MulCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	ca, cb, err := requireFloat64Pair(a, b)
	if err != nil {
		return nil, err
	}

	out := make([]float64, len(ca.data))
	simd.SliceMul(out, ca.data, cb.data)

	return &float64Column{name: ca.name, data: out}, nil
}

// DivCols returns element-wise division of two float64 columns.
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

// AddScalar adds a scalar value to each element of a float64 column.
func (e *Engine) AddScalar(col dataset.AnyColumn, val float64) (dataset.AnyColumn, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, err
	}

	out := make([]float64, len(c.data))
	simd.SliceAddScalar(out, c.data, val)

	return &float64Column{name: c.name, data: out}, nil
}

// MulScalar multiplies each element of a float64 column by a scalar value.
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

// Abs returns the absolute value of each element.
func (e *Engine) Abs(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, err
	}

	out := make([]float64, len(c.data))
	simd.SliceAbs(out, c.data)

	return &float64Column{name: c.name, data: out}, nil
}

// Neg returns the negation of each element.
func (e *Engine) Neg(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, err
	}

	out := make([]float64, len(c.data))
	simd.SliceNeg(out, c.data)

	return &float64Column{name: c.name, data: out}, nil
}

// Sign returns the sign of each element (-1, 0, or 1).
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

// Sqrt returns the square root of each element.
func (e *Engine) Sqrt(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, err
	}

	out := make([]float64, len(c.data))
	simd.SliceSqrt(out, c.data)

	return &float64Column{name: c.name, data: out}, nil
}

// Pow raises each element to the given exponent.
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
// NOTE: Using scalar math.* loops instead of dmath SIMD transcendentals.
// go-highway v0.0.12 has an AVX2 codegen bug: the _avx2.gen.go files emit
// EVEX-encoded (AVX-512) instructions, causing SIGILL on CPUs with AVX2
// but not AVX-512 (e.g., AMD EPYC 7763 on GitHub Actions).
// Tracked upstream: github.com/ajroetker/go-highway

// Exp returns e raised to the power of each element.
func (e *Engine) Exp(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Exp)
}

// Ln returns the natural logarithm of each element.
func (e *Engine) Ln(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Log)
}

// Log2 returns the base-2 logarithm of each element.
func (e *Engine) Log2(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Log2)
}

// Log10 returns the base-10 logarithm of each element.
func (e *Engine) Log10(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Log10)
}

// --- Transcendental: trigonometric ---

// Sin returns the sine of each element (radians).
func (e *Engine) Sin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Sin)
}

// Cos returns the cosine of each element (radians).
func (e *Engine) Cos(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Cos)
}

// Tan returns the tangent of each element (radians).
func (e *Engine) Tan(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Tan)
}

// Asin returns the arcsine of each element.
func (e *Engine) Asin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Asin)
}

// Acos returns the arccosine of each element.
func (e *Engine) Acos(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Acos)
}

// Atan returns the arctangent of each element.
func (e *Engine) Atan(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Atan)
}

// Atan2 returns the two-argument arctangent of (y, x).
func (e *Engine) Atan2(y, x dataset.AnyColumn) (dataset.AnyColumn, error) {
	cy, ok := y.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("Atan2: got %T: %w", y, ErrRequiresFloat64)
	}

	cx, ok := x.(*float64Column)
	if !ok {
		return nil, fmt.Errorf("Atan2: got %T: %w", x, ErrRequiresFloat64)
	}

	if len(cy.data) != len(cx.data) {
		return nil, fmt.Errorf("memory: column length mismatch: %d vs %d: %w", len(cy.data), len(cx.data), ErrLengthMismatch)
	}

	out := make([]float64, len(cy.data))
	for i := range cy.data {
		out[i] = math.Atan2(cy.data[i], cx.data[i])
	}

	return &float64Column{name: cy.name, data: out}, nil
}

// --- Transcendental: hyperbolic/special (scalar — see go-highway note above) ---

// Tanh returns the hyperbolic tangent of each element.
func (e *Engine) Tanh(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Tanh)
}

// Sigmoid returns the logistic sigmoid of each element.
func (e *Engine) Sigmoid(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, func(x float64) float64 {
		return 1.0 / (1.0 + math.Exp(-x))
	})
}

// Erf returns the Gauss error function of each element.
func (e *Engine) Erf(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Erf)
}

// --- Rounding ---

// Round rounds each element to the nearest integer.
func (e *Engine) Round(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Round)
}

// Floor returns the greatest integer ≤ each element.
func (e *Engine) Floor(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Floor)
}

// Ceil returns the smallest integer ≥ each element.
func (e *Engine) Ceil(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.scalarUnaryFloat64(col, math.Ceil)
}

// --- Bitwise (int64 only, direct inline loops) ---

// requireInt64Pair extracts two equal-length int64 columns.
func requireInt64Pair(a, b dataset.AnyColumn) (*int64Column, *int64Column, error) {
	ca, ok := a.(*int64Column)
	if !ok {
		return nil, nil, fmt.Errorf("memory: bitwise op requires int64 column, got %T: %w", a, ErrUnsupportedType)
	}

	cb, ok := b.(*int64Column)
	if !ok {
		return nil, nil, fmt.Errorf("memory: bitwise op requires int64 column, got %T: %w", b, ErrUnsupportedType)
	}

	if len(ca.data) != len(cb.data) {
		return nil, nil, fmt.Errorf("memory: column length mismatch: %d vs %d: %w", len(ca.data), len(cb.data), ErrLengthMismatch)
	}

	return ca, cb, nil
}

// BitAnd returns element-wise bitwise AND of two int64 columns.
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

// BitOr returns element-wise bitwise OR of two int64 columns.
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

// BitXor returns element-wise bitwise XOR of two int64 columns.
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

// BitNot returns the bitwise complement of each int64 element.
func (e *Engine) BitNot(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	c, ok := col.(*int64Column)
	if !ok {
		return nil, fmt.Errorf("BitNot: got %T: %w", col, ErrRequiresInt64)
	}

	out := make([]int64, len(c.data))
	for i, v := range c.data {
		out[i] = ^v
	}

	return &int64Column{name: c.name, data: out}, nil
}

// BitShiftLeft shifts each int64 element left by n bits.
func (e *Engine) BitShiftLeft(col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	c, ok := col.(*int64Column)
	if !ok {
		return nil, fmt.Errorf("BitShiftLeft: got %T: %w", col, ErrRequiresInt64)
	}

	out := make([]int64, len(c.data))
	for i, v := range c.data {
		out[i] = v << uint(n)
	}

	return &int64Column{name: c.name, data: out}, nil
}

// BitShiftRight shifts each int64 element right by n bits.
func (e *Engine) BitShiftRight(col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	c, ok := col.(*int64Column)
	if !ok {
		return nil, fmt.Errorf("BitShiftRight: got %T: %w", col, ErrRequiresInt64)
	}

	out := make([]int64, len(c.data))
	for i, v := range c.data {
		out[i] = v >> uint(n)
	}

	return &int64Column{name: c.name, data: out}, nil
}
