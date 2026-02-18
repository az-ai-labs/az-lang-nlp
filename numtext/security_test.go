package numtext

import (
	"strings"
	"sync"
	"testing"
)

// TestConcurrentSafety verifies all functions are safe for concurrent use.
func TestConcurrentSafety(t *testing.T) {
	var wg sync.WaitGroup

	const goroutines = 100

	for range goroutines {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic in concurrent call: %v", r)
				}
			}()

			Convert(123)
			Convert(-42)
			Convert(0)
			ConvertOrdinal(5)
			ConvertOrdinal(20)
			ConvertFloat("3.14", MathMode)
			ConvertFloat("3.14", DigitMode)
			Parse("yüz iyirmi üç")
			Parse("mənfi beş")
		})
	}

	wg.Wait()
}

// TestConvertLargeNumbers verifies Convert handles edge-case large numbers.
func TestConvertLargeNumbers(t *testing.T) {
	tests := []struct {
		name  string
		input int64
	}{
		{"max valid", 1_000_000_000_000_000_000},
		{"just over max", 1_000_000_000_000_000_001},
		{"max int64", 9223372036854775807},
		{"min int64", -9223372036854775808},
		{"negative max valid", -1_000_000_000_000_000_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Convert(%d) panicked: %v", tt.input, r)
				}
			}()
			_ = Convert(tt.input)
		})
	}
}

// TestParseMalformed verifies Parse handles malformed input gracefully.
func TestParseMalformed(t *testing.T) {
	malformed := []string{
		"",
		" ",
		"   ",
		"\t\n",
		"\xff\xfe",
		string([]byte{0x00}),
		strings.Repeat("bir ", 1000),
		"bir iki üç dörd",  // not a valid number
		"mənfi",            // negative with no number
		"mənfi mənfi bir",  // double negative
		"yüz yüz",          // weird repetition
	}

	for _, input := range malformed {
		t.Run("", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Parse(%q) panicked: %v", input, r)
				}
			}()
			_, _ = Parse(input)
		})
	}
}

// TestConvertFloatMalformed verifies ConvertFloat handles malformed input.
func TestConvertFloatMalformed(t *testing.T) {
	malformed := []string{
		"",
		"abc",
		"3.14.15",
		".",
		"..",
		"3.",
		"--3.14",
		"++3.14",
		"3.abc",
		"abc.3",
		"\xff\xfe",
		string([]byte{0x00}),
		strings.Repeat("9", 100) + ".1",
	}

	for _, input := range malformed {
		t.Run("", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ConvertFloat(%q) panicked: %v", input, r)
				}
			}()
			got := ConvertFloat(input, MathMode)
			if got != "" {
				// Some of these might produce valid output — that's OK.
				// We only verify no panic.
				_ = got
			}
		})
	}
}
