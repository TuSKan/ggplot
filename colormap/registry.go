package colormap

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Category groups colormaps along matplotlib's standard taxonomy. Used by
// Cmap.Category and NamesByCategory.
type Category int

const (
	// Sequential covers smooth, monotonic-lightness colormaps suitable for
	// continuous, ordered data without a natural midpoint (Blues, Reds, ...).
	Sequential Category = iota
	// PerceptuallyUniform colormaps preserve the perception of equal data
	// differences across the range (viridis, plasma, inferno, magma, cividis).
	PerceptuallyUniform
	// Diverging colormaps emphasize departure from a meaningful midpoint
	// (RdBu, Spectral, ...).
	Diverging
	// Cyclic colormaps wrap (twilight, hsv).
	Cyclic
	// Qualitative covers discrete categorical palettes (tab10, Set1, ...).
	Qualitative
	// Miscellaneous holds everything that doesn't fit cleanly elsewhere
	// (turbo, terrain, etc.).
	Miscellaneous
)

func (c Category) String() string {
	switch c {
	case Sequential:
		return "sequential"
	case PerceptuallyUniform:
		return "perceptually_uniform"
	case Diverging:
		return "diverging"
	case Cyclic:
		return "cyclic"
	case Qualitative:
		return "qualitative"
	case Miscellaneous:
		return "miscellaneous"
	default:
		return fmt.Sprintf("category(%d)", int(c))
	}
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Cmap)
)

// Register adds a Cmap to the global registry under its Name. Registration is
// idempotent for the exact same Cmap value, but registering a different Cmap
// under an already-taken name returns an error.
//
// Names are looked up case-insensitively; the canonical form is lowercase.
func Register(c Cmap) error {
	if c == nil {
		return fmt.Errorf("cannot register nil Cmap: %w", ErrParseColor)
	}

	name := normalizeName(c.Name())
	if name == "" {
		return fmt.Errorf("cannot register Cmap with empty name: %w", ErrParseColor)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if existing, ok := registry[name]; ok {
		if existing == c {
			return nil
		}

		return fmt.Errorf("colormap: name %q already registered: %w", name, ErrParseColor)
	}

	registry[name] = c

	return nil
}

// MustRegister registers c and panics on error. Intended for init-time use.
func MustRegister(c Cmap) {
	if err := Register(c); err != nil {
		panic(err)
	}
}

// Resolve looks up a Cmap by name (case-insensitive). Names with the suffix
// "_r" return the reversed form of the base colormap, matching matplotlib's
// convention. Returns an error for unknown names.
func Resolve(name string) (Cmap, error) {
	key := normalizeName(name)
	if key == "" {
		return nil, fmt.Errorf("empty cmap name: %w", ErrParseColor)
	}

	reversed := false
	if strings.HasSuffix(key, "_r") {
		reversed = true
		key = strings.TrimSuffix(key, "_r")
	}

	registryMu.RLock()

	c, ok := registry[key]

	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("colormap: unknown cmap %q: %w", name, ErrParseColor)
	}

	if reversed {
		return c.Reversed(), nil
	}

	return c, nil
}

// MustResolve is like Resolve but panics on error.
func MustResolve(name string) Cmap {
	c, err := Resolve(name)
	if err != nil {
		panic(err)
	}

	return c
}

// Names returns all registered cmap names sorted alphabetically. The "_r"
// reversed forms are not enumerated — they are derivable from each base name.
func Names() []string {
	registryMu.RLock()

	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}

	registryMu.RUnlock()
	sort.Strings(out)

	return out
}

// NamesByCategory returns the registered cmap names whose Category matches
// the argument, sorted alphabetically.
func NamesByCategory(cat Category) []string {
	registryMu.RLock()

	out := make([]string, 0, len(registry)/4)
	for k, c := range registry {
		if c.Category() == cat {
			out = append(out, k)
		}
	}

	registryMu.RUnlock()
	sort.Strings(out)

	return out
}

// normalizeName lower-cases the name and trims whitespace. Used for
// case-insensitive registry keys.
func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
