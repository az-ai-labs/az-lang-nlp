package normalize

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
)

var updateGolden = flag.Bool("update", false, "regenerate golden test files")

// goldenCase represents a single golden test case for normalization.
type goldenCase struct {
	Name          string `json:"name"`
	Input         string `json:"input"`
	WantWord      string `json:"want_word"`      // expected output of NormalizeWord
	WantText      string `json:"want_text"`      // expected output of Normalize (full text)
	WordOnly      bool   `json:"word_only"`      // if true, only NormalizeWord is tested (Input is a single word)
}

const goldenPath = "../data/golden/normalize.json"

func TestGolden(t *testing.T) {
	if *updateGolden {
		updateGoldenFile(t)
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("normalize.json not found, run with -update to generate")
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

			if tc.WantWord != "" {
				gotWord := NormalizeWord(tc.Input)
				if gotWord != tc.WantWord {
					t.Errorf("NormalizeWord(%q) = %q, want %q", tc.Input, gotWord, tc.WantWord)
				}
			}

			if !tc.WordOnly && tc.WantText != "" {
				gotText := Normalize(tc.Input)
				if gotText != tc.WantText {
					t.Errorf("Normalize(%q) = %q, want %q", tc.Input, gotText, tc.WantText)
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
		tc := &cases[i]
		tc.WantWord = NormalizeWord(tc.Input)
		if !tc.WordOnly {
			tc.WantText = Normalize(tc.Input)
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

	t.Log("golden file updated, review with: git diff data/golden/normalize.json")
}
