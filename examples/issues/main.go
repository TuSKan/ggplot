package main

import (
	"math"

	"github.com/gogpu/gg"
	_ "github.com/gogpu/gg/gpu" // opt-in GPU acceleration
)

func main() {
	const (
		w, h = 800, 500
		n    = 100
	)

	dc := gg.NewContext(w, h)
	defer dc.Close()

	// White background
	dc.SetRGB(1, 1, 1)
	dc.DrawRectangle(0, 0, w, h)
	dc.Fill()

	// Blue polyline stroke
	dc.SetRGB(0.2, 0.6, 0.85)
	dc.SetLineWidth(2)

	for i := 0; i < n; i++ {
		t := float64(i) * 0.1
		x := 50 + t*70 // 0..700 px
		y := 250 - math.Sin(t)*math.Exp(-t*0.1)*200

		if i == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
	}

	dc.Stroke() // ← BUG: renders as filled polygon, not stroked line

	dc.SavePNG("stroke_bug_gpu.png")
}
