package fonts

// Resolver acts as the pure logical cascade engine intercepting queries and mapping exact ties vs aliases overrides.
type Resolver struct {
	registry    *Registry
	config      FallbackConfig
	fontCache   *FontCache   // Handles query logic caching.
	faceCache   *FaceCache   // Bridges targeted explicit size bindings externally mapping.
	sourceCache *SourceCache // Tracks heavy `gogpu/gg/text` instances across massive loaded files.
}

// NewResolver connects a targeted explicit fallback configuration string dictionary against a runtime indexing database.
func NewResolver(registry *Registry, config FallbackConfig) *Resolver {
	return &Resolver{
		registry:    registry,
		config:      config,
		fontCache:   newFontCache(),
		faceCache:   newFaceCache(),
		sourceCache: newSourceCache(),
	}
}

// LoadFace exposes completely parallel-safe resolutions returning generic mapping handles to external renderers.
func (r *Resolver) LoadFace(req FaceRequest) (*FaceHandle, error) {
	// 1. Thread-safe lock checked quickly bypassing expensive loops identically.
	if handle, ok := r.faceCache.Get(req); ok {
		if handle == nil {
			return nil, ErrFontNotFound
		}
		return handle, nil
	}

	// 2. translate upstream explicit metrics routing.
	query := req.toQuery()
	best := r.Resolve(query)

	if best == nil {
		// Strict bounds blocked elements resulting missing maps globally terminating.
		r.faceCache.Set(req, nil)
		return nil, ErrFontNotFound
	}

	handle := &FaceHandle{
		Font:    best,
		Size:    req.Size,
		DPI:     req.DPI,
		sources: r.sourceCache, // Passing generic source limits tracking over cached limits.
	}

	// Bounds generated identical metrics mapping globally.
	r.faceCache.Set(req, handle)

	return handle, nil
}

// Resolve analyzes the configuration array applying bounded logic rules evaluating arrays step logic preventing infinite recursive spirals.
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

	// Phase 1: Straight explicit native search logic
	best := r.registry.Match(q)

	// Perfect mapped exact string overrides generic bounds constraints returning instantly
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

	// Blocking logic returning out when native maps fail exactly.
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
		// General universal standard defaults arrays
		cascade = r.config.SansSerif
	}

	for _, fb := range cascade {
		if normalizeFamily(fb) == normTarget {
			continue // prevents cyclic local blocks testing the element already tested completely.
		}

		fbQuery := q
		fbQuery.Family = fb

		if match := r.registry.Match(fbQuery); match != nil {
			if normalizeFamily(match.Family) == normalizeFamily(fb) {
				return match
			}
		}
	}

	// Phase 4: Final Generic Fallback Catchall
	// Simply yields the highest generated physical mapping returning the physical pointer rather than a system nil panic.
	if best != nil {
		return best
	}

	return nil
}

// Subsystem category generic parsing
func isMonoSub(f string) bool {
	return f == "monospace" || f == "mono"
}
func isSerifSub(f string) bool {
	return f == "serif"
}
func isEmojiSub(f string) bool {
	return f == "emoji" || f == "symbol"
}
