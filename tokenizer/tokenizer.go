// Package tokenizer splits Azerbaijani text into words, sentences, and
// structured tokens with byte offsets.
//
// The package provides two API layers:
//
//   - Structured: WordTokens and SentenceTokens return []Token with byte
//     offsets and type metadata. The invariant s[t.Start:t.End] == t.Text
//     holds for every token, and concatenating all token texts reconstructs
//     the original string.
//
//   - Convenience: Words and Sentences return []string for common use cases
//     where offsets and types are not needed.
//
// All functions are safe for concurrent use by multiple goroutines.
//
// Known limitations (v1.0):
//
//   - Sentence splitting does not track quote or parenthesis nesting.
//     Terminal punctuation inside quotes may cause false sentence breaks.
//   - Bare URLs without a protocol prefix (www.example.com) are not detected.
//     Only http:// and https:// prefixed URLs are recognized.
//   - Single-letter abbreviations (m., s., d.) are not in the built-in list
//     due to ambiguity with sentence-ending periods.
package tokenizer

import "fmt"

// wordsPerTokenEstimate is the estimated ratio of total tokens to word tokens,
// used to pre-allocate the words slice in the Words convenience function.
const wordsPerTokenEstimate = 2

// TokenType classifies a token.
type TokenType int

const (
	Word        TokenType = iota // Alphabetic word (any script), including hyphens and apostrophes
	Number                       // Digits, with decimal comma or thousand-separator dots
	Punctuation                  // Punctuation marks: . , ! ? : ; ( ) etc.
	Space                        // Contiguous whitespace (spaces, tabs, newlines)
	Symbol                       // Everything else: emoji, CJK, mathematical symbols, etc.
	URL                          // http:// or https:// prefixed sequences
	Email                        // user@domain.tld sequences
	Sentence                     // Used only by SentenceTokens — a full sentence
)

// String returns the name of the token type.
func (t TokenType) String() string {
	switch t {
	case Word:
		return "Word"
	case Number:
		return "Number"
	case Punctuation:
		return "Punctuation"
	case Space:
		return "Space"
	case Symbol:
		return "Symbol"
	case URL:
		return "URL"
	case Email:
		return "Email"
	case Sentence:
		return "Sentence"
	default:
		return fmt.Sprintf("TokenType(%d)", int(t))
	}
}

// Token represents a unit of text with its position and classification.
type Token struct {
	Text  string    // The token text
	Start int       // Byte offset in the original string (inclusive)
	End   int       // Byte offset in the original string (exclusive)
	Type  TokenType // Classification of the token
}

// String returns a debug representation, e.g. Word("salam")[0:5].
func (t Token) String() string {
	return fmt.Sprintf("%s(%q)[%d:%d]", t.Type, t.Text, t.Start, t.End)
}

// WordTokens splits text into all tokens with metadata.
// Returns Word, Number, Punctuation, Space, Symbol, URL, and Email tokens.
// The byte offset invariant s[t.Start:t.End] == t.Text holds for every token.
// Concatenating all token texts reconstructs the original string.
func WordTokens(s string) []Token {
	if s == "" {
		return nil
	}
	return wordTokens(s)
}

// Words returns only Word-type token texts from the text.
// Does not include Number, Punctuation, URL, Email, or other types.
// For full control, use WordTokens and filter by Type.
func Words(s string) []string {
	if s == "" {
		return nil
	}
	tokens := wordTokens(s)
	words := make([]string, 0, len(tokens)/wordsPerTokenEstimate)
	for _, t := range tokens {
		if t.Type == Word {
			words = append(words, t.Text)
		}
	}
	return words
}

// SentenceTokens splits text into sentence-level tokens with byte offsets.
// Each returned Token has Type=Sentence.
// Sentence boundaries are determined by terminal punctuation (. ? !) followed
// by whitespace and an uppercase letter, or by double newlines.
// A built-in abbreviation list prevents false breaks after common abbreviations.
func SentenceTokens(s string) []Token {
	if s == "" {
		return nil
	}
	return sentenceTokens(s)
}

// Sentences returns sentence strings from the text.
func Sentences(s string) []string {
	if s == "" {
		return nil
	}
	tokens := sentenceTokens(s)
	sentences := make([]string, len(tokens))
	for i, t := range tokens {
		sentences[i] = t.Text
	}
	return sentences
}
