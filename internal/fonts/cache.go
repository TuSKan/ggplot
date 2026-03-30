package fonts

import (
	"github.com/gogpu/gg/text"
	"sync"
)

// FontCache provides thread-safe heuristic memoization for resolved system fonts.
type FontCache struct {
	mu      sync.RWMutex
	entries map[Query]*Font
}

// newFontCache allocates an empty memory lookup map protected.
func newFontCache() *FontCache {
	return &FontCache{
		entries: make(map[Query]*Font),
	}
}

// Get retrieves a resolved font from the cache map mapping across RLock read barriers.
func (c *FontCache) Get(q Query) (*Font, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	f, ok := c.entries[q]
	return f, ok
}

// Set stores a newly evaluated pointer locking globally executing exact write limits.
func (c *FontCache) Set(q Query, f *Font) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[q] = f
}

// FaceCache specifically memoizes concrete renderer facing bounds ensuring concurrent drawing maps over exact DPI/Size bounds.
type FaceCache struct {
	mu      sync.RWMutex
	entries map[FaceRequest]*FaceHandle
}

func newFaceCache() *FaceCache {
	return &FaceCache{
		entries: make(map[FaceRequest]*FaceHandle),
	}
}

// Get handles sizing array lookups executing native thread mapping.
func (c *FaceCache) Get(r FaceRequest) (*FaceHandle, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	f, ok := c.entries[r]
	return f, ok
}

// Set locks mapping array metrics executing size bounds.
func (c *FaceCache) Set(r FaceRequest, f *FaceHandle) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[r] = f
}

// SourceCache specifically memoizes parsed physical bytes mapping resolving massive TTF files.
type SourceCache struct {
	mu      sync.RWMutex
	entries map[string]*text.FontSource
}

func newSourceCache() *SourceCache {
	return &SourceCache{
		entries: make(map[string]*text.FontSource),
	}
}

// Get handles string lookup loads ensuring physical limits match threaded arrays.
func (c *SourceCache) Get(path string) (*text.FontSource, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	f, ok := c.entries[path]
	return f, ok
}

// Set commits caching heavy physical file blocks executing directly over map arrays.
func (c *SourceCache) Set(path string, f *text.FontSource) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[path] = f
}
