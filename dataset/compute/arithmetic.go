package compute

import "github.com/ajroetker/go-highway/hwy"

// --- Arithmetic operations ---
// Element-wise SIMD arithmetic. Maps to single hardware instructions
// (VADDPD, VMULPD, etc.) on supported architectures.

// Add returns element-wise a + b.
func Add[T Lanes](a, b Vec[T]) Vec[T] { return hwy.Add(a, b) }

// Sub returns element-wise a - b.
func Sub[T Lanes](a, b Vec[T]) Vec[T] { return hwy.Sub(a, b) }

// Mul returns element-wise a * b.
func Mul[T Lanes](a, b Vec[T]) Vec[T] { return hwy.Mul(a, b) }

// Div returns element-wise a / b (floats only).
func Div[T Floats](a, b Vec[T]) Vec[T] { return hwy.Div(a, b) }

// Neg returns element-wise -v.
func Neg[T Lanes](v Vec[T]) Vec[T] { return hwy.Neg(v) }

// Abs returns element-wise |v|.
func Abs[T Lanes](v Vec[T]) Vec[T] { return hwy.Abs(v) }

// Min returns element-wise min(a, b).
func Min[T Lanes](a, b Vec[T]) Vec[T] { return hwy.Min(a, b) }

// Max returns element-wise max(a, b).
func Max[T Lanes](a, b Vec[T]) Vec[T] { return hwy.Max(a, b) }

// FMA returns fused multiply-add: a*b + c with single rounding (floats only).
func FMA[T Floats](a, b, c Vec[T]) Vec[T] { return hwy.FMA(a, b, c) }

// MulAdd returns a*b + c (floats only). Alias for FMA.
func MulAdd[T Floats](a, b, c Vec[T]) Vec[T] { return hwy.MulAdd(a, b, c) }
