package memory

import (
	"runtime"
	"slices"
	"sync"
)

// parallelSortThreshold is the minimum slice length before we spawn goroutines.
// Below this, pdqsort (slices.SortFunc) is already fast enough and goroutine
// overhead would dominate.
const parallelSortThreshold = 1 << 16 // 64K

// parallelSortFunc sorts a slice using a parallel merge-sort strategy.
// For small slices (< 64K), it falls back to slices.SortFunc (pdqsort).
// For large slices, it splits the work across GOMAXPROCS goroutines,
// sorts each chunk independently, then merges. This dramatically reduces
// wall-clock time for 100M+ element sorts.
func parallelSortFunc[T any](data []T, cmpFn func(a, b T) int) {
	if len(data) < parallelSortThreshold {
		slices.SortFunc(data, cmpFn)
		return
	}

	pSort(data, cmpFn, maxDepth())
}

// maxDepth returns the recursion depth limit for parallel splitting.
// Each level doubles the goroutine count, so depth = log2(GOMAXPROCS).
func maxDepth() int {
	procs := runtime.GOMAXPROCS(0)

	depth := 0
	for p := procs; p > 1; p >>= 1 {
		depth++
	}

	return depth
}

// pSort recursively splits work across goroutines up to maxDepth,
// then falls back to sequential pdqsort for each chunk and merges results.
func pSort[T any](data []T, cmpFn func(a, b T) int, depth int) {
	n := len(data)
	if depth <= 0 || n < parallelSortThreshold {
		slices.SortFunc(data, cmpFn)
		return
	}

	mid := n / 2

	var wg sync.WaitGroup

	wg.Go(func() {
		pSort(data[:mid], cmpFn, depth-1)
	})

	pSort(data[mid:], cmpFn, depth-1)
	wg.Wait()

	// In-place merge of two sorted halves
	pMerge(data, mid, cmpFn)
}

// pMerge merges two sorted halves data[:mid] and data[mid:] in-place.
// Uses a temporary buffer for the left half to achieve O(n) merge.
func pMerge[T any](data []T, mid int, cmpFn func(a, b T) int) {
	left := make([]T, mid)
	copy(left, data[:mid])

	i, j, k := 0, mid, 0
	for i < len(left) && j < len(data) {
		if cmpFn(left[i], data[j]) <= 0 {
			data[k] = left[i]
			i++
		} else {
			data[k] = data[j]
			j++
		}

		k++
	}

	for i < len(left) {
		data[k] = left[i]
		i++
		k++
	}
}
