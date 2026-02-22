package spell

import "testing"

func FuzzIsCorrect(f *testing.F) {
	f.Add("kitab")
	f.Add("ketab")
	f.Add("gözəl")
	f.Add("gozel")
	f.Add("")
	f.Add("a")
	f.Add("   ")
	f.Add("xyzabc")
	f.Add("sosial-iqtisadi")
	f.Add("Bakı'nın")
	f.Add("\xff\xfe")
	f.Add("\x00")
	f.Add("kitablarımızdan")

	f.Fuzz(func(t *testing.T, word string) {
		// Must not panic on any input.
		_ = IsCorrect(word)
	})
}

func FuzzSuggest(f *testing.F) {
	f.Add("kitab", 2)
	f.Add("ketab", 2)
	f.Add("gozel", 1)
	f.Add("", 0)
	f.Add("xyzabc", 2)
	f.Add("\xff\xfe", 2)
	f.Add("\x00", 1)
	f.Add("a", 2)

	f.Fuzz(func(t *testing.T, word string, maxDist int) {
		// Must not panic on any input.
		_ = Suggest(word, maxDist)
	})
}

func FuzzCorrect(f *testing.F) {
	f.Add("Bu ketab gozeldir")
	f.Add("kitab")
	f.Add("")
	f.Add("   ")
	f.Add("123 ketab")
	f.Add("\xff\xfe")
	f.Add("\x00")
	f.Add("sosial-iqtisadi")

	f.Fuzz(func(t *testing.T, text string) {
		result := Correct(text)

		// Idempotency: correcting already-corrected text should be stable.
		if second := Correct(result); second != result {
			t.Errorf("not idempotent:\ninput:  %q\nfirst:  %q\nsecond: %q", text, result, second)
		}
	})
}
