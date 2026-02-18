// Unexported conversion functions for Azerbaijani number-to-text conversion.
package numtext

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	growConvert = 64  // estimated bytes for a full cardinal conversion
	growFloat   = 128 // estimated bytes for a decimal conversion
	maxDenomFD  = 3   // max fractional digits with a named denominator
)

// convert converts an int64 to Azerbaijani cardinal text.
// Returns "" if abs(n) exceeds maxAbs.
func convert(n int64) string {
	if n > maxAbs || n < -maxAbs {
		return ""
	}
	if n == 0 {
		return wordZero
	}

	negative := n < 0
	if negative {
		n = -n
	}

	var b strings.Builder
	b.Grow(growConvert)

	if negative {
		b.WriteString(wordNegative)
	}

	for _, mag := range magnitudes {
		count := n / mag.value
		if count > 0 {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			// "bir min" -> "min" (omit "bir" before "min" only)
			if mag.value == 1_000 && count == 1 {
				b.WriteString(mag.word)
			} else {
				writeGroup(&b, count)
				b.WriteByte(' ')
				b.WriteString(mag.word)
			}
			n %= mag.value
		}
	}

	if n > 0 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		writeGroup(&b, n)
	}

	return b.String()
}

// writeGroup writes a number in [0, 999] as Azerbaijani text into b.
// Callers must ensure n > 0.
func writeGroup(b *strings.Builder, n int64) {
	h := n / hundred
	if h == 1 {
		b.WriteString(wordHundred)
	} else if h > 1 {
		b.WriteString(ones[h])
		b.WriteByte(' ')
		b.WriteString(wordHundred)
	}

	r := n % hundred
	t := r / 10
	o := r % 10

	if t > 0 {
		if h > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(tens[t])
	}

	if o > 0 {
		if h > 0 || t > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(ones[o])
	}
}

// convertOrdinal converts an int64 to Azerbaijani ordinal text.
// Returns "" if abs(n) exceeds maxAbs.
func convertOrdinal(n int64) string {
	if n > maxAbs || n < -maxAbs {
		return ""
	}

	negative := n < 0
	absN := n
	if negative {
		absN = -n
	}

	cardinal := convert(absN)

	lv := lastVowel(cardinal)
	if lv == 0 {
		return ""
	}

	var suffix string
	lastRune, _ := utf8.DecodeLastRuneInString(cardinal)
	if isVowel(lastRune) {
		suffix = ordinalShortSuffix(lv)
	} else {
		suffix = ordinalFullSuffix(lv)
	}

	result := cardinal + suffix
	if negative {
		return wordNegative + " " + result
	}
	return result
}

// convertFloat converts a decimal number string to Azerbaijani text using
// the given Mode (MathMode or DigitMode).
func convertFloat(s string, mode Mode) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	negative := false
	switch s[0] {
	case '-':
		negative = true
		s = s[1:]
	case '+':
		s = s[1:]
	}

	sepIdx := strings.IndexAny(s, ".,")

	if sepIdx == -1 {
		// No decimal separator; treat as plain integer.
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return ""
		}
		if negative {
			val = -val
		}
		return convert(val)
	}

	wholePart := s[:sepIdx]
	fracPart := s[sepIdx+1:]

	if (wholePart != "" && !allDigits(wholePart)) || !allDigits(fracPart) || fracPart == "" {
		return ""
	}

	if wholePart == "" {
		wholePart = "0"
	}
	wholeVal, err := strconv.ParseInt(wholePart, 10, 64)
	if err != nil {
		return ""
	}

	// Suppress "mənfi" prefix for negative zero (e.g. "-0.0").
	if negative && wholeVal == 0 && allZeros(fracPart) {
		negative = false
	}

	wholeText := convert(wholeVal)
	if wholeText == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(growFloat)

	if negative {
		b.WriteString(wordNegative)
		b.WriteByte(' ')
	}
	b.WriteString(wholeText)

	switch mode {
	case MathMode:
		fracDigits := len(fracPart)

		// Parse fractional part as integer (leading zeros are significant for
		// the denominator but not for the numerator value).
		numeratorVal, err := strconv.ParseInt(fracPart, 10, 64)
		if err != nil {
			return ""
		}
		numeratorText := convert(numeratorVal)
		if numeratorText == "" {
			return ""
		}

		var denomWord string
		if fracDigits <= maxDenomFD {
			denomWord = denominators[fracDigits]
		} else {
			// Compose denominator for fracDigits > 3: convert(10^fracDigits) + locative suffix.
			denomBase := powerOf10Text(fracDigits)
			if denomBase == "" {
				return ""
			}
			denomWord = denomBase + locativeSuffix(denomBase)
		}

		b.WriteByte(' ')
		b.WriteString(wordExact)
		b.WriteByte(' ')
		b.WriteString(denomWord)
		b.WriteByte(' ')
		b.WriteString(numeratorText)

	case DigitMode:
		b.WriteByte(' ')
		b.WriteString(wordComma)
		for _, ch := range fracPart {
			d := int(ch - '0')
			b.WriteByte(' ')
			b.WriteString(ones[d])
		}
	}

	return b.String()
}

// powerOf10Text returns the Azerbaijani text for 10^exp.
// Used for composing denominators beyond 3 fractional digits.
func powerOf10Text(exp int) string {
	if exp < 0 || exp >= len(powersOf10) {
		return ""
	}
	return convert(powersOf10[exp])
}

// lastVowel scans s backwards and returns the last rune that is an Azerbaijani vowel.
// Returns 0 if no vowel is found.
func lastVowel(s string) rune {
	for s != "" {
		r, size := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-size]
		if isVowel(r) {
			return r
		}
	}
	return 0
}

// isVowel reports whether r is an Azerbaijani vowel.
func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'ə', 'ı', 'i', 'o', 'ö', 'u', 'ü':
		return true
	}
	return false
}

// ordinalFullSuffix returns the full ordinal suffix for a cardinal ending in a consonant.
// The suffix is selected by vowel harmony based on the last vowel v.
func ordinalFullSuffix(v rune) string {
	switch v {
	case 'a', 'ı':
		return "ıncı"
	case 'e', 'ə', 'i':
		return "inci"
	case 'o', 'u':
		return "uncu"
	case 'ö', 'ü':
		return "üncü"
	}
	return ""
}

// ordinalShortSuffix returns the short ordinal suffix for a cardinal ending in a vowel.
// The initial vowel of the suffix is dropped to avoid a vowel clash.
func ordinalShortSuffix(v rune) string {
	switch v {
	case 'a', 'ı':
		return "ncı"
	case 'e', 'ə', 'i':
		return "nci"
	case 'o', 'u':
		return "ncu"
	case 'ö', 'ü':
		return "ncü"
	}
	return ""
}

// locativeSuffix returns the Azerbaijani locative case suffix ("da" or "də")
// based on vowel harmony of the last vowel in s.
// Back vowels (a, ı, o, u) -> "da"; front vowels (e, ə, i, ö, ü) -> "də".
func locativeSuffix(s string) string {
	lv := lastVowel(s)
	switch lv {
	case 'a', 'ı', 'o', 'u':
		return "da"
	default:
		return "də"
	}
}

// allDigits reports whether s consists entirely of ASCII digit characters.
// An empty string returns false.
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// allZeros reports whether s consists entirely of '0' characters.
func allZeros(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '0' {
			return false
		}
	}
	return true
}
