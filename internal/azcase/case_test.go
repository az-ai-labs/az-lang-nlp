package azcase

import "testing"

func TestLower(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		r    rune
		want rune
	}{
		{"ascii I to dotless", 'I', '\u0131'},
		{"dotted İ to i", '\u0130', 'i'},
		{"lowercase a", 'A', 'a'},
		{"already lowercase", 'b', 'b'},
		{"schwa upper", '\u018f', '\u0259'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Lower(tt.r); got != tt.want {
				t.Errorf("Lower(%q) = %q, want %q", tt.r, got, tt.want)
			}
		})
	}
}

func TestUpper(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		r    rune
		want rune
	}{
		{"i to dotted İ", 'i', '\u0130'},
		{"dotless ı to I", '\u0131', 'I'},
		{"lowercase a", 'a', 'A'},
		{"already upper", 'B', 'B'},
		{"schwa lower", '\u0259', '\u018f'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Upper(tt.r); got != tt.want {
				t.Errorf("Upper(%q) = %q, want %q", tt.r, got, tt.want)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"turkic I", "KITAB", "kıtab"},
		{"dotted İ", "\u0130stanbul", "istanbul"},
		{"mixed", "Az\u0259rbaycan", "az\u0259rbaycan"},
		{"empty", "", ""},
		{"already lower", "kitab", "kitab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ToLower(tt.input); got != tt.want {
				t.Errorf("ToLower(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func BenchmarkToLower_AlreadyLower(b *testing.B) {
	s := "kitablarımızdan"
	for b.Loop() {
		ToLower(s)
	}
}

func BenchmarkToLower_MixedCase(b *testing.B) {
	s := "Kitablarımızdan"
	for b.Loop() {
		ToLower(s)
	}
}

func BenchmarkToLower_AllUpper(b *testing.B) {
	s := "AZƏRBAYCAN"
	for b.Loop() {
		ToLower(s)
	}
}
