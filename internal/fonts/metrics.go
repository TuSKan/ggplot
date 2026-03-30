package fonts

// Extents describes the geometric bounding layout for a specific rendered string returning drawing metrics mappings generically.
type Extents struct {
	Width      float64
	Height     float64
	Ascent     float64
	Descent    float64
	LineHeight float64
}
