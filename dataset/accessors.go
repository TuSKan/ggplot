package dataset

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/TuSKan/ggplot/dataset/compute"
)

type Float64Opt = func([]float64) []float64
type Int64Opt = func([]int64) []int64
type StringOpt = func([]string) []string

// Float64 returns the float64 values of the named column, optionally
// transformed by a chain of Float64Opts. With no opts, the returned slice
// aliases the underlying column data (zero-copy). Any opt forces a copy
// before the chain runs, so callers may freely mutate the result.
func (d Dataset) Float64(name string, opts ...Float64Opt) ([]float64, error) {
	if d.tbl == nil {
		return nil, fmt.Errorf("dataset: Float64() on uncollected Dataset — call Collect(ctx) first")
	}
	if d.err != nil {
		return nil, d.err
	}
	col, err := GetColumn[float64](d.tbl, name)
	if err != nil {
		return nil, err
	}
	vals := col.Values()
	if len(opts) > 0 {
		vals = slices.Clone(vals) // copy once, opts mutate freely
		for _, opt := range opts {
			vals = opt(vals)
		}
	}
	return vals, nil
}

func (d Dataset) Int64(name string, opts ...Int64Opt) ([]int64, error) {
	if d.tbl == nil {
		return nil, fmt.Errorf("dataset: Int64() on uncollected Dataset — call Collect(ctx) first")
	}
	if d.err != nil {
		return nil, d.err
	}
	col, err := GetColumn[int64](d.tbl, name)
	if err != nil {
		return nil, err
	}
	vals := col.Values()
	if len(opts) > 0 {
		vals = slices.Clone(vals) // copy once, opts mutate freely
		for _, opt := range opts {
			vals = opt(vals)
		}
	}
	return vals, nil
}

func (d Dataset) Strings(name string, opts ...StringOpt) ([]string, error) {
	if d.tbl == nil {
		return nil, fmt.Errorf("dataset: Strings() on uncollected Dataset — call Collect(ctx) first")
	}
	if d.err != nil {
		return nil, d.err
	}
	col, err := GetColumn[string](d.tbl, name)
	if err != nil {
		return nil, err
	}
	vals := col.Values()
	if len(opts) > 0 {
		vals = slices.Clone(vals) // copy once, opts mutate freely
		for _, opt := range opts {
			vals = opt(vals)
		}
	}
	return vals, nil
}

func (d Dataset) Bools(name string) ([]bool, error) {
	if d.tbl == nil {
		return nil, fmt.Errorf("dataset: Bools() on uncollected Dataset — call Collect(ctx) first")
	}
	if d.err != nil {
		return nil, d.err
	}
	col, err := GetColumn[bool](d.tbl, name)
	if err != nil {
		return nil, err
	}
	return col.Values(), nil
}

// Sorted sorts the slice in place. Generic over any ordered type.
func Sorted[T cmp.Ordered](s []T) []T {
	slices.Sort(s)
	return s
}

// Clamp returns an option that clamps slice elements to the range [min, max].
// Example: Clamp[int64](-5, 5), Clamp(0.0, 1.0)
func Clamp[T cmp.Ordered](min, max T) func([]T) []T {
	return func(s []T) []T {
		// Fast paths using SIMD compute for primitive types
		switch v := any(s).(type) {
		case []float64:
			minF, maxF := any(min).(float64), any(max).(float64)
			n := len(v)
			lanes := compute.NumLanes[float64]()
			minVec := compute.Set(minF)
			maxVec := compute.Set(maxF)
			i := 0
			for ; i <= n-lanes; i += lanes {
				vec := compute.Load(v[i:])
				vec = compute.Max(vec, minVec)
				vec = compute.Min(vec, maxVec)
				compute.Store(vec, v[i:])
			}
			if i < n {
				vec := compute.LoadSlice(v[i:])
				vec = compute.Max(vec, minVec)
				vec = compute.Min(vec, maxVec)
				compute.StoreSlice(vec, v[i:])
			}
			return s
		case []int64:
			minI, maxI := any(min).(int64), any(max).(int64)
			n := len(v)
			lanes := compute.NumLanes[int64]()
			minVec := compute.Set(minI)
			maxVec := compute.Set(maxI)
			i := 0
			for ; i <= n-lanes; i += lanes {
				vec := compute.Load(v[i:])
				vec = compute.Max(vec, minVec)
				vec = compute.Min(vec, maxVec)
				compute.Store(vec, v[i:])
			}
			if i < n {
				vec := compute.LoadSlice(v[i:])
				vec = compute.Max(vec, minVec)
				vec = compute.Min(vec, maxVec)
				compute.StoreSlice(vec, v[i:])
			}
			return s
		default:
			// Fallback scalar loop for non-accelerated types
			for i, val := range s {
				if val < min {
					s[i] = min
				} else if val > max {
					s[i] = max
				}
			}
			return s
		}
	}
}

// Clean drops NaN and ±Inf from a float64 slice.
func Clean(s []float64) []float64 {
	n := len(s)
	lanes := compute.NumLanes[float64]()
	outLen := 0
	i := 0
	for ; i <= n-lanes; i += lanes {
		v := compute.Load(s[i:])
		mask := compute.IsFinite(v)
		outLen += compute.CompressStore(v, mask, s[outLen:])
	}
	if i < n {
		for _, v := range s[i:] {
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				s[outLen] = v
				outLen++
			}
		}
	}
	return s[:outLen]
}

// Abs applies math.Abs to all elements in place.
func Abs(s []float64) []float64 {
	n := len(s)
	lanes := compute.NumLanes[float64]()
	i := 0
	for ; i <= n-lanes; i += lanes {
		v := compute.Load(s[i:])
		v = compute.Abs(v)
		compute.Store(v, s[i:])
	}
	if i < n {
		v := compute.LoadSlice(s[i:])
		v = compute.Abs(v)
		compute.StoreSlice(v, s[i:])
	}
	return s
}

// FillNaN replaces all NaNs with the provided value.
func FillNaN(fill float64) Float64Opt {
	return func(s []float64) []float64 {
		n := len(s)
		lanes := compute.NumLanes[float64]()
		fillVec := compute.Set(fill)
		i := 0
		for ; i <= n-lanes; i += lanes {
			val := compute.Load(s[i:])
			mask := compute.IsNaN(val)
			res := compute.IfThenElse(mask, fillVec, val)
			compute.Store(res, s[i:])
		}
		if i < n {
			val := compute.LoadSlice(s[i:])
			mask := compute.IsNaN(val)
			res := compute.IfThenElse(mask, fillVec, val)
			compute.StoreSlice(res, s[i:])
		}
		return s
	}
}
