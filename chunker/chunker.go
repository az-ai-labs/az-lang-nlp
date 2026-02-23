// Package chunker splits Azerbaijani text into overlapping or non-overlapping
// chunks suitable for RAG/LLM pipelines.
//
// Three strategies are provided:
//
//   - BySize: pure rune-count splitting with no language awareness.
//   - BySentence: sentence-boundary aware splitting via the tokenizer package.
//   - Recursive: hierarchical splitting (paragraph > sentence > word > rune)
//     with greedy merge-back. This is the default used by the Chunks convenience
//     function.
//
// Two API layers:
//
//   - Structured: BySize, BySentence, and Recursive return []Chunk with byte
//     offsets and chunk index. The invariant text[c.Start:c.End] == c.Text
//     holds for every chunk produced from valid UTF-8 input.
//   - Convenience: Chunks returns []string for common use cases where offsets
//     are not needed.
//
// All functions are safe for concurrent use by multiple goroutines.
//
// Known limitations (v1.0):
//
//   - BySentence inherits the tokenizer's sentence-boundary limitations:
//     no quote/parenthesis nesting, no single-letter abbreviation handling.
//   - Recursive paragraph splitting handles "\n\n" only, not "\r\n\r\n".
//   - The size parameter is a target for BySentence, not a hard cap.
//     A single sentence exceeding size is emitted as-is.
package chunker

import (
	"fmt"
	"unicode/utf8"
)

const (
	maxChunks        = 10_000 // safety cap on output slice length
	defaultChunkSize = 512    // default target chunk size in runes
	defaultOverlap   = 50     // default overlap in runes
	minChunkRunes    = 10     // chunks shorter than this are merged with neighbor
)

// Chunk represents a text segment with metadata for RAG pipelines.
//
// Byte-offset invariant: for every Chunk c produced from input text,
// text[c.Start:c.End] == c.Text. This holds for all valid UTF-8 input.
type Chunk struct {
	Text  string `json:"text"`  // The chunk content
	Start int    `json:"start"` // Byte offset in original string (inclusive)
	End   int    `json:"end"`   // Byte offset in original string (exclusive)
	Index int    `json:"index"` // Zero-based chunk index
}

// String returns a debug representation, e.g. Chunk(0)[0:42](42 bytes).
func (c Chunk) String() string {
	return fmt.Sprintf("Chunk(%d)[%d:%d](%d bytes)", c.Index, c.Start, c.End, len(c.Text))
}

// validate checks common preconditions shared by all strategies.
// Returns false if the input should be rejected (caller returns nil).
func validate(text string) bool {
	return text != "" && utf8.ValidString(text)
}

// clampOverlap ensures overlap is within valid bounds relative to size.
// Returns the clamped overlap value.
func clampOverlap(size, overlap int) int {
	if overlap < 0 {
		return 0
	}
	if overlap >= size {
		return size - 1
	}
	return overlap
}

// BySize splits text into chunks of size runes with overlap rune overlap.
// This is a pure rune-count split with no language awareness.
// Returns nil for empty text, invalid UTF-8, or size <= 0.
func BySize(text string, size, overlap int) []Chunk {
	if !validate(text) || size <= 0 {
		return nil
	}
	overlap = clampOverlap(size, overlap)
	return bySize(text, size, overlap)
}

// bySize is the unexported implementation of BySize, also used as a fallback
// by the Recursive strategy for fragments that cannot be split further.
func bySize(text string, size, overlap int) []Chunk {
	if size <= 0 || text == "" {
		return nil
	}

	totalRunes := utf8.RuneCountInString(text)
	if totalRunes == 0 {
		return nil
	}

	step := size - overlap
	if step <= 0 {
		step = 1
	}

	// Pre-compute rune-to-byte offset map for the input.
	runeOffsets := buildRuneOffsets(text)

	capHint := min(totalRunes/step+1, maxChunks)
	chunks := make([]Chunk, 0, capHint)
	runePos := 0

	for runePos < totalRunes && len(chunks) < maxChunks {
		endRune := min(runePos+size, totalRunes)

		chunkRuneLen := endRune - runePos
		if chunkRuneLen < minChunkRunes && chunkRuneLen < size && len(chunks) > 0 {
			// Merge short trailing fragment with previous chunk.
			prev := &chunks[len(chunks)-1]
			prev.Text = text[prev.Start:runeOffsets[endRune]]
			prev.End = runeOffsets[endRune]
			break
		}

		startByte := runeOffsets[runePos]
		endByte := runeOffsets[endRune]

		chunks = append(chunks, Chunk{
			Text:  text[startByte:endByte],
			Start: startByte,
			End:   endByte,
			Index: len(chunks),
		})

		runePos += step
	}

	return chunks
}

// buildRuneOffsets returns a slice mapping rune index -> byte offset.
// The returned slice has len(runeOffsets) == runeCount + 1, where the
// last element is len(text).
func buildRuneOffsets(text string) []int {
	n := utf8.RuneCountInString(text)
	offsets := make([]int, 0, n+1)
	for i := range text {
		offsets = append(offsets, i)
	}
	offsets = append(offsets, len(text))
	return offsets
}

// Chunks splits text using the Recursive strategy with default parameters
// (size=512, overlap=50).
func Chunks(text string) []string {
	cs := Recursive(text, defaultChunkSize, defaultOverlap)
	if len(cs) == 0 {
		return nil
	}
	result := make([]string, len(cs))
	for i, c := range cs {
		result[i] = c.Text
	}
	return result
}
