package normalize

import (
	"testing"

	"github.com/az-ai-labs/az-lang-nlp/azcase"
)

func FuzzNormalize(f *testing.F) {
	f.Add("gozel soz")
	f.Add("gözəl söz")
	f.Add("seher")
	f.Add("server test hello")
	f.Add("ac")
	f.Add("Azerbaycan")
	f.Add("")
	f.Add("   ")
	f.Add("https://gov.az gozel")
	f.Add("user@mail.az")
	f.Add("123 gozel")
	f.Add("\xff\xfe")
	f.Add("\x00")
	f.Add("sosial-iqtisadi")
	f.Add("Bakı'nın")

	f.Fuzz(func(t *testing.T, s string) {
		result := Normalize(s)

		// Idempotency: applying twice must produce the same result.
		if second := Normalize(result); second != result {
			t.Errorf("not idempotent:\ninput:  %q\nfirst:  %q\nsecond: %q", s, result, second)
		}
	})
}

func FuzzNormalizeWord(f *testing.F) {
	f.Add("gozel")
	f.Add("gözəl")
	f.Add("seher")
	f.Add("server")
	f.Add("ac")
	f.Add("Azerbaycan")
	f.Add("")
	f.Add("e")
	f.Add("\xff\xfe")
	f.Add("\x00")
	f.Add("GOZEL")
	f.Add("kitab")

	f.Fuzz(func(t *testing.T, word string) {
		result := NormalizeWord(word)

		// Idempotency.
		if second := NormalizeWord(result); second != result {
			t.Errorf("not idempotent:\ninput:  %q\nfirst:  %q\nsecond: %q", word, result, second)
		}

		// Rune count must be preserved after NFC composition
		// (diacritics replace 1 rune with 1 rune, but NFC may merge
		// base + combining mark into a single precomposed rune).
		nfcWord := azcase.ComposeNFC(word)
		if len([]rune(result)) != len([]rune(nfcWord)) {
			t.Errorf("rune count changed:\ninput:  %q (nfc runes=%d)\noutput: %q (runes=%d)",
				word, len([]rune(nfcWord)), result, len([]rune(result)))
		}
	})
}
