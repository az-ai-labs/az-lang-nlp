package validate

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/az-ai-labs/az-lang-nlp/azcase"
	"github.com/az-ai-labs/az-lang-nlp/detect"
	"github.com/az-ai-labs/az-lang-nlp/spell"
	"github.com/az-ai-labs/az-lang-nlp/tokenizer"
)

// ── Homoglyph table ────────────────────────────────────────────────────

// cyrToLat maps Cyrillic homoglyph runes to their Latin equivalents.
var cyrToLat = map[rune]rune{
	'а': 'a', 'е': 'e', 'о': 'o', 'р': 'p', 'с': 'c', 'х': 'x', 'у': 'y',
	'А': 'A', 'Е': 'E', 'О': 'O', 'Р': 'P', 'С': 'C', 'Х': 'X', 'У': 'Y',
	'і': 'i', 'І': 'I', // U+0456 / U+0406
	'ј': 'j', 'Ј': 'J', // U+0458 / U+0408
}

// latToCyr maps Latin homoglyph runes to their Cyrillic equivalents.
var latToCyr = map[rune]rune{
	'a': 'а', 'e': 'е', 'o': 'о', 'p': 'р', 'c': 'с', 'x': 'х', 'y': 'у',
	'A': 'А', 'E': 'Е', 'O': 'О', 'P': 'Р', 'C': 'С', 'X': 'Х', 'Y': 'У',
	'i': 'і', 'I': 'І', // U+0456 / U+0406
	'j': 'ј', 'J': 'Ј', // U+0458 / U+0408
}

// ── Spelling check ─────────────────────────────────────────────────────

// maxEditDist is the maximum edit distance for spell.Suggest calls.
const maxEditDist = 2

// appendSpellingIssues detects misspelled words via spell.IsCorrect.
// Skips non-Word tokens, empty tokens, digit-containing tokens, and
// title-case unknown words (proper noun heuristic).
// Spelling is skipped entirely when the dominant script is Cyrillic.
func appendSpellingIssues(issues []Issue, tokens []tokenizer.Token, det detect.Result) []Issue {
	// The spell module requires Latin-script input.
	if det.Script == detect.ScriptCyrl {
		return issues
	}

	for i := range tokens {
		if len(issues) >= maxIssues {
			return issues
		}

		tok := &tokens[i]
		if tok.Type != tokenizer.Word || tok.Text == "" {
			continue
		}
		if azcase.ContainsDigit(tok.Text) {
			continue
		}
		// Skip title-case unknown words (likely proper nouns).
		if azcase.IsTitleCase(tok.Text) {
			continue
		}
		if spell.IsCorrect(tok.Text) {
			continue
		}

		suggestion := ""
		if suggestions := spell.Suggest(tok.Text, maxEditDist); len(suggestions) > 0 {
			suggestion = azcase.ApplyCase(tok.Text, suggestions[0].Term)
		}

		issues = append(issues, Issue{
			Text:       tok.Text,
			Start:      tok.Start,
			End:        tok.End,
			Type:       Spelling,
			Severity:   Error,
			Message:    "unknown word",
			Suggestion: suggestion,
		})
	}

	return issues
}

// ── Punctuation check ──────────────────────────────────────────────────

const (
	minDoubleSpaces     = 2 // minimum spaces in a Space token to flag
	minConsecutivePunct = 2 // minimum identical punctuation chars to flag
	ellipsisLength      = 3 // three dots are a valid ellipsis
)

// appendPunctuationIssues detects spacing and repetition errors in the
// token stream.
func appendPunctuationIssues(issues []Issue, tokens []tokenizer.Token) []Issue {
	for i := range tokens {
		if len(issues) >= maxIssues {
			return issues
		}

		tok := &tokens[i]

		// Double/multiple spaces.
		if tok.Type == tokenizer.Space && strings.Count(tok.Text, " ") >= minDoubleSpaces {
			issues = append(issues, Issue{
				Text:       tok.Text,
				Start:      tok.Start,
				End:        tok.End,
				Type:       Punctuation,
				Severity:   Warning,
				Message:    "multiple spaces",
				Suggestion: " ",
			})
			continue
		}

		// Space before punctuation.
		if tok.Type == tokenizer.Space && i+1 < len(tokens) {
			next := &tokens[i+1]
			if next.Type == tokenizer.Punctuation && isSpaceSensitivePunct(next.Text) {
				issues = append(issues, Issue{
					Text:       tok.Text,
					Start:      tok.Start,
					End:        tok.End,
					Type:       Punctuation,
					Severity:   Warning,
					Message:    "space before punctuation",
					Suggestion: "",
				})
				continue
			}
		}

		// Missing space after sentence-ending punctuation.
		if tok.Type == tokenizer.Punctuation && isSentenceEnd(tok.Text) && i+1 < len(tokens) {
			next := &tokens[i+1]
			if next.Type == tokenizer.Word || next.Type == tokenizer.Number {
				issues = append(issues, Issue{
					Text:       tok.Text,
					Start:      tok.Start,
					End:        tok.End,
					Type:       Punctuation,
					Severity:   Warning,
					Message:    "missing space after punctuation",
					Suggestion: tok.Text + " ",
				})
				continue
			}
		}

		// Multiple consecutive identical punctuation (exclude ... ellipsis).
		// The tokenizer emits each punctuation character as a separate
		// token, so we detect runs of consecutive identical tokens.
		// Only report on the first token in a run.
		if tok.Type == tokenizer.Punctuation &&
			(i == 0 || tokens[i-1].Type != tokenizer.Punctuation || tokens[i-1].Text != tok.Text) {
			count := 1
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].Type != tokenizer.Punctuation || tokens[j].Text != tok.Text {
					break
				}
				count++
			}
			if count >= minConsecutivePunct {
				r, _ := utf8.DecodeRuneInString(tok.Text)
				if r != '.' || count != ellipsisLength {
					last := &tokens[i+count-1]
					issues = append(issues, Issue{
						Text:       strings.Repeat(tok.Text, count),
						Start:      tok.Start,
						End:        last.End,
						Type:       Punctuation,
						Severity:   Info,
						Message:    "repeated punctuation",
						Suggestion: tok.Text,
					})
				}
			}
		}
	}

	return issues
}

// isSpaceSensitivePunct reports whether the punctuation text should
// not be preceded by a space.
func isSpaceSensitivePunct(text string) bool {
	switch text {
	case ",", ".", ";", ":", "!", "?":
		return true
	}
	return false
}

// isSentenceEnd reports whether the punctuation text ends a sentence.
func isSentenceEnd(text string) bool {
	switch text {
	case ".", "!", "?":
		return true
	}
	return false
}

// ── Layout check (homoglyphs) ──────────────────────────────────────────

// appendLayoutIssues detects homoglyph characters from the wrong script.
func appendLayoutIssues(issues []Issue, tokens []tokenizer.Token, det detect.Result) []Issue {
	if det.Confidence < minDetectConfidence {
		return issues
	}
	if det.Script != detect.ScriptLatn && det.Script != detect.ScriptCyrl {
		return issues
	}

	for i := range tokens {
		if len(issues) >= maxIssues {
			return issues
		}

		tok := &tokens[i]
		if tok.Type != tokenizer.Word {
			continue
		}

		replaced, found := replaceHomoglyphs(tok.Text, det.Script)
		if !found {
			continue
		}

		issues = append(issues, Issue{
			Text:       tok.Text,
			Start:      tok.Start,
			End:        tok.End,
			Type:       Layout,
			Severity:   Error,
			Message:    "contains characters from wrong script (possible keyboard layout error)",
			Suggestion: replaced,
		})
	}

	return issues
}

// replaceHomoglyphs scans word for runes from the non-dominant script
// that are in the homoglyph table, replacing them with dominant-script
// equivalents. Returns the replaced string and true if any replacements
// were made.
func replaceHomoglyphs(word string, dominant detect.Script) (string, bool) {
	// Quick scan: check if any replacement is needed.
	needsReplacement := false
	for _, r := range word {
		switch dominant {
		case detect.ScriptLatn:
			if unicode.Is(unicode.Cyrillic, r) {
				if _, ok := cyrToLat[r]; ok {
					needsReplacement = true
				}
			}
		case detect.ScriptCyrl:
			if unicode.Is(unicode.Latin, r) {
				if _, ok := latToCyr[r]; ok {
					needsReplacement = true
				}
			}
		}
		if needsReplacement {
			break
		}
	}

	if !needsReplacement {
		return word, false
	}

	// Build the replaced string.
	var sb strings.Builder
	sb.Grow(len(word))

	for _, r := range word {
		switch dominant {
		case detect.ScriptLatn:
			if lat, ok := cyrToLat[r]; ok {
				sb.WriteRune(lat)
				continue
			}
		case detect.ScriptCyrl:
			if cyr, ok := latToCyr[r]; ok {
				sb.WriteRune(cyr)
				continue
			}
		}
		sb.WriteRune(r)
	}

	return sb.String(), true
}

// ── Mixed script check ─────────────────────────────────────────────────

// appendMixedScriptIssues detects tokens entirely in a non-dominant
// script. Independent pass with no dependency on layout check results.
// Tokens where ALL letter runes are homoglyphs are skipped (layout
// check handles those).
func appendMixedScriptIssues(issues []Issue, tokens []tokenizer.Token, det detect.Result) []Issue {
	if det.Confidence < minDetectConfidence {
		return issues
	}
	if det.Script != detect.ScriptLatn && det.Script != detect.ScriptCyrl {
		return issues
	}

	for i := range tokens {
		if len(issues) >= maxIssues {
			return issues
		}

		tok := &tokens[i]
		if tok.Type != tokenizer.Word {
			continue
		}

		if !isNonDominantScript(tok.Text, det.Script) {
			continue
		}

		// Skip tokens where all letter runes are homoglyphs.
		// The layout check handles those.
		if allHomoglyphs(tok.Text, det.Script) {
			continue
		}

		issues = append(issues, Issue{
			Text:       tok.Text,
			Start:      tok.Start,
			End:        tok.End,
			Type:       MixedScript,
			Severity:   Info,
			Message:    "text in different script",
			Suggestion: "",
		})
	}

	return issues
}

// isNonDominantScript reports whether all letter runes in word belong
// to a script different from the dominant one.
func isNonDominantScript(word string, dominant detect.Script) bool {
	hasLetter := false
	for _, r := range word {
		if !unicode.IsLetter(r) {
			continue
		}
		hasLetter = true
		switch dominant {
		case detect.ScriptLatn:
			if !unicode.Is(unicode.Cyrillic, r) {
				return false
			}
		case detect.ScriptCyrl:
			if !unicode.Is(unicode.Latin, r) {
				return false
			}
		default:
			return false
		}
	}
	return hasLetter
}

// allHomoglyphs reports whether every letter rune in word is a known
// homoglyph character. Used to avoid double-reporting with layout check.
// Returns false for words with no letter runes (avoids vacuous truth).
func allHomoglyphs(word string, dominant detect.Script) bool {
	hasLetter := false
	for _, r := range word {
		if !unicode.IsLetter(r) {
			continue
		}
		hasLetter = true
		switch dominant {
		case detect.ScriptLatn:
			if _, ok := cyrToLat[r]; !ok {
				return false
			}
		case detect.ScriptCyrl:
			if _, ok := latToCyr[r]; !ok {
				return false
			}
		default:
			return false
		}
	}
	return hasLetter
}
