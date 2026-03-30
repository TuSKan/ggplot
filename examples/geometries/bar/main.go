package main

import (
	"log"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/dataset/arrow"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/output/ui"
	"github.com/TuSKan/ggplot/pkg/plot"
)

func main() {
	regions := []string{"North", "South", "East", "West", "North", "North", "South", "East", "North", "West", "North", "North", "South", "East"}

	buf := arrow.NewBuffer(len(regions))
	region := buf.String("region")

	copy(region, regions)

	ds, err := buf.Build()
	if err != nil {
		log.Fatalln(err)
	}

	// Declarative pipeline mapping
	p := plot.New(ds).
		AddLayer(
			geom.Bar(geom.Opts{
				Color: "#000000",
				Fill:  "#4C72B0",
				Width: 0.8,
			}),
			aes.X(aes.Col("region")),
		)

	out, err := p.Render(800, 600)
	if err != nil {
		log.Fatalln(err)
	}

	// Output
	if err := ui.NewGPUWindowPresenter().Show(out); err != nil {
		log.Fatalln("Presenter :", err)
	}
}
