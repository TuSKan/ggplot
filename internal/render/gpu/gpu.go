package gpu

import (
	"github.com/TuSKan/ggplot/internal/render"
)

// Backend is the optional GPU accelerated renderer.
type Backend struct{}

// Verify at compile time that gpu.Backend implements render.Backend
var _ render.Backend = (*Backend)(nil)

func (b *Backend) SetClipRect(r render.Rect)                                                        {}
func (b *Backend) ClearClip()                                                                       {}
func (b *Backend) DrawPoint(x, y, radius float64, s render.Style)                                   {}
func (b *Backend) DrawLine(x1, y1, x2, y2 float64, s render.Style)                                  {}
func (b *Backend) DrawPolygon(points []render.Point, s render.Style)                                {}
func (b *Backend) DrawRect(r render.Rect, s render.Style)                                           {}
func (b *Backend) DrawText(text string, x, y, size float64, alignH, alignV float64, s render.Style) {}
func (b *Backend) Save(path string) error                                                           { return nil }
