# Dataset Benchmark Results

> **CPU**: 11th Gen Intel Core i9-11980HK @ 2.60GHz (16 threads)  
> **OS**: Windows amd64  
> **Go**: 1.26.2  
> **SIMD**: go-highway v0.0.12 (scalar fallback mode — `GOEXPERIMENT=simd` enables AVX-512)

---

## Architecture

```
                    ┌─────────────────────────┐
                    │   dataset/engine.go     │
                    │   (MathKernel,          │
                    │    Aggregator, Selector) │
                    └────────┬────────────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
        Arrow Engine    Memory Engine    (SQL Engine)
              │              │
      ┌───────┴───────┐     │
      ▼               ▼     ▼
  Arrow compute   go-highway  go-highway
  (native kernels) (gaps)    (all ops)
```

**Policy**: Arrow official kernels first → go-highway fills gaps → stdlib fallback.

---

## SIMD Compute Primitives (`dataset/compute`)

Scalar fallback mode (without `GOEXPERIMENT=simd`):

| Operation | Size | Time | Allocs |
|---|---|---|---|
| **SliceSum** float64 | 1M | 8.5ms | 250K |
| **SliceMinMax** float64 | 1M | 13.5ms | 375K |

> With `GOEXPERIMENT=simd` and AVX-512 (8 lanes detected), expect **10–20× improvement** on these primitives.

---

## Sort Primitives (`dataset/sort`)

| Operation | Size | Time | Allocs | Notes |
|---|---|---|---|---|
| **Sort** (RadixSort) | 1M | 18.9ms | 5 | go-highway SIMD radix |
| **NthElement** (O(n) partial) | 1M | **2.3ms** | **0** | Median/quantile kernel |
| **SortIndices** (stdlib pdqsort) | 1M | 4.6ms | 1 | Indirect sort via `slices.SortFunc` |

> **NthElement** is the key win: O(n) vs O(n log n) for Median/percentile computation.

---

## Arrow Engine (`dataset/arrow`)

### Aggregator

| Operation | 1K | 100K | 1M | 10M | Allocs | Backend |
|---|---|---|---|---|---|---|
| **Sum** | 0.4µs | 6.0µs | 130µs | 3.7ms | 8 | `math.Float64.Sum` (Arrow) |
| **Mean** | 0.4µs | 6.0µs | 126µs | 3.7ms | 8 | `math.Float64.Sum` (Arrow) |
| **MinMax** | — | — | — | 10.3ms | 16 | `simd.SliceMinMax` (pure Go) |
| **Median** | 9.7µs | 1.4ms | **12ms** | **120ms** | 9 | `dsort.NthElement` (highway) |
| **Variance** | 1.5µs | 93µs | 1.0ms | 17.4ms | 8 | `math.Float64.Sum` + scalar loop |

> **Median** uses O(n) `NthElement` partial sort instead of O(n log n) full sort.  
> Previous implementation: **602ms at 1M** → now **12ms** = **50× speedup**.

### Selector

| Operation | 1K | 100K | 1M | 10M | Allocs | Backend |
|---|---|---|---|---|---|---|
| **Slice** (zero-copy) | 280ns | 330ns | 134ns | 115ns | 3 | Arrow `array.NewSlice` |
| **Take** | 14µs | 657µs | 7.1ms | — | 34 | `compute.TakeArray` (Arrow) |
| **SortIndices** | 23µs | 6.7ms | 79.7ms | — | 2 | `compute.SortIndicesArray` (Arrow) |
| **Filter** | 38µs | 2.0ms | 18.8ms | — | 118 | `compute.FilterArray` (Arrow) |

### MathKernel

| Operation | 1K | 100K | 1M | 10M | Allocs | Backend |
|---|---|---|---|---|---|---|
| **Abs** | 10µs | 256µs | 1.3ms | 10.7ms | 34 | `compute.AbsoluteValue` (Arrow) |
| **AddCols** (col+col) | 12µs | 257µs | 1.2ms | 10.6ms | 35 | `compute.Add` (Arrow) |
| **MulScalar** (col×val) | 12µs | 223µs | 1.1ms | 10.5ms | 37 | `compute.Multiply` (Arrow) |
| **Ln** | 20µs | 1.4ms | 10.7ms | 105ms | 34 | `compute.Ln` (Arrow) |
| **Sin** | 19µs | 1.9ms | 18.3ms | 180ms | 34 | `compute.Sin` (Arrow) |
| **Exp** | 22µs | 1.5ms | 5.8ms | 57ms | 9 | `dmath.Exp` (highway) |
| **Sqrt** | 18µs | 871µs | 6.2ms | 44ms | 9 | `math.Sqrt` (stdlib) |
| **Sigmoid** | 14µs | 749µs | 4.0ms | 37ms | 9 | `dmath.Sigmoid` (highway) |
| **Floor** | 17µs | 883µs | 4.5ms | 43ms | 9 | `math.Floor` (stdlib) |
| **BitShiftLeft** (int64) | 9µs | 491µs | 2.2ms | 22ms | 9 | scalar loop |

> Arrow native kernels (Abs, Add, Mul, Ln, Sin) handle null propagation and type dispatch.  
> Highway kernels (Exp, Sigmoid) read via `Float64Values()` → copy → transform.  
> Stdlib fallbacks (Sqrt, Floor) use element-wise scalar loops.

### 10M Head-to-Head — Arrow Compute vs Highway vs Stdlib

| Backend | Operation | 10M Time | Throughput |
|---|---|---|---|
| **Arrow compute** | Abs | 10.7ms | 7.5 GB/s |
| **Arrow compute** | AddCols | 10.6ms | 15 GB/s (2 inputs) |
| **Arrow compute** | MulScalar | 10.5ms | 7.6 GB/s |
| **Arrow compute** | Sin | 180ms | 0.4 GB/s |
| **Arrow compute** | Ln | 105ms | 0.8 GB/s |
| **Highway** | Exp | 57ms | 1.4 GB/s |
| **Highway** | Sigmoid | 37ms | 2.2 GB/s |
| **Stdlib** | Sqrt | 44ms | 1.8 GB/s |
| **Stdlib** | Floor | 43ms | 1.9 GB/s |
| **Scalar** | BitShiftLeft | 22ms | 3.6 GB/s |

> **Arithmetic** (Abs, Add, Mul) scales linearly at ~10ms/10M = **1µs/1K elements**.  
> **Transcendentals** (Sin, Ln) are 10–18× more expensive per element.  
> **Highway** Exp/Sigmoid outperforms Arrow Sin/Ln despite being the "fallback" — these are vectorized transforms vs Arrow's scalar kernel dispatch.

### Filler

| Operation | 1K | 100K | 1M | Allocs |
|---|---|---|---|---|
| **FillDown** | 9.4µs | 889µs | 7.2ms | 8 |
| **FillUp** | 23µs | 2.0ms | 18.0ms | 22 |
| **ReplaceNA** | 7.7µs | 758µs | 6.9ms | 8 |

### Windower

| Operation | 1K | 100K | 1M | Allocs |
|---|---|---|---|---|
| **Lag** | 6.9µs | 605µs | 5.2ms | 8 |
| **Lead** | 6.5µs | 562µs | 5.1ms | 8 |
| **CumSum** | 6.8µs | 550µs | 5.3ms | 8 |
| **CumMax** | 6.8µs | 520µs | 5.2ms | 8 |
| **CumMin** | 6.3µs | 518µs | 5.1ms | 8 |
| **Rank** | 22µs | 7.4ms | 87ms | 11 |
| **DenseRank** | 23µs | 7.5ms | 88ms | 11 |

### Frame Verbs

| Operation | 1K | 100K | 1M | 10M | Allocs |
|---|---|---|---|---|---|
| **Head(100)** | 728ns | 667ns | 570ns | 563ns | 13 |

---

## Memory Engine (`dataset/memory`)

### Aggregator

| Operation | 1K | 100K | 1M | 10M | Allocs |
|---|---|---|---|---|---|
| **Sum** | 869ns | 87µs | 899µs | 9.5ms | 2 |
| **Mean** | — | — | — | 9.7ms | 2 |
| **MinMax** | 448ns | 68µs | 706µs | 13.4ms | 4 |
| **Median** | — | — | — | 119ms | 3 |
| **Variance** | — | — | — | 32.2ms | 2 |
| **Count** | 34ns | 36ns | 31ns | 29ns | 1 |

### Selector

| Operation | 1K | 100K | 1M | 10M | Allocs |
|---|---|---|---|---|---|
| **Slice** (sub-slice) | 26ns | 27ns | 23ns | 22ns | 1 |
| **Take** (scatter-gather) | 1.0µs | 97µs | 1.5ms | — | 1 |
| **SortIndices** (parallel) | 49µs | 11.8ms | 162ms | — | varies |

### Frame Verbs

| Operation | 1K | 100K | 1M | 10M |
|---|---|---|---|---|
| **Head(100)** | 312ns | 347ns | 275ns | 266ns |
| **Select** | 495ns | 539ns | 373ns | 362ns |
| **Arrange** | 58µs | 12.4ms | 171ms | — |
| **GroupBy+Summarize** | 56µs | 5.9ms | 48.8ms | — |

---

## Arrow vs Memory — 10M Head-to-Head

### Aggregator (10M rows)

| Operation | Arrow | Memory | Winner | Speedup |
|---|---|---|---|---|
| **Sum** | **3.7ms** | 9.5ms | Arrow | **2.6×** |
| **Mean** | **3.7ms** | 9.7ms | Arrow | **2.6×** |
| **MinMax** | **10.3ms** | 13.4ms | Arrow | **1.3×** |
| **Median** | **120ms** | 119ms | ~same | ~1.0× |
| **Variance** | **17.4ms** | 32.2ms | Arrow | **1.8×** |
| **Count** | ~30ns | **29ns** | ~same | — |
| **Slice** | 115ns | **22ns** | Memory | **5.2×** |

### MathKernel (10M rows)

| Operation | Arrow | Memory | Winner | Speedup |
|---|---|---|---|---|
| **Abs** | **10.7ms** | 29.7ms | Arrow | **2.8×** |
| **AddCols** | **10.6ms** | 21.3ms | Arrow | **2.0×** |
| **MulScalar** | **10.5ms** | 18.0ms | Arrow | **1.7×** |
| **Ln** | 105ms | **14.7ms** | Memory | **7.1×** |
| **Sin** | 180ms | **136ms** | Memory | **1.3×** |
| **Exp** | 57ms | **48ms** | Memory | **1.2×** |
| **Sqrt** | **44ms** | 55ms | Arrow | **1.2×** |
| **Sigmoid** | 37ms | **28ms** | Memory | **1.3×** |
| **Floor** | 43ms | **37ms** | Memory | **1.2×** |
| **BitShiftLeft** | 22ms | **11ms** | Memory | **2.0×** |

> **Arrow wins** aggregations: Sum/Mean 2.6×, Variance 1.8×, MinMax 1.3× — Arrow `math.Float64.Sum` is vectorized.  
> **Arrow wins** arithmetic MathKernel (Abs 2.8×, Add 2.0×, Mul 1.7×) — native Datum dispatch.  
> **Memory wins** transcendentals (Ln 7.1×) — direct `[]float64` avoids Datum overhead.  
> **Median** is ~tied — both use single NthElement + O(n/2) max-scan of left partition.

### Allocation Profile (10M rows)

| Operation | Arrow Allocs | Arrow Bytes | Memory Allocs | Memory Bytes |
|---|---|---|---|---|
| **Sum** | 8 | 824 B | 2 | 56 B |
| **Mean** | 8 | 824 B | 2 | 56 B |
| **MinMax** | 16 | 1.6 KB | 4 | 112 B |
| **Variance** | 8 | 824 B | 2 | 56 B |
| **Median** | 9 | 80 MB | 3 | 80 MB |

> Both engines now achieve **constant, near-zero allocations** for all aggregations except Median.  
> Median allocates 80 MB (copy of 10M float64s) — unavoidable since NthElement is destructive.

---

## Source Mapping

Every engine operation maps to a specific backend:

### Arrow Engine — What Uses What

| Category | Arrow Official | Highway (gap fill) | Stdlib Fallback |
|---|---|---|---|
| **Sum/Mean** | `math.Float64.Sum` | — | — |
| **MinMax** | — | `simd.SliceMinMax` | — |
| **Median** | — | `dsort.NthElement` | — |
| **Variance** | `math.Float64.Sum` | — | scalar loop |
| **SortIndices** | `compute.SortIndicesArray` | — | — |
| **Filter/Take** | `compute.FilterArray/TakeArray` | — | — |
| **Add/Sub/Mul/Div** | `compute.Add/Sub/Mul/Div` | — | — |
| **Abs/Neg/Sign** | `compute.AbsoluteValue/Negate/Sign` | — | — |
| **Pow** | `compute.Power` | — | — |
| **Ln/Log2/Log10** | `compute.Ln/Log2/Log10` | — | — |
| **Sin/Cos/Tan** | `compute.Sin/Cos/Tan` | — | — |
| **Asin/Acos/Atan** | `compute.Asin/Acos/Atan` | — | — |
| **Atan2** | `compute.Atan2` | — | — |
| **Exp** | — | `dmath.Exp` | — |
| **Tanh** | — | `dmath.Tanh` | — |
| **Sigmoid** | — | `dmath.Sigmoid` | — |
| **Erf** | — | `dmath.Erf` | — |
| **Sqrt** | — | — | `math.Sqrt` |
| **Round/Floor/Ceil** | — | — | `math.Round/Floor/Ceil` |
| **Bitwise** | — | — | scalar `&\|^~<<>>` |

### Memory Engine

All operations use highway SIMD transforms where available, stdlib `math.X` for the rest.
No Arrow dependency.

---

## SIMD Dispatch

```
go-highway v0.0.12 runtime dispatch:

AMD64:  AVX-512 (8 lanes) → AVX2 (4 lanes) → scalar
ARM64:  NEON (2 lanes) → scalar
Other:  pure Go scalar fallback

Build:
  Default:             scalar fallback (works everywhere)
  GOEXPERIMENT=simd:   native SIMD via simd/archsimd (Go 1.26+)
```

---

## How to Run

```bash
# All tests
go test ./dataset/... -v

# Sort primitives
go test ./dataset/sort/ -bench=. -benchmem -run=^$

# Arrow engine (aggregators + MathKernel)
go test ./dataset/arrow/ -bench="BenchmarkArrow(Sum|Mean|MinMax|Median|Variance|Exp|Ln|Sin|Sqrt|Abs|Floor|Sigmoid|AddCols|MulScalar|BitShift)" -benchmem -run=^$

# Arrow engine (all except GroupBy)
go test ./dataset/arrow/ -bench="BenchmarkArrow[^G]" -benchmem -run=^$

# Memory engine
go test ./dataset/memory/ -bench=. -benchmem -run=^$

# With SIMD enabled (AMD64)
GOEXPERIMENT=simd go test ./dataset/... -bench=. -benchmem -run=^$
```
