// Package keywords extracts keywords from Azerbaijani text.
//
// Two algorithms are provided:
//
//   - TF-IDF: term frequency weighted by corpus-frequency-based inverse
//     document frequency. Best for ranking terms by discriminative power.
//     IDF is a proxy derived from corpus token frequencies, not true
//     document-level frequency.
//   - TextRank: graph-based co-occurrence ranking using PageRank. Requires
//     no corpus. Best for extracting central concepts from a single document.
//
// Two API layers:
//
//   - Structured: ExtractTFIDF and ExtractTextRank return []Keyword with
//     stems, scores, and counts.
//   - Convenience: Keywords returns []string of keyword stems.
//
// All functions are safe for concurrent use by multiple goroutines.
//
// Known limitations:
//
//   - Suffix homonymy (e.g. -ma/-mə verbal noun vs negation) is not
//     disambiguated. "gəlmə" (the act of coming) and "gəlmə" (don't come)
//     produce the same stem.
//   - Vowel-drop stems may occasionally miss groupings (e.g. oğlum → oğlu
//     instead of oğul).
//   - Foreign words pass through morph.Stem unchanged and are treated as
//     unique stems.
//   - Non-Azerbaijani Latin text (English, Turkish, etc.) will pass through
//     the pipeline and may appear as keywords since it bypasses morphological
//     analysis and stopword filtering.
//   - Input must be Azerbaijani Latin. Cyrillic input is not transliterated
//     automatically; use translit.CyrillicToLatin first.
//
// Input must be in NFC form. Use normalize.Normalize to restore diacritics
// from ASCII-degraded text.
package keywords

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/az-ai-labs/az-lang-nlp/internal/azcase"
	"github.com/az-ai-labs/az-lang-nlp/morph"
	"github.com/az-ai-labs/az-lang-nlp/normalize"
	"github.com/az-ai-labs/az-lang-nlp/tokenizer"
)

const (
	maxInputBytes = 1 << 20 // 1 MiB input guard
	defaultTopN   = 10      // default number of keywords returned
	minStemRunes  = 2       // minimum stem length to be a candidate

	// TextRank parameters
	textrankDamping    = 0.85   // PageRank damping factor
	textrankMaxIter    = 30     // maximum PageRank iterations
	textrankEpsilon    = 0.0001 // convergence threshold
	textrankWindowSize = 3      // co-occurrence sliding window size

	// Safety caps
	maxHyphenParts = 8 // max hyphen-separated parts per token
)

// Keyword represents a single extracted keyword with its score.
type Keyword struct {
	Stem  string  `json:"stem"`
	Score float64 `json:"score"`
	Count int     `json:"count"`
}

// pipeline runs normalize -> tokenize -> hyphen filter -> stem -> lowercase -> stopword filter.
// Returns the filtered lowercase stems ready for scoring.
func pipeline(text string) []string {
	if text == "" || len(text) > maxInputBytes {
		return nil
	}

	clean := normalize.Normalize(text)
	words := tokenizer.Words(clean)

	// Filter pathological hyphenation BEFORE stemming to prevent
	// CPU amplification in morph's per-part FSM processing.
	safe := make([]string, 0, len(words))
	for _, w := range words {
		if strings.Count(w, "-") < maxHyphenParts {
			safe = append(safe, w)
		}
	}

	stems := morph.Stems(safe)

	filtered := make([]string, 0, len(stems))
	for _, s := range stems {
		low := azcase.ToLower(s)
		if utf8.RuneCountInString(low) < minStemRunes {
			continue
		}
		if isStopword(low) {
			continue
		}
		filtered = append(filtered, low)
	}

	return filtered
}

// ExtractTFIDF returns the top keywords from text scored by TF-IDF.
// TF is normalized by document length. IDF uses corpus-frequency proxy
// from the embedded frequency dictionary.
// Results are sorted by score descending, with lexicographic tie-breaking.
// Returns nil for empty text or text exceeding maxInputBytes.
func ExtractTFIDF(text string, topN int) []Keyword {
	filtered := pipeline(text)
	if len(filtered) == 0 {
		return nil
	}
	if topN <= 0 {
		topN = defaultTopN
	}

	candidates := scoreTFIDF(filtered)
	slices.SortStableFunc(candidates, cmpKeyword)

	if len(candidates) > topN {
		candidates = candidates[:topN]
	}
	return candidates
}

// ExtractTextRank returns the top keywords from text scored by TextRank.
// Builds a co-occurrence graph and runs PageRank to rank terms.
// Results are sorted by score descending, with lexicographic tie-breaking.
// Returns nil for empty text or text exceeding maxInputBytes.
func ExtractTextRank(text string, topN int) []Keyword {
	filtered := pipeline(text)
	if len(filtered) == 0 {
		return nil
	}
	if topN <= 0 {
		topN = defaultTopN
	}

	candidates := scoreTextRank(filtered)
	slices.SortStableFunc(candidates, cmpKeyword)

	if len(candidates) > topN {
		candidates = candidates[:topN]
	}
	return candidates
}

// Keywords returns the stems of the top 10 keywords from text using
// TextRank with default parameters. Convenience wrapper over ExtractTextRank.
// Returns nil when no keywords are found.
func Keywords(text string) []string {
	kws := ExtractTextRank(text, defaultTopN)
	if len(kws) == 0 {
		return nil
	}
	result := make([]string, len(kws))
	for i, kw := range kws {
		result[i] = kw.Stem
	}
	return result
}

func cmpKeyword(a, b Keyword) int {
	if a.Score != b.Score {
		if a.Score > b.Score {
			return -1
		}
		return 1
	}
	return strings.Compare(a.Stem, b.Stem)
}
