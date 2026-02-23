package sentiment

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPol   Sentiment
		wantPos   bool // Score > 0
		wantNeg   bool // Score < 0
		minPos    int  // minimum positive word count
		minNeg    int  // minimum negative word count
	}{
		{
			name:    "positive sentence",
			input:   "Bu film çox gözəl və maraqlı idi",
			wantPol: Positive,
			wantPos: true,
			minPos:  2,
		},
		{
			name:    "negative sentence",
			input:   "Bu pis və çirkin bir hadisədir",
			wantPol: Negative,
			wantNeg: true,
			minNeg:  2,
		},
		{
			name:    "neutral no sentiment words",
			input:   "Bakı Azərbaycanın paytaxtıdır",
			wantPol: Neutral,
		},
		{
			name:    "empty input",
			input:   "",
			wantPol: Neutral,
		},
		{
			name:    "single positive word",
			input:   "gözəl",
			wantPol: Positive,
			wantPos: true,
			minPos:  1,
		},
		{
			name:    "single negative word",
			input:   "dəhşətli",
			wantPol: Negative,
			wantNeg: true,
			minNeg:  1,
		},
		{
			name:    "mixed sentiment",
			input:   "Yaxşı amma bahalı",
			wantPol: Positive, // yaxşı(0.8) outweighs bahalı(-0.4)
			wantPos: true,
			minPos:  1,
			minNeg:  1,
		},
		{
			name:    "numbers only",
			input:   "123 456 789",
			wantPol: Neutral,
		},
		{
			name:    "oversized input",
			input:   strings.Repeat("a", maxInputBytes+1),
			wantPol: Neutral,
		},
		{
			name:    "positive with inflection",
			input:   "Sevgi hər şeydən güclüdür",
			wantPol: Positive,
			wantPos: true,
			minPos:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Analyze(tt.input)
			if got.Sentiment != tt.wantPol {
				t.Errorf("Sentiment = %v, want %v (score=%.3f, pos=%d, neg=%d)",
					got.Sentiment, tt.wantPol, got.Score, got.Positive, got.Negative)
			}
			if tt.wantPos && got.Score <= 0 {
				t.Errorf("Score = %.3f, want > 0", got.Score)
			}
			if tt.wantNeg && got.Score >= 0 {
				t.Errorf("Score = %.3f, want < 0", got.Score)
			}
			if got.Positive < tt.minPos {
				t.Errorf("Positive = %d, want >= %d", got.Positive, tt.minPos)
			}
			if got.Negative < tt.minNeg {
				t.Errorf("Negative = %d, want >= %d", got.Negative, tt.minNeg)
			}
		})
	}
}

func TestScore(t *testing.T) {
	score := Score("Bu çox gözəl bir gündür")
	if score <= 0 {
		t.Errorf("Score(%q) = %.3f, want > 0", "Bu çox gözəl bir gündür", score)
	}
}

func TestIsPositive(t *testing.T) {
	if !IsPositive("Gözəl və maraqlı") {
		t.Error("IsPositive(\"Gözəl və maraqlı\") = false, want true")
	}
	if IsPositive("Pis və çirkin") {
		t.Error("IsPositive(\"Pis və çirkin\") = true, want false")
	}
}

func TestSentimentEnum(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		tests := []struct {
			s    Sentiment
			want string
		}{
			{Negative, "Negative"},
			{Neutral, "Neutral"},
			{Positive, "Positive"},
			{Sentiment(42), "Sentiment(42)"},
		}
		for _, tt := range tests {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("Sentiment(%d).String() = %q, want %q", int(tt.s), got, tt.want)
			}
		}
	})

	t.Run("JSON round-trip", func(t *testing.T) {
		for _, s := range []Sentiment{Negative, Neutral, Positive} {
			data, err := json.Marshal(s)
			if err != nil {
				t.Fatalf("Marshal(%v): %v", s, err)
			}
			var got Sentiment
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal(%s): %v", data, err)
			}
			if got != s {
				t.Errorf("round-trip: got %v, want %v", got, s)
			}
		}
	})

	t.Run("UnmarshalJSON error", func(t *testing.T) {
		var s Sentiment
		if err := json.Unmarshal([]byte(`"Unknown"`), &s); err == nil {
			t.Error("expected error for unknown sentiment string")
		}
		if err := json.Unmarshal([]byte(`123`), &s); err == nil {
			t.Error("expected error for non-string JSON")
		}
	})
}

func TestResultString(t *testing.T) {
	r := Result{
		Sentiment: Positive,
		Score:     0.75,
		Positive:  3,
		Negative:  1,
		Total:     10,
	}
	got := r.String()
	if !strings.Contains(got, "Positive") || !strings.Contains(got, "0.75") {
		t.Errorf("Result.String() = %q, want to contain Positive and 0.75", got)
	}
}

func TestLexiconLoaded(t *testing.T) {
	if len(lexicon) == 0 {
		t.Fatal("lexicon is empty; embedding failed")
	}
	// Spot-check a few known entries.
	checks := []string{"yaxşı", "pis", "gözəl", "nifrət"}
	for _, stem := range checks {
		if _, ok := lexicon[stem]; !ok {
			t.Errorf("lexicon missing expected stem %q", stem)
		}
	}
}

func TestIsNonLinguistic(t *testing.T) {
	tests := []struct {
		word string
		want bool
	}{
		{"123", true},
		{"...", true},
		{"hello", false},
		{"test123", false},
		{"gözəl", false},
	}
	for _, tt := range tests {
		if got := isNonLinguistic(tt.word); got != tt.want {
			t.Errorf("isNonLinguistic(%q) = %v, want %v", tt.word, got, tt.want)
		}
	}
}

func BenchmarkAnalyze(b *testing.B) {
	text := "Bu film çox gözəl və maraqlı idi, amma bəzi hissələri darıxdırıcı idi"
	b.SetBytes(int64(len(text)))
	b.ResetTimer()
	for b.Loop() {
		Analyze(text)
	}
}
