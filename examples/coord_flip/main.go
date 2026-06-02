// Example: Orientation — every geometry flipped to horizontal.
//
// Demonstrates geom.WithOrientation(geom.Horizontal) across all
// directional geometries, plus CoordFlip() for point/line geoms.
package main

import (
	"context"
	"log"
	"math"
	"math/rand"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/file"
)

func main() {
	ctx := context.Background()
	eng := memory.NewEngine(ctx)
	_, srcFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(srcFile)

	// ---------- shared data generators ----------

	catDS := func() dataset.Dataset {
		ds, _ := dataset.NewDataset(eng,
			eng.NewStringColumn("category", []string{"Go", "Rust", "Python", "Java", "C++"}),
			eng.NewFloat64Column("value", []float64{72, 68, 91, 54, 47}),
		)

		return ds
	}

	numDS := func(n int, seed int64) ([]float64, []float64) {
		rng := rand.New(rand.NewSource(seed))
		xs := make([]float64, n)
		ys := make([]float64, n)

		for i := range xs {
			xs[i] = float64(i)
			ys[i] = math.Sin(float64(i)*0.15) + rng.Float64()*0.4
		}

		return xs, ys
	}

	// ---------- 1. Horizontal Bar (Col) ----------
	log.Println("01_horizontal_bar.png")

	pBar := ggplot.New(catDS(), aes.X("category"), aes.Y("value")).
		Layer(geom.Col(
			geom.WithFill("#3b82f6"), geom.WithAlpha(0.85),
			geom.WithOrientation(geom.Horizontal),
		)).
		Labs(ggplot.Title("Horizontal Bar"), ggplot.XLab("Language"), ggplot.YLab("Score")).
		Theme("minimal")
	file.Save(ctx, pBar, filepath.Join(dir, "01_horizontal_bar.png"), 700, 450) //nolint:errcheck // error checked by test assertion.

	// ---------- 2. Horizontal Histogram ----------
	log.Println("02_horizontal_histogram.png")

	rng := rand.New(rand.NewSource(99))

	vals := make([]float64, 500)
	for i := range vals {
		vals[i] = rng.NormFloat64()*15 + 50
	}

	dsHist, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", vals),
	)
	pHist := ggplot.New(dsHist, aes.X("x")).
		Layer(geom.Histogram(
			geom.WithBins(25), geom.WithFill("#8b5cf6"), geom.WithAlpha(0.8),
			geom.WithOrientation(geom.Horizontal),
		)).
		Labs(ggplot.Title("Horizontal Histogram"), ggplot.XLab("Value"), ggplot.YLab("Count")).
		Theme("minimal")
	file.Save(ctx, pHist, filepath.Join(dir, "02_horizontal_histogram.png"), 700, 450) //nolint:errcheck // error checked by test assertion.

	// ---------- 3. Horizontal Points (scatter) ----------
	log.Println("03_horizontal_scatter.png")

	xs, ys := numDS(80, 42)
	dsPt, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs),
		eng.NewFloat64Column("y", ys),
	)
	pPt := ggplot.New(dsPt, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(3), geom.WithColor("#ef4444"), geom.WithAlpha(0.7))).
		CoordFlip().
		Labs(ggplot.Title("Flipped Scatter")).
		Theme("minimal")
	file.Save(ctx, pPt, filepath.Join(dir, "03_horizontal_scatter.png"), 700, 450) //nolint:errcheck // error checked by test assertion.

	// ---------- 4. Horizontal Line ----------
	log.Println("04_horizontal_line.png")

	xs2, ys2 := numDS(60, 7)
	dsLine, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", xs2),
		eng.NewFloat64Column("y", ys2),
	)
	pLine := ggplot.New(dsLine, aes.X("x"), aes.Y("y")).
		Layer(geom.Line(geom.WithColor("#0ea5e9"), geom.WithLineWidth(2))).
		CoordFlip().
		Labs(ggplot.Title("Flipped Line")).
		Theme("minimal")
	file.Save(ctx, pLine, filepath.Join(dir, "04_horizontal_line.png"), 700, 450) //nolint:errcheck // error checked by test assertion.

	// ---------- 5. Horizontal Multi-Line (color groups) ----------
	log.Println("05_horizontal_multiline.png")

	n := 50
	mlx := make([]float64, 0, n*3)
	mly := make([]float64, 0, n*3)

	mlg := make([]string, 0, n*3)
	for i := range n {
		t := float64(i) * 0.15
		mlx = append(mlx, float64(i), float64(i), float64(i))
		mly = append(mly,
			math.Sin(t),
			math.Sin(t+math.Pi/3)*0.8,
			math.Sin(t+2*math.Pi/3)*0.6,
		)
		mlg = append(mlg, "Signal A", "Signal B", "Signal C")
	}

	dsML, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("t", mlx),
		eng.NewFloat64Column("v", mly),
		eng.NewStringColumn("signal", mlg),
	)
	pML := ggplot.New(dsML, aes.X("t"), aes.Y("v"), aes.Color("signal")).
		Layer(geom.Line(geom.WithLineWidth(2))).
		CoordFlip().
		Labs(ggplot.Title("Flipped Multi-Line"), ggplot.XLab("Time"), ggplot.YLab("Value")).
		Theme("minimal")
	file.Save(ctx, pML, filepath.Join(dir, "05_horizontal_multiline.png"), 700, 450) //nolint:errcheck // error checked by test assertion.

	// ---------- 6. Horizontal Step ----------
	log.Println("06_horizontal_step.png")

	stepX := make([]float64, 20)
	stepY := make([]float64, 20)

	for i := range stepX {
		stepX[i] = float64(i)
		stepY[i] = math.Floor(math.Sin(float64(i)*0.4)*3) + 5
	}

	dsStep, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", stepX),
		eng.NewFloat64Column("y", stepY),
	)
	pStep := ggplot.New(dsStep, aes.X("x"), aes.Y("y")).
		Layer(geom.Step(geom.WithColor("#f59e0b"), geom.WithLineWidth(2))).
		CoordFlip().
		Labs(ggplot.Title("Flipped Step")).
		Theme("minimal")
	file.Save(ctx, pStep, filepath.Join(dir, "06_horizontal_step.png"), 700, 450) //nolint:errcheck // error checked by test assertion.

	// ---------- 7. Horizontal Area ----------
	log.Println("07_horizontal_area.png")

	areaX := make([]float64, 40)
	areaY := make([]float64, 40)

	for i := range areaX {
		areaX[i] = float64(i)
		areaY[i] = math.Sin(float64(i)*0.2)*3 + 5
	}

	dsArea, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", areaX),
		eng.NewFloat64Column("y", areaY),
	)
	pArea := ggplot.New(dsArea, aes.X("x"), aes.Y("y")).
		Layer(geom.Area(
			geom.WithFill("#10b981"), geom.WithAlpha(0.5),
			geom.WithOrientation(geom.Horizontal),
		)).
		Labs(ggplot.Title("Horizontal Area")).
		Theme("minimal")
	file.Save(ctx, pArea, filepath.Join(dir, "07_horizontal_area.png"), 700, 450) //nolint:errcheck // error checked by test assertion.

	// ---------- 8. Horizontal Density ----------
	log.Println("08_horizontal_density.png")

	dVals := make([]float64, 300)

	rng2 := rand.New(rand.NewSource(123))
	for i := range dVals {
		dVals[i] = rng2.NormFloat64()*10 + 50
	}

	dsDens, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", dVals),
	)
	pDens := ggplot.New(dsDens, aes.X("x")).
		Layer(geom.Density(
			geom.WithFill("#ec4899"), geom.WithAlpha(0.5),
			geom.WithOrientation(geom.Horizontal),
		)).
		Labs(ggplot.Title("Horizontal Density")).
		Theme("minimal")
	file.Save(ctx, pDens, filepath.Join(dir, "08_horizontal_density.png"), 700, 450) //nolint:errcheck // error checked by test assertion.

	// ---------- 9. Horizontal Boxplot ----------
	log.Println("09_horizontal_boxplot.png")

	bxGroups := make([]string, 0, 120)
	bxVals := make([]float64, 0, 120)

	rng3 := rand.New(rand.NewSource(77))

	for range 60 {
		bxGroups = append(bxGroups, "Control", "Treatment")
		bxVals = append(bxVals,
			rng3.NormFloat64()*5+20,
			rng3.NormFloat64()*8+35,
		)
	}

	dsBox, _ := dataset.NewDataset(eng,
		eng.NewStringColumn("group", bxGroups),
		eng.NewFloat64Column("score", bxVals),
	)
	pBox := ggplot.New(dsBox, aes.X("group"), aes.Y("score")).
		Layer(geom.Boxplot(
			geom.WithFill("#6366f1"), geom.WithAlpha(0.7),
			geom.WithOrientation(geom.Horizontal),
		)).
		Labs(ggplot.Title("Horizontal Boxplot"), ggplot.XLab("Group"), ggplot.YLab("Score")).
		Theme("minimal")
	file.Save(ctx, pBox, filepath.Join(dir, "09_horizontal_boxplot.png"), 700, 450) //nolint:errcheck // error checked by test assertion.

	// ---------- 10. Horizontal Smooth (LOESS) ----------
	log.Println("10_horizontal_smooth.png")

	smX := make([]float64, 100)
	smY := make([]float64, 100)
	rng4 := rand.New(rand.NewSource(55))

	for i := range smX {
		smX[i] = float64(i) * 0.1
		smY[i] = math.Sin(smX[i]) + rng4.NormFloat64()*0.3
	}

	dsSm, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", smX),
		eng.NewFloat64Column("y", smY),
	)
	pSm := ggplot.New(dsSm, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(2), geom.WithAlpha(0.3), geom.WithColor("#94a3b8"))).
		Layer(geom.Smooth(geom.WithColor("#ef4444"), geom.WithLineWidth(2))).
		CoordFlip().
		Labs(ggplot.Title("Flipped Smooth (LOESS)")).
		Theme("minimal")
	file.Save(ctx, pSm, filepath.Join(dir, "10_horizontal_smooth.png"), 700, 450) //nolint:errcheck // error checked by test assertion.

	// ---------- 11. Flipped HLine / VLine ----------
	log.Println("11_flipped_reflines.png")

	dsPt2, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8}),
		eng.NewFloat64Column("y", []float64{12, 18, 15, 22, 20, 25, 23, 28}),
	)
	pRef := ggplot.New(dsPt2, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(4), geom.WithColor("#0ea5e9"))).
		Layer(geom.HLine(geom.WithIntercept(20), geom.WithColor("#ef4444"), geom.WithLineWidth(1.5))).
		Layer(geom.VLine(geom.WithIntercept(4), geom.WithColor("#22c55e"), geom.WithLineWidth(1.5))).
		CoordFlip().
		Labs(ggplot.Title("Flipped Reference Lines")).
		Theme("minimal")
	file.Save(ctx, pRef, filepath.Join(dir, "11_flipped_reflines.png"), 700, 450) //nolint:errcheck // error checked by test assertion.

	// ---------- 12. Flipped Rug ----------
	log.Println("12_flipped_rug.png")

	rugX := make([]float64, 80)
	rugY := make([]float64, 80)

	rng5 := rand.New(rand.NewSource(33))
	for i := range rugX {
		rugX[i] = rng5.NormFloat64()*3 + 5
		rugY[i] = rng5.NormFloat64()*2 + 3
	}

	dsRug, _ := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", rugX),
		eng.NewFloat64Column("y", rugY),
	)
	pRug := ggplot.New(dsRug, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(2), geom.WithAlpha(0.5), geom.WithColor("#8b5cf6"))).
		Layer(geom.Rug(geom.WithAlpha(0.4), geom.WithColor("#8b5cf6"))).
		CoordFlip().
		Labs(ggplot.Title("Flipped Scatter + Rug")).
		Theme("minimal")
	file.Save(ctx, pRug, filepath.Join(dir, "12_flipped_rug.png"), 700, 450) //nolint:errcheck // error checked by test assertion.

	log.Println("All 12 orientation examples generated.")
}
