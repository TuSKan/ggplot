// Package math provides SIMD-accelerated mathematical transforms for the
// dataset engines.
//
// Slice transforms apply a transcendental function element-wise to an entire
// slice using SIMD vectorization (AVX-512, AVX2, NEON, or scalar fallback).
//
// Vector-level functions operate on individual SIMD register vectors for use
// in custom compute kernels.
//
// All functions are generic over hwy.Floats (float32, float64, Float16, BFloat16)
// and provide ~4 ULP accuracy.
package math

import (
	"github.com/ajroetker/go-highway/hwy"
	"github.com/ajroetker/go-highway/hwy/contrib/algo"
)

// --- Slice Transforms ---
// Apply a mathematical function element-wise to an entire slice.
// dst and src must have the same length. dst may alias src for in-place.

// Exp applies e^x element-wise: dst[i] = exp(src[i]).
func Exp[T hwy.Floats](dst, src []T) { algo.ExpTransform(src, dst) }

// Log applies ln(x) element-wise: dst[i] = log(src[i]).
func Log[T hwy.Floats](dst, src []T) { algo.LogTransform(src, dst) }

// Sin applies sin(x) element-wise: dst[i] = sin(src[i]).
func Sin[T hwy.Floats](dst, src []T) { algo.SinTransform(src, dst) }

// Cos applies cos(x) element-wise: dst[i] = cos(src[i]).
func Cos[T hwy.Floats](dst, src []T) { algo.CosTransform(src, dst) }

// Tanh applies tanh(x) element-wise: dst[i] = tanh(src[i]).
func Tanh[T hwy.Floats](dst, src []T) { algo.TanhTransform(src, dst) }

// Sigmoid applies the logistic function element-wise: dst[i] = 1/(1+e^(-src[i])).
func Sigmoid[T hwy.Floats](dst, src []T) { algo.SigmoidTransform(src, dst) }

// Erf applies the error function element-wise: dst[i] = erf(src[i]).
func Erf[T hwy.Floats](dst, src []T) { algo.ErfTransform(src, dst) }
