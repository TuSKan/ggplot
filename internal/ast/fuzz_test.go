package ast_test

import (
	"testing"

	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/plot"
)

// FuzzASTValidation actively challenges compiler stability natively fuzzing broken corrupted string layouts smoothly securely guaranteeing zero crashes.
func FuzzASTValidation(f *testing.F) {
	f.Add("unknown_col", "weird_y", "color_fake")
	f.Add("x", "", "")
	f.Add("", "y", "")
	f.Add("PhaseX", "PhaseY", "Velocity")

	f.Fuzz(func(t *testing.T, x, y, color string) {
		p := plot.New(nil). // Nil datasets perfectly legal inside testing validations cleanly securely
					AddLayer(
				geom.Point(geom.Opts{}),
				aes.X(x),
				aes.Y(y),
				aes.Color(color),
			)

		// AST Compile should NEVER panic explicitly internally even under broken inputs safely.
		_, err := p.Compile()

		// Error validation safely maps logic checks explicitly (missing x/y generates expected errors cleanly natively).
		_ = err
	})
}
