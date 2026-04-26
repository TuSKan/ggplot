package fonts

// Registry indexes system fonts discovered by scanning standard OS font directories.
// Each entry records the font's file path, family name, weight, and style.
type Registry struct {
	fonts []Font
}

// NewRegistry discovers and indexes all fonts in the standard OS font directories.
// On Linux this scans /usr/share/fonts; on macOS /Library/Fonts; on Windows C:\Windows\Fonts.
func NewRegistry() (*Registry, error) {
	discovered, _ := DiscoverFonts(DefaultDirs())

	r := &Registry{
		fonts: discovered,
	}

	return r, nil
}

// Match finds the best font in the registry for the given family, weight, and style,
// using a scoring heuristic that prefers exact matches over partial ones.
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
