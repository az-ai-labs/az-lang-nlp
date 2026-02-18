package morph

import (
	"sort"
	"testing"
)

func TestIsKnownStem(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"noun kitab", "kitab", true},
		{"verb gəl", "gəl", true},
		{"adj gözəl", "gözəl", true},
		{"short stem ev", "ev", true},
		{"short stem su", "su", true},
		{"prefix of real word", "kita", false},
		{"nonexistent", "xyznonexistent", false},
		{"empty", "", false},
		{"ana", "ana", true},
		{"baba", "baba", true},
		{"gün", "gün", true},
		{"gecə", "gecə", true},
		{"sevgi", "sevgi", true},
		{"dəniz", "dəniz", true},
		{"alma", "alma", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isKnownStem(tt.input); got != tt.want {
				t.Errorf("isKnownStem(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestStemPOS(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  byte
	}{
		{"noun kitab", "kitab", 'N'},
		{"verb gəl", "gəl", 'V'},
		{"adj gözəl (also noun, A wins alphabetically)", "gözəl", 'A'},
		{"noun dəniz", "dəniz", 'N'},
		{"not found", "xyznotfound", 0},
		{"empty", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stemPOS(tt.input); got != tt.want {
				t.Errorf("stemPOS(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDictIntegrity(t *testing.T) {
	const minEntries = 10000
	if len(dictLemmas) < minEntries {
		t.Fatalf("dictionary has %d entries, want at least %d", len(dictLemmas), minEntries)
	}
	if !sort.StringsAreSorted(dictLemmas) {
		t.Fatal("dictLemmas is not sorted")
	}
}

func BenchmarkIsKnownStem(b *testing.B) {
	for b.Loop() {
		isKnownStem("kitab")
	}
}
