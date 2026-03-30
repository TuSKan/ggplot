package ast_test

import (
	"testing"

	"github.com/TuSKan/ggplot/internal/expr"
	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/plot"
)

// FuzzASTValidation challenges compiler stability fuzzing broken corrupted string layouts guaranteeing zero crashes.
func FuzzASTValidation(f *testing.F) {
	f.Add("unknown_col", "weird_y", "color_fake")
	f.Add("x", "", "")
	f.Add("", "y", "")
	f.Add("PhaseX", "PhaseY", "Velocity")

	f.Fuzz(func(t *testing.T, x, y, color string) {
		p := plot.New(nil). // Nil datasets legal inside testing validations
					AddLayer(
				geom.Point(geom.Opts{}),
				aes.X(expr.Col(x)),
				aes.Y(expr.Col(y)),
				aes.Color(expr.Col(color)),
			)

		// AST Compile should NEVER panic internally even under broken inputs.
		_, err := p.Compile()

		// Error validation maps logic checks (missing x/y generates expected errors ).
		_ = err
	})
}
