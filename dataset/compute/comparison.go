package compute

import "github.com/ajroetker/go-highway/hwy"

// --- Comparison operations ---
// Element-wise SIMD comparisons returning masks.

// Equal returns a mask where a[i] == b[i].
func Equal[T Lanes](a, b Vec[T]) Mask[T] { return hwy.Equal(a, b) }

// NotEqual returns a mask where a[i] != b[i].
func NotEqual[T Lanes](a, b Vec[T]) Mask[T] { return hwy.NotEqual(a, b) }

// LessThan returns a mask where a[i] < b[i].
func LessThan[T Lanes](a, b Vec[T]) Mask[T] { return hwy.LessThan(a, b) }

// LessEqual returns a mask where a[i] <= b[i].
func LessEqual[T Lanes](a, b Vec[T]) Mask[T] { return hwy.LessEqual(a, b) }

// GreaterThan returns a mask where a[i] > b[i].
func GreaterThan[T Lanes](a, b Vec[T]) Mask[T] { return hwy.GreaterThan(a, b) }

// GreaterEqual returns a mask where a[i] >= b[i].
func GreaterEqual[T Lanes](a, b Vec[T]) Mask[T] { return hwy.GreaterEqual(a, b) }
