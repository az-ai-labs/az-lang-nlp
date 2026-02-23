package chunker

import (
	"strings"
	"unicode/utf8"

	"github.com/az-ai-labs/az-lang-nlp/tokenizer"
)

// paragraphSeparator is the delimiter used for paragraph-level splitting.
const paragraphSeparator = "\n\n"

// Recursive splits text hierarchically: paragraph > sentence > word > rune.
// After splitting, adjacent small pieces are greedily merged back up to
// size runes. Overlap is applied as rune-count overlap between merged chunks.
//
// Returns nil for empty text, invalid UTF-8, or size <= 0.
func Recursive(text string, size, overlap int) []Chunk {
	if !validate(text) || size <= 0 {
		return nil
	}
	overlap = clampOverlap(size, overlap)

	// Split the text into leaf fragments that are each <= size runes.
	fragments := splitRecursive(text, size)
	if len(fragments) == 0 {
		return nil
	}

	// Greedy merge: combine adjacent fragments up to the target size.
	merged := mergeFragments(text, fragments, size)

	// Apply overlap and build final chunks.
	return applyOverlap(text, merged, overlap)
}

// fragment represents a text segment with byte offsets, used internally
// during the split-and-merge pipeline.
type fragment struct {
	start int // byte offset (inclusive)
	end   int // byte offset (exclusive)
}

// splitRecursive breaks text into fragments that are each <= size runes,
// using a hierarchy of separators: paragraph > sentence > word > rune.
func splitRecursive(text string, size int) []fragment {
	root := fragment{start: 0, end: len(text)}
	return splitFragment(text, root, size, levelParagraph)
}

// splitLevel indicates the current depth in the recursive hierarchy.
type splitLevel int

const (
	levelParagraph splitLevel = iota
	levelSentence
	levelWord
	levelRune
)

// splitFragment recursively splits a fragment into pieces <= size runes.
func splitFragment(text string, frag fragment, size int, level splitLevel) []fragment {
	fragText := text[frag.start:frag.end]
	if utf8.RuneCountInString(fragText) <= size {
		return []fragment{frag}
	}

	var parts []fragment

	switch level {
	case levelParagraph:
		parts = splitByParagraph(frag, fragText)
	case levelSentence:
		parts = splitByTokens(frag, tokenizer.SentenceTokens(fragText))
	case levelWord:
		parts = splitByTokens(frag, tokenizer.WordTokens(fragText))
	default: // levelRune or any future level beyond levelWord
		return splitByRune(text, frag, size)
	}

	// If splitting at this level produced no useful split (still one piece),
	// descend to the next level.
	if len(parts) <= 1 {
		return splitFragment(text, frag, size, level+1)
	}

	// Recursively split any oversized parts at the next level.
	result := make([]fragment, 0, len(parts))
	for _, p := range parts {
		pText := text[p.start:p.end]
		if utf8.RuneCountInString(pText) <= size {
			result = append(result, p)
		} else {
			result = append(result, splitFragment(text, p, size, level+1)...)
		}
	}

	return result
}

// splitByParagraph splits a fragment on "\n\n" boundaries.
// The separator is attached to the preceding segment to preserve byte coverage.
func splitByParagraph(frag fragment, fragText string) []fragment {
	sep := paragraphSeparator
	var result []fragment
	pos := 0
	for {
		idx := strings.Index(fragText[pos:], sep)
		if idx < 0 {
			break
		}
		end := pos + idx + len(sep)
		result = append(result, fragment{start: frag.start + pos, end: frag.start + end})
		pos = end
	}
	if pos < len(fragText) {
		result = append(result, fragment{start: frag.start + pos, end: frag.end})
	}
	return result
}

// splitByTokens splits a fragment using the given tokenizer function.
// Used for both sentence and word level splitting.
func splitByTokens(frag fragment, tokens []tokenizer.Token) []fragment {
	if len(tokens) <= 1 {
		return []fragment{frag}
	}

	result := make([]fragment, len(tokens))
	for i, t := range tokens {
		result[i] = fragment{
			start: frag.start + t.Start,
			end:   frag.start + t.End,
		}
	}
	return result
}

// splitByRune splits a fragment into pieces of exactly size runes.
// This is the terminal level — no further recursion.
func splitByRune(text string, frag fragment, size int) []fragment {
	fragText := text[frag.start:frag.end]
	offsets := buildRuneOffsets(fragText)
	totalRunes := len(offsets) - 1

	capHint := min(totalRunes/size+1, maxChunks)
	result := make([]fragment, 0, capHint)
	for runePos := 0; runePos < totalRunes && len(result) < maxChunks; runePos += size {
		endRune := min(runePos+size, totalRunes)
		result = append(result, fragment{
			start: frag.start + offsets[runePos],
			end:   frag.start + offsets[endRune],
		})
	}
	return result
}

// mergeFragments greedily merges adjacent fragments up to size runes.
func mergeFragments(text string, frags []fragment, size int) []fragment {
	if len(frags) == 0 {
		return nil
	}

	merged := make([]fragment, 0, len(frags))
	current := frags[0]
	currentRunes := utf8.RuneCountInString(text[current.start:current.end])

	emit := func() {
		if currentRunes >= minChunkRunes || len(merged) == 0 {
			merged = append(merged, current)
		} else {
			merged[len(merged)-1].end = current.end
		}
	}

	for i := 1; i < len(frags); i++ {
		nextRunes := utf8.RuneCountInString(text[frags[i].start:frags[i].end])

		if currentRunes+nextRunes <= size {
			current.end = frags[i].end
			currentRunes += nextRunes
		} else {
			emit()
			current = frags[i]
			currentRunes = nextRunes
		}
	}

	emit()
	return merged
}

// applyOverlap converts merged fragments into Chunks, applying rune-count
// overlap between adjacent chunks.
func applyOverlap(text string, frags []fragment, overlap int) []Chunk {
	if len(frags) == 0 {
		return nil
	}

	chunks := make([]Chunk, 0, len(frags))

	for i, f := range frags {
		if len(chunks) >= maxChunks {
			break
		}

		startByte := f.start

		// For chunks after the first, extend the start backwards by overlap runes
		// into the previous fragment's territory.
		if i > 0 && overlap > 0 {
			startByte = walkBackRunes(text, f.start, frags[i-1].end, overlap)
		}

		chunks = append(chunks, Chunk{
			Text:  text[startByte:f.end],
			Start: startByte,
			End:   f.end,
			Index: len(chunks),
		})
	}

	return chunks
}

// walkBackRunes walks backwards from pos by up to n runes, but not past limit.
// Returns the new byte offset.
func walkBackRunes(text string, pos, limit, n int) int {
	result := pos
	for range n {
		if result <= limit {
			break
		}
		_, size := utf8.DecodeLastRuneInString(text[:result])
		if size == 0 {
			break
		}
		result -= size
	}
	return result
}
