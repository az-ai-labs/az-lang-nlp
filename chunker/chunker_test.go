package chunker

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// verifyInvariants checks the byte-offset invariant for every chunk:
// text[c.Start:c.End] == c.Text, and that indices are sequential.
func verifyInvariants(t *testing.T, input string, chunks []Chunk) {
	t.Helper()
	for i, c := range chunks {
		if c.Index != i {
			t.Errorf("chunk %d has Index=%d, want %d", i, c.Index, i)
		}
		if c.Start < 0 || c.End > len(input) || c.Start > c.End {
			t.Errorf("chunk %d has invalid offsets [%d:%d] for input len %d",
				i, c.Start, c.End, len(input))
			continue
		}
		if got := input[c.Start:c.End]; got != c.Text {
			t.Errorf("chunk %d offset invariant broken: input[%d:%d]=%q, Text=%q",
				i, c.Start, c.End, got, c.Text)
		}
	}
}

// ---------------------------------------------------------------------------
// BySize
// ---------------------------------------------------------------------------

func TestBySize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		size    int
		overlap int
		want    int // expected chunk count, -1 to skip count check
	}{
		{"empty string", "", 10, 0, 0},
		{"size zero", "hello", 0, 0, 0},
		{"size negative", "hello", -1, 0, 0},
		{"invalid utf8", "\xff\xfe", 10, 0, 0},

		{"single rune", "a", 10, 0, 1},
		{"exact fit", "abcde", 5, 0, 1},
		{"two chunks no overlap", "abcdefghij", 5, 0, 2},
		{"overlap zero", "abcdefghij", 5, 0, 2},

		{"azerbaijani diacritics", "Bakı şəhəri gözəldir.", 10, 0, -1},
		{"overlap between chunks", "abcdefghijklmnop", 8, 3, -1},
		{"overlap equals size minus one", "abcdefgh", 4, 3, -1},
		{"overlap exceeds size (clamped)", "abcdefgh", 4, 10, -1},
		{"negative overlap treated as zero", "abcdefgh", 4, -5, 2},

		{"short trailing fragment merged", "abcdefghijk", 5, 0, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BySize(tt.input, tt.size, tt.overlap)

			if tt.want == 0 {
				if got != nil {
					t.Errorf("expected nil, got %d chunks", len(got))
				}
				return
			}

			if tt.want > 0 && len(got) != tt.want {
				t.Errorf("expected %d chunks, got %d", tt.want, len(got))
			}

			if len(got) > 0 {
				verifyInvariants(t, tt.input, got)
			}
		})
	}
}

func TestBySizeOverlapContent(t *testing.T) {
	// Verify that overlap actually re-includes characters from the previous chunk.
	input := "abcdefghijklmnopqrst" // 20 runes
	chunks := BySize(input, 10, 5)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// The second chunk should start 5 runes before where the first chunk ended.
	first := chunks[0]
	second := chunks[1]

	// The overlap region should be present in both chunks.
	overlapText := input[second.Start:first.End]
	if !strings.HasSuffix(first.Text, overlapText) {
		t.Errorf("overlap text %q not a suffix of first chunk %q", overlapText, first.Text)
	}
	if !strings.HasPrefix(second.Text, overlapText) {
		t.Errorf("overlap text %q not a prefix of second chunk %q", overlapText, second.Text)
	}
}

// ---------------------------------------------------------------------------
// BySentence
// ---------------------------------------------------------------------------

func TestBySentence(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		size    int
		overlap int
		want    int // expected chunk count, -1 to skip count check
	}{
		{"empty string", "", 100, 0, 0},
		{"size zero", "Salam.", 0, 0, 0},
		{"invalid utf8", "\xff\xfe", 100, 0, 0},

		{"single sentence fits", "Bakı gözəldir.", 100, 0, 1},
		{"single sentence exceeds size", "Bakı gözəldir.", 5, 0, 1},
		{"two sentences fit in one chunk", "Salam. Necəsən?", 100, 0, 1},
		{"two sentences split", "Birinci cümlə burada. İkinci cümlə orada.", 22, 0, 2},

		{"azerbaijani abbreviation not split", "Prof. Əliyev gəldi.", 100, 0, 1},
		{"multiple sentences with overlap", "Bir. İki. Üç. Dörd.", 10, 5, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BySentence(tt.input, tt.size, tt.overlap)

			if tt.want == 0 {
				if got != nil {
					t.Errorf("expected nil, got %d chunks", len(got))
				}
				return
			}

			if tt.want > 0 && len(got) != tt.want {
				t.Errorf("expected %d chunks, got %d", tt.want, len(got))
			}

			if len(got) > 0 {
				verifyInvariants(t, tt.input, got)
			}
		})
	}
}

func TestBySentenceSingleSentenceExceedsSize(t *testing.T) {
	// A single long sentence should be emitted as-is even if it exceeds size.
	input := "Bu çox uzun bir cümlə olmalıdır ki ölçüdən böyük olsun."
	chunks := BySentence(input, 10, 0)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for oversized sentence, got %d", len(chunks))
	}
	if chunks[0].Text != input {
		t.Errorf("oversized sentence not preserved: got %q", chunks[0].Text)
	}
	verifyInvariants(t, input, chunks)
}

func TestBySentenceOverlapProgressGuarantee(t *testing.T) {
	// Regression: when overlap >= total rune count of all sentences in a group,
	// groupStart must still advance to prevent an infinite loop producing
	// 10,000 duplicate chunks.
	input := "Bir. İki. Üç. Dörd. Beş."
	chunks := BySentence(input, 20, 19)

	if len(chunks) > 100 {
		t.Fatalf("expected reasonable chunk count, got %d (degenerate loop?)", len(chunks))
	}
	verifyInvariants(t, input, chunks)
}

// ---------------------------------------------------------------------------
// Recursive
// ---------------------------------------------------------------------------

func TestRecursive(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		size    int
		overlap int
		want    int // expected chunk count, -1 to skip count check
	}{
		{"empty string", "", 100, 0, 0},
		{"size zero", "Salam.", 0, 0, 0},
		{"invalid utf8", "\xff\xfe", 100, 0, 0},

		{"short text fits", "Salam, dünya!", 100, 0, 1},
		{"paragraph split", "Birinci paraqraf.\n\nİkinci paraqraf.", 20, 0, 2},

		{"sentence fallback within paragraph",
			"Birinci cümlə. İkinci cümlə. Üçüncü cümlə.",
			20, 0, -1},

		{"word fallback for long sentence",
			strings.Repeat("sözlər ", 50),
			30, 0, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Recursive(tt.input, tt.size, tt.overlap)

			if tt.want == 0 {
				if got != nil {
					t.Errorf("expected nil, got %d chunks", len(got))
				}
				return
			}

			if tt.want > 0 && len(got) != tt.want {
				t.Errorf("expected %d chunks, got %d", tt.want, len(got))
			}

			if len(got) > 0 {
				verifyInvariants(t, tt.input, got)
			}
		})
	}
}

func TestRecursiveParagraphThenSentence(t *testing.T) {
	// Two paragraphs, second has two sentences that need splitting.
	input := "Qısa paraqraf.\n\nBirinci uzun cümlə burada yazılıb. İkinci uzun cümlə orada yazılıb."
	chunks := Recursive(input, 40, 0)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	verifyInvariants(t, input, chunks)

	// First chunk should be the first paragraph.
	if !strings.Contains(chunks[0].Text, "Qısa paraqraf.") {
		t.Errorf("first chunk should contain first paragraph, got %q", chunks[0].Text)
	}
}

// ---------------------------------------------------------------------------
// Chunks (convenience)
// ---------------------------------------------------------------------------

func TestChunks(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got := Chunks(""); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("short text", func(t *testing.T) {
		got := Chunks("Salam, dünya!")
		if len(got) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(got))
		}
		if got[0] != "Salam, dünya!" {
			t.Errorf("unexpected chunk: %q", got[0])
		}
	})
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestWhitespaceOnly(t *testing.T) {
	// Whitespace-only input is valid UTF-8 and non-empty.
	chunks := BySize("   \t\n  ", 5, 0)
	if chunks != nil {
		verifyInvariants(t, "   \t\n  ", chunks)
	}
}

func TestSingleRune(t *testing.T) {
	input := "a"
	for _, fn := range []struct {
		name string
		call func() []Chunk
	}{
		{"BySize", func() []Chunk { return BySize(input, 10, 0) }},
		{"BySentence", func() []Chunk { return BySentence(input, 10, 0) }},
		{"Recursive", func() []Chunk { return Recursive(input, 10, 0) }},
	} {
		t.Run(fn.name, func(t *testing.T) {
			got := fn.call()
			if len(got) != 1 {
				t.Fatalf("expected 1 chunk, got %d", len(got))
			}
			if got[0].Text != "a" {
				t.Errorf("expected %q, got %q", "a", got[0].Text)
			}
			verifyInvariants(t, input, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Concurrent safety
// ---------------------------------------------------------------------------

func TestConcurrentSafety(t *testing.T) {
	input := "Birinci cümlə. İkinci cümlə. Üçüncü cümlə.\n\nYeni paraqraf burada."
	const goroutines = 100

	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			BySize(input, 20, 5)
			BySentence(input, 20, 5)
			Recursive(input, 20, 5)
			Chunks(input)
		})
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkBySize(b *testing.B) {
	input := strings.Repeat("Bakı şəhəri gözəldir. ", 1000)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		BySize(input, defaultChunkSize, defaultOverlap)
	}
}

func BenchmarkBySentence(b *testing.B) {
	input := strings.Repeat("Bakı şəhəri gözəldir. ", 1000)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		BySentence(input, defaultChunkSize, defaultOverlap)
	}
}

func BenchmarkRecursive(b *testing.B) {
	input := strings.Repeat("Bakı şəhəri gözəldir. ", 1000)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		Recursive(input, defaultChunkSize, defaultOverlap)
	}
}

// ---------------------------------------------------------------------------
// Examples
// ---------------------------------------------------------------------------

func ExampleChunks() {
	chunks := Chunks("Salam, dünya!")
	fmt.Println(chunks)
	// Output:
	// [Salam, dünya!]
}

func ExampleBySize() {
	chunks := BySize("abcdefghij", 5, 0)
	for _, c := range chunks {
		fmt.Printf("[%d:%d] %q\n", c.Start, c.End, c.Text)
	}
	// Output:
	// [0:5] "abcde"
	// [5:10] "fghij"
}

func ExampleBySentence() {
	chunks := BySentence("Birinci cümlə. İkinci cümlə.", 100, 0)
	for _, c := range chunks {
		fmt.Printf("[%d:%d] %q\n", c.Start, c.End, c.Text)
	}
	// Output:
	// [0:33] "Birinci cümlə. İkinci cümlə."
}
