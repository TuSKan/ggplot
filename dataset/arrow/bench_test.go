package arrow_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	arroweng "github.com/TuSKan/ggplot/dataset/arrow"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func makeBenchDS(b *testing.B, n int) dataset.Table {
	b.Helper()
	eng := arroweng.NewEngine(context.Background(), memory.DefaultAllocator)

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

// makeBenchDSWithNulls creates a dataset where ~10% of float values are null.
func makeBenchDSWithNulls(b *testing.B, n int) (dataset.Table, dataset.AnyColumn) {
	b.Helper()
	eng := arroweng.NewEngine(context.Background(), memory.DefaultAllocator)

	schema := dataset.NewSchema(dataset.FloatCol("x"))
	builder := eng.NewBuilder(schema)
	xApp := builder.Float64("x")
	xApp.Reserve(n)
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < n; i++ {
		if rng.Float64() < 0.1 {
			xApp.AppendNull()
		} else {
			xApp.Append(rng.Float64() * 1000)
		}
	}
	ds, err := builder.Build()
	if err != nil {
		b.Fatal(err)
	}
	col, _ := ds.Column("x")
	return ds, col
}

// --- Aggregator Benchmarks ---

func BenchmarkArrowSum(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.Sum(col)
			}
		})
	}
}

func BenchmarkArrowMean(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.Mean(col)
			}
		})
	}
}

func BenchmarkArrowMinMax(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _, _ = eng.MinMax(col)
			}
		})
	}
}

func BenchmarkArrowMedian(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.Median(col)
			}
		})
	}
}

func BenchmarkArrowVariance(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.Variance(col)
			}
		})
	}
}

// --- Selector Benchmarks ---

func BenchmarkArrowSlice(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.Slice(col, 0, n/2)
			}
		})
	}
}

func BenchmarkArrowTake(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")

			rng := rand.New(rand.NewSource(99))
			half := n / 2
			indices := make([]int, half)
			for i := range indices {
				indices[i] = rng.Intn(n)
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.Select(col, indices)
			}
		})
	}
}

func BenchmarkArrowSortIndices(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.SortIndices(col)
			}
		})
	}
}

func BenchmarkArrowFilter(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			pred := dataset.Gt("x", 500.0)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = dataset.From(ds).Filter(pred).Collect(context.Background())
			}
		})
	}
}

// --- Filler Benchmarks ---

func BenchmarkArrowFillDown(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			_, col := makeBenchDSWithNulls(b, n)
			eng := arroweng.NewEngine(context.Background(), memory.DefaultAllocator)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.Fill(col, dataset.FillDown)
			}
		})
	}
}

func BenchmarkArrowFillUp(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			_, col := makeBenchDSWithNulls(b, n)
			eng := arroweng.NewEngine(context.Background(), memory.DefaultAllocator)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.Fill(col, dataset.FillUp)
			}
		})
	}
}

func BenchmarkArrowReplaceNA(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			_, col := makeBenchDSWithNulls(b, n)
			eng := arroweng.NewEngine(context.Background(), memory.DefaultAllocator)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.ReplaceNA(col, 0)
			}
		})
	}
}

// --- Windower Benchmarks ---

func BenchmarkArrowLag(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.Lag(col, 1)
			}
		})
	}
}

func BenchmarkArrowLead(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.Lead(col, 1)
			}
		})
	}
}

func BenchmarkArrowCumSum(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.CumSum(col)
			}
		})
	}
}

func BenchmarkArrowCumMax(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.CumMax(col)
			}
		})
	}
}

func BenchmarkArrowCumMin(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.CumMin(col)
			}
		})
	}
}

func BenchmarkArrowRank(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.Rank(col)
			}
		})
	}
}

func BenchmarkArrowDenseRank(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.DenseRank(col)
			}
		})
	}
}

// --- Frame Verb Benchmarks ---

func BenchmarkArrowFrameHead(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = dataset.From(ds).Head(100).Collect(context.Background())
			}
		})
	}
}

func BenchmarkArrowGroupBySummarize(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
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

func benchMathUnary(b *testing.B, name string, fn func(*arroweng.Engine, dataset.AnyColumn) (dataset.AnyColumn, error)) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = fn(eng, col)
			}
		})
	}
}

func BenchmarkArrowExp(b *testing.B) {
	benchMathUnary(b, "Exp", func(e *arroweng.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) {
		return e.Exp(c)
	})
}

func BenchmarkArrowLn(b *testing.B) {
	benchMathUnary(b, "Ln", func(e *arroweng.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) {
		return e.Ln(c)
	})
}

func BenchmarkArrowSin(b *testing.B) {
	benchMathUnary(b, "Sin", func(e *arroweng.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) {
		return e.Sin(c)
	})
}

func BenchmarkArrowSqrt(b *testing.B) {
	benchMathUnary(b, "Sqrt", func(e *arroweng.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) {
		return e.Sqrt(c)
	})
}

func BenchmarkArrowAbs(b *testing.B) {
	benchMathUnary(b, "Abs", func(e *arroweng.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) {
		return e.Abs(c)
	})
}

func BenchmarkArrowMathFloor(b *testing.B) {
	benchMathUnary(b, "Floor", func(e *arroweng.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) {
		return e.Floor(c)
	})
}

func BenchmarkArrowSigmoid(b *testing.B) {
	benchMathUnary(b, "Sigmoid", func(e *arroweng.Engine, c dataset.AnyColumn) (dataset.AnyColumn, error) {
		return e.Sigmoid(c)
	})
}

func BenchmarkArrowAddCols(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.AddCols(col, col)
			}
		})
	}
}

func BenchmarkArrowMulScalar(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("x")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.MulScalar(col, 3.14)
			}
		})
	}
}

func BenchmarkArrowBitShiftLeft(b *testing.B) {
	for _, n := range []int{1_000, 100_000, 1_000_000, 10_000_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ds := makeBenchDS(b, n)
			eng := dataset.GetEngine(ds).(*arroweng.Engine)
			col, _ := ds.Column("id")
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = eng.BitShiftLeft(col, 4)
			}
		})
	}
}
