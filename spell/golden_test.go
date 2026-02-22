package spell

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
)

var updateGolden = flag.Bool("update", false, "regenerate golden test files")

// goldenCase represents a single golden test case for spell checking.
type goldenCase struct {
	Name            string `json:"name"`
	Input           string `json:"input"`
	WantIsCorrect   bool   `json:"want_is_correct"`
	WantCorrectWord string `json:"want_correct_word"`
	WantCorrectText string `json:"want_correct_text"`
	// WordOnly, if true, skips the full-text Correct test (Input is a single word).
	WordOnly bool `json:"word_only"`
}

const goldenPath = "../data/golden/spell.json"

func TestGolden(t *testing.T) {
	if *updateGolden {
		updateGoldenFile(t)
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("spell.json not found, run with -update to generate")
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

			gotIsCorrect := IsCorrect(tc.Input)
			if gotIsCorrect != tc.WantIsCorrect {
				t.Errorf("IsCorrect(%q) = %v, want %v", tc.Input, gotIsCorrect, tc.WantIsCorrect)
			}

			gotCorrectWord := CorrectWord(tc.Input)
			if gotCorrectWord != tc.WantCorrectWord {
				t.Errorf("CorrectWord(%q) = %q, want %q", tc.Input, gotCorrectWord, tc.WantCorrectWord)
			}

			if !tc.WordOnly {
				gotCorrectText := Correct(tc.Input)
				if gotCorrectText != tc.WantCorrectText {
					t.Errorf("Correct(%q) = %q, want %q", tc.Input, gotCorrectText, tc.WantCorrectText)
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
		tc.WantIsCorrect = IsCorrect(tc.Input)
		tc.WantCorrectWord = CorrectWord(tc.Input)
		if !tc.WordOnly {
			tc.WantCorrectText = Correct(tc.Input)
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

	t.Log("golden file updated, review with: git diff data/golden/spell.json")
}
