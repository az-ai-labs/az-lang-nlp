package keywords

import (
	"slices"
	"testing"
)

func FuzzExtractTFIDF(f *testing.F) {
	f.Add("kitab")
	f.Add("Azərbaycan iqtisadiyyatı inkişaf edir")
	f.Add("")
	f.Add("a")
	f.Add("və bu da o")
	f.Add("\xff\xfe")
	f.Add("\x00")
	f.Add("sosial-iqtisadi")

	f.Fuzz(func(t *testing.T, text string) {
		a := ExtractTFIDF(text, 5)
		b := ExtractTFIDF(text, 5)
		if !slices.Equal(a, b) {
			t.Errorf("non-deterministic:\n  a = %v\n  b = %v", a, b)
		}
	})
}

func FuzzExtractTextRank(f *testing.F) {
	f.Add("kitab")
	f.Add("Azərbaycan iqtisadiyyatı inkişaf edir")
	f.Add("")
	f.Add("a")
	f.Add("və bu da o")
	f.Add("\xff\xfe")
	f.Add("\x00")
	f.Add("sosial-iqtisadi")

	f.Fuzz(func(t *testing.T, text string) {
		a := ExtractTextRank(text, 5)
		b := ExtractTextRank(text, 5)
		if !slices.Equal(a, b) {
			t.Errorf("non-deterministic:\n  a = %v\n  b = %v", a, b)
		}
	})
}

func FuzzKeywords(f *testing.F) {
	f.Add("kitab")
	f.Add("Azərbaycan iqtisadiyyatı inkişaf edir")
	f.Add("")
	f.Add("a")
	f.Add("və bu da o")
	f.Add("\xff\xfe")
	f.Add("\x00")
	f.Add("sosial-iqtisadi")

	f.Fuzz(func(t *testing.T, text string) {
		a := Keywords(text)
		b := Keywords(text)
		if !slices.Equal(a, b) {
			t.Errorf("non-deterministic:\n  a = %v\n  b = %v", a, b)
		}
	})
}
