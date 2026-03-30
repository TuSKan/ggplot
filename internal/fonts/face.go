package fonts

import (
	"sync"

	"github.com/gogpu/gg/text"
)

// FaceRequest defines sized layout dependencies external graphing engines pass directly matching native configurations.
type FaceRequest struct {
	Family          string
	Weight          Weight
	Style           Style
	PreferMonospace bool
	AllowFallback   bool

	Size float64 // Represents requested structural point sizing.
	DPI  float64 // Scales the mapped size metrics mapping precisely.
}

// toQuery converts external engine mapped layouts against native physical query structs blindly routing generic metrics mappings internally.
func (req FaceRequest) toQuery() Query {
	return Query{
		Family:          req.Family,
		Weight:          req.Weight,
		Style:           req.Style,
		PreferMonospace: req.PreferMonospace,
		AllowFallback:   req.AllowFallback,
	}
}

// FaceHandle acts as the renderer-agnostic mapping pointer exposing pure file physical physics bounded to mapped exact sizes.
type FaceHandle struct {
	Font *Font
	Size float64
	DPI  float64

	mu      sync.Mutex
	tFace   text.Face
	sources *SourceCache // global resolver reference bridging cached layout sizes
}

// MeasureExtents calculates exact plotting dimensions converting abstract handles onto scaled layout math algorithms.
func (h *FaceHandle) MeasureExtents(s string) (Extents, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Lazy evaluation bounding metrics tracking cached elements scaling arrays statically.
	if h.tFace == nil {
		var src *text.FontSource

		if cached, ok := h.sources.Get(h.Font.Path); ok {
			src = cached
		} else {
			loaded, err := text.NewFontSourceFromFile(h.Font.Path)
			if err != nil {
				return Extents{}, err
			}
			h.sources.Set(h.Font.Path, loaded)
			src = loaded
		}

		// Scales raw requested layout boundaries adjusting for true point dimensions resolving calculating metrics.
		adjustedSize := h.Size * (h.DPI / 72.0)
		h.tFace = src.Face(adjustedSize)
	}

	w, ht := text.Measure(s, h.tFace)
	metrics := h.tFace.Metrics()

	return Extents{
		Width:      w,
		Height:     ht,
		Ascent:     metrics.Ascent,
		Descent:    metrics.Descent,
		LineHeight: metrics.LineHeight(),
	}, nil
}

// TextFace returns the mapped graphics explicit bounding context allowing external Ebiten / GG layers structural drawing commands globally mapping.
func (h *FaceHandle) TextFace() text.Face {
	h.mu.Lock()
	if h.tFace == nil {
		h.mu.Unlock()
		_, _ = h.MeasureExtents("")
		h.mu.Lock()
	}
	defer h.mu.Unlock()
	return h.tFace
}
