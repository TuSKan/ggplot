package main

import (
	"log"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/plot"
)

func main() {
	// A typical explicit pipeline establishing native layout limits.
	p := plot.New(nil). // 'nil' simulates a bound dataset structurally mapped out.
		AddLayer(
			geom.Point(geom.Opts{Radius: 2.0, Opacity: 0.8}),
			aes.X("displacement"),
			aes.Y("horsepower"),
			aes.Color("cylinders"),
		)

	// The declarative compiler asserts requirements organically securing safety natively.
	plan, err := p.Compile()
	if err != nil {
		log.Fatalf("Compiler safely caught bounding explicit errors organically: %v", err)
	}

	log.Printf("Scatter logically generated compilation structures properly: %+v", plan)
}
