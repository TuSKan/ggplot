// pkg/theme/theme.go
package theme

// Theme contains aesthetic definitions (font face, margins, grid lines).
type Theme struct {
	Name string
}

// Classic returns a clean structural bounding layout configuration.
func Classic() Theme {
	return Theme{Name: "classic"}
}
