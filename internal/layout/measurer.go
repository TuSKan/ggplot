package layout

// TextMeasurer defines an externalized callback allowing specific renderers (SVG, Canvas, Ebiten)
// to accurately provide bounding font boxes enabling explicit layout calculations deterministically.
type TextMeasurer interface {
	MeasureText(text string, size float64) (width, height float64)
}

// DummyMeasurer simply maps constant character offsets testing pure layouts deterministically.
type DummyMeasurer struct{}

// MeasureText asserts constant sizing constraints avoiding native dependencies.
func (d DummyMeasurer) MeasureText(text string, size float64) (float64, float64) {
	// e.g., 8px width per char, 15px fixed height
	return float64(len(text)) * (size * 0.5), size * 1.5
}

// Guarantee interface match
var _ TextMeasurer = DummyMeasurer{}
