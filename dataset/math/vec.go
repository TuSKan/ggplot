package math

import (
	"github.com/ajroetker/go-highway/hwy"
	hwmath "github.com/ajroetker/go-highway/hwy/contrib/math"
)

// --- Vector-Level Math ---
// These operate on individual SIMD register vectors for use in custom
// compute kernels. All functions provide ~4 ULP accuracy.

// Vec is the SIMD vector type re-exported for convenience.
type Vec[T hwy.Lanes] = hwy.Vec[T]

// ExpVec computes e^v element-wise on a SIMD vector.
func ExpVec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseExpVec(v) }

// Exp2Vec computes 2^v element-wise on a SIMD vector.
func Exp2Vec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseExp2Vec(v) }

// LogVec computes ln(v) element-wise on a SIMD vector.
func LogVec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseLogVec(v) }

// Log2Vec computes log₂(v) element-wise on a SIMD vector.
func Log2Vec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseLog2Vec(v) }

// Log10Vec computes log₁₀(v) element-wise on a SIMD vector.
func Log10Vec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseLog10Vec(v) }

// SinVec computes sin(v) element-wise on a SIMD vector.
func SinVec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseSinVec(v) }

// CosVec computes cos(v) element-wise on a SIMD vector.
func CosVec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseCosVec(v) }

// TanhVec computes tanh(v) element-wise on a SIMD vector.
func TanhVec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseTanhVec(v) }

// SinhVec computes sinh(v) element-wise on a SIMD vector.
func SinhVec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseSinhVec(v) }

// CoshVec computes cosh(v) element-wise on a SIMD vector.
func CoshVec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseCoshVec(v) }

// AsinhVec computes arcsinh(v) element-wise on a SIMD vector.
func AsinhVec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseAsinhVec(v) }

// AcoshVec computes arccosh(v) element-wise on a SIMD vector.
func AcoshVec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseAcoshVec(v) }

// AtanhVec computes arctanh(v) element-wise on a SIMD vector.
func AtanhVec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseAtanhVec(v) }

// SigmoidVec computes 1/(1+e^(-v)) element-wise on a SIMD vector.
func SigmoidVec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseSigmoidVec(v) }

// ErfVec computes erf(v) element-wise on a SIMD vector.
func ErfVec[T hwy.Floats](v Vec[T]) Vec[T] { return hwmath.BaseErfVec(v) }

// PowVec computes base^exp element-wise on SIMD vectors.
func PowVec[T hwy.Floats](base, exp Vec[T]) Vec[T] { return hwmath.BasePowVec(base, exp) }
