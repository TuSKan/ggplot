package fonts

// Resolver maps font requests to loaded font faces, applying a CSS-like
// cascade: exact match → family alias → weight fallback → system default.
type Resolver struct {
	registry    *Registry
	config      FallbackConfig
	fontCache   *FontCache   // Caches resolved Font pointers by query.
	faceCache   *FaceCache   // Caches FaceHandle results by request parameters.
	sourceCache *SourceCache // Caches loaded text.FontSource instances to avoid re-parsing.
}

// NewResolver creates a Resolver backed by the given font registry and fallback configuration.
func NewResolver(registry *Registry, config FallbackConfig) *Resolver {
	return &Resolver{
		registry:    registry,
		config:      config,
		fontCache:   newFontCache(),
		faceCache:   newFaceCache(),
		sourceCache: newSourceCache(),
	}
}

// LoadFace returns a font.Face for the given family, size, weight, and style.
// Results are cached; concurrent calls for the same parameters share one Face.
func (r *Resolver) LoadFace(req FaceRequest) (*FaceHandle, error) {
	// Fast path: return cached result.
	if handle, ok := r.faceCache.Get(req); ok {
		if handle == nil {
			return nil, ErrFontNotFound
		}
		return handle, nil
	}

	// Build a query from the request and resolve via cascade.
	query := req.toQuery()
	best := r.Resolve(query)

	if best == nil {
		// Cache the miss to avoid repeated lookups.
		r.faceCache.Set(req, nil)
		return nil, ErrFontNotFound
	}

	handle := &FaceHandle{
		Font:    best,
		Size:    req.Size,
		DPI:     req.DPI,
		sources: r.sourceCache,
	}

	// Cache the successful result.
	r.faceCache.Set(req, handle)

	return handle, nil
}

// Resolve finds the best font for the given query, using the cached result if available.
func (r *Resolver) Resolve(q Query) *Font {
	if f, ok := r.fontCache.Get(q); ok {
		return f
	}

	best := r.resolveCascade(q)

	r.fontCache.Set(q, best)
	return best
}

func (r *Resolver) resolveCascade(q Query) *Font {
	normTarget := normalizeFamily(q.Family)

	// Phase 1: Direct registry search.
	best := r.registry.Match(q)

	// Exact family match — return immediately.
	if best != nil && q.Family != "" && normalizeFamily(best.Family) == normTarget {
		return best
	}

	// Phase 2: Explicit Aliases
	if aliases, ok := r.config.Aliases[normTarget]; ok && q.Family != "" {
		for _, alias := range aliases {
			aliasQuery := q
			aliasQuery.Family = alias
			if aliasMatch := r.registry.Match(aliasQuery); aliasMatch != nil {
				if normalizeFamily(aliasMatch.Family) == normalizeFamily(alias) {
					return aliasMatch
				}
			}
		}
	}

	// No fallback allowed — strict mode.
	if !q.AllowFallback {
		return nil
	}

	// Phase 3: Structural Fallback Cascade
	var cascade []string

	if q.PreferMonospace || isMonoSub(normTarget) {
		cascade = r.config.Monospace
	} else if isSerifSub(normTarget) {
		cascade = r.config.Serif
	} else if isEmojiSub(normTarget) {
		cascade = r.config.Emoji
	} else {
		// Default: sans-serif cascade.
		cascade = r.config.SansSerif
	}

	for _, fb := range cascade {
		if normalizeFamily(fb) == normTarget {
			continue // Skip already-tested family.
		}

		fbQuery := q
		fbQuery.Family = fb

		if match := r.registry.Match(fbQuery); match != nil {
			if normalizeFamily(match.Family) == normalizeFamily(fb) {
				return match
			}
		}
	}

	// Phase 4: Return the best match found in Phase 1 (may be a loose match).
	if best != nil {
		return best
	}

	return nil
}

// Category helpers for font classification.
func isMonoSub(f string) bool {
	return f == "monospace" || f == "mono"
}
func isSerifSub(f string) bool {
	return f == "serif"
}
func isEmojiSub(f string) bool {
	return f == "emoji" || f == "symbol"
}
