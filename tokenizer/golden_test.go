package tokenizer

import (
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update", false, "regenerate golden test files")

// goldenCase represents a single golden test case.
// Most cases only need Words and Sentences (string comparisons).
// WordTokens is optional — only for cases where offsets and types matter.
type goldenCase struct {
	Name           string  `json:"name"`
	Input          string  `json:"input"`
	Words          []string `json:"words"`
	Sentences      []string `json:"sentences"`
	WordTokens     []Token  `json:"word_tokens,omitempty"`
}

const goldenPath = "../data/golden/tokenizer.json"

func TestGolden(t *testing.T) {
	if *updateGolden {
		updateGoldenFile(t)
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("golden.json not found, run with -update to generate")
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

			// Always verify reconstruction invariant
			wordTokens := WordTokens(tc.Input)
			verifyInvariants(t, tc.Input, wordTokens)

			sentenceTokens := SentenceTokens(tc.Input)
			verifyInvariants(t, tc.Input, sentenceTokens)

			// Compare words
			gotWords := Words(tc.Input)
			compareStringSlice(t, "Words", tc.Words, gotWords)

			// Compare sentences
			gotSentences := Sentences(tc.Input)
			compareStringSlice(t, "Sentences", tc.Sentences, gotSentences)

			// Compare full word tokens if specified
			if len(tc.WordTokens) > 0 {
				compareTokenSlice(t, "WordTokens", tc.WordTokens, wordTokens)
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
		tc.Words = Words(tc.Input)
		tc.Sentences = Sentences(tc.Input)
		if len(tc.WordTokens) > 0 {
			tc.WordTokens = WordTokens(tc.Input)
		}
	}

	out, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatalf("marshaling golden data: %v", err)
	}

	// Ensure trailing newline
	out = append(out, '\n')

	if err := os.WriteFile(goldenPath, out, 0644); err != nil {
		t.Fatalf("writing golden file: %v", err)
	}

	t.Log("golden file updated, review with: git diff data/golden/tokenizer.json")
}

func compareStringSlice(t *testing.T, label string, want, got []string) {
	t.Helper()

	if len(want) == 0 && len(got) == 0 {
		return
	}

	if len(got) != len(want) {
		t.Errorf("%s: got %d items, want %d\n  got:  %v\n  want: %v",
			label, len(got), len(want), got, want)
		return
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d]: got %q, want %q", label, i, got[i], want[i])
		}
	}
}

func compareTokenSlice(t *testing.T, label string, want, got []Token) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("%s: got %d tokens, want %d", label, len(got), len(want))
		printTokenDiff(t, label, want, got)
		return
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d]:\n  got:  %s\n  want: %s", label, i, got[i], want[i])
		}
	}
}

func printTokenDiff(t *testing.T, _ string, want, got []Token) {
	t.Helper()

	var sb strings.Builder
	sb.WriteString("  want:\n")
	for _, tok := range want {
		sb.WriteString("    " + tok.String() + "\n")
	}
	sb.WriteString("  got:\n")
	for _, tok := range got {
		sb.WriteString("    " + tok.String() + "\n")
	}
	t.Log(sb.String())
}
