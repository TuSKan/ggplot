package theme

import "image/color"

// Element is a sealed interface for theme element types.
// Only ElementText, ElementLine, ElementRect, and ElementBlank implement it.
type Element interface {
	element() // sealed marker — only package types satisfy this
}

// ElementText controls text appearance (titles, labels, annotations).
type ElementText struct {
	Family     string      // font family ("sans-serif", "serif", "mono"); "" = inherit
	Size       float64     // font size in points; 0 = inherit
	Color      color.Color // text color; nil = inherit
	Bold       bool
	Italic     bool
	Hjust      float64 // horizontal justification [0,1] (0=left, 0.5=center, 1=right)
	Vjust      float64 // vertical justification [0,1]
	Angle      float64 // rotation in degrees
	Margin     Margin  // spacing around the text element
	LineHeight float64 // line spacing multiplier; 0 = default 1.2
}

func (ElementText) element() {}

// ElementLine controls line appearance (axes, grid lines, ticks).
type ElementLine struct {
	Color    color.Color // line color; nil = inherit
	Size     float64     // line width in pixels; 0 = inherit
	Linetype []float64   // dash pattern: nil = solid, e.g. {4,4} = dashed
	Lineend  string      // "butt", "round", "square"; "" = inherit
}

func (ElementLine) element() {}

// ElementRect controls rectangle appearance (backgrounds, borders).
type ElementRect struct {
	Fill     color.Color // interior fill; nil = inherit
	Color    color.Color // border stroke; nil = inherit
	Size     float64     // border width in pixels; 0 = inherit
	Linetype []float64   // border dash pattern
}

func (ElementRect) element() {}

// ElementBlank suppresses drawing of the element entirely.
// When encountered during inheritance resolution, the element is hidden.
type ElementBlank struct{}

func (ElementBlank) element() {}

// Margin holds spacing around an element (top, right, bottom, left in px).
type Margin struct {
	Top, Right, Bottom, Left float64
}

// IsBlank reports whether an Element is ElementBlank.
func IsBlank(e Element) bool {
	_, ok := e.(ElementBlank)
	return ok
}

// --- Merge helpers ---

// MergeText returns a new ElementText with zero-value fields in child
// filled from parent. Non-zero child fields take precedence.
func MergeText(child, parent ElementText) ElementText {
	if child.Family == "" {
		child.Family = parent.Family
	}

	if child.Size == 0 {
		child.Size = parent.Size
	}

	if child.Color == nil {
		child.Color = parent.Color
	}

	// Bold/Italic: only inherit if child is default (false).
	// This is intentional — a child that sets Bold=false should stay false,
	// but we can't distinguish "explicitly false" from "zero value" in Go.
	// So we always inherit Bold/Italic from parent unless child sets them.
	// Users who need to un-bold use an explicit ElementText with Bold=false
	// at the specific element path.

	if child.LineHeight == 0 {
		child.LineHeight = parent.LineHeight
	}

	if child.Margin == (Margin{}) {
		child.Margin = parent.Margin
	}

	return child
}

// MergeLine returns a new ElementLine with zero-value fields in child
// filled from parent.
func MergeLine(child, parent ElementLine) ElementLine {
	if child.Color == nil {
		child.Color = parent.Color
	}

	if child.Size == 0 {
		child.Size = parent.Size
	}

	if child.Linetype == nil {
		child.Linetype = parent.Linetype
	}

	if child.Lineend == "" {
		child.Lineend = parent.Lineend
	}

	return child
}

// MergeRect returns a new ElementRect with zero-value fields in child
// filled from parent.
func MergeRect(child, parent ElementRect) ElementRect {
	if child.Fill == nil {
		child.Fill = parent.Fill
	}

	if child.Color == nil {
		child.Color = parent.Color
	}

	if child.Size == 0 {
		child.Size = parent.Size
	}

	if child.Linetype == nil {
		child.Linetype = parent.Linetype
	}

	return child
}
