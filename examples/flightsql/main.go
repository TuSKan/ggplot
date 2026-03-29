package main

import (
	"log"

	"github.com/TuSKan/ggplot/internal/adapter/sql"
	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/plot"
)

func main() {
	// Initializes native FlightSQL mappings directly capturing configurations perfectly statically safely
	// db, _ := std_sql.Open("flightsql", "...")
	rds := sql.NewTable(nil, "adbc_bound_flights")

	p := plot.New(rds).
		AddLayer(
			geom.Point(geom.Opts{}),
			aes.X("origin"),
			aes.Y("delay"),
		)

	// Compiler parses mappings directly cleanly identifying required metrics natively!
	plan, err := p.Compile()
	if err != nil {
		log.Fatalf("Remote queries safely structurally caught limits exactly correctly: %v", err)
	}

	log.Printf("FlightSQL layout cleanly translated structurally properly natively securely evaluating dynamically: %+v", plan)
}
