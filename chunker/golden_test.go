package chunker

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"testing"
)

var updateGolden = flag.Bool("update", false, "regenerate golden test files")

// goldenCase represents a single golden test case for chunking.
type goldenCase struct {
	Name    string  `json:"name"`
	Input   string  `json:"input"`
	Size    int     `json:"size"`
	Overlap int     `json:"overlap"`
	WantBySize     []Chunk `json:"by_size"`
	WantBySentence []Chunk `json:"by_sentence"`
	WantRecursive  []Chunk `json:"recursive"`
}

const goldenPath = "../data/golden/chunker.json"

func TestGolden(t *testing.T) {
	if *updateGolden {
		updateGoldenFile(t)
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("chunker.json not found, run with -update to generate")
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

			gotBS := BySize(tc.Input, tc.Size, tc.Overlap)
			gotSe := BySentence(tc.Input, tc.Size, tc.Overlap)
			gotRe := Recursive(tc.Input, tc.Size, tc.Overlap)

			if msg := diffChunks(gotBS, tc.WantBySize); msg != "" {
				t.Errorf("BySize(%q, %d, %d): %s", tc.Name, tc.Size, tc.Overlap, msg)
			}

			if msg := diffChunks(gotSe, tc.WantBySentence); msg != "" {
				t.Errorf("BySentence(%q, %d, %d): %s", tc.Name, tc.Size, tc.Overlap, msg)
			}

			if msg := diffChunks(gotRe, tc.WantRecursive); msg != "" {
				t.Errorf("Recursive(%q, %d, %d): %s", tc.Name, tc.Size, tc.Overlap, msg)
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
		tc.WantBySize = BySize(tc.Input, tc.Size, tc.Overlap)
		tc.WantBySentence = BySentence(tc.Input, tc.Size, tc.Overlap)
		tc.WantRecursive = Recursive(tc.Input, tc.Size, tc.Overlap)
	}

	out, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatalf("marshaling golden data: %v", err)
	}

	out = append(out, '\n')

	if err := os.WriteFile(goldenPath, out, 0o644); err != nil {
		t.Fatalf("writing golden file: %v", err)
	}

	t.Log("golden file updated, review with: git diff data/golden/chunker.json")
}

func diffChunks(got, want []Chunk) string {
	if len(got) != len(want) {
		gotJSON, _ := json.Marshal(got)
		wantJSON, _ := json.Marshal(want)
		return "length mismatch:\n  got  " + string(gotJSON) + "\n  want " + string(wantJSON)
	}
	for i := range got {
		if got[i].Text != want[i].Text || got[i].Start != want[i].Start ||
			got[i].End != want[i].End || got[i].Index != want[i].Index {
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			return fmt.Sprintf("mismatch at [%d]:\n  got  %s\n  want %s", i, gotJSON, wantJSON)
		}
	}
	return ""
}
