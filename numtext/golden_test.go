package numtext

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
)

var updateGolden = flag.Bool("update", false, "regenerate golden test files")

type goldenCase struct {
	Name     string `json:"name"`
	Input    int64  `json:"input"`
	Cardinal string `json:"cardinal"`
	Ordinal  string `json:"ordinal"`
}

const goldenPath = "../data/golden/numtext.json"

func TestGolden(t *testing.T) {
	if *updateGolden {
		updateGoldenFile(t)
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("golden file not found, run with -update to generate")
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

			gotCardinal := Convert(tc.Input)
			if gotCardinal != tc.Cardinal {
				t.Errorf("Convert(%d) = %q, want %q", tc.Input, gotCardinal, tc.Cardinal)
			}

			gotOrdinal := ConvertOrdinal(tc.Input)
			if gotOrdinal != tc.Ordinal {
				t.Errorf("ConvertOrdinal(%d) = %q, want %q", tc.Input, gotOrdinal, tc.Ordinal)
			}

			// Round-trip for cardinal (skip negatives and zero — Parse returns 0 for zero text)
			if tc.Input != 0 {
				parsed, err := Parse(gotCardinal)
				if err != nil {
					t.Errorf("Parse(%q) error: %v", gotCardinal, err)
				} else if parsed != tc.Input {
					t.Errorf("Parse(Convert(%d)) = %d", tc.Input, parsed)
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
		tc.Cardinal = Convert(tc.Input)
		tc.Ordinal = ConvertOrdinal(tc.Input)
	}

	out, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatalf("marshaling golden data: %v", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(goldenPath, out, 0644); err != nil {
		t.Fatalf("writing golden file: %v", err)
	}

	t.Log("golden file updated, review with: git diff data/golden/numtext.json")
}
