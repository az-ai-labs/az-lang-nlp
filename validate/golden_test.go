package validate

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
)

var updateGolden = flag.Bool("update", false, "regenerate golden test files")

// goldenCase represents a single golden test case for validation.
type goldenCase struct {
	Name       string `json:"name"`
	Input      string `json:"input"`
	WantScore  int    `json:"want_score"`
	WantIssues []Issue `json:"want_issues"`
}

const goldenPath = "../data/golden/validate.json"

func TestGolden(t *testing.T) {
	if *updateGolden {
		updateGoldenFile(t)
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("validate.json not found, run with -update to generate")
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

			got := Validate(tc.Input)

			if got.Score != tc.WantScore {
				t.Errorf("Validate(%q).Score = %d, want %d", tc.Name, got.Score, tc.WantScore)
			}

			gotJSON, _ := json.Marshal(got.Issues)
			wantJSON, _ := json.Marshal(tc.WantIssues)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("Validate(%q).Issues mismatch:\n  got  %s\n  want %s", tc.Name, gotJSON, wantJSON)
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
		report := Validate(tc.Input)
		tc.WantScore = report.Score
		tc.WantIssues = report.Issues
	}

	out, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatalf("marshaling golden data: %v", err)
	}

	out = append(out, '\n')

	if err := os.WriteFile(goldenPath, out, 0644); err != nil {
		t.Fatalf("writing golden file: %v", err)
	}

	t.Log("golden file updated, review with: git diff data/golden/validate.json")
}
