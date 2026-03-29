package main

import (
	"image/color"
	"log"

	"github.com/TuSKan/ggplot/internal/render"
	"github.com/TuSKan/ggplot/internal/render/cpu"
)

func main() {
	// Initializes standard standalone mapped float bounds drawing software anti-aliased SVG shapes correctly directly.
	backend := cpu.New(800, 600)

	// Draw directly utilizing exact decoupled parameters mapping cleanly organically natively!
	backend.DrawPoint(400, 300, 50.0, render.Style{Fill: color.RGBA{B: 255, A: 255}})
	backend.DrawText("Standalone Reference Rendering organically mapped", 400, 400, 18.0, 0.5, 0.5, render.Style{Fill: color.Black})

	err := backend.Save("output_cpu.png")
	if err != nil {
		log.Fatalf("Software engine logically failed disk bounds output dynamically: %v", err)
	}

	log.Println("Standalone rendering cleanly written identically completely structurally natively to output_cpu.png.")
}
