package colormap

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gogpu/gg"
)

// Parse converts a color literal to gg.RGBA. Accepted forms:
//
//   - Hex: "#RGB", "#RGBA", "#RRGGBB", "#RRGGBBAA" (delegates to gg.ParseHex;
//     leading '#' optional).
//   - CSS / X11 named colors: "red", "coral", "rebeccapurple", … (147 names).
//   - Matplotlib aliases: "tab:blue", "tab:orange", … through "tab:cyan".
//   - Functional rgb / rgba: "rgb(255,0,0)", "rgba(0,128,0,0.5)". Channels
//     accept 0–255 integers or "<n>%" (0–100).
//   - Functional hsl: "hsl(120,100%,50%)" — h in degrees [0,360), s and l
//     in percent.
//
// Empty strings return an error. Comparisons are case-insensitive for named
// colors and the leading function keyword.
func Parse(s string) (gg.RGBA, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return gg.RGBA{}, fmt.Errorf("empty color string: %w", ErrParseColor)
	}

	// Hex first — covers "#abc" / "abc123" / "#aabbccdd" without scanning
	// the named-color table. gg.ParseHex itself rejects bad strings.
	if isHexCandidate(raw) {
		c, err := gg.ParseHex(raw)
		if err == nil {
			return c, nil
		}
	}

	lower := strings.ToLower(raw)

	// Functional notation.
	if strings.HasPrefix(lower, "rgb(") || strings.HasPrefix(lower, "rgba(") {
		return parseRGBFunc(lower)
	}

	if strings.HasPrefix(lower, "hsl(") {
		return parseHSLFunc(lower)
	}

	// Matplotlib "tab:*" aliases.
	if after, ok := strings.CutPrefix(lower, "tab:"); ok {
		if c, ok := tabColors[after]; ok {
			return c, nil
		}

		return gg.RGBA{}, fmt.Errorf("colormap: unknown tab alias %q: %w", raw, ErrParseColor)
	}

	// CSS / X11 named colors.
	if c, ok := namedColors[lower]; ok {
		return c, nil
	}

	return gg.RGBA{}, fmt.Errorf("colormap: cannot parse color %q: %w", raw, ErrParseColor)
}

// MustParse is like Parse but panics on error.
func MustParse(s string) gg.RGBA {
	c, err := Parse(s)
	if err != nil {
		panic(err)
	}

	return c
}

// ParseRGB parses a color spec and returns its normalized [0,1] RGB components.
// On empty or invalid input, defR/defG/defB are returned unchanged.
func ParseRGB(spec string, defR, defG, defB float64) (r, g, b float64) {
	if spec == "" {
		return defR, defG, defB
	}

	c, err := Parse(spec)
	if err != nil {
		return defR, defG, defB
	}

	return c.R, c.G, c.B
}

// isHexCandidate reports whether s could plausibly be a hex literal — used
// to skip a futile gg.ParseHex call for obvious named colors.
func isHexCandidate(s string) bool {
	if len(s) == 0 {
		return false
	}

	if s[0] == '#' {
		return true
	}
	// Bare hex: 3, 4, 6, or 8 hex digits with no other characters.
	switch len(s) {
	case 3, 4, 6, 8:
		for i := range len(s) {
			c := s[i]

			isHex := ('0' <= c && c <= '9') ||
				('a' <= c && c <= 'f') ||
				('A' <= c && c <= 'F')
			if !isHex {
				return false
			}
		}

		return true
	}

	return false
}

// parseRGBFunc handles "rgb(r,g,b)" and "rgba(r,g,b,a)".
func parseRGBFunc(s string) (gg.RGBA, error) {
	open := strings.IndexByte(s, '(')

	end := strings.IndexByte(s, ')')
	if open < 0 || end < 0 || end <= open {
		return gg.RGBA{}, fmt.Errorf("colormap: malformed rgb literal %q: %w", s, ErrParseColor)
	}

	body := s[open+1 : end]

	parts := splitArgs(body)
	if len(parts) != 3 && len(parts) != 4 {
		return gg.RGBA{}, fmt.Errorf("colormap: rgb requires 3 or 4 components, got %d in %q: %w", len(parts), s, ErrParseColor)
	}

	r, err := parseChannelByte(parts[0])
	if err != nil {
		return gg.RGBA{}, err
	}

	g, err := parseChannelByte(parts[1])
	if err != nil {
		return gg.RGBA{}, err
	}

	b, err := parseChannelByte(parts[2])
	if err != nil {
		return gg.RGBA{}, err
	}

	a := 1.0
	if len(parts) == 4 {
		a, err = parseAlpha(parts[3])
		if err != nil {
			return gg.RGBA{}, err
		}
	}

	return gg.RGBA{R: r, G: g, B: b, A: a}, nil
}

// parseHSLFunc handles "hsl(h,s%,l%)" — h in degrees, s and l in percent.
func parseHSLFunc(s string) (gg.RGBA, error) {
	open := strings.IndexByte(s, '(')

	end := strings.IndexByte(s, ')')
	if open < 0 || end < 0 || end <= open {
		return gg.RGBA{}, fmt.Errorf("colormap: malformed hsl literal %q: %w", s, ErrParseColor)
	}

	body := s[open+1 : end]

	parts := splitArgs(body)
	if len(parts) != 3 {
		return gg.RGBA{}, fmt.Errorf("colormap: hsl requires 3 components, got %d in %q: %w", len(parts), s, ErrParseColor)
	}

	hRaw := strings.TrimSuffix(parts[0], "deg")

	h, err := strconv.ParseFloat(strings.TrimSpace(hRaw), 64)
	if err != nil {
		return gg.RGBA{}, fmt.Errorf("colormap: hsl hue %q: %w", parts[0], err)
	}

	sat, err := parsePercent(parts[1])
	if err != nil {
		return gg.RGBA{}, err
	}

	light, err := parsePercent(parts[2])
	if err != nil {
		return gg.RGBA{}, err
	}
	// Normalize hue to [0,360); gg.HSL accepts the angle in degrees.
	for h < 0 {
		h += 360
	}

	for h >= 360 {
		h -= 360
	}

	return gg.HSL(h, sat, light), nil
}

// splitArgs splits on commas (and whitespace as fallback) — handles both
// "0,0,0" and "0 0 0" CSS notations.
func splitArgs(body string) []string {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}

	if strings.Contains(body, ",") {
		parts := strings.Split(body, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}

		return parts
	}

	return strings.Fields(body)
}

// parseChannelByte parses an rgb channel: integer 0–255 or "<n>%" 0–100.
func parseChannelByte(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if before, ok := strings.CutSuffix(s, "%"); ok {
		v, err := strconv.ParseFloat(before, 64)
		if err != nil {
			return 0, fmt.Errorf("colormap: bad channel %q: %w", s, err)
		}

		return clamp01(v / 100.0), nil
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("colormap: bad channel %q: %w", s, err)
	}

	return clamp01(v / 255.0), nil
}

// parseAlpha parses an alpha component: 0–1 float or "<n>%".
func parseAlpha(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if before, ok := strings.CutSuffix(s, "%"); ok {
		v, err := strconv.ParseFloat(before, 64)
		if err != nil {
			return 0, fmt.Errorf("colormap: bad alpha %q: %w", s, err)
		}

		return clamp01(v / 100.0), nil
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("colormap: bad alpha %q: %w", s, err)
	}

	return clamp01(v), nil
}

// parsePercent parses an "<n>%" or bare 0–1 fraction.
func parsePercent(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if before, ok := strings.CutSuffix(s, "%"); ok {
		v, err := strconv.ParseFloat(before, 64)
		if err != nil {
			return 0, fmt.Errorf("colormap: bad percent %q: %w", s, err)
		}

		return clamp01(v / 100.0), nil
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("colormap: bad percent %q: %w", s, err)
	}

	return clamp01(v), nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}

	if v > 1 {
		return 1
	}

	return v
}
