package sort_test

import (
	"math"
	"slices"
	"testing"

	dsort "github.com/TuSKan/ggplot/dataset/sort"
)

func TestSort_Float64(t *testing.T) {
	data := []float64{5, 3, 8, 1, 9, 2, 7, 4, 6}
	dsort.Sort(data)
	if !slices.IsSorted(data) {
		t.Errorf("Sort did not produce sorted output: %v", data)
	}
}

func TestSort_Int64(t *testing.T) {
	data := []int64{10, -5, 3, 100, -200, 42}
	dsort.Sort(data)
	if !slices.IsSorted(data) {
		t.Errorf("Sort did not produce sorted output: %v", data)
	}
}

func TestNthElement_Median_Odd(t *testing.T) {
	data := []float64{9, 1, 5, 3, 7}
	n := len(data)
	mid := n / 2
	dsort.NthElement(data, mid)
	// data[mid] should be the median = 5
	if data[mid] != 5 {
		t.Errorf("NthElement median = %v, want 5", data[mid])
	}
	// All elements before mid should be <= data[mid]
	for i := 0; i < mid; i++ {
		if data[i] > data[mid] {
			t.Errorf("data[%d]=%v > data[%d]=%v", i, data[i], mid, data[mid])
		}
	}
	// All elements after mid should be >= data[mid]
	for i := mid + 1; i < n; i++ {
		if data[i] < data[mid] {
			t.Errorf("data[%d]=%v < data[%d]=%v", i, data[i], mid, data[mid])
		}
	}
}

func TestNthElement_Median_Even(t *testing.T) {
	data := []float64{8, 2, 6, 4}
	n := len(data)
	mid := n / 2
	dsort.NthElement(data, mid)
	upper := data[mid]
	dsort.NthElement(data[:mid], mid-1)
	median := (data[mid-1] + upper) / 2
	if median != 5 {
		t.Errorf("Median = %v, want 5", median)
	}
}

func TestSortIndicesFloat64(t *testing.T) {
	data := []float64{3, 1, 4, 1, 5}
	idx := dsort.SortIndicesFloat64(data)
	// Expected order: indices for sorted [1, 1, 3, 4, 5]
	want := []float64{1, 1, 3, 4, 5}
	for i, j := range idx {
		if data[j] != want[i] {
			t.Errorf("sorted[%d] = data[%d] = %v, want %v", i, j, data[j], want[i])
		}
	}
}

func TestSortIndicesInt64(t *testing.T) {
	data := []int64{30, 10, 40, 20}
	idx := dsort.SortIndicesInt64(data)
	want := []int64{10, 20, 30, 40}
	for i, j := range idx {
		if data[j] != want[i] {
			t.Errorf("sorted[%d] = data[%d] = %v, want %v", i, j, data[j], want[i])
		}
	}
}

func TestSortIndicesString(t *testing.T) {
	data := []string{"banana", "apple", "cherry"}
	idx := dsort.SortIndicesString(data)
	want := []string{"apple", "banana", "cherry"}
	for i, j := range idx {
		if data[j] != want[i] {
			t.Errorf("sorted[%d] = data[%d] = %q, want %q", i, j, data[j], want[i])
		}
	}
}

func TestNthElement_Large(t *testing.T) {
	n := 10000
	data := make([]float64, n)
	for i := range data {
		data[i] = float64(n - i) // descending
	}
	mid := n / 2
	dsort.NthElement(data, mid)
	// Median should be (n/2 + 1) for 1-indexed = 5001 for 10000 elements
	want := float64(mid + 1)
	if math.Abs(data[mid]-want) > 0.5 {
		t.Errorf("NthElement(10000, %d) = %v, want ~%v", mid, data[mid], want)
	}
}

// --- Benchmarks ---

func BenchmarkSort_Float64_1M(b *testing.B) {
	src := make([]float64, 1_000_000)
	for i := range src {
		src[i] = float64(1_000_000 - i)
	}
	data := make([]float64, len(src))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(data, src)
		dsort.Sort(data)
	}
}

func BenchmarkNthElement_Float64_1M(b *testing.B) {
	src := make([]float64, 1_000_000)
	for i := range src {
		src[i] = float64(1_000_000 - i)
	}
	data := make([]float64, len(src))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(data, src)
		dsort.NthElement(data, len(data)/2)
	}
}

func BenchmarkSortIndicesFloat64_1M(b *testing.B) {
	data := make([]float64, 1_000_000)
	for i := range data {
		data[i] = float64(1_000_000 - i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dsort.SortIndicesFloat64(data)
	}
}
