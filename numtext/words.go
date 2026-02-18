// Word tables for Azerbaijani number-to-text conversion.
package numtext

const (
	maxAbs  int64 = 1_000_000_000_000_000_000
	hundred int64 = 100

	wordNegative = "mənfi"
	wordHundred  = "yüz"
	wordExact    = "tam"
	wordComma    = "vergül"
	wordZero     = "sıfır"
)

var ones = [10]string{
	"sıfır",
	"bir",
	"iki",
	"üç",
	"dörd",
	"beş",
	"altı",
	"yeddi",
	"səkkiz",
	"doqquz",
}

// tens is indexed by tens digit (1–9); index 0 is unused.
var tens = [10]string{
	"",
	"on",
	"iyirmi",
	"otuz",
	"qırx",
	"əlli",
	"altmış",
	"yetmiş",
	"səksən",
	"doxsan",
}

type magnitude struct {
	value int64
	word  string
}

// magnitudes lists named powers of ten from largest to smallest.
// yüz (100) is handled separately within group conversion and is not listed here.
var magnitudes = []magnitude{
	{value: 1_000_000_000_000_000_000, word: "kvintilyon"},
	{value: 1_000_000_000_000_000, word: "kvadrilyon"},
	{value: 1_000_000_000_000, word: "trilyon"},
	{value: 1_000_000_000, word: "milyard"},
	{value: 1_000_000, word: "milyon"},
	{value: 1_000, word: "min"},
}

// denominators maps the number of fractional digits (1–3) to the Azerbaijani
// denominator word used in math-mode decimal reading.
// Index 0 is unused. Denominators beyond 3 digits are composed programmatically.
var denominators = [4]string{"", "onda", "yüzdə", "mində"}

// powersOf10 maps exponent (0–18) to the corresponding int64 value.
// Used by powerOf10Text to avoid computing 10^exp at runtime.
var powersOf10 = [19]int64{
	1,
	10,
	100,
	1_000,
	10_000,
	100_000,
	1_000_000,
	10_000_000,
	100_000_000,
	1_000_000_000,
	10_000_000_000,
	100_000_000_000,
	1_000_000_000_000,
	10_000_000_000_000,
	100_000_000_000_000,
	1_000_000_000_000_000,
	10_000_000_000_000_000,
	100_000_000_000_000_000,
	1_000_000_000_000_000_000,
}
