package compute

import "github.com/ajroetker/go-highway/hwy"

// --- Math operations ---
// SIMD math functions on vectors (floats only). Maps to hardware
// instructions where available (VSQRTPD on AVX, FSQRT on NEON).

// Sqrt returns element-wise √v.
func Sqrt[T Floats](v Vec[T]) Vec[T] { return hwy.Sqrt(v) }

// RSqrt returns element-wise approximate 1/√v.
func RSqrt[T Floats](v Vec[T]) Vec[T] { return hwy.RSqrt(v) }

// RSqrtNewtonRaphson returns element-wise 1/√v refined with Newton-Raphson.
func RSqrtNewtonRaphson[T Floats](v Vec[T]) Vec[T] { return hwy.RSqrtNewtonRaphson(v) }

// RSqrtPrecise returns element-wise 1/√v with full precision.
func RSqrtPrecise[T Floats](v Vec[T]) Vec[T] { return hwy.RSqrtPrecise(v) }

// Pow returns element-wise base^exp.
func Pow[T Floats](base, exp Vec[T]) Vec[T] { return hwy.Pow(base, exp) }
