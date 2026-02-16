package tokenizer

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/BarsNLP/barsnlp/translit"
)

// verifyInvariants checks two invariants that must hold for every tokenization:
//   - Byte offset invariant: input[t.Start:t.End] == t.Text for every token.
//   - Reconstruction invariant: concatenating all token texts reproduces the input.
func verifyInvariants(t *testing.T, input string, tokens []Token) {
	t.Helper()
	for i, tok := range tokens {
		if got := input[tok.Start:tok.End]; got != tok.Text {
			t.Errorf("token %d offset invariant broken: input[%d:%d]=%q, Text=%q",
				i, tok.Start, tok.End, got, tok.Text)
		}
	}
	var buf strings.Builder
	for _, tok := range tokens {
		buf.WriteString(tok.Text)
	}
	if buf.String() != input {
		t.Errorf("reconstruction invariant broken:\ngot:  %q\nwant: %q", buf.String(), input)
	}
}

// ---------------------------------------------------------------------------
// WordTokens — comprehensive table-driven tests
// ---------------------------------------------------------------------------

func TestWordTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []Token
	}{
		// -- Basic word tokens --

		{"simple ASCII word", "hello", []Token{
			{Text: "hello", Start: 0, End: 5, Type: Word},
		}},
		{"two words", "foo bar", []Token{
			{Text: "foo", Start: 0, End: 3, Type: Word},
			{Text: " ", Start: 3, End: 4, Type: Space},
			{Text: "bar", Start: 4, End: 7, Type: Word},
		}},
		{"Azerbaijani special chars", "\u0259\u015f\u00f6\u00fc\u011f\u00e7\u0131\u0130", []Token{
			{Text: "\u0259\u015f\u00f6\u00fc\u011f\u00e7\u0131\u0130", Start: 0, End: 16, Type: Word},
		}},

		// -- Number tokens --

		{"plain digits", "42", []Token{
			{Text: "42", Start: 0, End: 2, Type: Number},
		}},
		{"thousand separator", "1.000.000", []Token{
			{Text: "1.000.000", Start: 0, End: 9, Type: Number},
		}},
		{"decimal comma", "3,14", []Token{
			{Text: "3,14", Start: 0, End: 4, Type: Number},
		}},
		{"dot not decimal (two digits after dot)", "3.14", []Token{
			{Text: "3", Start: 0, End: 1, Type: Number},
			{Text: ".", Start: 1, End: 2, Type: Punctuation},
			{Text: "14", Start: 2, End: 4, Type: Number},
		}},
		{"trailing comma not decimal", "3,", []Token{
			{Text: "3", Start: 0, End: 1, Type: Number},
			{Text: ",", Start: 1, End: 2, Type: Punctuation},
		}},
		{"sign is separate token", "-5", []Token{
			{Text: "-", Start: 0, End: 1, Type: Punctuation},
			{Text: "5", Start: 1, End: 2, Type: Number},
		}},

		// -- Invalid thousand grouping --

		{"invalid thousand grouping splits", "1.00.0", []Token{
			{Text: "1", Start: 0, End: 1, Type: Number},
			{Text: ".", Start: 1, End: 2, Type: Punctuation},
			{Text: "00", Start: 2, End: 4, Type: Number},
			{Text: ".", Start: 4, End: 5, Type: Punctuation},
			{Text: "0", Start: 5, End: 6, Type: Number},
		}},

		// -- Number-unit split --

		{"number-unit split", "5km", []Token{
			{Text: "5", Start: 0, End: 1, Type: Number},
			{Text: "km", Start: 1, End: 3, Type: Word},
		}},

		// -- Punctuation --

		{"single punctuation marks", ".,!?", []Token{
			{Text: ".", Start: 0, End: 1, Type: Punctuation},
			{Text: ",", Start: 1, End: 2, Type: Punctuation},
			{Text: "!", Start: 2, End: 3, Type: Punctuation},
			{Text: "?", Start: 3, End: 4, Type: Punctuation},
		}},
		{"parentheses", "(a)", []Token{
			{Text: "(", Start: 0, End: 1, Type: Punctuation},
			{Text: "a", Start: 1, End: 2, Type: Word},
			{Text: ")", Start: 2, End: 3, Type: Punctuation},
		}},

		// -- Whitespace merging --

		{"multiple spaces merge", "a  \t\n b", []Token{
			{Text: "a", Start: 0, End: 1, Type: Word},
			{Text: "  \t\n ", Start: 1, End: 6, Type: Space},
			{Text: "b", Start: 6, End: 7, Type: Word},
		}},

		// -- Symbol tokens --

		{"emoji produces symbol tokens", "\U0001F3D9\uFE0F", []Token{
			{Text: "\U0001F3D9", Start: 0, End: 4, Type: Symbol},
			{Text: "\uFE0F", Start: 4, End: 7, Type: Symbol},
		}},
		{"CJK characters are letters", "\u4E2D\u6587", []Token{
			{Text: "\u4E2D\u6587", Start: 0, End: 6, Type: Word},
		}},
		{"dollar sign is symbol", "$", []Token{
			{Text: "$", Start: 0, End: 1, Type: Symbol},
		}},
		{"math symbol", "\u00b1", []Token{
			{Text: "\u00b1", Start: 0, End: 2, Type: Symbol},
		}},
		{"percent is punctuation", "%", []Token{
			{Text: "%", Start: 0, End: 1, Type: Punctuation},
		}},

		// -- Hyphen joining --

		{"hyphen between letters", "sosial-iqtisadi", []Token{
			{Text: "sosial-iqtisadi", Start: 0, End: 15, Type: Word},
		}},
		{"hyphen letter-digit", "F-16", []Token{
			{Text: "F-16", Start: 0, End: 4, Type: Word},
		}},
		{"hyphen digit-letter", "COVID-19", []Token{
			{Text: "COVID-19", Start: 0, End: 8, Type: Word},
		}},

		// -- Hyphen NOT joining --

		{"leading hyphen", "-test", []Token{
			{Text: "-", Start: 0, End: 1, Type: Punctuation},
			{Text: "test", Start: 1, End: 5, Type: Word},
		}},
		{"trailing hyphen", "test-", []Token{
			{Text: "test", Start: 0, End: 4, Type: Word},
			{Text: "-", Start: 4, End: 5, Type: Punctuation},
		}},
		{"double hyphen splits", "test--word", []Token{
			{Text: "test", Start: 0, End: 4, Type: Word},
			{Text: "--", Start: 4, End: 6, Type: Punctuation},
			{Text: "word", Start: 6, End: 10, Type: Word},
		}},
		{"en-dash splits", "test\u2013word", []Token{
			{Text: "test", Start: 0, End: 4, Type: Word},
			{Text: "\u2013", Start: 4, End: 7, Type: Punctuation},
			{Text: "word", Start: 7, End: 11, Type: Word},
		}},
		{"em-dash splits", "test\u2014word", []Token{
			{Text: "test", Start: 0, End: 4, Type: Word},
			{Text: "\u2014", Start: 4, End: 7, Type: Punctuation},
			{Text: "word", Start: 7, End: 11, Type: Word},
		}},

		// -- Apostrophe joining --

		{"apostrophe U+0027 joins", "Bak\u0131'n\u0131n", []Token{
			{Text: "Bak\u0131'n\u0131n", Start: 0, End: 10, Type: Word},
		}},
		{"right single quote U+2019 joins", "Bak\u0131\u2019n\u0131n", []Token{
			{Text: "Bak\u0131\u2019n\u0131n", Start: 0, End: 12, Type: Word},
		}},
		{"modifier letter apostrophe U+02BC joins", "Bak\u0131\u02BCn\u0131n", []Token{
			{Text: "Bak\u0131\u02BCn\u0131n", Start: 0, End: 11, Type: Word},
		}},

		// -- Apostrophe NOT joining --

		{"leading apostrophe", "'test", []Token{
			{Text: "'", Start: 0, End: 1, Type: Punctuation},
			{Text: "test", Start: 1, End: 5, Type: Word},
		}},
		{"trailing apostrophe", "test'", []Token{
			{Text: "test", Start: 0, End: 4, Type: Word},
			{Text: "'", Start: 4, End: 5, Type: Punctuation},
		}},
		{"quoted word", "'test'", []Token{
			{Text: "'", Start: 0, End: 1, Type: Punctuation},
			{Text: "test", Start: 1, End: 5, Type: Word},
			{Text: "'", Start: 5, End: 6, Type: Punctuation},
		}},

		// -- URL detection --

		{"https URL", "https://gov.az/doc", []Token{
			{Text: "https://gov.az/doc", Start: 0, End: 18, Type: URL},
		}},
		{"http URL", "http://example.com", []Token{
			{Text: "http://example.com", Start: 0, End: 18, Type: URL},
		}},
		{"URL with trailing punctuation stripped", "https://gov.az.", []Token{
			{Text: "https://gov.az", Start: 0, End: 14, Type: URL},
			{Text: ".", Start: 14, End: 15, Type: Punctuation},
		}},

		// -- Email detection --

		{"simple email", "user@mail.az", []Token{
			{Text: "user@mail.az", Start: 0, End: 12, Type: Email},
		}},
		{"complex email", "test.user+tag@domain.co.az", []Token{
			{Text: "test.user+tag@domain.co.az", Start: 0, End: 26, Type: Email},
		}},
		{"email with trailing dot", "user@mail.az.", []Token{
			{Text: "user@mail.az", Start: 0, End: 12, Type: Email},
			{Text: ".", Start: 12, End: 13, Type: Punctuation},
		}},

		// -- Leading-dot email rejection --

		{"leading dot email rejected", ".user@mail.az", []Token{
			{Text: ".", Start: 0, End: 1, Type: Punctuation},
			{Text: "user@mail.az", Start: 1, End: 13, Type: Email},
		}},

		// -- Bare protocol edge cases --

		{"bare http protocol only", "http://", []Token{
			{Text: "http", Start: 0, End: 4, Type: Word},
			{Text: ":", Start: 4, End: 5, Type: Punctuation},
			{Text: "/", Start: 5, End: 6, Type: Punctuation},
			{Text: "/", Start: 6, End: 7, Type: Punctuation},
		}},

		// -- Mixed content --

		{"mixed content sentence", "Prof. \u018eliyev 1.000 manat \u00f6d\u0259di.", []Token{
			{Text: "Prof", Start: 0, End: 4, Type: Word},
			{Text: ".", Start: 4, End: 5, Type: Punctuation},
			{Text: " ", Start: 5, End: 6, Type: Space},
			{Text: "\u018eliyev", Start: 6, End: 13, Type: Word},
			{Text: " ", Start: 13, End: 14, Type: Space},
			{Text: "1.000", Start: 14, End: 19, Type: Number},
			{Text: " ", Start: 19, End: 20, Type: Space},
			{Text: "manat", Start: 20, End: 25, Type: Word},
			{Text: " ", Start: 25, End: 26, Type: Space},
			{Text: "\u00f6d\u0259di", Start: 26, End: 33, Type: Word},
			{Text: ".", Start: 33, End: 34, Type: Punctuation},
		}},

		// -- Edge cases --

		{"empty string", "", nil},
		{"whitespace only", "   ", []Token{
			{Text: "   ", Start: 0, End: 3, Type: Space},
		}},
		{"single ASCII character", "a", []Token{
			{Text: "a", Start: 0, End: 1, Type: Word},
		}},
		{"single multi-byte rune", "\u0259", []Token{
			{Text: "\u0259", Start: 0, End: 2, Type: Word},
		}},

		// -- Non-ASCII Unicode digits (must not hang) --

		{"mathematical digit U+1D7E2 is symbol", "\U0001D7E2", []Token{
			{Text: "\U0001D7E2", Start: 0, End: 4, Type: Symbol},
		}},
		{"Arabic-Indic digit U+0660 is symbol", "\u0660", []Token{
			{Text: "\u0660", Start: 0, End: 2, Type: Symbol},
		}},
		{"non-ASCII digit absorbed into word", "a\U0001D7E2b", []Token{
			{Text: "a\U0001D7E2b", Start: 0, End: 6, Type: Word},
		}},

		// -- Malformed UTF-8 --

		{"malformed UTF-8 produces symbol tokens", "\xff\xfe", []Token{
			{Text: "\xff", Start: 0, End: 1, Type: Symbol},
			{Text: "\xfe", Start: 1, End: 2, Type: Symbol},
		}},

		// -- Mixed Latin + Cyrillic --

		{"mixed Latin and Cyrillic", "Bu \u041c\u043e\u0441\u043a\u0432\u0430-\u0434\u0430\u043d g\u0259ldi", []Token{
			{Text: "Bu", Start: 0, End: 2, Type: Word},
			{Text: " ", Start: 2, End: 3, Type: Space},
			{Text: "\u041c\u043e\u0441\u043a\u0432\u0430-\u0434\u0430\u043d", Start: 3, End: 22, Type: Word},
			{Text: " ", Start: 22, End: 23, Type: Space},
			{Text: "g\u0259ldi", Start: 23, End: 29, Type: Word},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WordTokens(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("WordTokens(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("WordTokens(%q): got %d tokens, want %d\ngot:  %v\nwant: %v",
					tt.input, len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("token %d: got %v, want %v", i, got[i], tt.want[i])
				}
			}
			verifyInvariants(t, tt.input, got)
		})
	}
}

// TestWordTokensLargeInput verifies that a large input does not panic
// and produces a non-empty result.
func TestWordTokensLargeInput(t *testing.T) {
	chunk := "Salam d\u00fcnya! Az\u0259rbaycan. "
	input := strings.Repeat(chunk, 50000) // > 1MB
	tokens := WordTokens(input)
	if len(tokens) == 0 {
		t.Error("expected non-empty token list for large input")
	}
	verifyInvariants(t, input, tokens)
}

// ---------------------------------------------------------------------------
// Words — convenience wrapper tests
// ---------------------------------------------------------------------------

func TestWords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"basic words", "Salam, d\u00fcnya!", []string{"Salam", "d\u00fcnya"}},
		{"numbers excluded", "5km test", []string{"km", "test"}},
		{"URLs excluded", "https://gov.az test", []string{"test"}},
		{"empty string", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Words(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("Words(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("Words(%q): got %d words, want %d\ngot:  %v\nwant: %v",
					tt.input, len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("word %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SentenceTokens — sentence splitting tests
// ---------------------------------------------------------------------------

func TestSentenceTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []Token
	}{
		// -- Basic sentence splitting --

		{"basic two sentences", "Birinci. \u0130kinci.", []Token{
			{Text: "Birinci.", Start: 0, End: 8, Type: Sentence},
			{Text: " \u0130kinci.", Start: 8, End: 17, Type: Sentence},
		}},

		// -- Abbreviation handling --

		{"abbreviation not split", "Prof. \u018eliyev g\u0259ldi.", []Token{
			{Text: "Prof. \u018eliyev g\u0259ldi.", Start: 0, End: 21, Type: Sentence},
		}},
		{"multi-part abbreviation", "Az.R. qanunu.", []Token{
			{Text: "Az.R. qanunu.", Start: 0, End: 13, Type: Sentence},
		}},
		{"multi-word abbreviation", "Kitablar v\u0259 s. sat\u0131ld\u0131.", []Token{
			{Text: "Kitablar v\u0259 s. sat\u0131ld\u0131.", Start: 0, End: 26, Type: Sentence},
		}},

		// -- Ellipsis --

		{"ellipsis splits", "Ola bil\u0259r... B\u0259lk\u0259.", []Token{
			{Text: "Ola bil\u0259r...", Start: 0, End: 13, Type: Sentence},
			{Text: " B\u0259lk\u0259.", Start: 13, End: 22, Type: Sentence},
		}},
		{"unicode ellipsis splits", "Ola bil\u0259r\u2026 B\u0259lk\u0259.", []Token{
			{Text: "Ola bil\u0259r\u2026", Start: 0, End: 13, Type: Sentence},
			{Text: " B\u0259lk\u0259.", Start: 13, End: 22, Type: Sentence},
		}},

		// -- Double newline --

		{"double newline splits", "Birinci\n\n\u0130kinci", []Token{
			{Text: "Birinci\n\n", Start: 0, End: 9, Type: Sentence},
			{Text: "\u0130kinci", Start: 9, End: 16, Type: Sentence},
		}},

		// -- No terminal punctuation --

		{"no terminal punctuation", "Bu test m\u0259tndir", []Token{
			{Text: "Bu test m\u0259tndir", Start: 0, End: 16, Type: Sentence},
		}},

		// -- Punctuation clusters --

		{"punctuation cluster splits", "N\u0259?! Do\u011fru.", []Token{
			{Text: "N\u0259?!", Start: 0, End: 5, Type: Sentence},
			{Text: " Do\u011fru.", Start: 5, End: 13, Type: Sentence},
		}},

		// -- Abbreviation at end --

		{"abbreviation at end of input", "O, prof.", []Token{
			{Text: "O, prof.", Start: 0, End: 8, Type: Sentence},
		}},

		// -- Single newline does not split --

		{"single newline no split", "Birinci\n\u0130kinci", []Token{
			{Text: "Birinci\n\u0130kinci", Start: 0, End: 15, Type: Sentence},
		}},

		// -- Empty string --

		{"empty string", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SentenceTokens(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("SentenceTokens(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("SentenceTokens(%q): got %d sentences, want %d\ngot:  %v\nwant: %v",
					tt.input, len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("sentence %d: got %v, want %v", i, got[i], tt.want[i])
				}
			}
			// Verify sentence invariants
			verifyInvariants(t, tt.input, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Sentences — convenience wrapper tests
// ---------------------------------------------------------------------------

func TestSentences(t *testing.T) {
	got := Sentences("Birinci. \u0130kinci.")
	want := []string{"Birinci.", " \u0130kinci."}
	if len(got) != len(want) {
		t.Fatalf("Sentences: got %d, want %d\ngot:  %v\nwant: %v",
			len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("sentence %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// TokenType.String
// ---------------------------------------------------------------------------

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		tt   TokenType
		want string
	}{
		{Word, "Word"},
		{Number, "Number"},
		{Punctuation, "Punctuation"},
		{Space, "Space"},
		{Symbol, "Symbol"},
		{URL, "URL"},
		{Email, "Email"},
		{Sentence, "Sentence"},
		{TokenType(99), "TokenType(99)"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.tt.String(); got != tt.want {
				t.Errorf("TokenType(%d).String() = %q, want %q", int(tt.tt), got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Token.String
// ---------------------------------------------------------------------------

func TestTokenString(t *testing.T) {
	tok := Token{Text: "salam", Start: 0, End: 5, Type: Word}
	want := `Word("salam")[0:5]`
	if got := tok.String(); got != want {
		t.Errorf("Token.String() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Integration with translit
// ---------------------------------------------------------------------------

func TestTranslitIntegration(t *testing.T) {
	input := translit.CyrillicToLatin("\u0411\u0430\u043a\u044b \u0448\u04d9\u04bb\u04d9\u0440\u0438 \u0433\u04e9\u0437\u04d9\u043b\u0434\u0438\u0440.")
	tokens := WordTokens(input)
	verifyInvariants(t, input, tokens)
	words := Words(input)
	if len(words) == 0 {
		t.Error("expected words from transliterated text")
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkWordTokens(b *testing.B) {
	input := strings.Repeat("Prof. \u018eliyev 1.000 manat \u00f6d\u0259di. Bak\u0131\u2019n\u0131n k\u00fc\u00e7\u0259l\u0259ri g\u00f6z\u0259ldir! ", 1000)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		WordTokens(input)
	}
}

func BenchmarkWords(b *testing.B) {
	input := strings.Repeat("Prof. \u018eliyev 1.000 manat \u00f6d\u0259di. Bak\u0131\u2019n\u0131n k\u00fc\u00e7\u0259l\u0259ri g\u00f6z\u0259ldir! ", 1000)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		Words(input)
	}
}

func BenchmarkSentenceTokens(b *testing.B) {
	input := strings.Repeat("Prof. \u018eliyev 1.000 manat \u00f6d\u0259di. Bak\u0131\u2019n\u0131n k\u00fc\u00e7\u0259l\u0259ri g\u00f6z\u0259ldir! ", 1000)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		SentenceTokens(input)
	}
}

func BenchmarkSentences(b *testing.B) {
	input := strings.Repeat("Prof. \u018eliyev 1.000 manat \u00f6d\u0259di. Bak\u0131\u2019n\u0131n k\u00fc\u00e7\u0259l\u0259ri g\u00f6z\u0259ldir! ", 1000)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		Sentences(input)
	}
}

// ---------------------------------------------------------------------------
// Examples
// ---------------------------------------------------------------------------

func ExampleWordTokens() {
	tokens := WordTokens("Salam, d\u00fcnya!")
	for _, t := range tokens {
		fmt.Printf("%s: %q\n", t.Type, t.Text)
	}
	// Output:
	// Word: "Salam"
	// Punctuation: ","
	// Space: " "
	// Word: "dünya"
	// Punctuation: "!"
}

func ExampleWords() {
	fmt.Println(Words("Bak\u0131'n\u0131n k\u00fc\u00e7\u0259l\u0259ri g\u00f6z\u0259ldir."))
	// Output:
	// [Bakı'nın küçələri gözəldir]
}

func ExampleSentenceTokens() {
	tokens := SentenceTokens("Birinci c\u00fcml\u0259. \u0130kinci c\u00fcml\u0259.")
	for _, t := range tokens {
		fmt.Printf("%q\n", t.Text)
	}
	// Output:
	// "Birinci cümlə."
	// " İkinci cümlə."
}

func ExampleSentences() {
	fmt.Println(Sentences("Birinci. \u0130kinci."))
	// Output:
	// [Birinci.  İkinci.]
}

// ---------------------------------------------------------------------------
// Fuzz tests
// ---------------------------------------------------------------------------

func FuzzWordTokens(f *testing.F) {
	f.Add("Salam, d\u00fcnya!")
	f.Add("user@mail.az")
	f.Add("https://gov.az")
	f.Add("1.000.000,50")
	f.Add("")
	f.Add("\xff\xfe")
	f.Add("h h h h h h h h")
	f.Add(".user@domain.com")
	f.Fuzz(func(t *testing.T, s string) {
		tokens := WordTokens(s)
		verifyInvariants(t, s, tokens)
	})
}

func FuzzSentenceTokens(f *testing.F) {
	f.Add("Birinci. \u0130kinci.")
	f.Add("Prof. \u018eliyev g\u0259ldi.")
	f.Add("Ola bil\u0259r... B\u0259lk\u0259.")
	f.Add("")
	f.Add("Az.R. qanunu.")
	f.Fuzz(func(t *testing.T, s string) {
		tokens := SentenceTokens(s)
		verifyInvariants(t, s, tokens)
	})
}

// ---------------------------------------------------------------------------
// Concurrent safety
// ---------------------------------------------------------------------------

func TestConcurrentSafety(t *testing.T) {
	input := "Prof. \u018eliyev 1.000 manat \u00f6d\u0259di. user@mail.az https://gov.az"
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			WordTokens(input)
			Words(input)
			SentenceTokens(input)
			Sentences(input)
		})
	}
	wg.Wait()
}
