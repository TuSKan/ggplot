package compute_test

import (
	"math"
	"testing"

	"github.com/TuSKan/ggplot/dataset/compute"
)

// --- Reduction tests ---

func TestSliceSum_Float64(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	got := compute.SliceSum(data)
	want := 55.0
	if got != want {
		t.Errorf("SliceSum(1..10) = %v, want %v", got, want)
	}
}

func TestSliceSum_Int64(t *testing.T) {
	data := []int64{1, 2, 3, 4, 5}
	got := compute.SliceSum(data)
	want := int64(15)
	if got != want {
		t.Errorf("SliceSum(1..5) = %v, want %v", got, want)
	}
}

func TestSliceSum_Empty(t *testing.T) {
	got := compute.SliceSum([]float64{})
	if got != 0 {
		t.Errorf("SliceSum(empty) = %v, want 0", got)
	}
}

func TestSliceMin_Float64(t *testing.T) {
	data := []float64{5, 3, 8, 1, 9, 2}
	got := compute.SliceMin(data)
	if got != 1 {
		t.Errorf("SliceMin = %v, want 1", got)
	}
}

func TestSliceMax_Float64(t *testing.T) {
	data := []float64{5, 3, 8, 1, 9, 2}
	got := compute.SliceMax(data)
	if got != 9 {
		t.Errorf("SliceMax = %v, want 9", got)
	}
}

func TestSliceMinMax_Float64(t *testing.T) {
	data := []float64{5, 3, 8, 1, 9, 2}
	lo, hi := compute.SliceMinMax(data)
	if lo != 1 || hi != 9 {
		t.Errorf("SliceMinMax = (%v, %v), want (1, 9)", lo, hi)
	}
}

func TestSliceMinMax_Single(t *testing.T) {
	data := []float64{42}
	lo, hi := compute.SliceMinMax(data)
	if lo != 42 || hi != 42 {
		t.Errorf("SliceMinMax([42]) = (%v, %v), want (42, 42)", lo, hi)
	}
}

func TestSliceSum_LargeSlice(t *testing.T) {
	// Test with a slice larger than typical SIMD width
	n := 1000
	data := make([]float64, n)
	want := 0.0
	for i := range data {
		data[i] = float64(i + 1)
		want += data[i]
	}
	got := compute.SliceSum(data)
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("SliceSum(1..%d) = %v, want %v", n, got, want)
	}
}

func TestSliceMinMax_Int64(t *testing.T) {
	data := []int64{10, -5, 3, 100, -200, 42}
	lo, hi := compute.SliceMinMax(data)
	if lo != -200 || hi != 100 {
		t.Errorf("SliceMinMax = (%v, %v), want (-200, 100)", lo, hi)
	}
}

// --- Vec-level operations ---

func TestLoad_Store_Roundtrip(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	lanes := compute.NumLanes[float64]()
	if lanes == 0 || lanes > len(data) {
		t.Skip("not enough data for vector width")
	}
	v := compute.Load(data)
	out := make([]float64, lanes)
	compute.Store(v, out)
	for i := 0; i < lanes; i++ {
		if out[i] != data[i] {
			t.Errorf("roundtrip[%d] = %v, want %v", i, out[i], data[i])
		}
	}
}

func TestAdd_Vec(t *testing.T) {
	a := compute.Set[float64](3)
	b := compute.Set[float64](5)
	c := compute.Add(a, b)
	got := compute.GetLane(c, 0)
	if got != 8 {
		t.Errorf("Add(3, 5)[0] = %v, want 8", got)
	}
}

func TestMul_Vec(t *testing.T) {
	a := compute.Set[float64](4)
	b := compute.Set[float64](7)
	c := compute.Mul(a, b)
	got := compute.GetLane(c, 0)
	if got != 28 {
		t.Errorf("Mul(4, 7)[0] = %v, want 28", got)
	}
}

func TestSqrt_Vec(t *testing.T) {
	v := compute.Set[float64](25)
	r := compute.Sqrt(v)
	got := compute.GetLane(r, 0)
	if got != 5 {
		t.Errorf("Sqrt(25)[0] = %v, want 5", got)
	}
}

func TestNumLanes(t *testing.T) {
	n := compute.NumLanes[float64]()
	if n < 1 {
		t.Errorf("NumLanes[float64]() = %d, want >= 1", n)
	}
	t.Logf("float64 SIMD lanes: %d", n)
}

// --- Benchmarks ---

func BenchmarkSliceSum_Float64_1M(b *testing.B) {
	data := make([]float64, 1_000_000)
	for i := range data {
		data[i] = float64(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compute.SliceSum(data)
	}
}

func BenchmarkSliceMinMax_Float64_1M(b *testing.B) {
	data := make([]float64, 1_000_000)
	for i := range data {
		data[i] = float64(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compute.SliceMinMax(data)
	}
}
