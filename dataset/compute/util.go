package compute

import "github.com/ajroetker/go-highway/hwy"

// --- Utility operations ---
// SIMD lane information and constant constructors.

// NumLanes returns the number of lanes in a SIMD vector for type T
// on the current hardware (e.g. 8 for float64 on AVX-512).
func NumLanes[T Lanes]() int { return hwy.NumLanes[T]() }

// SignBit returns a vector with the sign bit set in every lane.
// For floats: -0.0. For signed ints: minimum value. For unsigned: high bit.
func SignBit[T Lanes]() Vec[T] { return hwy.SignBit[T]() }

// Const returns a vector with all lanes set to the given float32 constant.
// Convenient for float literals: compute.Const[float64](0.5).
func Const[T Lanes](val float32) Vec[T] { return hwy.Const[T](val) }

// ConstValue converts a float32 constant to type T.
func ConstValue[T Lanes](val float32) T { return hwy.ConstValue[T](val) }
