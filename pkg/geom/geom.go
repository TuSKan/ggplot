package geom

// Opts defines geometry-specific parameters outside of mapped variable scales
// (like fixed radiuses or stroke opacities).
type Opts struct {
	Radius    float64
	Opacity   float64
	LineWidth float64
	Color     string  // Hex color e.g. "#4C72B0" statically.
	Fill      string  // Hex color functionally!
	Width     float64 // Relative width [0.0 - 1.0] statically.
	Bins      int     // Discrete capacity statically!
}
