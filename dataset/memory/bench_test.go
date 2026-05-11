package memory_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
)

func makeBenchDS(b *testing.B, n int) dataset.Table {
	b.Helper()

	eng := memory.NewEngine(context.Background())

	rng := rand.New(rand.NewSource(42))
	xs := make([]float64, n)
	ids := make([]int64, n)
	groups := make([]string, n)
	labels := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	for i := range xs {
		xs[i] = rng.Float64() * 1000
		ids[i] = int64(rng.Intn(1_000_000))
		groups[i] = labels[rng.Intn(len(labels))]
	}

	schema := dataset.NewSchema(
		dataset.FloatCol("x"),
		dataset.IntCol("id"),
		dataset.StringCol("group"),
	)

	ds, err := eng.FromColumns(schema,
		eng.NewFloat64Column("x", xs),
		eng.NewInt64Column("id", ids),
		eng.NewStringColumn("group", groups),
	)
	if err != nil {
		b.Fatal(err)
	}

	return ds
}

// --- Aggregator Benchmarks ---

func BenchmarkSum(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.Sum(col)
			}
		})
	}
}

func BenchmarkMean(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.Mean(col)
			}
		})
	}
}

func BenchmarkMinMax(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _, _ = eng.MinMax(col)
			}
		})
	}
}

func BenchmarkMedian(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.Median(col)
			}
		})
	}
}

func BenchmarkVariance(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.Variance(col)
			}
		})
	}
}

func BenchmarkCount(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.Count(col)
			}
		})
	}
}

// --- Selector Benchmarks ---

func BenchmarkSortIndices(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.SortIndices(col)
			}
		})
	}
}

func BenchmarkTake(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			rng := rand.New(rand.NewSource(99))
			half := n / 2

			indices := make([]int, half)
			for i := range indices {
				indices[i] = rng.Intn(n)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.Select(col, indices)
			}
		})
	}
}

func BenchmarkSlice(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.Slice(col, 0, n/2)
			}
		})
	}
}

// --- Windower Benchmarks ---

func BenchmarkLag(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.Lag(col, 1)
			}
		})
	}
}

func BenchmarkLead(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.Lead(col, 1)
			}
		})
	}
}

func BenchmarkCumSum(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.CumSum(col)
			}
		})
	}
}

func BenchmarkCumMax(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.CumMax(col)
			}
		})
	}
}

func BenchmarkCumMin(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.CumMin(col)
			}
		})
	}
}

func BenchmarkRank(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.Rank(col)
			}
		})
	}
}

func BenchmarkDenseRank(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.DenseRank(col)
			}
		})
	}
}

// --- Frame Verb Benchmarks ---

func BenchmarkFrameArrange(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = dataset.From(ds).Arrange("x").Collect(context.Background())
			}
		})
	}
}

func BenchmarkFrameHead(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = dataset.From(ds).Head(100).Collect(context.Background())
			}
		})
	}
}

func BenchmarkFrameSelect(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = dataset.From(ds).Select("x", "group").Collect(context.Background())
			}
		})
	}
}

func BenchmarkGroupBySummarize(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = dataset.From(ds).
					GroupBy("group").
					Summarize(
						dataset.Sum("total", "x"),
						dataset.Count("n", "x"),
						dataset.Mean("avg", "x"),
					).Collect(context.Background())
			}
		})
	}
}

// --- MathKernel Benchmarks ---

func benchMemMathUnary(b *testing.B, fn func(*memory.Engine, dataset.AnyColumn) (dataset.AnyColumn, error)) {
	b.Helper()

	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = fn(eng, col)
			}
		})
	}
}

func BenchmarkMemExp(b *testing.B) {
	benchMemMathUnary(b, func(e *memory.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) { return e.Exp(c) })
}

func BenchmarkMemLn(b *testing.B) {
	benchMemMathUnary(b, func(e *memory.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) { return e.Ln(c) })
}

func BenchmarkMemSin(b *testing.B) {
	benchMemMathUnary(b, func(e *memory.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) { return e.Sin(c) })
}

func BenchmarkMemSqrt(b *testing.B) {
	benchMemMathUnary(b, func(e *memory.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) { return e.Sqrt(c) })
}

func BenchmarkMemAbs(b *testing.B) {
	benchMemMathUnary(b, func(e *memory.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) { return e.Abs(c) })
}

func BenchmarkMemMathFloor(b *testing.B) {
	benchMemMathUnary(b, func(e *memory.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) { return e.Floor(c) })
}

func BenchmarkMemSigmoid(b *testing.B) {
	benchMemMathUnary(b, func(e *memory.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) { return e.Sigmoid(c) })
}

func BenchmarkMemAddCols(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.AddCols(col, col)
			}
		})
	}
}

func BenchmarkMemMulScalar(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("x")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.MulScalar(col, 3.14)
			}
		})
	}
}

func BenchmarkMemBitShiftLeft(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)

			eng, ok := dataset.GetEngine(ds).(*memory.Engine)
			if !ok {
				b.Fatalf("expected *memory.Engine, got %T", dataset.GetEngine(ds))
			}

			col, _ := ds.Column("id")

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				_, _ = eng.BitShiftLeft(col, 4)
			}
		})
	}
}
