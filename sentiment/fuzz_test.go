package sentiment

import (
	"math"
	"testing"
)

func FuzzAnalyze(f *testing.F) {
	f.Add("Bu gözəl bir gündür")
	f.Add("Pis hava")
	f.Add("")
	f.Add("123 456")
	f.Add("yaxşı pis yaxşı")

	f.Fuzz(func(t *testing.T, s string) {
		r := Analyze(s)

		// Score must be in [-1, 1] range.
		if r.Score < -1.0 || r.Score > 1.0 {
			t.Errorf("Score out of range: %.3f", r.Score)
		}

		// Score must not be NaN or Inf.
		if math.IsNaN(r.Score) || math.IsInf(r.Score, 0) {
			t.Errorf("Score is NaN or Inf: %v", r.Score)
		}

		// Sentiment must be one of the three valid values.
		switch r.Sentiment {
		case Negative, Neutral, Positive:
			// ok
		default:
			t.Errorf("invalid Sentiment: %d", r.Sentiment)
		}

		// Positive and Negative counts must be non-negative.
		if r.Positive < 0 {
			t.Errorf("Positive count negative: %d", r.Positive)
		}
		if r.Negative < 0 {
			t.Errorf("Negative count negative: %d", r.Negative)
		}

		// Total must be >= Positive + Negative.
		if r.Total < r.Positive+r.Negative {
			t.Errorf("Total (%d) < Positive (%d) + Negative (%d)", r.Total, r.Positive, r.Negative)
		}

		// Polarity consistency.
		if r.Score > 0 && r.Sentiment != Positive {
			t.Errorf("Score %.3f but Sentiment %v", r.Score, r.Sentiment)
		}
		if r.Score < 0 && r.Sentiment != Negative {
			t.Errorf("Score %.3f but Sentiment %v", r.Score, r.Sentiment)
		}
	})
}
