package memory_test

import (
	"context"
	"math"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	memeng "github.com/TuSKan/ggplot/dataset/memory"
)

func mathEngine() *memeng.Engine { return memeng.NewEngine(context.Background()) }

func mathCol(eng *memeng.Engine, vals []float64) dataset.AnyColumn {
	return eng.NewFloat64Column("x", vals)
}

func intCol(eng *memeng.Engine, vals []int64) dataset.AnyColumn {
	return eng.NewInt64Column("x", vals)
}

func assertClose(t *testing.T, name string, got, want float64) {
	t.Helper()

	if math.Abs(got-want) > 1e-6 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func getF64(col dataset.AnyColumn) float64 {
	return col.(interface{ Values() []float64 }).Values()[0] //nolint:errcheck,forcetypeassert // type guaranteed by test setup.
}

func getI64(col dataset.AnyColumn) int64 {
	return col.(interface{ Values() []int64 }).Values()[0] //nolint:errcheck,forcetypeassert // type guaranteed by test setup.
}

// --- Arithmetic ---

func TestMemAddCols(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	a := mathCol(eng, []float64{1, 2, 3})
	b := mathCol(eng, []float64{10, 20, 30})

	r, err := eng.AddCols(a, b)
	if err != nil {
		t.Fatal(err)
	}

	vals := r.(interface{ Values() []float64 }).Values() //nolint:errcheck,forcetypeassert // type guaranteed by test setup.
	if vals[0] != 11 || vals[1] != 22 || vals[2] != 33 {
		t.Errorf("AddCols = %v", vals)
	}
}

func TestMemMulScalar(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{2, 4, 6})
	r, _ := eng.MulScalar(col, 3)

	vals := r.(interface{ Values() []float64 }).Values() //nolint:errcheck,forcetypeassert // type guaranteed by test setup.
	if vals[0] != 6 || vals[1] != 12 || vals[2] != 18 {
		t.Errorf("MulScalar = %v", vals)
	}
}

// --- Unary ---

func TestMemAbs(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	r, _ := eng.Abs(mathCol(eng, []float64{-5}))
	assertClose(t, "Abs(-5)", getF64(r), 5)
}

func TestMemSqrt(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	r, _ := eng.Sqrt(mathCol(eng, []float64{25}))
	assertClose(t, "Sqrt(25)", getF64(r), 5)
}

func TestMemPow(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	r, _ := eng.Pow(mathCol(eng, []float64{3}), 4)
	assertClose(t, "Pow(3,4)", getF64(r), 81)
}

// --- Transcendental ---

func TestMemSin(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	r, _ := eng.Sin(mathCol(eng, []float64{math.Pi / 2}))
	assertClose(t, "Sin(π/2)", getF64(r), 1)
}

func TestMemExp(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	r, _ := eng.Exp(mathCol(eng, []float64{1}))
	assertClose(t, "Exp(1)", getF64(r), math.E)
}

func TestMemLn(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	r, _ := eng.Ln(mathCol(eng, []float64{math.E}))
	assertClose(t, "Ln(e)", getF64(r), 1)
}

func TestMemTanh(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	r, _ := eng.Tanh(mathCol(eng, []float64{0}))
	assertClose(t, "Tanh(0)", getF64(r), 0)
}

func TestMemSigmoid(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	r, _ := eng.Sigmoid(mathCol(eng, []float64{0}))
	assertClose(t, "Sigmoid(0)", getF64(r), 0.5)
}

// --- Rounding ---

func TestMemFloor(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	r, _ := eng.Floor(mathCol(eng, []float64{2.7}))
	assertClose(t, "Floor(2.7)", getF64(r), 2)
}

func TestMemCeil(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	r, _ := eng.Ceil(mathCol(eng, []float64{2.1}))
	assertClose(t, "Ceil(2.1)", getF64(r), 3)
}

// --- Bitwise ---

func TestMemBitAnd(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	a := intCol(eng, []int64{0xFF})
	b := intCol(eng, []int64{0x0F})

	r, _ := eng.BitAnd(a, b)
	if getI64(r) != 0x0F {
		t.Errorf("BitAnd = %X, want 0F", getI64(r))
	}
}

func TestMemBitShiftLeft(t *testing.T) {
	t.Parallel()

	eng := mathEngine()

	r, _ := eng.BitShiftLeft(intCol(eng, []int64{1}), 8)
	if getI64(r) != 256 {
		t.Errorf("ShiftLeft(1,8) = %d, want 256", getI64(r))
	}
}

// --- Interface check ---

var _ dataset.MathKernel = (*memeng.Engine)(nil)
