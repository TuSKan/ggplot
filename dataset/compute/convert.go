package compute

import "github.com/ajroetker/go-highway/hwy"

// --- Type cast operations ---
// Reinterpret or convert between numeric types in SIMD registers.

// AsInt32 reinterprets a float32 vector as int32.
func AsInt32(v Vec[float32]) Vec[int32] { return hwy.AsInt32(v) }

// AsFloat32 reinterprets an int32 vector as float32.
func AsFloat32(v Vec[int32]) Vec[float32] { return hwy.AsFloat32(v) }

// AsInt64 reinterprets a float64 vector as int64.
func AsInt64(v Vec[float64]) Vec[int64] { return hwy.AsInt64(v) }

// AsFloat64 reinterprets an int64 vector as float64.
func AsFloat64(v Vec[int64]) Vec[float64] { return hwy.AsFloat64(v) }
