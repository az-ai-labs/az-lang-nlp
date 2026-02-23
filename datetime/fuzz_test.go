package datetime

import (
	"strings"
	"testing"
	"time"
)

var fuzzRef = time.Date(2026, 2, 20, 10, 30, 0, 0, time.UTC)

func FuzzExtract(f *testing.F) {
	// Seed corpus covering all input categories.
	seeds := []string{
		// Natural text
		"5 mart 2026",
		"martda",
		"yanvardan",
		"martın 15-i",
		"mart ayının 3-ü",
		"5-ci mart 2026",
		"5-nci mart",
		// Numeric
		"2026-03-05",
		"05.03.2026",
		"05/03/2026",
		"14:30",
		"09:05:22",
		// Relative
		"bu gün",
		"bugün",
		"sabah",
		"dünən",
		"birigün",
		"srağagün",
		"keçən həftə",
		"gələn ay",
		"bu il",
		"3 gün əvvəl",
		"2 həftə sonra",
		// Weekday
		"bazar ertəsi",
		"çərşənbə axşamı",
		"cümə",
		"keçən cümə",
		// Time-of-day
		"saat 3",
		"axşam saat 7",
		"səhər saat 7",
		// Combined
		"5 mart 2026 14:30",
		// Edge cases
		"",
		"abc xyz",
		"32 mart 2026",
		"25:99",
		"\xff\xfe",
		"\x00sabah\x00",
		"\xC3",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		results := Extract(s, fuzzRef)

		for _, r := range results {
			// Offset invariant: matched text must equal the slice.
			if r.Start < 0 || r.End > len(s) || r.Start > r.End {
				t.Errorf("invalid offsets: Start=%d End=%d len=%d", r.Start, r.End, len(s))
				continue
			}
			if s[r.Start:r.End] != r.Text {
				t.Errorf("offset invariant: s[%d:%d]=%q != Text=%q", r.Start, r.End, s[r.Start:r.End], r.Text)
			}

			// Type must be valid.
			if r.Type < TypeDate || r.Type > TypeDuration {
				t.Errorf("invalid type: %d", r.Type)
			}

			// Time must be UTC.
			if r.Time.Location() != time.UTC {
				t.Errorf("non-UTC time: %v", r.Time.Location())
			}
		}

		// Parse must not panic either.
		_, _ = Parse(s, fuzzRef)
	})
}

// TestOversizedInput verifies that inputs exceeding maxInputBytes are rejected.
func TestOversizedInput(t *testing.T) {
	huge := strings.Repeat("a", maxInputBytes+1)
	got := Extract(huge, fuzzRef)
	if got != nil {
		t.Errorf("want nil for oversized input, got %d results", len(got))
	}

	_, err := Parse(huge, fuzzRef)
	if err == nil {
		t.Error("Parse: want error for oversized input, got nil")
	}
}

// TestExactlyMaxInput verifies that inputs at exactly maxInputBytes are processed.
func TestExactlyMaxInput(t *testing.T) {
	date := "2026-03-05"
	padding := strings.Repeat(" ", maxInputBytes-len(date))
	input := date + padding

	if len(input) != maxInputBytes {
		t.Fatalf("test setup: len=%d, want %d", len(input), maxInputBytes)
	}

	got := Extract(input, fuzzRef)
	if len(got) != 1 || got[0].Type != TypeDate {
		t.Errorf("want 1 TypeDate result for max-size input, got %v", got)
	}
}

// TestReDoSResistance verifies regex patterns complete quickly on adversarial input.
func TestReDoSResistance(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "repeated digits with dots",
			input: strings.Repeat("12.34.", 5000),
		},
		{
			name:  "repeated colons",
			input: strings.Repeat("12:34:", 5000),
		},
		{
			name:  "repeated dashes",
			input: strings.Repeat("2026-01-", 5000),
		},
		{
			name:  "repeated month words",
			input: strings.Repeat("mart ", 5000),
		},
		{
			name:  "long digit sequence",
			input: strings.Repeat("1234567890", 5000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			_ = Extract(tt.input, fuzzRef)
			elapsed := time.Since(start)

			const maxDuration = 2 * time.Second
			if elapsed > maxDuration {
				t.Errorf("took %v, exceeds %v limit", elapsed, maxDuration)
			}
		})
	}
}

// TestConcurrentSafety verifies the package is safe for concurrent use.
func TestConcurrentSafety(t *testing.T) {
	inputs := []string{
		"5 mart 2026",
		"2026-03-05",
		"14:30",
		"sabah",
		"keçən həftə",
	}

	const numGoroutines = 100
	done := make(chan bool, numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("goroutine %d panicked: %v", id, r)
				}
				done <- true
			}()

			for j := range 100 {
				input := inputs[j%len(inputs)]
				_ = Extract(input, fuzzRef)
			}
		}(i)
	}

	for range numGoroutines {
		<-done
	}
}

// TestMalformedUTF8 verifies handling of invalid UTF-8 sequences.
func TestMalformedUTF8(t *testing.T) {
	inputs := []string{
		"\xFF\xFE mart 2026",
		"5 \xC0\x80 mart",
		"sabah\xFF",
		"\xC3", // truncated multibyte
	}

	for _, in := range inputs {
		t.Run("", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Extract(%q) panicked: %v", in, r)
				}
			}()
			_ = Extract(in, fuzzRef)
		})
	}
}

// TestNullByteInjection verifies handling of embedded null bytes.
func TestNullByteInjection(t *testing.T) {
	inputs := []string{
		"\x00sabah",
		"sabah\x00",
		"5\x00mart\x002026",
	}

	for _, in := range inputs {
		t.Run("", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Extract(%q) panicked: %v", in, r)
				}
			}()
			_ = Extract(in, fuzzRef)
		})
	}
}
