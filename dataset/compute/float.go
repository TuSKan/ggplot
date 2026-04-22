package compute

import "github.com/ajroetker/go-highway/hwy"

// --- Float checking operations ---
// SIMD floating-point classification and rounding (floats only).

// IsNaN returns a mask where v[i] is NaN.
func IsNaN[T Floats](v Vec[T]) Mask[T] { return hwy.IsNaN(v) }

// IsInf returns a mask where v[i] is ±Inf.
// sign: 0 = either, >0 = +Inf only, <0 = -Inf only.
func IsInf[T Floats](v Vec[T], sign int) Mask[T] { return hwy.IsInf(v, sign) }

// IsFinite returns a mask where v[i] is finite (not NaN and not ±Inf).
func IsFinite[T Floats](v Vec[T]) Mask[T] { return hwy.IsFinite(v) }

// RoundToEven returns element-wise banker's rounding (round half to even).
func RoundToEven[T Floats](v Vec[T]) Vec[T] { return hwy.RoundToEven(v) }
