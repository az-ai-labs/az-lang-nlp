package ner

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
)

var updateGolden = flag.Bool("update", false, "regenerate golden test files")

// goldenCase represents a single golden test case.
type goldenCase struct {
	Name     string   `json:"name"`
	Input    string   `json:"input"`
	Entities []Entity `json:"entities"`
}

const goldenPath = "../data/golden/ner.json"

func TestGolden(t *testing.T) {
	if *updateGolden {
		updateGoldenFile(t)
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("ner.json not found, run with -update to generate")
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

			got := Recognize(tc.Input)

			// Verify offset invariant for every entity.
			for _, e := range got {
				if tc.Input[e.Start:e.End] != e.Text {
					t.Errorf("invariant broken: s[%d:%d]=%q != Text=%q",
						e.Start, e.End, tc.Input[e.Start:e.End], e.Text)
				}
			}

			compareEntities(t, tc.Entities, got)
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
		cases[i].Entities = Recognize(cases[i].Input)
	}

	out, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatalf("marshaling golden data: %v", err)
	}

	out = append(out, '\n')

	if err := os.WriteFile(goldenPath, out, 0644); err != nil {
		t.Fatalf("writing golden file: %v", err)
	}

	t.Log("golden file updated, review with: git diff data/golden/ner.json")
}
