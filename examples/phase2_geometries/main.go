// Phase 2: Geometries — Point, Line, Step, Bar, Histogram, Area, Density, Rug, HLine, VLine, Text, BoxPlot, Smooth
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
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	pointExample(dir)
	lineExample(dir)
	stepExample(dir)
	barExample(dir)
	histogramExample(dir)
	areaExample(dir)
	densityExample(dir)
	rugExample(dir)
	hlineVlineExample(dir)
	textExample(dir)
	boxplotExample(dir)
	smoothExample(dir)
}

func save(p *ggplot.Plot, dir, name string, w, h int) {
	out := filepath.Join(dir, name+".png")
	if err := p.Save(context.Background(), out, w, h); err != nil {
		log.Fatalln(err)
	}
	log.Printf("Saved %s", out)
}

// --- Point ---
func pointExample(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 150
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = rng.NormFloat64() * 5
		ys[i] = xs[i]*0.6 + rng.NormFloat64()*2
	}
	eng := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng, eng.NewFloat64Column("x", xs), eng.NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(3), geom.WithAlpha(0.7), geom.WithColor("#E74C3C"))).
		Labs(ggplot.Title("geom.Point"), ggplot.Subtitle("Scatter plot with random data")).
		Theme("dark")
	save(p, dir, "01_point", 800, 600)
}

// --- Line ---
func lineExample(dir string) {
	n := 100
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		t := float64(i) * 0.1
		xs[i] = t
		ys[i] = math.Sin(t) * math.Exp(-t*0.1)
	}
	eng2 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng2, eng2.NewFloat64Column("t", xs), eng2.NewFloat64Column("amplitude", ys))
	p := ggplot.New(ds, aes.X("t"), aes.Y("amplitude")).
		Layer(geom.Line(geom.WithColor("#3498DB"), geom.WithLineWidth(2))).
		Labs(ggplot.Title("geom.Line"), ggplot.Subtitle("Damped sine wave")).
		Theme("minimal")
	save(p, dir, "02_line", 800, 500)
}

// --- Step ---
func stepExample(dir string) {
	n := 40
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i)
		ys[i] = math.Floor(math.Sin(float64(i)*0.3)*4) + 5
	}
	eng3 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng3, eng3.NewFloat64Column("time", xs), eng3.NewFloat64Column("level", ys))
	p := ggplot.New(ds, aes.X("time"), aes.Y("level")).
		Layer(geom.Step(geom.WithColor("#2ECC71"), geom.WithLineWidth(2))).
		Labs(ggplot.Title("geom.Step"), ggplot.Subtitle("Staircase function")).
		Theme("bw")
	save(p, dir, "03_step", 800, 500)
}

// --- Bar ---
func barExample(dir string) {
	eng4 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng4,
		eng4.NewStringColumn("fruit", []string{"Apple", "Banana", "Cherry", "Date", "Elderberry"}),
		eng4.NewFloat64Column("sales", []float64{45, 32, 58, 21, 39}),
	)
	p := ggplot.New(ds, aes.X("fruit"), aes.Y("sales")).
		Layer(geom.Col(geom.WithFill("#9B59B6"), geom.WithAlpha(0.85))).
		Labs(ggplot.Title("geom.Col (Bar)"), ggplot.Subtitle("Fruit sales by category")).
		Theme("classic")
	save(p, dir, "04_bar", 800, 500)
}

// --- Histogram ---
func histogramExample(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 500
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = rng.NormFloat64()*15 + 50
	}
	eng5 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng5, eng5.NewFloat64Column("score", xs))
	p := ggplot.New(ds, aes.X("score")).
		Layer(geom.Histogram(geom.WithFill("#E67E22"), geom.WithAlpha(0.8))).
		Labs(ggplot.Title("geom.Histogram"), ggplot.Subtitle("Distribution of test scores")).
		Theme("dark")
	save(p, dir, "05_histogram", 800, 500)
}

// --- Area ---
func areaExample(dir string) {
	n := 80
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		t := float64(i) * 0.1
		xs[i] = t
		ys[i] = math.Sin(t) * math.Sin(t) * 3
	}
	eng6 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng6, eng6.NewFloat64Column("x", xs), eng6.NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Area(geom.WithFill("#1ABC9C"), geom.WithAlpha(0.6))).
		Labs(ggplot.Title("geom.Area"), ggplot.Subtitle("Filled area under sin²(x)")).
		Theme("minimal")
	save(p, dir, "06_area", 800, 500)
}

// --- Density ---
func densityExample(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 400
	xs := make([]float64, n)
	for i := range xs {
		if i < n/2 {
			xs[i] = rng.NormFloat64()*5 + 30
		} else {
			xs[i] = rng.NormFloat64()*3 + 50
		}
	}
	eng7 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng7, eng7.NewFloat64Column("value", xs))
	p := ggplot.New(ds, aes.X("value")).
		Layer(geom.Density(geom.WithFill("#3498DB"), geom.WithAlpha(0.5), geom.WithColor("#2C3E50"))).
		Labs(ggplot.Title("geom.Density"), ggplot.Subtitle("Kernel density estimation of bimodal data")).
		Theme("bw")
	save(p, dir, "07_density", 800, 500)
}

// --- Rug ---
func rugExample(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 80
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = rng.Float64() * 10
		ys[i] = math.Sin(xs[i]) + rng.NormFloat64()*0.3
	}
	eng8 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng8, eng8.NewFloat64Column("x", xs), eng8.NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(2.5), geom.WithColor("#8E44AD"), geom.WithAlpha(0.6))).
		Layer(geom.Rug(geom.WithAlpha(0.4), geom.WithColor("#8E44AD"))).
		Labs(ggplot.Title("geom.Rug"), ggplot.Subtitle("Marginal rug marks on scatter plot")).
		Theme("dark")
	save(p, dir, "08_rug", 800, 600)
}

// --- HLine + VLine ---
func hlineVlineExample(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 100
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = rng.Float64() * 20
		ys[i] = xs[i]*1.5 + rng.NormFloat64()*5
	}
	eng9 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng9, eng9.NewFloat64Column("x", xs), eng9.NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(2), geom.WithAlpha(0.6), geom.WithColor("#2980B9"))).
		Layer(geom.HLine(geom.WithIntercept(15), geom.WithColor("#E74C3C"), geom.WithLineWidth(1.5))).
		Layer(geom.VLine(geom.WithIntercept(10), geom.WithColor("#27AE60"), geom.WithLineWidth(1.5))).
		Labs(ggplot.Title("geom.HLine + geom.VLine"), ggplot.Subtitle("Reference lines at y=15, x=10")).
		Theme("minimal")
	save(p, dir, "09_hline_vline", 800, 600)
}

// --- Text ---
func textExample(dir string) {
	eng10 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng10,
		eng10.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng10.NewFloat64Column("y", []float64{2, 5, 3, 7, 4}),
		eng10.NewStringColumn("label", []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}),
	)
	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Label("label")).
		Layer(geom.Point(geom.WithSize(5), geom.WithColor("#E74C3C"))).
		Layer(geom.Text()).
		Labs(ggplot.Title("geom.Text"), ggplot.Subtitle("Labeled scatter points")).
		Theme("bw")
	save(p, dir, "10_text", 800, 600)
}

// --- BoxPlot ---
func boxplotExample(dir string) {
	rng := rand.New(rand.NewSource(42))
	var x, y []float64
	means := []float64{50, 65, 55}
	for g, m := range means {
		for i := 0; i < 50; i++ {
			x = append(x, float64(g+1))
			y = append(y, math.Max(0, m+rng.NormFloat64()*10))
		}
	}
	eng11 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng11, eng11.NewFloat64Column("group", x), eng11.NewFloat64Column("score", y))
	p := ggplot.New(ds, aes.X("group"), aes.Y("score")).
		Layer(geom.Boxplot(geom.WithFill("#E8E8E8"), geom.WithColor("#2C3E50"), geom.WithWidth(0.6))).
		Labs(ggplot.Title("geom.Boxplot"), ggplot.Subtitle("Three treatment groups")).
		Theme("classic")
	save(p, dir, "11_boxplot", 700, 500)
}

// --- Smooth ---
func smoothExample(dir string) {
	rng := rand.New(rand.NewSource(42))
	n := 80
	xs, ys := make([]float64, n), make([]float64, n)
	for i := range xs {
		xs[i] = float64(i) * 0.15
		ys[i] = math.Sin(xs[i]) + rng.NormFloat64()*0.4
	}
	eng12 := memory.NewEngine(context.Background())
	ds, _ := dataset.NewDataset(eng12, eng12.NewFloat64Column("x", xs), eng12.NewFloat64Column("y", ys))
	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(2), geom.WithAlpha(0.4), geom.WithColor("#BDC3C7"))).
		Layer(geom.Smooth(geom.WithColor("#E74C3C"), geom.WithLineWidth(2.5))).
		Labs(ggplot.Title("geom.Smooth"), ggplot.Subtitle("LOESS smooth over noisy sine")).
		Theme("dark")
	save(p, dir, "12_smooth", 800, 500)
}
