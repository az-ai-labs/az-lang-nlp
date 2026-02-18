// Tests for the numtext package: Convert, ConvertOrdinal, ConvertFloat, Parse.
package numtext

import (
	"fmt"
	"testing"
)

func TestConvert(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input int64
		want  string
	}{
		{"zero", 0, "sıfır"},
		{"one", 1, "bir"},
		{"nine", 9, "doqquz"},
		{"ten", 10, "on"},
		{"eleven", 11, "on bir"},
		{"nineteen", 19, "on doqquz"},
		{"twenty", 20, "iyirmi"},
		{"twenty-one", 21, "iyirmi bir"},
		{"forty-two", 42, "qırx iki"},
		{"ninety-nine", 99, "doxsan doqquz"},
		{"hundred", 100, "yüz"},
		{"hundred one", 101, "yüz bir"},
		{"two hundred", 200, "iki yüz"},
		{"three hundred fifty", 350, "üç yüz əlli"},
		{"nine hundred ninety-nine", 999, "doqquz yüz doxsan doqquz"},
		{"thousand", 1000, "min"},
		{"thousand one", 1001, "min bir"},
		{"two thousand", 2000, "iki min"},
		{"ten thousand", 10000, "on min"},
		{"hundred thousand", 100000, "yüz min"},
		{"million", 1000000, "bir milyon"},
		{"two million three hundred thousand ninety-five", 2300095, "iki milyon üç yüz min doxsan beş"},
		{"billion", 1000000000, "bir milyard"},
		{"trillion", 1_000_000_000_000, "bir trilyon"},
		{"kvintilyon", 1_000_000_000_000_000_000, "bir kvintilyon"},
		{"negative one", -1, "mənfi bir"},
		{"negative thousand", -1000, "mənfi min"},
		{"out of range positive", 1_000_000_000_000_000_001, ""},
		{"out of range negative", -1_000_000_000_000_000_001, ""},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Convert(tt.input)
			if got != tt.want {
				t.Errorf("Convert(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertOrdinal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input int64
		want  string
	}{
		{"one", 1, "birinci"},
		{"two", 2, "ikinci"},
		{"three", 3, "üçüncü"},
		{"four", 4, "dördüncü"},
		{"five", 5, "beşinci"},
		{"six", 6, "altıncı"},
		{"seven", 7, "yeddinci"},
		{"eight", 8, "səkkizinci"},
		{"nine", 9, "doqquzuncu"},
		{"ten", 10, "onuncu"},
		{"twenty", 20, "iyirminci"},
		{"thirty", 30, "otuzuncu"},
		{"forty", 40, "qırxıncı"},
		{"fifty", 50, "əllinci"},
		{"hundred", 100, "yüzüncü"},
		{"thousand", 1000, "mininci"},
		{"zero", 0, "sıfırıncı"},
		{"negative five", -5, "mənfi beşinci"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ConvertOrdinal(tt.input)
			if got != tt.want {
				t.Errorf("ConvertOrdinal(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertFloat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		mode  Mode
		want  string
	}{
		{"math 3.14", "3.14", MathMode, "üç tam yüzdə on dörd"},
		{"digit 3.14", "3.14", DigitMode, "üç vergül bir dörd"},
		{"math 0.5", "0.5", MathMode, "sıfır tam onda beş"},
		{"math 3.05", "3.05", MathMode, "üç tam yüzdə beş"},
		{"math 3.50", "3.50", MathMode, "üç tam yüzdə əlli"},
		{"math 3.0", "3.0", MathMode, "üç tam onda sıfır"},
		{"negative decimal", "-2.5", MathMode, "mənfi iki tam onda beş"},
		{"comma separator", "3,14", MathMode, "üç tam yüzdə on dörd"},
		{"no decimal", "3", MathMode, "üç"},
		{"no decimal mode ignored", "3", DigitMode, "üç"},
		{"empty", "", MathMode, ""},
		{"invalid", "abc", MathMode, ""},
		{"leading dot", ".5", MathMode, "sıfır tam onda beş"},
		{"digit 0.123", "0.123", DigitMode, "sıfır vergül bir iki üç"},
		{"negative zero math", "-0.0", MathMode, "sıfır tam onda sıfır"},
		{"negative zero digit", "-0.0", DigitMode, "sıfır vergül sıfır"},
		{"negative zero point five", "-0.5", MathMode, "mənfi sıfır tam onda beş"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ConvertFloat(tt.input, tt.mode)
			if got != tt.want {
				t.Errorf("ConvertFloat(%q, %v) = %q, want %q", tt.input, tt.mode, got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"zero", "sıfır", 0, false},
		{"one", "bir", 1, false},
		{"hundred", "yüz", 100, false},
		{"bir yüz lenient", "bir yüz", 100, false},
		{"thousand", "min", 1000, false},
		{"bir min lenient", "bir min", 1000, false},
		{"million", "bir milyon", 1000000, false},
		{"compound", "iki milyon üç yüz min doxsan beş", 2300095, false},
		{"negative", "mənfi beş", -5, false},
		{"whitespace", "  yüz   iyirmi   üç  ", 123, false},
		{"empty", "", 0, true},
		{"unknown word", "hello", 0, true},
		{"ordinal rejected", "beşinci", 0, true},
		{"overflow multiplication", "on səkkiz kvintilyon", 0, true},
		{"overflow accumulation", "bir kvintilyon bir kvintilyon", 0, true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse(%q) = %d, nil; want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("Parse(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	values := []int64{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
		20, 42, 99, 100, 101, 123, 999,
		1000, 1001, 9999, 10000, 100000, 999999,
		1000000, 2300095, 1000000000,
		-1, -42, -1000,
	}

	for _, n := range values {
		t.Run(fmt.Sprintf("%d", n), func(t *testing.T) {
			t.Parallel()
			text := Convert(n)
			if text == "" {
				t.Fatalf("Convert(%d) returned empty string", n)
			}
			got, err := Parse(text)
			if err != nil {
				t.Fatalf("Parse(Convert(%d)) = error: %v (text: %q)", n, err, text)
			}
			if got != n {
				t.Errorf("Parse(Convert(%d)) = %d, want %d (text: %q)", n, got, n, text)
			}
		})
	}
}

func ExampleConvert() {
	fmt.Println(Convert(123))
	// Output: yüz iyirmi üç
}

func ExampleConvertOrdinal() {
	fmt.Println(ConvertOrdinal(5))
	// Output: beşinci
}

func ExampleConvertFloat() {
	fmt.Println(ConvertFloat("3.14", MathMode))
	// Output: üç tam yüzdə on dörd
}

func ExampleParse() {
	n, _ := Parse("yüz iyirmi üç")
	fmt.Println(n)
	// Output: 123
}

func BenchmarkConvert(b *testing.B) {
	for b.Loop() {
		Convert(2300095)
	}
}

func BenchmarkConvertOrdinal(b *testing.B) {
	for b.Loop() {
		ConvertOrdinal(2300095)
	}
}

func BenchmarkConvertFloat(b *testing.B) {
	for b.Loop() {
		ConvertFloat("3.14", MathMode)
	}
}

func BenchmarkParse(b *testing.B) {
	for b.Loop() {
		Parse("iki milyon üç yüz min doxsan beş")
	}
}
