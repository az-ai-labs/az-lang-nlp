// Package spell provides spell checking for Azerbaijani text using the
// SymSpell (Symmetric Delete) algorithm with morphology-aware validation.
//
// The package provides four functions:
//
//   - IsCorrect reports whether a word is correctly spelled.
//   - Suggest returns ranked correction candidates for a misspelled word.
//   - CorrectWord corrects a single word, preserving its case pattern.
//   - Correct corrects all misspelled words in a text.
//
// Words are validated through a layered approach:
//
//  1. Direct frequency dictionary lookup (O(1) hash map).
//  2. Morphological analysis via [morph.Analyze]: if any analysis yields
//     a known stem, the word is considered valid.
//  3. Diacritic normalization via [normalize.NormalizeWord]: if the
//     ASCII-degraded form normalizes to a different known word, it is valid
//     (e.g. "gozel" is recognized as "gözəl").
//
// Correction uses stem-level SymSpell lookup: inflected words are decomposed
// via [morph.Analyze], the stem is corrected, and the word is reconstructed
// with the original suffixes.
//
// The frequency dictionary is embedded via //go:embed and parsed in init(),
// making the API stateless and safe for concurrent use by multiple goroutines.
//
// Known limitations:
//
//   - Compound word splitting is not supported (v2).
//   - Diacritic substitutions are not given reduced edit distance weight (v2).
//   - Title-case words not in the dictionary are left unchanged by Correct
//     to avoid over-correcting proper nouns.
//
// Input must be Azerbaijani Latin in NFC form.
package spell

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/az-ai-labs/az-lang-nlp/internal/azcase"
	"github.com/az-ai-labs/az-lang-nlp/morph"
	"github.com/az-ai-labs/az-lang-nlp/normalize"
	"github.com/az-ai-labs/az-lang-nlp/tokenizer"
)

// Suggestion represents a spelling correction candidate.
type Suggestion struct {
	Term      string `json:"term"`      // corrected word
	Distance  int    `json:"distance"`  // edit distance from input
	Frequency int64  `json:"frequency"` // corpus frequency (higher = more common)
}

// IsCorrect reports whether word is correctly spelled.
// A word is correct if it is a known dictionary word, a morphologically
// valid inflected form with a known stem, or an unambiguous ASCII-degraded
// form of a known word (e.g. "gozel" for "gözəl").
// Returns true for empty strings. Returns true for words exceeding
// maxWordBytes (not a spelling issue). Returns true for words shorter
// than minWordRunes (too short to meaningfully spell-check).
func IsCorrect(word string) bool {
	if word == "" {
		return true
	}
	if len(word) > maxWordBytes {
		return true
	}

	lower := azcase.ToLower(word)

	if utf8.RuneCountInString(lower) < minWordRunes {
		return true
	}

	// Hyphenated words: each non-empty part must be correct independently.
	// Cap the number of parts to prevent CPU amplification on pathological input.
	if idx := strings.IndexByte(lower, '-'); idx > 0 && idx < len(lower)-1 {
		parts := strings.Split(word, "-")
		if len(parts) > maxHyphenParts {
			return true
		}
		for _, part := range parts {
			if part != "" && !IsCorrect(part) {
				return false
			}
		}
		return true
	}

	// Apostrophe handling: validate only the pre-apostrophe stem.
	for i, r := range lower {
		if i > 0 && isApostrophe(r) && i < len(lower)-1 {
			return IsCorrect(lower[:i])
		}
	}

	// Words containing digits are not natural language misspellings
	// (identifiers, codes, mixed content); treat as correct.
	for _, r := range lower {
		if unicode.IsDigit(r) {
			return true
		}
	}

	// Direct frequency dictionary hit.
	if _, ok := words[lower]; ok {
		return true
	}

	// Morphological analysis: any decomposition with a known stem validates the word.
	analyses := morph.Analyze(lower)
	for _, a := range analyses {
		if len(a.Morphemes) > 0 && morph.IsKnownStem(azcase.ToLower(a.Stem)) {
			return true
		}
	}

	// Diacritic normalization: if the normalized form differs and is valid, accept.
	normalized := normalize.NormalizeWord(lower)
	if normalized != lower {
		if _, ok := words[normalized]; ok {
			return true
		}
		for _, a := range morph.Analyze(normalized) {
			if len(a.Morphemes) > 0 && morph.IsKnownStem(azcase.ToLower(a.Stem)) {
				return true
			}
		}
	}

	return false
}

// Suggest returns spelling correction candidates for word, sorted by
// edit distance ascending then frequency descending.
// Returns nil if the word is correct or empty.
// maxDist caps the maximum edit distance (clamped to maxEditDistance).
// Callers who want only the best match can take the first element.
func Suggest(word string, maxDist int) []Suggestion {
	if word == "" || IsCorrect(word) {
		return nil
	}
	if len(word) > maxWordBytes {
		return nil
	}

	lower := azcase.ToLower(word)

	if maxDist > maxEditDistance {
		maxDist = maxEditDistance
	}

	// Try whole-word lookup first.
	if results := lookup(lower, maxDist); len(results) > 0 {
		for i := range results {
			results[i].Term = applyCase(word, results[i].Term)
		}
		return results
	}

	// Stem-level correction: decompose, correct the stem, reconstruct.
	var results []Suggestion
	seen := make(map[string]struct{})

	analyses := morph.Analyze(lower)
	for _, a := range analyses {
		if len(a.Morphemes) == 0 {
			continue
		}
		stem := azcase.ToLower(a.Stem)
		if morph.IsKnownStem(stem) {
			continue // stem already correct, nothing to fix
		}

		stemSuggestions := lookup(stem, maxDist)
		suffix := suffixSurface(a)

		for _, ss := range stemSuggestions {
			reconstructed := ss.Term + suffix

			if _, dup := seen[reconstructed]; dup {
				continue
			}
			seen[reconstructed] = struct{}{}

			// Validate the reconstruction produces a valid morphological form.
			reanalyses := morph.Analyze(reconstructed)
			valid := false
			for _, ra := range reanalyses {
				if len(ra.Morphemes) > 0 && morph.IsKnownStem(azcase.ToLower(ra.Stem)) {
					valid = true
					break
				}
			}
			if !valid {
				continue
			}

			results = append(results, Suggestion{
				Term:      reconstructed,
				Distance:  ss.Distance,
				Frequency: ss.Frequency,
			})
		}
	}

	if len(results) == 0 {
		return nil
	}

	// Deduplicate (belt-and-suspenders, seen map already handles this).
	sortSuggestions(results)

	for i := range results {
		results[i].Term = applyCase(word, results[i].Term)
	}

	return results
}

// CorrectWord returns the corrected form of a single word.
// Returns the original word if it is correct or has no suggestions.
// Preserves the case pattern of the input (title-case, all-upper, lowercase).
func CorrectWord(word string) string {
	if word == "" || len(word) > maxWordBytes {
		return word
	}
	if IsCorrect(word) {
		return word
	}

	suggestions := Suggest(word, maxEditDistance)
	if len(suggestions) == 0 {
		return word
	}

	return suggestions[0].Term
}

// Correct returns text with misspelled words replaced by their
// top correction candidate. Words with no suggestions are left unchanged.
// Non-word tokens (spaces, punctuation, numbers) are preserved.
// Returns the input unchanged for empty or oversized (>1 MiB) input.
func Correct(text string) string {
	if text == "" || len(text) > maxInputBytes {
		return text
	}

	tokens := tokenizer.WordTokens(text)
	if len(tokens) == 0 {
		return text
	}

	var sb strings.Builder
	sb.Grow(len(text))

	for _, tok := range tokens {
		if tok.Type != tokenizer.Word {
			sb.WriteString(tok.Text)
			continue
		}

		// Leave title-case unknown words unchanged to avoid over-correcting
		// proper nouns (names, places, organizations).
		if isTitleCase(tok.Text) && !IsCorrect(tok.Text) {
			sb.WriteString(tok.Text)
			continue
		}

		sb.WriteString(CorrectWord(tok.Text))
	}

	return sb.String()
}

// isTitleCase reports whether s has its first rune uppercase and is not
// entirely uppercase (which would be an acronym or shouting).
func isTitleCase(s string) bool {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || !unicode.IsUpper(r) {
		return false
	}
	// Check that not all remaining runes are uppercase.
	rest := s[size:]
	if rest == "" {
		return false // single character, not really title-case
	}
	for _, c := range rest {
		if unicode.IsLetter(c) && !unicode.IsUpper(c) {
			return true // found a lowercase letter after the initial uppercase
		}
	}
	return false // all uppercase, not title-case
}

// applyCase transfers the case pattern of original onto corrected.
// Three modes: all-upper, title-case (first rune upper), lowercase.
func applyCase(original, corrected string) string {
	if original == "" || corrected == "" {
		return corrected
	}

	if isAllUpper(original) {
		return toUpper(corrected)
	}

	firstRune, _ := utf8.DecodeRuneInString(original)
	if unicode.IsUpper(firstRune) {
		return upperFirst(corrected)
	}

	return corrected
}

// isAllUpper reports whether every letter in s is uppercase.
func isAllUpper(s string) bool {
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return hasLetter
}

// toUpper returns s with Azerbaijani-aware uppercasing applied to every rune.
func toUpper(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		sb.WriteRune(azcase.Upper(r))
	}
	return sb.String()
}

// upperFirst returns s with its first rune Azerbaijani-uppercased.
func upperFirst(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || size == 0 {
		return s
	}
	var sb strings.Builder
	sb.Grow(len(s))
	sb.WriteRune(azcase.Upper(r))
	sb.WriteString(s[size:])
	return sb.String()
}

// isApostrophe reports whether r is an apostrophe character
// (ASCII apostrophe, right single quote, or modifier letter apostrophe).
func isApostrophe(r rune) bool {
	return r == '\'' || r == '\u2019' || r == '\u02BC'
}

// suffixSurface concatenates the surface forms of all morphemes in an analysis,
// producing the suffix string that follows the stem.
func suffixSurface(a morph.Analysis) string {
	var sb strings.Builder
	for _, m := range a.Morphemes {
		sb.WriteString(m.Surface)
	}
	return sb.String()
}
