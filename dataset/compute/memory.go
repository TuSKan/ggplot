package compute

import "github.com/ajroetker/go-highway/hwy"

// --- Load/Store operations ---
// These map directly to SIMD load/store instructions.
// Load/Store skip bounds checking for inner loops (caller guarantees length).
// LoadSlice/StoreSlice are safe variants that handle short slices.

// Load loads a vector from data (no bounds check — caller must guarantee len ≥ NumLanes).
func Load[T Lanes](data []T) Vec[T] { return hwy.Load(data) }

// LoadSlice safely loads a vector from data, handling short slices.
func LoadSlice[T Lanes](data []T) Vec[T] { return hwy.LoadSlice(data) }

// Store writes a vector to data (no bounds check).
func Store[T Lanes](v Vec[T], data []T) { hwy.Store(v, data) }

// StoreSlice safely writes a vector to data, handling short slices.
func StoreSlice[T Lanes](v Vec[T], data []T) { hwy.StoreSlice(v, data) }

// Set broadcasts a scalar value to all lanes of a vector.
func Set[T Lanes](val T) Vec[T] { return hwy.Set[T](val) }

// Zero returns a vector with all lanes set to zero.
func Zero[T Lanes]() Vec[T] { return hwy.Zero[T]() }

// MaskLoad loads a vector from data, zeroing lanes where mask is false.
func MaskLoad[T Lanes](mask Mask[T], data []T) Vec[T] { return hwy.MaskLoad(mask, data) }

// MaskStore writes only the lanes where mask is true.
func MaskStore[T Lanes](mask Mask[T], v Vec[T], data []T) { hwy.MaskStore(mask, v, data) }

// Load4 loads four interleaved vectors from data (AoS → SoA deinterleave).
func Load4[T Lanes](data []T) (a, b, c, d Vec[T]) { return hwy.Load4[T](data) }
