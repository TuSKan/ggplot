package compute

import "github.com/ajroetker/go-highway/hwy"

// --- Shuffle operations ---
// SIMD lane manipulation and permutation.

// GetLane extracts the value from the specified lane of a vector.
func GetLane[T Lanes](v Vec[T], lane int) T { return hwy.GetLane(v, lane) }

// Reverse reverses the order of all lanes in a vector.
func Reverse[T Lanes](v Vec[T]) Vec[T] { return hwy.Reverse(v) }

// Reverse2 reverses pairs of adjacent lanes.
func Reverse2[T Lanes](v Vec[T]) Vec[T] { return hwy.Reverse2(v) }

// Reverse4 reverses groups of 4 adjacent lanes.
func Reverse4[T Lanes](v Vec[T]) Vec[T] { return hwy.Reverse4(v) }

// Reverse8 reverses groups of 8 adjacent lanes.
func Reverse8[T Lanes](v Vec[T]) Vec[T] { return hwy.Reverse8(v) }

// Broadcast copies lane n to all other lanes.
func Broadcast[T Lanes](v Vec[T], lane int) Vec[T] { return hwy.Broadcast(v, lane) }

// Iota returns a vector with lanes set to [0, 1, 2, ...].
func Iota[T Lanes]() Vec[T] { return hwy.Iota[T]() }
