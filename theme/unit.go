package theme

// UnitType identifies the measurement system for a dimension.
type UnitType int

const (
	// UnitPt is points. At the standard 72 DPI, 1 pt = 1 px.
	UnitPt UnitType = iota
	// UnitCm is centimeters. 1 cm ≈ 28.3465 pt at 72 DPI.
	UnitCm
	// UnitInch is inches. 1 in = 72 pt at 72 DPI.
	UnitInch
	// UnitLines is multiples of the base text line-height.
	// Useful for margins that scale with font size.
	UnitLines
)

// Standard conversion factors at 72 DPI.
const (
	ptPerInch = 72.0
	ptPerCm   = 28.3464566929 // 72 / 2.54
)

// Unit is a dimension with an associated unit type.
type Unit struct {
	Value float64
	Type  UnitType
}

// Pt returns a Unit in points.
func Pt(v float64) Unit { return Unit{Value: v, Type: UnitPt} }

// Cm returns a Unit in centimeters.
func Cm(v float64) Unit { return Unit{Value: v, Type: UnitCm} }

// Inches returns a Unit in inches.
func Inches(v float64) Unit { return Unit{Value: v, Type: UnitInch} }

// Lines returns a Unit in line-height multiples.
func Lines(v float64) Unit { return Unit{Value: v, Type: UnitLines} }

// ToPixels converts the unit to pixel value.
// lineHeight is the base text line-height in pixels, used only for UnitLines.
func (u Unit) ToPixels(lineHeight float64) float64 {
	switch u.Type {
	case UnitCm:
		return u.Value * ptPerCm
	case UnitInch:
		return u.Value * ptPerInch
	case UnitLines:
		if lineHeight <= 0 {
			lineHeight = 14 // fallback: ~11pt font × 1.2 line spacing
		}

		return u.Value * lineHeight
	case UnitPt:
		fallthrough
	default:
		return u.Value
	}
}

// PlotMargin specifies the outer margins of the plot in typed units.
// A nil *PlotMargin means "use the theme's default Spacing margins".
type PlotMargin struct {
	Top    Unit
	Right  Unit
	Bottom Unit
	Left   Unit
}
