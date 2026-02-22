package keywords

import (
	"encoding/json"
	"flag"
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

			tfidfJSON, _ := json.Marshal(gotTFIDF)
			wantTFIDFJSON, _ := json.Marshal(tc.WantTFIDF)
			if string(tfidfJSON) != string(wantTFIDFJSON) {
				t.Errorf("ExtractTFIDF(%q):\n  got  %s\n  want %s", tc.Name, tfidfJSON, wantTFIDFJSON)
			}

			trJSON, _ := json.Marshal(gotTextRank)
			wantTRJSON, _ := json.Marshal(tc.WantTextRank)
			if string(trJSON) != string(wantTRJSON) {
				t.Errorf("ExtractTextRank(%q):\n  got  %s\n  want %s", tc.Name, trJSON, wantTRJSON)
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
