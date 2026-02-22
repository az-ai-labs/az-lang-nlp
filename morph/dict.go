package morph

import (
	"bytes"
	_ "embed"
)

//go:embed dict.txt
var dictRaw []byte

// minLineLen is the minimum valid line length in dict.txt:
// one byte for the POS tag plus at least one character for the lemma.
const minLineLen = 2

// Parsed dictionary data, populated by init().
var (
	dictLemmas []string        // sorted lemmas, kept for test integrity checks
	dictMap    map[string]byte // stem -> POS byte for O(1) lookups
)

func init() {
	// Parse dictRaw: each line is <POS_byte><lemma>\n
	// These lookups are for soft ranking in fsm.go walk() base case,
	// not as hard filters — an unknown stem does not block analysis.
	lines := bytes.Split(dictRaw, []byte("\n"))
	dictMap = make(map[string]byte, len(lines))
	dictLemmas = make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line) < minLineLen {
			continue
		}
		lemma := string(line[1:])
		if _, exists := dictMap[lemma]; !exists {
			dictMap[lemma] = line[0]
		}
		dictLemmas = append(dictLemmas, lemma)
	}
}

// IsKnownStem reports whether s is a known dictionary stem.
// Expects lowercase Azerbaijani Latin input.
// Results may change as the dictionary grows.
func IsKnownStem(s string) bool {
	return isKnownStem(s)
}

// isKnownStem reports whether s is a known dictionary stem.
// Expects lowercase Latin input.
func isKnownStem(s string) bool {
	if s == "" {
		return false
	}
	_, ok := dictMap[s]
	return ok
}

// stemPOS returns the POS byte for a known stem, or 0 if not found.
// Expects lowercase Latin input.
//
//nolint:unused // called from tests; integrated into FSM in PR #2
func stemPOS(s string) byte {
	if s == "" {
		return 0
	}
	return dictMap[s]
}
