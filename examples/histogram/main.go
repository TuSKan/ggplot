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
			geom.Point(geom.Opts{}), // Typically geom.Bar utilizing stat.MethodCount functionally
			aes.X("price"),
		)

	plan, err := p.Compile()
	if err != nil {
		log.Fatalf("Compiler natively caught structurally expected missing bounds flawlessly exactly correctly: %v", err)
	}

	log.Printf("Histogram strictly compiled natively: %+v", plan)
}
