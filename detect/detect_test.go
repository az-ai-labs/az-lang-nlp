package detect

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		in         string
		wantLang   Language
		wantScript Script
	}{
		{
			name:       "azerbaijani latin schwa",
			in:         "Salam, necəsən? Bu gün hava çox gözəldir.",
			wantLang:   Azerbaijani,
			wantScript: ScriptLatn,
		},
		{
			name:       "russian cyrillic",
			in:         "Привет, как у тебя дела сегодня?",
			wantLang:   Russian,
			wantScript: ScriptCyrl,
		},
		{
			name:       "english latin",
			in:         "Hello, how are you doing today?",
			wantLang:   English,
			wantScript: ScriptLatn,
		},
		{
			name:       "turkish latin",
			in:         "Türkiye'de yaşayan insanlar çalışkan ve güler yüzlüdür",
			wantLang:   Turkish,
			wantScript: ScriptLatn,
		},
		{
			name:       "azerbaijani cyrillic",
			in:         "Бу мәтн Азәрбајҹан дилиндәдир",
			wantLang:   Azerbaijani,
			wantScript: ScriptCyrl,
		},
		{
			name:       "azerbaijani latin xq signal",
			in:         "Bu gün yaxşı hava olacaq deyirlər",
			wantLang:   Azerbaijani,
			wantScript: ScriptLatn,
		},
		{
			name:       "azerbaijani latin schwa strong",
			in:         "Kitablarımızdan öyrəndiklərimiz çoxdur",
			wantLang:   Azerbaijani,
			wantScript: ScriptLatn,
		},
		{
			name:       "dotted I azerbaijani",
			in:         "İşlər yaxşı gedir, hər şey qaydasındadır",
			wantLang:   Azerbaijani,
			wantScript: ScriptLatn,
		},
		{
			name:       "dotted I turkish",
			in:         "İşler iyi gidiyor, her şey yolunda gidiyor",
			wantLang:   Turkish,
			wantScript: ScriptLatn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Detect(tt.in)
			if got.Lang != tt.wantLang {
				t.Errorf("Lang: got %s, want %s", got.Lang, tt.wantLang)
			}
			if got.Script != tt.wantScript {
				t.Errorf("Script: got %s, want %s", got.Script, tt.wantScript)
			}
			if got.Confidence <= 0 {
				t.Errorf("Confidence: got %v, want > 0", got.Confidence)
			}
		})
	}
}

func TestDetectAll(t *testing.T) {
	t.Parallel()
	t.Run("returns exactly four results", func(t *testing.T) {
		t.Parallel()
		results := DetectAll("Salam, necəsən? Bu gün hava çox gözəldir.")
		if len(results) != 4 {
			t.Fatalf("got %d results, want 4", len(results))
		}
	})

	t.Run("results sorted by descending confidence", func(t *testing.T) {
		t.Parallel()
		results := DetectAll("Salam, necəsən? Bu gün hava çox gözəldir.")
		for i := 1; i < len(results); i++ {
			if results[i].Confidence > results[i-1].Confidence {
				t.Errorf("results[%d].Confidence=%v > results[%d].Confidence=%v — not sorted",
					i, results[i].Confidence, i-1, results[i-1].Confidence)
			}
		}
	})

	t.Run("confidences sum to approximately 1.0", func(t *testing.T) {
		t.Parallel()
		results := DetectAll("Привет, как у тебя дела сегодня?")
		var sum float64
		for _, r := range results {
			sum += r.Confidence
		}
		if math.Abs(sum-1.0) > 1e-9 {
			t.Errorf("sum of confidences = %v, want 1.0", sum)
		}
	})

	t.Run("nil on empty input", func(t *testing.T) {
		t.Parallel()
		if got := DetectAll(""); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
}

func TestLang(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"azerbaijani", "Salam, necəsən? Bu gün hava çox gözəldir.", "az"},
		{"russian", "Привет, как у тебя дела сегодня?", "ru"},
		{"english", "Hello, how are you doing today?", "en"},
		{"turkish", "Türkiye'de yaşayan insanlar çalışkan ve güler yüzlüdür", "tr"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Lang(tt.in)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want Language
	}{
		{"empty", "", Unknown},
		{"whitespace", "   \t\n", Unknown},
		{"digits_only", "1234567890", Unknown},
		{"punctuation_only", "!!!...???", Unknown},
		{"few_letters", "test", Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Detect(tt.in)
			if got.Lang != tt.want {
				t.Errorf("got %s, want %s", got.Lang, tt.want)
			}
		})
	}
}

func TestLanguageJSON(t *testing.T) {
	t.Parallel()
	langs := []Language{Unknown, Azerbaijani, Russian, English, Turkish}

	for _, lang := range langs {
		t.Run(lang.String(), func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(lang)
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}

			var decoded Language
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("UnmarshalJSON: %v", err)
			}

			if decoded != lang {
				t.Errorf("round-trip: got %v, want %v", decoded, lang)
			}
		})
	}

	t.Run("unmarshal unknown string", func(t *testing.T) {
		t.Parallel()
		var l Language
		if err := l.UnmarshalJSON([]byte(`"Klingon"`)); err == nil {
			t.Error("want error for unknown language, got nil")
		}
	})

	t.Run("unmarshal non-string", func(t *testing.T) {
		t.Parallel()
		var l Language
		if err := l.UnmarshalJSON([]byte(`123`)); err == nil {
			t.Error("want error for non-string JSON, got nil")
		}
	})
}

func TestScriptJSON(t *testing.T) {
	t.Parallel()
	scripts := []Script{ScriptUnknown, ScriptLatn, ScriptCyrl}

	for _, sc := range scripts {
		name := sc.String()
		if name == "" {
			name = "ScriptUnknown"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(sc)
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}

			var decoded Script
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("UnmarshalJSON: %v", err)
			}

			if decoded != sc {
				t.Errorf("round-trip: got %v, want %v", decoded, sc)
			}
		})
	}

	t.Run("unmarshal unknown string", func(t *testing.T) {
		t.Parallel()
		var s Script
		if err := s.UnmarshalJSON([]byte(`"Glag"`)); err == nil {
			t.Error("want error for unknown script, got nil")
		}
	})

	t.Run("unmarshal non-string", func(t *testing.T) {
		t.Parallel()
		var s Script
		if err := s.UnmarshalJSON([]byte(`42`)); err == nil {
			t.Error("want error for non-string JSON, got nil")
		}
	})
}

func TestLanguageString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		lang Language
		want string
	}{
		{Unknown, "Unknown"},
		{Azerbaijani, "Azerbaijani"},
		{Russian, "Russian"},
		{English, "English"},
		{Turkish, "Turkish"},
		{Language(99), "Language(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			got := tt.lang.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScriptString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		script Script
		want   string
	}{
		{ScriptUnknown, ""},
		{ScriptLatn, "Latn"},
		{ScriptCyrl, "Cyrl"},
		{Script(99), "Script(99)"},
	}

	for _, tt := range tests {
		name := tt.want
		if name == "" {
			name = "ScriptUnknown"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := tt.script.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOversizedInputTruncated(t *testing.T) {
	t.Parallel()
	sentence := "Salam, necəsən? Bu gün hava çox gözəldir. "
	// Repeat until we exceed 1 MiB.
	repeat := (maxInputBytes / len(sentence)) + 2
	input := strings.Repeat(sentence, repeat)

	if len(input) <= maxInputBytes {
		t.Fatalf("test setup: input length %d must exceed maxInputBytes %d", len(input), maxInputBytes)
	}

	got := Detect(input)
	if got.Lang != Azerbaijani {
		t.Errorf("got %s, want Azerbaijani", got.Lang)
	}
}

func BenchmarkDetect(b *testing.B) {
	input := strings.Repeat("Bu gün hava çox gözəldir və biz sevincliyik. ", 100)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		Detect(input)
	}
}

func BenchmarkDetectShort(b *testing.B) {
	input := "Salam, necəsən?"
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		Detect(input)
	}
}

func BenchmarkDetectTrigram(b *testing.B) {
	// Text that triggers trigram fallback (Turkish-shared chars, no ə).
	input := strings.Repeat("Bu gün yaxşı bir gündür deyirlər. ", 100)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		Detect(input)
	}
}

func ExampleDetect() {
	r := Detect("Salam, necəsən? Bu gün hava çox gözəldir.")
	fmt.Println(r.Lang)
	// Output:
	// Azerbaijani
}

func ExampleLang() {
	fmt.Println(Lang("Hello, how are you doing today?"))
	// Output:
	// en
}

func ExampleDetectAll() {
	results := DetectAll("Привет, как у тебя дела сегодня?")
	fmt.Println(results[0].Lang)
	// Output:
	// Russian
}
