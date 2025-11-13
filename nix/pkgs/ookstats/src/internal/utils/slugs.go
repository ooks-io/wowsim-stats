package utils

import (
	"strings"
	"unicode"
)

// SafeSlugName converts an arbitrary player name to a safe lowercase filename without path separators.
// It preserves Unicode letters/numbers (including diacritics), matching frontend expectations
// that URLs may contain non-ASCII characters. Only path separators are removed and spaces -> '-'.
func SafeSlugName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	// Replace path separators explicitly
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")

	// Allow unicode letters/digits and '-', '_' ; replace spaces with '-'
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == ' ' {
			out = append(out, '-')
			continue
		}
		if r == '-' || r == '_' {
			out = append(out, r)
			continue
		}
		// Keep unicode letters and digits; drop other punctuation/symbols
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, r)
			continue
		}
		// else: drop
	}

	// Collapse multiple dashes
	cleaned := collapseDashes(out)

	if len(cleaned) == 0 {
		return "player"
	}

	// Trim leading/trailing '-'
	return strings.Trim(string(cleaned), "-")
}

// collapseDashes removes consecutive dashes, keeping only one
func collapseDashes(runes []rune) []rune {
	if len(runes) == 0 {
		return runes
	}

	cleaned := make([]rune, 0, len(runes))
	prevDash := false

	for _, r := range runes {
		if r == '-' {
			if !prevDash {
				cleaned = append(cleaned, r)
			}
			prevDash = true
		} else {
			cleaned = append(cleaned, r)
			prevDash = false
		}
	}

	return cleaned
}
