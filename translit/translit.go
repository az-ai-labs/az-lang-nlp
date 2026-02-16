// Package translit converts Azerbaijani text between Latin and Cyrillic alphabets.
//
// The Azerbaijani language has used three scripts historically: Arabic (pre-1929),
// Latin (1929-1939 and post-1991), and Cyrillic (1939-1991). This package handles
// conversion between the modern Latin and Soviet-era Cyrillic scripts.
//
// All functions are safe for concurrent use by multiple goroutines.
//
// Known lossy conversions (Cyrillic → Latin):
//   - Soft sign (Ь/ь) and hard sign (Ъ/ъ) are silently removed (no Latin equivalent).
//   - Г/г disambiguation depends on context and may differ from the original author's intent.
//
// Characters not in the Azerbaijani alphabet (digits, punctuation, emoji, CJK,
// non-Azerbaijani Cyrillic) pass through unchanged.
package translit

import (
	"strings"
	"unicode/utf8"
)

// CyrillicToLatin converts Azerbaijani Cyrillic text to Latin script.
//
// Contextual rules apply to Г/г: if the input contains Ҝ/ҝ (indicating Soviet
// orthography with separate letters for G and Q), then Г always maps to Q.
// Otherwise, Г maps to G before front vowels (ә, е, и, ө, ү) and to Q before
// back vowels (а, о, у, ы), consonants, or end of string.
//
// Soft sign (Ь) and hard sign (Ъ) are removed — they have no Latin equivalent.
func CyrillicToLatin(s string) string {
	if s == "" {
		return ""
	}

	hasGje := containsGje(s)

	var b strings.Builder
	b.Grow(len(s))

	for i, r := range s {
		switch r {
		case 'Г', 'г':
			rest := s[i+utf8.RuneLen(r):]
			b.WriteRune(resolveG(r == 'Г', rest, hasGje))
		case 'Ь', 'ь', 'Ъ', 'ъ':
			// Silently removed.
		default:
			if lat, ok := cyrToLat[r]; ok {
				b.WriteRune(lat)
			} else {
				b.WriteRune(r)
			}
		}
	}

	return b.String()
}

// LatinToCyrillic converts Azerbaijani Latin text to Cyrillic script.
//
// Each of the 32 Azerbaijani Latin letters maps 1:1 to a Cyrillic rune.
// Q maps to Г, G maps to Ҝ. Characters not in the Azerbaijani Latin
// alphabet pass through unchanged.
func LatinToCyrillic(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		if cyr, ok := latToCyr[r]; ok {
			b.WriteRune(cyr)
		} else {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// TODO(Rioverde): implement Arabic script to Latin conversion if needed.
func ArabicToLatin(s string) string { return s }

// TODO(Rioverde): implement Latin to Arabic script conversion if needed.
func LatinToArabic(s string) string { return s }
