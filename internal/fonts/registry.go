package fonts

// Registry provides a centralized locator index storing mapped physical font struct structs tracking states without cascade logic mappings.
type Registry struct {
	fonts []Font
}

// NewRegistry initializes an OS-level font store generating parsing iterators fetching system standard definitions out.
func NewRegistry() (*Registry, error) {
	discovered, _ := DiscoverFonts(DefaultDirs())

	r := &Registry{
		fonts: discovered,
	}

	return r, nil
}

// Match executes explicit scoring maps comparing geometric elements generating string weights without attempting mapping alias arrays loops locally.
func (r *Registry) Match(q Query) *Font {
	bestScore := -1 << 31
	var best *Font

	for i := range r.fonts {
		f := &r.fonts[i]
		s := score(*f, q)

		if s > bestScore {
			bestScore = s
			best = f
		} else if s == bestScore && best != nil {
			// stable tie-breaking logic returning identical alphabetical deterministic strings uniformly
			if f.FullName != best.FullName {
				if f.FullName < best.FullName {
					bestScore = s
					best = f
				}
			} else {
				if f.Path < best.Path {
					bestScore = s
					best = f
				} else if f.Path == best.Path {
					if f.Index < best.Index {
						bestScore = s
						best = f
					}
				}
			}
		}
	}
	return best
}

// Fonts fetches explicit lists parsed internally loaded saving allocations queries.
func (r *Registry) Fonts() []Font {
	return r.fonts
}
