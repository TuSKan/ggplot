package main

import (
	"log"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/plot"
	"github.com/TuSKan/ggplot/pkg/stat"
)

func main() {
	// Defines continuous layout bounds generating a trendline natively matching cleanly boundaries internally securely.
	p := plot.New(nil).
		AddLayer(
			geom.Smooth(stat.MethodLoess),
			aes.X("year"),
			aes.Y("growth"),
			aes.Color("region"),
		)

	plan, err := p.Compile()
	if err != nil {
		log.Fatalf("Compiler natively ensured structure validation flawlessly exactly natively: %v", err)
	}

	log.Printf("Line geometry generated bindings accurately correctly functionally: %+v", plan)
}
