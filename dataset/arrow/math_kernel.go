package arrow

import (
	"context"
	"fmt"
	"math"

	"github.com/TuSKan/ggplot/dataset"
	dmath "github.com/TuSKan/ggplot/dataset/math"

	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/compute"
)

// --- MathKernel: Arrow compute first, highway fills gaps ---

// applyUnaryCompute applies an Arrow compute unary function to a float64 column.
func (e *Engine) applyUnaryCompute(col dataset.AnyColumn, fn func(ctx context.Context, opts compute.ArithmeticOptions, arg compute.Datum) (compute.Datum, error)) (dataset.AnyColumn, error) {
	c, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: MathKernel requires float64 column, got %T", col)
	}
	ctx := compute.WithAllocator(e.Context(), e.alloc)
	result, err := fn(ctx, compute.ArithmeticOptions{}, compute.NewDatum(c.arr))
	if err != nil {
		return nil, err
	}
	defer result.Release()
	return &arrowFloat64Column{name: c.name, arr: result.(*compute.ArrayDatum).MakeArray().(*array.Float64)}, nil
}

// applyUnaryComputeNoOpts applies an Arrow compute unary function (no opts variant).
func (e *Engine) applyUnaryComputeNoOpts(col dataset.AnyColumn, fn func(ctx context.Context, arg compute.Datum) (compute.Datum, error)) (dataset.AnyColumn, error) {
	c, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: MathKernel requires float64 column, got %T", col)
	}
	ctx := compute.WithAllocator(e.Context(), e.alloc)
	result, err := fn(ctx, compute.NewDatum(c.arr))
	if err != nil {
		return nil, err
	}
	defer result.Release()
	return &arrowFloat64Column{name: c.name, arr: result.(*compute.ArrayDatum).MakeArray().(*array.Float64)}, nil
}

// applyBinaryCompute applies an Arrow compute binary function to two float64 columns.
func (e *Engine) applyBinaryCompute(a, b dataset.AnyColumn, fn func(ctx context.Context, opts compute.ArithmeticOptions, left, right compute.Datum) (compute.Datum, error)) (dataset.AnyColumn, error) {
	ca, ok := a.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: MathKernel requires float64 column, got %T", a)
	}
	cb, ok := b.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: MathKernel requires float64 column, got %T", b)
	}
	if ca.arr.Len() != cb.arr.Len() {
		return nil, fmt.Errorf("arrow: column length mismatch: %d vs %d", ca.arr.Len(), cb.arr.Len())
	}
	ctx := compute.WithAllocator(e.Context(), e.alloc)
	result, err := fn(ctx, compute.ArithmeticOptions{}, compute.NewDatum(ca.arr), compute.NewDatum(cb.arr))
	if err != nil {
		return nil, err
	}
	defer result.Release()
	return &arrowFloat64Column{name: ca.name, arr: result.(*compute.ArrayDatum).MakeArray().(*array.Float64)}, nil
}

// applySliceTransform applies a highway (dst, src) transform, returning a new column.
// Writes directly into an Arrow builder to avoid an intermediate Go-slice copy.
func (e *Engine) applySliceTransform(col dataset.AnyColumn, fn func(dst, src []float64)) (dataset.AnyColumn, error) {
	c, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: MathKernel requires float64 column, got %T", col)
	}
	src := c.arr.Float64Values()
	n := len(src)

	// Allocate the output buffer via Arrow builder (single allocation).
	b := array.NewFloat64Builder(e.alloc)
	b.Resize(n)
	// Grow the builder's data buffer to n elements so we can write directly.
	b.AppendValues(make([]float64, n), nil)
	arr := b.NewFloat64Array()
	b.Release()

	// The Arrow buffer is mutable right after NewFloat64Array (refcount=1).
	// Get a writable view of the underlying buffer.
	dst := arr.Float64Values()
	fn(dst, src)

	return &arrowFloat64Column{name: c.name, arr: arr}, nil
}

// applyScalarFunc applies a scalar math.X function element-wise.
func (e *Engine) applyScalarFunc(col dataset.AnyColumn, fn func(float64) float64) (dataset.AnyColumn, error) {
	c, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: MathKernel requires float64 column, got %T", col)
	}
	vals := c.arr.Float64Values()
	out := make([]float64, len(vals))
	for i, v := range vals {
		out[i] = fn(v)
	}
	return e.NewFloat64Column(c.name, out), nil
}

// --- Binary arithmetic (Arrow native) ---

func (e *Engine) AddCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyBinaryCompute(a, b, compute.Add)
}

func (e *Engine) SubCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyBinaryCompute(a, b, compute.Subtract)
}

func (e *Engine) MulCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyBinaryCompute(a, b, compute.Multiply)
}

func (e *Engine) DivCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyBinaryCompute(a, b, compute.Divide)
}

// --- Scalar arithmetic ---

func (e *Engine) AddScalar(col dataset.AnyColumn, val float64) (dataset.AnyColumn, error) {
	c, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: AddScalar requires float64 column, got %T", col)
	}
	ctx := compute.WithAllocator(e.Context(), e.alloc)
	result, err := compute.Add(ctx, compute.ArithmeticOptions{}, compute.NewDatum(c.arr), compute.NewDatum(val))
	if err != nil {
		return nil, err
	}
	defer result.Release()
	return &arrowFloat64Column{name: c.name, arr: result.(*compute.ArrayDatum).MakeArray().(*array.Float64)}, nil
}

func (e *Engine) MulScalar(col dataset.AnyColumn, val float64) (dataset.AnyColumn, error) {
	c, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: MulScalar requires float64 column, got %T", col)
	}
	ctx := compute.WithAllocator(e.Context(), e.alloc)
	result, err := compute.Multiply(ctx, compute.ArithmeticOptions{}, compute.NewDatum(c.arr), compute.NewDatum(val))
	if err != nil {
		return nil, err
	}
	defer result.Release()
	return &arrowFloat64Column{name: c.name, arr: result.(*compute.ArrayDatum).MakeArray().(*array.Float64)}, nil
}

// --- Unary numeric ---

func (e *Engine) Abs(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryCompute(col, compute.AbsoluteValue)
}

func (e *Engine) Neg(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryCompute(col, compute.Negate)
}

func (e *Engine) Sign(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryComputeNoOpts(col, compute.Sign)
}

// Sqrt — Arrow lacks this, use highway/stdlib
func (e *Engine) Sqrt(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyScalarFunc(col, math.Sqrt)
}

func (e *Engine) Pow(col dataset.AnyColumn, exp float64) (dataset.AnyColumn, error) {
	c, ok := col.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: Pow requires float64 column, got %T", col)
	}
	ctx := compute.WithAllocator(e.Context(), e.alloc)
	result, err := compute.Power(ctx, compute.ArithmeticOptions{}, compute.NewDatum(c.arr), compute.NewDatum(exp))
	if err != nil {
		return nil, err
	}
	defer result.Release()
	return &arrowFloat64Column{name: c.name, arr: result.(*compute.ArrayDatum).MakeArray().(*array.Float64)}, nil
}

// --- Transcendental: logarithmic ---

// Exp — Arrow lacks this, use highway
func (e *Engine) Exp(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applySliceTransform(col, dmath.Exp[float64])
}

// Ln — Arrow native
func (e *Engine) Ln(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryCompute(col, compute.Ln)
}

// Log2 — Arrow native
func (e *Engine) Log2(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryCompute(col, compute.Log2)
}

// Log10 — Arrow native
func (e *Engine) Log10(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryCompute(col, compute.Log10)
}

// --- Transcendental: trigonometric (Arrow native) ---

func (e *Engine) Sin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryCompute(col, compute.Sin)
}

func (e *Engine) Cos(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryCompute(col, compute.Cos)
}

func (e *Engine) Tan(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryCompute(col, compute.Tan)
}

func (e *Engine) Asin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryCompute(col, compute.Asin)
}

func (e *Engine) Acos(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryCompute(col, compute.Acos)
}

func (e *Engine) Atan(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyUnaryComputeNoOpts(col, compute.Atan)
}

func (e *Engine) Atan2(y, x dataset.AnyColumn) (dataset.AnyColumn, error) {
	cy, ok := y.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: Atan2 requires float64 columns, got %T", y)
	}
	cx, ok := x.(*arrowFloat64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: Atan2 requires float64 columns, got %T", x)
	}
	ctx := compute.WithAllocator(e.Context(), e.alloc)
	result, err := compute.Atan2(ctx, compute.NewDatum(cy.arr), compute.NewDatum(cx.arr))
	if err != nil {
		return nil, err
	}
	defer result.Release()
	return &arrowFloat64Column{name: cy.name, arr: result.(*compute.ArrayDatum).MakeArray().(*array.Float64)}, nil
}

// --- Transcendental: hyperbolic/special (highway — Arrow lacks these) ---

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

// Round — use stdlib for predictable half-away-from-zero behavior
func (e *Engine) Round(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyScalarFunc(col, math.Round)
}

// Floor — Arrow lacks this, use stdlib
func (e *Engine) Floor(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyScalarFunc(col, math.Floor)
}

// Ceil — Arrow lacks this, use stdlib
func (e *Engine) Ceil(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.applyScalarFunc(col, math.Ceil)
}

// --- Bitwise (int64 columns, Arrow native ShiftLeft/ShiftRight) ---

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
	c, ok := col.(*arrowInt64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: BitNot requires int64 column, got %T", col)
	}
	vals := c.arr.Int64Values()
	out := make([]int64, len(vals))
	for i, v := range vals {
		out[i] = ^v
	}
	return e.NewInt64Column(c.name, out), nil
}

func (e *Engine) BitShiftLeft(col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	c, ok := col.(*arrowInt64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: BitShiftLeft requires int64 column, got %T", col)
	}
	vals := c.arr.Int64Values()
	out := make([]int64, len(vals))
	for i, v := range vals {
		out[i] = v << uint(n)
	}
	return e.NewInt64Column(c.name, out), nil
}

func (e *Engine) BitShiftRight(col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	c, ok := col.(*arrowInt64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: BitShiftRight requires int64 column, got %T", col)
	}
	vals := c.arr.Int64Values()
	out := make([]int64, len(vals))
	for i, v := range vals {
		out[i] = v >> uint(n)
	}
	return e.NewInt64Column(c.name, out), nil
}

// applyBinaryInt64 applies a binary int64 function element-wise.
func (e *Engine) applyBinaryInt64(a, b dataset.AnyColumn, fn func(int64, int64) int64) (dataset.AnyColumn, error) {
	ca, ok := a.(*arrowInt64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: bitwise op requires int64 column, got %T", a)
	}
	cb, ok := b.(*arrowInt64Column)
	if !ok {
		return nil, fmt.Errorf("arrow: bitwise op requires int64 column, got %T", b)
	}
	va, vb := ca.arr.Int64Values(), cb.arr.Int64Values()
	if len(va) != len(vb) {
		return nil, fmt.Errorf("arrow: column length mismatch: %d vs %d", len(va), len(vb))
	}
	out := make([]int64, len(va))
	for i := range va {
		out[i] = fn(va[i], vb[i])
	}
	return e.NewInt64Column(ca.name, out), nil
}
