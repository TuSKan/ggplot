package cpu_test

import (
	"crypto/md5"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/TuSKan/ggplot/internal/fonts"
	"github.com/TuSKan/ggplot/internal/render"
	"github.com/TuSKan/ggplot/internal/render/cpu"
	"github.com/TuSKan/ggplot/pkg/theme"
)

func TestCPU_GoldenImage(t *testing.T) {
	reg, _ := fonts.NewRegistry()
	res := fonts.NewResolver(reg, fonts.DefaultFallbackConfig())
	b := cpu.New(400, 400, res)

	antiAliasedRed := color.RGBA{255, 0, 0, 255}
	green := color.RGBA{0, 255, 0, 255}
	halfBlue := color.RGBA{0, 0, 255, 128}

	// 1. Polygon
	poly := []render.Point{{X: 100, Y: 10}, {X: 150, Y: 10}, {X: 125, Y: 50}}
	b.DrawPolygon(poly, render.Style{Fill: green})

	// 2. Point (Anti-aliased circle with alpha-blend)
	b.DrawPoint(200, 25, 10, render.Style{Fill: halfBlue})

	// 3. Rect w/ stroke
	b.DrawRect(render.Rect{Min: render.Point{X: 10, Y: 10}, Max: render.Point{X: 50, Y: 50}}, render.Style{
		Fill:        antiAliasedRed,
		StrokeWidth: 2,
		Stroke:      color.Black,
	})

	// 4. Line
	b.DrawLine(10, 100, 390, 100, render.Style{StrokeWidth: 5, Stroke: color.RGBA{128, 128, 128, 255}})

	// 5. Native Text Rendering
	th := theme.Default()
	b.DrawText("Core Reference Engine", 200, 150, 0.5, 0.5, render.Style{Fill: color.Black}, th.Typography.Title)

	// 6. Clipping Rect constraints (draw huge circle clamped )
	b.SetClipRect(render.Rect{Min: render.Point{X: 200, Y: 200}, Max: render.Point{X: 300, Y: 300}})
	b.DrawPoint(250, 250, 100, render.Style{Fill: color.RGBA{255, 128, 0, 255}})
	b.ClearClip()

	path := filepath.Join(t.TempDir(), "golden_verification.png")
	if err := b.Save(path); err != nil {
		t.Fatalf("Failed pipeline output writing artifacts: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		t.Fatalf("Pipeline output PNG missing or generated fundamentally corrupt bounds")
	}

	hash := fmt.Sprintf("%x", md5.Sum(data))
	expectedHash := "e08c1ab53514b01b55e27d8424b0e1e0" // Computed

	if hash != expectedHash {
		// Log the expected hash to the console if it mutates so the dev knows the drift.
		t.Errorf("Golden Image MD5 mismatch determinism drifted.\nExpected: %s\nGot:      %s", expectedHash, hash)
	} else {
		t.Logf("Golden Output Hash verified : %s", hash)
	}
}
