// Package compute provides portable SIMD primitives for the dataset engines.
//
// This package is a thin, zero-overhead wrapper around go-highway (hwy).
// It re-exports the complete hwy core API so that engine code never imports
// hwy directly — allowing the SIMD backend to be swapped transparently.
//
// Architecture dispatch is automatic at runtime:
//
//	AMD64: AVX-512 → AVX2 → scalar
//	ARM64: SVE → NEON → scalar
//	Other: pure Go scalar fallback
//
// All functions are generic over [hwy.Lanes] (float32, float64, int32, int64).
package compute

import "github.com/ajroetker/go-highway/hwy"

// Vec is a re-export of hwy.Vec so callers never import hwy directly.
type Vec[T hwy.Lanes] = hwy.Vec[T]

// Mask is a re-export of hwy.Mask.
type Mask[T hwy.Lanes] = hwy.Mask[T]

// Lanes is the constraint for types supported by SIMD vectors.
type Lanes = hwy.Lanes

// Floats is the constraint for floating-point SIMD types.
type Floats = hwy.Floats
