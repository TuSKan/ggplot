package compute

import (
	"math"
)

// --- Slice-level element-wise ops ---
//
// Plain scalar loops — these are the fastest path for Go's current compiler.
//
// [golang/go#65592]: go-highway's Vec[T] heap-escapes through generic function calls (even with
// GOEXPERIMENT=simd) because the Go compiler cannot prove 64-byte Vec values
// stay on the stack through generic Load/Add/Store chains. This makes the
// hwy.Load-based SIMD path ~8x slower than direct loops due to per-vector
// heap allocations.
//
// For true SIMD vectorisation, use dmath.Exp/Log/Sin/Cos/Tanh/Sigmoid/Erf
// which manage their own internal vectorisation via Slice* transforms.
//
// Caller must ensure len(dst) == len(a) == len(b) for binary ops,
// len(dst) == len(src) for unary/scalar ops.

// --- Binary ---

// SliceAdd computes dst[i] = a[i] + b[i].
func SliceAdd[T Lanes](dst, a, b []T) {
	for i := range a {
		dst[i] = a[i] + b[i]
	}
}

// SliceSub computes dst[i] = a[i] - b[i].
func SliceSub[T Lanes](dst, a, b []T) {
	for i := range a {
		dst[i] = a[i] - b[i]
	}
}

// SliceMul computes dst[i] = a[i] * b[i].
func SliceMul[T Lanes](dst, a, b []T) {
	for i := range a {
		dst[i] = a[i] * b[i]
	}
}

// SliceDiv computes dst[i] = a[i] / b[i] (floats only).
func SliceDiv[T Floats](dst, a, b []T) {
	for i := range a {
		dst[i] = a[i] / b[i]
	}
}

// --- Scalar broadcast ---

// SliceAddScalar computes dst[i] = src[i] + val.
func SliceAddScalar[T Lanes](dst, src []T, val T) {
	for i, v := range src {
		dst[i] = v + val
	}
}

// SliceMulScalar computes dst[i] = src[i] * val.
func SliceMulScalar[T Lanes](dst, src []T, val T) {
	for i, v := range src {
		dst[i] = v * val
	}
}

// --- Unary ---

// SliceAbs computes dst[i] = |src[i]|.
func SliceAbs[T Lanes](dst, src []T) {
	for i, v := range src {
		if v < 0 {
			v = -v
		}

		dst[i] = v
	}
}

// SliceNeg computes dst[i] = -src[i].
func SliceNeg[T Lanes](dst, src []T) {
	for i, v := range src {
		dst[i] = -v
	}
}

// SliceSqrt computes dst[i] = √src[i] (float64 only).
func SliceSqrt(dst, src []float64) {
	for i, v := range src {
		dst[i] = math.Sqrt(v)
	}
}

// SlicePow computes dst[i] = src[i]^exp (float64 only).
func SlicePow(dst, src []float64, exp float64) {
	for i, v := range src {
		dst[i] = math.Pow(v, exp)
	}
}
