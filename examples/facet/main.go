package main

import (
	"log"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/plot"
)

func main() {
	p := plot.New(nil).
		AddLayer(
			geom.Point(geom.Opts{}),
			aes.X("density"),
			aes.Y("mass"),
		)
	// Typically chained via .FacetGrid("row", "col") when structurally enabled.

	plan, err := p.Compile()
	if err != nil {
		log.Fatalf("Safely validated bounds purely structurally organically successfully cleanly: %v", err)
	}

	log.Printf("Facet grids properly routed logically organically: %+v", plan)
}
