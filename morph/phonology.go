package morph

import (
	"unicode/utf8"

	"github.com/az-ai-labs/az-lang-nlp/internal/azcase"
)

// backVowels contains Azerbaijani back vowels (both cases).
var backVowels = map[rune]bool{
	'a': true, 'A': true,
	'\u0131': true, 'I': true, // ı (U+0131), I (U+0049)
	'o': true, 'O': true,
	'u': true, 'U': true,
}

// frontVowels contains Azerbaijani front vowels (both cases).
var frontVowels = map[rune]bool{
	'e': true, 'E': true,
	'\u0259': true, '\u018F': true, // ə (U+0259), Ə (U+018F)
	'i': true, '\u0130': true, // i (U+0069), İ (U+0130)
	'\u00F6': true, '\u00D6': true, // ö (U+00F6), Ö (U+00D6)
	'\u00FC': true, '\u00DC': true, // ü (U+00FC), Ü (U+00DC)
}

// voicelessCons contains Azerbaijani voiceless consonants (lowercase only,
// input is lowercased before consonant checks).
var voicelessCons = map[rune]bool{
	'p':      true,
	'\u00E7': true, // ç (U+00E7)
	't':      true,
	'k':      true,
	'q':      true,
	'f':      true,
	's':      true,
	'\u015F': true, // ş (U+015F)
	'x':      true,
	'h':      true,
}

// isVowel reports whether r is an Azerbaijani vowel (any case).
func isVowel(r rune) bool {
	return backVowels[r] || frontVowels[r]
}

// isBackVowel reports whether r is an Azerbaijani back vowel.
func isBackVowel(r rune) bool {
	return backVowels[r]
}

// isVoiceless reports whether r is a voiceless consonant.
// Expects lowercase input.
func isVoiceless(r rune) bool {
	return voicelessCons[r]
}

// lastVowel returns the last vowel rune in s, or 0 if none found.
// Iterates right-to-left using utf8.DecodeLastRuneInString.
func lastVowel(s string) rune {
	for i := len(s); i > 0; {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		if isVowel(r) {
			return r
		}
		i -= size
	}
	return 0
}

// isValidStem reports whether s can be a valid Azerbaijani stem.
// A valid stem has at least 2 runes and contains at least one vowel.
func isValidStem(s string) bool {
	runes := 0
	hasVowel := false
	for _, r := range s {
		runes++
		if isVowel(r) {
			hasVowel = true
		}
	}
	return runes >= 2 && hasVowel
}

// matchesBackFront reports whether the suffix vowel agrees with the
// stem's last vowel under back/front harmony.
// A back last-vowel requires the suffix vowel to be back; front requires front.
func matchesBackFront(stemLastVowel, suffixVowel rune) bool {
	if stemLastVowel == 0 {
		return true // no vowel in stem, accept any
	}
	stemBack := isBackVowel(azcase.Lower(stemLastVowel))
	suffBack := isBackVowel(azcase.Lower(suffixVowel))
	return stemBack == suffBack
}

// matchesFourWay reports whether the suffix vowel agrees with the
// stem's last vowel under four-way harmony (ı/i/u/ü).
// Back unrounded (a, ı) -> ı; back rounded (o, u) -> u;
// front unrounded (e, ə, i) -> i; front rounded (ö, ü) -> ü.
func matchesFourWay(stemLastVowel, suffixVowel rune) bool {
	if stemLastVowel == 0 {
		return true
	}
	expected := fourWayTarget(azcase.Lower(stemLastVowel))
	return azcase.Lower(suffixVowel) == expected
}

// fourWayTarget returns the expected suffix vowel for four-way harmony
// given the stem's last vowel (lowercase).
func fourWayTarget(v rune) rune {
	switch v {
	case 'a', '\u0131': // back unrounded
		return '\u0131' // ı
	case 'o', 'u': // back rounded
		return 'u'
	case 'e', '\u0259', 'i': // front unrounded (e, ə, i)
		return 'i'
	case '\u00F6', '\u00FC': // front rounded (ö, ü)
		return '\u00FC' // ü
	default:
		return 'i' // safe fallback
	}
}
