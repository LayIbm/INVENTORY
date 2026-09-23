// Package normalize provides string normalization utilities for
// human-readable fields stored in the device inventory.
package normalize

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Prepositions and articles that stay lowercase when they appear
// in the middle of a name (Spanish and English).
var lowerWords = map[string]bool{
	// Spanish
	"de": true, "del": true, "la": true, "las": true,
	"los": true, "el": true, "y": true, "e": true,
	"en": true, "da": true, "das": true, "dos": true,
	// English
	"of": true, "the": true, "and": true, "van": true,
	"von": true, "bin": true, "bint": true,
}

// Name converts a name string to Title Case following Spanish naming
// conventions:
//   - Leading/trailing whitespace stripped
//   - Multiple internal spaces collapsed to one
//   - Every word capitalized EXCEPT prepositions/articles that appear
//     in the middle of the name (e.g. "de", "del", "la")
//   - The FIRST word is always capitalized regardless
//   - Handles all-caps and all-lowercase input
//
// Examples:
//
//	"KEVIN ALEJANDRO FLORES" → "Kevin Alejandro Flores"
//	"ABRAHAM DE LA CRUZ"     → "Abraham de la Cruz"
//	"ana maria del rio"      → "Ana Maria del Rio"
func Name(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	// Normalize Unicode: decompose then recompose (handles NFD edge cases)
	// We do NOT strip accents — we want to keep á, é, í, ó, ú, ñ, ü, etc.
	t := transform.Chain(norm.NFC)
	normalized, _, _ := transform.String(t, s)
	_ = runes.Remove(runes.In(unicode.Mn)) // keep reference, not used for names

	// Collapse internal whitespace
	words := strings.Fields(normalized)
	for i, w := range words {
		lower := strings.ToLower(w)
		// Always capitalize the first word; lowercase prepositions elsewhere
		if i > 0 && lowerWords[lower] {
			words[i] = lower
		} else {
			words[i] = capitalizeFirst(w)
		}
	}
	return strings.Join(words, " ")
}

// capitalizeFirst uppercases the first rune of a word and lowercases the rest,
// preserving Unicode letters correctly (e.g. "ÁNGEL" → "Ángel").
func capitalizeFirst(w string) string {
	if w == "" {
		return w
	}
	runes := []rune(strings.ToLower(w))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// Email returns the email lowercased and trimmed.
func Email(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
