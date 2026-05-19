package ggplot_test

import (
	"context"
	"math"
	"math/rand"
	"testing"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
)

var benchSink any // prevent compiler from optimizing away Render calls

// --- Helpers ---

func benchEng() *memory.Engine {
	return memory.NewEngine(context.Background())
}

func benchPointDS(n int) dataset.Dataset {
	eng := benchEng()
	xs := make([]float64, n)
	ys := make([]float64, n)

	for i := range xs {
		xs[i] = float64(i)
		ys[i] = math.Sin(float64(i)*0.01) + rand.Float64()*0.1
	}

	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)

	return ds
}

func benchGroupedDS(n, nGroups int) dataset.Dataset {
	eng := benchEng()
	xs := make([]float64, n)
	ys := make([]float64, n)
	groups := make([]string, n)

	labels := make([]string, nGroups)
	for i := range labels {
		labels[i] = string(rune('A' + i))
	}

	for i := range n {
		xs[i] = float64(i % (n / nGroups))
		ys[i] = math.Sin(float64(i)*0.01) + rand.Float64()
		groups[i] = labels[i%nGroups]
	}

	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
		eng.NewStringColumn("group", groups),
	)

	return ds
}

func benchHistDS(n int) dataset.Dataset {
	eng := benchEng()

	xs := make([]float64, n)
	for i := range xs {
		xs[i] = rand.NormFloat64()*5 + 50
	}

	ds, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
	)

	return ds
}

// --- Benchmarks ---

func BenchmarkRender_Point_1K(b *testing.B) {
	ds := benchPointDS(1000)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point())
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		benchSink, _ = drawPlot(ctx, p, 800, 600)
	}
}

func BenchmarkRender_Point_10K(b *testing.B) {
	ds := benchPointDS(10_000)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point())
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		benchSink, _ = drawPlot(ctx, p, 800, 600)
	}
}

func BenchmarkRender_Point_100K(b *testing.B) {
	ds := benchPointDS(100_000)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point())
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		benchSink, _ = drawPlot(ctx, p, 800, 600)
	}
}

func BenchmarkRender_Line_10K(b *testing.B) {
	ds := benchPointDS(10_000)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Line())
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		benchSink, _ = drawPlot(ctx, p, 800, 600)
	}
}

func BenchmarkRender_Histogram_10K(b *testing.B) {
	ds := benchHistDS(10_000)
	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Histogram(geom.WithBins(50)))
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		benchSink, _ = drawPlot(ctx, p, 800, 600)
	}
}

func BenchmarkRender_Histogram_100K(b *testing.B) {
	ds := benchHistDS(100_000)
	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Histogram(geom.WithBins(100)))
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		benchSink, _ = drawPlot(ctx, p, 800, 600)
	}
}

func BenchmarkRender_Smooth_1K(b *testing.B) {
	ds := benchPointDS(1000)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Smooth(geom.WithSmoothPoints(80)))
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		benchSink, _ = drawPlot(ctx, p, 800, 600)
	}
}

func BenchmarkRender_Smooth_10K(b *testing.B) {
	ds := benchPointDS(10_000)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Smooth(geom.WithSmoothPoints(200)))
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		benchSink, _ = drawPlot(ctx, p, 800, 600)
	}
}

func BenchmarkRender_ColorGrouped_Point_10K(b *testing.B) {
	ds := benchGroupedDS(10_000, 5)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Point())
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		benchSink, _ = drawPlot(ctx, p, 800, 600)
	}
}

func BenchmarkRender_Density_10K(b *testing.B) {
	ds := benchHistDS(10_000)
	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Density(geom.WithDensityPoints(512)))
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		benchSink, _ = drawPlot(ctx, p, 800, 600)
	}
}

func BenchmarkRender_MultiLayer_10K(b *testing.B) {
	ds := benchPointDS(10_000)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithAlpha(0.3))).
		Layer(geom.Line()).
		Layer(geom.Rug())
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		benchSink, _ = drawPlot(ctx, p, 800, 600)
	}
}
