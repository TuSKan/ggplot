// Example: Multi-Group Scatter with Colour Mapping and Legend
//
// Demonstrates aes.Color("species") to automatically assign distinct
// colours per group, split data, and render a legend.
package main

import (
	"context"
	"log"
	"math/rand"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/colormap"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
)

func main() {
	// Simulated iris-like data: 3 species, each with clustered (x,y).
	species := []string{
		"setosa", "setosa", "setosa", "setosa", "setosa",
		"setosa", "setosa", "setosa", "setosa", "setosa",
		"setosa", "setosa", "setosa", "setosa", "setosa",
		"setosa", "setosa", "setosa", "setosa", "setosa",
		"versicolor", "versicolor", "versicolor", "versicolor", "versicolor",
		"versicolor", "versicolor", "versicolor", "versicolor", "versicolor",
		"versicolor", "versicolor", "versicolor", "versicolor", "versicolor",
		"versicolor", "versicolor", "versicolor", "versicolor", "versicolor",
		"virginica", "virginica", "virginica", "virginica", "virginica",
		"virginica", "virginica", "virginica", "virginica", "virginica",
		"virginica", "virginica", "virginica", "virginica", "virginica",
		"virginica", "virginica", "virginica", "virginica", "virginica",
	}

	n := len(species)
	sepalLen := make([]float64, n)
	sepalWid := make([]float64, n)

	for i := range n {
		switch species[i] {
		case "setosa":
			sepalLen[i] = 5.0 + rand.NormFloat64()*0.35
			sepalWid[i] = 3.4 + rand.NormFloat64()*0.38
		case "versicolor":
			sepalLen[i] = 5.9 + rand.NormFloat64()*0.52
			sepalWid[i] = 2.8 + rand.NormFloat64()*0.31
		case "virginica":
			sepalLen[i] = 6.6 + rand.NormFloat64()*0.64
			sepalWid[i] = 3.0 + rand.NormFloat64()*0.32
		}
	}

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("sepal_length", sepalLen),
		eng.NewFloat64Column("sepal_width", sepalWid),
		eng.NewStringColumn("species", species),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds,
		aes.X("sepal_length"),
		aes.Y("sepal_width"),
		aes.Color("species"),
	).
		Layer(geom.Point(geom.WithSize(4), geom.WithAlpha(0.8))).
		ScaleColor(colormap.Set1).
		Labs(
			ggplot.Title("Iris Sepal Dimensions"),
			ggplot.Subtitle("Coloured by species (Set1 palette)"),
			ggplot.XLab("Sepal Length (cm)"),
			ggplot.YLab("Sepal Width (cm)"),
		)

	_, filename, _, _ := runtime.Caller(0)

	outPath := filepath.Join(filepath.Dir(filename), "color_mapping.png")
	if err := p.Save(context.Background(), outPath, 900, 600); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Saved %s", outPath)
}
