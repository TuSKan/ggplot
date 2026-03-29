package aes

import "github.com/TuSKan/ggplot/internal/ast"

// Opt configures an aesthetic mapping on a layer or plot.
// It maps visual grammar attributes to dataset column strings.
type Opt func(*ast.AestheticMapping)

// X dictates the horizontal axis mapping.
func X(col string) Opt {
	return func(m *ast.AestheticMapping) { m.X = col }
}

// Y dictates the vertical axis mapping.
func Y(col string) Opt {
	return func(m *ast.AestheticMapping) { m.Y = col }
}

// Color dictates the color channel mapping (stroke or fill).
func Color(col string) Opt {
	return func(m *ast.AestheticMapping) { m.Color = col }
}
