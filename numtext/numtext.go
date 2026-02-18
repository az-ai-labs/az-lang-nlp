// Package numtext converts between numbers and Azerbaijani text representations.
//
// The package provides conversion in both directions:
//
//   - Convert turns an integer into cardinal Azerbaijani text.
//   - ConvertOrdinal produces ordinal forms with vowel-harmony suffixes.
//   - ConvertFloat converts decimal number strings to text.
//   - Parse turns Azerbaijani number text back into an integer.
//
// ConvertFloat supports two reading modes: mathematical ("üç tam yüzdə on dörd")
// and digit-by-digit ("üç vergül bir dörd"), controlled by the Mode parameter.
//
// All functions are safe for concurrent use by multiple goroutines.
//
// Known limitations:
//
//   - Integer range is limited to ±10^18 (kvintilyon).
//   - Decimal conversion supports up to 18 fractional digits.
//   - Parse handles cardinal text only; ordinal input returns an error.
//   - Composed denominator words for decimals beyond 3 digits (D>3) are
//     non-standard in Azerbaijani and provided as a best-effort extension.
package numtext

import "fmt"

// Mode controls how decimal numbers are read aloud.
type Mode int

const (
	// MathMode reads decimals as fractions: "üç tam yüzdə on dörd" (3.14).
	MathMode Mode = iota

	// DigitMode reads fractional digits individually: "üç vergül bir dörd" (3.14).
	DigitMode
)

// Convert returns the Azerbaijani cardinal text for n.
// Zero returns "sıfır". Negative numbers are prefixed with "mənfi".
// Numbers with absolute value exceeding 10^18 return an empty string.
func Convert(n int64) string {
	return convert(n)
}

// ConvertOrdinal returns the Azerbaijani ordinal text for n.
// Applies vowel-harmony suffix (-ıncı/-inci/-uncu/-üncü) to the cardinal form.
// When the cardinal text ends in a vowel, the suffix loses its initial vowel
// (e.g. "iyirmi" → "iyirminci", not "iyirmiinci").
// Zero returns "sıfırıncı". Negative ordinals prefix "mənfi" to the ordinal
// of the absolute value.
func ConvertOrdinal(n int64) string {
	return convertOrdinal(n)
}

// ConvertFloat converts a decimal number string to Azerbaijani text.
// Accepts dot or comma as decimal separator. The mode parameter controls
// the reading style: MathMode produces "üç tam yüzdə on dörd",
// DigitMode produces "üç vergül bir dörd".
//
// Input without a decimal separator is treated as an integer and mode is ignored.
//
// Returns an empty string for invalid input (empty, non-numeric, multiple
// separators).
func ConvertFloat(s string, mode Mode) string {
	return convertFloat(s, mode)
}

// Parse converts Azerbaijani cardinal number text to an integer.
// Input is whitespace-normalized and case-insensitive.
// Accepts both canonical ("yüz") and explicit ("bir yüz") forms for
// hundreds and thousands.
//
// Returns an error for empty, unparseable, or out-of-range input.
func Parse(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("numtext: empty input")
	}
	return parse(s)
}
