package chunker

import (
	"testing"
	"unicode/utf8"
)

// verifyChunkInvariants checks the byte-offset invariant and UTF-8 validity
// for all chunks produced from input.
func verifyChunkInvariants(t *testing.T, input string, chunks []Chunk) {
	t.Helper()
	for i, c := range chunks {
		if c.Start < 0 || c.End > len(input) || c.Start > c.End {
			t.Fatalf("chunk %d: invalid offsets [%d:%d] for input len %d",
				i, c.Start, c.End, len(input))
		}
		if got := input[c.Start:c.End]; got != c.Text {
			t.Fatalf("chunk %d: offset invariant broken: input[%d:%d]=%q, Text=%q",
				i, c.Start, c.End, got, c.Text)
		}
		if !utf8.ValidString(c.Text) {
			t.Fatalf("chunk %d: Text is not valid UTF-8: %q", i, c.Text)
		}
		if c.Index != i {
			t.Fatalf("chunk %d: Index=%d, want %d", i, c.Index, i)
		}
	}
}

func FuzzBySize(f *testing.F) {
	f.Add("Salam, dünya!", 10, 3)
	f.Add("", 5, 0)
	f.Add("a", 1, 0)
	f.Add("Bakı şəhəri gözəldir.", 5, 2)
	f.Add("abc", 100, 50)

	f.Fuzz(func(t *testing.T, s string, size, overlap int) {
		if !utf8.ValidString(s) {
			return
		}
		chunks := BySize(s, size, overlap)
		if chunks == nil {
			return
		}
		verifyChunkInvariants(t, s, chunks)
	})
}

func FuzzBySentence(f *testing.F) {
	f.Add("Birinci cümlə. İkinci cümlə.", 20, 5)
	f.Add("", 10, 0)
	f.Add("Prof. Əliyev gəldi.", 100, 0)
	f.Add("Bir. İki. Üç.", 5, 2)
	f.Add("Salam\n\nDünya", 10, 3)

	f.Fuzz(func(t *testing.T, s string, size, overlap int) {
		if !utf8.ValidString(s) {
			return
		}
		chunks := BySentence(s, size, overlap)
		if chunks == nil {
			return
		}
		verifyChunkInvariants(t, s, chunks)
	})
}

func FuzzRecursive(f *testing.F) {
	f.Add("Birinci paraqraf.\n\nİkinci paraqraf.", 15, 3)
	f.Add("", 10, 0)
	f.Add("Bir cümlə.", 100, 0)
	f.Add("Salam dünya hər kəs üçün gözəldir.", 10, 2)
	f.Add("a b c d e f g h i j k l m n", 5, 1)

	f.Fuzz(func(t *testing.T, s string, size, overlap int) {
		if !utf8.ValidString(s) {
			return
		}
		chunks := Recursive(s, size, overlap)
		if chunks == nil {
			return
		}
		verifyChunkInvariants(t, s, chunks)
	})
}
