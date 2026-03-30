package cpu_test

import (
	"image/color"
	"testing"

	"github.com/TuSKan/ggplot/internal/fonts"
	"github.com/TuSKan/ggplot/internal/render"
	"github.com/TuSKan/ggplot/internal/render/cpu"
)

func BenchmarkCPUBackend(b *testing.B) {
	reg, _ := fonts.NewRegistry()
	res := fonts.NewResolver(reg, fonts.DefaultFallbackConfig())
	backend := cpu.New(800, 600, res)
	style := render.Style{
		Fill: color.RGBA{255, 0, 0, 128},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for p := 0; p < 10000; p++ {
			x := float64(p % 800)
			y := float64((p / 800) % 800)
			backend.DrawPoint(x, y, 2.0, style)
		}
	}
}

func runDrawPointBenchmark(b *testing.B, count int) {
	reg, _ := fonts.NewRegistry()
	res := fonts.NewResolver(reg, fonts.DefaultFallbackConfig())
	backend := cpu.New(800, 800, res)
	style := render.Style{
		Fill: color.RGBA{255, 0, 0, 128},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for p := 0; p < count; p++ {
			x := float64(p % 800)
			y := float64((p / 800) % 800)
			backend.DrawPoint(x, y, 2.0, style)
		}
	}
}

func BenchmarkDrawPoint_10k(b *testing.B)  { runDrawPointBenchmark(b, 10_000) }
func BenchmarkDrawPoint_100k(b *testing.B) { runDrawPointBenchmark(b, 100_000) }
func BenchmarkDrawPoint_1M(b *testing.B)   { runDrawPointBenchmark(b, 1_000_000) }
