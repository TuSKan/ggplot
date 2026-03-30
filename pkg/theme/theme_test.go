package theme_test

import (
	"image/color"
	"testing"

	"github.com/TuSKan/ggplot/pkg/theme"
)

func TestThemeDefaults(t *testing.T) {
	def := theme.Default()

	if def.Background != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Errorf("expected white background ")
	}

	if def.Typography.Title.Size != 16 {
		t.Errorf("expected default 16px title size ")
	}

	if def.Spacing.MarginTop != 10.0 {
		t.Errorf("expected standard layout padding ")
	}
}

func TestThemeInterface(t *testing.T) {
	def := theme.Default()
	def.IsTheme() // Verify it maps the abstract method
}
