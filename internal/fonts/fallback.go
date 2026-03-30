package fonts

// FallbackConfig defines explicit string map overrides generating strict deterministic cascade trees.
type FallbackConfig struct {
	// Aliases maps specific query families to nearest approximations BEFORE cascading generic overrides.
	Aliases map[string][]string

	// Core platform agnostic fallback trees mapping general generic categories arrays.
	SansSerif []string
	Serif     []string
	Monospace []string
	Emoji     []string
}

// DefaultFallbackConfig returns sane universal fallbacks spanning Linux, Windows, and macOS mappings.
func DefaultFallbackConfig() FallbackConfig {
	return FallbackConfig{
		Aliases: map[string][]string{
			"helvetica":     {"arial", "liberation sans", "nimbus sans l"},
			"times":         {"times new roman", "liberation serif", "nimbus roman"},
			"courier":       {"courier new", "liberation mono", "nimbus mono ps"},
			"comic sans":    {"comic sans ms"},
			"segoe ui":      {"liberation sans"},
			"sf pro":        {"san francisco", "helvetica"},
			"menlo":         {"consolas", "dejavu sans mono"},
			"cascadia code": {"consolas", "courier new"},
		},
		SansSerif: []string{"arial", "ubuntu", "liberation sans", "dejavu sans", "roboto"},
		Serif:     []string{"times new roman", "liberation serif", "dejavu serif", "georgia"},
		Monospace: []string{"courier new", "consolas", "liberation mono", "dejavu sans mono", "menlo", "cascadia code"},
		Emoji:     []string{"apple color emoji", "segoe ui emoji", "noto color emoji", "joypixels"},
	}
}
