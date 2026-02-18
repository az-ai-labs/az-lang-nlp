package morph

import "testing"

func TestTryRestoreVowelDrop(t *testing.T) {
	tests := []struct {
		stem string
		want string
	}{
		// Successful restorations (FSM stems with suffix stripped).
		{"oğl", "oğul"},   // oğlum → oğl + um
		{"burn", "burun"},  // burnum → burn + um
		{"ağz", "ağız"},    // ağzım → ağz + ım
		{"aln", "alın"},    // alnı → aln + ı
		{"beyn", "beyin"},  // beyni → beyn + i
		{"ömr", "ömür"},    // ömrü → ömr + ü
		{"şəhr", "şəhər"}, // şəhri → şəhr + i
		{"boyn", "boyun"},  // boynu → boyn + u

		// No restoration needed / no dictionary match.
		{"kitab", ""},
		{"ev", ""},
		{"gəl", ""},
		{"", ""},

		// Single rune — too short.
		{"a", ""},

		// No consonant cluster — no restoration attempted.
		{"ana", ""},
		{"baba", ""},

		// All consonants with no preceding vowel — rejected.
		{"str", ""},

		// Valid stem without vowel drop — no restoration needed.
		{"dərd", ""},

		// Too short consonant-only — rejected.
		{"qr", ""},

		// No consonant cluster to split — no restoration.
		{"göz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.stem, func(t *testing.T) {
			got := tryRestoreVowelDrop(tt.stem)
			if got != tt.want {
				t.Errorf("tryRestoreVowelDrop(%q) = %q, want %q", tt.stem, got, tt.want)
			}
		})
	}
}

func BenchmarkTryRestoreVowelDrop(b *testing.B) {
	stems := []string{"oğl", "burn", "ağz", "aln", "beyn", "kitab", "ev", "str"}
	for b.Loop() {
		for _, s := range stems {
			tryRestoreVowelDrop(s)
		}
	}
}
