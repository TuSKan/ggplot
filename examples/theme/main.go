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
			aes.X("category"),
			aes.Y("value"),
		)
	// Apply layout modifiers dynamically (e.g., .WithTheme(theme.Classic())) mapping constraints cleanly.

	plan, err := p.Compile()
	if err != nil {
		log.Fatalf("Safely tracked boundary cleanly exactly properly organically effectively safely: %v", err)
	}

	log.Printf("Themes organically effectively safely dynamically compiled statically appropriately: %+v", plan)
}
