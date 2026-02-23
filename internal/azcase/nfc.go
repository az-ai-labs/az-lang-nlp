package azcase

import "strings"

// nfcReplacer composes known Azerbaijani NFD pairs in a single pass.
var nfcReplacer = strings.NewReplacer(
	// Lowercase
	"o\u0308", "\u00f6", // o + diaeresis -> ö
	"u\u0308", "\u00fc", // u + diaeresis -> ü
	"c\u0327", "\u00e7", // c + cedilla   -> ç
	"s\u0327", "\u015f", // s + cedilla   -> ş
	"g\u0306", "\u011f", // g + breve     -> ğ
	// Uppercase
	"O\u0308", "\u00d6", // O + diaeresis -> Ö
	"U\u0308", "\u00dc", // U + diaeresis -> Ü
	"C\u0327", "\u00c7", // C + cedilla   -> Ç
	"S\u0327", "\u015e", // S + cedilla   -> Ş
	"G\u0306", "\u011e", // G + breve     -> Ğ
	"I\u0307", "\u0130", // I + dot above -> İ
)

// ComposeNFC replaces known NFD decomposed sequences for the 6 Azerbaijani
// letters with diacritics: ö, ü, ç, ş, ğ, İ.
// This is NOT full Unicode NFC — only Azerbaijani-specific pairs.
// For full NFC, preprocess with golang.org/x/text/unicode/norm externally.
func ComposeNFC(s string) string {
	// Fast path: scan for combining marks U+0306, U+0307, U+0308, U+0327.
	hasCombiner := false
	for _, r := range s {
		if r == 0x0306 || r == 0x0307 || r == 0x0308 || r == 0x0327 {
			hasCombiner = true
			break
		}
	}
	if !hasCombiner {
		return s
	}

	return nfcReplacer.Replace(s)
}
