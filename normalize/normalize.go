// Package normalize restores missing Azerbaijani diacritics in text.
//
// Diacritics are restored using dictionary lookup against the morph
// package's stem dictionary. Words not found in the dictionary
// or with ambiguous mappings (e.g. "seher" could be "səhər" or "şəhər")
// are returned unchanged.
//
// Two functions are provided:
//
//   - Normalize processes full text: tokenizes, restores each word, reassembles.
//   - NormalizeWord processes a single word.
//
// Input must be Azerbaijani Latin in NFC form.
//
// All functions are safe for concurrent use by multiple goroutines.
//
// Known limitations:
//
//   - Inflected words not in the stem dictionary are not restored.
//   - Approximately 49 ambiguous ASCII forms are never restored (by design).
//   - The i/ı distinction is not resolved (both are valid after Turkic lowering).
//   - Full Unicode NFC normalization is not performed; input must already be NFC.
//   - Worst-case CPU cost is O(2^N) per word where N is the number of substitutable
//     positions (capped at 10). Callers processing untrusted input should apply
//     timeouts or rate limiting.
package normalize

import (
	"strings"

	"github.com/az-ai-labs/az-lang-nlp/internal/azcase"
	"github.com/az-ai-labs/az-lang-nlp/tokenizer"
)

// maxInputBytes is the maximum input size for Normalize.
// Inputs exceeding this are returned unchanged.
const maxInputBytes = 1 << 20 // 1 MiB

// maxHyphenParts is the maximum number of segments a hyphenated word may have.
// Words split into more parts than this are returned unchanged to avoid
// processing pathological inputs.
const maxHyphenParts = 8

// Normalize restores missing Azerbaijani diacritics in text.
// Tokenizes the input, restores each word independently, and reassembles.
// Unknown or ambiguous words are left unchanged.
// Returns the input unchanged for empty or oversized (>1 MiB) input.
func Normalize(s string) string {
	if s == "" || len(s) > maxInputBytes {
		return s
	}
	s = azcase.ComposeNFC(s)

	tokens := tokenizer.WordTokens(s)
	if len(tokens) == 0 {
		return s
	}

	var b strings.Builder
	b.Grow(len(s) + len(s)/2) // diacritics replace 1-byte ASCII with 2-byte UTF-8

	for _, tok := range tokens {
		if tok.Type == tokenizer.Word {
			b.WriteString(restoreWordToken(tok.Text))
		} else {
			b.WriteString(tok.Text)
		}
	}

	return b.String()
}

// NormalizeWord restores diacritics on a single word.
// Returns the input unchanged if the word is unknown or ambiguous.
func NormalizeWord(word string) string {
	if word == "" {
		return word
	}
	word = azcase.ComposeNFC(word)
	if len(word) > maxWordBytes {
		return word
	}
	return restoreWordToken(word)
}

// restoreWordToken handles compound words (hyphens, apostrophes)
// before delegating to restoreWord for each part.
func restoreWordToken(word string) string {
	// Handle hyphenated words: restore each part independently.
	if idx := strings.IndexByte(word, '-'); idx > 0 && idx < len(word)-1 {
		return restoreHyphenated(word)
	}

	// Handle apostrophe suffixes: restore the stem part only.
	for i, r := range word {
		if i > 0 && azcase.IsApostrophe(r) && i < len(word)-1 {
			stem := word[:i]
			suffix := word[i:]
			return restoreWord(stem) + suffix
		}
	}

	return restoreWord(word)
}

// restoreHyphenated splits on hyphens, restores each part, and rejoins.
// Words with more than maxHyphenParts segments are returned unchanged.
func restoreHyphenated(word string) string {
	parts := strings.Split(word, "-")
	if len(parts) <= 1 || len(parts) > maxHyphenParts {
		return word
	}
	for i, part := range parts {
		if part != "" {
			parts[i] = restoreWord(part)
		}
	}
	return strings.Join(parts, "-")
}
