package arrow_test

import (
	"context"
	"math"
	"testing"

	"github.com/apache/arrow-go/v18/arrow/memory"

	"github.com/TuSKan/ggplot/dataset"
	arroweng "github.com/TuSKan/ggplot/dataset/arrow"
)

func mathEngine() *arroweng.Engine {
	return arroweng.NewEngine(context.Background(), memory.DefaultAllocator)
}

func mathCol(eng *arroweng.Engine, vals []float64) dataset.AnyColumn {
	return eng.NewFloat64Column("x", vals)
}

func intCol(eng *arroweng.Engine, vals []int64) dataset.AnyColumn {
	return eng.NewInt64Column("x", vals)
}

func assertClose(t *testing.T, name string, got, want float64) {
	t.Helper()

	if math.Abs(got-want) > 1e-6 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func getF64(col dataset.AnyColumn) float64 {
	type valuer interface{ Values() []float64 }
	return col.(valuer).Values()[0] //nolint:errcheck,forcetypeassert // type guaranteed by test setup.
}

func getI64(col dataset.AnyColumn) int64 {
	type valuer interface{ Values() []int64 }
	return col.(valuer).Values()[0] //nolint:errcheck,forcetypeassert // type guaranteed by test setup.
}

// --- Arithmetic ---

func TestArrowAddCols(t *testing.T) {
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

func TestArrowMulScalar(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{2, 4, 6})

	r, err := eng.MulScalar(col, 3)
	if err != nil {
		t.Fatal(err)
	}

	vals := r.(interface{ Values() []float64 }).Values() //nolint:errcheck,forcetypeassert // type guaranteed by test setup.
	if vals[0] != 6 || vals[1] != 12 || vals[2] != 18 {
		t.Errorf("MulScalar = %v", vals)
	}
}

// --- Unary ---

func TestArrowAbs(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{-5})
	r, _ := eng.Abs(col)
	assertClose(t, "Abs(-5)", getF64(r), 5)
}

func TestArrowNeg(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{7})
	r, _ := eng.Neg(col)
	assertClose(t, "Neg(7)", getF64(r), -7)
}

func TestArrowSqrt(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{25})
	r, _ := eng.Sqrt(col)
	assertClose(t, "Sqrt(25)", getF64(r), 5)
}

func TestArrowPow(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{3})
	r, _ := eng.Pow(col, 4)
	assertClose(t, "Pow(3,4)", getF64(r), 81)
}

// --- Transcendental ---

func TestArrowSin(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{math.Pi / 2})
	r, _ := eng.Sin(col)
	assertClose(t, "Sin(π/2)", getF64(r), 1)
}

func TestArrowCos(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{0})
	r, _ := eng.Cos(col)
	assertClose(t, "Cos(0)", getF64(r), 1)
}

func TestArrowLn(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{math.E})
	r, _ := eng.Ln(col)
	assertClose(t, "Ln(e)", getF64(r), 1)
}

func TestArrowExp(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{1})
	r, _ := eng.Exp(col)
	assertClose(t, "Exp(1)", getF64(r), math.E)
}

func TestArrowTanh(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{0})
	r, _ := eng.Tanh(col)
	assertClose(t, "Tanh(0)", getF64(r), 0)
}

func TestArrowSigmoid(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{0})
	r, _ := eng.Sigmoid(col)
	assertClose(t, "Sigmoid(0)", getF64(r), 0.5)
}

// --- Rounding ---

func TestArrowRound(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{3.7})
	r, _ := eng.Round(col)
	assertClose(t, "Round(3.7)", getF64(r), 4)
}

func TestArrowFloor(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{2.7})
	r, _ := eng.Floor(col)
	assertClose(t, "Floor(2.7)", getF64(r), 2)
}

func TestArrowCeil(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := mathCol(eng, []float64{2.1})
	r, _ := eng.Ceil(col)
	assertClose(t, "Ceil(2.1)", getF64(r), 3)
}

// --- Bitwise ---

func TestArrowBitAnd(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	a := intCol(eng, []int64{0xFF})
	b := intCol(eng, []int64{0x0F})

	r, _ := eng.BitAnd(a, b)
	if getI64(r) != 0x0F {
		t.Errorf("BitAnd = %X, want 0F", getI64(r))
	}
}

func TestArrowBitShiftLeft(t *testing.T) {
	t.Parallel()

	eng := mathEngine()
	col := intCol(eng, []int64{1})

	r, _ := eng.BitShiftLeft(col, 8)
	if getI64(r) != 256 {
		t.Errorf("ShiftLeft(1,8) = %d, want 256", getI64(r))
	}
}

// --- Interface check ---

var _ dataset.MathKernel = (*arroweng.Engine)(nil)
