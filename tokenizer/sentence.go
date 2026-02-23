package tokenizer

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/az-ai-labs/az-lang-nlp/internal/azcase"
)

// abbreviations maps common Azerbaijani abbreviations (lowercase, with trailing dot)
// to true. Used to suppress false sentence breaks after abbreviated words.
// "az." is included to support greedy forward matching to "az.r.".
var abbreviations = map[string]bool{
	"prof.": true, "dos.": true, "ak.": true, "dr.": true,
	"az.": true, "az.r.": true, "ar.": true,
	"b.e.": true, "m.e.": true, "e.ə.": true,
	"vb.": true,
	"km.": true, "kq.": true, "sm.": true, "min.": true,
}

// sentenceTokens splits s into sentence-level tokens.
// Adjacent tokens cover the entire input without gaps or overlaps:
// concatenating all Token.Text values reconstructs s exactly.
func sentenceTokens(s string) []Token {
	tokens := make([]Token, 0, len(s)/40+1)
	sentStart := 0 // byte offset where the current sentence begins

	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])

		// Double newline forces a sentence break regardless of punctuation.
		if r == '\n' && i+1 < len(s) && s[i+1] == '\n' {
			// Consume all consecutive newlines as part of the current sentence.
			j := i
			for j < len(s) && s[j] == '\n' {
				j++
			}
			tokens = append(tokens, Token{
				Text:  s[sentStart:j],
				Start: sentStart,
				End:   j,
				Type:  Sentence,
			})
			sentStart = j
			i = j
			continue
		}

		// Check for terminal punctuation: . ? !
		if r == '.' || r == '?' || r == '!' {
			// Handle ellipsis: three consecutive dots or the Unicode ellipsis character.
			if r == '.' && i+2 < len(s) && s[i+1] == '.' && s[i+2] == '.' {
				// Consume all consecutive dots (handles "..." and "....")
				j := i
				for j < len(s) && s[j] == '.' {
					j++
				}
				if followedByWhitespaceUppercase(s, j) {
					breakPos := j
					tokens = append(tokens, Token{
						Text:  s[sentStart:breakPos],
						Start: sentStart,
						End:   breakPos,
						Type:  Sentence,
					})
					sentStart = breakPos
				}
				i = j
				continue
			}

			// Single dot: check for abbreviation.
			if r == '.' {
				if isAbbreviation(s, i) {
					i += size
					continue
				}
			}

			// Terminal punctuation: consume the entire cluster (e.g. "?!", "???").
			j := i + size
			for j < len(s) {
				nr, ns := utf8.DecodeRuneInString(s[j:])
				if nr == '.' || nr == '?' || nr == '!' {
					j += ns
				} else {
					break
				}
			}

			if followedByWhitespaceUppercase(s, j) {
				tokens = append(tokens, Token{
					Text:  s[sentStart:j],
					Start: sentStart,
					End:   j,
					Type:  Sentence,
				})
				sentStart = j
			}
			i = j
			continue
		}

		// Unicode ellipsis U+2026.
		if r == '\u2026' {
			j := i + size
			if followedByWhitespaceUppercase(s, j) {
				tokens = append(tokens, Token{
					Text:  s[sentStart:j],
					Start: sentStart,
					End:   j,
					Type:  Sentence,
				})
				sentStart = j
			}
			i = j
			continue
		}

		i += size
	}

	// Emit the final sentence if there is remaining text.
	if sentStart < len(s) {
		tokens = append(tokens, Token{
			Text:  s[sentStart:],
			Start: sentStart,
			End:   len(s),
			Type:  Sentence,
		})
	}

	return tokens
}

// followedByWhitespaceUppercase reports whether position pos in s is followed
// by at least one whitespace character and then an uppercase letter.
func followedByWhitespaceUppercase(s string, pos int) bool {
	i := pos
	foundSpace := false
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if unicode.IsSpace(r) {
			foundSpace = true
			i += size
		} else {
			return foundSpace && unicode.IsUpper(r)
		}
	}
	return false
}

// isAbbreviation checks whether the dot at byte position dotPos is part of
// a known abbreviation rather than a sentence-ending period.
// It also handles the special multi-word abbreviation "və s." pattern.
func isAbbreviation(s string, dotPos int) bool {
	// Extract the word immediately before the dot.
	word, wordStart := wordBefore(s, dotPos)
	if word == "" {
		return false
	}

	lower := azcase.ToLower(word)
	candidate := lower + "."

	// Special case: "və s." — if the word is "s" and the previous word is "və",
	// suppress the sentence break.
	if lower == "s" {
		prevWord, _ := wordBefore(s, wordStart)
		if strings.EqualFold(prevWord, "və") {
			return true
		}
	}

	if !abbreviations[candidate] {
		return false
	}

	// Greedy forward matching: check if the abbreviation extends further.
	// For example, after matching "az.", check if what follows forms "az.r.".
	afterDot := dotPos + 1
	return greedyAbbreviation(s, candidate, afterDot)
}

// greedyAbbreviation tries to extend a matched abbreviation prefix forward.
// It returns true once no further extension is possible, confirming the abbreviation.
// For example: prefix="az.", pos points to text after the dot.
// If next chars are "r.", it checks "az.r." — if that is also an abbreviation, recurse.
func greedyAbbreviation(s, prefix string, pos int) bool {
	// Try to read the next word and dot to extend the abbreviation.
	// The next segment must be: word + "." immediately adjacent (no whitespace).
	if pos >= len(s) {
		return true // abbreviation at end of input
	}

	// Read next word characters (letters only, no whitespace allowed).
	j := pos
	for j < len(s) {
		r, size := utf8.DecodeRuneInString(s[j:])
		if unicode.IsLetter(r) {
			j += size
		} else {
			break
		}
	}

	if j == pos || j >= len(s) || s[j] != '.' {
		return true // no extension possible, current match stands
	}

	// We have a potential extension: prefix + nextWord + "."
	nextWord := azcase.ToLower(s[pos:j])
	extended := prefix + nextWord + "."

	if abbreviations[extended] {
		// The extended form is also an abbreviation; recurse past its dot.
		return greedyAbbreviation(s, extended, j+1)
	}

	return true // extension not recognized, current match stands
}

// wordBefore extracts the word immediately before byte position pos.
// A word consists of consecutive letters (unicode.IsLetter).
// Returns the word text and the byte offset where the word starts.
// Returns ("", pos) if no word is found.
func wordBefore(s string, pos int) (string, int) {
	// Skip any dots immediately before pos (for multi-part abbreviations like "b.e.").
	i := pos
	for i > 0 {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		if r == '.' {
			i -= size
		} else {
			break
		}
	}

	// Now walk back over letters.
	end := i
	for i > 0 {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		if unicode.IsLetter(r) {
			i -= size
		} else {
			break
		}
	}

	if i == end {
		return "", pos
	}

	return s[i:end], i
}
