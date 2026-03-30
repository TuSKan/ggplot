package fonts

import "strings"

func score(f Font, q Query) int {
	score := 0

	// family match dominates absolutely
	if f.Family == normalizeFamily(q.Family) {
		score += 10000
	}

	// monospace preference constraint checking
	if q.PreferMonospace && f.IsMonospace {
		score += 500
	} else if q.PreferMonospace && !f.IsMonospace {
		score -= 500
	} else if !q.PreferMonospace && f.IsMonospace {
		// strongly penalize generic matching attempts grabbing Mono fonts unexpectedly
		score -= 1000
	}

	// exact style match constraint
	if f.Style == q.Style {
		score += 200
	} else {
		score -= 100 // style mismatch is less desirable than an unchecked state.
	}

	// weight proportional proximity
	// The closer the actual 100-900 index bound, the less deduction applied.
	score -= abs(int(f.Weight) - int(q.Weight))

	return score
}

// Simple integer math constraint helper.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func normalizeFamily(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	return s
}

// Exhaustive lookup map expanding upon strings like "Black" or "Demi-bold".
func parseWeight(sub string) Weight {
	s := strings.ToLower(sub)

	switch {
	case strings.Contains(s, "thin"):
		return WeightThin
	case strings.Contains(s, "extra light") || strings.Contains(s, "extralight"):
		return WeightExtraLight
	case strings.Contains(s, "light"):
		return WeightLight
	case strings.Contains(s, "medium"):
		return WeightMedium
	case strings.Contains(s, "semi bold") || strings.Contains(s, "semibold") || strings.Contains(s, "demi"):
		return WeightSemiBold
	case strings.Contains(s, "extra bold") || strings.Contains(s, "extrabold") || strings.Contains(s, "heavy"):
		return WeightExtraBold
	case strings.Contains(s, "black") || strings.Contains(s, "super"):
		return WeightBlack
	case strings.Contains(s, "bold"):
		return WeightBold
	default:
		return WeightNormal
	}
}

func parseStyle(sub string) Style {
	s := strings.ToLower(sub)

	switch {
	case strings.Contains(s, "italic"):
		return StyleItalic
	case strings.Contains(s, "oblique"):
		return StyleOblique
	default:
		return StyleNormal
	}
}
