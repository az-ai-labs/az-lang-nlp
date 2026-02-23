package chunker

import (
	"unicode/utf8"

	"github.com/az-ai-labs/az-lang-nlp/tokenizer"
)

// BySentence groups sentences into chunks up to size runes.
// Sentences are detected via tokenizer.SentenceTokens, which handles
// Azerbaijani abbreviations and double-newline paragraph breaks.
//
// Overlap re-includes whole trailing sentences from the previous chunk.
// When the last sentence of the previous chunk exceeds the overlap budget,
// overlap is skipped for that boundary.
//
// A single sentence exceeding size is emitted as-is (size is a target,
// not a hard cap). Original inter-sentence whitespace is preserved.
//
// Returns nil for empty text, invalid UTF-8, or size <= 0.
func BySentence(text string, size, overlap int) []Chunk {
	if !validate(text) || size <= 0 {
		return nil
	}
	overlap = clampOverlap(size, overlap)
	return bySentence(text, size, overlap)
}

// bySentence is the unexported implementation of BySentence.
func bySentence(text string, size, overlap int) []Chunk {
	sentences := tokenizer.SentenceTokens(text)
	if len(sentences) == 0 {
		return nil
	}

	chunks := make([]Chunk, 0, len(sentences)/2+1)
	groupStart := 0 // index into sentences for current group

	for groupStart < len(sentences) && len(chunks) < maxChunks {
		// Accumulate sentences until we reach or exceed the target size.
		groupEnd := groupStart
		runeCount := 0

		for groupEnd < len(sentences) {
			sentRunes := utf8.RuneCountInString(sentences[groupEnd].Text)
			if runeCount > 0 && runeCount+sentRunes > size {
				break
			}
			runeCount += sentRunes
			groupEnd++
		}

		// If no sentences were added (first sentence exceeds size), take at least one.
		if groupEnd == groupStart {
			groupEnd = groupStart + 1
		}

		startByte := sentences[groupStart].Start
		endByte := sentences[groupEnd-1].End

		chunks = append(chunks, Chunk{
			Text:  text[startByte:endByte],
			Start: startByte,
			End:   endByte,
			Index: len(chunks),
		})

		// Compute overlap: walk backwards from groupEnd to find sentences
		// that fit within the overlap rune budget. Ensure groupStart
		// advances by at least one sentence to guarantee progress.
		overlapSentences := 0
		if overlap > 0 && groupEnd < len(sentences) {
			overlapRunes := 0
			for i := groupEnd - 1; i >= groupStart; i-- {
				sentRunes := utf8.RuneCountInString(sentences[i].Text)
				if overlapRunes+sentRunes > overlap {
					break
				}
				overlapRunes += sentRunes
				overlapSentences++
			}
		}

		nextStart := groupEnd - overlapSentences
		if nextStart <= groupStart {
			nextStart = groupStart + 1
		}
		groupStart = nextStart
	}

	return chunks
}
