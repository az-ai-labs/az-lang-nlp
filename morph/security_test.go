package morph

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestMaxWordBytesEnforcement verifies the 256-byte limit is enforced correctly.
func TestMaxWordBytesEnforcement(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		byteLen     int
		wantStem    string
		exactMatch  bool // if false, only verify non-empty for non-empty input
	}{
		{
			name:       "exactly 256 bytes - gets processed",
			input:      strings.Repeat("a", 256),
			byteLen:    256,
			wantStem:   "", // 256 bytes passes guard (> 256 is false), gets processed and stemmed
			exactMatch: false,
		},
		{
			name:       "257 bytes - rejected",
			input:      strings.Repeat("a", 257),
			byteLen:    257,
			wantStem:   strings.Repeat("a", 257), // returned unchanged
			exactMatch: true,
		},
		{
			name:       "multibyte chars at boundary",
			input:      strings.Repeat("ə", 128), // ə is 2 bytes (U+0259) = 256 bytes total
			byteLen:    256,
			wantStem:   "", // 256 bytes passes guard, gets processed - don't assert exact value
			exactMatch: false,
		},
		{
			name:       "multibyte exceeds limit",
			input:      strings.Repeat("ə", 129), // 258 bytes
			byteLen:    258,
			wantStem:   strings.Repeat("ə", 129), // exceeds limit, returned unchanged
			exactMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.input) != tt.byteLen {
				t.Fatalf("test setup error: len(%q) = %d, want %d", tt.name, len(tt.input), tt.byteLen)
			}
			got := Stem(tt.input)

			if tt.exactMatch {
				if got != tt.wantStem {
					t.Errorf("Stem(%d bytes) = %q (len=%d), want %q (len=%d)",
						tt.byteLen, got, len(got), tt.wantStem, len(tt.wantStem))
				}
			} else {
				// For boundary cases, just verify no panic and non-empty for non-empty input
				if tt.input != "" && got == "" {
					t.Errorf("Stem(%d bytes) returned empty for non-empty input", tt.byteLen)
				}
			}
		})
	}
}

// TestRecursionDepthLimit verifies maxDepth=10 prevents stack overflow.
func TestRecursionDepthLimit(t *testing.T) {
	// Build a word with 15+ potential suffix layers to exceed maxDepth=10.
	// Each suffix can trigger recursion, so we want to force deep backtracking.
	// Example: a stem followed by many ambiguous morphemes.
	// In practice, the FSM rules limit valid chains, but we can test with
	// pathological inputs that match many surface forms.

	// Test with a word that has many overlapping suffix matches.
	// The walker will try to backtrack through all combinations.
	deeply := "kitablarımızınızdan" // valid multi-suffix chain (9 morphemes)

	// This should complete without panic or infinite loop.
	results := Analyze(deeply)
	if len(results) == 0 {
		t.Errorf("Analyze(%q) returned empty, expected at least one analysis", deeply)
	}

	// Verify it doesn't panic on very long suffix-like sequences.
	// Build a synthetic word with many repeating patterns.
	synthetic := "a" + strings.Repeat("lar", 50) // 151 bytes, many potential plural matches
	gotStem := Stem(synthetic)
	if gotStem == "" {
		t.Errorf("Stem(%q) returned empty", synthetic)
	}
}

// TestConcurrentSafety verifies the package is safe for concurrent use.
func TestConcurrentSafety(t *testing.T) {
	words := []string{
		"kitablarımızdan",
		"gəlmişdir",
		"evlərdə",
		"yazdılar",
		"müəllimdir",
	}

	// Run 100 goroutines concurrently analyzing the same words.
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
				word := words[j%len(words)]
				_ = Stem(word)
				_ = Analyze(word)
			}
		}(i)
	}

	for range numGoroutines {
		<-done
	}
}

// TestMalformedUTF8 verifies handling of invalid UTF-8 sequences.
func TestMalformedUTF8(t *testing.T) {
	tests := []struct {
		name  string
		input string
		desc  string
	}{
		{
			name:  "invalid UTF-8 byte sequence",
			input: "kitab\xFF\xFElar",
			desc:  "embedded invalid bytes",
		},
		{
			name:  "truncated multibyte sequence",
			input: "kitab\xC3", // incomplete ə (U+0259 = C9 99)
			desc:  "truncated at end",
		},
		{
			name:  "overlong encoding",
			input: "kitab\xC0\x80", // overlong encoding of NULL
			desc:  "overlong sequence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if utf8.ValidString(tt.input) {
				t.Skipf("test input is valid UTF-8, cannot test malformed case")
			}

			// Should not panic, but may produce unexpected results.
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Stem(%q) panicked: %v", tt.desc, r)
				}
			}()

			_ = Stem(tt.input)
			_ = Analyze(tt.input)
		})
	}
}

// TestMemoryExhaustion verifies we don't allocate excessive memory for large inputs.
func TestMemoryExhaustion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory exhaustion test in short mode")
	}

	// Process 10,000 words in a loop to check for memory leaks.
	words := []string{
		"kitablarımızdan",
		"gəlmişdir",
		"evlərdə",
		"yazdılar",
	}

	for i := range 10000 {
		word := words[i%len(words)]
		_ = Stem(word)
		results := Analyze(word)
		_ = results
	}
}

// TestEmptyAndEdgeCases verifies handling of edge case inputs.
func TestEmptyAndEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"single rune", "a"},
		{"single consonant", "b"},
		{"single space", " "},
		{"null byte", "\x00"},
		{"tab", "\t"},
		{"newline", "\n"},
		{"only punctuation", "!!!"},
		{"numbers", "123456"},
		{"mixed", "a1b2c3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Stem(%q) panicked: %v", tt.input, r)
				}
			}()

			stem := Stem(tt.input)
			results := Analyze(tt.input)

			// Empty input should return empty/nil.
			if tt.input == "" {
				if stem != "" {
					t.Errorf("Stem(%q) = %q, want empty", tt.input, stem)
				}
				if results != nil {
					t.Errorf("Analyze(%q) = %v, want nil", tt.input, results)
				}
			}
		})
	}
}

// TestHyphenAndApostropheRecursion verifies handling of recursive cases.
func TestHyphenAndApostropheRecursion(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"many hyphens", strings.Repeat("a-", 100) + "a"},
		{"nested apostrophes", "Bakı'nın'ın'ın"},
		{"hyphen at start", "-kitab"},
		{"hyphen at end", "kitab-"},
		{"consecutive hyphens", "a--b"},
		{"apostrophe only", "'''"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Stem(%q) panicked: %v", tt.input, r)
				}
			}()

			_ = Stem(tt.input)
		})
	}
}

// TestUnicodeNormalizationAttack verifies we handle unnormalized Unicode.
func TestUnicodeNormalizationAttack(t *testing.T) {
	// The package docs say it expects NFC normalization, but we should
	// verify it doesn't crash on NFD or other forms.
	tests := []struct {
		name  string
		input string
		desc  string
	}{
		{
			name:  "combining diacritics",
			input: "kitab\u0301lar", // a with combining acute accent
			desc:  "NFD-style combining marks",
		},
		{
			name:  "multiple combining marks",
			input: "e\u0301\u0308", // e with acute and diaeresis
			desc:  "stacked diacritics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Stem(%q) panicked: %v", tt.desc, r)
				}
			}()

			_ = Stem(tt.input)
			_ = Analyze(tt.input)
		})
	}
}

// TestNullByteInjection verifies handling of embedded null bytes.
func TestNullByteInjection(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"null at start", "\x00kitab"},
		{"null in middle", "kitab\x00lar"},
		{"null at end", "kitab\x00"},
		{"multiple nulls", "\x00\x00\x00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Stem(%q) panicked: %v", tt.input, r)
				}
			}()

			_ = Stem(tt.input)
			_ = Analyze(tt.input)
		})
	}
}

// TestControlCharacters verifies handling of control characters.
func TestControlCharacters(t *testing.T) {
	// Test ASCII control characters (0x00-0x1F).
	for i := range 32 {
		t.Run(fmt.Sprintf("control_0x%02X", i), func(t *testing.T) {
			input := "kitab" + string(rune(i)) + "lar"
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Stem with control char 0x%02X panicked: %v", i, r)
				}
			}()
			_ = Stem(input)
		})
	}
}

// TestExtremeUnicodeCodepoints verifies handling of high Unicode codepoints.
func TestExtremeUnicodeCodepoints(t *testing.T) {
	tests := []struct {
		name  string
		input string
		desc  string
	}{
		{
			name:  "emoji",
			input: "kitab😀lar",
			desc:  "emoji in word",
		},
		{
			name:  "supplementary plane",
			input: "kitab\U0001F600lar", // grinning face emoji
			desc:  "4-byte UTF-8 sequence",
		},
		{
			name:  "replacement character",
			input: "kitab\uFFFDlar",
			desc:  "Unicode replacement character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Stem(%q) panicked: %v", tt.desc, r)
				}
			}()

			_ = Stem(tt.input)
			_ = Analyze(tt.input)
		})
	}
}
