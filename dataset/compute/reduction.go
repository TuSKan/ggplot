package compute

import "github.com/ajroetker/go-highway/hwy"

// --- Reduction operations ---
// Horizontal reductions across SIMD vector lanes.
// These reduce a Vec[T] to a scalar T.

// ReduceSum computes the sum of all lanes in the vector.
func ReduceSum[T Lanes](v Vec[T]) T { return hwy.ReduceSum(v) }

// ReduceMin computes the minimum lane value in the vector.
func ReduceMin[T Lanes](v Vec[T]) T { return hwy.ReduceMin(v) }

// ReduceMax computes the maximum lane value in the vector.
func ReduceMax[T Lanes](v Vec[T]) T { return hwy.ReduceMax(v) }

// --- Slice-level reductions ---
// Pure Go scalar implementations — zero allocations.
// When GOEXPERIMENT=simd is enabled, the compiler auto-vectorizes these loops.

// SliceSum computes the sum of all elements in a slice.
func SliceSum[T Lanes](data []T) T {
	var total T
	for _, v := range data {
		total += v
	}
	return total
}

// SliceMin computes the minimum element in a slice.
func SliceMin[T Lanes](data []T) T {
	if len(data) == 0 {
		var zero T
		return zero
	}
	result := data[0]
	for _, v := range data[1:] {
		if v < result {
			result = v
		}
	}
	return result
}

// SliceMax computes the maximum element in a slice.
func SliceMax[T Lanes](data []T) T {
	if len(data) == 0 {
		var zero T
		return zero
	}
	result := data[0]
	for _, v := range data[1:] {
		if v > result {
			result = v
		}
	}
	return result
}

// SliceMinMax computes both min and max of a slice in a single pass.
func SliceMinMax[T Lanes](data []T) (min, max T) {
	if len(data) == 0 {
		return
	}
	min, max = data[0], data[0]
	for _, v := range data[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return
}
