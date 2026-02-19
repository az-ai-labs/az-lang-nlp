package ner

import (
	"strings"
	"testing"
	"time"
)

// TestOversizedInput verifies that inputs exceeding maxInputBytes are rejected.
func TestOversizedInput(t *testing.T) {
	huge := strings.Repeat("a", maxInputBytes+1)
	got := Recognize(huge)
	if got != nil {
		t.Errorf("want nil for oversized input, got %d entities", len(got))
	}
}

// TestExactlyMaxInput verifies that inputs at exactly maxInputBytes are processed.
func TestExactlyMaxInput(t *testing.T) {
	// Build input that is exactly maxInputBytes with a phone at the start.
	phone := "+994501234567"
	padding := strings.Repeat(" ", maxInputBytes-len(phone))
	input := phone + padding

	if len(input) != maxInputBytes {
		t.Fatalf("test setup: len=%d, want %d", len(input), maxInputBytes)
	}

	got := Recognize(input)
	if len(got) != 1 || got[0].Type != Phone {
		t.Errorf("want 1 Phone entity for max-size input, got %v", got)
	}
}

// TestReDoSResistance verifies regex patterns complete quickly on adversarial input.
func TestReDoSResistance(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "repeated at signs",
			input: strings.Repeat("a@", 5000),
		},
		{
			name:  "repeated plus signs in phone-like",
			input: strings.Repeat("+994", 5000),
		},
		{
			name:  "repeated AZ prefix",
			input: strings.Repeat("AZ12", 5000),
		},
		{
			name:  "nested dots for email",
			input: strings.Repeat("a.", 5000) + "@" + strings.Repeat("b.", 5000) + "com",
		},
		{
			name:  "many almost-FINs",
			input: strings.Repeat("ABCDEFG ", 5000),
		},
		{
			name:  "long digit sequence",
			input: strings.Repeat("1234567890", 5000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			_ = Recognize(tt.input)
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
		"FIN: 5ARPXK2",
		"+994501234567",
		"info@example.com",
		"AZ21NABZ00000000137010001944",
		"10-AA-123",
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
				_ = Recognize(input)
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
		"FIN: \xFF\xFE12345",
		"+994\xC0\x80501234567",
		"user@\xFFdomain.com",
		"\xC3", // truncated multibyte
	}

	for _, in := range inputs {
		t.Run("", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Recognize(%q) panicked: %v", in, r)
				}
			}()
			_ = Recognize(in)
		})
	}
}

// TestNullByteInjection verifies handling of embedded null bytes.
func TestNullByteInjection(t *testing.T) {
	inputs := []string{
		"\x00+994501234567",
		"+994\x00501234567",
		"info@\x00example.com",
	}

	for _, in := range inputs {
		t.Run("", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Recognize(%q) panicked: %v", in, r)
				}
			}()
			_ = Recognize(in)
		})
	}
}
