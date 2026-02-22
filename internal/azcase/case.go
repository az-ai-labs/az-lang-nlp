// Package azcase provides Azerbaijani (Turkic) case conversion.
//
// Azerbaijani uses dotted/dotless I variants:
//   - I (U+0049) lowercases to ı (U+0131, dotless small i)
//   - İ (U+0130, dotted capital I) lowercases to i (U+0069)
//   - i (U+0069) uppercases to İ (U+0130, dotted capital I)
//   - ı (U+0131, dotless small i) uppercases to I (U+0049)
//
// All other runes use standard Unicode case mapping.
//
// All functions are safe for concurrent use.
package azcase

import (
	"strings"
	"unicode"
)

// Lower returns the Azerbaijani-aware lowercase form of r.
func Lower(r rune) rune {
	switch r {
	case 'I':
		return '\u0131' // I -> ı
	case '\u0130':
		return 'i' // İ -> i
	default:
		return unicode.ToLower(r)
	}
}

// Upper returns the Azerbaijani-aware uppercase form of r.
func Upper(r rune) rune {
	switch r {
	case 'i':
		return '\u0130' // i -> İ
	case '\u0131':
		return 'I' // ı -> I
	default:
		return unicode.ToUpper(r)
	}
}

// ToLower returns s with Azerbaijani-aware lowercasing applied to every rune.
func ToLower(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		b.WriteRune(Lower(r))
	}
	return b.String()
}
