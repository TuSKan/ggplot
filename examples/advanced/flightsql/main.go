package main

import (
	"log"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/dataset/sql"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/plot"
)

func main() {
	// Initializes native FlightSQL mappings directly capturing configurations statically
	// db, _ := std_sql.Open("flightsql", "...")
	rds := sql.NewTable(nil, "adbc_bound_flights")

	p := plot.New(rds).
		AddLayer(
			geom.Point(geom.Opts{}),
			aes.X(aes.Col("origin")),
			aes.Y(aes.Col("delay")),
		)

	// Compiler parses mappings directly identifying required metrics!
	plan, err := p.Compile()
	if err != nil {
		log.Fatalf("Remote queries caught limits exactly : %v", err)
	}

	log.Printf("FlightSQL layout translated evaluating : %+v", plan)
}
