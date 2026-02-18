// Text-to-number parsing for Azerbaijani cardinal text.
package numtext

import (
	"fmt"
	"strings"
)

// wordValues maps each Azerbaijani cardinal word to its numeric value.
// Built at package level to avoid repeated allocation on every parse call.
var wordValues = map[string]int64{
	"sıfır":      0,
	"bir":        1,
	"iki":        2,
	"üç":         3,
	"dörd":       4,
	"beş":        5,
	"altı":       6,
	"yeddi":      7,
	"səkkiz":     8,
	"doqquz":     9,
	"on":         10,
	"iyirmi":     20,
	"otuz":       30,
	"qırx":       40,
	"əlli":       50,
	"altmış":     60,
	"yetmiş":     70,
	"səksən":     80,
	"doxsan":     90,
	"yüz":        100,
	"min":        1_000,
	"milyon":     1_000_000,
	"milyard":    1_000_000_000,
	"trilyon":    1_000_000_000_000,
	"kvadrilyon": 1_000_000_000_000_000,
	"kvintilyon": 1_000_000_000_000_000_000,
}

// parse converts Azerbaijani cardinal number text to int64.
//
// Limitation: strings.ToLower does not correctly fold Azerbaijani-specific
// uppercase letters. Specifically, İ (U+0130, capital I with dot above)
// lowercases to "i̇" (i + U+0307 combining dot) rather than plain "i".
// Likewise, I (U+0049) lowercases to "i" in Go, but Azerbaijani convention
// maps it to "ı". Since all entries in wordValues are already lowercase
// Azerbaijani, callers are expected to supply lowercase input; mixed-case
// input may silently fail to match on those specific letters.
func parse(s string) (int64, error) {
	// Normalize whitespace and case.
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	tokens := strings.Fields(s) // splits on any run of whitespace

	if len(tokens) == 0 {
		return 0, fmt.Errorf("numtext: empty input")
	}

	// Detect optional negative prefix.
	sign := int64(1)
	if tokens[0] == wordNegative {
		sign = -1
		tokens = tokens[1:]
		if len(tokens) == 0 {
			return 0, fmt.Errorf("numtext: empty input after %q", wordNegative)
		}
	}

	// Handle lone "sıfır" before entering the general loop.
	if len(tokens) == 1 && tokens[0] == wordZero {
		return 0, nil
	}

	var (
		current int64 // sum of fully resolved magnitude groups
		group   int64 // 0–999 accumulator for the group under construction
	)

	for _, tok := range tokens {
		val, ok := wordValues[tok]
		if !ok {
			return 0, fmt.Errorf("numtext: unknown word %q", tok)
		}

		if val == 0 {
			// "sıfır" makes no sense inside a compound number.
			return 0, fmt.Errorf("numtext: unexpected sıfır in compound")
		}

		if val < hundred {
			// Ones or tens digit — accumulate into the current group.
			group += val
		} else if val == hundred {
			// "yüz" multiplies whatever precedes it within the group.
			// If nothing precedes it, treat as implicit "bir yüz" (100).
			if group == 0 {
				group = 1
			}
			group *= hundred
		} else {
			// Magnitude word (min, milyon, milyard, …).
			// If nothing precedes it, treat as implicit "bir min" etc.
			if group == 0 {
				group = 1
			}
			// Overflow-checked multiplication and accumulation.
			if group > maxAbs/val {
				return 0, fmt.Errorf("numtext: out of range")
			}
			product := group * val
			if current > maxAbs-product {
				return 0, fmt.Errorf("numtext: out of range")
			}
			current += product
			group = 0
		}
	}

	result := current + group
	result *= sign

	// Guard against inputs that exceed the representable range.
	// maxAbs equals 10^18, which is also math.MaxInt64 rounded down.
	abs := result
	if abs < 0 {
		abs = -abs
	}
	if abs > maxAbs {
		return 0, fmt.Errorf("numtext: out of range")
	}

	return result, nil
}
