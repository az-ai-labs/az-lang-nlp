package sentiment

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
)

var updateGolden = flag.Bool("update", false, "regenerate golden test files")

// goldenCase represents a single golden test case for sentiment analysis.
type goldenCase struct {
	Name              string `json:"name"`
	Input             string `json:"input"`
	WantSentiment     string `json:"want_sentiment"`
	WantScorePositive *bool  `json:"want_score_positive,omitempty"` // nil for neutral
}

const goldenPath = "../data/golden/sentiment.json"

func TestGolden(t *testing.T) {
	if *updateGolden {
		updateGoldenFile(t)
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("sentiment.json not found, run with -update to generate")
		}
		t.Fatalf("reading golden file: %v", err)
	}

	var cases []goldenCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("parsing golden file: %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			got := Analyze(tc.Input)

			if got.Sentiment.String() != tc.WantSentiment {
				t.Errorf("Sentiment: got %q, want %q", got.Sentiment.String(), tc.WantSentiment)
			}

			if tc.WantScorePositive != nil {
				scoreIsPositive := got.Score > 0
				if scoreIsPositive != *tc.WantScorePositive {
					t.Errorf("Score positivity: got score=%.4f (positive=%v), want positive=%v",
						got.Score, scoreIsPositive, *tc.WantScorePositive)
				}
			}
		})
	}
}

func updateGoldenFile(t *testing.T) {
	t.Helper()

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden file for update: %v", err)
	}

	var cases []goldenCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("parsing golden file for update: %v", err)
	}

	for i := range cases {
		got := Analyze(cases[i].Input)
		cases[i].WantSentiment = got.Sentiment.String()

		switch got.Sentiment {
		case Positive:
			v := true
			cases[i].WantScorePositive = &v
		case Negative:
			v := false
			cases[i].WantScorePositive = &v
		default:
			cases[i].WantScorePositive = nil
		}
	}

	out, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatalf("marshaling golden data: %v", err)
	}

	out = append(out, '\n')

	if err := os.WriteFile(goldenPath, out, 0644); err != nil {
		t.Fatalf("writing golden file: %v", err)
	}

	t.Log("golden file updated, review with: git diff data/golden/sentiment.json")
}
