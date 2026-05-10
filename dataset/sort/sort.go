// Package sort provides SIMD-accelerated sorting for the dataset engines.
//
// Sort dispatches to the best available algorithm per type:
//   - RadixSort for integers and floats (O(n), SIMD-accelerated)
//   - VQSort as comparison-based fallback (vectorized quicksort)
//   - Sorting networks for very small arrays (≤32 elements)
//
// NthElement provides O(n) partial sorting for computing quantiles
// (Median, percentiles) without a full sort.
//
// Architecture dispatch is automatic:
//
//	AMD64: AVX-512 → AVX2 → scalar
//	ARM64: NEON → scalar
//	Other: pure Go fallback
package sort

import (
	"cmp"
	"slices"

	"github.com/ajroetker/go-highway/hwy"
	hwysort "github.com/ajroetker/go-highway/hwy/contrib/sort"
)

// Sort sorts data in-place using the best algorithm for the type.
// Uses SIMD RadixSort (O(n)) for integers and floats.
func Sort[T hwy.Lanes](data []T) { hwysort.Sort(data) }

// VQSort sorts data in-place using vectorized quicksort.
// O(n log n) comparison-based with SIMD-accelerated partitioning.
func VQSort[T hwy.Lanes](data []T) { hwysort.VQSort(data) }

// NthElement rearranges data such that data[k] is the element that would
// be at position k if data were fully sorted. Elements before k are ≤ data[k],
// elements after are ≥ data[k]. Average O(n).
//
// This is the core primitive for Median and quantile computation:
//
//	NthElement(data, n/2)  // data[n/2] is now the median
func NthElement[T hwy.Lanes](data []T, k int) { hwysort.NthElement(data, k) }

// Small sorts data of length ≤32 using a SIMD sorting network.
// For larger data, use Sort instead.
func Small[T hwy.Lanes](data []T) { hwysort.SortSmall(data) }

// IndicesFloat64 returns the indices that would sort the data.
// Uses Go's stdlib pdqsort (slices.SortFunc) on (value, index) pairs.
func IndicesFloat64(data []float64) []int {
	return sortIndices(data, cmp.Compare)
}

// IndicesInt64 returns the indices that would sort the data.
func IndicesInt64(data []int64) []int {
	return sortIndices(data, cmp.Compare)
}

// IndicesString returns the indices that would sort the data.
func IndicesString(data []string) []int {
	return sortIndices(data, cmp.Compare)
}

// sortIndices is the generic indirect sort using stdlib pdqsort.
func sortIndices[T any](data []T, cmpFn func(a, b T) int) []int {
	n := len(data)
	if n == 0 {
		return nil
	}

	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	slices.SortFunc(indices, func(a, b int) int {
		return cmpFn(data[a], data[b])
	})

	return indices
}
