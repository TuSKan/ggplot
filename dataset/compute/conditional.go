package compute

import "github.com/ajroetker/go-highway/hwy"

// --- Conditional operations ---
// SIMD masked selection and clamping.

// IfThenElse returns a[i] where mask is true, b[i] otherwise.
func IfThenElse[T Lanes](mask Mask[T], a, b Vec[T]) Vec[T] { return hwy.IfThenElse(mask, a, b) }

// IfThenElseZero returns v[i] where mask is true, 0 otherwise.
func IfThenElseZero[T Lanes](mask Mask[T], v Vec[T]) Vec[T] { return hwy.IfThenElseZero(mask, v) }

// IfThenZeroElse returns 0 where mask is true, v[i] otherwise.
func IfThenZeroElse[T Lanes](mask Mask[T], v Vec[T]) Vec[T] { return hwy.IfThenZeroElse(mask, v) }

// ZeroIfNegative returns 0 where v[i] < 0, v[i] otherwise (ReLU).
func ZeroIfNegative[T Lanes](v Vec[T]) Vec[T] { return hwy.ZeroIfNegative(v) }
