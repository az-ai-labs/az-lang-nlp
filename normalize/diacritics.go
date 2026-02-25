package normalize

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/az-ai-labs/az-lang-nlp/azcase"
	"github.com/az-ai-labs/az-lang-nlp/morph"
)

// maxSubstitutablePositions caps the variant generation to avoid
// combinatorial explosion. 2^10 = 1024 candidates max.
const maxSubstitutablePositions = 10

// maxWordBytes is the maximum byte length for a word to attempt
// diacritic restoration on. Matches the morph package limit.
const maxWordBytes = 256

// asciiToDiacritic maps lowercase ASCII characters to their possible
// Azerbaijani diacritic equivalents. Applied AFTER Turkic-aware lowercasing.
//
// The 'i' -> 'ı' mapping is intentionally excluded:
// after azcase.Lower, 'i' means confirmed dotted-i (from lowercase 'i' or uppercase 'İ'),
// and 'ı' means confirmed dotless-i (from uppercase 'I').
// Both are already correct after Turkic lowering.
var asciiToDiacritic = [128]rune{
	'e': '\u0259', // ə
	'o': '\u00f6', // ö
	'u': '\u00fc', // ü
	'g': '\u011f', // ğ
	'c': '\u00e7', // ç
	's': '\u015f', // ş
}

// hasDiacriticAlt reports whether the rune has a diacritic alternative.
func hasDiacriticAlt(r rune) bool {
	return r < 128 && asciiToDiacritic[r] != 0
}

// containsDiacritics reports whether s contains any Azerbaijani diacritical characters.
func containsDiacritics(s string) bool {
	for _, r := range s {
		switch r {
		case '\u0259', '\u00f6', '\u00fc', '\u011f', '\u00e7', '\u015f', // lower: ə ö ü ğ ç ş
			'\u018f', '\u00d6', '\u00dc', '\u011e', '\u00c7', '\u015e': // upper: Ə Ö Ü Ğ Ç Ş
			return true
		}
	}
	return false
}

// toLowerRunes returns the runes of s with Azerbaijani-aware lowercasing.
// Combines lowering and rune conversion in a single pass to avoid
// an intermediate string allocation.
func toLowerRunes(s string) []rune {
	runes := make([]rune, 0, utf8.RuneCountInString(s))
	for _, r := range s {
		runes = append(runes, azcase.Lower(r))
	}
	return runes
}

// restoreWord attempts to restore diacritics on a single word.
// Returns the original word unchanged if:
//   - the word is empty or too long
//   - it already contains diacritical characters
//   - it has no substitutable characters
//   - it is already a known dictionary stem
//   - it has too many substitutable positions
//   - zero or multiple dictionary matches (ambiguous/unknown)
func restoreWord(word string) string {
	if word == "" || len(word) > maxWordBytes {
		return word
	}

	if containsDiacritics(word) {
		return word
	}

	// Lowercase and convert to runes in a single pass.
	runes := toLowerRunes(word)
	var positions []int
	for i, r := range runes {
		if hasDiacriticAlt(r) {
			positions = append(positions, i)
		}
	}

	if len(positions) == 0 {
		return word
	}

	// If the ASCII form is already a known stem, do not modify it.
	// This prevents changing valid words like "ac" (hungry) to "aç" (open).
	lowered := string(runes)
	if morph.IsKnownStem(lowered) {
		return word
	}

	if len(positions) > maxSubstitutablePositions {
		return word
	}

	// Generate variants lazily and check against dictionary.
	// Short-circuit on second match (ambiguous).
	totalVariants := 1 << len(positions)
	matchCount := 0
	matchMask := 0

	candidate := make([]rune, len(runes))
	for mask := 1; mask < totalVariants; mask++ {
		copy(candidate, runes)
		for bit, pos := range positions {
			if mask&(1<<bit) != 0 {
				candidate[pos] = asciiToDiacritic[runes[pos]]
			}
		}

		if morph.IsKnownStem(string(candidate)) {
			matchCount++
			if matchCount == 1 {
				matchMask = mask
			} else {
				return word
			}
		}
	}

	if matchCount != 1 {
		return word
	}

	// Reconstruct the unique match from the saved bitmask.
	for bit, pos := range positions {
		if matchMask&(1<<bit) != 0 {
			runes[pos] = asciiToDiacritic[runes[pos]]
		}
	}
	return restoreCase(word, runes)
}

// restoreCase applies the case pattern from the original word to the
// restored runes. Original uppercase positions become uppercase in the output.
func restoreCase(original string, restored []rune) string {
	if utf8.RuneCountInString(original) != len(restored) {
		return original
	}

	var b strings.Builder
	b.Grow(len(original) + len(restored))
	i := 0
	for _, origR := range original {
		if unicode.IsUpper(origR) {
			b.WriteRune(azcase.Upper(restored[i]))
		} else {
			b.WriteRune(restored[i])
		}
		i++
	}
	return b.String()
}
