package azcase

import "testing"

func TestComposeNFC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"already NFC", "kitab", "kitab"},
		{"empty", "", ""},
		{"ascii only", "hello world", "hello world"},
		{"o diaeresis lower", "o\u0308n", "\u00f6n"},
		{"u diaeresis lower", "u\u0308z", "\u00fcz"},
		{"c cedilla lower", "c\u0327ay", "\u00e7ay"},
		{"s cedilla lower", "s\u0327\u0259h\u0259r", "\u015f\u0259h\u0259r"},
		{"g breve lower", "g\u0306\u00f6z\u0259l", "\u011f\u00f6z\u0259l"},
		{"I dot above", "I\u0307stanbul", "\u0130stanbul"},
		{"O diaeresis upper", "O\u0308n", "\u00d6n"},
		{"U diaeresis upper", "U\u0308lk\u0259", "\u00dclk\u0259"},
		{"C cedilla upper", "C\u0327ay", "\u00c7ay"},
		{"S cedilla upper", "S\u0327\u0259h\u0259r", "\u015e\u0259h\u0259r"},
		{"G breve upper", "G\u0306\u00f6z\u0259l", "\u011e\u00f6z\u0259l"},
		{"mixed NFC and NFD", "g\u0306\u00f6z\u0259l \u00e7ay", "\u011f\u00f6z\u0259l \u00e7ay"},
		{"no azerbaijani combiners", "caf\u0301e", "caf\u0301e"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ComposeNFC(tt.input); got != tt.want {
				t.Errorf("ComposeNFC(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func BenchmarkComposeNFC_AlreadyNFC(b *testing.B) {
	s := "Azərbaycan gözəl ölkədir"
	for b.Loop() {
		ComposeNFC(s)
	}
}

func BenchmarkComposeNFC_HasCombiners(b *testing.B) {
	s := "go\u0308z\u0259l o\u0308lk\u0259"
	for b.Loop() {
		ComposeNFC(s)
	}
}
