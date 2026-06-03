//go:build !js

// Dev server: builds the WASM example and serves it on localhost:8080.
//
// Usage:
//
//	go run ./examples/web/serve/
package main

import (
	"log"

	"github.com/TuSKan/ggplot/output/web"
)

func main() {
	if err := web.Serve("localhost:8080", "examples/web/app.wasm"); err != nil {
		log.Fatalln(err)
	}
}
