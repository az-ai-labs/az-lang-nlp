package spell

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// TestIsCorrect
// ---------------------------------------------------------------------------

func TestIsCorrect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "empty string",
			input: "",
			want:  true,
		},
		{
			name:  "known dictionary word",
			input: "kitab",
			want:  true,
		},
		{
			name:  "known high-frequency word",
			input: "və",
			want:  true,
		},
		{
			name:  "morphologically valid inflected form",
			input: "kitablar",
			want:  true,
		},
		{
			name:  "normalizable ASCII-degraded form",
			input: "gozel",
			want:  true,
		},
		{
			name:  "genuinely misspelled word",
			input: "ketab",
			want:  false,
		},
		{
			name:  "word shorter than minWordRunes - single ASCII",
			input: "a",
			want:  true,
		},
		{
			name:  "word shorter than minWordRunes - single rune",
			input: "o",
			want:  true,
		},
		{
			name:  "word exceeding maxWordBytes",
			input: strings.Repeat("a", maxWordBytes+1),
			want:  true,
		},
		{
			name:  "hyphenated valid word",
			input: "sosial-iqtisadi",
			want:  true,
		},
		{
			name:  "apostrophe suffix",
			input: "Bakı'nın",
			want:  true,
		},
		{
			name:  "totally unknown word",
			input: "xyzabc",
			want:  false,
		},
		{
			name:  "correct word with diacritics",
			input: "gözəl",
			want:  true,
		},
		{
			name:  "longer inflected form",
			input: "kitablarımızdan",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsCorrect(tt.input)
			if got != tt.want {
				t.Errorf("IsCorrect(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSuggest
// ---------------------------------------------------------------------------

func TestSuggest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		maxDist     int
		wantNil     bool
		wantMaxDist int // if non-zero, first suggestion must have Distance <= wantMaxDist
	}{
		{
			name:    "empty string",
			input:   "",
			maxDist: 2,
			wantNil: true,
		},
		{
			name:    "correct word returns nil",
			input:   "kitab",
			maxDist: 2,
			wantNil: true,
		},
		{
			name:        "misspelled word has suggestions",
			input:       "ketab",
			maxDist:     2,
			wantNil:     false,
			wantMaxDist: 2,
		},
		{
			name:    "correct word with diacritics returns nil",
			input:   "gözəl",
			maxDist: 2,
			wantNil: true,
		},
		{
			name:    "normalizable word returns nil",
			input:   "gozel",
			maxDist: 2,
			wantNil: true,
		},
		{
			name:    "unknown gibberish returns nil",
			input:   "xyzabc123",
			maxDist: 2,
			wantNil: true,
		},
		{
			name:        "maxDist clamped to maxEditDistance",
			input:       "ketab",
			maxDist:     5,
			wantNil:     false,
			wantMaxDist: maxEditDistance,
		},
		{
			name:    "correct inflected form returns nil",
			input:   "kitablar",
			maxDist: 2,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Suggest(tt.input, tt.maxDist)

			if tt.wantNil {
				if got != nil {
					t.Errorf("Suggest(%q, %d) = %v, want nil", tt.input, tt.maxDist, got)
				}
				return
			}

			if len(got) == 0 {
				t.Errorf("Suggest(%q, %d) = nil, want non-nil suggestions", tt.input, tt.maxDist)
				return
			}

			if tt.wantMaxDist > 0 && got[0].Distance > tt.wantMaxDist {
				t.Errorf("Suggest(%q, %d)[0].Distance = %d, want <= %d",
					tt.input, tt.maxDist, got[0].Distance, tt.wantMaxDist)
			}
		})
	}
}

// TestSuggestFirstSuggestion verifies the first suggestion for "ketab" is "kitab".
func TestSuggestFirstSuggestion(t *testing.T) {
	t.Parallel()
	got := Suggest("ketab", 2)
	if len(got) == 0 {
		t.Fatal("Suggest(\"ketab\", 2) returned no suggestions")
	}
	if got[0].Term != "kitab" {
		t.Errorf("Suggest(\"ketab\", 2)[0].Term = %q, want %q", got[0].Term, "kitab")
	}
}

// TestSuggestSortOrder verifies suggestions are sorted by distance ascending,
// then frequency descending.
func TestSuggestSortOrder(t *testing.T) {
	t.Parallel()
	got := Suggest("ketab", 2)
	if len(got) < 2 {
		t.Skip("not enough suggestions to test sort order")
	}
	for i := 1; i < len(got); i++ {
		a, b := got[i-1], got[i]
		if a.Distance > b.Distance {
			t.Errorf("suggestions not sorted by distance: got[%d].Distance=%d > got[%d].Distance=%d",
				i-1, a.Distance, i, b.Distance)
		}
		if a.Distance == b.Distance && a.Frequency < b.Frequency {
			t.Errorf("same-distance suggestions not sorted by frequency desc: got[%d].Frequency=%d < got[%d].Frequency=%d",
				i-1, a.Frequency, i, b.Frequency)
		}
	}
}

// ---------------------------------------------------------------------------
// TestCorrectWord
// ---------------------------------------------------------------------------

func TestCorrectWord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantExact  string // if non-empty, exact match expected
		wantChange bool   // if true, result must differ from input
	}{
		{
			name:      "empty string",
			input:     "",
			wantExact: "",
		},
		{
			name:      "correct word unchanged",
			input:     "kitab",
			wantExact: "kitab",
		},
		{
			name:       "misspelled word corrected",
			input:      "ketab",
			wantExact:  "kitab",
			wantChange: true,
		},
		{
			name:      "all-uppercase case preserved",
			input:     "KETAB",
			wantExact: "KİTAB",
		},
		{
			name:      "title-case preserved",
			input:     "Ketab",
			wantExact: "Kitab",
		},
		{
			name:      "no suggestion returns original",
			input:     "xyzabc123",
			wantExact: "xyzabc123",
		},
		{
			name:      "word exceeding maxWordBytes returned unchanged",
			input:     strings.Repeat("a", maxWordBytes+1),
			wantExact: strings.Repeat("a", maxWordBytes+1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CorrectWord(tt.input)

			if tt.wantExact != "" || tt.input == "" {
				if got != tt.wantExact {
					t.Errorf("CorrectWord(%q) = %q, want %q", tt.input, got, tt.wantExact)
				}
				return
			}

			if tt.wantChange && got == tt.input {
				t.Errorf("CorrectWord(%q) = %q (unchanged), want a correction", tt.input, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestCorrect
// ---------------------------------------------------------------------------

func TestCorrect(t *testing.T) {
	t.Parallel()

	oversized := strings.Repeat("a", maxInputBytes+1)

	tests := []struct {
		name      string
		input     string
		wantExact string
	}{
		{
			name:      "empty string",
			input:     "",
			wantExact: "",
		},
		{
			name:      "already correct text unchanged",
			input:     "bu kitab",
			wantExact: "bu kitab",
		},
		{
			name:      "oversized input returned unchanged",
			input:     oversized,
			wantExact: oversized,
		},
		{
			name:      "numbers preserved",
			input:     "123 ketab",
			wantExact: "123 kitab",
		},
		{
			name:      "misspelled word corrected",
			input:     "Bu ketab gozeldir",
			wantExact: "Bu kitab gözəldir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Correct(tt.input)
			if got != tt.wantExact {
				t.Errorf("Correct(%q) = %q, want %q", tt.input, got, tt.wantExact)
			}
		})
	}
}

// TestCorrectPreservesSpacesAndPunctuation verifies that spaces and punctuation
// are preserved in output.
func TestCorrectPreservesSpacesAndPunctuation(t *testing.T) {
	t.Parallel()
	// "yaxsi" may get corrected to something, but spaces and commas must survive.
	input := "kitab , yaxsi ."
	got := Correct(input)

	// Verify spaces and punctuation are present.
	for _, ch := range []string{",", ".", " "} {
		if !strings.Contains(got, ch) {
			t.Errorf("Correct(%q) = %q, missing %q in output", input, got, ch)
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkIsCorrect(b *testing.B) {
	for b.Loop() {
		IsCorrect("kitab")
	}
}

func BenchmarkIsCorrectMiss(b *testing.B) {
	for b.Loop() {
		IsCorrect("ketab")
	}
}

func BenchmarkSuggest(b *testing.B) {
	for b.Loop() {
		Suggest("ketab", 2)
	}
}

func BenchmarkCorrect(b *testing.B) {
	text := "Bu ketab cox gozeldir və biz onu oxuyuruq"
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		Correct(text)
	}
}

// ---------------------------------------------------------------------------
// Example Functions
// ---------------------------------------------------------------------------

func ExampleIsCorrect() {
	fmt.Println(IsCorrect("kitab"))
	fmt.Println(IsCorrect("ketab"))
	// Output:
	// true
	// false
}

func ExampleSuggest() {
	suggestions := Suggest("ketab", 2)
	if len(suggestions) > 0 {
		fmt.Println(suggestions[0].Term)
	}
	// Output:
	// kitab
}

func ExampleCorrectWord() {
	fmt.Println(CorrectWord("ketab"))
	// Output:
	// kitab
}

func ExampleCorrect() {
	fmt.Println(Correct("Bu ketab gozeldir"))
	// Output:
	// Bu kitab gözəldir
}

// ---------------------------------------------------------------------------
// Security Tests
// ---------------------------------------------------------------------------

// TestMaxWordBytesEnforcement verifies the 256-byte limit is enforced correctly.
func TestMaxWordBytesEnforcement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		byteLen  int
		wantTrue bool // IsCorrect must return true (too long = not a spell error)
	}{
		{
			name:     "exactly 256 bytes - gets processed",
			input:    strings.Repeat("a", maxWordBytes),
			byteLen:  maxWordBytes,
			wantTrue: false, // 256 bytes is within the guard (> maxWordBytes is false), so it's processed
		},
		{
			name:     "257 bytes - exceeds limit, returned as correct",
			input:    strings.Repeat("a", maxWordBytes+1),
			byteLen:  maxWordBytes + 1,
			wantTrue: true,
		},
		{
			name:     "multibyte exactly 256 bytes",
			input:    strings.Repeat("ə", 128), // ə is 2 bytes = 256 total
			byteLen:  256,
			wantTrue: false, // within limit, processed normally
		},
		{
			name:     "multibyte 258 bytes - exceeds limit",
			input:    strings.Repeat("ə", 129), // 258 bytes
			byteLen:  258,
			wantTrue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.input) != tt.byteLen {
				t.Fatalf("test setup error: len(%q) = %d, want %d", tt.name, len(tt.input), tt.byteLen)
			}

			got := IsCorrect(tt.input)
			if tt.wantTrue && !got {
				t.Errorf("IsCorrect(%d bytes) = false, want true (oversize should be accepted)", tt.byteLen)
			}

			// Also verify CorrectWord returns unchanged for oversize.
			if tt.byteLen > maxWordBytes {
				corrected := CorrectWord(tt.input)
				if corrected != tt.input {
					t.Errorf("CorrectWord(%d bytes) changed the input, want unchanged", tt.byteLen)
				}
			}
		})
	}
}

// TestConcurrentSafety verifies the package is safe for concurrent use.
func TestConcurrentSafety(t *testing.T) {
	words := []string{
		"kitab",
		"ketab",
		"gözəl",
		"gozel",
		"kitablar",
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
				word := words[j%len(words)]
				_ = IsCorrect(word)
				_ = Suggest(word, 2)
				_ = CorrectWord(word)
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
			input: "kitab\xC3",
			desc:  "truncated at end",
		},
		{
			name:  "overlong encoding",
			input: "kitab\xC0\x80",
			desc:  "overlong sequence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if utf8.ValidString(tt.input) {
				t.Skipf("test input is valid UTF-8, cannot test malformed case")
			}

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panicked on %s: %v", tt.desc, r)
				}
			}()

			_ = IsCorrect(tt.input)
			_ = Suggest(tt.input, 2)
			_ = CorrectWord(tt.input)
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
					t.Errorf("panicked on %q: %v", tt.input, r)
				}
			}()

			_ = IsCorrect(tt.input)
			_ = Suggest(tt.input, 2)
			_ = CorrectWord(tt.input)
		})
	}
}

// TestControlCharacters verifies handling of ASCII control characters (0x00-0x1F).
func TestControlCharacters(t *testing.T) {
	for i := range 32 {
		t.Run(fmt.Sprintf("control_0x%02X", i), func(t *testing.T) {
			input := "kitab" + string(rune(i)) + "lar"
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panicked with control char 0x%02X: %v", i, r)
				}
			}()
			_ = IsCorrect(input)
			_ = CorrectWord(input)
		})
	}
}

// TestOversizedInput verifies Correct returns input unchanged when it exceeds 1 MiB.
func TestOversizedInput(t *testing.T) {
	// 1 MiB + 1 byte
	oversized := strings.Repeat("a", maxInputBytes+1)
	got := Correct(oversized)
	if got != oversized {
		t.Errorf("Correct with %d byte input: got len=%d, want len=%d (unchanged)",
			len(oversized), len(got), len(oversized))
	}
}

// ---------------------------------------------------------------------------
// Internal Function Tests
// ---------------------------------------------------------------------------

// TestDamerauLevenshtein verifies the edit distance function.
func TestDamerauLevenshtein(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{
			name: "identical strings",
			a:    "kitab",
			b:    "kitab",
			want: 0,
		},
		{
			name: "one substitution",
			a:    "kitab",
			b:    "ketab",
			want: 1,
		},
		{
			name: "one deletion",
			a:    "kitab",
			b:    "ktab",
			want: 1,
		},
		{
			name: "one insertion",
			a:    "kitab",
			b:    "kiitab",
			want: 1,
		},
		{
			name: "both empty",
			a:    "",
			b:    "",
			want: 0,
		},
		{
			name: "a empty",
			a:    "",
			b:    "abc",
			want: 3,
		},
		{
			name: "b empty",
			a:    "abc",
			b:    "",
			want: 3,
		},
		{
			name: "transposition",
			a:    "ab",
			b:    "ba",
			want: 1,
		},
		{
			name: "unicode characters",
			a:    "gözəl",
			b:    "gozel",
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := damerauLevenshtein(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("damerauLevenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestGenerateDeletes verifies delete variant generation.
func TestGenerateDeletes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		dist      int
		wantCount int
		wantNil   bool
	}{
		{
			name:      "3 chars dist=1 produces 3 deletes",
			input:     "abc",
			dist:      1,
			wantCount: 3, // "bc", "ac", "ab"
		},
		{
			name:      "3 chars dist=2 produces 6 deletes",
			input:     "abc",
			dist:      2,
			wantCount: 6, // dist=1: "bc","ac","ab"; dist=2: "c","b","a"
		},
		{
			name:    "empty string dist=1 produces nil",
			input:   "",
			dist:    1,
			wantNil: true,
		},
		{
			name:      "single char dist=1 produces 1 delete",
			input:     "a",
			dist:      1,
			wantCount: 1, // ""
		},
		{
			name:    "zero dist produces nil",
			input:   "abc",
			dist:    0,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := generateDeletes(tt.input, tt.dist)

			if tt.wantNil {
				if got != nil {
					t.Errorf("generateDeletes(%q, %d) = %v (len=%d), want nil",
						tt.input, tt.dist, got, len(got))
				}
				return
			}

			if len(got) != tt.wantCount {
				t.Errorf("generateDeletes(%q, %d) produced %d deletes, want %d: %v",
					tt.input, tt.dist, len(got), tt.wantCount, got)
			}

			// Verify all results are unique.
			seen := make(map[string]struct{}, len(got))
			for _, d := range got {
				if _, dup := seen[d]; dup {
					t.Errorf("generateDeletes(%q, %d) produced duplicate: %q", tt.input, tt.dist, d)
				}
				seen[d] = struct{}{}
			}

			// Verify none equals the original.
			for _, d := range got {
				if d == tt.input {
					t.Errorf("generateDeletes(%q, %d) contains original string", tt.input, tt.dist)
				}
			}
		})
	}
}

// TestGenerateDeletesContainsSingleCharDelete verifies the empty string appears
// for single-char input at dist=1.
func TestGenerateDeletesContainsSingleCharDelete(t *testing.T) {
	t.Parallel()
	got := generateDeletes("a", 1)
	if len(got) != 1 || got[0] != "" {
		t.Errorf("generateDeletes(\"a\", 1) = %v, want [\"\"]", got)
	}
}
