package tokenizer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// wordTokens splits s into tokens using a rune-by-rune state machine.
// The caller guarantees s is non-empty.
//
// Rule priority (highest first):
//   - URL detection (http:// or https://)
//   - Email detection (backtrack from @)
//   - Number grouping (dot as thousand separator, comma as decimal)
//   - Hyphen joining (single U+002D between letter/digit)
//   - Apostrophe joining (U+0027, U+2019, U+02BC between letters)
//   - Default unicode classification
func wordTokens(s string) []Token {
	tokens := make([]Token, 0, len(s)/4+1)

	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])

		// Rule 1: URL detection — check for http:// or https://
		if (r == 'h' || r == 'H') && i+7 <= len(s) {
			if end, ok := scanURL(s, i); ok {
				tokens = append(tokens, Token{Text: s[i:end], Start: i, End: end, Type: URL})
				i = end
				continue
			}
		}

		// Rule 2: Email detection — when we see @, backtrack for local part
		if r == '@' {
			if start, end, ok := scanEmail(s, i); ok {
				// If we already emitted tokens that overlap the local part, replace them.
				tokens = trimTokensForEmail(tokens, start)
				tokens = append(tokens, Token{Text: s[start:end], Start: start, End: end, Type: Email})
				i = end
				continue
			}
		}

		// Whitespace: merge contiguous into one Space token
		if unicode.IsSpace(r) {
			start := i
			i += size
			for i < len(s) {
				nr, ns := utf8.DecodeRuneInString(s[i:])
				if !unicode.IsSpace(nr) {
					break
				}
				i += ns
			}
			tokens = append(tokens, Token{Text: s[start:i], Start: start, End: i, Type: Space})
			continue
		}

		// Digits: scan a number token with possible thousand-separator dots and decimal comma
		if unicode.IsDigit(r) {
			tok := scanNumber(s, i)
			tokens = append(tokens, tok)
			i = tok.End
			continue
		}

		// Letters: scan a word token with possible hyphens and apostrophes
		if unicode.IsLetter(r) {
			tok := scanWord(s, i)
			tokens = append(tokens, tok)
			i = tok.End
			continue
		}

		// Punctuation: collect contiguous same-class punctuation
		// Special case: consecutive hyphens, en-dash, em-dash are always punctuation
		if unicode.IsPunct(r) {
			start := i
			i += size
			// Merge consecutive punctuation of the same rune for cases like "--"
			if r == '-' {
				for i < len(s) {
					nr, ns := utf8.DecodeRuneInString(s[i:])
					if nr != '-' {
						break
					}
					i += ns
				}
			}
			tokens = append(tokens, Token{Text: s[start:i], Start: start, End: i, Type: Punctuation})
			continue
		}

		// Fallback: treat unclassified runes as Symbol
		tokens = append(tokens, Token{Text: s[i : i+size], Start: i, End: i + size, Type: Symbol})
		i += size
	}

	return tokens
}

// scanURL checks if s[pos:] starts with http:// or https:// and consumes
// until whitespace or end of string. Strips a single trailing punctuation
// mark (. , ! ?) from the URL text.
func scanURL(s string, pos int) (end int, ok bool) {
	rest := s[pos:]
	prefix := ""
	if len(rest) >= 8 && (rest[0] == 'h' || rest[0] == 'H') &&
		(rest[1] == 't' || rest[1] == 'T') &&
		(rest[2] == 't' || rest[2] == 'T') &&
		(rest[3] == 'p' || rest[3] == 'P') {
		if (rest[4] == 's' || rest[4] == 'S') && rest[5] == ':' && rest[6] == '/' && rest[7] == '/' {
			prefix = "https://"
		} else if rest[4] == ':' && rest[5] == '/' && rest[6] == '/' {
			prefix = "http://"
		}
	}
	if prefix == "" {
		return 0, false
	}

	// Must have at least one character after the protocol
	if len(rest) <= len(prefix) {
		return 0, false
	}

	// Consume until whitespace or end
	end = pos + len(rest)
	for j := pos + len(prefix); j < len(s); {
		r, size := utf8.DecodeRuneInString(s[j:])
		if unicode.IsSpace(r) {
			end = j
			break
		}
		j += size
	}

	// Strip a single trailing punctuation: . , ! ?
	if end > pos+len(prefix) {
		last, lastSize := utf8.DecodeLastRuneInString(s[pos:end])
		if last == '.' || last == ',' || last == '!' || last == '?' {
			end -= lastSize
		}
	}

	// Validate: URL must have content after protocol
	if end <= pos+len(prefix) {
		return 0, false
	}

	return end, true
}

// scanEmail detects an email around the @ at position atPos.
// It backtracks to find the local part and scans forward for the domain.
// Returns the byte offsets [start, end) and whether a valid email was found.
func scanEmail(s string, atPos int) (start, end int, ok bool) {
	// Backtrack for local part: [a-zA-Z0-9._%+-]+
	start = atPos
	for start > 0 {
		r, size := utf8.DecodeLastRuneInString(s[:start])
		if isEmailLocalChar(r) {
			start -= size
		} else {
			break
		}
	}
	if start == atPos {
		return 0, 0, false
	}

	// Skip leading dots — RFC 5321 disallows dots as the first character.
	for start < atPos && s[start] == '.' {
		start++
	}
	if start == atPos {
		return 0, 0, false
	}

	// Scan forward for domain: [a-zA-Z0-9.-]+
	end = atPos + 1 // skip @
	for end < len(s) {
		r, size := utf8.DecodeRuneInString(s[end:])
		if isEmailDomainChar(r) {
			end += size
		} else {
			break
		}
	}

	// Strip trailing dots from the domain before validation.
	// A domain like "example.com." has a trailing dot that is not part of the TLD.
	for end > atPos+1 && s[end-1] == '.' {
		end--
	}

	// Validate domain has at least one dot and a TLD of 2+ alpha chars
	domain := s[atPos+1 : end]
	lastDot := strings.LastIndex(domain, ".")
	if lastDot < 1 {
		return 0, 0, false
	}
	tld := domain[lastDot+1:]
	if len(tld) < 2 || !isAllAlpha(tld) {
		return 0, 0, false
	}

	return start, end, true
}

// scanNumber reads a number token starting at position pos.
// Handles thousand-separator dots (groups of exactly 3) and decimal commas.
func scanNumber(s string, pos int) Token {
	i := pos

	// Consume initial digits
	for i < len(s) && isDigitByte(s[i]) {
		i++
	}

	// Try thousand-separator dots: \d{1,3}(\.\d{3})+
	for i < len(s) && s[i] == '.' {
		// Peek ahead: must be exactly 3 digits followed by non-digit or end
		if i+4 <= len(s) && isDigitByte(s[i+1]) && isDigitByte(s[i+2]) && isDigitByte(s[i+3]) {
			// Check that after the 3 digits there is NOT another digit
			if i+4 >= len(s) || !isDigitByte(s[i+4]) {
				i += 4
				continue
			}
		}
		break
	}

	// Try decimal comma: must be followed by at least one digit
	if i < len(s) && s[i] == ',' {
		if i+1 < len(s) && isDigitByte(s[i+1]) {
			i++ // skip comma
			for i < len(s) && isDigitByte(s[i]) {
				i++
			}
		}
	}

	return Token{Text: s[pos:i], Start: pos, End: i, Type: Number}
}

// scanWord reads a word token starting at position pos.
// A word begins with a letter and may contain digits (e.g. "A4"), single
// hyphens (U+002D) between letters/digits, and apostrophes (U+0027,
// U+2019, U+02BC) between letters.
func scanWord(s string, pos int) Token {
	i := pos

	// Consume the initial run of letters and digits (letter-started alphanumeric run).
	// This keeps identifiers like "A4" together as a single word.
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			i += size
		} else {
			break
		}
	}

	// Try to extend with hyphens and apostrophes
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])

		// Hyphen joining: single U+002D, preceded by letter/digit, followed by letter/digit
		if r == '-' {
			next := i + size
			if next < len(s) {
				nr, _ := utf8.DecodeRuneInString(s[next:])
				// Must not be a double hyphen, and next char must be letter/digit
				if nr == '-' {
					break
				}
				if unicode.IsLetter(nr) || unicode.IsDigit(nr) {
					i = next
					i = consumeWordOrDigitRun(s, i)
					continue
				}
			}
			break
		}

		// Apostrophe joining: U+0027, U+2019, U+02BC between letters
		if r == '\'' || r == '\u2019' || r == '\u02BC' {
			next := i + size
			if next < len(s) {
				nr, _ := utf8.DecodeRuneInString(s[next:])
				if unicode.IsLetter(nr) {
					// Check that the preceding character is a letter
					pr, _ := utf8.DecodeLastRuneInString(s[pos:i])
					if unicode.IsLetter(pr) {
						i = next
						// Consume following letters (not digits — apostrophe only joins letters)
						for i < len(s) {
							lr, ls := utf8.DecodeRuneInString(s[i:])
							if !unicode.IsLetter(lr) {
								break
							}
							i += ls
						}
						continue
					}
				}
			}
			break
		}

		break
	}

	return Token{Text: s[pos:i], Start: pos, End: i, Type: Word}
}

// trimTokensForEmail removes any tokens that overlap with the email local part
// starting at emailStart. This handles the case where we already emitted Word
// tokens for the local part before encountering the @ sign.
func trimTokensForEmail(tokens []Token, emailStart int) []Token {
	for len(tokens) > 0 {
		last := tokens[len(tokens)-1]
		if last.Start >= emailStart {
			tokens = tokens[:len(tokens)-1]
		} else if last.End > emailStart {
			// Partial overlap: trim the token
			tokens[len(tokens)-1] = Token{
				Text:  last.Text[:emailStart-last.Start],
				Start: last.Start,
				End:   emailStart,
				Type:  last.Type,
			}
			break
		} else {
			break
		}
	}
	return tokens
}

// consumeWordOrDigitRun consumes a contiguous run of letters and digits.
func consumeWordOrDigitRun(s string, pos int) int {
	for pos < len(s) {
		r, size := utf8.DecodeRuneInString(s[pos:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		pos += size
	}
	return pos
}

// isEmailLocalChar returns true for characters valid in the local part of an email.
func isEmailLocalChar(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	return r == '.' || r == '_' || r == '%' || r == '+' || r == '-'
}

// isEmailDomainChar returns true for characters valid in the domain part of an email.
func isEmailDomainChar(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	return r == '.' || r == '-'
}

// isAllAlpha returns true if every rune in s is an ASCII letter.
func isAllAlpha(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return true
}

// isDigitByte returns true for ASCII digit bytes.
func isDigitByte(b byte) bool {
	return b >= '0' && b <= '9'
}
