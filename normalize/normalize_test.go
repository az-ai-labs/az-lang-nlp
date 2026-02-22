package normalize

import (
	"strings"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// NormalizeWord — table-driven tests
// ---------------------------------------------------------------------------

func TestNormalizeWord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		// -- Known dictionary words restored --

		{"restore gozel", "gozel", "gözəl"},
		{"restore soz", "soz", "söz"},
		{"restore goz", "goz", "göz"},
		{"restore oz", "oz", "öz"},
		{"restore dovlet", "dovlet", "dövlət"},
		{"restore corek", "corek", "çörək"},
		{"restore azerbaycan", "azerbaycan", "azərbaycan"},
		{"restore muellim", "muellim", "müəllim"},
		{"restore sagird", "sagird", "şagird"},

		// -- Already correct words unchanged --

		{"already correct gözəl", "gözəl", "gözəl"},
		{"already correct kitab", "kitab", "kitab"},
		{"already correct ev", "ev", "ev"},
		{"already correct söz", "söz", "söz"},

		// -- Ambiguous words unchanged --

		{"ambiguous seher", "seher", "seher"}, // səhər or şəhər

		// -- Valid ASCII word unchanged --

		{"valid ascii ac", "ac", "ac"}, // ac (hungry) is a valid stem

		// -- Unknown/foreign words unchanged --

		{"foreign server", "server", "server"},
		{"foreign computer", "computer", "computer"},
		{"foreign test", "test", "test"},
		{"foreign hello", "hello", "hello"},

		// -- Case preservation --

		{"title case Gozel", "Gozel", "Gözəl"},
		{"title case Azerbaycan", "Azerbaycan", "Azərbaycan"},
		{"all caps GOZEL", "GOZEL", "GÖZƏL"},
		{"all caps AZERBAYCAN", "AZERBAYCAN", "AZƏRBAYCAN"},

		// -- Turkic-I handling --

		// After azLower: I -> ı (dotless), which has no diacritic alt, so word is unchanged.
		{"uppercase I preserved", "KITAB", "KITAB"},

		// -- No substitutable characters --

		{"restore gel", "gel", "gəl"},   // g->ğ and e->ə: only gəl matches
		{"no subs ev", "ev", "ev"},       // e is substitutable but ev is already a known stem
		{"only ascii latin", "park", "park"},

		// -- Apostrophe handling --

		{"apostrophe stem restored", "soz'un", "söz'un"},
		{"apostrophe curly quote", "soz\u2019un", "söz\u2019un"},

		// -- Hyphenated words --

		{"hyphenated restore", "gozel-kitab", "gözəl-kitab"},

		// -- Edge cases --

		{"empty string", "", ""},
		{"single char a", "a", "a"},
		{"single char e", "e", "e"},
		{"very long word", strings.Repeat("abcdefghij", 30), strings.Repeat("abcdefghij", 30)}, // >maxWordBytes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizeWord(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeWord(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Normalize — full text tests
// ---------------------------------------------------------------------------

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		// -- Basic text restoration --

		{"simple sentence", "gozel soz", "gözəl söz"},
		{"mixed known unknown", "gozel server", "gözəl server"},
		{"preserve spacing", "gozel  soz", "gözəl  söz"},
		{"preserve punctuation", "gozel, soz!", "gözəl, söz!"},

		// -- URLs, emails, numbers pass through --

		{"url unchanged", "https://gov.az gozel", "https://gov.az gözəl"},
		{"email unchanged", "user@mail.az gozel", "user@mail.az gözəl"},
		{"number unchanged", "123 gozel", "123 gözəl"},
		{"thousand separator", "1.000 gozel", "1.000 gözəl"},

		// -- Hyphenated words --

		{"hyphenated both parts", "sosial-iqtisadi", "sosial-iqtisadi"}, // likely not in dict as stems

		// -- Apostrophe suffixes --

		// The stem before apostrophe is restored, suffix left unchanged.

		// -- Cyrillic pass-through --

		{"cyrillic unchanged", "\u0411\u0430\u043a\u044b gozel", "\u0411\u0430\u043a\u044b gözəl"},

		// -- Ambiguous in context --

		{"ambiguous stays", "seher gozaldir", "seher gozaldir"}, // seher: ambiguous (səhər/şəhər); gozaldir: inflected form, stem-only lookup

		// -- Edge cases --

		{"empty string", "", ""},
		{"whitespace only", "   ", "   "},
		{"single word", "gozel", "gözəl"},

		// -- Idempotency --

		{"idempotent already restored", "gözəl söz", "gözəl söz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Normalize(tt.input)
			if got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Idempotency
// ---------------------------------------------------------------------------

func TestNormalizeIdempotent(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"gozel soz",
		"gözəl söz",
		"seher",
		"server test hello",
		"ac",
		"Azerbaycan",
		"",
		"123 test",
		"https://gov.az",
	}

	for _, input := range inputs {
		first := Normalize(input)
		second := Normalize(first)
		if first != second {
			t.Errorf("not idempotent for %q: first=%q, second=%q", input, first, second)
		}
	}
}

func TestNormalizeWordIdempotent(t *testing.T) {
	t.Parallel()

	words := []string{"gozel", "gözəl", "seher", "server", "ac", "Azerbaycan", "", "ev"}

	for _, word := range words {
		first := NormalizeWord(word)
		second := NormalizeWord(first)
		if first != second {
			t.Errorf("not idempotent for %q: first=%q, second=%q", word, first, second)
		}
	}
}

// ---------------------------------------------------------------------------
// Rune count preservation for NormalizeWord
// ---------------------------------------------------------------------------

func TestNormalizeWordRuneCount(t *testing.T) {
	t.Parallel()

	words := []string{"gozel", "soz", "goz", "oz", "seher", "server", "ac", "ev", "kitab"}

	for _, word := range words {
		got := NormalizeWord(word)
		if len([]rune(got)) != len([]rune(word)) {
			t.Errorf("NormalizeWord(%q) rune count changed: input=%d, output=%d",
				word, len([]rune(word)), len([]rune(got)))
		}
	}
}

// ---------------------------------------------------------------------------
// Edge cases: malformed UTF-8, null bytes, max input
// ---------------------------------------------------------------------------

func TestNormalizeEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"malformed utf8", "\xff\xfe gozel"},
		{"null byte", "gozel\x00soz"},
		{"control chars", "gozel\t\nsoz"},
		{"max input", strings.Repeat("gozel ", 200000)}, // ~1.2 MiB -> returned unchanged
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Must not panic.
			_ = Normalize(tt.input)
		})
	}
}

func TestNormalizeWordEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"malformed utf8", "\xff\xfe"},
		{"null byte", "gozel\x00"},
		{"single byte 0xff", "\xff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Must not panic.
			_ = NormalizeWord(tt.input)
		})
	}
}

// ---------------------------------------------------------------------------
// Max input enforcement
// ---------------------------------------------------------------------------

func TestNormalizeMaxInput(t *testing.T) {
	t.Parallel()

	big := strings.Repeat("a", maxInputBytes+1)
	got := Normalize(big)
	if got != big {
		t.Error("expected oversized input to be returned unchanged")
	}
}

// ---------------------------------------------------------------------------
// Concurrent safety
// ---------------------------------------------------------------------------

func TestConcurrentSafety(t *testing.T) {
	t.Parallel()

	input := "gozel soz seher server ac Azerbaycan"
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			Normalize(input)
			NormalizeWord("gozel")
			NormalizeWord("seher")
		})
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkNormalizeWord(b *testing.B) {
	for b.Loop() {
		NormalizeWord("azerbaycan")
	}
}

func BenchmarkNormalizeWordUnchanged(b *testing.B) {
	for b.Loop() {
		NormalizeWord("server")
	}
}

func BenchmarkNormalize(b *testing.B) {
	input := "Bu gozel seherde yasayiram. Azerbaycan gozal olkedir."
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		Normalize(input)
	}
}

func BenchmarkNormalizeLarge(b *testing.B) {
	input := strings.Repeat("Bu gozel seherde yasayiram. ", 1000)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		Normalize(input)
	}
}
