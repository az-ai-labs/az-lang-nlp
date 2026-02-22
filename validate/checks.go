package validate

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/az-ai-labs/az-lang-nlp/detect"
	"github.com/az-ai-labs/az-lang-nlp/internal/azcase"
	"github.com/az-ai-labs/az-lang-nlp/spell"
	"github.com/az-ai-labs/az-lang-nlp/tokenizer"
)

// ── Homoglyph table ────────────────────────────────────────────────────

// cyrToLat returns the Latin equivalent of a Cyrillic homoglyph rune,
// or (0, false) if r is not a known confusable.
func cyrToLat(r rune) (rune, bool) {
	switch r {
	case 'а':
		return 'a', true
	case 'е':
		return 'e', true
	case 'о':
		return 'o', true
	case 'р':
		return 'p', true
	case 'с':
		return 'c', true
	case 'х':
		return 'x', true
	case 'у':
		return 'y', true
	case 'А':
		return 'A', true
	case 'Е':
		return 'E', true
	case 'О':
		return 'O', true
	case 'Р':
		return 'P', true
	case 'С':
		return 'C', true
	case 'Х':
		return 'X', true
	case 'У':
		return 'Y', true
	}
	return 0, false
}

// latToCyr returns the Cyrillic equivalent of a Latin homoglyph rune,
// or (0, false) if r is not a known confusable.
func latToCyr(r rune) (rune, bool) {
	switch r {
	case 'a':
		return 'а', true
	case 'e':
		return 'е', true
	case 'o':
		return 'о', true
	case 'p':
		return 'р', true
	case 'c':
		return 'с', true
	case 'x':
		return 'х', true
	case 'y':
		return 'у', true
	case 'A':
		return 'А', true
	case 'E':
		return 'Е', true
	case 'O':
		return 'О', true
	case 'P':
		return 'Р', true
	case 'C':
		return 'С', true
	case 'X':
		return 'Х', true
	case 'Y':
		return 'У', true
	}
	return 0, false
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
		if containsDigit(tok.Text) {
			continue
		}
		// Skip title-case unknown words (likely proper nouns).
		if isTitleCase(tok.Text) {
			continue
		}
		if spell.IsCorrect(tok.Text) {
			continue
		}

		suggestion := ""
		if suggestions := spell.Suggest(tok.Text, maxEditDist); len(suggestions) > 0 {
			suggestion = applyCase(tok.Text, suggestions[0].Term)
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

// containsDigit reports whether s contains any digit rune.
func containsDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// isTitleCase reports whether s has its first rune uppercase and is not
// entirely uppercase (which would be an acronym).
func isTitleCase(s string) bool {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || !unicode.IsUpper(r) {
		return false
	}
	rest := s[size:]
	if rest == "" {
		return false
	}
	for _, c := range rest {
		if unicode.IsLetter(c) && !unicode.IsUpper(c) {
			return true
		}
	}
	return false
}

// applyCase transfers the case pattern of original onto corrected.
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

func toUpper(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		sb.WriteRune(azcase.Upper(r))
	}
	return sb.String()
}

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
		if tok.Type == tokenizer.Space && countSpaces(tok.Text) >= minDoubleSpaces {
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
		if tok.Type == tokenizer.Punctuation {
			if i > 0 && tokens[i-1].Type == tokenizer.Punctuation && tokens[i-1].Text == tok.Text {
				// Continuation of a run already reported.
			} else {
				count := 1
				for j := i + 1; j < len(tokens); j++ {
					if tokens[j].Type != tokenizer.Punctuation || tokens[j].Text != tok.Text {
						break
					}
					count++
				}
				if count >= minConsecutivePunct {
					r, _ := utf8.DecodeRuneInString(tok.Text)
					if !(r == '.' && count == ellipsisLength) {
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

// countSpaces counts the number of space characters (U+0020) in s.
func countSpaces(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
		}
	}
	return n
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
				if _, ok := cyrToLat(r); ok {
					needsReplacement = true
				}
			}
		case detect.ScriptCyrl:
			if unicode.Is(unicode.Latin, r) {
				if _, ok := latToCyr(r); ok {
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
			if lat, ok := cyrToLat(r); ok {
				sb.WriteRune(lat)
				continue
			}
		case detect.ScriptCyrl:
			if cyr, ok := latToCyr(r); ok {
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
func allHomoglyphs(word string, dominant detect.Script) bool {
	for _, r := range word {
		if !unicode.IsLetter(r) {
			continue
		}
		switch dominant {
		case detect.ScriptLatn:
			if _, ok := cyrToLat(r); !ok {
				return false
			}
		case detect.ScriptCyrl:
			if _, ok := latToCyr(r); !ok {
				return false
			}
		default:
			return false
		}
	}
	return true
}
