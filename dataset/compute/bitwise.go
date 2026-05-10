package compute

import "github.com/ajroetker/go-highway/hwy"

// Integers is a re-export of hwy.Integers for use by callers.
type (
	Integers = hwy.Integers
)

// --- Bitwise operations ---
// SIMD bitwise operations. And/Or/Xor/Not/AndNot work on all Lanes.
// ShiftLeft/ShiftRight are restricted to Integers.

// And returns element-wise a & b.
func And[T Lanes](a, b Vec[T]) Vec[T] { return hwy.And(a, b) }

// Or returns element-wise a | b.
func Or[T Lanes](a, b Vec[T]) Vec[T] { return hwy.Or(a, b) }

// Xor returns element-wise a ^ b.
func Xor[T Lanes](a, b Vec[T]) Vec[T] { return hwy.Xor(a, b) }

// Not returns element-wise ^v (bitwise complement).
func Not[T Lanes](v Vec[T]) Vec[T] { return hwy.Not(v) }

// AndNot returns element-wise (^a) & b.
func AndNot[T Lanes](a, b Vec[T]) Vec[T] { return hwy.AndNot(a, b) }

// ShiftLeft returns element-wise v << bits (integers only).
func ShiftLeft[T Integers](v Vec[T], bits int) Vec[T] { return hwy.ShiftLeft(v, bits) }

// ShiftRight returns element-wise v >> bits (integers only).
// Arithmetic for signed, logical for unsigned.
func ShiftRight[T Integers](v Vec[T], bits int) Vec[T] { return hwy.ShiftRight(v, bits) }
