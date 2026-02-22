package keywords

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"testing"
)

var updateGolden = flag.Bool("update", false, "regenerate golden test files")

// goldenCase represents a single golden test case for keyword extraction.
type goldenCase struct {
	Name         string    `json:"name"`
	Input        string    `json:"input"`
	WantTFIDF    []Keyword `json:"want_tfidf"`
	WantTextRank []Keyword `json:"want_textrank"`
	WantKeywords []string  `json:"want_keywords"`
}

const goldenPath = "../data/golden/keywords.json"

func TestGolden(t *testing.T) {
	if *updateGolden {
		updateGoldenFile(t)
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("keywords.json not found, run with -update to generate")
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

			gotTFIDF := ExtractTFIDF(tc.Input, 5)
			gotTextRank := ExtractTextRank(tc.Input, 5)
			gotKeywords := Keywords(tc.Input)

			if msg := diffKeywords(gotTFIDF, tc.WantTFIDF); msg != "" {
				t.Errorf("ExtractTFIDF(%q): %s", tc.Name, msg)
			}

			if msg := diffKeywords(gotTextRank, tc.WantTextRank); msg != "" {
				t.Errorf("ExtractTextRank(%q): %s", tc.Name, msg)
			}

			kwJSON, _ := json.Marshal(gotKeywords)
			wantKWJSON, _ := json.Marshal(tc.WantKeywords)
			if string(kwJSON) != string(wantKWJSON) {
				t.Errorf("Keywords(%q):\n  got  %s\n  want %s", tc.Name, kwJSON, wantKWJSON)
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
		tc := &cases[i]
		tc.WantTFIDF = ExtractTFIDF(tc.Input, 5)
		tc.WantTextRank = ExtractTextRank(tc.Input, 5)
		tc.WantKeywords = Keywords(tc.Input)
	}

	out, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatalf("marshaling golden data: %v", err)
	}

	out = append(out, '\n')

	if err := os.WriteFile(goldenPath, out, 0644); err != nil {
		t.Fatalf("writing golden file: %v", err)
	}

	t.Log("golden file updated, review with: git diff data/golden/keywords.json")
}

// scoreEpsilon tolerates cross-platform float64 non-determinism
// (last significant digit may differ between macOS and Linux).
const scoreEpsilon = 1e-13

func diffKeywords(got, want []Keyword) string {
	if len(got) != len(want) {
		gotJSON, _ := json.Marshal(got)
		wantJSON, _ := json.Marshal(want)
		return "length mismatch:\n  got  " + string(gotJSON) + "\n  want " + string(wantJSON)
	}
	for i := range got {
		if got[i].Stem != want[i].Stem || got[i].Count != want[i].Count {
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			return fmt.Sprintf("stem/count mismatch at [%d]:\n  got  %s\n  want %s", i, gotJSON, wantJSON)
		}
		if math.Abs(got[i].Score-want[i].Score) > scoreEpsilon {
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			return fmt.Sprintf("score mismatch at [%d]:\n  got  %s\n  want %s", i, gotJSON, wantJSON)
		}
	}
	return ""
}
