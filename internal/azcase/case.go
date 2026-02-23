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
	"unicode/utf8"
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

// ToUpper returns s with Azerbaijani-aware uppercasing applied to every rune.
func ToUpper(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		sb.WriteRune(Upper(r))
	}
	return sb.String()
}

// UpperFirst returns s with its first rune Azerbaijani-uppercased.
func UpperFirst(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || size == 0 {
		return s
	}
	var sb strings.Builder
	sb.Grow(len(s))
	sb.WriteRune(Upper(r))
	sb.WriteString(s[size:])
	return sb.String()
}

// IsTitleCase reports whether s has its first rune uppercase and is not
// entirely uppercase (which would be an acronym).
func IsTitleCase(s string) bool {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || !unicode.IsUpper(r) {
		return false
	}
	rest := s[size:]
	if rest == "" {
		return false
	}
	for _, c := range rest {
		if unicode.IsLetter(c) && !unicode.IsUpper(c) {
			return true
		}
	}
	return false
}

// IsAllUpper reports whether every letter in s is uppercase.
func IsAllUpper(s string) bool {
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return hasLetter
}

// ApplyCase transfers the case pattern of original onto corrected.
// Three modes: all-upper, title-case (first rune upper), lowercase.
func ApplyCase(original, corrected string) string {
	if original == "" || corrected == "" {
		return corrected
	}
	if IsAllUpper(original) {
		return ToUpper(corrected)
	}
	firstRune, _ := utf8.DecodeRuneInString(original)
	if unicode.IsUpper(firstRune) {
		return UpperFirst(corrected)
	}
	return corrected
}

// ContainsDigit reports whether s contains any digit rune.
func ContainsDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
