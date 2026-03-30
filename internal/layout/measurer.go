package layout

import (
	"github.com/TuSKan/ggplot/internal/fonts"
	"github.com/TuSKan/ggplot/pkg/theme"
)

// TextMeasurer defines an externalized callback allowing specific renderers (SVG, Canvas, Ebiten)
// to provide bounding font boxes enabling explicit layout calculations deterministically.
type TextMeasurer interface {
	MeasureText(text string, f theme.FontConfig) (width, height float64)
}

// DummyMeasurer simply maps constant character offsets testing pure layouts deterministically.
type DummyMeasurer struct{}

// MeasureText asserts constant sizing constraints avoiding native dependencies.
func (d DummyMeasurer) MeasureText(text string, f theme.FontConfig) (float64, float64) {
	// e.g., 8px width per char, 15px fixed height
	return float64(len(text)) * (f.Size * 0.5), f.Size * 1.5
}

// FontMeasurer directly computes actual structural dimensions executing native limits.
type FontMeasurer struct {
	Resolver *fonts.Resolver
}

// MeasureText leverages `fonts.Resolver` bounds parsing completely explicit caching bounds.
func (m *FontMeasurer) MeasureText(text string, f theme.FontConfig) (float64, float64) {
	if text == "" {
		return 0, f.Size * 1.2
	}

	handle, err := m.Resolver.LoadFace(f.ToFaceRequest(96.0))
	if err != nil {
		// Fallback geometry mapping exactly.
		return float64(len(text)) * (f.Size * 0.5), f.Size * 1.5
	}

	extents, err := handle.MeasureExtents(text)
	if err != nil {
		return float64(len(text)) * (f.Size * 0.5), f.Size * 1.5
	}

	return extents.Width, extents.Height
}

// Guarantee interface match
var _ TextMeasurer = DummyMeasurer{}
