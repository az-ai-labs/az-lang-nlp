package morph

import (
	"bytes"
	_ "embed"
	"sort"
)

//go:embed dict.txt
var dictRaw []byte

// minLineLen is the minimum valid line length in dict.txt:
// one byte for the POS tag plus at least one character for the lemma.
const minLineLen = 2

// Parsed dictionary data, populated by init().
var (
	dictLemmas []string // sorted lemmas for binary search
	dictPOS    []byte   // parallel slice: dictPOS[i] is the POS byte for dictLemmas[i]
)

func init() {
	// Parse dictRaw: each line is <POS_byte><lemma>\n
	// The generator sorts by lemma, so dictLemmas is already sorted.
	// These lookups are for soft ranking in fsm.go walk() base case,
	// not as hard filters — an unknown stem does not block analysis.
	lines := bytes.Split(dictRaw, []byte("\n"))
	dictLemmas = make([]string, 0, len(lines))
	dictPOS = make([]byte, 0, len(lines))
	for _, line := range lines {
		if len(line) < minLineLen {
			continue
		}
		dictPOS = append(dictPOS, line[0])
		dictLemmas = append(dictLemmas, string(line[1:]))
	}
}

// isKnownStem reports whether s is a known dictionary stem.
// Expects lowercase Latin input.
// Designed for soft ranking in fsm.go walk() base case, not hard filtering.
func isKnownStem(s string) bool {
	if s == "" {
		return false
	}
	i := sort.SearchStrings(dictLemmas, s)
	return i < len(dictLemmas) && dictLemmas[i] == s
}

// stemPOS returns the POS byte for a known stem, or 0 if not found.
// Expects lowercase Latin input.
//
//nolint:unused // called from tests; integrated into FSM in PR #2
func stemPOS(s string) byte {
	if s == "" {
		return 0
	}
	i := sort.SearchStrings(dictLemmas, s)
	if i < len(dictLemmas) && dictLemmas[i] == s {
		return dictPOS[i]
	}
	return 0
}
